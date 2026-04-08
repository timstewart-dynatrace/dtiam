// Package boundary provides boundary management commands.
package boundary

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the boundary command.
var Cmd = &cobra.Command{
	Use:   "boundary",
	Short: "Attach, detach, and inspect boundaries on policy bindings",
	Long: `Commands for attaching and detaching boundaries from policy bindings.

Boundaries restrict the scope of a policy binding to a subset of resources
such as management zones, app IDs, or schema IDs. Use these commands to
manage which boundaries are associated with specific group/policy bindings.`,
	Example: `  # Attach a boundary to a binding
  dtiam boundary attach --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # Detach a boundary from a binding
  dtiam boundary detach --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # List policies using a specific boundary
  dtiam boundary list-attached BOUNDARY_UUID`,
}

func init() {
	Cmd.AddCommand(attachCmd)
	Cmd.AddCommand(detachCmd)
	Cmd.AddCommand(listAttachedCmd)
	Cmd.AddCommand(createAppBoundaryCmd)
	Cmd.AddCommand(createSchemaBoundaryCmd)
}

// buildBoundaryQuery builds a boundary query string from a prefix, operator, and list of IDs.
func buildBoundaryQuery(prefix, operator string, ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("%q", id)
	}
	return fmt.Sprintf("%s %s (%s)", prefix, operator, strings.Join(quoted, ", "))
}

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach a boundary to a policy binding",
	Long: `Attach a boundary to an existing policy binding identified by group and policy UUIDs.

The boundary constrains the scope of the policy for the specified group. All three
flags (--group, --policy, --boundary) are required.`,
	Example: `  # Attach a boundary to a group/policy binding
  dtiam boundary attach --group abc-123 --policy def-456 --boundary ghi-789

  # Preview with dry run
  dtiam boundary attach --group abc-123 --policy def-456 --boundary ghi-789 --dry-run

  # Short flags
  dtiam boundary attach -g abc-123 -p def-456 -b ghi-789`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		boundaryID, _ := cmd.Flags().GetString("boundary")

		if groupID == "" || policyID == "" || boundaryID == "" {
			return fmt.Errorf("--group, --policy, and --boundary are all required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would attach boundary %s to binding (group=%s, policy=%s)", boundaryID, groupID, policyID)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		if err := handler.AddBoundary(ctx, groupID, policyID, boundaryID); err != nil {
			return err
		}

		printer.PrintSuccess("Boundary attached successfully")
		return nil
	},
}

func init() {
	attachCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
	attachCmd.Flags().StringP("policy", "p", "", "Policy UUID (required)")
	attachCmd.Flags().StringP("boundary", "b", "", "Boundary UUID (required)")
}

var detachCmd = &cobra.Command{
	Use:   "detach",
	Short: "Detach a boundary from a policy binding",
	Long: `Detach a boundary from an existing policy binding identified by group and policy UUIDs.

Removing a boundary widens the scope of the policy for the specified group back to
its unrestricted state. All three flags (--group, --policy, --boundary) are required.`,
	Example: `  # Detach a boundary from a group/policy binding
  dtiam boundary detach --group abc-123 --policy def-456 --boundary ghi-789

  # Preview with dry run
  dtiam boundary detach --group abc-123 --policy def-456 --boundary ghi-789 --dry-run

  # Short flags
  dtiam boundary detach -g abc-123 -p def-456 -b ghi-789`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		boundaryID, _ := cmd.Flags().GetString("boundary")

		if groupID == "" || policyID == "" || boundaryID == "" {
			return fmt.Errorf("--group, --policy, and --boundary are all required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would detach boundary %s from binding (group=%s, policy=%s)", boundaryID, groupID, policyID)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		if err := handler.RemoveBoundary(ctx, groupID, policyID, boundaryID); err != nil {
			return err
		}

		printer.PrintSuccess("Boundary detached successfully")
		return nil
	},
}

func init() {
	detachCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
	detachCmd.Flags().StringP("policy", "p", "", "Policy UUID (required)")
	detachCmd.Flags().StringP("boundary", "b", "", "Boundary UUID (required)")
}

var listAttachedCmd = &cobra.Command{
	Use:   "list-attached IDENTIFIER",
	Short: "List policies using a boundary",
	Long: `List all policy bindings that reference a given boundary.

IDENTIFIER can be a boundary UUID or name. If a name is provided it will be
resolved to the corresponding UUID automatically.`,
	Example: `  # List policies attached to a boundary by UUID
  dtiam boundary list-attached abc-123-def-456

  # List policies attached to a boundary by name
  dtiam boundary list-attached "Production Zone Boundary"

  # Output as JSON
  dtiam boundary list-attached abc-123-def-456 -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		boundary, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if boundary == nil {
			return fmt.Errorf("boundary %q not found", args[0])
		}

		uuid, _ := boundary["uuid"].(string)
		attached, err := handler.GetAttachedPolicies(ctx, uuid)
		if err != nil {
			return err
		}

		if len(attached) == 0 {
			fmt.Println("No policies are using this boundary.")
			return nil
		}

		return printer.Print(attached, nil)
	},
}

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
			appHandler := resources.NewAppHandler(c, environment)
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
			schemaHandler := resources.NewSchemaHandler(c, environment)
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
