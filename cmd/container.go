package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
)

// Container operations are proxied through /endpoints/{id}/docker/containers/...
// which passes directly to the Docker Engine API.

func containerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manage containers on a Docker/Swarm environment",
	}

	var envID int
	envFlag := func(c *cobra.Command) {
		c.Flags().IntVar(&envID, "env", 0, "Environment ID (required)")
	}

	// list
	var showAll bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List containers on an environment",
		Example: `  portainerctl container list --env 2
  portainerctl container list --env 2 --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			all := "false"
			if showAll {
				all = "true"
			}
			var containers []map[string]interface{}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/json?all=%s", envID, all)
			if err := c.Get(path, &containers); err != nil {
				return err
			}
			rows := [][]string{}
			for _, ct := range containers {
				id := fmt.Sprintf("%v", ct["Id"])
				if len(id) > 12 {
					id = id[:12]
				}
				names := ""
				if ns, ok := ct["Names"].([]interface{}); ok && len(ns) > 0 {
					names = fmt.Sprintf("%v", ns[0])
				}
				image := fmt.Sprintf("%v", ct["Image"])
				state := fmt.Sprintf("%v", ct["State"])
				status := fmt.Sprintf("%v", ct["Status"])
				rows = append(rows, []string{id, names, image, state, status})
			}
			output.Table([]string{"ID", "NAME", "IMAGE", "STATE", "STATUS"}, rows)
			return nil
		},
	}
	listCmd.Flags().BoolVar(&showAll, "all", false, "Show all containers (including stopped)")
	envFlag(listCmd)

	// inspect
	inspectCmd := &cobra.Command{
		Use:   "inspect <container-id>",
		Short: "Inspect a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/json", envID, args[0])
			if err := c.Get(path, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	envFlag(inspectCmd)

	// logs
	var logsTail int
	var logsTimestamps bool
	logsCmd := &cobra.Command{
		Use:   "logs <container-id>",
		Short: "Fetch container logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			tail := "100"
			if logsTail > 0 {
				tail = strconv.Itoa(logsTail)
			}
			timestamps := "false"
			if logsTimestamps {
				timestamps = "true"
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/logs?stdout=true&stderr=true&tail=%s&timestamps=%s",
				envID, args[0], tail, timestamps)
			data, err := c.RawGet(path)
			if err != nil {
				return err
			}
			// Docker log stream has 8-byte headers per frame; strip them for readability
			fmt.Print(stripDockerLogHeaders(data))
			return nil
		},
	}
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show from the end")
	logsCmd.Flags().BoolVar(&logsTimestamps, "timestamps", false, "Show timestamps")
	envFlag(logsCmd)

	// start
	startCmd := &cobra.Command{
		Use:   "start <container-id>",
		Short: "Start a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/start", envID, args[0])
			if err := c.Post(path, nil, nil); err != nil {
				return err
			}
			output.Success("Container " + args[0] + " started.")
			return nil
		},
	}
	envFlag(startCmd)

	// stop
	var stopTimeout int
	stopCmd := &cobra.Command{
		Use:   "stop <container-id>",
		Short: "Stop a running container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/stop?t=%d", envID, args[0], stopTimeout)
			if err := c.Post(path, nil, nil); err != nil {
				return err
			}
			output.Success("Container " + args[0] + " stopped.")
			return nil
		},
	}
	stopCmd.Flags().IntVar(&stopTimeout, "timeout", 10, "Seconds to wait before killing")
	envFlag(stopCmd)

	// restart
	restartCmd := &cobra.Command{
		Use:   "restart <container-id>",
		Short: "Restart a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/restart", envID, args[0])
			if err := c.Post(path, nil, nil); err != nil {
				return err
			}
			output.Success("Container " + args[0] + " restarted.")
			return nil
		},
	}
	envFlag(restartCmd)

	// kill
	var killSignal string
	killCmd := &cobra.Command{
		Use:   "kill <container-id>",
		Short: "Kill a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/kill?signal=%s", envID, args[0], killSignal)
			if err := c.Post(path, nil, nil); err != nil {
				return err
			}
			output.Success("Signal " + killSignal + " sent to container " + args[0])
			return nil
		},
	}
	killCmd.Flags().StringVar(&killSignal, "signal", "SIGKILL", "Signal to send")
	envFlag(killCmd)

	// remove
	var forceRemove, removeVolumes bool
	removeCmd := &cobra.Command{
		Use:   "remove <container-id>",
		Short: "Remove a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s?force=%v&v=%v",
				envID, args[0], forceRemove, removeVolumes)
			if err := c.Delete(path); err != nil {
				return err
			}
			output.Success("Container " + args[0] + " removed.")
			return nil
		},
	}
	removeCmd.Flags().BoolVar(&forceRemove, "force", false, "Force removal of a running container")
	removeCmd.Flags().BoolVar(&removeVolumes, "volumes", false, "Remove associated volumes")
	envFlag(removeCmd)

	// stats
	statsCmd := &cobra.Command{
		Use:   "stats <container-id>",
		Short: "Get container resource usage statistics (single snapshot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/stats?stream=false", envID, args[0])
			var result interface{}
			if err := c.Get(path, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	envFlag(statsCmd)

	// top
	topCmd := &cobra.Command{
		Use:   "top <container-id>",
		Short: "List processes running in a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/endpoints/%d/docker/containers/%s/top", envID, args[0])
			var result map[string]interface{}
			if err := c.Get(path, &result); err != nil {
				return err
			}
			if titles, ok := result["Titles"].([]interface{}); ok {
				headers := make([]string, len(titles))
				for i, t := range titles {
					headers[i] = fmt.Sprintf("%v", t)
				}
				rows := [][]string{}
				if procs, ok := result["Processes"].([]interface{}); ok {
					for _, proc := range procs {
						if p, ok := proc.([]interface{}); ok {
							row := make([]string, len(p))
							for i, v := range p {
								row[i] = fmt.Sprintf("%v", v)
							}
							rows = append(rows, row)
						}
					}
				}
				output.Table(headers, rows)
			} else {
				output.JSON(result)
			}
			return nil
		},
	}
	envFlag(topCmd)

	// image-status  (Portainer-specific, not raw Docker)
	imageStatusCmd := &cobra.Command{
		Use:   "image-status <container-id>",
		Short: "Check if a newer image is available for a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("--env is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/docker/%d/containers/%s/image_status", envID, args[0])
			var result interface{}
			if err := c.Get(path, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	envFlag(imageStatusCmd)

	cmd.AddCommand(listCmd, inspectCmd, logsCmd, startCmd, stopCmd,
		restartCmd, killCmd, removeCmd, statsCmd, topCmd, imageStatusCmd)
	return cmd
}

// stripDockerLogHeaders removes the 8-byte multiplexed stream headers from Docker log output.
func stripDockerLogHeaders(data []byte) string {
	result := []byte{}
	i := 0
	for i < len(data) {
		if i+8 > len(data) {
			result = append(result, data[i:]...)
			break
		}
		// bytes 4-7 are the frame size in big-endian
		size := int(data[i+4])<<24 | int(data[i+5])<<16 | int(data[i+6])<<8 | int(data[i+7])
		i += 8
		if i+size > len(data) {
			size = len(data) - i
		}
		result = append(result, data[i:i+size]...)
		i += size
	}
	return string(result)
}
