package account

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var (
	notificationsStartFlag      string
	notificationsEndFlag        string
	notificationsTypesFlag      []string
	notificationsSeveritiesFlag []string
	notificationsEnvsFlag       []string
	notificationsCapsFlag       []string
)

func init() {
	notificationsCmd.Flags().StringVar(&notificationsStartFlag, "start", "",
		"Start of the window (ISO-8601)")
	notificationsCmd.Flags().StringVar(&notificationsEndFlag, "end", "",
		"End of the window (ISO-8601)")
	notificationsCmd.Flags().StringSliceVar(&notificationsTypesFlag, "type", nil,
		"Filter by type: FORECAST, BUDGET, COST, BYOK_REVOKED, BYOK_ACTIVATED, ENVIRONMENT_UPGRADE, ENVIRONMENT_DOWNGRADE")
	notificationsCmd.Flags().StringSliceVar(&notificationsSeveritiesFlag, "severity", nil,
		"Filter by severity: SEVERE, WARN, INFO")
	notificationsCmd.Flags().StringSliceVar(&notificationsEnvsFlag, "environment", nil,
		"Filter by environment ID (repeatable or comma-separated)")
	notificationsCmd.Flags().StringSliceVar(&notificationsCapsFlag, "capability", nil,
		"Filter by capability key, e.g. FULLSTACK_MONITORING")
}

var notificationsCmd = &cobra.Command{
	Use:     "notifications",
	Aliases: []string{"notification"},
	Short:   "List account notifications",
	Long: `List account-level notifications: budget, cost, forecast,
bring-your-own-key, and environment upgrade/downgrade events, newest first.

Every matching notification is returned; dtiam follows the API's pages.
Requires the account-uac-read OAuth scope.

Filter values are validated locally, so a mistyped type or severity fails with a
clear message instead of silently matching nothing.`,
	Example: `  # All recent notifications
  dtiam account notifications

  # Only severe ones
  dtiam account notifications --severity SEVERE

  # Budget and cost notifications for a window
  dtiam account notifications --type BUDGET,COST \
    --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z

  # Notifications for one environment and capability
  dtiam account notifications --environment abc12345 --capability FULLSTACK_MONITORING

  # Machine-friendly output
  dtiam account notifications --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		query := resources.NotificationQuery{
			StartDateTime: notificationsStartFlag,
			EndDateTime:   notificationsEndFlag,
			Types:         upperAll(notificationsTypesFlag),
			Severities:    upperAll(notificationsSeveritiesFlag),
			Environments:  notificationsEnvsFlag,
			Capabilities:  upperAll(notificationsCapsFlag),
		}

		// Validate before creating a client so a typo does not cost an SSO round trip.
		if err := query.Validate(); err != nil {
			return err
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		printer := cli.GlobalState.NewPrinter()

		result, err := resources.NewNotificationHandler(c).Query(context.Background(), query)
		if err != nil {
			return err
		}

		if result.HasMore {
			printer.PrintWarning(
				"Stopped after %d notifications (page limit reached); narrow the window to see the rest",
				result.TotalCount)
		}

		return printer.Print(result.Records, output.NotificationColumns())
	},
}

// upperAll upper-cases every element, so users can pass lowercase filter values.
func upperAll(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strings.ToUpper(strings.TrimSpace(v)))
	}
	return out
}
