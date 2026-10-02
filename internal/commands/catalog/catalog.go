// Package catalog provides the "commands" command: a machine-readable listing
// of every dtiam command, built from the live command tree.
package catalog

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/safety"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/version"
)

// SchemaVersion is bumped when the catalog's shape changes incompatibly.
const SchemaVersion = 1

// Catalog is the full listing.
type Catalog struct {
	SchemaVersion int               `json:"schema_version" yaml:"schema_version"`
	Tool          string            `json:"tool" yaml:"tool"`
	Version       string            `json:"version" yaml:"version"`
	CommandModel  string            `json:"command_model" yaml:"command_model"`
	SafetyLevels  map[string]string `json:"safety_levels" yaml:"safety_levels"`
	GlobalFlags   map[string]Flag   `json:"global_flags" yaml:"global_flags"`
	Commands      []Command         `json:"commands" yaml:"commands"`
}

// Command describes one runnable command.
type Command struct {
	Path      string          `json:"path" yaml:"path"`
	Usage     string          `json:"usage,omitempty" yaml:"usage,omitempty"`
	Short     string          `json:"short" yaml:"short"`
	Aliases   []string        `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Operation string          `json:"operation" yaml:"operation"`
	Mutating  bool            `json:"mutating" yaml:"mutating"`
	Flags     map[string]Flag `json:"flags,omitempty" yaml:"flags,omitempty"`
	// Escalations name flags that raise the operation, e.g. --replace -> delete.
	Escalations map[string]string `json:"escalations,omitempty" yaml:"escalations,omitempty"`
	Example     string            `json:"example,omitempty" yaml:"example,omitempty"`
}

// Flag describes one flag.
type Flag struct {
	Type        string `json:"type" yaml:"type"`
	Shorthand   string `json:"shorthand,omitempty" yaml:"shorthand,omitempty"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Description string `json:"description" yaml:"description"`
}

// Cmd is the commands command.
var Cmd = &cobra.Command{
	Use:   "commands",
	Short: "List every command, machine-readably",
	Long: `List every dtiam command with its usage, flags, and what it does to the
account (read, create, update or delete) -- the same classification the context
safety levels enforce.

Intended for AI agents and tooling: "dtiam commands -o json" describes the whole
CLI in one call, so an agent need not walk --help. --brief lists only each
command's path, operation and summary. The table view is always brief.`,
	Example: `  # Full catalog for an agent
  dtiam commands -o json

  # Just the commands and what they do
  dtiam commands --brief -o json

  # Human-readable overview
  dtiam commands`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		brief, _ := cmd.Flags().GetBool("brief")
		cat := Build(cmd.Root())
		printer := cli.GlobalState.NewPrinter()

		format := cli.GlobalState.GetOutput()
		tabular := format == output.FormatTable || format == output.FormatWide || format == output.FormatCSV
		if (brief || tabular) && !cli.GlobalState.Agent {
			rows := make([]map[string]any, 0, len(cat.Commands))
			for _, c := range cat.Commands {
				rows = append(rows, map[string]any{"path": c.Path, "operation": c.Operation, "short": c.Short})
			}
			return printer.Print(rows, []output.Column{
				{Key: "path", Header: "COMMAND"},
				{Key: "operation", Header: "OPERATION"},
				{Key: "short", Header: "DESCRIPTION"},
			})
		}
		if brief {
			for i := range cat.Commands {
				cat.Commands[i].Flags, cat.Commands[i].Example, cat.Commands[i].Usage = nil, "", ""
			}
		}
		return printer.PrintAny(cat)
	},
}

func init() {
	Cmd.Flags().Bool("brief", false, "Only path, operation and summary for each command")
}

// Build walks the command tree from root.
func Build(root *cobra.Command) Catalog {
	cat := Catalog{
		SchemaVersion: SchemaVersion,
		Tool:          "dtiam",
		Version:       version.Version,
		CommandModel:  "dtiam [global-flags] <verb> [<resource>] [<identifier>] [flags]",
		SafetyLevels: map[string]string{
			"readonly":  "only read operations",
			"no-delete": "read, create and update; no delete",
			"readwrite": "everything (default)",
		},
		GlobalFlags: flags(root.PersistentFlags()),
	}

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			if !sub.HasSubCommands() {
				cat.Commands = append(cat.Commands, describe(sub))
			}
			walk(sub)
		}
	}
	walk(root)
	sort.Slice(cat.Commands, func(i, j int) bool { return cat.Commands[i].Path < cat.Commands[j].Path })
	return cat
}

func describe(c *cobra.Command) Command {
	op, ok := cli.DeclaredOperation(c)
	if !ok {
		op = safety.Delete // matches the safety check's fail-closed default
	}
	out := Command{
		Path:      cli.RelativePath(c),
		Usage:     c.UseLine(),
		Short:     c.Short,
		Aliases:   c.Aliases,
		Operation: string(op),
		Mutating:  op != safety.Read,
		Flags:     flags(c.LocalNonPersistentFlags()),
		Example:   strings.TrimSpace(c.Example),
	}
	for key, v := range c.Annotations {
		if flag, ok := strings.CutPrefix(key, "dtiam.io/escalate/"); ok {
			if out.Escalations == nil {
				out.Escalations = map[string]string{}
			}
			out.Escalations[flag] = v
		}
	}
	return out
}

func flags(fs *pflag.FlagSet) map[string]Flag {
	out := map[string]Flag{}
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		_, required := f.Annotations[cobra.BashCompOneRequiredFlag]
		out[f.Name] = Flag{
			Type:        f.Value.Type(),
			Shorthand:   f.Shorthand,
			Default:     f.DefValue,
			Required:    required,
			Description: f.Usage,
		}
	})
	if len(out) == 0 {
		return nil
	}
	return out
}
