package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/auth"
	"github.com/jtimothystewart/dtiam/pkg/output"
	"github.com/jtimothystewart/dtiam/pkg/resources"
)

var (
	appsEnvironmentFlag    string
	schemasEnvironmentFlag string
)

var appsCmd = &cobra.Command{
	Use:     "apps [identifier]",
	Aliases: []string{"app"},
	Short:   "List apps from the App Engine Registry",
	Long:    "List apps from the Dynatrace App Engine Registry. Requires --environment flag or DTIAM_ENVIRONMENT_URL.",
	Example: `  # List all apps in an environment
  dtiam get apps --environment abc12345

  # Get a specific app by ID
  dtiam get apps dynatrace.dashboards --environment abc12345

  # Output as JSON
  dtiam get apps --environment abc12345 -o json

  # Use environment from config/env var
  DTIAM_ENVIRONMENT_URL=abc12345 dtiam get apps`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		envURL := appsEnvironmentFlag
		if envURL == "" {
			envURL = cli.GlobalState.EnvironmentURL()
		}
		if envURL == "" {
			return fmt.Errorf("--environment flag or DTIAM_ENVIRONMENT_URL is required")
		}

		c, err := common.CreateEnvironmentClient(auth.AppEngineScopes)
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewAppHandler(c, envURL)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			app, err := handler.Get(ctx, args[0])
			if err != nil {
				return err
			}
			return printer.PrintSingle(app, output.AppColumns())
		}

		apps, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(apps, output.AppColumns())
	},
}

func init() {
	appsCmd.Flags().StringVar(&appsEnvironmentFlag, "environment", "", "Environment ID or URL (required)")
}
