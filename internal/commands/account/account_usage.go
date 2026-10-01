package account

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/output"
	"github.com/jtimothystewart/dtiam/pkg/resources"
)

var (
	usageStartFlag        string
	usageEndFlag          string
	usageEnvironmentsFlag []string
	usageCapabilitiesFlag []string
	costStartFlag         string
	costEndFlag           string
)

func init() {
	environmentUsageCmd.Flags().StringVar(&usageStartFlag, "start", "",
		"Start of the window, e.g. 2026-09-01T00:00:00Z (required)")
	environmentUsageCmd.Flags().StringVar(&usageEndFlag, "end", "",
		"End of the window, e.g. 2026-10-01T00:00:00Z (required)")
	environmentUsageCmd.Flags().StringSliceVar(&usageEnvironmentsFlag, "environment", nil,
		"Restrict to these environment IDs")
	environmentUsageCmd.Flags().StringSliceVar(&usageCapabilitiesFlag, "capability", nil,
		"Restrict to these capability keys")
	_ = environmentUsageCmd.MarkFlagRequired("start")
	_ = environmentUsageCmd.MarkFlagRequired("end")

	environmentCostCmd.Flags().StringVar(&costStartFlag, "start", "",
		"Start of the window, e.g. 2026-09-01T00:00:00Z (required)")
	environmentCostCmd.Flags().StringVar(&costEndFlag, "end", "",
		"End of the window, e.g. 2026-10-01T00:00:00Z (required)")
	_ = environmentCostCmd.MarkFlagRequired("start")
	_ = environmentCostCmd.MarkFlagRequired("end")
}

var environmentUsageCmd = &cobra.Command{
	Use:     "environment-usage [SUBSCRIPTION]",
	Aliases: []string{"env-usage"},
	Short:   "Show subscription usage split by environment",
	Long: `Show Dynatrace Platform Subscription usage broken down by monitoring
environment over a time window.

This is distinct from "dtiam account subscriptions", which reports only the usage
totals embedded in the subscription record. This command calls the dedicated
per-environment usage endpoint, so it can attribute consumption to individual
environments.

The subscription may be given by UUID or name. With no argument, the account's
only subscription is used; if there are several, the UUID is required.

Requires the account-uac-read OAuth scope.`,
	Example: `  # Usage per environment for last month
  dtiam account environment-usage --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z

  # A specific subscription
  dtiam account environment-usage abc-123 \
    --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z

  # Narrow to one environment and capability
  dtiam account environment-usage \
    --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z \
    --environment abc12345 --capability full_stack_monitoring

  # Machine-friendly output
  dtiam account environment-usage \
    --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewSubscriptionHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		subUUID, err := resolveSubscriptionUUID(ctx, handler, args)
		if err != nil {
			return err
		}

		result, err := handler.EnvironmentUsage(
			ctx, subUUID, usageStartFlag, usageEndFlag, usageEnvironmentsFlag, usageCapabilitiesFlag)
		if err != nil {
			return err
		}

		return printUsageOrCost(printer, result, output.EnvironmentUsageColumns())
	},
}

var environmentCostCmd = &cobra.Command{
	Use:     "environment-cost [SUBSCRIPTION]",
	Aliases: []string{"env-cost"},
	Short:   "Show subscription cost split by environment",
	Long: `Show Dynatrace Platform Subscription cost broken down by monitoring
environment over a time window.

Note this endpoint lives on the v3 Subscription API while subscription listing,
usage, and forecast remain on v2. dtiam handles the version difference
internally.

Requires the account-uac-read OAuth scope.`,
	Example: `  # Cost per environment for last month
  dtiam account environment-cost --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z

  # A specific subscription, as JSON
  dtiam account environment-cost abc-123 \
    --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z -o json

  # Machine-friendly output
  dtiam account environment-cost \
    --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewSubscriptionHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		subUUID, err := resolveSubscriptionUUID(ctx, handler, args)
		if err != nil {
			return err
		}

		result, err := handler.EnvironmentCost(ctx, subUUID, costStartFlag, costEndFlag)
		if err != nil {
			return err
		}

		return printUsageOrCost(printer, result, output.EnvironmentCostColumns())
	},
}

// resolveSubscriptionUUID resolves an optional subscription argument to a UUID.
//
// With no argument it falls back to the account's single subscription, and
// reports the ambiguity rather than guessing when there is more than one.
func resolveSubscriptionUUID(
	ctx context.Context, handler *resources.SubscriptionHandler, args []string,
) (string, error) {
	if len(args) > 0 {
		sub, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return "", err
		}
		if sub == nil {
			return "", fmt.Errorf("subscription %q not found", args[0])
		}
		uuid, _ := sub["uuid"].(string)
		if uuid == "" {
			return "", fmt.Errorf("subscription %q has no UUID", args[0])
		}
		return uuid, nil
	}

	subs, err := handler.List(ctx, nil)
	if err != nil {
		return "", err
	}
	switch len(subs) {
	case 0:
		return "", fmt.Errorf("no subscriptions found for this account")
	case 1:
		uuid, _ := subs[0]["uuid"].(string)
		if uuid == "" {
			return "", fmt.Errorf("the account's subscription has no UUID")
		}
		return uuid, nil
	default:
		return "", fmt.Errorf(
			"this account has %d subscriptions; specify which one (see 'dtiam account subscriptions')",
			len(subs))
	}
}

// printUsageOrCost renders the data array as a table when present, and falls
// back to the whole structure otherwise.
//
// Both endpoints wrap their rows in {data, lastModifiedTime}. Printing the rows
// as a table is the useful default, but the shape varies by capability, so an
// unexpected payload is shown in full rather than silently rendered as empty.
func printUsageOrCost(printer *output.Printer, result map[string]any, columns []output.Column) error {
	raw, ok := result["data"]
	if !ok {
		return printer.PrintAny(result)
	}

	rows, ok := raw.([]any)
	if !ok {
		return printer.PrintAny(result)
	}

	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if m, ok := row.(map[string]any); ok {
			items = append(items, m)
		}
	}
	if len(items) == 0 {
		return printer.PrintAny(result)
	}

	return printer.Print(items, columns)
}
