package boundary

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/auth"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var createAppBoundaryCmd = &cobra.Command{
	Use:   "create-app-boundary NAME",
	Short: "Create a boundary scoped to specific app IDs",
	Long: `Create a boundary that restricts access to specific Dynatrace apps.

Uses the shared:app-id boundary query format. By default creates an IN boundary
(allow only listed apps). Use --not-in to create an exclusion boundary instead.

If --environment is provided and --skip-validation is not set, each app ID is
validated against the App Engine Registry before creating the boundary.`,
	Example: `  # Create a boundary allowing specific apps
  dtiam boundary create-app-boundary "Dashboard Apps" --app-ids dynatrace.dashboards,dynatrace.notebooks

  # Create an exclusion boundary
  dtiam boundary create-app-boundary "No Classic" --app-ids dynatrace.classic.smartscape --not-in

  # Validate app IDs against environment
  dtiam boundary create-app-boundary "My Apps" --app-ids dynatrace.dashboards --environment abc12345

  # Skip validation
  dtiam boundary create-app-boundary "My Apps" --app-ids dynatrace.dashboards --skip-validation

  # Preview without creating
  dtiam boundary create-app-boundary "My Apps" --app-ids dynatrace.dashboards --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		appIDsStr, _ := cmd.Flags().GetString("app-ids")
		notIn, _ := cmd.Flags().GetBool("not-in")
		environment, _ := cmd.Flags().GetString("environment")
		description, _ := cmd.Flags().GetString("description")
		skipValidation, _ := cmd.Flags().GetBool("skip-validation")

		if appIDsStr == "" {
			return fmt.Errorf("--app-ids is required")
		}

		appIDs := strings.Split(appIDsStr, ",")
		for i := range appIDs {
			appIDs[i] = strings.TrimSpace(appIDs[i])
		}

		operator := "IN"
		if notIn {
			operator = "NOT IN"
		}

		query := buildBoundaryQuery("shared:app-id", operator, appIDs)

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create boundary %q with query:", name)
			fmt.Fprintf(os.Stderr, "  %s\n", query)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		// Validate app IDs if environment is provided
		if environment != "" && !skipValidation {
			// Validation calls an environment-served API, which needs its own
			// scopes or an environment token; the boundary itself is account-level.
			envClient, err := common.CreateEnvironmentClient(auth.AppEngineScopes)
			if err != nil {
				return err
			}
			defer envClient.Close()
			appHandler := resources.NewAppHandler(envClient, environment)
			ctx := context.Background()
			for _, appID := range appIDs {
				_, err := appHandler.Get(ctx, appID)
				if err != nil {
					return fmt.Errorf("app %q not found in environment %s: %w", appID, environment, err)
				}
			}
			if cli.GlobalState.IsVerbose() {
				fmt.Fprintf(os.Stderr, "All %d app IDs validated\n", len(appIDs))
			}
		}

		boundaryHandler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		var desc *string
		if description != "" {
			desc = &description
		}

		result, err := boundaryHandler.Create(ctx, name, nil, &query, desc)
		if err != nil {
			return err
		}

		printer.PrintSuccess("App boundary %q created", name)
		return printer.PrintDetail(result)
	},
}

func init() {
	createAppBoundaryCmd.Flags().String("app-ids", "", "Comma-separated app IDs (required)")
	createAppBoundaryCmd.Flags().Bool("not-in", false, "Use NOT IN (exclude apps instead of allow)")
	createAppBoundaryCmd.Flags().String("environment", "", "Environment ID for app validation")
	createAppBoundaryCmd.Flags().String("description", "", "Boundary description")
	createAppBoundaryCmd.Flags().Bool("skip-validation", false, "Skip app ID validation")
}

var createSchemaBoundaryCmd = &cobra.Command{
	Use:   "create-schema-boundary NAME",
	Short: "Create a boundary scoped to specific schema IDs",
	Long: `Create a boundary that restricts access to specific Settings 2.0 schemas.

Uses the settings:schemaId boundary query format. By default creates an IN boundary
(allow only listed schemas). Use --not-in to create an exclusion boundary instead.

If --environment is provided and --skip-validation is not set, each schema ID is
validated against the Settings API before creating the boundary.`,
	Example: `  # Create a boundary allowing specific schemas
  dtiam boundary create-schema-boundary "Alerting Only" --schema-ids builtin:alerting.profile,builtin:alerting.maintenance-window

  # Create an exclusion boundary
  dtiam boundary create-schema-boundary "No Spans" --schema-ids builtin:span-attribute,builtin:span-capture-rule --not-in

  # Validate schema IDs against environment
  dtiam boundary create-schema-boundary "My Schemas" --schema-ids builtin:alerting.profile --environment abc12345

  # Preview without creating
  dtiam boundary create-schema-boundary "My Schemas" --schema-ids builtin:alerting.profile --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		schemaIDsStr, _ := cmd.Flags().GetString("schema-ids")
		notIn, _ := cmd.Flags().GetBool("not-in")
		environment, _ := cmd.Flags().GetString("environment")
		description, _ := cmd.Flags().GetString("description")
		skipValidation, _ := cmd.Flags().GetBool("skip-validation")

		if schemaIDsStr == "" {
			return fmt.Errorf("--schema-ids is required")
		}

		schemaIDs := strings.Split(schemaIDsStr, ",")
		for i := range schemaIDs {
			schemaIDs[i] = strings.TrimSpace(schemaIDs[i])
		}

		operator := "IN"
		if notIn {
			operator = "NOT IN"
		}

		query := buildBoundaryQuery("settings:schemaId", operator, schemaIDs)

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create boundary %q with query:", name)
			fmt.Fprintf(os.Stderr, "  %s\n", query)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		// Validate schema IDs if environment is provided
		if environment != "" && !skipValidation {
			// Validation calls an environment-served API, which needs its own
			// scopes or an environment token; the boundary itself is account-level.
			envClient, err := common.CreateEnvironmentClient(auth.SettingsSchemaScopes)
			if err != nil {
				return err
			}
			defer envClient.Close()
			schemaHandler := resources.NewSchemaHandler(envClient, environment)
			ctx := context.Background()
			for _, schemaID := range schemaIDs {
				_, err := schemaHandler.Get(ctx, schemaID)
				if err != nil {
					return fmt.Errorf("schema %q not found in environment %s: %w", schemaID, environment, err)
				}
			}
			if cli.GlobalState.IsVerbose() {
				fmt.Fprintf(os.Stderr, "All %d schema IDs validated\n", len(schemaIDs))
			}
		}

		boundaryHandler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		var desc *string
		if description != "" {
			desc = &description
		}

		result, err := boundaryHandler.Create(ctx, name, nil, &query, desc)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Schema boundary %q created", name)
		return printer.PrintDetail(result)
	},
}

func init() {
	createSchemaBoundaryCmd.Flags().String("schema-ids", "", "Comma-separated schema IDs (required)")
	createSchemaBoundaryCmd.Flags().Bool("not-in", false, "Use NOT IN (exclude schemas instead of allow)")
	createSchemaBoundaryCmd.Flags().String("environment", "", "Environment ID for schema validation")
	createSchemaBoundaryCmd.Flags().String("description", "", "Boundary description")
	createSchemaBoundaryCmd.Flags().Bool("skip-validation", false, "Skip schema ID validation")
}
