// Package describe provides commands for displaying detailed resource information.
package describe

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the describe command.
var Cmd = &cobra.Command{
	Use:   "describe",
	Short: "Show detailed information about a resource",
	Long:  "Commands for displaying detailed information about IAM resources.",
}

func init() {
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(userCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(environmentCmd)
	Cmd.AddCommand(boundaryCmd)
	Cmd.AddCommand(serviceUserCmd)
}

var groupCmd = &cobra.Command{
	Use:   "group IDENTIFIER",
	Short: "Show detailed information about a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		// Get expanded group with members and policies
		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		// Get expanded information
		expanded, err := handler.GetExpanded(ctx, utils.StringFrom(group, "uuid"))
		if err != nil {
			expanded = group
		}

		return printer.PrintDetail(expanded)
	},
}

var userCmd = &cobra.Command{
	Use:   "user IDENTIFIER",
	Short: "Show detailed information about a user",
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

		// Get expanded information
		uid, _ := user["uid"].(string)
		if uid != "" {
			expanded, err := handler.GetExpanded(ctx, uid)
			if err == nil {
				user = expanded
			}
		}

		return printer.PrintDetail(user)
	},
}

var policyCmd = &cobra.Command{
	Use:   "policy IDENTIFIER",
	Short: "Show detailed information about a policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewPolicyHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		policy, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if policy == nil {
			return fmt.Errorf("policy %q not found", args[0])
		}

		return printer.PrintDetail(policy)
	},
}

var environmentCmd = &cobra.Command{
	Use:     "environment IDENTIFIER",
	Aliases: []string{"env"},
	Short:   "Show detailed information about an environment",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewEnvironmentHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		env, err := handler.Get(ctx, args[0])
		if err != nil {
			env, err = handler.GetByName(ctx, args[0])
			if err != nil {
				return err
			}
		}
		if env == nil {
			return fmt.Errorf("environment %q not found", args[0])
		}

		return printer.PrintDetail(env)
	},
}

var boundaryCmd = &cobra.Command{
	Use:   "boundary IDENTIFIER",
	Short: "Show detailed information about a boundary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		boundary, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if boundary == nil {
			return fmt.Errorf("boundary %q not found", args[0])
		}

		// Get attached policies
		uuid, _ := boundary["uuid"].(string)
		if uuid != "" {
			attached, err := handler.GetAttachedPolicies(ctx, uuid)
			if err == nil {
				boundary["attached_policies"] = attached
				boundary["attached_count"] = len(attached)
			}
		}

		return printer.PrintDetail(boundary)
	},
}

var serviceUserCmd = &cobra.Command{
	Use:     "service-user IDENTIFIER",
	Aliases: []string{"serviceuser"},
	Short:   "Show detailed information about a service user",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		user, err := handler.Get(ctx, args[0])
		if err != nil {
			user, err = handler.GetByName(ctx, args[0])
			if err != nil {
				return err
			}
		}
		if user == nil {
			return fmt.Errorf("service user %q not found", args[0])
		}

		// Get expanded information
		uid, _ := user["uid"].(string)
		if uid != "" {
			expanded, err := handler.GetExpanded(ctx, uid)
			if err == nil {
				user = expanded
			}
		}

		return printer.PrintDetail(user)
	},
}

// Helper columns for describe output
func detailColumns() []output.Column {
	return []output.Column{} // Detail view doesn't use columns
}
