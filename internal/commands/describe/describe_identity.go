package describe

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

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
