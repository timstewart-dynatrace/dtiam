// Package get provides commands for listing and retrieving resources.
package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the get command.
var Cmd = &cobra.Command{
	Use:   "get",
	Short: "List or retrieve resources",
	Long:  "Commands for listing and retrieving IAM resources.",
}

func init() {
	Cmd.AddCommand(groupsCmd)
	Cmd.AddCommand(usersCmd)
	Cmd.AddCommand(policiesCmd)
	Cmd.AddCommand(bindingsCmd)
	Cmd.AddCommand(environmentsCmd)
	Cmd.AddCommand(boundariesCmd)
}

var groupsCmd = &cobra.Command{
	Use:     "groups [identifier]",
	Aliases: []string{"group"},
	Short:   "List groups or get a specific group",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single group
			group, err := resources.GetOrResolve(ctx, handler, args[0])
			if err != nil {
				return err
			}
			if group == nil {
				return fmt.Errorf("group %q not found", args[0])
			}
			return printer.PrintSingle(group, output.GroupColumns())
		}

		// List all groups
		groups, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.GroupColumns())
	},
}

var usersCmd = &cobra.Command{
	Use:     "users [identifier]",
	Aliases: []string{"user"},
	Short:   "List users or get a specific user",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single user
			user, err := handler.Get(ctx, args[0])
			if err != nil {
				// Try by email
				user, err = handler.GetByEmail(ctx, args[0])
				if err != nil {
					return err
				}
			}
			if user == nil {
				return fmt.Errorf("user %q not found", args[0])
			}
			return printer.PrintSingle(user, output.UserColumns())
		}

		// List all users
		users, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(users, output.UserColumns())
	},
}

var policiesCmd = &cobra.Command{
	Use:     "policies [identifier]",
	Aliases: []string{"policy"},
	Short:   "List policies or get a specific policy",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		level, _ := cmd.Flags().GetString("level")
		levelID, _ := cmd.Flags().GetString("level-id")

		var handler *resources.PolicyHandler
		if level != "" && level != "account" {
			if levelID == "" {
				return fmt.Errorf("--level-id is required when using --level")
			}
			handler = resources.NewPolicyHandlerWithLevel(c, level, levelID)
		} else {
			handler = resources.NewPolicyHandler(c)
		}

		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single policy
			policy, err := resources.GetOrResolve(ctx, handler, args[0])
			if err != nil {
				return err
			}
			if policy == nil {
				return fmt.Errorf("policy %q not found", args[0])
			}
			return printer.PrintSingle(policy, output.PolicyColumns())
		}

		// List policies
		allLevels, _ := cmd.Flags().GetBool("all-levels")
		var policies []map[string]any

		if allLevels {
			policies, err = handler.ListAllLevels(ctx)
		} else {
			policies, err = handler.List(ctx, nil)
		}

		if err != nil {
			return err
		}

		return printer.Print(policies, output.PolicyColumns())
	},
}

func init() {
	policiesCmd.Flags().String("level", "account", "Policy level (account, environment, global)")
	policiesCmd.Flags().String("level-id", "", "Level ID (required for environment level)")
	policiesCmd.Flags().Bool("all-levels", false, "List policies from all levels")
}

var bindingsCmd = &cobra.Command{
	Use:     "bindings",
	Aliases: []string{"binding"},
	Short:   "List policy bindings",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")

		var bindings []map[string]any

		if groupID != "" {
			bindings, err = handler.GetForGroup(ctx, groupID)
		} else if policyID != "" {
			binding, err := handler.GetForPolicy(ctx, policyID)
			if err != nil {
				return err
			}
			bindings = []map[string]any{binding}
		} else {
			bindings, err = handler.List(ctx, nil)
		}

		if err != nil {
			return err
		}

		return printer.Print(bindings, output.BindingColumns())
	},
}

func init() {
	bindingsCmd.Flags().String("group", "", "Filter bindings by group UUID")
	bindingsCmd.Flags().String("policy", "", "Filter bindings by policy UUID")
}

var environmentsCmd = &cobra.Command{
	Use:     "environments [identifier]",
	Aliases: []string{"envs", "env"},
	Short:   "List environments or get a specific environment",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewEnvironmentHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single environment
			env, err := handler.Get(ctx, args[0])
			if err != nil {
				// Try by name
				env, err = handler.GetByName(ctx, args[0])
				if err != nil {
					return err
				}
			}
			if env == nil {
				return fmt.Errorf("environment %q not found", args[0])
			}
			return printer.PrintSingle(env, output.EnvironmentColumns())
		}

		// List all environments
		envs, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(envs, output.EnvironmentColumns())
	},
}

var boundariesCmd = &cobra.Command{
	Use:     "boundaries [identifier]",
	Aliases: []string{"boundary"},
	Short:   "List boundaries or get a specific boundary",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single boundary
			boundary, err := resources.GetOrResolve(ctx, handler, args[0])
			if err != nil {
				return err
			}
			if boundary == nil {
				return fmt.Errorf("boundary %q not found", args[0])
			}
			return printer.PrintSingle(boundary, output.BoundaryColumns())
		}

		// List all boundaries
		boundaries, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(boundaries, output.BoundaryColumns())
	},
}
