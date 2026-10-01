// Package config provides configuration management commands.
package config

import (
	"github.com/spf13/cobra"
)

// Cmd is the config command.
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dtiam contexts, credentials, and configuration",
	Long: `Commands for managing dtiam contexts and credentials.

dtiam uses a kubeconfig-style configuration file to store multiple contexts
and credential sets. Each context references an account UUID and a named
credential. Switch between contexts to manage different Dynatrace accounts.`,
	Example: `  # View the current configuration
  dtiam config view

  # Set up credentials and a context
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY
  dtiam config set-context prod --account-uuid abc-123 --credentials-ref prod
  dtiam config use-context prod

  # List all contexts
  dtiam config get-contexts

  # Show config file location
  dtiam config path`,
}

func init() {
	Cmd.AddCommand(viewCmd)
	Cmd.AddCommand(pathCmd)
	Cmd.AddCommand(getContextsCmd)
	Cmd.AddCommand(currentContextCmd)
	Cmd.AddCommand(useContextCmd)
	Cmd.AddCommand(setContextCmd)
	Cmd.AddCommand(deleteContextCmd)
	Cmd.AddCommand(setCredentialsCmd)
	Cmd.AddCommand(deleteCredentialsCmd)
	Cmd.AddCommand(getCredentialsCmd)
	Cmd.AddCommand(migrateSecretsCmd)
	Cmd.AddCommand(keyringStatusCmd)
}
