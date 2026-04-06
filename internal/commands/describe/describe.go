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
	Long: `Display detailed information about a specific IAM resource.

Unlike 'get' which lists resources in a table, 'describe' shows all fields
for a single resource including nested objects and metadata. The IDENTIFIER
can be a UUID or a human-readable name (or email for users).`,
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
	Short: "Show detailed group info including members and policies",
	Long: `Display detailed information about a specific IAM group.

Shows all group fields including UUID, name, description, owner, member list,
and associated policies. The IDENTIFIER can be a group UUID or group name.`,
	Example: `  # Describe a group by UUID
  dtiam describe group 3a4b5c6d-7e8f-9a0b-1c2d-3e4f5a6b7c8d

  # Describe a group by name
  dtiam describe group "Platform Engineers"

  # Machine-friendly JSON output for scripting
  dtiam describe group "Platform Engineers" -o json --plain`,
	Args: cobra.ExactArgs(1),
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
	Short: "Show detailed user info including group memberships",
	Long: `Display detailed information about a specific IAM user.

Shows all user fields including UID, email, name, status, group memberships,
and associated permissions. The IDENTIFIER can be a user UID or email address.`,
	Example: `  # Describe a user by UID
  dtiam describe user 8f2e4a6b-1c3d-5e7f-9a0b-2c4d6e8f0a1b

  # Describe a user by email
  dtiam describe user alice@example.com

  # Machine-friendly JSON output for scripting
  dtiam describe user alice@example.com -o json --plain`,
	Args: cobra.ExactArgs(1),
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
	Short: "Show detailed policy info including statements and conditions",
	Long: `Display detailed information about a specific IAM policy.

Shows all policy fields including UUID, name, description, level, policy
statements, and conditions. The IDENTIFIER can be a policy UUID or policy name.`,
	Example: `  # Describe a policy by UUID
  dtiam describe policy a1b2c3d4-e5f6-7890-abcd-ef1234567890

  # Describe a policy by name
  dtiam describe policy "Environment Read-Only"

  # Machine-friendly JSON output for scripting
  dtiam describe policy "Environment Read-Only" -o json --plain`,
	Args: cobra.ExactArgs(1),
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
	Short:   "Show detailed environment info including state and management zones",
	Long: `Display detailed information about a specific Dynatrace environment.

Shows all environment fields including ID, name, state, management zones,
and configuration details. The IDENTIFIER can be an environment ID or name.`,
	Example: `  # Describe an environment by ID
  dtiam describe environment abc12345

  # Describe an environment by name
  dtiam describe environment "Production US-East"

  # Machine-friendly JSON output for scripting
  dtiam describe environment abc12345 -o json --plain`,
	Args: cobra.ExactArgs(1),
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
	Short: "Show detailed boundary info including query and attached policies",
	Long: `Display detailed information about a specific IAM boundary.

Shows all boundary fields including UUID, name, description, boundary query,
and attached policies. The IDENTIFIER can be a boundary UUID or boundary name.`,
	Example: `  # Describe a boundary by UUID
  dtiam describe boundary f1e2d3c4-b5a6-9870-fedc-ba0987654321

  # Describe a boundary by name
  dtiam describe boundary "Production Management Zone"

  # Machine-friendly JSON output for scripting
  dtiam describe boundary "Production Management Zone" -o json --plain`,
	Args: cobra.ExactArgs(1),
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
	Short:   "Show detailed service user info including OAuth clients",
	Long: `Display detailed information about a specific IAM service user.

Shows all service user fields including UID, name, description, status,
group memberships, and OAuth client details. The IDENTIFIER can be a
service user UID or name.`,
	Example: `  # Describe a service user by UID
  dtiam describe service-user 7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d

  # Describe a service user by name
  dtiam describe service-user "CI/CD Pipeline Bot"

  # Machine-friendly JSON output for scripting
  dtiam describe service-user "CI/CD Pipeline Bot" -o json --plain`,
	Args: cobra.ExactArgs(1),
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
