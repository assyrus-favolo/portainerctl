package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/portainer/portainerctl/internal/client"
	"github.com/portainer/portainerctl/internal/output"
)

// portaineree.User from spec
type user struct {
	ID       int    `json:"Id"`
	Username string `json:"Username"`
	Role     int    `json:"Role"` // 1=admin, 2=standard (portainer.UserRole is an int)
}

// portainer.Team from spec
type team struct {
	ID   int    `json:"Id"`
	Name string `json:"Name"`
}

// portainer.TeamMembership from spec
type teamMembership struct {
	ID     int `json:"Id"`
	TeamID int `json:"TeamID"`
	UserID int `json:"UserID"`
	Role   int `json:"Role"` // portainer.MembershipRole: 1=leader, 2=member
}

// portaineree.Role from spec
type role struct {
	ID          int    `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Priority    int    `json:"Priority"`
}

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage Portainer users",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var users []user
			if err := c.Get("/users", &users); err != nil {
				return err
			}
			rows := [][]string{}
			for _, u := range users {
				roleLabel := "standard"
				if u.Role == 1 {
					roleLabel = "admin"
				}
				rows = append(rows, []string{strconv.Itoa(u.ID), u.Username, roleLabel})
			}
			output.Table([]string{"ID", "USERNAME", "ROLE"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get user details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/users/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	meCmd := &cobra.Command{
		Use:   "me",
		Short: "Get details for the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/users/me", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var createUsername, createPassword string
	var createRole int
	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new user",
		Example: `  portainerctl user create --username alice --password secret --role 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if createUsername == "" || createPassword == "" {
				return fmt.Errorf("--username and --password are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"Username": createUsername,
				"Password": createPassword,
				"Role":     createRole,
			}
			var result interface{}
			if err := c.Post("/users", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&createUsername, "username", "", "Username")
	createCmd.Flags().StringVar(&createPassword, "password", "", "Password")
	createCmd.Flags().IntVar(&createRole, "role", 2, "Role: 1=admin, 2=standard")

	var newPassword string
	passwdCmd := &cobra.Command{
		Use:   "passwd <id>",
		Short: "Update a user's password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if newPassword == "" {
				return fmt.Errorf("--password is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{"Password": newPassword}
			if err := c.Put("/users/"+args[0]+"/passwd", body, nil); err != nil {
				return err
			}
			output.Success("Password updated for user " + args[0])
			return nil
		},
	}
	passwdCmd.Flags().StringVar(&newPassword, "password", "", "New password")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/users/" + args[0]); err != nil {
				return err
			}
			output.Success("User " + args[0] + " deleted.")
			return nil
		},
	}

	membershipsCmd := &cobra.Command{
		Use:   "memberships <id>",
		Short: "List team memberships for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var memberships []teamMembership
			if err := c.Get("/users/"+args[0]+"/memberships", &memberships); err != nil {
				return err
			}
			rows := [][]string{}
			for _, m := range memberships {
				roleLabel := "member"
				if m.Role == 1 {
					roleLabel = "leader"
				}
				rows = append(rows, []string{strconv.Itoa(m.ID), strconv.Itoa(m.TeamID), roleLabel})
			}
			output.Table([]string{"ID", "TEAM", "ROLE"}, rows)
			return nil
		},
	}

	tokensCmd := &cobra.Command{
		Use:   "tokens <id>",
		Short: "List API tokens for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/users/"+args[0]+"/tokens", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var tokenDesc string
	createTokenCmd := &cobra.Command{
		Use:   "create-token <user-id>",
		Short: "Create an API token for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenDesc == "" {
				return fmt.Errorf("--description is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{"Description": tokenDesc}
			var result interface{}
			if err := c.Post("/users/"+args[0]+"/tokens", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createTokenCmd.Flags().StringVar(&tokenDesc, "description", "", "Token description")

	deleteTokenCmd := &cobra.Command{
		Use:   "delete-token <user-id> <key-id>",
		Short: "Delete an API token",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/users/" + args[0] + "/tokens/" + args[1]); err != nil {
				return err
			}
			output.Success("Token " + args[1] + " deleted.")
			return nil
		},
	}

	namespacesCmd := &cobra.Command{
		Use:   "namespaces <id>",
		Short: "List Kubernetes namespaces accessible by a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/users/"+args[0]+"/namespaces", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	gitCredsCmd := &cobra.Command{
		Use:   "git-credentials <id>",
		Short: "List Git credentials for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/users/"+args[0]+"/gitcredentials", &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, meCmd, createCmd, passwdCmd, deleteCmd,
		membershipsCmd, tokensCmd, createTokenCmd, deleteTokenCmd, namespacesCmd, gitCredsCmd)
	return cmd
}

func teamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage Portainer teams",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var teams []team
			if err := c.Get("/teams", &teams); err != nil {
				return err
			}
			rows := [][]string{}
			for _, t := range teams {
				rows = append(rows, []string{strconv.Itoa(t.ID), t.Name})
			}
			output.Table([]string{"ID", "NAME"}, rows)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get team details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Get("/teams/"+args[0], &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}

	var teamName string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamName == "" {
				return fmt.Errorf("--name is required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var result interface{}
			if err := c.Post("/teams", map[string]interface{}{"Name": teamName}, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&teamName, "name", "", "Team name")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/teams/" + args[0]); err != nil {
				return err
			}
			output.Success("Team " + args[0] + " deleted.")
			return nil
		},
	}

	membershipsCmd := &cobra.Command{
		Use:   "memberships <id>",
		Short: "List members of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var memberships []teamMembership
			if err := c.Get("/teams/"+args[0]+"/memberships", &memberships); err != nil {
				return err
			}
			rows := [][]string{}
			for _, m := range memberships {
				roleLabel := "member"
				if m.Role == 1 {
					roleLabel = "leader"
				}
				rows = append(rows, []string{strconv.Itoa(m.ID), strconv.Itoa(m.UserID), roleLabel})
			}
			output.Table([]string{"ID", "USER", "ROLE"}, rows)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd, createCmd, deleteCmd, membershipsCmd)
	return cmd
}

func teamMembershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team-membership",
		Short: "Manage team memberships",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all team memberships",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var memberships []teamMembership
			if err := c.Get("/team_memberships", &memberships); err != nil {
				return err
			}
			rows := [][]string{}
			for _, m := range memberships {
				roleLabel := "member"
				if m.Role == 1 {
					roleLabel = "leader"
				}
				rows = append(rows, []string{
					strconv.Itoa(m.ID),
					strconv.Itoa(m.TeamID),
					strconv.Itoa(m.UserID),
					roleLabel,
				})
			}
			output.Table([]string{"ID", "TEAM", "USER", "ROLE"}, rows)
			return nil
		},
	}

	var addTeamID, addUserID, addRole int
	addCmd := &cobra.Command{
		Use:     "add",
		Short:   "Add a user to a team",
		Example: `  portainerctl team-membership add --team 2 --user 5 --role 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if addTeamID == 0 || addUserID == 0 {
				return fmt.Errorf("--team and --user are required")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{"TeamID": addTeamID, "UserID": addUserID, "Role": addRole}
			var result interface{}
			if err := c.Post("/team_memberships", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	addCmd.Flags().IntVar(&addTeamID, "team", 0, "Team ID")
	addCmd.Flags().IntVar(&addUserID, "user", 0, "User ID")
	addCmd.Flags().IntVar(&addRole, "role", 2, "Role: 1=leader, 2=member")

	removeCmd := &cobra.Command{
		Use:   "remove <membership-id>",
		Short: "Remove a team membership",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/team_memberships/" + args[0]); err != nil {
				return err
			}
			output.Success("Membership " + args[0] + " removed.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd)
	return cmd
}

func roleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "List Portainer RBAC roles",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var roles []role
			if err := c.Get("/roles", &roles); err != nil {
				return err
			}
			rows := [][]string{}
			for _, r := range roles {
				rows = append(rows, []string{
					strconv.Itoa(r.ID),
					r.Name,
					r.Description,
					strconv.Itoa(r.Priority),
				})
			}
			output.Table([]string{"ID", "NAME", "DESCRIPTION", "PRIORITY"}, rows)
			return nil
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func resourceControlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource-control",
		Short: "Manage resource access controls (ownership)",
	}

	var rcBody string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a resource control",
		RunE: func(cmd *cobra.Command, args []string) error {
			if rcBody == "" {
				return fmt.Errorf("--body is required (JSON)")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var body, result interface{}
			if err := parseJSON(rcBody, &body); err != nil {
				return err
			}
			if err := c.Post("/resource_controls", body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	createCmd.Flags().StringVar(&rcBody, "body", "", "JSON body for resource control")

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a resource control",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rcBody == "" {
				return fmt.Errorf("--body is required (JSON)")
			}
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			var body, result interface{}
			if err := parseJSON(rcBody, &body); err != nil {
				return err
			}
			if err := c.Put("/resource_controls/"+args[0], body, &result); err != nil {
				return err
			}
			output.JSON(result)
			return nil
		},
	}
	updateCmd.Flags().StringVar(&rcBody, "body", "", "JSON body for resource control update")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a resource control",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.MustClient()
			if err != nil {
				return err
			}
			if err := c.Delete("/resource_controls/" + args[0]); err != nil {
				return err
			}
			output.Success("Resource control " + args[0] + " deleted.")
			return nil
		},
	}

	cmd.AddCommand(createCmd, updateCmd, deleteCmd)
	return cmd
}
