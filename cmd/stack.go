package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
	"github.com/spf13/cobra"
)

// portaineree.Stack — only scalar fields used in list table.
// GitConfig is gittypes.RepoConfig which has URL string at top level.
type gitRepoConfig struct {
	URL            string             `json:"URL"`
	ReferenceName  string             `json:"ReferenceName"`
	ConfigFilePath string             `json:"ConfigFilePath"`
	Authentication *gitAuthentication `json:"Authentication"`
}

type gitAuthentication struct {
	Username          string `json:"Username"`
	AuthorizationType int    `json:"AuthorizationType"`
}

type stackEnv struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type stackOption struct {
	Prune bool `json:"Prune"`
}

type stack struct {
	ID         int            `json:"Id"`
	Name       string         `json:"Name"`
	Type       int            `json:"Type"`
	EndpointID int            `json:"EndpointId"`
	Status     int            `json:"Status"`
	CreatedBy  string         `json:"CreatedBy"`
	GitConfig  *gitRepoConfig `json:"GitConfig"`
	Env        []stackEnv     `json:"Env"`
	Option     *stackOption   `json:"Option"`
}

type gitAuthOptions struct {
	username      string
	authType      string
	passwordStdin bool
}

func addGitAuthFlags(cmd *cobra.Command, opts *gitAuthOptions) {
	cmd.Flags().StringVar(&opts.username, "git-username", "", "Git username (or PORTAINERCTL_GIT_USERNAME)")
	cmd.Flags().StringVar(&opts.authType, "git-auth-type", "", "Git authorization type: basic or token (or PORTAINERCTL_GIT_AUTH_TYPE)")
	cmd.Flags().BoolVar(&opts.passwordStdin, "git-password-stdin", false, "Read the Git password/token from standard input")
}

func (opts gitAuthOptions) credentials(stdin io.Reader) (gitAuthentication, string, bool, error) {
	username := opts.username
	if username == "" {
		username = os.Getenv("PORTAINERCTL_GIT_USERNAME")
	}

	password, passwordSet := os.LookupEnv("PORTAINERCTL_GIT_PASSWORD")
	passwordSet = passwordSet && password != ""
	if opts.passwordStdin {
		if passwordSet {
			return gitAuthentication{}, "", false, fmt.Errorf("use either --git-password-stdin or PORTAINERCTL_GIT_PASSWORD, not both")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return gitAuthentication{}, "", false, fmt.Errorf("reading Git password from stdin: %w", err)
		}
		password = strings.TrimRight(string(data), "\r\n")
		passwordSet = true
	}

	authType := opts.authType
	if authType == "" {
		authType = os.Getenv("PORTAINERCTL_GIT_AUTH_TYPE")
	}
	credential := gitAuthentication{Username: username}
	switch strings.ToLower(authType) {
	case "", "basic":
		credential.AuthorizationType = 0
	case "token":
		credential.AuthorizationType = 1
	default:
		return gitAuthentication{}, "", false, fmt.Errorf("invalid Git authorization type %q: use basic or token", authType)
	}

	return credential, password, passwordSet, nil
}

func stackTypeLabel(t int) string {
	switch t {
	case 1:
		return "swarm"
	case 2:
		return "compose"
	case 3:
		return "kubernetes"
	default:
		return strconv.Itoa(t)
	}
}

func stackStatusLabel(s int) string {
	switch s {
	case 1:
		return "active"
	case 2:
		return "inactive"
	default:
		return "unknown"
	}
}

func stackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage stacks (Compose, Swarm, Kubernetes manifests)",
	}

	var listEnvID int
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := "/stacks"
			if listEnvID > 0 {
				path += fmt.Sprintf("?filters={\"EndpointID\":%d}", listEnvID)
			}
			var stacks []stack
			if err := c.Get(path, &stacks); err != nil {
				return err
			}
			rows := [][]string{}
			for _, s := range stacks {
				git := ""
				if s.GitConfig != nil {
					git = s.GitConfig.URL
				}
				rows = append(rows, []string{
					strconv.Itoa(s.ID),
					s.Name,
					stackTypeLabel(s.Type),
					stackStatusLabel(s.Status),
					strconv.Itoa(s.EndpointID),
					s.CreatedBy,
					git,
				})
			}
			output.Table([]string{"ID", "NAME", "TYPE", "STATUS", "ENV", "CREATED BY", "GIT"}, rows)
			return nil
		},
	}
	listCmd.Flags().IntVar(&listEnvID, "env", 0, "Filter by environment ID")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get stack details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/stacks/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	getByNameCmd := &cobra.Command{
		Use:   "get-by-name <n>",
		Short: "Get a stack by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/stacks/name/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	fileCmd := &cobra.Command{
		Use:   "file <id>",
		Short: "Get the compose/manifest file for a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			// Response is stacks.stackFileResponse with StackFileContent string
			var result map[string]interface{}
			if err := c.Get("/stacks/"+args[0]+"/file", &result); err != nil {
				return err
			}
			if content, ok := result["StackFileContent"].(string); ok {
				fmt.Println(content)
			} else {
				output.JSON(result)
			}
			return nil
		},
	}

	var deployName, deployFile, deployRepo, deployBranch, deployPath string
	var deployEnvID int
	var deployGitAuth gitAuthOptions

	deployComposeCmd := &cobra.Command{
		Use:     "deploy-compose",
		Short:   "Deploy a Compose stack from a local file",
		Example: `  portainerctl stack deploy-compose --name myapp --env 2 --file docker-compose.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployFile == "" {
				return fmt.Errorf("--name, --env, and --file are required")
			}
			data, err := os.ReadFile(deployFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":             deployName,
				"StackFileContent": string(data),
				"Env":              []interface{}{},
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/standalone/string?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployComposeCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployComposeCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target environment ID")
	deployComposeCmd.Flags().StringVar(&deployFile, "file", "", "Path to compose file")

	deploySwarmCmd := &cobra.Command{
		Use:     "deploy-swarm",
		Short:   "Deploy a Swarm stack from a local file",
		Example: `  portainerctl stack deploy-swarm --name myapp --env 2 --file docker-stack.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployFile == "" {
				return fmt.Errorf("--name, --env, and --file are required")
			}
			data, err := os.ReadFile(deployFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			// Fetch swarm ID from the environment's docker info
			var dockerInfo map[string]interface{}
			swarmID := ""
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/info", deployEnvID), &dockerInfo); err == nil {
				if swarm, ok := dockerInfo["Swarm"].(map[string]interface{}); ok {
					if cluster, ok := swarm["Cluster"].(map[string]interface{}); ok {
						if id, ok := cluster["ID"].(string); ok {
							swarmID = id
						}
					}
				}
			}
			body := map[string]interface{}{
				"Name":             deployName,
				"StackFileContent": string(data),
				"SwarmID":          swarmID,
				"Env":              []interface{}{},
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/swarm/string?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deploySwarmCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deploySwarmCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target environment ID")
	deploySwarmCmd.Flags().StringVar(&deployFile, "file", "", "Path to stack file")

	deployGitCmd := &cobra.Command{
		Use:     "deploy-git",
		Short:   "Deploy a Compose stack from a Git repository",
		Example: `  portainerctl stack deploy-git --name myapp --env 2 --repo https://github.com/org/repo --branch main --path docker-compose.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployRepo == "" {
				return fmt.Errorf("--name, --env, and --repo are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			credential, password, passwordSet, err := deployGitAuth.credentials(cmd.InOrStdin())
			if err != nil {
				return err
			}
			authenticated := passwordSet || credential.Username != ""
			if authenticated && password == "" {
				return fmt.Errorf("Git authentication requires a password/token via --git-password-stdin or PORTAINERCTL_GIT_PASSWORD")
			}
			body := map[string]interface{}{
				"Name":                        deployName,
				"RepositoryURL":               deployRepo,
				"RepositoryReferenceName":     "refs/heads/" + deployBranch,
				"ComposeFile":                 deployPath,
				"Env":                         []interface{}{},
				"RepositoryAuthentication":    authenticated,
				"RepositoryUsername":          credential.Username,
				"RepositoryPassword":          password,
				"RepositoryAuthorizationType": credential.AuthorizationType,
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/standalone/repository?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployGitCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployGitCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target environment ID")
	deployGitCmd.Flags().StringVar(&deployRepo, "repo", "", "Git repository URL")
	deployGitCmd.Flags().StringVar(&deployBranch, "branch", "main", "Git branch")
	deployGitCmd.Flags().StringVar(&deployPath, "path", "docker-compose.yml", "Compose file path in repo")
	addGitAuthFlags(deployGitCmd, &deployGitAuth)

	deployK8sCmd := &cobra.Command{
		Use:     "deploy-k8s",
		Short:   "Deploy a Kubernetes manifest stack from a local file",
		Example: `  portainerctl stack deploy-k8s --name myapp --env 4 --file manifest.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployFile == "" {
				return fmt.Errorf("--name, --env, and --file are required")
			}
			data, err := os.ReadFile(deployFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"StackName":        deployName,
				"StackFileContent": string(data),
				"Namespace":        "default",
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/kubernetes/string?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployK8sCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployK8sCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target Kubernetes environment ID")
	deployK8sCmd.Flags().StringVar(&deployFile, "file", "", "Path to manifest file")

	deployK8sGitCmd := &cobra.Command{
		Use:     "deploy-k8s-git",
		Short:   "Deploy a Kubernetes stack from a Git repository",
		Example: `  portainerctl stack deploy-k8s-git --name myapp --env 4 --repo https://github.com/org/repo --branch main --path manifest.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployEnvID == 0 || deployRepo == "" {
				return fmt.Errorf("--name, --env, and --repo are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			credential, password, passwordSet, err := deployGitAuth.credentials(cmd.InOrStdin())
			if err != nil {
				return err
			}
			authenticated := passwordSet || credential.Username != ""
			if authenticated && password == "" {
				return fmt.Errorf("Git authentication requires a password/token via --git-password-stdin or PORTAINERCTL_GIT_PASSWORD")
			}
			body := map[string]interface{}{
				"StackName":                   deployName,
				"RepositoryURL":               deployRepo,
				"RepositoryReferenceName":     "refs/heads/" + deployBranch,
				"ManifestFile":                deployPath,
				"RepositoryAuthentication":    authenticated,
				"RepositoryUsername":          credential.Username,
				"RepositoryPassword":          password,
				"RepositoryAuthorizationType": credential.AuthorizationType,
				"Namespace":                   "default",
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/create/kubernetes/repository?endpointId=%d", deployEnvID)
			if err := c.Post(path, body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployK8sGitCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployK8sGitCmd.Flags().IntVar(&deployEnvID, "env", 0, "Target Kubernetes environment ID")
	deployK8sGitCmd.Flags().StringVar(&deployRepo, "repo", "", "Git repository URL")
	deployK8sGitCmd.Flags().StringVar(&deployBranch, "branch", "main", "Git branch")
	deployK8sGitCmd.Flags().StringVar(&deployPath, "path", "manifest.yaml", "Manifest file path in repo")
	addGitAuthFlags(deployK8sGitCmd, &deployGitAuth)

	var redeployEnvID int
	var redeployBranch string
	var redeployPrune, redeployRepull bool
	var redeployGitAuth gitAuthOptions
	redeployCmd := &cobra.Command{
		Use:   "redeploy <id>",
		Short: "Redeploy a GitOps-backed stack (pull latest from Git)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var existing stack
			if err := c.Get("/stacks/"+args[0], &existing); err != nil {
				return fmt.Errorf("reading current stack settings: %w", err)
			}
			if existing.GitConfig == nil {
				return fmt.Errorf("stack %s is not backed by Git", args[0])
			}

			credential, password, passwordSet, err := redeployGitAuth.credentials(cmd.InOrStdin())
			if err != nil {
				return err
			}
			authenticated := existing.GitConfig.Authentication != nil
			if authenticated {
				if credential.Username == "" {
					credential.Username = existing.GitConfig.Authentication.Username
				}
				if redeployGitAuth.authType == "" && os.Getenv("PORTAINERCTL_GIT_AUTH_TYPE") == "" {
					credential.AuthorizationType = existing.GitConfig.Authentication.AuthorizationType
				}
			}
			if passwordSet {
				authenticated = true
			}
			if existing.GitConfig.Authentication == nil && passwordSet && password == "" {
				return fmt.Errorf("new Git authentication requires a non-empty password/token")
			}
			if !authenticated && credential.Username != "" && !passwordSet {
				return fmt.Errorf("new Git authentication requires a password/token via --git-password-stdin or PORTAINERCTL_GIT_PASSWORD")
			}

			reference := existing.GitConfig.ReferenceName
			if redeployBranch != "" {
				reference = redeployBranch
				if !strings.HasPrefix(reference, "refs/") {
					reference = "refs/heads/" + reference
				}
			}
			prune := existing.Option != nil && existing.Option.Prune
			if cmd.Flags().Changed("prune") {
				prune = redeployPrune
			}
			endpointID := redeployEnvID
			if endpointID == 0 {
				endpointID = existing.EndpointID
			}
			if endpointID == 0 {
				return fmt.Errorf("--env is required because the stack has no associated environment ID")
			}
			path := fmt.Sprintf("/stacks/%s/git/redeploy?endpointId=%d", args[0], endpointID)
			body := map[string]interface{}{
				"RepositoryReferenceName":     reference,
				"RepositoryAuthentication":    authenticated,
				"RepositoryUsername":          credential.Username,
				"RepositoryPassword":          password,
				"RepositoryAuthorizationType": credential.AuthorizationType,
				"Env":                         existing.Env,
				"Prune":                       prune,
				"RepullImageAndRedeploy":      redeployRepull,
				"PullImage":                   redeployRepull,
				"StackName":                   existing.Name,
			}
			var result interface{}
			if err := c.Put(path, body, &result); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " redeployed from Git.")
			return nil
		},
	}
	redeployCmd.Flags().IntVar(&redeployEnvID, "env", 0, "Environment ID (required for stacks created before v1.18.0)")
	redeployCmd.Flags().StringVar(&redeployBranch, "branch", "", "Repository branch or full ref (default: preserve current ref)")
	redeployCmd.Flags().BoolVar(&redeployPrune, "prune", false, "Prune services no longer referenced (default: preserve current setting)")
	redeployCmd.Flags().BoolVar(&redeployRepull, "repull-image", false, "Re-pull images and force redeployment")
	addGitAuthFlags(redeployCmd, &redeployGitAuth)

	var startEnvID int
	startCmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Start a stopped stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if startEnvID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			path := fmt.Sprintf("/stacks/%s/start?endpointId=%d", args[0], startEnvID)
			if err := c.Post(path, nil, &result); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " started.")
			return nil
		},
	}
	startCmd.Flags().IntVar(&startEnvID, "env", 0, "Environment ID (required)")

	var stopEnvID int
	stopCmd := &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a running stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if stopEnvID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/stacks/%s/stop?endpointId=%d", args[0], stopEnvID)
			if err := c.Post(path, nil, nil); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " stopped.")
			return nil
		},
	}
	stopCmd.Flags().IntVar(&stopEnvID, "env", 0, "Environment ID (required)")

	var deleteEnvID int
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deleteEnvID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/stacks/%s?endpointId=%d", args[0], deleteEnvID)
			if err := c.Delete(path); err != nil {
				return err
			}
			output.Success("Stack " + args[0] + " deleted.")
			return nil
		},
	}
	deleteCmd.Flags().IntVar(&deleteEnvID, "env", 0, "Environment ID (required)")

	imagesStatusCmd := &cobra.Command{
		Use:   "image-status <id>",
		Short: "Get image update status for a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/stacks/"+args[0]+"/images_status", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, getByNameCmd, fileCmd,
		deployComposeCmd, deploySwarmCmd, deployGitCmd,
		deployK8sCmd, deployK8sGitCmd,
		redeployCmd, startCmd, stopCmd, deleteCmd, imagesStatusCmd)
	return cmd
}
