package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portainerctl",
	Short: "portainerctl — CLI for Portainer Business Edition 2.39",
	Long: `portainerctl is a command-line interface for Portainer Business Edition 2.39.1.

It covers the full Portainer BE API surface: environments, stacks, containers,
Kubernetes workloads, users, teams, RBAC, edge compute, registries, GitOps,
webhooks, backups, licensing, and observability.

Kubernetes passthrough:
  portainerctl kubectl --env <id> -- get pods -n default

Docker passthrough:
  portainerctl docker --env <id> -- ps -a

Configuration:
  portainerctl config add-context --name prod --url https://portainer.example.com --token <pat>
  portainerctl config use-context prod

Environment variable overrides:
  PORTAINERCTL_URL     Portainer server URL
  PORTAINERCTL_TOKEN   API token (PAT)
  PORTAINERCTL_INSECURE=true  Skip TLS verification`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
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
