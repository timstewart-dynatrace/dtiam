package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "Display the current configuration",
	Long: `Display the full configuration file in YAML format.

Secrets are masked by default. Use --show-secrets to reveal client secrets.`,
	Example: `  # View configuration with masked secrets
  dtiam config view

  # View configuration with secrets visible
  dtiam config view --show-secrets

  # Pipe to a file for backup
  dtiam config view --show-secrets > config-backup.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Mask secrets unless --show-secrets is provided
		showSecrets, _ := cmd.Flags().GetBool("show-secrets")
		if !showSecrets {
			for i := range cfg.Credentials {
				cfg.Credentials[i].Credential.ClientSecret = config.MaskSecret(cfg.Credentials[i].Credential.ClientSecret)
			}
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	viewCmd.Flags().Bool("show-secrets", false, "Show unmasked secrets")
}

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Display the configuration file path",
	Long:  "Print the absolute path to the dtiam configuration file.",
	Example: `  # Show the config file path
  dtiam config path`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.GetConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}
