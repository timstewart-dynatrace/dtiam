# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/jtimothystewart/GO-IAM-CLI/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/jtimothystewart/GO-IAM-CLI/releases/tag/v1.0.0
