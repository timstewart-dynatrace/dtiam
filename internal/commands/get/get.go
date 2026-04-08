// Package get provides commands for listing and retrieving resources.
package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the get command.
var Cmd = &cobra.Command{
	Use:   "get",
	Short: "List or retrieve resources",
	Long:  "Commands for listing and retrieving IAM resources.",
}

func init() {
	Cmd.AddCommand(groupsCmd)
	Cmd.AddCommand(usersCmd)
	Cmd.AddCommand(policiesCmd)
	Cmd.AddCommand(bindingsCmd)
	Cmd.AddCommand(environmentsCmd)
	Cmd.AddCommand(boundariesCmd)
	Cmd.AddCommand(tokensCmd)
	Cmd.AddCommand(appsCmd)
	Cmd.AddCommand(schemasCmd)
}

var groupsCmd = &cobra.Command{
	Use:     "groups [identifier]",
	Aliases: []string{"group"},
	Short:   "List IAM groups or get a specific group by UUID or name",
	Example: `  # List all groups
  dtiam get groups

  # Get a specific group by UUID
  dtiam get groups 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific group by name
  dtiam get groups "My Team"

  # Output as JSON
  dtiam get groups -o json

  # Output as YAML
  dtiam get groups -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get groups --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single group
			group, err := resources.GetOrResolve(ctx, handler, args[0])
			if err != nil {
				return err
			}
			if group == nil {
				return fmt.Errorf("group %q not found", args[0])
			}
			return printer.PrintSingle(group, output.GroupColumns())
		}

		// List all groups
		groups, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.GroupColumns())
	},
}

var usersCmd = &cobra.Command{
	Use:     "users [identifier]",
	Aliases: []string{"user"},
	Short:   "List IAM users or get a specific user by UID or email",
	Example: `  # List all users
  dtiam get users

  # Get a specific user by UID
  dtiam get users 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific user by email
  dtiam get users alice@example.com

  # Output as JSON
  dtiam get users -o json

  # Output as YAML
  dtiam get users -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get users --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single user
			user, err := handler.Get(ctx, args[0])
			if err != nil {
				// Try by email
				user, err = handler.GetByEmail(ctx, args[0])
				if err != nil {
					return err
				}
			}
			if user == nil {
				return fmt.Errorf("user %q not found", args[0])
			}
			return printer.PrintSingle(user, output.UserColumns())
		}

		// List all users
		users, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(users, output.UserColumns())
	},
}

var policiesCmd = &cobra.Command{
	Use:     "policies [identifier]",
	Aliases: []string{"policy"},
	Short:   "List IAM policies or get a specific policy by UUID or name",
	Example: `  # List all account-level policies
  dtiam get policies

  # Get a specific policy by UUID
  dtiam get policies 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific policy by name
  dtiam get policies "AppEngine - Reader"

  # List policies at a specific level
  dtiam get policies --level environment --level-id abc12345

  # List policies from all levels (account, environment, global)
  dtiam get policies --all-levels

  # Output as JSON
  dtiam get policies -o json

  # Output as YAML
  dtiam get policies --all-levels -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get policies --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		level, _ := cmd.Flags().GetString("level")
		levelID, _ := cmd.Flags().GetString("level-id")

		var handler *resources.PolicyHandler
		if level != "" && level != "account" {
			if levelID == "" {
				return fmt.Errorf("--level-id is required when using --level")
			}
			handler = resources.NewPolicyHandlerWithLevel(c, level, levelID)
		} else {
			handler = resources.NewPolicyHandler(c)
		}

		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single policy
			policy, err := resources.GetOrResolve(ctx, handler, args[0])
			if err != nil {
				return err
			}
			if policy == nil {
				return fmt.Errorf("policy %q not found", args[0])
			}
			return printer.PrintSingle(policy, output.PolicyColumns())
		}

		// List policies
		allLevels, _ := cmd.Flags().GetBool("all-levels")
		var policies []map[string]any

		if allLevels {
			policies, err = handler.ListAllLevels(ctx)
		} else {
			policies, err = handler.List(ctx, nil)
		}

		if err != nil {
			return err
		}

		return printer.Print(policies, output.PolicyColumns())
	},
}

func init() {
	policiesCmd.Flags().String("level", "account", "Policy level (account, environment, global)")
	policiesCmd.Flags().String("level-id", "", "Level ID (required for environment level)")
	policiesCmd.Flags().Bool("all-levels", false, "List policies from all levels")
}

var bindingsCmd = &cobra.Command{
	Use:     "bindings",
	Aliases: []string{"binding"},
	Short:   "List policy-to-group bindings, optionally filtered by group or policy",
	Example: `  # List all bindings
  dtiam get bindings

  # Filter bindings by group UUID
  dtiam get bindings --group 12345678-abcd-1234-abcd-1234567890ab

  # Filter bindings by policy UUID
  dtiam get bindings --policy 87654321-dcba-4321-dcba-ba0987654321

  # Output as JSON
  dtiam get bindings -o json

  # Output as YAML
  dtiam get bindings -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get bindings --plain`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")

		var bindings []map[string]any

		if groupID != "" {
			bindings, err = handler.GetForGroup(ctx, groupID)
		} else if policyID != "" {
			binding, err := handler.GetForPolicy(ctx, policyID)
			if err != nil {
				return err
			}
			bindings = []map[string]any{binding}
		} else {
			bindings, err = handler.List(ctx, nil)
		}

		if err != nil {
			return err
		}

		return printer.Print(bindings, output.BindingColumns())
	},
}

func init() {
	bindingsCmd.Flags().String("group", "", "Filter bindings by group UUID")
	bindingsCmd.Flags().String("policy", "", "Filter bindings by policy UUID")
}

var environmentsCmd = &cobra.Command{
	Use:     "environments [identifier]",
	Aliases: []string{"envs", "env"},
	Short:   "List Dynatrace environments or get a specific environment by ID or name",
	Example: `  # List all environments
  dtiam get environments

  # Get a specific environment by ID
  dtiam get environments abc12345

  # Get a specific environment by name
  dtiam get environments "Production"

  # Output as JSON
  dtiam get environments -o json

  # Output as YAML
  dtiam get environments -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get environments --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewEnvironmentHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single environment
			env, err := handler.Get(ctx, args[0])
			if err != nil {
				// Try by name
				env, err = handler.GetByName(ctx, args[0])
				if err != nil {
					return err
				}
			}
			if env == nil {
				return fmt.Errorf("environment %q not found", args[0])
			}
			return printer.PrintSingle(env, output.EnvironmentColumns())
		}

		// List all environments
		envs, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(envs, output.EnvironmentColumns())
	},
}

var boundariesCmd = &cobra.Command{
	Use:     "boundaries [identifier]",
	Aliases: []string{"boundary"},
	Short:   "List permission boundaries or get a specific boundary by UUID or name",
	Example: `  # List all boundaries
  dtiam get boundaries

  # Get a specific boundary by UUID
  dtiam get boundaries 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific boundary by name
  dtiam get boundaries "Production MZ Only"

  # Output as JSON
  dtiam get boundaries -o json

  # Output as YAML
  dtiam get boundaries -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get boundaries --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single boundary
			boundary, err := resources.GetOrResolve(ctx, handler, args[0])
			if err != nil {
				return err
			}
			if boundary == nil {
				return fmt.Errorf("boundary %q not found", args[0])
			}
			return printer.PrintSingle(boundary, output.BoundaryColumns())
		}

		// List all boundaries
		boundaries, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(boundaries, output.BoundaryColumns())
	},
}

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

var (
	appsEnvironmentFlag   string
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

		c, err := common.CreateClient()
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

		c, err := common.CreateClient()
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
