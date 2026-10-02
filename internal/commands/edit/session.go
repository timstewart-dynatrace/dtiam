package edit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	diffpkg "github.com/timstewart-dynatrace/dtiam/v3/pkg/diff"
)

// errNoChanges reports that the edited document matches the live resource.
var errNoChanges = errors.New("no changes")

// resource abstracts one editable resource kind: how to load it, which fields
// may be edited, and how to save an edit. Tests substitute fakes.
type resource struct {
	kind string
	// fields are the editable spec fields, in display order.
	fields []string
	// readOnly are live fields shown as comments for context.
	readOnly []string
	// defaults are the empty value of fields the API may omit, such as an
	// empty description or tag list.
	defaults map[string]any
	// load returns the live resource and its UUID.
	load func(identifier string) (live map[string]any, uuid string, err error)
	// save applies spec (editable fields only) to the resource.
	save func(uuid string, live, spec map[string]any) error
}

// document is the YAML written for the user to edit.
type document struct {
	Kind     string         `yaml:"kind"`
	Metadata map[string]any `yaml:"metadata"`
	Spec     map[string]any `yaml:"spec"`
}

// render produces the editable YAML for a live resource.
func (r *resource) render(live map[string]any, uuid string) ([]byte, error) {
	spec := map[string]any{}
	for _, f := range r.fields {
		spec[f] = r.valueOf(live, f)
	}

	var buf bytes.Buffer
	name, _ := live["name"].(string)
	fmt.Fprintf(&buf, "# Editing %s %q\n", r.kind, name)
	buf.WriteString("# Change fields under spec, then save and close the editor.\n")
	buf.WriteString("# Close without changes to cancel. Lines starting with # are ignored.\n")
	for _, f := range r.readOnly {
		if v, ok := live[f]; ok && v != nil {
			fmt.Fprintf(&buf, "#   %s: %v (read-only)\n", f, v)
		}
	}

	out, err := yaml.Marshal(document{
		Kind:     r.kind,
		Metadata: map[string]any{"uuid": uuid},
		Spec:     spec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render %s: %w", r.kind, err)
	}
	buf.Write(out)
	return buf.Bytes(), nil
}

// valueOf returns a live field, or its default when the API omitted it.
func (r *resource) valueOf(live map[string]any, field string) any {
	if v, ok := live[field]; ok && v != nil {
		return v
	}
	if d, ok := r.defaults[field]; ok {
		return d
	}
	return ""
}

// comparable returns the live resource with omitted editable fields filled
// in, so an untouched empty field does not show up as a change.
func (r *resource) comparable(live map[string]any) map[string]any {
	out := make(map[string]any, len(live)+len(r.fields))
	for k, v := range live {
		out[k] = v
	}
	for _, f := range r.fields {
		out[f] = r.valueOf(live, f)
	}
	return out
}

// parse reads an edited document and returns its spec, restricted to the
// editable fields. The kind and UUID must not change: editing a different
// object than the one loaded would apply the change to the wrong resource.
func (r *resource) parse(data []byte, uuid string) (map[string]any, error) {
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("the edited file is not valid YAML: %w", err)
	}
	if !strings.EqualFold(doc.Kind, r.kind) {
		return nil, fmt.Errorf("kind changed from %s to %q; edit cannot change a resource's kind", r.kind, doc.Kind)
	}
	if got, _ := doc.Metadata["uuid"].(string); got != uuid {
		return nil, fmt.Errorf("metadata.uuid changed from %s to %q; it identifies the resource and cannot be edited", uuid, got)
	}

	allowed := map[string]bool{}
	for _, f := range r.fields {
		allowed[f] = true
	}
	spec := map[string]any{}
	for k, v := range doc.Spec {
		if !allowed[k] {
			return nil, fmt.Errorf("spec.%s cannot be edited (editable: %s)", k, strings.Join(r.fields, ", "))
		}
		spec[k] = v
	}
	if name, _ := spec["name"].(string); strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("spec.name cannot be empty")
	}
	return spec, nil
}

// options control one edit run.
type options struct {
	identifier string
	// fromFile starts from previously saved edits instead of the live resource.
	fromFile string
	dryRun   bool
	// editor edits the file at path in place.
	editor func(path string) error
	// confirm asks whether to apply the diff.
	confirm func(diffpkg.ResourceDiff) bool
	// report prints a line for the user.
	report func(format string, args ...any)
}

// run performs load -> edit -> diff -> confirm -> save. On any failure after
// the editor closed, the edited file is kept and its path is in the error so
// the work is not lost.
func run(r *resource, o options) (diffpkg.ResourceDiff, error) {
	live, uuid, err := r.load(o.identifier)
	if err != nil {
		return diffpkg.ResourceDiff{}, err
	}

	var content []byte
	if o.fromFile != "" {
		if content, err = os.ReadFile(o.fromFile); err != nil {
			return diffpkg.ResourceDiff{}, fmt.Errorf("failed to read %s: %w", o.fromFile, err)
		}
	} else if content, err = r.render(live, uuid); err != nil {
		return diffpkg.ResourceDiff{}, err
	}

	f, err := os.CreateTemp("", "dtiam-edit-*.yaml")
	if err != nil {
		return diffpkg.ResourceDiff{}, fmt.Errorf("failed to create a temp file: %w", err)
	}
	path := f.Name()
	_, werr := f.Write(content)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(path)
		return diffpkg.ResourceDiff{}, fmt.Errorf("failed to write %s: %v", path, errors.Join(werr, cerr))
	}

	keep := func(err error) error {
		return fmt.Errorf("%w\nYour edits are saved in %s; resume with: dtiam edit %s %s --from-file %s",
			err, path, strings.ToLower(r.kind), quote(o.identifier), path)
	}

	if err := o.editor(path); err != nil {
		return diffpkg.ResourceDiff{}, keep(fmt.Errorf("the editor failed: %w", err))
	}
	edited, err := os.ReadFile(path)
	if err != nil {
		return diffpkg.ResourceDiff{}, keep(err)
	}
	spec, err := r.parse(edited, uuid)
	if err != nil {
		return diffpkg.ResourceDiff{}, keep(err)
	}

	name, _ := live["name"].(string)
	d := diffpkg.Compare(r.kind, name, spec, r.comparable(live))
	if !d.HasChanges() {
		_ = os.Remove(path)
		return d, errNoChanges
	}

	if o.dryRun {
		o.report("Dry run: not applied. Your edits are in %s", path)
		return d, nil
	}
	if !o.confirm(d) {
		return d, keep(errors.New("not applied"))
	}
	if err := r.save(uuid, live, spec); err != nil {
		return d, keep(err)
	}
	_ = os.Remove(path)
	return d, nil
}

// quote wraps an identifier in quotes when it contains spaces.
func quote(s string) string {
	if strings.ContainsAny(s, " \t") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
