package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
)

// parseJSON is a shared helper used across command files
func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

func licenseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Manage Portainer BE licenses",
	}

	infoCmd := &cobra.Command{
		Use: "info", Short: "Show license information",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/licenses/info", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List all licenses",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/licenses", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	var licenseKey string
	addCmd := &cobra.Command{
		Use: "add", Short: "Add a license key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if licenseKey == "" { return fmt.Errorf("--key is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"LicenseKeys": []string{licenseKey}}
			var result interface{}
			if err := c.Post("/licenses/add", body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	addCmd.Flags().StringVar(&licenseKey, "key", "", "License key")

	var removeKeys []string
	removeCmd := &cobra.Command{
		Use: "remove", Short: "Remove license keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(removeKeys) == 0 { return fmt.Errorf("--keys is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"LicenseKeys": removeKeys}
			if err := c.Post("/licenses/remove", body, nil); err != nil { return err }
			output.Success("License(s) removed.")
			return nil
		},
	}
	removeCmd.Flags().StringSliceVar(&removeKeys, "keys", nil, "License keys to remove")

	cmd.AddCommand(infoCmd, listCmd, addCmd, removeCmd)
	return cmd
}

func backupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage Portainer backups",
	}

	var backupPass, backupOutput string
	createCmd := &cobra.Command{
		Use: "create", Short: "Create a backup of Portainer data",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"Password": backupPass}
			data, err := c.RawGet("/backup")
			if err != nil {
				// POST for backup
				_ = c.Post("/backup", body, nil)
				output.Success("Backup initiated.")
				return nil
			}
			outFile := backupOutput
			if outFile == "" { outFile = "portainer-backup.tar.gz" }
			if err := os.WriteFile(outFile, data, 0600); err != nil { return err }
			output.Success("Backup saved to " + outFile)
			return nil
		},
	}
	createCmd.Flags().StringVar(&backupPass, "password", "", "Password to encrypt the backup")
	createCmd.Flags().StringVar(&backupOutput, "output", "", "Output file path (default: portainer-backup.tar.gz)")

	// S3 settings
	s3SettingsCmd := &cobra.Command{
		Use: "s3-settings", Short: "Get S3 backup settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/backup/s3/settings", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	s3StatusCmd := &cobra.Command{
		Use: "s3-status", Short: "Get S3 backup status",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/backup/s3/status", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	s3ExecuteCmd := &cobra.Command{
		Use: "s3-execute", Short: "Execute an S3 backup now",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Post("/backup/s3/execute", nil, nil); err != nil { return err }
			output.Success("S3 backup initiated.")
			return nil
		},
	}

	cmd.AddCommand(createCmd, s3SettingsCmd, s3StatusCmd, s3ExecuteCmd)
	return cmd
}

func settingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage Portainer server settings",
	}

	getCmd := &cobra.Command{
		Use: "get", Short: "Get current settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/settings", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	publicCmd := &cobra.Command{
		Use: "public", Short: "Get public settings (no auth required)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/settings/public", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	var patchJSON string
	updateCmd := &cobra.Command{
		Use: "update", Short: "Update settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if patchJSON == "" { return fmt.Errorf("--patch is required (JSON)") }
			c, err := client.MustClient(); if err != nil { return err }
			var body, result interface{}
			if err := parseJSON(patchJSON, &body); err != nil { return err }
			if err := c.Put("/settings", body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	updateCmd.Flags().StringVar(&patchJSON, "patch", "", "JSON patch body")

	sslCmd := &cobra.Command{
		Use: "ssl", Short: "Get SSL certificate info",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/ssl", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	experimentalCmd := &cobra.Command{
		Use: "experimental", Short: "Get experimental feature flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/settings/experimental", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(getCmd, publicCmd, updateCmd, sslCmd, experimentalCmd)
	return cmd
}

func systemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System information and status",
	}

	statusCmd := &cobra.Command{
		Use: "status", Short: "Get Portainer system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/system/status", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	infoCmd := &cobra.Command{
		Use: "info", Short: "Get system info",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/system/info", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	versionCmd := &cobra.Command{
		Use: "version", Short: "Get Portainer version",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result map[string]interface{}
			if err := c.Get("/system/version", &result); err != nil { return err }
			if v, ok := result["ServerVersion"].(string); ok {
				fmt.Println(v)
			} else {
				output.JSON(result)
			}
			return nil
		},
	}

	nodesCmd := &cobra.Command{
		Use: "nodes", Short: "Get licensed node count",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/system/nodes", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(statusCmd, infoCmd, versionCmd, nodesCmd)
	return cmd
}

func userActivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity",
		Short: "View user activity and authentication logs",
	}

	authLogsCmd := &cobra.Command{
		Use: "auth-logs", Short: "View authentication logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/useractivity/authlogs", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	logsCmd := &cobra.Command{
		Use: "logs", Short: "View user activity logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/useractivity/logs", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	authLogsCsvCmd := &cobra.Command{
		Use: "auth-logs-csv", Short: "Download authentication logs as CSV",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			data, err := c.RawGet("/useractivity/authlogs.csv")
			if err != nil { return err }
			fmt.Print(string(data))
			return nil
		},
	}

	logsCsvCmd := &cobra.Command{
		Use: "logs-csv", Short: "Download user activity logs as CSV",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			data, err := c.RawGet("/useractivity/logs.csv")
			if err != nil { return err }
			fmt.Print(string(data))
			return nil
		},
	}

	cmd.AddCommand(authLogsCmd, logsCmd, authLogsCsvCmd, logsCsvCmd)
	return cmd
}

func observabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observability",
		Short: "Manage alerting rules and settings",
	}

	alertsCmd := &cobra.Command{
		Use: "alerts", Short: "List active alerts",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/observability/alerting/alerts", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	rulesCmd := &cobra.Command{
		Use: "rules", Short: "List alerting rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/observability/alerting/rules", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	settingsListCmd := &cobra.Command{
		Use: "settings", Short: "List alerting settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/observability/alerting/settings", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	connectivityCmd := &cobra.Command{
		Use: "connectivity", Short: "Check alerting connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/observability/alerting/connectivity", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(alertsCmd, rulesCmd, settingsListCmd, connectivityCmd)
	return cmd
}

func customTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage custom app templates",
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List custom templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/custom_templates", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use: "get <id>", Short: "Get a custom template", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/custom_templates/"+args[0], &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	fileCmd := &cobra.Command{
		Use: "file <id>", Short: "Get the compose file for a custom template", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result map[string]interface{}
			if err := c.Get("/custom_templates/"+args[0]+"/file", &result); err != nil { return err }
			if content, ok := result["FileContent"].(string); ok {
				fmt.Println(content)
			} else {
				output.JSON(result)
			}
			return nil
		},
	}

	var tmplTitle, tmplDesc, tmplFile string
	var tmplType int
	createCmd := &cobra.Command{
		Use: "create", Short: "Create a custom template from a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tmplTitle == "" || tmplFile == "" { return fmt.Errorf("--title and --file are required") }
			data, err := os.ReadFile(tmplFile)
			if err != nil { return fmt.Errorf("reading file: %w", err) }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{
				"Title":            tmplTitle,
				"Description":      tmplDesc,
				"Type":             tmplType,
				"FileContent":      string(data),
				"Variables":        []interface{}{},
			}
			var result interface{}
			if err := c.Post("/custom_templates/create/string", body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&tmplTitle, "title", "", "Template title")
	createCmd.Flags().StringVar(&tmplDesc, "description", "", "Template description")
	createCmd.Flags().StringVar(&tmplFile, "file", "", "Compose file path")
	createCmd.Flags().IntVar(&tmplType, "type", 2, "Template type: 1=container, 2=compose, 3=swarm")

	deleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a custom template", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Delete("/custom_templates/" + args[0]); err != nil { return err }
			output.Success("Template " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, fileCmd, createCmd, deleteCmd)
	return cmd
}

func gitopsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitops",
		Short: "GitOps repository utilities",
	}

	var repoURL, repoRef, repoUser, repoPass, repoFile string
	var repoTLSSkip bool

	refsCmd := &cobra.Command{
		Use: "refs", Short: "List branches and tags in a Git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoURL == "" { return fmt.Errorf("--url is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{
				"RepositoryURL":            repoURL,
				"Username":                 repoUser,
				"Password":                 repoPass,
				"TLSSkipVerify":            repoTLSSkip,
			}
			var result interface{}
			if err := c.Post("/gitops/repo/refs", body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	refsCmd.Flags().StringVar(&repoURL, "url", "", "Repository URL")
	refsCmd.Flags().StringVar(&repoUser, "user", "", "Username")
	refsCmd.Flags().StringVar(&repoPass, "pass", "", "Password or token")
	refsCmd.Flags().BoolVar(&repoTLSSkip, "tls-skip", false, "Skip TLS verification")

	searchCmd := &cobra.Command{
		Use: "search", Short: "Search for files in a Git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoURL == "" { return fmt.Errorf("--url is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{
				"RepositoryURL":            repoURL,
				"RepositoryReferenceName":  repoRef,
				"Username":                 repoUser,
				"Password":                 repoPass,
				"TLSSkipVerify":            repoTLSSkip,
			}
			var result interface{}
			if err := c.Post("/gitops/repo/files/search", body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	searchCmd.Flags().StringVar(&repoURL, "url", "", "Repository URL")
	searchCmd.Flags().StringVar(&repoRef, "ref", "refs/heads/main", "Branch/tag reference")
	searchCmd.Flags().StringVar(&repoUser, "user", "", "Username")
	searchCmd.Flags().StringVar(&repoPass, "pass", "", "Password or token")
	searchCmd.Flags().BoolVar(&repoTLSSkip, "tls-skip", false, "Skip TLS verification")

	previewCmd := &cobra.Command{
		Use: "preview", Short: "Preview a file from a Git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoURL == "" || repoFile == "" { return fmt.Errorf("--url and --file are required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{
				"RepositoryURL":            repoURL,
				"RepositoryReferenceName":  repoRef,
				"TargetFile":               repoFile,
				"Username":                 repoUser,
				"Password":                 repoPass,
				"TLSSkipVerify":            repoTLSSkip,
			}
			var result interface{}
			if err := c.Post("/gitops/repo/file/preview", body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	previewCmd.Flags().StringVar(&repoURL, "url", "", "Repository URL")
	previewCmd.Flags().StringVar(&repoRef, "ref", "refs/heads/main", "Branch/tag reference")
	previewCmd.Flags().StringVar(&repoFile, "file", "", "File path in repository")
	previewCmd.Flags().StringVar(&repoUser, "user", "", "Username")
	previewCmd.Flags().StringVar(&repoPass, "pass", "", "Password or token")
	previewCmd.Flags().BoolVar(&repoTLSSkip, "tls-skip", false, "Skip TLS verification")

	cmd.AddCommand(refsCmd, searchCmd, previewCmd)
	return cmd
}

func cloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Manage KaaS (cloud-provisioned Kubernetes) credentials and clusters",
	}

	credsCmd := &cobra.Command{
		Use: "credentials", Short: "List cloud credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/cloud/credentials", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	getCredCmd := &cobra.Command{
		Use: "credential <id>", Short: "Get a cloud credential", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/cloud/credentials/"+args[0], &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	deleteCredCmd := &cobra.Command{
		Use: "delete-credential <id>", Short: "Delete a cloud credential", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Delete("/cloud/credentials/" + args[0]); err != nil { return err }
			output.Success("Cloud credential " + args[0] + " deleted.")
			return nil
		},
	}

	var providerID int
	infoCmd := &cobra.Command{
		Use: "info <provider>", Short: "Get provider info (e.g. amazon, azure, gke)", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/cloud/%s/info", args[0])
			if providerID > 0 {
				path += fmt.Sprintf("?credentialId=%d", providerID)
			}
			var result interface{}
			if err := c.Get(path, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	infoCmd.Flags().IntVar(&providerID, "credential", 0, "Credential ID")

	gitCredsCmd := &cobra.Command{
		Use: "git-credentials", Short: "List cloud Git credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/cloud/gitcredentials", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(credsCmd, getCredCmd, deleteCredCmd, infoCmd, gitCredsCmd)
	return cmd
}

func policyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage Portainer governance policies",
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List all policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/policies", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use: "get <id>", Short: "Get a policy", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/policies/"+args[0], &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	templatesCmd := &cobra.Command{
		Use: "templates", Short: "List policy templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/policies/templates", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	metadataCmd := &cobra.Command{
		Use: "metadata", Short: "Get policy metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get("/policies/metadata", &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a policy", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Delete("/policies/" + args[0]); err != nil { return err }
			output.Success("Policy " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, templatesCmd, metadataCmd, deleteCmd)
	return cmd
}

func supportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "support",
		Short: "Download support bundles and debug logs",
	}

	downloadCmd := &cobra.Command{
		Use: "download", Short: "Download a support bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient(); if err != nil { return err }
			data, err := c.RawGet("/support/download")
			if err != nil { return err }
			outFile := "portainer-support-bundle.zip"
			if err := os.WriteFile(outFile, data, 0600); err != nil { return err }
			output.Success("Support bundle saved to " + outFile)
			return nil
		},
	}

	debugLogCmd := &cobra.Command{
		Use: "debug-log <enable|disable>", Short: "Enable or disable debug logging", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val := args[0]
			if val != "enable" && val != "disable" {
				return fmt.Errorf("argument must be 'enable' or 'disable'")
			}
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"enable": val == "enable"}
			if err := c.Post("/support/debug_log", body, nil); err != nil { return err }
			output.Success(fmt.Sprintf("Debug logging %sd.", val))
			return nil
		},
	}

	cmd.AddCommand(downloadCmd, debugLogCmd)
	return cmd
}
