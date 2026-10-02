package group

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

var updateCmd = &cobra.Command{
	Use:   "update IDENTIFIER",
	Short: "Rename a group or change its description",
	Long: `Change a group's name and/or description. The group can be identified by
UUID or name. Fields you do not pass keep their current values.

Uses PUT /groups/{uuid}. To manage a group declaratively, use "dtiam apply",
which updates existing groups the same way.`,
	Example: `  # Rename a group
  dtiam group update "Platform Team" --name "Platform Engineering"

  # Change only the description
  dtiam group update 8f6e5d4c-3b2a-1098-7654-321fedcba098 --description "Owns the platform"

  # Preview
  dtiam group update "Platform Team" --name "Platform Engineering" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := map[string]any{}
		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name cannot be empty")
			}
			data["name"] = name
		}
		if cmd.Flags().Changed("description") {
			desc, _ := cmd.Flags().GetString("description")
			data["description"] = desc
		}
		if len(data) == 0 {
			return fmt.Errorf("nothing to update: pass --name and/or --description")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would update group %q:", args[0])
			return printer.PrintAny(data)
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid := utils.StringFrom(group, "uuid")
		if _, err := handler.Update(ctx, uuid, data); err != nil {
			return err
		}

		printer.PrintSuccess("Group %q updated", args[0])
		return nil
	},
}

func init() {
	updateCmd.Flags().StringP("name", "n", "", "New group name")
	updateCmd.Flags().StringP("description", "d", "", "New group description (pass \"\" to clear it)")
}
