package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/portainer/portainerctl/internal/output"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portainerctl",
	Short: "portainerctl — CLI for Portainer Business Edition 2.39",
	Long: `portainerctl is a command-line interface for Portainer Business Edition 2.39.1.

It covers the full Portainer BE API surface: environments, stacks, containers,
Kubernetes workloads, users, teams, RBAC, edge compute, registries, GitOps,
webhooks, backups, licensing, and observability.

Output format (applies to all commands):
  -o table   Human-readable table (default)
  -o json    JSON
  -o yaml    YAML

Examples:
  portainerctl env list
  portainerctl env list -o json
  portainerctl env list -o yaml
  portainerctl stack get 5 -o json | jq '.GitConfig.URL'
  portainerctl kubectl --env 4 -- get pods -n default

Configuration:
  portainerctl config add-context --name prod --url https://portainer.example.com --token <pat>
  portainerctl config use-context prod

Environment variable overrides:
  PORTAINERCTL_URL      Portainer server URL
  PORTAINERCTL_TOKEN    API token (PAT)
  PORTAINERCTL_INSECURE=true  Skip TLS verification
  PORTAINERCTL_GIT_USERNAME   Git repository username
  PORTAINERCTL_GIT_PASSWORD   Git repository password/token
  PORTAINERCTL_GIT_AUTH_TYPE  Git authorization type: basic or token`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		f := strings.ToLower(output.Format)
		if f != "table" && f != "json" && f != "yaml" && f != "yml" {
			return fmt.Errorf("invalid output format %q — must be table, json, or yaml", output.Format)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Persistent flag: available on every subcommand
	rootCmd.PersistentFlags().StringVarP(
		&output.Format, "output", "o", "table",
		`Output format: table (default), json, yaml`,
	)

	rootCmd.AddCommand(
		configCmd(),
		envCmd(),
		envGroupCmd(),
		stackCmd(),
		edgeStackCmd(),
		edgeGroupCmd(),
		edgeJobCmd(),
		edgeConfigCmd(),
		edgeUpdateCmd(),
		containerCmd(),
		volumeCmd(),
		networkCmd(),
		imageCmd(),
		kubeCmd(),
		dockerProxyCmd(),
		helmCmd(),
		userCmd(),
		teamCmd(),
		teamMembershipCmd(),
		registryCmd(),
		webhookCmd(),
		resourceControlCmd(),
		roleCmd(),
		tagCmd(),
		licenseCmd(),
		backupCmd(),
		settingsCmd(),
		systemCmd(),
		userActivityCmd(),
		observabilityCmd(),
		customTemplateCmd(),
		gitopsCmd(),
		cloudCmd(),
		policyCmd(),
		supportCmd(),
	)
}
