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

var policiesCmd = &cobra.Command{
	Use:     "policies [identifier]",
	Aliases: []string{"policy"},
	Short:   "List IAM policies or get a specific policy by UUID or name",
	Example: `  # List all account-level policies
  dtiam get policies

  # Get a specific policy by UUID
  dtiam get policies 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific policy by name
  dtiam get policies "AppEngine - Reader"

  # List policies at a specific level
  dtiam get policies --level environment --level-id abc12345

  # List policies from all levels (account, environment, global)
  dtiam get policies --all-levels

  # Output as JSON
  dtiam get policies -o json

  # Output as YAML
  dtiam get policies --all-levels -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get policies --plain`,
	Args: cobra.MaximumNArgs(1),
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
