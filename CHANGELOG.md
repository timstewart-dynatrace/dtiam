# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
