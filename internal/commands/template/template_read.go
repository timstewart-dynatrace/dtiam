package template

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	tmpl "github.com/timstewart-dynatrace/dtiam/v3/pkg/template"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Long:  `List all available templates including built-in and custom templates.`,
	Example: `  # List all templates
  dtiam template list

  # Output as JSON
  dtiam template list -o json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()

		var all []map[string]any

		for _, t := range tmpl.ListBuiltin() {
			all = append(all, map[string]any{
				"name":      t.Name,
				"source":    t.Source,
				"variables": strings.Join(t.Vars, ", "),
			})
		}

		store, err := tmpl.NewStore()
		if err != nil {
			return err
		}

		custom, err := store.List()
		if err != nil {
			return err
		}
		for _, t := range custom {
			all = append(all, map[string]any{
				"name":      t.Name,
				"source":    t.Source,
				"variables": strings.Join(t.Vars, ", "),
			})
		}

		if len(all) == 0 {
			fmt.Println("No templates found.")
			return nil
		}

		return printer.Print(all, TemplateColumns())
	},
}

var showCmd = &cobra.Command{
	Use:   "show NAME",
	Short: "Show template content and required variables",
	Long:  `Display the content of a template and list its required variables.`,
	Example: `  # Show a built-in template
  dtiam template show policy-readonly

  # Show a custom template
  dtiam template show my-template`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content, source, err := getTemplate(name)
		if err != nil {
			return err
		}

		vars := tmpl.ExtractVariables(string(content))

		fmt.Printf("Template: %s (%s)\n", name, source)
		if len(vars) > 0 {
			fmt.Printf("Variables: %s\n", strings.Join(vars, ", "))
		}
		fmt.Println("---")
		fmt.Print(string(content))
		return nil
	},
}

var renderCmd = &cobra.Command{
	Use:   "render NAME",
	Short: "Render a template with variables",
	Long:  `Render a template to stdout with the given variable substitutions.`,
	Example: `  # Render a policy template
  dtiam template render policy-readonly --set name=MyReadOnlyPolicy

  # Render with multiple variables
  dtiam template render group-team --set name=DevOps --set description="DevOps Team"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		setFlags, _ := cmd.Flags().GetStringSlice("set")

		vars, err := tmpl.ParseSetFlags(setFlags)
		if err != nil {
			return err
		}

		content, _, err := getTemplate(args[0])
		if err != nil {
			return err
		}

		rendered, err := tmpl.RenderTemplate(string(content), vars)
		if err != nil {
			return err
		}

		fmt.Print(rendered)
		return nil
	},
}

func init() {
	renderCmd.Flags().StringSlice("set", nil, "Set template variable as key=value (repeatable)")
}
