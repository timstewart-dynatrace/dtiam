// Package diff provides the diff command: preview what apply would change.
package diff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	diffpkg "github.com/timstewart-dynatrace/dtiam/v3/pkg/diff"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
	tmpl "github.com/timstewart-dynatrace/dtiam/v3/pkg/template"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

// Cmd is the diff command.
var Cmd = &cobra.Command{
	Use:   "diff",
	Short: "Show what applying a file would change",
	Long: `Compare the resources in a file against what exists in the account, and
report what "dtiam apply" would change.

This is a read-only command: it fetches the live resource for each entry in the
file and compares it field by field. Nothing is created or modified.

Only fields present in the file are compared. The API returns server-managed
fields a spec will never mention -- uuid, createdAt, owner -- and reporting those
as differences would bury the real changes.

Lists are compared without regard to order, because the API returns members,
scopes, and zones in an order the caller does not control.

Exits 1 when there are changes and 0 when everything is up to date, so it can
gate a pipeline on drift. Use --exit-zero to always exit 0.`,
	Example: `  # What would apply change?
  dtiam diff -f resources.yaml

  # With template variables, as apply would render them
  dtiam diff -f policy-template.yaml --set name=MyPolicy --set env=production

  # Machine-readable
  dtiam diff -f resources.yaml --plain

  # Drift detection in CI: non-zero exit means the account differs from the file
  dtiam diff -f desired-state.yaml --plain || echo "drift detected"

  # Report drift without failing the build
  dtiam diff -f resources.yaml --exit-zero`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		setFlags, _ := cmd.Flags().GetStringSlice("set")
		exitZero, _ := cmd.Flags().GetBool("exit-zero")

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Render template variables exactly as apply would, so the diff reflects
		// what apply would actually send.
		if len(setFlags) > 0 {
			vars, err := tmpl.ParseSetFlags(setFlags)
			if err != nil {
				return err
			}
			rendered, err := tmpl.RenderTemplate(string(content), vars)
			if err != nil {
				return err
			}
			content = []byte(rendered)
		}

		specs, err := parseDocuments(content)
		if err != nil {
			return err
		}
		if len(specs) == 0 {
			return fmt.Errorf("no resources found in %s", file)
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer func() { _ = c.Close() }()

		ctx := context.Background()
		printer := cli.GlobalState.NewPrinter()

		diffs := make([]diffpkg.ResourceDiff, 0, len(specs))
		for _, s := range specs {
			d, err := compareResource(ctx, c, s.kind, s.spec)
			if err != nil {
				return fmt.Errorf("%s %q: %w", s.kind, s.name(), err)
			}
			diffs = append(diffs, d)
		}

		if err := render(printer, diffs); err != nil {
			return err
		}

		summary := diffpkg.Summarize(diffs)
		fmt.Fprintf(os.Stderr, "\n%s\n", summary.String())

		// A non-zero exit on drift is what makes this usable as a pipeline gate.
		// It is a result, not a failure, so no error line is printed.
		if summary.HasChanges() && !exitZero {
			return cli.ErrSilentExit
		}
		return nil
	},
}

func init() {
	Cmd.Flags().StringP("file", "f", "", "Resource definition file (required)")
	Cmd.Flags().StringSlice("set", nil, "Set template variable as key=value (repeatable)")
	Cmd.Flags().Bool("exit-zero", false, "Exit 0 even when there are changes")
	_ = Cmd.MarkFlagRequired("file")
}

// docSpec is one parsed document from the input file.
type docSpec struct {
	kind string
	spec map[string]any
}

// name returns the spec's name, for error messages.
func (d docSpec) name() string {
	return utils.StringFrom(d.spec, "name")
}

// parseDocuments splits and parses the input into resource specs.
func parseDocuments(content []byte) ([]docSpec, error) {
	var out []docSpec

	for i, doc := range splitYAMLDocuments(content) {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		var resource map[string]any
		if err := yaml.Unmarshal(doc, &resource); err != nil {
			if err := json.Unmarshal(doc, &resource); err != nil {
				return nil, fmt.Errorf("document %d: failed to parse (expected YAML or JSON): %w", i+1, err)
			}
		}

		kind, _ := resource["kind"].(string)
		spec, _ := resource["spec"].(map[string]any)
		if kind == "" {
			return nil, fmt.Errorf("document %d: missing 'kind' field", i+1)
		}
		if spec == nil {
			return nil, fmt.Errorf("document %d: missing 'spec' field", i+1)
		}

		out = append(out, docSpec{kind: kind, spec: spec})
	}

	return out, nil
}

// splitYAMLDocuments splits a YAML byte slice on document separators.
//
// Blank documents are left in and skipped by the caller, so document numbers in
// error messages still match the positions a user sees in the file.
func splitYAMLDocuments(content []byte) [][]byte {
	return bytes.Split(content, []byte("\n---"))
}

// compareResource fetches the live resource and diffs the spec against it.
//
// A resource that cannot be found yields a create diff rather than an error,
// since "does not exist yet" is the normal state before a first apply.
func compareResource(ctx context.Context, c *client.Client, kind string, spec map[string]any) (diffpkg.ResourceDiff, error) {
	name := utils.StringFrom(spec, "name")

	switch strings.ToLower(kind) {
	case "group":
		if name == "" {
			return diffpkg.ResourceDiff{}, fmt.Errorf("group spec requires a 'name' field")
		}
		live, err := resources.NewGroupHandler(c).GetByName(ctx, name)
		if err != nil {
			return diffpkg.ResourceDiff{}, err
		}
		return diffpkg.Compare("Group", name, spec, live), nil

	case "policy":
		if name == "" {
			return diffpkg.ResourceDiff{}, fmt.Errorf("policy spec requires a 'name' field")
		}
		live, err := resources.GetPolicyByName(ctx, resources.NewPolicyHandler(c), name)
		if err != nil {
			return diffpkg.ResourceDiff{}, err
		}
		return diffpkg.Compare("Policy", name, spec, live), nil

	case "boundary":
		if name == "" {
			return diffpkg.ResourceDiff{}, fmt.Errorf("boundary spec requires a 'name' field")
		}
		live, err := resources.NewBoundaryHandler(c).GetByName(ctx, name)
		if err != nil {
			return diffpkg.ResourceDiff{}, err
		}
		return diffpkg.Compare("Boundary", name, spec, live), nil

	case "binding":
		// A binding has no name; it is identified by its group and policy pair,
		// so it is matched against the group's existing bindings rather than
		// looked up by name.
		groupID := utils.StringFrom(spec, "group")
		policyID := utils.StringFrom(spec, "policy")
		if groupID == "" || policyID == "" {
			return diffpkg.ResourceDiff{}, fmt.Errorf("binding spec requires 'group' and 'policy' fields")
		}
		return compareBinding(ctx, c, spec, groupID, policyID)

	default:
		return diffpkg.ResourceDiff{}, fmt.Errorf(
			"unsupported resource kind: %s (expected Group, Policy, Boundary, or Binding)", kind)
	}
}

// compareBinding checks whether a group already has a binding to a policy.
func compareBinding(ctx context.Context, c *client.Client, spec map[string]any, groupID, policyID string) (diffpkg.ResourceDiff, error) {
	label := fmt.Sprintf("group=%s policy=%s", groupID, policyID)

	policies, err := resources.NewGroupHandler(c).GetPolicies(ctx, groupID)
	if err != nil {
		// A group with no bindings yet is not an error condition worth failing
		// the whole diff over; treat it as "nothing bound".
		return diffpkg.Compare("Binding", label, spec, nil), nil
	}

	for _, p := range policies {
		if strings.EqualFold(p, policyID) {
			return diffpkg.ResourceDiff{
				Kind:    "Binding",
				Name:    label,
				Change:  diffpkg.ChangeNone,
				Message: "already bound",
			}, nil
		}
	}

	return diffpkg.Compare("Binding", label, spec, nil), nil
}

// render prints the diff table and the per-field detail.
func render(printer *output.Printer, diffs []diffpkg.ResourceDiff) error {
	rows := make([]map[string]any, 0, len(diffs))
	for _, d := range diffs {
		rows = append(rows, map[string]any{
			"kind":    d.Kind,
			"name":    d.Name,
			"change":  string(d.Change),
			"details": d.Message,
			"fields":  fieldMaps(d.Fields),
		})
	}

	if err := printer.Print(rows, Columns()); err != nil {
		return err
	}

	// In --plain the field detail is already in the JSON above, so printing it
	// again as prose would be noise. In a terminal it is the useful part.
	if cli.GlobalState.IsPlain() {
		return nil
	}

	for _, d := range diffs {
		if len(d.Fields) == 0 {
			continue
		}
		fmt.Fprintf(os.Stderr, "\n%s %q:\n", d.Kind, d.Name)
		for _, f := range d.Fields {
			fmt.Fprintf(os.Stderr, "  %s\n    - have: %s\n    + want: %s\n", f.Field, f.Have, f.Want)
		}
	}
	return nil
}

// fieldMaps converts field changes for structured output.
func fieldMaps(fields []diffpkg.FieldChange) []map[string]any {
	out := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		out = append(out, map[string]any{"field": f.Field, "want": f.Want, "have": f.Have})
	}
	return out
}

// Columns returns the columns for diff output.
func Columns() []output.Column {
	return []output.Column{
		{Key: "kind", Header: "KIND"},
		{Key: "name", Header: "NAME"},
		{Key: "change", Header: "CHANGE"},
		{Key: "details", Header: "DETAILS"},
	}
}
