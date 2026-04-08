# Roadmap — dtiam v2.0.0

This document consolidates all implementation phases for bringing dtiam to full feature parity with the Python implementations and aligning architecture with [dtctl](https://github.com/dynatrace-oss/dtctl).

> **Target**: v2.0.0 release with 100% feature parity + dtctl architectural patterns + >70% test coverage.

## Status Overview

| Phase | Name | Status | Description |
|-------|------|--------|-------------|
| 1 | Foundation | **Done** | Safe type assertions, URL constants, client consolidation |
| 1.5 | Command Standards | **Done** | Centralized prompts, --plain JSON override, Example help text |
| 2 | Architecture Alignment | Pending | Resty client, Viper config, Logrus logging, struct-tag output |
| 3 | New Resource Types | Pending | Platform tokens, apps, schemas, zones, account capabilities |
| 4 | Advanced Operations | Pending | Group clone/setup, parameterized policies, app/schema boundaries |
| 5 | Templates & Apply | Pending | Template engine, declarative apply, bulk create-groups-with-policies |
| 6 | Caching & Analysis | Pending | TTL cache, permission gaps, permission diff, risk scoring |
| 7 | Polish & Release | Pending | >70% coverage, full docs, v2.0.0 tag |
| 8 | Retroactive Tests | Pending | Test coverage for all existing (pre-Phase 2) features |

---

## Phase 2 — Architecture Alignment

**Goal**: Adopt dtctl patterns for HTTP client, config, logging, errors, and output without changing command behavior.

**Prerequisites**: Phase 1 + 1.5 (done)

| Task | Description | Key Files |
|------|-------------|-----------|
| 2.1 HTTP Client | Rewrite to go-resty with retry/backoff | `internal/client/client.go` |
| 2.2 Config + XDG | Viper env binding, adrg/xdg paths | `internal/config/loader.go` |
| 2.3 Logging | Logrus structured logging | `internal/logging/logger.go` (new) |
| 2.4 Errors | cockroachdb/errors, exit codes 1-5 | `internal/cli/root.go`, all resources |
| 2.5 Struct Output | Typed structs with `json`+`table` tags | `internal/output/structprinter.go` (new) |
| 2.7 Suggestions | Levenshtein for unknown commands/flags | `internal/suggest/suggest.go` (new) |
| 2.8 Config Fields | api-url, scopes, env-url, env-token per credential | `internal/config/config.go` |

**Acceptance**: All existing commands work identically. Exit codes differentiated. Struct-tag output for groups/policies/users. Typo suggestions.

---

## Phase 3 — New Resource Types

**Goal**: Add resource types from Python that are missing in Go.

**Prerequisites**: Phase 2

| Task | Description | Python Source |
|------|-------------|--------------|
| 3.1 Platform Tokens | CRUD for `/platform-tokens` | `resources/platform_tokens.py` |
| 3.2 Apps | App Engine Registry (read-only, --environment) | `resources/apps.py` |
| 3.3 Schemas | Settings 2.0 schemas (read-only, --environment) | `resources/schemas.py` |
| 3.4 Zones | Management zones (legacy, list/export/compare) | `commands/zones.py` |
| 3.5 Account Capabilities | Capability flags per subscription | `commands/account.py` |

---

## Phase 4 — Advanced Operations

**Goal**: Group cloning, parameterized policies, app/schema boundary creation, user enhancements.

**Prerequisites**: Phase 2 + 3

| Task | Description |
|------|-------------|
| 4.1 Group Clone | `group clone SOURCE --name NEW --include-members --include-policies` |
| 4.2 Parameterized Policies | `--param key=value` on binding creation |
| 4.3 App Boundaries | `boundary create-app-boundary NAME --app-ids ...` |
| 4.4 Schema Boundaries | `boundary create-schema-boundary NAME --schema-ids ...` |
| 4.5 Group Setup | `group setup --name NAME --policies-file FILE` |
| 4.6 User Enhancements | `user info`, `user bulk-add-groups`, `user bulk-remove-groups` |
| 4.7 Per-Resource Exports | `export environments`, `export users`, `export bindings`, etc. |

---

## Phase 5 — Templates & Apply

**Goal**: Template engine, declarative apply command, bulk create-groups-with-policies.

**Prerequisites**: Phase 2-4

| Task | Description |
|------|-------------|
| 5.1 Template Engine | Go text/template with `--set key=value` |
| 5.2 Template Commands | list, show, render, apply, save, delete, path |
| 5.3 Apply Command | `dtiam apply -f resource.yaml` with auto-detect and create-or-update |
| 5.4 Export as Template | `export policy --as-template` |
| 5.5 Bulk Groups+Policies | CSV-based group + binding + boundary creation |

---

## Phase 6 — Caching & Analysis Enhancements

**Goal**: Real caching, enhanced analysis with risk scoring and permission diff.

**Prerequisites**: Phase 5

| Task | Description |
|------|-------------|
| 6.1 TTL Cache | Generic in-memory cache with sync.RWMutex, 5min default TTL |
| 6.2 Cache Commands | stats, clear, keys, set-ttl, reset-stats (functional) |
| 6.3 Enhanced Least-Privilege | Risk scoring 1-10, --min-severity, unused/duplicate detection |
| 6.4 Permission Gaps | Find unused policies, unbound groups |
| 6.5 Permission Diff | `analyze diff-permissions ENTITY1 ENTITY2` |

---

## Phase 7 — Polish & v2.0.0 Release

**Goal**: >70% test coverage, complete documentation, version bump.

**Prerequisites**: All prior phases

| Task | Description |
|------|-------------|
| 7.1 Test Coverage | Mock HTTP server, flag parsing tests, dry-run tests, >70% coverage |
| 7.2 Documentation | Update CLAUDE.md, COMMANDS.md, README.md, ARCHITECTURE.md |
| 7.3 Version Bump | `pkg/version/version.go` → 2.0.0, git tag |
| 7.4 Verification | Build, test, lint, smoke test all commands |

---

## Phase 8 — Retroactive Test Coverage

**Goal**: Comprehensive test coverage for all features that existed before Phase 2.

| Task | Description |
|------|-------------|
| 8.1 Resource Handler Tests | Mock HTTP tests for all 9 handlers (groups, users, policies, bindings, boundaries, environments, service-users, limits, subscriptions) |
| 8.2 Command Tests | Flag parsing, required args, dry-run behavior for all ~50 subcommands |
| 8.3 Output Tests | Table, JSON, YAML, CSV, wide, plain output for each resource type |
| 8.4 Auth Tests | OAuth token refresh, bearer token, client ID extraction |
| 8.5 Bulk/Export Tests | File parsing (CSV/JSON/YAML), error continuation, export formats |
| 8.6 Analyze Tests | Permission parsing, matrix generation, least-privilege detection |
| 8.7 Prompt Tests | Confirm/ConfirmDelete with force=true/false, plain mode skip |
| 8.8 Integration Tests | End-to-end command execution with mock server |

---

## Feature Gap Summary (vs Python-IAM-CLI v3.13.0)

**Already implemented (~85%)**:
All core CRUD, user/group/service-user management, bulk ops, export, analyze, account limits/subscriptions/forecast/check-capacity, boundary attach/detach, multi-context auth.

**Remaining (~15%)**:
Templates, platform tokens, apps, schemas, zones, group clone/setup, parameterized policies, app/schema boundaries, caching, permission gaps/diff, user info/bulk-groups, per-resource exports, config enhancements (api-url, scopes, env-url/token).

All gaps are tracked in `.claude/phases/PHASE-02 through PHASE-08`.
