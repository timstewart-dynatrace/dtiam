# Phase 04 — Advanced Operations
Status: PENDING

## Goal
Group cloning, parameterized policies, app/schema boundary creation.

## Prerequisites
- Phase 2 complete (architecture alignment)
- Phase 3 complete (apps and schemas handlers needed for boundary validation)

## Reference
- Python-IAM-CLI source: /Users/Shared/GitHub/Python-IAM-CLI
- Key files: src/dtiam/commands/group.py (clone), src/dtiam/commands/boundary.py (create-app/schema-boundary)

## Tasks

### 4.1 Group Clone
- [ ] Add `group clone SOURCE --name NEW_NAME` subcommand to group/group.go
- Flags:
  - `--name` (required) — name for new group
  - `--description` — optional description
  - `--include-members` — copy group members to new group
  - `--include-policies` — copy policy bindings to new group
- Logic:
  1. Resolve source group by name/UUID
  2. Create new group with --name and --description
  3. If --include-members: iterate source members, add each to new group via GroupHandler.AddMember
  4. If --include-policies: iterate source bindings, create each in new group via BindingHandler.Create
- Dry-run: show what would be cloned without executing
- Example help text required

### 4.2 Parameterized Policies
- [ ] Understand `${bindParam:name}` syntax in policy statement queries
- [ ] Add `--param key=value` repeatable flag to `create binding` (create/create.go)
- [ ] Pass parameters array in binding creation payload:
  ```json
  {
    "policyUuid": "...",
    "groups": ["..."],
    "parameters": {
      "paramName": "paramValue"
    }
  }
  ```
- [ ] Update `describe policy` to show bind parameters when present
- [ ] Update BindingHandler.Create to accept optional parameters map

### 4.3 App Boundaries
- [ ] Add `boundary create-app-boundary NAME` subcommand to boundary/boundary.go
- Flags:
  - `--app-ids` (required, comma-separated or repeatable)
  - `--not-in` — use NOT IN instead of IN (exclude apps)
  - `--environment` — for validation against App Engine Registry
  - `--description` — optional
  - `--skip-validation` — skip app ID validation
- Logic:
  1. If not --skip-validation: validate each app ID via AppHandler.Get (from Phase 3)
  2. Generate boundary query: `shared:app-id IN ("app1", "app2")` or `NOT IN`
  3. Create boundary via BoundaryHandler.Create
- Dry-run: show generated query without creating

### 4.4 Schema Boundaries
- [ ] Add `boundary create-schema-boundary NAME` subcommand
- Same pattern as app boundaries but for settings schemas:
  - `--schema-ids` (required)
  - `--not-in`, `--environment`, `--description`, `--skip-validation`
  - Query: `settings:schemaId IN ("builtin:alerting.profile", ...)`
  - Validate via SchemaHandler.Get (from Phase 3)

### 4.5 Group Setup
- [ ] Add `group setup --name NAME --policies-file FILE` subcommand
  - Reads a YAML/JSON file with policies, bindings, and optional boundaries
  - Creates group, then applies all policy bindings from file
  - One-step provisioning for new teams
  - Flags: `--name`, `--description`, `--policies-file`, `--dry-run`

### 4.6 User Info & Bulk Group Operations
- [ ] Add `user info IDENTIFIER` subcommand — detailed user view (alias for `describe user`)
- [ ] Add `user bulk-add-groups --user EMAIL --groups "g1,g2,g3"` — add to multiple groups in one call
- [ ] Add `user bulk-remove-groups --user EMAIL --groups "g1,g2,g3" --force` — remove from multiple groups

### 4.7 Export Individual Resource Types
- [ ] Enhance `export` to support per-resource exports:
  - `export environments`, `export users`, `export bindings`, `export boundaries`, `export service-users`
  - Same flags as `export all` (--output, --format, --detailed)

## Key Files
- MODIFY: `internal/commands/group/group.go`, `internal/commands/create/create.go`, `internal/commands/boundary/boundary.go`
- MODIFY: `internal/resources/bindings.go` (add parameters to Create)

## Acceptance Criteria
- [ ] `dtiam group clone "Source" --name "Copy" --include-members --include-policies` works
- [ ] `dtiam create binding --group X --policy Y --param env=production` sends parameters
- [ ] `dtiam boundary create-app-boundary "My Apps" --app-ids dynatrace.dashboards,dynatrace.logs` validates and creates
- [ ] `dtiam boundary create-schema-boundary "My Schemas" --schema-ids builtin:alerting.profile` validates and creates
- [ ] All new commands have Example help text
- [ ] All new commands support --dry-run and --force where applicable
- [ ] Tests for clone logic, parameter passing, boundary query generation

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
