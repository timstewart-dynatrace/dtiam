package get

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/output"
	"github.com/jtimothystewart/dtiam/pkg/resources"
)

var tokensCmd = &cobra.Command{
	Use:     "tokens [identifier]",
	Aliases: []string{"token"},
	Short:   "List platform tokens or get a specific token by ID or name",
	Example: `  # List all platform tokens
  dtiam get tokens

  # Get a specific token by ID
  dtiam get tokens abc-123

  # Output as JSON
  dtiam get tokens -o json

  # Machine-friendly output
  dtiam get tokens --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewTokenHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			token, err := handler.Get(ctx, args[0])
			if err != nil {
				return err
			}
			return printer.PrintSingle(token, output.TokenColumns())
		}

		tokens, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(tokens, output.TokenColumns())
	},
}
