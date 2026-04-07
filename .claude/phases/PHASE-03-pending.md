# Phase 03 — New Resource Types
Status: PENDING

## Goal
Add resource types from Python-IAM-CLI that are missing in Go: platform tokens, apps, schemas, account capabilities.

## Prerequisites
- Phase 2 complete (resty client, struct-tag output, logging)

## Reference
- Python-IAM-CLI source: /Users/Shared/GitHub/Python-IAM-CLI
- Key files: src/dtiam/resources/tokens.py, src/dtiam/resources/apps.py, src/dtiam/resources/schemas.py

## Tasks

### 3.1 Platform Tokens (CRUD)
- [ ] Create `internal/resources/tokens.go`:
  - `TokenHandler` with List, Get, Create, Delete
  - Typed `PlatformToken` struct with `json` + `table` tags
  - API: uses IAM accounts endpoint /platform-tokens (verify from Python source)
- [ ] Add commands:
  - `get tokens [ID]` — list or get (add to get/get.go)
  - `describe token ID` — detailed view (add to describe/describe.go)
  - `create token --name NAME --scopes SCOPES --expires-in 30d` (add to create/create.go)
  - `delete token ID` (add to delete/delete.go)
- [ ] Flags: --name, --scopes, --expires-in, --save-token PATH
- [ ] On create, display the secret (cannot be retrieved later) and optionally save to file

### 3.2 Apps (App Engine Registry)
- [ ] Create `internal/resources/apps.go`:
  - `AppHandler` with List, Get
  - API: `https://{env-id}.apps.dynatrace.com/platform/app-engine/registry/v1/apps`
  - Requires `--environment` flag or `DTIAM_ENVIRONMENT_URL` env var
  - Environment can be: env-id (abc12345) or full URL
- [ ] Add `get apps [ID]` command:
  - `--environment` flag (required)
  - `--ids` flag (output only app IDs, useful for policy conditions)
  - `--name` filter

### 3.3 Schemas (Settings API)
- [ ] Create `internal/resources/schemas.go`:
  - `SchemaHandler` with List, Get
  - API: `https://{env-id}.live.dynatrace.com/api/v2/settings/schemas`
  - Requires `--environment` flag
- [ ] Add `get schemas [ID]` command:
  - `--environment` flag (required)
  - `--builtin` flag (filter to builtin: schemas only)
  - `--search` / `--name` filter
  - `--ids` flag (output only schema IDs)

### 3.4 Zones (Legacy)
- [ ] Create `internal/resources/zones.go`:
  - `ZoneHandler` with List, Get
  - API: `{environment_url}/api/v2/entities?entitySelector=type("MANAGEMENT_ZONE")`
  - Requires `--environment` flag or config environment-url
- [ ] Add commands:
  - `get zones [NAME]` — list management zones (--name filter, --environment)
  - `zones list` — alias (legacy compatibility with Python-IAM-CLI)
  - `zones export` — export zones to YAML/JSON (--output-dir, --format)
  - `zones compare-groups` — compare zone names with IAM group names
- [ ] Mark as legacy/deprecated in help text

### 3.5 Account Capabilities
- [ ] Add `account capabilities [SUBSCRIPTION]` subcommand
  - Uses existing `SubscriptionHandler.GetCapabilities()` (internal/resources/subscriptions.go)
  - Optional subscription UUID/name argument
  - Table output with capability name, enabled status

## Key Files
- CREATE: `internal/resources/tokens.go`, `internal/resources/apps.go`, `internal/resources/schemas.go`
- MODIFY: `internal/commands/get/get.go`, `internal/commands/create/create.go`, `internal/commands/delete/delete.go`, `internal/commands/describe/describe.go`, `internal/commands/account/account.go`

## Acceptance Criteria
- [ ] `dtiam get tokens` lists platform tokens
- [ ] `dtiam create token --name "CI Token" --scopes "account-idm-read"` creates token
- [ ] `dtiam get apps --environment abc12345` lists apps
- [ ] `dtiam get schemas --environment abc12345 --builtin` lists builtin schemas
- [ ] `dtiam account capabilities` shows capabilities
- [ ] All new commands have Example help text
- [ ] All new commands support -o json/yaml/table/wide/csv and --plain
- [ ] Tests for each new handler

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
