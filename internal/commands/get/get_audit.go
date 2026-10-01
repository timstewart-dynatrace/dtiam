package get

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var (
	auditStartFlag       string
	auditEndFlag         string
	auditFilterFlag      string
	auditEventTypeFlag   string
	auditUserFlag        string
	auditLimitFlag       int
	auditScanLimitFlag   int
	auditResultLimitFlag int
	auditAddFieldsFlag   []string
)

func init() {
	auditLogsCmd.Flags().StringVar(&auditStartFlag, "start", "",
		"Start of the window (ISO-8601, epoch ms, or relative like 'now-24h')")
	auditLogsCmd.Flags().StringVar(&auditEndFlag, "end", "",
		"End of the window (ISO-8601, epoch ms, or relative like 'now')")
	auditLogsCmd.Flags().StringVar(&auditFilterFlag, "filter", "",
		"Raw audit filter expression (overrides --event-type and --user)")
	auditLogsCmd.Flags().StringVar(&auditEventTypeFlag, "event-type", "",
		"Filter by event type, e.g. CREATE, UPDATE, DELETE")
	auditLogsCmd.Flags().StringVar(&auditUserFlag, "user", "",
		"Filter by the user who performed the action")
	auditLogsCmd.Flags().IntVar(&auditLimitFlag, "limit", 0,
		"Maximum number of entries to return")
	auditLogsCmd.Flags().IntVar(&auditScanLimitFlag, "scan-limit-gb", 0,
		"Server-side scan limit in gigabytes")
	auditLogsCmd.Flags().IntVar(&auditResultLimitFlag, "result-limit-mb", 0,
		"Maximum result size in megabytes")
	auditLogsCmd.Flags().StringSliceVar(&auditAddFieldsFlag, "add-fields", nil,
		"Additional audit fields to include in the response")
}

var auditLogsCmd = &cobra.Command{
	Use:     "audit-logs",
	Aliases: []string{"audit-log", "audits", "audit"},
	Short:   "List account audit log entries",
	Long: `List audit log entries for the Dynatrace account.

The audit log records who changed what in the account: group membership edits,
policy binding changes, user provisioning, and token creation. It is the only
API that answers "who did this", so it is the starting point for any access
review or incident investigation.

Requires the account-audit-logs-read OAuth scope.

The API enforces server-side scan and result-size limits. When a limit is hit it
returns a partial result with a warning rather than an error; dtiam surfaces
those warnings on stderr so a truncated audit trail is never presented as
complete.`,
	Example: `  # Entries from the last 24 hours
  dtiam get audit-logs --start now-24h

  # Only deletions, most recent 50
  dtiam get audit-logs --start now-7d --event-type DELETE --limit 50

  # Everything one user did last week
  dtiam get audit-logs --start now-7d --user alice@example.com

  # A raw filter expression for anything the flags do not cover
  dtiam get audit-logs --start now-30d --filter 'resource = "GROUP"'

  # Bound an expensive query on an account with a long history
  dtiam get audit-logs --start now-90d --scan-limit-gb 5

  # Machine-friendly (skip prompts, JSON output)
  dtiam get audit-logs --start now-24h --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewAuditHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		query := resources.AuditQuery{
			StartTime:               auditStartFlag,
			EndTime:                 auditEndFlag,
			Filter:                  buildAuditFilter(),
			AddFields:               auditAddFieldsFlag,
			Limit:                   auditLimitFlag,
			ScanLimitGigabyte:       auditScanLimitFlag,
			ResultSizeLimitMegabyte: auditResultLimitFlag,
		}

		if cli.GlobalState.IsVerbose() {
			fmt.Fprintf(os.Stderr, "Querying audit log (start=%q end=%q filter=%q)\n",
				query.StartTime, query.EndTime, query.Filter)
		}

		result, err := handler.Query(ctx, query)
		if err != nil {
			return err
		}

		// Warnings go to stderr so stdout stays parseable, but they must be loud:
		// a partial audit result that looks complete is actively misleading.
		for _, w := range result.Warnings {
			printer.PrintWarning("Audit API warning: %s", w)
		}

		if cli.GlobalState.IsVerbose() {
			fmt.Fprintf(os.Stderr, "Retrieved %d audit entries\n", len(result.Audits))
		}

		return printer.Print(result.Audits, output.AuditColumns())
	},
}

// buildAuditFilter turns the convenience flags into an audit filter expression.
// An explicit --filter wins, so users are never blocked by the flags' coverage.
func buildAuditFilter() string {
	if auditFilterFlag != "" {
		return auditFilterFlag
	}

	var clauses []string
	if auditEventTypeFlag != "" {
		clauses = append(clauses, fmt.Sprintf("eventType = %q", auditEventTypeFlag))
	}
	if auditUserFlag != "" {
		clauses = append(clauses, fmt.Sprintf("user = %q", auditUserFlag))
	}

	filter := ""
	for i, clause := range clauses {
		if i > 0 {
			filter += " and "
		}
		filter += clause
	}
	return filter
}
