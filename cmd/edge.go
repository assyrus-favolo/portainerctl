package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
)

// edgegroups.decoratedEdgeGroup — returned by GET /edge_groups
type edgeGroup struct {
	ID      int    `json:"Id"`
	Name    string `json:"Name"`
	Dynamic bool   `json:"Dynamic"`
	EndpointsCount int `json:"EndpointsCount"`
}

// portainer.EdgeJob — returned by GET /edge_jobs
type edgeJob struct {
	ID             int    `json:"Id"`
	Name           string `json:"Name"`
	CronExpression string `json:"CronExpression"`
	Recurring      bool   `json:"Recurring"`
	Created        int64  `json:"Created"`
}

// edgeStackListResponseItem — returned by GET /edge_stacks
type edgeStack struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name"`
	CreationDate int64  `json:"CreationDate"`
	CreatedBy    string `json:"CreatedBy"`
	EntryPoint   string `json:"EntryPoint"`
	DeploymentType int  `json:"DeploymentType"`
}

func edgeStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge-stack",
		Short: "Manage Edge stacks",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all Edge stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var stacks []edgeStack
			if err := c.Get("/edge_stacks", &stacks); err != nil {
				return err
			}
			rows := [][]string{}
			for _, s := range stacks {
				rows = append(rows, []string{
					strconv.Itoa(s.ID),
					s.Name,
					s.CreatedBy,
					s.EntryPoint,
				})
			}
			output.Table([]string{"ID", "NAME", "CREATED BY", "ENTRY POINT"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get Edge stack details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_stacks/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	fileCmd := &cobra.Command{
		Use:   "file <id>",
		Short: "Get the compose file for an Edge stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			// Response: github.com/...edgestacks.stackFileResponse with StackFileContent string
			var result map[string]interface{}
			if err := c.Get("/edge_stacks/"+args[0]+"/file", &result); err != nil {
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
	var deployGroups []int

	deployCmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy an Edge stack from a local file",
		Example: `  portainerctl edge-stack deploy --name myapp --file docker-compose.yml --groups 1,2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployFile == "" {
				return fmt.Errorf("--name and --file are required")
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
				"EdgeGroups":       deployGroups,
				"DeploymentType":   0,
			}
			var result interface{}
			if err := c.Post("/edge_stacks/create/string", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployCmd.Flags().StringVar(&deployFile, "file", "", "Path to compose file")
	deployCmd.Flags().IntSliceVar(&deployGroups, "groups", nil, "Edge group IDs (comma-separated)")

	deployGitCmd := &cobra.Command{
		Use:     "deploy-git",
		Short:   "Deploy an Edge stack from Git",
		Example: `  portainerctl edge-stack deploy-git --name myapp --repo https://github.com/org/repo --branch main --path docker-compose.yml --groups 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deployName == "" || deployRepo == "" {
				return fmt.Errorf("--name and --repo are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":                    deployName,
				"RepositoryURL":           deployRepo,
				"RepositoryReferenceName": "refs/heads/" + deployBranch,
				"FilePathInRepository":    deployPath,
				"EdgeGroups":              deployGroups,
				"DeploymentType":          0,
			}
			var result interface{}
			if err := c.Post("/edge_stacks/create/repository", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	deployGitCmd.Flags().StringVar(&deployName, "name", "", "Stack name")
	deployGitCmd.Flags().StringVar(&deployRepo, "repo", "", "Git repository URL")
	deployGitCmd.Flags().StringVar(&deployBranch, "branch", "main", "Branch")
	deployGitCmd.Flags().StringVar(&deployPath, "path", "docker-compose.yml", "Compose file path in repo")
	deployGitCmd.Flags().IntSliceVar(&deployGroups, "groups", nil, "Edge group IDs")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an Edge stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/edge_stacks/" + args[0]); err != nil {
				return err
			}
			output.Success("Edge stack " + args[0] + " deleted.")
			return nil
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Get deployment status of an Edge stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_stacks/"+args[0]+"/status", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, fileCmd, deployCmd, deployGitCmd, deleteCmd, statusCmd)
	return cmd
}

func edgeGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge-group",
		Short: "Manage Edge groups",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all Edge groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var groups []edgeGroup
			if err := c.Get("/edge_groups", &groups); err != nil {
				return err
			}
			rows := [][]string{}
			for _, g := range groups {
				dynamic := "false"
				if g.Dynamic {
					dynamic = "true"
				}
				rows = append(rows, []string{
					strconv.Itoa(g.ID),
					g.Name,
					dynamic,
					strconv.Itoa(g.EndpointsCount),
				})
			}
			output.Table([]string{"ID", "NAME", "DYNAMIC", "ENDPOINTS"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get Edge group details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_groups/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var groupName string
	var groupDynamic bool
	var groupTags []int
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an Edge group",
		RunE: func(cmd *cobra.Command, args []string) error {
			if groupName == "" {
				return fmt.Errorf("--name is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":    groupName,
				"Dynamic": groupDynamic,
				"TagIDs":  groupTags,
			}
			var result interface{}
			if err := c.Post("/edge_groups", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&groupName, "name", "", "Group name")
	createCmd.Flags().BoolVar(&groupDynamic, "dynamic", false, "Use dynamic membership based on tags")
	createCmd.Flags().IntSliceVar(&groupTags, "tags", nil, "Tag IDs for dynamic membership")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an Edge group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/edge_groups/" + args[0]); err != nil {
				return err
			}
			output.Success("Edge group " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, deleteCmd)
	return cmd
}

func edgeJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge-job",
		Short: "Manage Edge jobs (scheduled scripts on edge environments)",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all Edge jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var jobs []edgeJob
			if err := c.Get("/edge_jobs", &jobs); err != nil {
				return err
			}
			rows := [][]string{}
			for _, j := range jobs {
				recurring := "false"
				if j.Recurring {
					recurring = "true"
				}
				rows = append(rows, []string{
					strconv.Itoa(j.ID),
					j.Name,
					j.CronExpression,
					recurring,
				})
			}
			output.Table([]string{"ID", "NAME", "CRON", "RECURRING"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get Edge job details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_jobs/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var jobName, jobCron, jobFile string
	var jobGroups []int
	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an Edge job from a script file",
		Example: `  portainerctl edge-job create --name cleanup --cron "0 2 * * *" --file cleanup.sh --groups 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobName == "" || jobCron == "" || jobFile == "" {
				return fmt.Errorf("--name, --cron, and --file are required")
			}
			data, err := os.ReadFile(jobFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Name":              jobName,
				"CronExpression":    jobCron,
				"ScriptFileContent": string(data),
				"EdgeGroups":        jobGroups,
				"Endpoints":         []int{},
				"Recurring":         true,
			}
			var result interface{}
			if err := c.Post("/edge_jobs/create/string", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&jobName, "name", "", "Job name")
	createCmd.Flags().StringVar(&jobCron, "cron", "", "Cron expression (e.g. '0 2 * * *')")
	createCmd.Flags().StringVar(&jobFile, "file", "", "Script file path")
	createCmd.Flags().IntSliceVar(&jobGroups, "groups", nil, "Edge group IDs")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an Edge job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/edge_jobs/" + args[0]); err != nil {
				return err
			}
			output.Success("Edge job " + args[0] + " deleted.")
			return nil
		},
	}

	tasksCmd := &cobra.Command{
		Use:   "tasks <id>",
		Short: "List task results for an Edge job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_jobs/"+args[0]+"/tasks", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, deleteCmd, tasksCmd)
	return cmd
}

func edgeConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge-config",
		Short: "Manage Edge configurations (file distribution)",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all Edge configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_configurations", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get Edge configuration details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_configurations/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an Edge configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/edge_configurations/" + args[0]); err != nil {
				return err
			}
			output.Success("Edge configuration " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, deleteCmd)
	return cmd
}

func edgeUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge-update",
		Short: "Manage Edge agent update schedules",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Edge update schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_update_schedules", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get an Edge update schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_update_schedules/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	activeCmd := &cobra.Command{
		Use:   "active",
		Short: "List active update schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_update_schedules/active", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	agentVersionsCmd := &cobra.Command{
		Use:   "agent-versions",
		Short: "List available agent versions for updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/edge_update_schedules/agent_versions", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an Edge update schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/edge_update_schedules/" + args[0]); err != nil {
				return err
			}
			output.Success("Edge update schedule " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, activeCmd, agentVersionsCmd, deleteCmd)
	return cmd
}
