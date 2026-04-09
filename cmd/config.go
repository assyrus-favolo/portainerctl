package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/config"
	"github.com/portainer/portainerctl/internal/output"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage portainerctl configuration and contexts",
	}

	// add-context
	var ctxName, ctxURL, ctxToken string
	var ctxInsecure bool
	addCtx := &cobra.Command{
		Use:   "add-context",
		Short: "Add or update a named context",
		Example: `  portainerctl config add-context --name prod --url https://portainer.example.com --token pt_abc123
  portainerctl config add-context --name lab --url https://lab:9443 --token pt_xyz --insecure`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ctxName == "" || ctxURL == "" || ctxToken == "" {
				return fmt.Errorf("--name, --url, and --token are required")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.AddContext(config.Context{
				Name:     ctxName,
				URL:      ctxURL,
				Token:    ctxToken,
				Insecure: ctxInsecure,
			})
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = ctxName
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Context %q saved.", ctxName))
			return nil
		},
	}
	addCtx.Flags().StringVar(&ctxName, "name", "", "Context name")
	addCtx.Flags().StringVar(&ctxURL, "url", "", "Portainer server URL (e.g. https://portainer.example.com)")
	addCtx.Flags().StringVar(&ctxToken, "token", "", "API token (PAT)")
	addCtx.Flags().BoolVar(&ctxInsecure, "insecure", false, "Skip TLS certificate verification")

	// use-context
	useCtx := &cobra.Command{
		Use:   "use-context <name>",
		Short: "Switch the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			found := false
			for _, ctx := range cfg.Contexts {
				if ctx.Name == args[0] {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("context %q not found", args[0])
			}
			cfg.CurrentContext = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Switched to context %q.", args[0]))
			return nil
		},
	}

	// get-contexts
	getCtxs := &cobra.Command{
		Use:   "get-contexts",
		Short: "List all configured contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			rows := [][]string{}
			for _, ctx := range cfg.Contexts {
				current := ""
				if ctx.Name == cfg.CurrentContext {
					current = "*"
				}
				insecure := ""
				if ctx.Insecure {
					insecure = "true"
				}
				rows = append(rows, []string{current, ctx.Name, ctx.URL, insecure})
			}
			output.Table([]string{"", "NAME", "URL", "INSECURE"}, rows)
			return nil
		},
	}

	// delete-context
	delCtx := &cobra.Command{
		Use:   "delete-context <name>",
		Short: "Remove a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if !cfg.RemoveContext(args[0]) {
				return fmt.Errorf("context %q not found", args[0])
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Context %q removed.", args[0]))
			return nil
		},
	}

	// view
	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "Print the path to the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigPath()
			if err != nil {
				return err
			}
			fmt.Println(path)
			_, statErr := os.Stat(path)
			if os.IsNotExist(statErr) {
				fmt.Println("(file does not exist yet)")
			}
			return nil
		},
	}

	cmd.AddCommand(addCtx, useCtx, getCtxs, delCtx, viewCmd)
	return cmd
}
