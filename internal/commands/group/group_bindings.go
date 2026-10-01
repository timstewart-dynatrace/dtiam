package group

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var bindingsCmd = &cobra.Command{
	Use:   "bindings IDENTIFIER",
	Short: "List policy bindings for a group",
	Long: `List all policy bindings associated with a group.

The group can be identified by UUID or name. Returns binding details
including the policy and level information.`,
	Example: `  # List bindings by group name
  dtiam group bindings "Production Team"

  # List bindings by group UUID
  dtiam group bindings 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam group bindings "Production Team" -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, groupHandler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		bindings, err := bindingHandler.GetForGroup(ctx, uuid)
		if err != nil {
			return err
		}

		return printer.Print(bindings, output.BindingColumns())
	},
}
