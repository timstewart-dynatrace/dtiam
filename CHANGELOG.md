# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.2] - 2026-04-16

### Added

- **Policies-with-boundaries doc** — `docs/POLICIES_WITH_BOUNDARIES.md` ported from Python-dtiam; documents the six rules for resolving effective permissions when boundaries apply, with seven worked examples and a dtiam-specific command table
- **Command hierarchy diagram** — `images/04-command-hierarchy_930x500.svg` embedded in `docs/COMMANDS.md`; visualizes the four tiers (core CRUD verbs, resource-scoped commands, specialized workflows, resource vocabulary) with a `MARKDOWN_TABLE_ALTERNATIVE` fallback
- **README/ARCHITECTURE cross-links** — both now reference `POLICIES_WITH_BOUNDARIES.md`

## [2.0.1] - 2026-04-16

### Added

- **Architecture diagrams** — three SVG diagrams (`images/01-architecture_930x500.svg`, `02-two-tier-bindings_930x500.svg`, `03-oauth2-flow_930x500.svg`) embedded in `README.md`, `docs/ARCHITECTURE.md`, and `docs/QUICK_START.md`, with `MARKDOWN_TABLE_ALTERNATIVE` text fallbacks for environments that strip images

## [2.0.0] - 2026-04-08

### Added

- **Template engine** — `internal/template/` package with Go `text/template` renderer, XDG-based template store, and 5 built-in templates (group-team, policy-readonly, policy-admin, binding-simple, boundary-mz)
- **Template commands** — `template list/show/render/apply/save/delete/path` for managing and using resource templates
- **Apply command** — `dtiam apply -f resource.yaml` declarative resource creation with auto-detect kind, `--set` template variables, `--dry-run`, and multi-document YAML support
- **Bulk groups+policies** — `bulk create-groups-with-policies --file FILE` creates groups and binds policies from CSV/YAML/JSON in one step
- **Export template enhancement** — `export policy --as-template` now uses Go template syntax (`{{.name}}`) compatible with `dtiam template apply`

## [1.5.0] - 2026-04-08

### Added

- **Group clone** — `group clone SOURCE --name NEW [--include-members] [--include-policies]` clones a group with optional members and policy bindings
- **App boundaries** — `boundary create-app-boundary NAME --app-ids ...` creates boundaries scoped to specific Dynatrace apps with optional validation
- **Schema boundaries** — `boundary create-schema-boundary NAME --schema-ids ...` creates boundaries scoped to specific Settings 2.0 schemas with optional validation
- **Group setup** — `group setup --name NAME --policies-file FILE` one-step group provisioning from YAML/JSON policy definitions
- **Parameterized policies** — `create binding --param key=value` passes bind parameters for `${bindParam:name}` substitution in policy statements

## [1.4.0] - 2026-04-08

### Added

- **Account capabilities** — `account capabilities [SUBSCRIPTION]` command to list capability flags from subscriptions
- **Per-resource exports** — `export environments`, `export users`, `export bindings`, `export boundaries`, `export service-users` subcommands with `--detailed` enrichment
- **User info** — `user info IDENTIFIER` command as alias for `describe user` (Python CLI parity)

## [1.3.0] - 2026-04-08

### Added

- **Platform tokens** — `get tokens`, `create token`, `delete token` commands for IAM platform token management
- **App Engine Registry** — `get apps` command with `--environment` flag for listing apps
- **Settings schemas** — `get schemas` command with `--environment` and `--name` filter for Settings 2.0 schemas
- **Resty HTTP client** — replaced stdlib `net/http` with `go-resty/resty/v2` for built-in retry, debug mode, and request hooks
- **Viper config** — automatic `DTIAM_*` env var binding via `spf13/viper`, XDG paths via `adrg/xdg`
- **Structured logging** — `internal/logging/` package with `sirupsen/logrus` (verbosity levels, HTTP request logging)
- **Diagnostic errors** — `internal/diagnostic/` package with exit codes (auth=3, not-found=4, forbidden=5) and troubleshooting suggestions
- **Command suggestions** — `internal/suggest/` Levenshtein engine for typo correction
- **Struct-tag output** — `internal/output/structprinter.go` reads `table` struct tags via reflection for type-safe column rendering
- **Typed resource structs** — `internal/resources/types.go` with `json` + `table` tags for all 12 resource types
- **Credential enhancements** — `api-url`, `scopes`, `environment-url`, `environment-token` fields per credential
- **New env vars** — `DTIAM_API_URL`, `DTIAM_SCOPES`, `DTIAM_ENVIRONMENT_URL`, `DTIAM_ENVIRONMENT_TOKEN`, `DTIAM_OUTPUT`, `DTIAM_VERBOSE`
- **Comprehensive test coverage** — 737 tests across 26 packages (resource handlers, commands, output, auth, prompt, CLI state)

### Fixed

- **Bulk flag conflict** — removed duplicate `-f` shorthand on `remove-users-from-group` (was `--file` and `--force` both using `-f`)

## [1.2.1] - 2026-04-07

### Fixed

- **Dead code removal** — removed unused `detailColumns()` function and stale import in `describe` package
- **Loop simplification** — simplified append loop to `append(fields, result.Permissions...)` in `analyze` package
- **Boundary cleanup** — removed unused `columns` variable and dead loop in `boundary` package
- **Error handling** — check all `SetContext()`/`UseContext()` error returns in config tests

## [1.2.0] - 2026-04-06

### Added

- **Centralized confirmation prompts** — new `internal/prompt` package with `Confirm()` and `ConfirmDelete()` functions; replaces inline `bufio.NewReader`/`fmt.Scanln` implementations across all destructive commands
- **Safe type assertion helpers** — new `internal/utils/safemap.go` with `StringFrom`, `IntFrom`, `BoolFrom`, `SliceFrom`, `MapFrom`, `StringSliceFrom` for safely extracting values from `map[string]any` API responses without panic risk
- **API URL constants** — centralized all Dynatrace API base URLs and paths in `internal/client/urls.go`; eliminates hardcoded URLs scattered across 6 resource handlers
- **Example help text** — added Cobra `Example` fields to all ~50 CLI subcommands with real-world usage patterns including `--dry-run`, `--force`, `--plain`, and `-o json` examples
- **Command standards** — new `.claude/rules/command-standards.md` defining mandatory patterns for all commands (output through printer, `--plain` behavior, dry-run, confirmation, error handling)
- **Phase planning** — v2.0.0 refactor phase docs in `.claude/phases/`
- **Tests** — unit tests for safemap utilities and URL constants

### Changed

- **`--plain` mode JSON override** — `--plain` flag now forces JSON output when table/wide format is selected, ensuring machine-consumable output for AI agents and scripts
- **Standardized confirmation flow** — all destructive operations now use `prompt.ConfirmDelete()` with consistent `--force` flag behavior instead of ad-hoc implementations
- **Consistent dry-run output** — all dry-run messages use `printer.PrintWarning()` instead of raw `fmt.Printf`
- **Bulk force flag** — standardized to lowercase `-f` (was `-F`) for consistency with other commands
- **Client consolidation** — merged duplicate `tokenProviderAdapter` (3 copies) and `createClient()` (2 copies) into single implementations in `common/client.go`
- **Safe type assertions** — replaced ~40 unsafe bare `.(string)` assertions with `StringFrom()` helpers, preventing panics on unexpected API response shapes

### Fixed

- **Panic prevention** — eliminated potential panics from unguarded type assertions on API response maps throughout all resource handlers
- **Bulk confirmation** — `bulk remove-users-from-group` was using `fmt.Scanln` instead of the centralized confirmation prompt

## [1.1.1] - 2025-01-21

### Fixed

- **API URL correction** — fixed policies, bindings, and boundaries handlers to use correct `/repo/` endpoint path (`https://api.dynatrace.com/iam/v1/repo/...` instead of `/accounts/{uuid}/repo/...`); affected `policies.go`, `bindings.go`, `boundaries.go`
- **Name resolution fallback** — fixed `GetOrResolve` in `handler.go` to properly fall back to list search when direct GET returns 404; now searches by UUID fields (`uuid`, `uid`, `id`) then by name; fixes `describe group` failures on valid groups

## [1.1.0] - 2025-01-21

### Added

- **Bulk operations** — process multiple resources from CSV/YAML/JSON files:
  - `bulk add-users-to-group` / `bulk remove-users-from-group` — manage group membership at scale
  - `bulk create-groups` — create multiple groups from file
  - `bulk create-bindings` — create policy bindings from file
  - `bulk export-group-members` — export group membership to file
- **Export commands** — backup and migration support:
  - `export all` — export all IAM resources (groups, policies, bindings, boundaries) to files
  - `export group` — export single group with members and policy bindings
  - `export policy` — export single policy, optionally as a reusable template
- **Permissions analysis** — comprehensive IAM analysis tools:
  - `analyze user-permissions` / `analyze group-permissions` — calculate effective permissions from policy bindings
  - `analyze permissions-matrix` — generate cross-reference matrix of permissions by policy or group
  - `analyze policy` — analyze a policy's permission statements and binding usage
  - `analyze least-privilege` — identify policies granting excessive permissions
  - `analyze effective-user` / `analyze effective-group` — query Dynatrace resolution API for effective permissions
- **Permissions utilities** — new `internal/utils/permissions.go` with `ParseStatementQuery()`, `PermissionsCalculator`, `PermissionsMatrix`, and `EffectivePermissionsAPI`
- **Account enhancements** — `account check-capacity` for pre-flight capacity checks; `account limits --summary` for usage percentages
- **Validation** — comprehensive validation script (`scripts/validate.sh`) and `make validate` target
- **Unit tests** — added tests for permissions parsing, output formatting, configuration management, and HTTP client error handling

### Changed

- **Boundary query syntax** — updated to modern Dynatrace format: `environment:management-zone IN ("Zone")` with `storage:dt.security_context` and `settings:dt.security_context` queries (replaces legacy `managementZone.name = "Zone"`)
- **Pre-push checklist** — enhanced CLAUDE.md with mandatory version management checklist

### Documentation

- Full documentation for bulk, export, and analyze commands in COMMANDS.md
- Updated README.md with new command groups and resources table

## [1.0.0] - 2025-01-20

### Added

- **Initial release** — Go implementation of dtiam CLI (converted from Python)
- **kubectl-style commands** — `get`, `describe`, `create`, `delete` with consistent verb-noun syntax
- **Multi-context configuration** — named contexts with separate credentials, XDG Base Directory support
- **Dual authentication** — OAuth2 with automatic token refresh (recommended) and static bearer token
- **Resource handlers** — full CRUD for groups, users, service users, policies, bindings, boundaries, environments, limits, and subscriptions
- **Output formats** — table, wide, JSON, YAML, CSV, and plain modes
- **Safety features** — dry-run mode, confirmation prompts for destructive operations, verbose debugging
- **HTTP client** — exponential backoff retry logic with rate limit handling (429 responses)
- **Name resolution** — user-friendly name-to-UUID resolution for all resource identifiers
- **Cross-platform** — single binary for Linux, macOS (Intel + Apple Silicon), and Windows
- **Build system** — Makefile with build/test/lint targets and goreleaser for multi-platform releases

### Documentation

- CLAUDE.md with development workflow standards and mandatory pre-push checklist
- README.md with installation, authentication, and usage guide
- docs/QUICK_START.md, docs/COMMANDS.md, docs/ARCHITECTURE.md, docs/API_REFERENCE.md

[Unreleased]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/timstewart-dynatrace/GO-dtiam/releases/tag/v1.0.0
