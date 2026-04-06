// Package account provides account management commands.
package account

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the account command.
var Cmd = &cobra.Command{
	Use:   "account",
	Short: "View account limits, capacity, and subscriptions",
	Long: `Commands for viewing account limits and subscriptions.

Use these commands to inspect resource limits, check remaining capacity before
provisioning, list active subscriptions, and view subscription forecasts.`,
	Example: `  # Show all account limits
  dtiam account limits

  # Show limits with usage summary
  dtiam account limits --summary

  # Check capacity for a specific limit
  dtiam account check-capacity user-limit --additional 10

  # List subscriptions
  dtiam account subscriptions

  # View subscription forecast
  dtiam account forecast`,
}

func init() {
	Cmd.AddCommand(limitsCmd)
	Cmd.AddCommand(checkCapacityCmd)
	Cmd.AddCommand(subscriptionsCmd)
	Cmd.AddCommand(forecastCmd)
}

var limitsCmd = &cobra.Command{
	Use:   "limits",
	Short: "List account limits and usage",
	Long: `List all account resource limits.

By default, shows raw limit values. Use --summary to include usage percentages
and highlight limits that are near or at capacity.`,
	Example: `  # List all account limits
  dtiam account limits

  # Show summary with usage percentages
  dtiam account limits --summary

  # Output as JSON
  dtiam account limits -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewLimitsHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		summary, _ := cmd.Flags().GetBool("summary")

		if summary {
			result, err := handler.GetSummary(ctx)
			if err != nil {
				return err
			}

			limits, ok := result["limits"].([]map[string]any)
			if !ok {
				return fmt.Errorf("failed to get limits")
			}

			fmt.Printf("Total limits: %v\n", result["total_limits"])
			fmt.Printf("Near capacity: %v\n", result["limits_near_capacity"])
			fmt.Printf("At capacity: %v\n", result["limits_at_capacity"])
			fmt.Println()

			return printer.Print(limits, output.LimitColumns())
		}

		limits, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(limits, output.LimitColumns())
	},
}

func init() {
	limitsCmd.Flags().Bool("summary", false, "Show summary with usage percentages")
}

var checkCapacityCmd = &cobra.Command{
	Use:   "check-capacity LIMIT_NAME",
	Short: "Check if there is capacity for additional resources",
	Long: `Check whether the account has remaining capacity for a given resource limit.

Specify the limit name as a positional argument and optionally how many additional
resources you plan to provision with --additional (defaults to 1).`,
	Example: `  # Check if one more user can be added
  dtiam account check-capacity user-limit

  # Check if 10 more groups can be created
  dtiam account check-capacity group-limit --additional 10

  # Output as JSON
  dtiam account check-capacity user-limit -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		additional, _ := cmd.Flags().GetInt("additional")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewLimitsHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		result, err := handler.CheckCapacity(ctx, args[0], additional)
		if err != nil {
			return err
		}

		return printer.PrintDetail(result)
	},
}

func init() {
	checkCapacityCmd.Flags().Int("additional", 1, "Number of additional resources to check")
}

var subscriptionsCmd = &cobra.Command{
	Use:   "subscriptions [IDENTIFIER]",
	Short: "List subscriptions or get details for one",
	Long: `List all account subscriptions or retrieve details for a specific subscription.

When called without arguments, lists all subscriptions. When given a subscription
UUID or name, displays full details for that subscription.`,
	Example: `  # List all subscriptions
  dtiam account subscriptions

  # Get details for a specific subscription by UUID
  dtiam account subscriptions abc-123-def-456

  # Get details by name
  dtiam account subscriptions "Enterprise Plan"

  # Output as JSON
  dtiam account subscriptions -o json`,
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

		if len(args) > 0 {
			sub, err := handler.Get(ctx, args[0])
			if err != nil {
				sub, err = handler.GetByName(ctx, args[0])
				if err != nil {
					return err
				}
			}
			if sub == nil {
				return fmt.Errorf("subscription %q not found", args[0])
			}
			return printer.PrintDetail(sub)
		}

		subs, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(subs, output.SubscriptionColumns())
	},
}

var forecastCmd = &cobra.Command{
	Use:   "forecast [SUBSCRIPTION_UUID]",
	Short: "Get subscription usage forecast",
	Long: `Get a usage forecast for subscriptions.

When called without arguments, returns an aggregate forecast for all subscriptions.
When given a specific subscription UUID, returns the forecast for that subscription only.`,
	Example: `  # Get aggregate forecast for all subscriptions
  dtiam account forecast

  # Get forecast for a specific subscription
  dtiam account forecast abc-123-def-456

  # Output as JSON
  dtiam account forecast -o json`,
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

		var subUUID *string
		if len(args) > 0 {
			subUUID = &args[0]
		}

		forecast, err := handler.GetForecast(ctx, subUUID)
		if err != nil {
			return err
		}

		return printer.PrintDetail(forecast)
	},
}
