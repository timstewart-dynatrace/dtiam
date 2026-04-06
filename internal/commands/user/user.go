// Package user provides user management commands.
package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the user command.
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  "Commands for managing user operations like group membership.",
}

func init() {
	Cmd.AddCommand(addToGroupsCmd)
	Cmd.AddCommand(removeFromGroupsCmd)
	Cmd.AddCommand(replaceGroupsCmd)
	Cmd.AddCommand(listGroupsCmd)
	Cmd.AddCommand(createCmd)
}

var addToGroupsCmd = &cobra.Command{
	Use:   "add-to-groups EMAIL",
	Short: "Add a user to groups",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		groupsStr, _ := cmd.Flags().GetString("groups")

		if groupsStr == "" {
			return fmt.Errorf("--groups is required")
		}

		groups := strings.Split(groupsStr, ",")
		for i := range groups {
			groups[i] = strings.TrimSpace(groups[i])
		}

		if cli.GlobalState.IsDryRun() {
			fmt.Printf("Would add user %s to groups: %s\n", email, groupsStr)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		// Resolve group names to UUIDs
		groupHandler := resources.NewGroupHandler(c)
		groupUUIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			group, err := resources.GetOrResolve(ctx, groupHandler, g)
			if err != nil {
				return fmt.Errorf("failed to resolve group %q: %w", g, err)
			}
			if group == nil {
				return fmt.Errorf("group %q not found", g)
			}
			groupUUIDs = append(groupUUIDs, utils.StringFrom(group, "uuid"))
		}

		if err := handler.AddToGroups(ctx, email, groupUUIDs); err != nil {
			return err
		}

		printer.PrintSuccess("User %s added to %d group(s)", email, len(groupUUIDs))
		return nil
	},
}

func init() {
	addToGroupsCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs or names (required)")
}

var removeFromGroupsCmd = &cobra.Command{
	Use:   "remove-from-groups EMAIL",
	Short: "Remove a user from groups",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		groupsStr, _ := cmd.Flags().GetString("groups")

		if groupsStr == "" {
			return fmt.Errorf("--groups is required")
		}

		groups := strings.Split(groupsStr, ",")
		for i := range groups {
			groups[i] = strings.TrimSpace(groups[i])
		}

		if cli.GlobalState.IsDryRun() {
			fmt.Printf("Would remove user %s from groups: %s\n", email, groupsStr)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		// Resolve group names to UUIDs
		groupHandler := resources.NewGroupHandler(c)
		groupUUIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			group, err := resources.GetOrResolve(ctx, groupHandler, g)
			if err != nil {
				return fmt.Errorf("failed to resolve group %q: %w", g, err)
			}
			if group == nil {
				return fmt.Errorf("group %q not found", g)
			}
			groupUUIDs = append(groupUUIDs, utils.StringFrom(group, "uuid"))
		}

		if err := handler.RemoveFromGroups(ctx, email, groupUUIDs); err != nil {
			return err
		}

		printer.PrintSuccess("User %s removed from %d group(s)", email, len(groupUUIDs))
		return nil
	},
}

func init() {
	removeFromGroupsCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs or names (required)")
}

var replaceGroupsCmd = &cobra.Command{
	Use:   "replace-groups EMAIL",
	Short: "Replace all group memberships for a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		groupsStr, _ := cmd.Flags().GetString("groups")

		var groups []string
		if groupsStr != "" {
			groups = strings.Split(groupsStr, ",")
			for i := range groups {
				groups[i] = strings.TrimSpace(groups[i])
			}
		}

		if cli.GlobalState.IsDryRun() {
			fmt.Printf("Would replace groups for user %s with: %s\n", email, groupsStr)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		// Resolve group names to UUIDs
		groupHandler := resources.NewGroupHandler(c)
		groupUUIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			group, err := resources.GetOrResolve(ctx, groupHandler, g)
			if err != nil {
				return fmt.Errorf("failed to resolve group %q: %w", g, err)
			}
			if group == nil {
				return fmt.Errorf("group %q not found", g)
			}
			groupUUIDs = append(groupUUIDs, utils.StringFrom(group, "uuid"))
		}

		if err := handler.ReplaceGroups(ctx, email, groupUUIDs); err != nil {
			return err
		}

		printer.PrintSuccess("User %s groups replaced with %d group(s)", email, len(groupUUIDs))
		return nil
	},
}

func init() {
	replaceGroupsCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs or names")
}

var listGroupsCmd = &cobra.Command{
	Use:   "list-groups IDENTIFIER",
	Short: "List groups a user belongs to",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		user, err := handler.Get(ctx, args[0])
		if err != nil {
			user, err = handler.GetByEmail(ctx, args[0])
			if err != nil {
				return err
			}
		}
		if user == nil {
			return fmt.Errorf("user %q not found", args[0])
		}

		uid, _ := user["uid"].(string)
		groups, err := handler.GetGroups(ctx, uid)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.GroupColumns())
	},
}

var createCmd = &cobra.Command{
	Use:   "create EMAIL",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		groupsStr, _ := cmd.Flags().GetString("groups")

		if cli.GlobalState.IsDryRun() {
			fmt.Printf("Would create user: %s\n", email)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		var firstNamePtr, lastNamePtr *string
		if firstName != "" {
			firstNamePtr = &firstName
		}
		if lastName != "" {
			lastNamePtr = &lastName
		}

		var groups []string
		if groupsStr != "" {
			groups = strings.Split(groupsStr, ",")
			for i := range groups {
				groups[i] = strings.TrimSpace(groups[i])
			}
		}

		user, err := handler.Create(ctx, email, firstNamePtr, lastNamePtr, groups)
		if err != nil {
			return err
		}

		printer.PrintSuccess("User created successfully")
		return printer.Print([]map[string]any{user}, output.UserColumns())
	},
}

func init() {
	createCmd.Flags().String("first-name", "", "User's first name")
	createCmd.Flags().String("last-name", "", "User's last name")
	createCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs")
}
