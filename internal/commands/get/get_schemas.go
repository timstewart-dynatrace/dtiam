package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/auth"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var schemasCmd = &cobra.Command{
	Use:     "schemas [identifier]",
	Aliases: []string{"schema"},
	Short:   "List settings schemas from the Environment API",
	Long:    "List Settings 2.0 schemas from a Dynatrace environment. Requires --environment flag or DTIAM_ENVIRONMENT_URL.",
	Example: `  # List all schemas
  dtiam get schemas --environment abc12345

  # Get a specific schema
  dtiam get schemas builtin:alerting.profile --environment abc12345

  # Search schemas by name
  dtiam get schemas --environment abc12345 --name alerting

  # Output as JSON
  dtiam get schemas --environment abc12345 -o json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		envURL := schemasEnvironmentFlag
		if envURL == "" {
			envURL = cli.GlobalState.EnvironmentURL()
		}
		if envURL == "" {
			return fmt.Errorf("--environment flag or DTIAM_ENVIRONMENT_URL is required")
		}

		c, err := common.CreateEnvironmentClient(auth.SettingsSchemaScopes)
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewSchemaHandler(c, envURL)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			schema, err := handler.Get(ctx, args[0])
			if err != nil {
				return err
			}
			return printer.PrintSingle(schema, output.SchemaColumns())
		}

		nameFilter, _ := cmd.Flags().GetString("name")
		var schemas []map[string]any

		if nameFilter != "" {
			schemas, err = handler.Search(ctx, nameFilter)
		} else {
			schemas, err = handler.List(ctx, nil)
		}

		if err != nil {
			return err
		}

		return printer.Print(schemas, output.SchemaColumns())
	},
}

func init() {
	schemasCmd.Flags().StringVar(&schemasEnvironmentFlag, "environment", "", "Environment ID or URL (required)")
	schemasCmd.Flags().String("name", "", "Filter schemas by name pattern")
}
