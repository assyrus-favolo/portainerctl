package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
)

// portaineree.Registry - fields used in table display
type registry struct {
	ID             int    `json:"Id"`
	Name           string `json:"Name"`
	URL            string `json:"URL"`
	Type           int    `json:"Type"`
	Authentication bool   `json:"Authentication"`
	Username       string `json:"Username"`
}

// portainer.Webhook - spec fields: Id, Token, ResourceId, EndpointId, Type
type webhook struct {
	ID         int    `json:"Id"`
	Token      string `json:"Token"`
	ResourceID string `json:"ResourceId"` // spec uses ResourceId (lowercase d)
	EndpointID int    `json:"EndpointId"`
	Type       int    `json:"Type"`
}

// portainer.Tag - spec uses "ID" (uppercase) not "Id"
type tag struct {
	ID   int    `json:"ID"`
	Name string `json:"Name"`
}

func registryTypeLabel(t int) string {
	switch t {
	case 1:
		return "quay"
	case 2:
		return "azure"
	case 3:
		return "custom"
	case 4:
		return "gitlab"
	case 5:
		return "proget"
	case 6:
		return "dockerhub"
	case 7:
		return "ecr"
	default:
		return strconv.Itoa(t)
	}
}

func registryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage container registries",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all registries",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var registries []registry
			if err := c.Get("/registries", &registries); err != nil {
				return err
			}
			rows := [][]string{}
			for _, r := range registries {
				auth := "no"
				if r.Authentication {
					auth = "yes"
				}
				rows = append(rows, []string{
					strconv.Itoa(r.ID),
					r.Name,
					r.URL,
					registryTypeLabel(r.Type),
					auth,
					r.Username,
				})
			}
			output.Table([]string{"ID", "NAME", "URL", "TYPE", "AUTH", "USERNAME"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get registry details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/registries/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var regName, regURL, regUser, regPass string
	var regType int
	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a registry",
		Example: `  portainerctl registry create --name myregistry --url registry.example.com --user admin --pass secret --type 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if regName == "" || regURL == "" {
				return fmt.Errorf("--name and --url are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":           regName,
				"URL":            regURL,
				"Type":           regType,
				"Authentication": regUser != "",
				"Username":       regUser,
				"Password":       regPass,
			}
			var result interface{}
			if err := c.Post("/registries", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&regName, "name", "", "Registry name")
	createCmd.Flags().StringVar(&regURL, "url", "", "Registry URL")
	createCmd.Flags().StringVar(&regUser, "user", "", "Username")
	createCmd.Flags().StringVar(&regPass, "pass", "", "Password")
	createCmd.Flags().IntVar(&regType, "type", 3, "Registry type: 1=quay, 2=azure, 3=custom, 4=gitlab, 5=proget, 6=dockerhub, 7=ecr")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/registries/" + args[0]); err != nil {
				return err
			}
			output.Success("Registry " + args[0] + " deleted.")
			return nil
		},
	}

	reposCmd := &cobra.Command{
		Use:   "repositories <id> <repo-name>",
		Short: "List tags for a repository in a registry",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get(fmt.Sprintf("/registries/%s/repositories/%s/tags", args[0], args[1]), &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var pingURL, pingUser, pingPass string
	pingCmd := &cobra.Command{
		Use:   "ping",
		Short: "Test connectivity to a registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pingURL == "" {
				return fmt.Errorf("--url is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{"URL": pingURL, "Username": pingUser, "Password": pingPass}
			if err := c.Post("/registries/ping", body, nil); err != nil {
				return err
			}
			output.Success("Registry is reachable.")
			return nil
		},
	}
	pingCmd.Flags().StringVar(&pingURL, "url", "", "Registry URL")
	pingCmd.Flags().StringVar(&pingUser, "user", "", "Username")
	pingCmd.Flags().StringVar(&pingPass, "pass", "", "Password")

	cmd.AddCommand(listCmd, getCmd, createCmd, deleteCmd, reposCmd, pingCmd)
	return cmd
}

func webhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var webhooks []webhook
			if err := c.Get("/webhooks", &webhooks); err != nil {
				return err
			}
			rows := [][]string{}
			for _, w := range webhooks {
				rows = append(rows, []string{
					strconv.Itoa(w.ID),
					w.Token,
					w.ResourceID,
					strconv.Itoa(w.EndpointID),
					strconv.Itoa(w.Type),
				})
			}
			output.Table([]string{"ID", "TOKEN", "RESOURCE", "ENV", "TYPE"}, rows)
			return nil
		},
	}

	var whResource string
	var whEnvID, whType int
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			if whResource == "" || whEnvID == 0 {
				return fmt.Errorf("--resource and --env are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"ResourceID":  whResource,
				"EndpointID":  whEnvID,
				"WebhookType": whType,
			}
			var result interface{}
			if err := c.Post("/webhooks", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&whResource, "resource", "", "Resource ID (e.g. service ID)")
	createCmd.Flags().IntVar(&whEnvID, "env", 0, "Environment ID")
	createCmd.Flags().IntVar(&whType, "type", 1, "Webhook type: 1=service")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/webhooks/" + args[0]); err != nil {
				return err
			}
			output.Success("Webhook " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, createCmd, deleteCmd)
	return cmd
}

func tagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage environment tags",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var tags []tag
			if err := c.Get("/tags", &tags); err != nil {
				return err
			}
			rows := [][]string{}
			for _, t := range tags {
				rows = append(rows, []string{strconv.Itoa(t.ID), t.Name})
			}
			output.Table([]string{"ID", "NAME"}, rows)
			return nil
		},
	}

	var tagName string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tagName == "" {
				return fmt.Errorf("--name is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Post("/tags", map[string]interface{}{"Name": tagName}, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&tagName, "name", "", "Tag name")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/tags/" + args[0]); err != nil {
				return err
			}
			output.Success("Tag " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, createCmd, deleteCmd)
	return cmd
}
