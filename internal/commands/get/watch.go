package get

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/watch"
)

// addWatchFlags registers the watch flags on a get subcommand.
func addWatchFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("watch", "w", false,
		"Poll and reprint when the result changes (Ctrl-C to stop)")
	cmd.Flags().Duration("watch-interval", watch.DefaultInterval,
		"Polling interval for --watch")
}

// watchRequested reports whether --watch was passed.
func watchRequested(cmd *cobra.Command) bool {
	on, _ := cmd.Flags().GetBool("watch")
	return on
}

// runWatch polls list and reprints whenever the result changes.
//
// It returns nil on a clean stop (Ctrl-C or SIGTERM), because the user asking the
// watch to end is not a failure. A poll error is reported and the watch continues:
// a transient API hiccup should not kill a session someone left running.
func runWatch(cmd *cobra.Command, list watch.ListFunc, columns []output.Column) error {
	interval, _ := cmd.Flags().GetDuration("watch-interval")
	printer := cli.GlobalState.NewPrinter()

	// Watching is inherently interactive. In --plain the output is a JSON stream
	// with no delimiter, which is not something a caller can parse incrementally,
	// so refuse rather than emit a format nobody can consume.
	if cli.GlobalState.IsPlain() {
		return fmt.Errorf(
			"--watch is not supported with --plain: the output would be an unparseable JSON stream. " +
				"Poll the command on a timer instead")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	events := make(chan watch.Event)
	go watch.Watch(ctx, list, watch.Options{Interval: interval}, events)

	fmt.Fprintf(os.Stderr, "Watching every %s. Press Ctrl-C to stop.\n", effectiveInterval(interval))

	for ev := range events {
		if ev.Err != nil {
			// Keep going: the next poll may succeed.
			printer.PrintWarning("Poll %d failed: %v", ev.Iteration, ev.Err)
			continue
		}

		if !ev.Changed {
			continue
		}

		// Separate successive renders so a scrollback is readable. The timestamp
		// goes to stderr so stdout remains just the data.
		if ev.Iteration > 1 {
			fmt.Fprintf(os.Stderr, "\n--- changed at %s (%d item(s)) ---\n",
				time.Now().Format(time.RFC3339), len(ev.Items))
		}

		if err := printer.Print(ev.Items, columns); err != nil {
			return err
		}
	}

	fmt.Fprintln(os.Stderr, "\nWatch stopped.")
	return nil
}

// effectiveInterval mirrors the clamping the watch package applies, so the
// message does not promise an interval that will not be used.
func effectiveInterval(d time.Duration) time.Duration {
	if d < watch.MinInterval {
		return watch.DefaultInterval
	}
	return d
}
