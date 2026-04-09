package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/config"
	"github.com/portainer/portainerctl/internal/output"
)

// kubeCmd provides both:
// 1. Direct Portainer Kubernetes API wrappers (namespaces, applications, ingresses, etc.)
// 2. A kubectl passthrough that constructs a kubeconfig using the Portainer proxy

func kubeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl",
		Short: "Run kubectl commands against a Portainer-managed Kubernetes environment",
		Long: `Run any kubectl command against a Kubernetes environment managed by Portainer.
portainerctl proxies the request through the Portainer API using your PAT.

Example:
  portainerctl kubectl --env 4 -- get pods -n default
  portainerctl kubectl --env 4 -- apply -f manifest.yaml
  portainerctl kubectl --env 4 -- get nodes`,
		DisableFlagParsing: false,
	}

	var envID int
	var rawKubectl bool
	cmd.Flags().IntVar(&envID, "env", 0, "Kubernetes environment ID (required)")
	cmd.Flags().BoolVar(&rawKubectl, "raw", false, "Use system kubectl with a generated kubeconfig (requires kubectl installed)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if envID == 0 {
			return fmt.Errorf("--env is required")
		}
		// Separate portainerctl flags from kubectl args using --
		kubectlArgs := args

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		ctx, err := cfg.Current()
		if err != nil {
			return err
		}

		if rawKubectl {
			return runKubectlViaProxy(ctx, envID, kubectlArgs)
		}

		// Default: call Portainer's Kubernetes proxy directly
		c := client.New(ctx)
		if len(kubectlArgs) == 0 {
			return fmt.Errorf("no kubectl arguments provided — example: portainerctl kubectl --env 4 -- get pods")
		}
		return kubeProxyCall(c, envID, kubectlArgs)
	}

	// Kubernetes resource subcommands (Portainer API, not raw kubectl)
	cmd.AddCommand(
		kubeNamespacesCmd(),
		kubeApplicationsCmd(),
		kubeServicesCmd(),
		kubeIngressesCmd(),
		kubeSecretsCmd(),
		kubeConfigMapsCmd(),
		kubeVolumesCmd(),
		kubeDashboardCmd(),
		kubeNodesCmd(),
		kubeHelmCmd(),
	)

	return cmd
}

// kubeProxyCall translates simple kubectl-style args into Portainer K8s proxy REST calls.
// This gives basic get/describe/delete without requiring kubectl to be installed.
func kubeProxyCall(c *client.Client, envID int, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: portainerctl kubectl --env <id> -- <verb> <resource> [name] [-n namespace]")
	}
	verb := strings.ToLower(args[0])
	resource := strings.ToLower(args[1])
	name := ""
	namespace := "default"

	for i, a := range args {
		if (a == "-n" || a == "--namespace") && i+1 < len(args) {
			namespace = args[i+1]
		}
		if i == 2 && !strings.HasPrefix(a, "-") {
			name = a
		}
	}

	// Map resource to Portainer's Kubernetes proxy path
	k8sPath := buildK8sPath(resource, namespace, name)
	fullPath := fmt.Sprintf("/endpoints/%d/kubernetes%s", envID, k8sPath)

	switch verb {
	case "get", "describe":
		var result interface{}
		if err := c.Get(fullPath, &result); err != nil {
			return err
		}
		output.JSON(result)
	case "delete":
		if name == "" {
			return fmt.Errorf("delete requires a resource name")
		}
		if err := c.Delete(fullPath); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("%s %q deleted.", resource, name))
	default:
		return fmt.Errorf("verb %q not supported in proxy mode — use --raw for full kubectl support", verb)
	}
	return nil
}

func buildK8sPath(resource, namespace, name string) string {
	// Maps common resource types to their Kubernetes API paths
	namespacedResources := map[string]string{
		"pod": "pods", "pods": "pods",
		"deployment": "deployments", "deployments": "deployments",
		"service": "services", "services": "services",
		"configmap": "configmaps", "configmaps": "configmaps",
		"secret": "secrets", "secrets": "secrets",
		"ingress": "ingresses", "ingresses": "ingresses",
		"job": "jobs", "jobs": "jobs",
		"cronjob": "cronjobs", "cronjobs": "cronjobs",
		"statefulset": "statefulsets", "statefulsets": "statefulsets",
		"daemonset": "daemonsets", "daemonsets": "daemonsets",
		"replicaset": "replicasets", "replicasets": "replicasets",
		"persistentvolumeclaim": "persistentvolumeclaims", "pvc": "persistentvolumeclaims",
	}
	clusterResources := map[string]string{
		"node": "nodes", "nodes": "nodes",
		"namespace": "namespaces", "namespaces": "namespaces",
		"persistentvolume": "persistentvolumes", "pv": "persistentvolumes",
		"clusterrole": "clusterroles", "clusterroles": "clusterroles",
		"clusterrolebinding": "clusterrolebindings", "clusterrolebindings": "clusterrolebindings",
	}

	if canonical, ok := clusterResources[resource]; ok {
		base := "/api/v1/" + canonical
		if name != "" {
			base += "/" + name
		}
		return base
	}
	if canonical, ok := namespacedResources[resource]; ok {
		// Determine API group
		apiGroup := "api/v1"
		appsResources := map[string]bool{"deployments": true, "statefulsets": true, "daemonsets": true, "replicasets": true}
		if appsResources[canonical] {
			apiGroup = "apis/apps/v1"
		}
		base := fmt.Sprintf("/%s/namespaces/%s/%s", apiGroup, namespace, canonical)
		if name != "" {
			base += "/" + name
		}
		return base
	}
	// Fallback: pass through as-is
	return fmt.Sprintf("/api/v1/namespaces/%s/%s", namespace, resource)
}

// runKubectlViaProxy generates a kubeconfig that proxies through Portainer and runs kubectl.
func runKubectlViaProxy(ctx *config.Context, envID int, args []string) error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found in PATH — install kubectl or omit --raw flag")
	}

	// Build a minimal kubeconfig using Portainer as the server proxy
	serverURL := fmt.Sprintf("%s/api/endpoints/%d/kubernetes", strings.TrimRight(ctx.URL, "/"), envID)
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: %v
  name: portainer
contexts:
- context:
    cluster: portainer
    user: portainer
  name: portainer
current-context: portainer
users:
- name: portainer
  user:
    token: %s
`, serverURL, ctx.Insecure, ctx.Token)

	// Write to a temp file
	tmpFile, err := os.CreateTemp("", "portainerctl-kubeconfig-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp kubeconfig: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(kubeconfig); err != nil {
		return err
	}
	tmpFile.Close()

	kubectlArgs := append([]string{"--kubeconfig", tmpFile.Name()}, args...)
	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	kubectlCmd.Stdin = os.Stdin
	return kubectlCmd.Run()
}

func kubeNamespacesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "namespaces", Short: "Manage Kubernetes namespaces"}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List namespaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result map[string]interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/namespaces", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	getCmd := &cobra.Command{
		Use: "get <namespace>", Short: "Get namespace details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/namespaces/%s", envID, args[0]), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	getCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}

func kubeApplicationsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "applications", Short: "Manage Kubernetes applications"}
	var envID int
	var namespace string

	listCmd := &cobra.Command{
		Use: "list", Short: "List applications across all namespaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/kubernetes/%d/applications", envID)
			if namespace != "" { path += "?namespace=" + namespace }
			var result interface{}
			if err := c.Get(path, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	listCmd.Flags().StringVar(&namespace, "namespace", "", "Filter by namespace")

	restartCmd := &cobra.Command{
		Use: "restart <namespace> <kind> <name>", Short: "Restart an application", Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/kubernetes/%d/namespaces/%s/applications/%s/%s/restart", envID, args[0], args[1], args[2])
			if err := c.Post(path, nil, nil); err != nil { return err }
			output.Success(fmt.Sprintf("Application %s/%s restarted.", args[0], args[2]))
			return nil
		},
	}
	restartCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd, restartCmd)
	return cmd
}

func kubeServicesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "services", Short: "List Kubernetes services"}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/services", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd)
	return cmd
}

func kubeIngressesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ingresses", Short: "Manage Kubernetes ingresses"}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List all ingresses",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/ingresses", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd)
	return cmd
}

func kubeSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "secrets", Short: "List Kubernetes secrets"}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List all secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/secrets", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd)
	return cmd
}

func kubeConfigMapsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "configmaps", Short: "List Kubernetes ConfigMaps"}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List all ConfigMaps",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/configmaps", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd)
	return cmd
}

func kubeVolumesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "volumes", Short: "Manage Kubernetes volumes (PVCs)"}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List all PersistentVolumeClaims",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/volumes", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd)
	return cmd
}

func kubeDashboardCmd() *cobra.Command {
	var envID int
	cmd := &cobra.Command{
		Use: "dashboard", Short: "Get Kubernetes cluster dashboard summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/dashboard", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	cmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	return cmd
}

func kubeNodesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "nodes", Short: "Manage Kubernetes nodes"}
	var envID int

	limitsCmd := &cobra.Command{
		Use: "limits", Short: "Get node resource limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/nodes_limits", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	limitsCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	drainCmd := &cobra.Command{
		Use: "drain <node-name>", Short: "Drain a Kubernetes node", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Post(fmt.Sprintf("/kubernetes/%d/nodes/%s/drain", envID, args[0]), nil, nil); err != nil { return err }
			output.Success("Node " + args[0] + " drain initiated.")
			return nil
		},
	}
	drainCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	metricsCmd := &cobra.Command{
		Use: "metrics", Short: "Get node metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/kubernetes/%d/metrics/nodes", envID), &result); err != nil { return err }
			// Format as table if possible
			if m, ok := result.(map[string]interface{}); ok {
				rows := [][]string{}
				for name, data := range m {
					d, _ := data.(map[string]interface{})
					cpu := fmt.Sprintf("%v", d["CPU"])
					mem := fmt.Sprintf("%v", d["Memory"])
					rows = append(rows, []string{name, cpu, mem})
				}
				if len(rows) > 0 {
					output.Table([]string{"NODE", "CPU", "MEMORY"}, rows)
					return nil
				}
			}
			output.JSON(result)
			return nil
		},
	}
	metricsCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(limitsCmd, drainCmd, metricsCmd)
	return cmd
}

func kubeHelmCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "helm", Short: "Manage Helm releases on a Kubernetes environment"}
	var envID int
	var namespace string

	listCmd := &cobra.Command{
		Use: "list", Short: "List Helm releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/endpoints/%d/kubernetes/helm", envID)
			if namespace != "" { path += "?namespace=" + namespace }
			var result interface{}
			if err := c.Get(path, &result); err != nil { return err }
			// Try to format as table
			if releases, ok := result.([]interface{}); ok {
				rows := [][]string{}
				for _, r := range releases {
					rel, _ := r.(map[string]interface{})
					rows = append(rows, []string{
						fmt.Sprintf("%v", rel["name"]),
						fmt.Sprintf("%v", rel["namespace"]),
						fmt.Sprintf("%v", rel["chart"]),
						fmt.Sprintf("%v", rel["status"]),
						fmt.Sprintf("%v", rel["revision"]),
					})
				}
				output.Table([]string{"NAME", "NAMESPACE", "CHART", "STATUS", "REVISION"}, rows)
				return nil
			}
			output.JSON(result)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	listCmd.Flags().StringVar(&namespace, "namespace", "", "Filter by namespace")

	historyCmd := &cobra.Command{
		Use: "history <release>", Short: "Show release history", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/endpoints/%d/kubernetes/helm/%s/history", envID, args[0])
			if namespace != "" { path += "?namespace=" + namespace }
			var result interface{}
			if err := c.Get(path, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	historyCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	historyCmd.Flags().StringVar(&namespace, "namespace", "", "Namespace")

	var rollbackRevision int
	rollbackCmd := &cobra.Command{
		Use: "rollback <release>", Short: "Roll back a Helm release", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"revision": rollbackRevision, "namespace": namespace}
			if err := c.Post(fmt.Sprintf("/endpoints/%d/kubernetes/helm/%s/rollback", envID, args[0]), body, nil); err != nil { return err }
			output.Success(fmt.Sprintf("Release %s rolled back to revision %d.", args[0], rollbackRevision))
			return nil
		},
	}
	rollbackCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	rollbackCmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace")
	rollbackCmd.Flags().IntVar(&rollbackRevision, "revision", 0, "Target revision (0 = previous)")

	deleteCmd := &cobra.Command{
		Use: "delete <release>", Short: "Uninstall a Helm release", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/endpoints/%d/kubernetes/helm/%s", envID, args[0])
			if namespace != "" { path += "?namespace=" + namespace }
			if err := c.Delete(path); err != nil { return err }
			output.Success("Release " + args[0] + " uninstalled.")
			return nil
		},
	}
	deleteCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	deleteCmd.Flags().StringVar(&namespace, "namespace", "", "Namespace")

	cmd.AddCommand(listCmd, historyCmd, rollbackCmd, deleteCmd)
	return cmd
}

func helmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm",
		Short: "Manage Helm chart templates and repositories",
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List available Helm chart templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/templates/helm", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	var userID int
	reposCmd := &cobra.Command{
		Use: "repos", Short: "List Helm repositories for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == 0 { return fmt.Errorf("--user is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/users/%s/helm/repositories", strconv.Itoa(userID)), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	reposCmd.Flags().IntVar(&userID, "user", 0, "User ID")

	cmd.AddCommand(listCmd, reposCmd)
	return cmd
}

func dockerProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Run docker CLI commands against a Portainer-managed Docker environment",
		Long: `Proxies docker CLI commands through Portainer using your PAT.
Requires docker CLI to be installed locally.

Example:
  portainerctl docker --env 2 -- ps -a
  portainerctl docker --env 2 -- images
  portainerctl docker --env 2 -- inspect my-container`,
	}

	var envID int
	cmd.Flags().IntVar(&envID, "env", 0, "Docker environment ID (required)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if envID == 0 {
			return fmt.Errorf("--env is required")
		}
		_, err := exec.LookPath("docker")
		if err != nil {
			return fmt.Errorf("docker CLI not found in PATH")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		ctx, err := cfg.Current()
		if err != nil {
			return err
		}

		// The Portainer Docker proxy endpoint accepts standard Docker API calls
		proxyURL := fmt.Sprintf("%s/api/endpoints/%d/docker", strings.TrimRight(ctx.URL, "/"), envID)

		dockerArgs := append([]string{"-H", proxyURL}, args...)
		dockerCmd := exec.Command("docker", dockerArgs...)
		dockerCmd.Env = append(os.Environ(),
			"DOCKER_TLS_VERIFY=0",
			fmt.Sprintf("DOCKER_HOST=%s", proxyURL),
		)
		// Inject auth header via DOCKER_CUSTOM_HEADERS if supported, else note limitation
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr
		dockerCmd.Stdin = os.Stdin
		fmt.Fprintf(os.Stderr, "# Connecting to: %s\n", proxyURL)
		return dockerCmd.Run()
	}

	return cmd
}
