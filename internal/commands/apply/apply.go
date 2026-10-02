// Package apply provides the declarative apply command.
package apply

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	diffpkg "github.com/jtimothystewart/dtiam/pkg/diff"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	tmpl "github.com/jtimothystewart/dtiam/pkg/template"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

// Cmd is the apply command.
var Cmd = &cobra.Command{
	Use:   "apply",
	Short: "Create or update resources from a file",
	Long: `Apply a resource definition from a YAML or JSON file.

The file must contain a 'kind' field (Group, Policy, Boundary, Binding) and a
'spec' field with the resource properties.

Resources are matched by name (bindings by their group and policy). A missing
resource is created; an existing one is updated to match the spec, and one that
already matches is left unchanged, so applying the same file twice is safe.

Multiple resources can be defined in a single file using YAML document
separators (---).

Template variables can be substituted using --set key=value flags.`,
	Example: `  # Create a group from a YAML file
  dtiam apply -f group.yaml

  # Create a policy with template variables
  dtiam apply -f policy-template.yaml --set name=MyPolicy --set env=production

  # Preview what would be created
  dtiam apply -f resources.yaml --dry-run

  # Apply multiple resources from one file (YAML --- separators)
  dtiam apply -f all-resources.yaml

  # Machine-friendly output
  dtiam apply -f group.yaml --plain`,
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		setFlags, _ := cmd.Flags().GetStringSlice("set")

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Apply template variables if any --set flags provided
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

		// Split YAML documents
		docs := splitYAMLDocuments(content)

		printer := cli.GlobalState.NewPrinter()

		for i, doc := range docs {
			if len(bytes.TrimSpace(doc)) == 0 {
				continue
			}

			var resource map[string]any

			// Try YAML first, then JSON
			if err := yaml.Unmarshal(doc, &resource); err != nil {
				if err := json.Unmarshal(doc, &resource); err != nil {
					return fmt.Errorf("document %d: failed to parse (expected YAML or JSON): %w", i+1, err)
				}
			}

			kind, _ := resource["kind"].(string)
			spec, _ := resource["spec"].(map[string]any)
			if kind == "" {
				return fmt.Errorf("document %d: missing 'kind' field", i+1)
			}
			if spec == nil {
				return fmt.Errorf("document %d: missing 'spec' field", i+1)
			}

			if cli.GlobalState.IsDryRun() {
				printer.PrintWarning("Would create or update %s (run 'dtiam diff' to see which):", kind)
				rendered, _ := yaml.Marshal(spec)
				fmt.Fprintf(os.Stderr, "%s\n", rendered)
				continue
			}

			if err := applyResource(kind, spec, printer); err != nil {
				return fmt.Errorf("document %d (%s): %w", i+1, kind, err)
			}
		}

		return nil
	},
}

func init() {
	Cmd.Flags().StringP("file", "f", "", "Resource definition file (required)")
	Cmd.Flags().StringSlice("set", nil, "Set template variable as key=value (repeatable)")
	_ = Cmd.MarkFlagRequired("file")
}

// splitYAMLDocuments splits a YAML byte slice into individual documents.
func splitYAMLDocuments(content []byte) [][]byte {
	var docs [][]byte
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var current bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if current.Len() > 0 {
				docs = append(docs, bytes.Clone(current.Bytes()))
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}

	if current.Len() > 0 {
		docs = append(docs, current.Bytes())
	}

	return docs
}

// applyResource creates a resource, or updates it when one with the same name
// already exists. A resource whose live state already matches the spec is
// left alone, so applying the same file twice is a no-op.
func applyResource(kind string, spec map[string]any, printer interface {
	PrintSuccess(string, ...any)
	PrintDetail(map[string]any) error
}) error {
	c, err := common.CreateClient()
	if err != nil {
		return err
	}
	defer c.Close()

	ctx := context.Background()
	name := utils.StringFrom(spec, "name")

	switch strings.ToLower(kind) {
	case "group":
		if name == "" {
			return fmt.Errorf("group spec requires a 'name' field")
		}
		handler := resources.NewGroupHandler(c)
		live, err := handler.GetByName(ctx, name)
		if err != nil {
			return err
		}
		if live == nil {
			if _, err := handler.Create(ctx, spec); err != nil {
				return err
			}
			printer.PrintSuccess("Group %q created", name)
			return nil
		}
		if !diffpkg.Compare("Group", name, spec, live).HasChanges() {
			printer.PrintSuccess("Group %q unchanged", name)
			return nil
		}
		if _, err := handler.Update(ctx, utils.StringFrom(live, "uuid"), spec); err != nil {
			return err
		}
		printer.PrintSuccess("Group %q updated", name)
		return nil

	case "policy":
		if name == "" {
			return fmt.Errorf("policy spec requires a 'name' field")
		}
		handler := resources.NewPolicyHandler(c)
		live, err := resources.GetPolicyByName(ctx, handler, name)
		if err != nil {
			return err
		}
		if live == nil {
			if _, err := handler.Create(ctx, spec); err != nil {
				return err
			}
			printer.PrintSuccess("Policy %q created", name)
			return nil
		}
		if !diffpkg.Compare("Policy", name, spec, live).HasChanges() {
			printer.PrintSuccess("Policy %q unchanged", name)
			return nil
		}
		if _, err := handler.Update(ctx, utils.StringFrom(live, "uuid"), mergePolicy(live, spec)); err != nil {
			return err
		}
		printer.PrintSuccess("Policy %q updated", name)
		return nil

	case "boundary":
		if name == "" {
			return fmt.Errorf("boundary spec requires a 'name' field")
		}
		handler := resources.NewBoundaryHandler(c)
		var zones []string
		if z, ok := spec["zones"].([]any); ok {
			for _, zone := range z {
				if s, ok := zone.(string); ok {
					zones = append(zones, s)
				}
			}
		}
		var query *string
		if q, ok := spec["boundaryQuery"].(string); ok {
			query = &q
		}
		var desc *string
		if d, ok := spec["description"].(string); ok {
			desc = &d
		}
		live, err := handler.GetByName(ctx, name)
		if err != nil {
			return err
		}
		if live == nil {
			if _, err := handler.Create(ctx, name, zones, query, desc); err != nil {
				return err
			}
			printer.PrintSuccess("Boundary %q created", name)
			return nil
		}
		// zones is a spec-side shorthand for boundaryQuery and has no live
		// counterpart, so it would always compare as changed; leave it out.
		comparable := make(map[string]any, len(spec))
		for k, v := range spec {
			if k != "zones" {
				comparable[k] = v
			}
		}
		if len(zones) == 0 && !diffpkg.Compare("Boundary", name, comparable, live).HasChanges() {
			printer.PrintSuccess("Boundary %q unchanged", name)
			return nil
		}
		if _, err := handler.Update(ctx, utils.StringFrom(live, "uuid"), &name, zones, query, desc); err != nil {
			return err
		}
		printer.PrintSuccess("Boundary %q updated", name)
		return nil

	case "binding":
		groupUUID := utils.StringFrom(spec, "group")
		policyUUID := utils.StringFrom(spec, "policy")
		if groupUUID == "" || policyUUID == "" {
			return fmt.Errorf("binding spec requires 'group' and 'policy' fields")
		}
		bound, err := resources.NewGroupHandler(c).GetPolicies(ctx, groupUUID)
		if err != nil {
			return err
		}
		for _, p := range bound {
			if strings.EqualFold(p, policyUUID) {
				printer.PrintSuccess("Binding unchanged (group=%s, policy=%s)", groupUUID, policyUUID)
				return nil
			}
		}
		if _, err := resources.NewBindingHandler(c).Create(ctx, groupUUID, policyUUID, nil, nil); err != nil {
			return err
		}
		printer.PrintSuccess("Binding created (group=%s, policy=%s)", groupUUID, policyUUID)
		return nil

	default:
		return fmt.Errorf("unsupported resource kind: %s (expected Group, Policy, Boundary, or Binding)", kind)
	}
}

// mergePolicy builds a policy update body. PUT on a policy requires name,
// description and statementQuery, so fields the spec omits keep their live
// values rather than being cleared.
func mergePolicy(live, spec map[string]any) map[string]any {
	body := map[string]any{}
	for _, field := range []string{"name", "description", "statementQuery", "tags"} {
		if v, ok := live[field]; ok && v != nil {
			body[field] = v
		}
	}
	for _, field := range []string{"name", "description", "statementQuery", "tags"} {
		if v, ok := spec[field]; ok {
			body[field] = v
		}
	}
	if _, ok := body["description"]; !ok {
		body["description"] = ""
	}
	return body
}
