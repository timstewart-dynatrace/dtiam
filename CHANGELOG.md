# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.1] - 2026-04-07

### Fixed

- Remove unused `detailColumns()` function and stale import in `describe` package
- Simplify loop to `append(fields, result.Permissions...)` in `analyze` package
- Remove unused `columns` variable and dead loop in `boundary` package
- Check all `SetContext()`/`UseContext()` error returns in config tests

## [1.2.0] - 2026-04-06

### Added

- Centralized `internal/prompt` package for confirmation prompts (`Confirm`, `ConfirmDelete`)
- Safe type assertion helpers in `internal/utils/safemap.go` (`StringFrom`, `IntFrom`, `BoolFrom`, `SliceFrom`, `MapFrom`, `StringSliceFrom`)
- API URL constants centralized in `internal/client/urls.go`
- `Example` help text on all ~50 CLI subcommands with real-world usage patterns
- Mandatory command standards documentation (`.claude/rules/command-standards.md`)
- Phase planning docs for v2.0.0 refactor (`.claude/phases/`)
- Tests for safemap utilities and URL constants

### Changed

- `--plain` mode now forces JSON output when table/wide format is selected (AI/machine consumption)
- All destructive operations use centralized `prompt.ConfirmDelete()` instead of inline implementations
- All dry-run messages use `printer.PrintWarning()` instead of raw `fmt.Printf`
- Bulk `--force` flag standardized to lowercase `-f` (was `-F`)
- Consolidated duplicate `tokenProviderAdapter` from 3 copies to 1 in `common/client.go`
- Consolidated duplicate `createClient()` from get/describe packages to `common.CreateClient()`
- Replaced ~40 unsafe bare `.(string)` type assertions with safe `StringFrom()` helpers
- Extracted hardcoded API URLs from 6 resource handlers to centralized constants

### Fixed

- Potential panics from unguarded type assertions on API response maps
- Bulk remove-users-from-group using `fmt.Scanln` instead of proper confirmation prompt

## [1.1.1] - 2025-01-21

### Fixed

- Fixed API URLs for policies, bindings, and boundaries handlers to use correct `/repo/` endpoint
  - Changed from `/accounts/{uuid}/repo/...` to `https://api.dynatrace.com/iam/v1/repo/...`
  - Affected files: `policies.go`, `bindings.go`, `boundaries.go`
- Fixed `GetOrResolve` function in `handler.go` to properly fall back to list search when direct GET returns 404
  - Now searches list by UUID fields (`uuid`, `uid`, `id`) and then by name
  - Fixes `describe group` command that was failing even when groups existed

## [1.1.0] - 2025-01-21

### Added

- Comprehensive validation script (`scripts/validate.sh`) and `make validate` target
- Unit tests for:
  - `internal/utils/permissions_test.go` - Permission parsing and matrix tests
  - `internal/output/format_test.go` - Output format tests
  - `internal/config/config_test.go` - Configuration management tests
  - `internal/client/client_test.go` - HTTP client and API error tests
- Bulk operations commands:
  - `bulk add-users-to-group` - Add multiple users from CSV/YAML/JSON file
  - `bulk remove-users-from-group` - Remove multiple users from file
  - `bulk create-groups` - Create multiple groups from file
  - `bulk create-bindings` - Create multiple policy bindings from file
  - `bulk export-group-members` - Export group members to file
- Export commands:
  - `export all` - Export all IAM resources to files
  - `export group` - Export single group with members and policies
  - `export policy` - Export single policy (optionally as template)
- Analyze commands for permissions analysis:
  - `analyze user-permissions` - Calculate effective permissions for a user
  - `analyze group-permissions` - Calculate effective permissions for a group
  - `analyze permissions-matrix` - Generate permissions matrix for policies/groups
  - `analyze policy` - Analyze a policy's permissions and bindings
  - `analyze least-privilege` - Identify policies with excessive permissions
  - `analyze effective-user` - Get user permissions via Dynatrace API
  - `analyze effective-group` - Get group permissions via Dynatrace API
- Permissions utilities (`internal/utils/permissions.go`):
  - `ParseStatementQuery()` - Parse policy statement queries
  - `PermissionsCalculator` - Calculate effective permissions
  - `PermissionsMatrix` - Generate permissions matrices
  - `EffectivePermissionsAPI` - Call Dynatrace resolution API
- Account commands:
  - `account check-capacity` - Check capacity for additional resources
  - `account limits --summary` - Summary with usage percentages

### Changed

- Updated boundary query format to use modern Dynatrace syntax:
  - Changed from `managementZone.name = "Zone"` to `environment:management-zone IN ("Zone")`
  - Added `storage:dt.security_context` and `settings:dt.security_context` queries
- Enhanced CLAUDE.md with mandatory pre-push checklist for version management

### Documentation

- Added full documentation for bulk, export, and analyze commands in COMMANDS.md
- Updated README.md with new command groups
- Updated CLAUDE.md project structure with new packages

## [1.0.0] - 2025-01-20

### Added

- Initial Go implementation of dtiam CLI (converted from Python)
- kubectl-style command structure with `get`, `describe`, `create`, `delete` commands
- Multi-context configuration management with XDG Base Directory support
- OAuth2 authentication with automatic token refresh
- Bearer token authentication for quick testing
- Resource handlers for all core IAM resources:
  - Groups (list, get, create, update, delete, members)
  - Users (list, get, create, delete, group management)
  - Service Users (list, get, create, update, delete, group management)
  - Policies (list, get, create, delete with level support)
  - Bindings (list, create, delete with boundary support)
  - Boundaries (list, get, create, delete, attach/detach)
  - Environments (list, get)
  - Limits (list, check-capacity)
  - Subscriptions (list, forecast)
- Multiple output formats: table, wide, JSON, YAML, CSV, plain
- Dry-run mode for previewing changes
- Verbose mode for debugging
- HTTP client with exponential backoff retry logic
- Rate limit handling (429 responses)
- Name-to-UUID resolution for user-friendly commands
- Confirmation prompts for destructive operations
- Cross-platform support (Linux, macOS, Windows)
- Makefile for build, test, lint operations
- goreleaser configuration for multi-platform releases

### Documentation

- Comprehensive CLAUDE.md with development workflow standards
- README.md with installation and usage instructions
- docs/QUICK_START.md - Getting started guide
- docs/COMMANDS.md - Full command reference
- docs/ARCHITECTURE.md - Technical design documentation
- docs/API_REFERENCE.md - Programmatic usage guide

[Unreleased]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/timstewart-dynatrace/GO-dtiam/releases/tag/v1.0.0
