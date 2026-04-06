# Phase 05 — Template System and Apply
Status: PENDING

## Goal
Go template engine, template commands, declarative `apply` command, bulk create-groups-with-policies.

## Prerequisites
- Phase 2 complete (struct-tag output, XDG paths for template storage)
- Phase 3 complete (resource handlers for apply routing)
- Phase 4 complete (parameterized policies for template variable support)

## Reference
- go-dtctl-main: pkg/util/template/template.go (Go template with --set), pkg/apply/apply.go
- Python-IAM-CLI: src/dtiam/commands/template.py, src/dtiam/utils/templates.py

## Tasks

### 5.1 Template Engine
- [ ] Create `internal/template/engine.go`:
  - Go `text/template` based renderer
  - `RenderTemplate(content string, vars map[string]string) (string, error)`
  - `ParseSetFlags(flags []string) (map[string]string, error)` — parse `--set key=value`
  - Custom template functions: `default` (provide default values)
  - Template syntax: `{{.name}}`, `{{.name | default "value"}}`
- [ ] Create `internal/template/store.go`:
  - Template storage at `$XDG_DATA_HOME/dtiam/templates/`
  - `ListTemplates() ([]Template, error)`
  - `GetTemplate(name string) (*Template, error)`
  - `SaveTemplate(name string, content []byte) error`
  - `DeleteTemplate(name string) error`
- [ ] Create `internal/template/builtin.go`:
  - Embed built-in templates using Go `embed` package
  - Templates for common IAM patterns:
    - `group-team`: group with description template
    - `policy-readonly`: read-only policy template
    - `policy-admin`: admin policy template
    - `binding-simple`: group-to-policy binding
    - `boundary-mz`: management zone boundary

### 5.2 Template Commands
- [ ] Create `internal/commands/template/template.go` with subcommands:
  - `template list` — list available templates (built-in + custom)
  - `template show NAME` — display template content and required variables
  - `template render NAME --set key=value` — render to stdout without creating
  - `template apply NAME --set key=value` — render and create resource
  - `template save NAME --file FILE` — save custom template
  - `template delete NAME` — delete custom template (with --force)
  - `template path` — show templates directory
- [ ] Register in cmd/dtiam/main.go

### 5.3 Apply Command
- [ ] Create `internal/commands/apply/apply.go`:
  - `dtiam apply -f resource.yaml`
  - Auto-detect resource type from `kind` field in YAML/JSON
  - Route to appropriate handler: Group → GroupHandler.Create, Policy → PolicyHandler.Create, etc.
  - Support `--set key=value` for template variables in the file
  - Support `--dry-run` to preview
  - Handle create vs update (check if resource exists by name/UUID)
- [ ] Create `internal/format/detect.go`:
  - `DetectFormat(data []byte) (string, error)` — "json" or "yaml"
  - `ValidateAndConvert(data []byte) ([]byte, error)` — always return JSON
  - `YAMLToJSON(data []byte) ([]byte, error)`
  - `JSONToYAML(data []byte) ([]byte, error)`
- [ ] Register in cmd/dtiam/main.go

### 5.4 Export as Template
- [ ] Enhance `export policy --as-template` in export/export.go
  - Convert output to Go template syntax: `{{.name}}`, `{{.statement}}`
  - Useful for creating reusable policy templates from existing policies

### 5.5 Bulk Create-Groups-With-Policies
- [ ] Add `bulk create-groups-with-policies --file FILE` to bulk/bulk.go
- CSV columns: group_name, description, policy_name, level, level_id, management_zones, boundary_name, parameters
- Logic per row:
  1. Create group (or find existing by name)
  2. Resolve policy by name
  3. Create binding between group and policy
  4. If management_zones: create/find boundary, attach to binding
  5. If parameters: parse and include in binding
- Flags: `--file`, `--continue-on-error`, `--dry-run`

## Key Files
- CREATE: `internal/template/engine.go`, `internal/template/store.go`, `internal/template/builtin.go`
- CREATE: `internal/commands/template/template.go`, `internal/commands/apply/apply.go`
- CREATE: `internal/format/detect.go`
- MODIFY: `internal/commands/export/export.go`, `internal/commands/bulk/bulk.go`, `cmd/dtiam/main.go`

## Acceptance Criteria
- [ ] `dtiam template list` shows built-in templates
- [ ] `dtiam template render policy-readonly --set name=MyPolicy` outputs rendered YAML
- [ ] `dtiam template apply policy-readonly --set name=MyPolicy` creates the policy
- [ ] `dtiam apply -f group.yaml` creates/updates resource from file
- [ ] `dtiam apply -f group.yaml --dry-run` previews without creating
- [ ] `dtiam bulk create-groups-with-policies -f groups.csv` works end-to-end
- [ ] Template engine tests: rendering, variable substitution, missing variables, defaults
- [ ] All commands have Example help text

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
