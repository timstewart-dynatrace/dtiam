package get

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var bindingsCmd = &cobra.Command{
	Use:     "bindings",
	Aliases: []string{"binding"},
	Short:   "List policy-to-group bindings, optionally filtered by group or policy",
	Example: `  # List all bindings
  dtiam get bindings

  # Filter bindings by group UUID
  dtiam get bindings --group 12345678-abcd-1234-abcd-1234567890ab

  # Filter bindings by policy UUID
  dtiam get bindings --policy 87654321-dcba-4321-dcba-ba0987654321

  # Output as JSON
  dtiam get bindings -o json

  # Output as YAML
  dtiam get bindings -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get bindings --plain`,
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

		if watchRequested(cmd) {
			// Capture the same filter the single-shot path uses, so watching a
			// filtered view keeps the filter on every poll.
			return runWatch(cmd, func(ctx context.Context) ([]map[string]any, error) {
				switch {
				case groupID != "":
					return handler.GetForGroup(ctx, groupID)
				case policyID != "":
					binding, err := handler.GetForPolicy(ctx, policyID)
					if err != nil {
						return nil, err
					}
					return []map[string]any{binding}, nil
				default:
					return handler.List(ctx, nil)
				}
			}, output.BindingColumns())
		}

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
