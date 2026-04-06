// Package cache provides cache management commands.
package cache

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the cache command.
var Cmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the in-memory resource cache",
	Long: `Commands for managing the in-memory cache.

The Go implementation of dtiam does not use a persistent cache. Each CLI
invocation starts with a fresh in-memory cache that is discarded on exit.
These commands are provided for compatibility and future use.`,
	Example: `  # Clear the cache
  dtiam cache clear

  # Show cache statistics
  dtiam cache stats`,
}

func init() {
	Cmd.AddCommand(clearCmd)
	Cmd.AddCommand(statsCmd)
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the in-memory cache",
	Long: `Clear all cached API responses.

In the current Go implementation this is a no-op because the cache is
not persistent across CLI invocations.`,
	Example: `  # Clear the cache
  dtiam cache clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// In-memory cache is cleared on each CLI invocation
		// This is a no-op in the Go implementation
		fmt.Println("Cache cleared")
		return nil
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	Long: `Display cache hit/miss statistics and memory usage.

In the current Go implementation the cache is not persistent, so stats
reflect only the current invocation (effectively empty).`,
	Example: `  # Show cache statistics
  dtiam cache stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// In-memory cache stats
		// The Go implementation doesn't maintain persistent cache stats
		fmt.Println("Cache is not persistent in this implementation.")
		fmt.Println("Each CLI invocation starts with a fresh cache.")
		return nil
	},
}
