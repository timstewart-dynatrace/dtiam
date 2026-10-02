package main

import (
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/catalog"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/safety"
)

var registerOnce sync.Once

func tree(t *testing.T) *cobra.Command {
	t.Helper()
	registerOnce.Do(registerCommands)
	return cli.RootCmd
}

func leaves(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if !c.HasSubCommands() {
			out = append(out, c)
		}
		for _, s := range c.Commands() {
			walk(s)
		}
	}
	walk(root)
	return out
}

// TestEveryCommandDeclaresAnOperation keeps the safety check complete: a new
// command must appear in the operations table (directly or through its verb).
func TestEveryCommandDeclaresAnOperation(t *testing.T) {
	for _, c := range leaves(tree(t)) {
		if c == cli.RootCmd || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if _, ok := cli.DeclaredOperation(c); !ok {
			t.Errorf("%q declares no operation; add it to the operations table in cmd/dtiam/operations.go", cli.RelativePath(c))
		}
	}
}

// TestOperationsTableHasNoStaleEntries catches renamed or removed commands.
func TestOperationsTableHasNoStaleEntries(t *testing.T) {
	paths := map[string]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		paths[cli.RelativePath(c)] = true
		for _, s := range c.Commands() {
			walk(s)
		}
	}
	walk(tree(t))
	for path := range operations {
		if !paths[path] {
			t.Errorf("operations table names %q, which is not a command", path)
		}
	}
}

func TestSpotCheckClassifications(t *testing.T) {
	root := tree(t)
	tests := map[string]safety.Operation{
		"get groups":                safety.Read,
		"delete group":              safety.Delete,
		"create token":              safety.Create,
		"apply":                     safety.Update,
		"boundary detach":           safety.Delete,
		"boundary attach":           safety.Update,
		"group remove-member":       safety.Delete,
		"user replace-groups":       safety.Delete,
		"bulk create-groups":        safety.Create,
		"bulk export-group-members": safety.Read,
		"token deactivate":          safety.Update,
		"template apply":            safety.Create,
		"template save":             safety.Read,
		"config set-context":        safety.Read,
	}
	for path, want := range tests {
		c, _, err := root.Find(strings.Fields(path))
		if err != nil {
			t.Fatalf("find %q: %v", path, err)
		}
		if got, _ := cli.DeclaredOperation(c); got != want {
			t.Errorf("%q = %q, want %q", path, got, want)
		}
	}
}

func TestGrantPermissionReplaceEscalatesToDelete(t *testing.T) {
	c, _, err := tree(t).Find([]string{"group", "grant-permission"})
	if err != nil {
		t.Fatal(err)
	}
	if op, _ := cli.EffectiveOperation(c); op != safety.Create {
		t.Fatalf("without --replace: %q, want create", op)
	}
	if err := c.Flags().Set("replace", "true"); err != nil {
		t.Fatalf("set --replace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Flags().Set("replace", "false")
		c.Flags().Lookup("replace").Changed = false
	})
	if op, _ := cli.EffectiveOperation(c); op != safety.Delete {
		t.Errorf("with --replace: %q, want delete", op)
	}
}

func TestCatalogCoversTheTreeWithMatchingOperations(t *testing.T) {
	root := tree(t)
	cat := catalog.Build(root)
	if cat.SchemaVersion != catalog.SchemaVersion || cat.Tool != "dtiam" {
		t.Errorf("header = %+v", cat)
	}
	if _, ok := cat.GlobalFlags["agent"]; !ok {
		t.Error("global flags are missing --agent")
	}

	byPath := map[string]catalog.Command{}
	for _, c := range cat.Commands {
		byPath[c.Path] = c
	}
	for _, leaf := range leaves(root) {
		path := cli.RelativePath(leaf)
		if leaf == root || leaf.Name() == "help" || leaf.Name() == "completion" {
			continue
		}
		entry, ok := byPath[path]
		if !ok {
			t.Errorf("catalog is missing %q", path)
			continue
		}
		op, _ := cli.DeclaredOperation(leaf)
		if entry.Operation != string(op) || entry.Mutating != (op != safety.Read) {
			t.Errorf("%q: catalog says %s (mutating=%v), tree says %s", path, entry.Operation, entry.Mutating, op)
		}
	}
	if byPath["group grant-permission"].Escalations["replace"] != "delete" {
		t.Error("catalog does not report the --replace escalation")
	}
}

func TestUnknownSubcommandFails(t *testing.T) {
	root := tree(t)
	get, _, err := root.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	if get.Args == nil || get.Args(get, []string{"groupz"}) == nil {
		t.Error(`"dtiam get groupz" must be an error, not get's help with exit 0`)
	}
	if get.Args(get, nil) != nil {
		t.Error(`bare "dtiam get" should still be allowed (it shows help)`)
	}
}
