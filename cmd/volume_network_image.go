package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
)

func volumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage Docker volumes on an environment",
	}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result map[string]interface{}
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/volumes", envID), &result); err != nil { return err }
			volumes, _ := result["Volumes"].([]interface{})
			rows := [][]string{}
			for _, v := range volumes {
				vol, _ := v.(map[string]interface{})
				rows = append(rows, []string{
					fmt.Sprintf("%v", vol["Name"]),
					fmt.Sprintf("%v", vol["Driver"]),
					fmt.Sprintf("%v", vol["Mountpoint"]),
				})
			}
			output.Table([]string{"NAME", "DRIVER", "MOUNTPOINT"}, rows)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	inspectCmd := &cobra.Command{
		Use: "inspect <name>", Short: "Inspect a volume", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/volumes/%s", envID, args[0]), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	inspectCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	var volDriver string
	createCmd := &cobra.Command{
		Use: "create <name>", Short: "Create a volume", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"Name": args[0], "Driver": volDriver}
			var result interface{}
			if err := c.Post(fmt.Sprintf("/endpoints/%d/docker/volumes/create", envID), body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	createCmd.Flags().StringVar(&volDriver, "driver", "local", "Volume driver")

	removeCmd := &cobra.Command{
		Use: "remove <name>", Short: "Remove a volume", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Delete(fmt.Sprintf("/endpoints/%d/docker/volumes/%s", envID, args[0])); err != nil { return err }
			output.Success("Volume " + args[0] + " removed.")
			return nil
		},
	}
	removeCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd, inspectCmd, createCmd, removeCmd)
	return cmd
}

func networkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage Docker networks on an environment",
	}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var networks []map[string]interface{}
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/networks", envID), &networks); err != nil { return err }
			rows := [][]string{}
			for _, n := range networks {
				id := fmt.Sprintf("%v", n["Id"])
				if len(id) > 12 { id = id[:12] }
				rows = append(rows, []string{id, fmt.Sprintf("%v", n["Name"]), fmt.Sprintf("%v", n["Driver"]), fmt.Sprintf("%v", n["Scope"])})
			}
			output.Table([]string{"ID", "NAME", "DRIVER", "SCOPE"}, rows)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	inspectCmd := &cobra.Command{
		Use: "inspect <id>", Short: "Inspect a network", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/networks/%s", envID, args[0]), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	inspectCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	var netDriver string
	createCmd := &cobra.Command{
		Use: "create <name>", Short: "Create a network", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			body := map[string]interface{}{"Name": args[0], "Driver": netDriver}
			var result interface{}
			if err := c.Post(fmt.Sprintf("/endpoints/%d/docker/networks/create", envID), body, &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	createCmd.Flags().StringVar(&netDriver, "driver", "bridge", "Network driver")

	removeCmd := &cobra.Command{
		Use: "remove <id>", Short: "Remove a network", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			if err := c.Delete(fmt.Sprintf("/endpoints/%d/docker/networks/%s", envID, args[0])); err != nil { return err }
			output.Success("Network " + args[0] + " removed.")
			return nil
		},
	}
	removeCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd, inspectCmd, createCmd, removeCmd)
	return cmd
}

func imageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage Docker images on an environment",
	}
	var envID int

	listCmd := &cobra.Command{
		Use: "list", Short: "List images",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var images []map[string]interface{}
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/images/json", envID), &images); err != nil { return err }
			rows := [][]string{}
			for _, img := range images {
				id := fmt.Sprintf("%v", img["Id"])
				if len(id) > 19 { id = id[7:19] } // strip sha256: prefix and truncate
				tags := ""
				if rt, ok := img["RepoTags"].([]interface{}); ok && len(rt) > 0 {
					tags = fmt.Sprintf("%v", rt[0])
				}
				size := fmt.Sprintf("%v", img["Size"])
				rows = append(rows, []string{id, tags, size})
			}
			output.Table([]string{"ID", "TAGS", "SIZE"}, rows)
			return nil
		},
	}
	listCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	inspectCmd := &cobra.Command{
		Use: "inspect <id>", Short: "Inspect an image", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/endpoints/%d/docker/images/%s/json", envID, args[0]), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	inspectCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	var pullImage string
	pullCmd := &cobra.Command{
		Use: "pull", Short: "Pull an image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			if pullImage == "" { return fmt.Errorf("--image is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/endpoints/%d/docker/images/create?fromImage=%s", envID, pullImage)
			data, err := c.RawGet(path)
			if err != nil { return err }
			fmt.Println(string(data))
			return nil
		},
	}
	pullCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	pullCmd.Flags().StringVar(&pullImage, "image", "", "Image to pull (e.g. nginx:latest)")

	var forceRemove bool
	removeCmd := &cobra.Command{
		Use: "remove <id>", Short: "Remove an image", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			path := fmt.Sprintf("/endpoints/%d/docker/images/%s?force=%v", envID, args[0], forceRemove)
			if err := c.Delete(path); err != nil { return err }
			output.Success("Image " + args[0] + " removed.")
			return nil
		},
	}
	removeCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")
	removeCmd.Flags().BoolVar(&forceRemove, "force", false, "Force removal")

	// Portainer-specific image update check (docker/{environmentId}/images)
	updateCheckCmd := &cobra.Command{
		Use: "update-check", Short: "Check for image updates across an environment (Portainer-specific)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 { return fmt.Errorf("--env is required") }
			c, err := client.MustClient(); if err != nil { return err }
			var result interface{}
			if err := c.Get(fmt.Sprintf("/docker/%d/images", envID), &result); err != nil { return err }
			output.JSON(result)
			return nil
		},
	}
	updateCheckCmd.Flags().IntVar(&envID, "env", 0, "Environment ID")

	cmd.AddCommand(listCmd, inspectCmd, pullCmd, removeCmd, updateCheckCmd)
	return cmd
}
