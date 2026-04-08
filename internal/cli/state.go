// Package cli provides the root CLI command and global state.
package cli

import (
	"os"

	"github.com/jtimothystewart/dtiam/internal/output"
)

// State holds the global CLI state.
type State struct {
	// Context is the context override from command line.
	Context string
	// Output is the output format.
	Output output.Format
	// Verbose enables verbose output.
	Verbose bool
	// Plain disables colors and interactive features.
	Plain bool
	// DryRun enables dry-run mode (preview without changes).
	DryRun bool
}

// GlobalState is the global CLI state instance.
var GlobalState = &State{
	Output: output.FormatTable,
}

// GetContext returns the context override, or empty string if not set.
func (s *State) GetContext() string {
	return s.Context
}

// GetOutput returns the output format.
func (s *State) GetOutput() output.Format {
	return s.Output
}

// GetOutputFormat returns the output format (alias for GetOutput).
func (s *State) GetOutputFormat() output.Format {
	return s.Output
}

// IsVerbose returns true if verbose mode is enabled.
func (s *State) IsVerbose() bool {
	return s.Verbose
}

// IsPlain returns true if plain mode is enabled.
func (s *State) IsPlain() bool {
	return s.Plain
}

// IsDryRun returns true if dry-run mode is enabled.
func (s *State) IsDryRun() bool {
	return s.DryRun
}

// EnvironmentURL returns the environment URL from the DTIAM_ENVIRONMENT_URL env var.
func (s *State) EnvironmentURL() string {
	return os.Getenv("DTIAM_ENVIRONMENT_URL")
}

// NewPrinter creates a new printer with the current state settings.
// When --plain is set, table and wide formats are automatically converted
// to JSON for machine/AI consumption.
func (s *State) NewPrinter() *output.Printer {
	format := s.Output
	if s.Plain && (format == output.FormatTable || format == output.FormatWide) {
		format = output.FormatJSON
	}
	return output.NewPrinter(format, s.Plain)
}
