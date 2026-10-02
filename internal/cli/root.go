package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/version"
)

var (
	// Flags
	contextFlag string
	outputFlag  string
	verboseFlag bool
	plainFlag   bool
	dryRunFlag  bool
	agentFlag   bool
)

// RootCmd is the root command for dtiam.
var RootCmd = &cobra.Command{
	Use:   "dtiam",
	Short: "A kubectl-inspired CLI for Dynatrace IAM",
	Long: `dtiam is a command-line tool for managing Dynatrace Identity and Access Management (IAM) resources.

It provides a consistent interface for managing groups, users, policies, bindings,
boundaries, environments, and service users.

DISCLAIMER: This tool is provided "as-is" without warranty. Use at your own risk.
This is an independent, community-developed tool and is NOT produced, endorsed,
or supported by Dynatrace.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Update global state from flags
		GlobalState.Context = contextFlag
		GlobalState.Verbose = verboseFlag
		GlobalState.Plain = plainFlag
		GlobalState.DryRun = dryRunFlag

		// Agent mode first: it implies --plain and must capture stdout before
		// the command can write anything.
		if agentFlag || os.Getenv(EnvAgent) != "" {
			startAgentMode(cmd)
		}

		// Imply --plain when running under a coding agent. An agent has no
		// terminal to answer a confirmation prompt at and no use for ANSI
		// colors, so the interactive defaults are actively wrong there. An
		// explicit --plain=false still wins, and DTIAM_NO_AGENT_DETECT opts out.
		if !cmd.Flags().Changed("plain") {
			if agent := DetectAgent(); agent.Detected {
				GlobalState.Plain = true
				GlobalState.AgentName = agent.Name
				if verboseFlag {
					fmt.Fprintf(os.Stderr,
						"Detected %s via %s; enabling --plain. Set %s=1 to disable.\n",
						agent.Name, agent.Source, EnvDisableAgentDetection)
				}
			}
		}

		if outputFlag != "" {
			format, err := output.ParseFormat(outputFlag)
			if err != nil {
				return err
			}
			GlobalState.Output = format
		}

		// Enforce the context's safety level before the command runs, so a
		// blocked command makes no API call at all.
		return checkSafety(cmd)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Global flags
	RootCmd.PersistentFlags().StringVar(&contextFlag, "context", "", "Override current context")
	RootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output format: table, wide, json, yaml, csv, plain")
	RootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	RootCmd.PersistentFlags().BoolVar(&plainFlag, "plain", false, "Disable colors and interactive features")
	RootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Preview changes without applying them")
	RootCmd.PersistentFlags().BoolVarP(&agentFlag, "agent", "A", false,
		"Agent mode: write one JSON envelope {ok, result, error, context} to stdout")

	// Bind cobra flags to Viper for automatic env var support
	_ = config.V.BindPFlag("context", RootCmd.PersistentFlags().Lookup("context"))
	_ = config.V.BindPFlag("output", RootCmd.PersistentFlags().Lookup("output"))
	_ = config.V.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))
	_ = config.V.BindPFlag("plain", RootCmd.PersistentFlags().Lookup("plain"))
	_ = config.V.BindPFlag("dry_run", RootCmd.PersistentFlags().Lookup("dry-run"))

	// Add version command
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dtiam version %s\n", version.Version)
		if version.Commit != "unknown" {
			fmt.Printf("  commit: %s\n", version.Commit)
		}
		if version.Date != "unknown" {
			fmt.Printf("  built:  %s\n", version.Date)
		}
	},
}

// ErrSilentExit makes a command exit non-zero without printing an error line.
//
// It exists for commands whose non-zero exit is a result rather than a failure --
// "dtiam diff" signalling drift, for example. Those commands have already printed
// the information the user needs, so an "Error:" line on top of it would be
// misleading.
var ErrSilentExit = errors.New("silent exit")

// Execute runs the root command.
func Execute() {
	err := RootCmd.Execute()

	// In agent mode every outcome, including errors, is reported in the
	// envelope on stdout rather than as an "Error:" line on stderr.
	if GlobalState.Agent {
		os.Exit(finishAgentMode(err))
	}
	// A usage error (unknown flag, bad arguments) stops cobra before
	// PersistentPreRunE, so agent mode never started. Report it in an envelope
	// anyway when the agent asked for one.
	if err != nil && agentRequested() {
		startAgentMode(nil)
		os.Exit(finishAgentMode(err))
	}

	if err != nil {
		if errors.Is(err, ErrSilentExit) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// agentRequested reports whether agent mode was asked for. It also scans the
// raw arguments, because a usage error can stop cobra before flags are parsed.
func agentRequested() bool {
	if agentFlag || os.Getenv(EnvAgent) != "" {
		return true
	}
	for _, a := range os.Args[1:] {
		if a == "--" {
			break
		}
		if a == "-A" || a == "--agent" || a == "--agent=true" {
			return true
		}
	}
	return false
}

// AddCommand adds a command to the root command.
func AddCommand(cmd *cobra.Command) {
	RootCmd.AddCommand(cmd)
}
