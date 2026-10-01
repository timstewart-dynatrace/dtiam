// Package template provides template management commands.
package template

import (
	"github.com/spf13/cobra"
)

// Cmd is the template command.
var Cmd = &cobra.Command{
	Use:   "template",
	Short: "Manage and use resource templates",
	Long: `Commands for managing IAM resource templates.

Templates are reusable YAML definitions with variable placeholders that can be
rendered and applied to create resources. Built-in templates are provided for
common patterns. Custom templates can be saved and managed.`,
	Example: `  # List all available templates
  dtiam template list

  # Show a template's content and required variables
  dtiam template show policy-readonly

  # Render a template with variables
  dtiam template render policy-readonly --set name=MyPolicy

  # Render and create the resource
  dtiam template apply policy-readonly --set name=MyPolicy

  # Save a custom template
  dtiam template save my-template --file template.yaml

  # Show templates directory
  dtiam template path`,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(renderCmd)
	Cmd.AddCommand(applyCmd)
	Cmd.AddCommand(saveCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(pathCmd)
}
