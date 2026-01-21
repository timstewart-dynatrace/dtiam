# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/jtimothystewart/GO-IAM-CLI/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/jtimothystewart/GO-IAM-CLI/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/jtimothystewart/GO-IAM-CLI/releases/tag/v1.0.0
