// Package edit provides "dtiam edit": change a group, policy or boundary in
// $EDITOR, review the diff, and apply it.
package edit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	diffpkg "github.com/timstewart-dynatrace/dtiam/v3/pkg/diff"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/prompt"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

// Cmd is the edit command.
var Cmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a group, policy or boundary in your editor",
	Long: `Open a resource in your editor as YAML, then review and apply the change.

The editor is $VISUAL, then $EDITOR, then vi. Only the fields under spec can be
changed; read-only fields are shown as comments. After you save and close the
editor, dtiam shows a field-by-field diff and asks before applying (--force
skips the question). Closing without changes does nothing.

If anything goes wrong after editing -- invalid YAML, a rejected change, or a
"no" at the prompt -- your edits are kept in a file and the error tells you how
to resume with --from-file.

edit needs an interactive terminal, so it refuses to run with --plain or
--agent; use "dtiam apply -f" for automation.`,
	Example: `  # Edit a policy's statement
  dtiam edit policy "Read Only"

  # Rename a group
  dtiam edit group "Platform Team"

  # Change a boundary's query, previewing only
  dtiam edit boundary "Production" --dry-run

  # Resume edits that failed to apply
  dtiam edit policy "Read Only" --from-file /tmp/dtiam-edit-123.yaml`,
}

func init() {
	for _, c := range []*cobra.Command{groupCmd, policyCmd, boundaryCmd} {
		c.Flags().BoolP("force", "f", false, "Apply without asking after showing the diff")
		c.Flags().String("from-file", "", "Start from previously saved edits instead of the live resource")
		Cmd.AddCommand(c)
	}
}

var groupCmd = newKindCmd("group", "Edit a group's name and description",
	`  # Edit by name
  dtiam edit group "Platform Team"

  # Edit by UUID
  dtiam edit group 8f6e5d4c-3b2a-1098-7654-321fedcba098`, groupResource)

var policyCmd = newKindCmd("policy", "Edit a policy's name, description, statement and tags",
	`  # Edit a policy statement
  dtiam edit policy "Read Only"

  # Preview without applying
  dtiam edit policy "Read Only" --dry-run`, policyResource)

var boundaryCmd = newKindCmd("boundary", "Edit a boundary's name and query",
	`  # Edit a boundary query
  dtiam edit boundary "Production"`, boundaryResource)

// newKindCmd builds the subcommand for one resource kind.
func newKindCmd(kind, short, example string, build func(*client.Client) *resource) *cobra.Command {
	return &cobra.Command{
		Use:     kind + " IDENTIFIER",
		Short:   short,
		Long:    short + ". The IDENTIFIER is a UUID or name. See \"dtiam edit --help\".",
		Example: example,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cli.GlobalState.IsPlain() || cli.GlobalState.Agent || !stdinIsTerminal() {
				return fmt.Errorf("edit needs an interactive terminal; for automation use 'dtiam apply -f FILE'")
			}
			force, _ := cmd.Flags().GetBool("force")
			fromFile, _ := cmd.Flags().GetString("from-file")

			c, err := common.CreateClient()
			if err != nil {
				return err
			}
			defer c.Close()

			printer := cli.GlobalState.NewPrinter()
			d, err := run(build(c), options{
				identifier: args[0],
				fromFile:   fromFile,
				dryRun:     cli.GlobalState.IsDryRun(),
				editor:     launchEditor,
				confirm: func(d diffpkg.ResourceDiff) bool {
					printDiff(d)
					return prompt.Confirm("Apply these changes?", force)
				},
				report: printer.PrintWarning,
			})
			switch {
			case errors.Is(err, errNoChanges):
				printer.PrintMessage("No changes; %s %q left as it was.", kind, args[0])
				return nil
			case err != nil:
				return err
			case cli.GlobalState.IsDryRun():
				printDiff(d)
				return nil
			}
			printer.PrintSuccess("%s %q updated", strings.ToUpper(kind[:1])+kind[1:], d.Name)
			return nil
		},
	}
}

// groupResource edits name and description via PUT /groups/{uuid}.
func groupResource(c *client.Client) *resource {
	h := resources.NewGroupHandler(c)
	return &resource{
		kind:     "Group",
		fields:   []string{"name", "description"},
		readOnly: []string{"uuid", "owner", "createdAt"},
		load: func(id string) (map[string]any, string, error) {
			return loadVia(id, "group", func(ctx context.Context) (map[string]any, error) {
				return resources.GetOrResolve(ctx, h, id)
			})
		},
		save: func(uuid string, _ map[string]any, spec map[string]any) error {
			_, err := h.Update(context.Background(), uuid, spec)
			return err
		},
	}
}

// policyResource edits name, description, statementQuery and tags. The policy
// list omits statementQuery, so the full record is always fetched by UUID.
func policyResource(c *client.Client) *resource {
	h := resources.NewPolicyHandler(c)
	return &resource{
		kind:     "Policy",
		fields:   []string{"name", "description", "statementQuery", "tags"},
		readOnly: []string{"uuid", "category", "levelType", "levelId"},
		defaults: map[string]any{"tags": []any{}},
		load: func(id string) (map[string]any, string, error) {
			return loadVia(id, "policy", func(ctx context.Context) (map[string]any, error) {
				match, err := resources.GetOrResolve(ctx, h, id)
				if err != nil || match == nil {
					return match, err
				}
				return h.Get(ctx, utils.StringFrom(match, "uuid"))
			})
		},
		save: func(uuid string, _ map[string]any, spec map[string]any) error {
			_, err := h.Update(context.Background(), uuid, spec)
			return err
		},
	}
}

// boundaryResource edits name and boundaryQuery.
func boundaryResource(c *client.Client) *resource {
	h := resources.NewBoundaryHandler(c)
	return &resource{
		kind:     "Boundary",
		fields:   []string{"name", "boundaryQuery"},
		readOnly: []string{"uuid", "levelType", "levelId"},
		load: func(id string) (map[string]any, string, error) {
			return loadVia(id, "boundary", func(ctx context.Context) (map[string]any, error) {
				return resources.GetOrResolve(ctx, h, id)
			})
		},
		save: func(uuid string, _ map[string]any, spec map[string]any) error {
			name := utils.StringFrom(spec, "name")
			query := utils.StringFrom(spec, "boundaryQuery")
			_, err := h.Update(context.Background(), uuid, &name, nil, &query, nil)
			return err
		},
	}
}

// loadVia runs a lookup and turns "not found" into a clear error.
func loadVia(id, kind string, get func(context.Context) (map[string]any, error)) (map[string]any, string, error) {
	live, err := get(context.Background())
	if err != nil {
		return nil, "", err
	}
	if live == nil {
		return nil, "", fmt.Errorf("%s %q not found", kind, id)
	}
	uuid := utils.StringFrom(live, "uuid")
	if uuid == "" {
		return nil, "", fmt.Errorf("%s %q has no UUID", kind, id)
	}
	return live, uuid, nil
}

// editorCommand returns the configured editor: $VISUAL, then $EDITOR, then vi.
func editorCommand() string {
	for _, v := range []string{"VISUAL", "EDITOR"} {
		if e := strings.TrimSpace(os.Getenv(v)); e != "" {
			return e
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// launchEditor opens path in the editor and waits for it to exit. The editor
// value may carry arguments ("code --wait"), so it runs through the shell.
func launchEditor(path string) error {
	editor := editorCommand()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		parts := strings.Fields(editor)
		cmd = exec.Command(parts[0], append(parts[1:], path)...)
	} else {
		cmd = exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", editor, err)
	}
	return nil
}

// printDiff shows the field changes on stderr, like "dtiam diff".
func printDiff(d diffpkg.ResourceDiff) {
	fmt.Fprintf(os.Stderr, "\n%s %q:\n", d.Kind, d.Name)
	for _, f := range d.Fields {
		fmt.Fprintf(os.Stderr, "  %s\n    - have: %s\n    + want: %s\n", f.Field, f.Have, f.Want)
	}
	fmt.Fprintln(os.Stderr)
}

// stdinIsTerminal reports whether stdin is interactive.
func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
