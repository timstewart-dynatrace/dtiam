package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

func exportResourceToDir(resourceName, outputDir, format, prefix string, data []map[string]any) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.%s", prefix, resourceName, format))
	if err := writeData(data, filePath, format); err != nil {
		return err
	}

	fmt.Printf("Exported %d %s to %s\n", len(data), resourceName, filePath)
	return nil
}

// addResourceExportFlags adds common flags to a resource export command.
func addResourceExportFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", ".", "Output directory")
	cmd.Flags().StringP("format", "f", "csv", "Output format (csv, json, yaml)")
	cmd.Flags().StringP("prefix", "p", "dtiam", "File name prefix")
	cmd.Flags().BoolP("detailed", "d", false, "Include detailed/enriched data")
}

// getResourceExportFlags retrieves common resource export flag values.
func getResourceExportFlags(cmd *cobra.Command) (outputDir, format, prefix string, detailed bool) {
	outputDir, _ = cmd.Flags().GetString("output")
	format, _ = cmd.Flags().GetString("format")
	prefix, _ = cmd.Flags().GetString("prefix")
	detailed, _ = cmd.Flags().GetBool("detailed")
	return
}

var environmentsCmd = &cobra.Command{
	Use:     "environments",
	Aliases: []string{"environment", "envs", "env"},
	Short:   "Export all environments to a file",
	Long:    `Export all Dynatrace environments to a file in the specified format.`,
	Example: `  # Export environments as CSV
  dtiam export environments

  # Export as JSON to a specific directory
  dtiam export environments -o ./backup -f json

  # Export as YAML with custom prefix
  dtiam export environments -f yaml -p myaccount`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, format, prefix, _ := getResourceExportFlags(cmd)

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewEnvironmentHandler(c)
		ctx := context.Background()

		data, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return exportResourceToDir("environments", outputDir, format, prefix, data)
	},
}

func init() { addResourceExportFlags(environmentsCmd) }

var usersCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "Export all users to a file",
	Long: `Export all IAM users to a file in the specified format.

With --detailed, includes group membership counts and group names for each user.`,
	Example: `  # Export users as CSV
  dtiam export users

  # Export with group membership details
  dtiam export users --detailed -f json

  # Export to a specific directory
  dtiam export users -o ./backup -f yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, format, prefix, detailed := getResourceExportFlags(cmd)

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		ctx := context.Background()

		data, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		if detailed {
			for i, user := range data {
				userID := utils.StringFrom(user, "uid")
				groups, err := handler.GetGroups(ctx, userID)
				if err == nil {
					data[i]["group_count"] = len(groups)
					var names []string
					for _, g := range groups {
						if name, ok := g["name"].(string); ok {
							names = append(names, name)
						}
					}
					data[i]["group_names"] = names
				}
			}
		}

		return exportResourceToDir("users", outputDir, format, prefix, data)
	},
}

func init() { addResourceExportFlags(usersCmd) }

var bindingsCmd = &cobra.Command{
	Use:     "bindings",
	Aliases: []string{"binding"},
	Short:   "Export all policy bindings to a file",
	Long: `Export all policy bindings to a file in the specified format.

With --detailed, enriches bindings with group and policy names.`,
	Example: `  # Export bindings as CSV
  dtiam export bindings

  # Export with enriched group/policy names
  dtiam export bindings --detailed -f json

  # Export to a specific directory
  dtiam export bindings -o ./backup`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, format, prefix, detailed := getResourceExportFlags(cmd)

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		data, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		if detailed {
			groupHandler := resources.NewGroupHandler(c)
			policyHandler := resources.NewPolicyHandler(c)

			for i, binding := range data {
				if groupUUID, ok := binding["groupUuid"].(string); ok {
					group, err := groupHandler.Get(ctx, groupUUID)
					if err == nil && group != nil {
						data[i]["group_name"] = group["name"]
					}
				}
				if policyUUID, ok := binding["policyUuid"].(string); ok {
					policy, err := policyHandler.Get(ctx, policyUUID)
					if err == nil && policy != nil {
						data[i]["policy_name"] = policy["name"]
					}
				}
			}
		}

		return exportResourceToDir("bindings", outputDir, format, prefix, data)
	},
}

func init() { addResourceExportFlags(bindingsCmd) }

var boundariesCmd = &cobra.Command{
	Use:     "boundaries",
	Aliases: []string{"boundary"},
	Short:   "Export all boundaries to a file",
	Long: `Export all permission boundaries to a file in the specified format.

With --detailed, includes full boundary details and attached policy counts.`,
	Example: `  # Export boundaries as CSV
  dtiam export boundaries

  # Export with attached policy details
  dtiam export boundaries --detailed -f json

  # Export to a specific directory
  dtiam export boundaries -o ./backup -f yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, format, prefix, detailed := getResourceExportFlags(cmd)

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		data, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		if detailed {
			var detailedData []map[string]any
			for _, boundary := range data {
				boundaryID := utils.StringFrom(boundary, "uuid")
				detail, err := handler.Get(ctx, boundaryID)
				if err == nil && detail != nil {
					attached, err := handler.GetAttachedPolicies(ctx, boundaryID)
					if err == nil {
						detail["attached_policies"] = attached
						detail["attached_policy_count"] = len(attached)
					}
					detailedData = append(detailedData, detail)
				} else {
					detailedData = append(detailedData, boundary)
				}
			}
			data = detailedData
		}

		return exportResourceToDir("boundaries", outputDir, format, prefix, data)
	},
}

func init() { addResourceExportFlags(boundariesCmd) }

var serviceUsersCmd = &cobra.Command{
	Use:     "service-users",
	Aliases: []string{"service-user", "serviceusers", "serviceuser"},
	Short:   "Export all service users to a file",
	Long:    `Export all service users to a file in the specified format.`,
	Example: `  # Export service users as CSV
  dtiam export service-users

  # Export as JSON to a specific directory
  dtiam export service-users -o ./backup -f json

  # Export as YAML
  dtiam export service-users -f yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, format, prefix, _ := getResourceExportFlags(cmd)

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		ctx := context.Background()

		data, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return exportResourceToDir("service-users", outputDir, format, prefix, data)
	},
}

func init() { addResourceExportFlags(serviceUsersCmd) }
