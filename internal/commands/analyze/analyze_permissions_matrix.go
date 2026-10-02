package analyze

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

var permissionsMatrixCmd = &cobra.Command{
	Use:   "permissions-matrix",
	Short: "Generate a permissions matrix",
	Long: `Generate a permissions matrix.

Shows which permissions are granted by each policy or group.
Useful for security audits and compliance reviews.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		exportFile, _ := cmd.Flags().GetString("export")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		matrixGen := utils.NewPermissionsMatrix(c)
		ctx := context.Background()

		var result *utils.MatrixResult
		var nameField string

		if scope == "groups" {
			result, err = matrixGen.GenerateGroupMatrix(ctx)
			nameField = "group_name"
		} else {
			result, err = matrixGen.GeneratePolicyMatrix(ctx)
			nameField = "policy_name"
		}

		if err != nil {
			return err
		}

		// Export to CSV
		if exportFile != "" {
			file, err := os.Create(exportFile)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			writer := csv.NewWriter(file)
			defer writer.Flush()

			if len(result.Matrix) > 0 {
				// Get fields from first row
				var fields []string
				fields = append(fields, nameField)
				fields = append(fields, result.Permissions...)

				if err := writer.Write(fields); err != nil {
					return err
				}

				for _, row := range result.Matrix {
					var values []string
					values = append(values, fmt.Sprintf("%v", row[nameField]))
					for _, perm := range result.Permissions {
						if row[perm] == true {
							values = append(values, "true")
						} else {
							values = append(values, "false")
						}
					}
					if err := writer.Write(values); err != nil {
						return err
					}
				}
			}

			fmt.Printf("Exported matrix to %s\n", exportFile)
			return nil
		}

		// Check output format
		format := cli.GlobalState.GetOutputFormat()
		if format == output.FormatJSON || format == output.FormatYAML {
			printer := cli.GlobalState.NewPrinter()
			return printer.PrintAny(result)
		}

		// Table output
		fmt.Println()
		fmt.Printf("=== Permissions Matrix (%s) ===\n", scope)
		if scope == "groups" {
			fmt.Printf("Total groups: %d\n", result.GroupCount)
		} else {
			fmt.Printf("Total policies: %d\n", result.PolicyCount)
		}
		fmt.Printf("Unique permissions: %d\n", result.PermissionCount)
		fmt.Println()

		if len(result.Matrix) == 0 {
			fmt.Println("No data found.")
			return nil
		}

		// Create table with limited columns for readability
		table := tablewriter.NewWriter(os.Stdout)
		headers := []string{"Name"}

		// Limit to first 5 permissions
		displayPerms := result.Permissions
		if len(displayPerms) > 5 {
			displayPerms = displayPerms[:5]
		}
		for _, perm := range displayPerms {
			// Shorten permission name
			short := perm
			if strings.Contains(perm, ":") {
				parts := strings.Split(perm, ":")
				short = parts[len(parts)-1]
			}
			if len(short) > 15 {
				short = short[:15]
			}
			headers = append(headers, short)
		}
		if len(result.Permissions) > 5 {
			headers = append(headers, "...")
		}
		table.SetHeader(headers)
		table.SetBorder(false)

		// Limit rows to 20
		displayRows := result.Matrix
		if len(displayRows) > 20 {
			displayRows = displayRows[:20]
		}

		for _, row := range displayRows {
			cells := []string{fmt.Sprintf("%v", row[nameField])}
			for _, perm := range displayPerms {
				if row[perm] == true {
					cells = append(cells, "✓")
				} else {
					cells = append(cells, "")
				}
			}
			if len(result.Permissions) > 5 {
				cells = append(cells, "")
			}
			table.Append(cells)
		}

		if len(result.Matrix) > 20 {
			fmt.Printf("(Showing first 20 of %d rows)\n", len(result.Matrix))
		}

		table.Render()
		fmt.Println()
		fmt.Println("Use --export to get full matrix as CSV")

		return nil
	},
}

func init() {
	permissionsMatrixCmd.Flags().StringP("scope", "s", "policies", "Scope: policies or groups")
	permissionsMatrixCmd.Flags().StringP("export", "e", "", "Export to CSV file")
}
