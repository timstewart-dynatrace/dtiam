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

var environmentsCmd = &cobra.Command{
	Use:     "environments [identifier]",
	Aliases: []string{"envs", "env"},
	Short:   "List Dynatrace environments or get a specific environment by ID or name",
	Example: `  # List all environments
  dtiam get environments

  # Get a specific environment by ID
  dtiam get environments abc12345

  # Get a specific environment by name
  dtiam get environments "Production"

  # Output as JSON
  dtiam get environments -o json

  # Output as YAML
  dtiam get environments -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get environments --plain`,
	Args: cobra.MaximumNArgs(1),
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
