// Package token provides platform token lifecycle commands.
package token

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/prompt"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

// Cmd is the token command.
var Cmd = &cobra.Command{
	Use:     "token",
	Aliases: []string{"tokens"},
	Short:   "Platform token lifecycle commands",
	Long: `Commands for managing existing platform tokens: deactivate, reactivate, and
change when they expire.

Create, list and delete tokens with "dtiam create token", "dtiam get tokens" and
"dtiam delete token". Tokens are identified by token ID (the TOKEN ID column of
"get tokens"). Requires the platform-token:tokens:manage OAuth scope.`,
	Example: `  # Temporarily disable a token
  dtiam token deactivate dt0s16.ABC123

  # Re-enable it
  dtiam token activate dt0s16.ABC123

  # Extend its expiration
  dtiam token set-expiration dt0s16.ABC123 --date 2027-06-30T00:00:00Z`,
}

func init() {
	Cmd.AddCommand(deactivateCmd)
	Cmd.AddCommand(activateCmd)
	Cmd.AddCommand(setExpirationCmd)

	deactivateCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	setExpirationCmd.Flags().String("date", "", "New expiration date, RFC 3339 (e.g. 2027-01-01T00:00:00Z)")
	_ = setExpirationCmd.MarkFlagRequired("date")
}

var deactivateCmd = &cobra.Command{
	Use:   "deactivate TOKEN_ID",
	Short: "Deactivate a platform token so it stops authenticating",
	Long: `Set a platform token's status to INACTIVE. Anything using the token loses
access immediately. The token is kept and can be re-enabled with
"dtiam token activate"; use "dtiam delete token" to remove it permanently.

Asks for confirmation unless --force or --plain is set.`,
	Example: `  # Deactivate with confirmation
  dtiam token deactivate dt0s16.ABC123

  # Preview
  dtiam token deactivate dt0s16.ABC123 --dry-run

  # Without a prompt (automation)
  dtiam token deactivate dt0s16.ABC123 --force --plain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would deactivate platform token %s", args[0])
			return nil
		}
		if !prompt.Confirm(fmt.Sprintf("Deactivate platform token %q? Anything using it will lose access.", args[0]),
			force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		return setStatus(args[0], resources.TokenStatusInactive, "deactivated")
	},
}

var activateCmd = &cobra.Command{
	Use:   "activate TOKEN_ID",
	Short: "Reactivate a deactivated platform token",
	Long:  `Set a platform token's status back to ACTIVE so it authenticates again.`,
	Example: `  # Reactivate a token
  dtiam token activate dt0s16.ABC123

  # Preview
  dtiam token activate dt0s16.ABC123 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cli.GlobalState.IsDryRun() {
			cli.GlobalState.NewPrinter().PrintWarning("Would activate platform token %s", args[0])
			return nil
		}
		return setStatus(args[0], resources.TokenStatusActive, "activated")
	},
}

var setExpirationCmd = &cobra.Command{
	Use:   "set-expiration TOKEN_ID --date DATE",
	Short: "Change when a platform token expires",
	Long: `Change a platform token's expiration date. The date is RFC 3339, e.g.
2027-01-01T00:00:00Z. It is checked locally before any request is made.`,
	Example: `  # Extend a token
  dtiam token set-expiration dt0s16.ABC123 --date 2027-06-30T00:00:00Z

  # Preview
  dtiam token set-expiration dt0s16.ABC123 --date 2027-06-30T00:00:00Z --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		date, _ := cmd.Flags().GetString("date")
		printer := cli.GlobalState.NewPrinter()

		// Validate before the dry-run check, so a preview never approves a date
		// the real call would reject.
		if _, err := time.Parse(time.RFC3339, date); err != nil {
			return fmt.Errorf("invalid --date %q: use RFC 3339, e.g. 2027-01-01T00:00:00Z", date)
		}

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would set platform token %s to expire at %s", args[0], date)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := resources.NewTokenHandler(c).SetExpiration(context.Background(), args[0], date); err != nil {
			return err
		}
		printer.PrintSuccess("Platform token %q now expires at %s", args[0], date)
		return nil
	},
}

// setStatus applies a status change and reports it.
func setStatus(tokenID, status, verb string) error {
	c, err := common.CreateClient()
	if err != nil {
		return err
	}
	defer c.Close()

	if err := resources.NewTokenHandler(c).SetStatus(context.Background(), tokenID, status); err != nil {
		return err
	}
	cli.GlobalState.NewPrinter().PrintSuccess("Platform token %q %s", tokenID, verb)
	return nil
}
