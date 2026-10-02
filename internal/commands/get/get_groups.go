package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var groupsCmd = &cobra.Command{
	Use:     "groups [identifier]",
	Aliases: []string{"group"},
	Short:   "List IAM groups or get a specific group by UUID or name",
	Example: `  # List all groups
  dtiam get groups

  # Get a specific group by UUID
  dtiam get groups 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific group by name
  dtiam get groups "My Team"

  # Output as JSON
  dtiam get groups -o json

  # Output as YAML
  dtiam get groups -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get groups --plain`,
	Args: cobra.MaximumNArgs(1),
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

		if watchRequested(cmd) {
			return runWatch(cmd, func(ctx context.Context) ([]map[string]any, error) {
				return handler.List(ctx, nil)
			}, output.GroupColumns())
		}

		// List all groups
		groups, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.GroupColumns())
	},
}
