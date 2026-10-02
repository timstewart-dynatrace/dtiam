package account

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
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
only ACTIVE subscription is used; if there is none or more than one, specify it.

One row is printed per environment, capability and period. dtiam follows the
API's pages (v3 returns at most 50 records per page).

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

		return printUsageOrCost(printer, result, "usage", output.EnvironmentUsageColumns())
	},
}

var environmentCostCmd = &cobra.Command{
	Use:     "environment-cost [SUBSCRIPTION]",
	Aliases: []string{"env-cost"},
	Short:   "Show subscription cost split by environment",
	Long: `Show Dynatrace Platform Subscription cost broken down by monitoring
environment over a time window.

Per-environment cost and usage both use the v3 Subscription API; subscription
listing and forecast remain on v2. dtiam handles the version difference
internally and follows every page. With no argument, the account's only ACTIVE
subscription is used.

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

		return printUsageOrCost(printer, result, "cost", output.EnvironmentCostColumns())
	},
}

// resolveSubscriptionUUID resolves an optional subscription argument to a UUID.
//
// With no argument it picks the account's single ACTIVE subscription. Accounts
// carry expired and pending subscriptions alongside the current one (ten on the
// account this was verified against), so "the only subscription" almost never
// applied and the commands refused to run without an explicit UUID.
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
	return pickDefaultSubscription(subs)
}

// pickDefaultSubscription chooses the subscription to use when none is named:
// the only subscription, or else the only ACTIVE one.
func pickDefaultSubscription(subs []map[string]any) (string, error) {
	if len(subs) == 0 {
		return "", fmt.Errorf("no subscriptions found for this account")
	}

	candidates := subs
	if len(subs) > 1 {
		candidates = nil
		for _, sub := range subs {
			if status, _ := sub["status"].(string); strings.EqualFold(status, "ACTIVE") {
				candidates = append(candidates, sub)
			}
		}
	}

	switch len(candidates) {
	case 1:
		uuid, _ := candidates[0]["uuid"].(string)
		if uuid == "" {
			return "", fmt.Errorf("the subscription has no UUID")
		}
		return uuid, nil
	case 0:
		return "", fmt.Errorf(
			"this account has %d subscriptions and none is ACTIVE; specify which one (see 'dtiam account subscriptions')",
			len(subs))
	default:
		return "", fmt.Errorf(
			"this account has %d ACTIVE subscriptions; specify which one (see 'dtiam account subscriptions')",
			len(candidates))
	}
}

// printUsageOrCost prints one row per usage or cost record.
func printUsageOrCost(printer *output.Printer, result map[string]any, itemsKey string, columns []output.Column) error {
	return printer.Print(resources.FlattenEnvironmentData(result, itemsKey), columns)
}
