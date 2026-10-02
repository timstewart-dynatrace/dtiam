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

var boundariesCmd = &cobra.Command{
	Use:     "boundaries [identifier]",
	Aliases: []string{"boundary"},
	Short:   "List permission boundaries or get a specific boundary by UUID or name",
	Example: `  # List all boundaries
  dtiam get boundaries

  # Get a specific boundary by UUID
  dtiam get boundaries 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific boundary by name
  dtiam get boundaries "Production MZ Only"

  # Output as JSON
  dtiam get boundaries -o json

  # Output as YAML
  dtiam get boundaries -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get boundaries --plain`,
	Args: cobra.MaximumNArgs(1),
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
