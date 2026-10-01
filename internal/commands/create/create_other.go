package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var boundaryCmd = &cobra.Command{
	Use:   "boundary",
	Short: "Create a new boundary",
	Example: `  # Create a boundary with management zones
  dtiam create boundary --name "Production" --zone "Production" --zone "Staging"

  # Create a boundary with a custom query
  dtiam create boundary --name "Apps Only" --query 'shared:app-id IN ("dynatrace.dashboards")'

  # Create a boundary with a description
  dtiam create boundary --name "Production" --zone "Production" --description "Production boundary"

  # Dry run to preview
  dtiam create boundary --name "Production" --zone "Production" --dry-run

  # Machine-friendly output
  dtiam create boundary --name "Production" --zone "Production" -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		zones, _ := cmd.Flags().GetStringSlice("zone")
		query, _ := cmd.Flags().GetString("query")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if len(zones) == 0 && query == "" {
			return fmt.Errorf("either --zone or --query is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create boundary: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		var queryPtr, descPtr *string
		if query != "" {
			queryPtr = &query
		}
		if description != "" {
			descPtr = &description
		}

		boundary, err := handler.Create(ctx, name, zones, queryPtr, descPtr)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Boundary created successfully")
		return printer.Print([]map[string]any{boundary}, output.BoundaryColumns())
	},
}

func init() {
	boundaryCmd.Flags().StringP("name", "n", "", "Boundary name (required)")
	boundaryCmd.Flags().StringSliceP("zone", "z", nil, "Management zone names")
	boundaryCmd.Flags().StringP("query", "q", "", "Boundary query")
	boundaryCmd.Flags().StringP("description", "d", "", "Boundary description")
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Create a new platform token",
	Long: `Create a new platform token. The token value is only returned once during creation
and cannot be retrieved later — save it immediately.`,
	Example: `  # Create a token with name
  dtiam create token --name "CI Token"

  # Create a token with scopes and expiration
  dtiam create token --name "CI Token" --scopes "account-idm-read,iam-policies-management" --expires-in 30d

  # Dry run
  dtiam create token --name "CI Token" --dry-run

  # Machine-friendly output
  dtiam create token --name "CI Token" -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		scopesStr, _ := cmd.Flags().GetString("scopes")
		expiresIn, _ := cmd.Flags().GetString("expires-in")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create platform token: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewTokenHandler(c)
		ctx := context.Background()

		var scopes []string
		if scopesStr != "" {
			scopes = append(scopes, splitAndTrim(scopesStr)...)
		}

		token, err := handler.Create(ctx, name, scopes, expiresIn)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Platform token created successfully")

		// Show the token value prominently — it cannot be retrieved later
		if tokenValue, ok := token["token"].(string); ok && tokenValue != "" {
			printer.PrintWarning("Save this token now — it cannot be retrieved later:")
			printer.PrintMessage("%s", tokenValue)
		}

		return printer.Print([]map[string]any{token}, output.TokenColumns())
	},
}

func init() {
	tokenCmd.Flags().StringP("name", "n", "", "Token name (required)")
	tokenCmd.Flags().String("scopes", "", "Comma-separated scopes")
	tokenCmd.Flags().String("expires-in", "", "Token expiration (e.g., 30d, 1y)")
}

// splitAndTrim splits a comma-separated string and trims whitespace.
func splitAndTrim(s string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
