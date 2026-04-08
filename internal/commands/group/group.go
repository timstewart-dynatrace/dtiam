// Package group provides advanced group management commands.
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
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the group command.
var Cmd = &cobra.Command{
	Use:   "group",
	Short: "Advanced group management commands",
	Long: `Commands for advanced group operations such as listing members,
adding and removing members, and viewing policy bindings for a group.

Groups can be identified by UUID or name in all subcommands.`,
	Example: `  # List members of a group
  dtiam group members "Production Team"

  # Add a user to a group
  dtiam group add-member "Production Team" --email user@example.com

  # View policy bindings for a group
  dtiam group bindings "Production Team"`,
}

func init() {
	Cmd.AddCommand(membersCmd)
	Cmd.AddCommand(addMemberCmd)
	Cmd.AddCommand(removeMemberCmd)
	Cmd.AddCommand(bindingsCmd)
	Cmd.AddCommand(cloneCmd)
	Cmd.AddCommand(setupCmd)
}

var membersCmd = &cobra.Command{
	Use:   "members IDENTIFIER",
	Short: "List members of a group",
	Long: `List all members of a group.

The group can be identified by UUID or name. Returns user details
for all members in table format by default.`,
	Example: `  # List members by group name
  dtiam group members "Production Team"

  # List members by group UUID
  dtiam group members 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam group members "Production Team" -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		members, err := handler.GetMembers(ctx, uuid)
		if err != nil {
			return err
		}

		return printer.Print(members, output.UserColumns())
	},
}

var addMemberCmd = &cobra.Command{
	Use:   "add-member IDENTIFIER",
	Short: "Add a user to a group",
	Long: `Add a user to a group by specifying the group and user email.

The group can be identified by UUID or name. The user is specified
via the --email flag.`,
	Example: `  # Add a user to a group by name
  dtiam group add-member "Production Team" --email user@example.com

  # Add a user to a group by UUID
  dtiam group add-member 8f6e5d4c-3b2a-1098-7654-321fedcba098 --email user@example.com

  # Dry run preview
  dtiam group add-member "Production Team" --email user@example.com --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			return fmt.Errorf("--email is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would add user %s to group %s", email, args[0])
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		if err := handler.AddMember(ctx, uuid, email); err != nil {
			return err
		}

		printer.PrintSuccess("User %s added to group", email)
		return nil
	},
}

func init() {
	addMemberCmd.Flags().StringP("email", "e", "", "User email to add (required)")
}

var removeMemberCmd = &cobra.Command{
	Use:   "remove-member IDENTIFIER",
	Short: "Remove a user from a group",
	Long: `Remove a user from a group by specifying the group and user UID.

The group can be identified by UUID or name. The user is specified
via the --user flag with their UID.`,
	Example: `  # Remove a user from a group by name
  dtiam group remove-member "Production Team" --user USER_UID

  # Remove a user from a group by UUID
  dtiam group remove-member 8f6e5d4c-3b2a-1098-7654-321fedcba098 --user USER_UID

  # Dry run preview
  dtiam group remove-member "Production Team" --user USER_UID --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, _ := cmd.Flags().GetString("user")
		if userID == "" {
			return fmt.Errorf("--user is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would remove user %s from group %s", userID, args[0])
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		if err := handler.RemoveMember(ctx, uuid, userID); err != nil {
			return err
		}

		printer.PrintSuccess("User removed from group")
		return nil
	},
}

func init() {
	removeMemberCmd.Flags().StringP("user", "u", "", "User UID to remove (required)")
}

var cloneCmd = &cobra.Command{
	Use:   "clone SOURCE",
	Short: "Clone a group with optional members and policies",
	Long: `Clone an existing group to create a new group with the same configuration.

By default, only the group name and description are cloned. Use --include-members
to copy all group members and --include-policies to copy all policy bindings
(including boundaries) to the new group.`,
	Example: `  # Clone a group with a new name
  dtiam group clone "Production Team" --name "Staging Team"

  # Clone with members and policies
  dtiam group clone "Production Team" --name "Staging Team" --include-members --include-policies

  # Clone with a custom description
  dtiam group clone "Production Team" --name "Staging Team" --description "Staging environment team"

  # Preview what would be cloned
  dtiam group clone "Production Team" --name "Staging Team" --include-members --include-policies --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		includeMembers, _ := cmd.Flags().GetBool("include-members")
		includePolicies, _ := cmd.Flags().GetBool("include-policies")

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would clone group %q as %q", args[0], name)
			if includeMembers {
				printer.PrintWarning("  --include-members: would copy all members")
			}
			if includePolicies {
				printer.PrintWarning("  --include-policies: would copy all policy bindings")
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		ctx := context.Background()

		// Resolve source group
		source, err := resources.GetOrResolve(ctx, groupHandler, args[0])
		if err != nil {
			return err
		}
		if source == nil {
			return fmt.Errorf("source group %q not found", args[0])
		}
		sourceUUID := utils.StringFrom(source, "uuid")

		// Use source description if none provided
		if description == "" {
			if srcDesc, ok := source["description"].(string); ok {
				description = srcDesc
			}
		}

		// Create new group
		newGroup, err := groupHandler.Create(ctx, map[string]any{
			"name":        name,
			"description": description,
		})
		if err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}
		newUUID := utils.StringFrom(newGroup, "uuid")
		printer.PrintSuccess("Group %q created (UUID: %s)", name, newUUID)

		// Copy members
		if includeMembers {
			members, err := groupHandler.GetMembers(ctx, sourceUUID)
			if err != nil {
				return fmt.Errorf("failed to get source members: %w", err)
			}
			for _, member := range members {
				email, _ := member["email"].(string)
				if email == "" {
					continue
				}
				if err := groupHandler.AddMember(ctx, newUUID, email); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to add member %s: %v\n", email, err)
					continue
				}
				if cli.GlobalState.IsVerbose() {
					fmt.Fprintf(os.Stderr, "  Added member: %s\n", email)
				}
			}
			printer.PrintSuccess("  Copied %d member(s)", len(members))
		}

		// Copy policy bindings
		if includePolicies {
			bindingHandler := resources.NewBindingHandler(c)
			bindings, err := bindingHandler.GetForGroup(ctx, sourceUUID)
			if err != nil {
				return fmt.Errorf("failed to get source bindings: %w", err)
			}
			for _, binding := range bindings {
				policyUUID := utils.StringFrom(binding, "policyUuid")
				var boundaries []string
				if bList, ok := binding["boundaries"].([]string); ok {
					boundaries = bList
				}
				if _, err := bindingHandler.Create(ctx, newUUID, policyUUID, boundaries, nil); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to copy binding for policy %s: %v\n", policyUUID, err)
					continue
				}
				if cli.GlobalState.IsVerbose() {
					fmt.Fprintf(os.Stderr, "  Copied binding: policy %s\n", policyUUID)
				}
			}
			printer.PrintSuccess("  Copied %d binding(s)", len(bindings))
		}

		return nil
	},
}

func init() {
	cloneCmd.Flags().StringP("name", "n", "", "Name for the new group (required)")
	cloneCmd.Flags().StringP("description", "d", "", "Description for the new group")
	cloneCmd.Flags().Bool("include-members", false, "Copy group members to the new group")
	cloneCmd.Flags().Bool("include-policies", false, "Copy policy bindings to the new group")
	_ = cloneCmd.MarkFlagRequired("name")
}

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

var bindingsCmd = &cobra.Command{
	Use:   "bindings IDENTIFIER",
	Short: "List policy bindings for a group",
	Long: `List all policy bindings associated with a group.

The group can be identified by UUID or name. Returns binding details
including the policy and level information.`,
	Example: `  # List bindings by group name
  dtiam group bindings "Production Team"

  # List bindings by group UUID
  dtiam group bindings 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam group bindings "Production Team" -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, groupHandler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		bindings, err := bindingHandler.GetForGroup(ctx, uuid)
		if err != nil {
			return err
		}

		return printer.Print(bindings, output.BindingColumns())
	},
}
