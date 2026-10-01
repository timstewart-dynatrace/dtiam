package group

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create a group and bind policies from a file",
	Long: `One-step group provisioning: create a group and apply policy bindings from a YAML or JSON file.

The policies file should contain a list of policy references with optional boundaries:

  policies:
    - name: "ReadOnly Policy"
      boundaries:
        - "boundary-uuid-1"
    - name: "Admin Policy"

Each policy is resolved by name or UUID, then bound to the newly created group.`,
	Example: `  # Create a group and bind policies
  dtiam group setup --name "New Team" --policies-file policies.yaml

  # With a description
  dtiam group setup --name "New Team" --description "Team description" --policies-file policies.yaml

  # Preview what would be created
  dtiam group setup --name "New Team" --policies-file policies.yaml --dry-run`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		policiesFile, _ := cmd.Flags().GetString("policies-file")

		// Load policies file
		fileData, err := os.ReadFile(policiesFile)
		if err != nil {
			return fmt.Errorf("failed to read policies file: %w", err)
		}

		var fileContent struct {
			Policies []struct {
				Name       string   `json:"name" yaml:"name"`
				UUID       string   `json:"uuid" yaml:"uuid"`
				Boundaries []string `json:"boundaries" yaml:"boundaries"`
			} `json:"policies" yaml:"policies"`
		}

		// Try YAML first, then JSON
		if err := yaml.Unmarshal(fileData, &fileContent); err != nil {
			if err := json.Unmarshal(fileData, &fileContent); err != nil {
				return fmt.Errorf("failed to parse policies file (expected YAML or JSON): %w", err)
			}
		}

		if len(fileContent.Policies) == 0 {
			return fmt.Errorf("no policies found in file")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create group %q and bind %d policies:", name, len(fileContent.Policies))
			for _, p := range fileContent.Policies {
				id := p.Name
				if id == "" {
					id = p.UUID
				}
				if len(p.Boundaries) > 0 {
					fmt.Fprintf(os.Stderr, "  - %s (boundaries: %v)\n", id, p.Boundaries)
				} else {
					fmt.Fprintf(os.Stderr, "  - %s\n", id)
				}
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		policyHandler := resources.NewPolicyHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		ctx := context.Background()

		// Create group
		groupData := map[string]any{"name": name}
		if description != "" {
			groupData["description"] = description
		}
		newGroup, err := groupHandler.Create(ctx, groupData)
		if err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}
		newUUID := utils.StringFrom(newGroup, "uuid")
		printer.PrintSuccess("Group %q created (UUID: %s)", name, newUUID)

		// Bind policies
		for _, p := range fileContent.Policies {
			identifier := p.Name
			if identifier == "" {
				identifier = p.UUID
			}

			policy, err := resources.GetOrResolve(ctx, policyHandler, identifier)
			if err != nil || policy == nil {
				fmt.Fprintf(os.Stderr, "  Warning: policy %q not found, skipping\n", identifier)
				continue
			}
			policyUUID := utils.StringFrom(policy, "uuid")

			if _, err := bindingHandler.Create(ctx, newUUID, policyUUID, p.Boundaries, nil); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to bind policy %q: %v\n", identifier, err)
				continue
			}
			if cli.GlobalState.IsVerbose() {
				fmt.Fprintf(os.Stderr, "  Bound policy: %s\n", identifier)
			}
		}

		printer.PrintSuccess("  Bound %d policy/policies from %s", len(fileContent.Policies), policiesFile)
		return nil
	},
}

func init() {
	setupCmd.Flags().StringP("name", "n", "", "Name for the new group (required)")
	setupCmd.Flags().StringP("description", "d", "", "Group description")
	setupCmd.Flags().StringP("policies-file", "f", "", "YAML or JSON file with policy definitions (required)")
	_ = setupCmd.MarkFlagRequired("name")
	_ = setupCmd.MarkFlagRequired("policies-file")
}
