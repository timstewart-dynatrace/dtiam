# Roadmap — dtiam v2.0.0

This document consolidates all implementation phases for bringing dtiam to full feature parity with the Python implementations and aligning architecture with [dtctl](https://github.com/dynatrace-oss/dtctl).

> **Target**: v2.0.0 release with 100% feature parity + dtctl architectural patterns + >70% test coverage.

## Status Overview

| Phase | Name | Status | Description |
|-------|------|--------|-------------|
| 1 | Foundation | **Done** | Safe type assertions, URL constants, client consolidation |
| 1.5 | Command Standards | **Done** | Centralized prompts, --plain JSON override, Example help text |
| 2 | Architecture Alignment | **Done** | Resty client, Viper config, Logrus logging, struct-tag output |
| 3 | Quick Wins | **Done** | Account capabilities command, per-resource exports, user info alias |
| 4 | Advanced Operations | **Done** | Group clone, app/schema boundaries, group setup, parameterized policies |
| 5 | Templates & Apply | Pending | Template engine, template commands, declarative apply, bulk groups+policies |
| 6 | Analysis & Caching | Pending | Permission diff/gaps, enhanced least-privilege, functional cache |
| 7 | Zones (Legacy) | Pending | Management zones via entities API (legacy, low priority) |
| 8 | Polish & v2.0.0 Release | Pending | >70% coverage, full doc sweep, version bump to 2.0.0 |
| R8 | Retroactive Tests | **Done** | 737 tests across 26 packages — resource handlers, commands, output, auth, prompt |

---

## Completed Phases

### Phase 1 — Foundation (Done)
Safe type assertions (`safemap.go`), centralized API URL constants (`urls.go`), client consolidation.

### Phase 1.5 — Command Standards (Done)
Centralized `prompt.ConfirmDelete()`, `--plain` JSON override, `Example` help text on all commands. Standards documented in `.claude/rules/command-standards.md`.

### Phase 2 — Architecture Alignment (Done)
Resty HTTP client with retry/backoff, Viper config with env binding, Logrus structured logging, diagnostic errors with exit codes, struct-tag output (`structprinter.go`), Levenshtein command/flag suggestions, config credential enhancements (api-url, scopes, env-url/token), XDG base directory support.

### Phase R8 — Retroactive Test Coverage (Done)
737 tests across 26 packages covering: all 13 resource handlers (mock HTTP), all command packages (flag parsing, dry-run, args validation), output formats (table, JSON, YAML, CSV, wide, plain), auth (OAuth refresh, bearer, client ID extraction), prompt (force/plain skip), bulk/export file parsing, analyze permission logic.

---

## Phase 3 — Quick Wins

**Goal**: Small-effort features that complete partial implementations and fill obvious gaps.

**Prerequisites**: Phase 2 (done)

| Task | Description | Effort |
|------|-------------|--------|
| 3.1 Account Capabilities | `account capabilities [SUBSCRIPTION]` — handler already exists (`SubscriptionHandler.GetCapabilities()`), just needs command + columns | Small |
| 3.2 Per-Resource Exports | `export environments`, `export users`, `export bindings`, `export boundaries`, `export service-users` — individual export subcommands | Small |
| 3.3 User Info Alias | `user info IDENTIFIER` — alias for `describe user`, for Python CLI parity | Trivial |

---

## Phase 4 — Advanced Operations

**Goal**: Group cloning, parameterized policies, app/schema boundary creation helpers.

**Prerequisites**: Phase 2 (done), apps/schemas handlers (done)

| Task | Description |
|------|-------------|
| 4.1 Group Clone | `group clone SOURCE --name NEW --include-members --include-policies` |
| 4.2 App Boundaries | `boundary create-app-boundary NAME --app-ids ... [--not-in] [--environment]` with validation |
| 4.3 Schema Boundaries | `boundary create-schema-boundary NAME --schema-ids ... [--not-in] [--environment]` with validation |
| 4.4 Group Setup | `group setup --name NAME --policies-file FILE` — one-step group provisioning |
| 4.5 Parameterized Policies | `--param key=value` on `create binding` for `${bindParam:name}` substitution |

---

## Phase 5 — Templates & Apply

**Goal**: Template engine, template commands, declarative apply command.

**Prerequisites**: Phase 4 (parameterized policies needed for template variable support)

| Task | Description |
|------|-------------|
| 5.1 Template Engine | Go `text/template` renderer with `--set key=value`, stored at `$XDG_DATA_HOME/dtiam/templates/` |
| 5.2 Template Commands | `template list/show/render/apply/save/delete/path` |
| 5.3 Apply Command | `dtiam apply -f resource.yaml` — auto-detect kind, create-or-update, `--dry-run` |
| 5.4 Export as Template | Enhance `export policy --as-template` with full Go template syntax |
| 5.5 Bulk Groups+Policies | `bulk create-groups-with-policies --file FILE` — CSV-based group + binding + boundary creation |

---

## Phase 6 — Analysis & Caching

**Goal**: Enhanced analysis with permission diff and risk scoring, functional in-memory cache.

**Prerequisites**: Phase 2 (done)

| Task | Description |
|------|-------------|
| 6.1 Permission Diff | `analyze diff-permissions ENTITY1 ENTITY2` — compare effective permissions |
| 6.2 Permission Gaps | `analyze permission-gaps` — find unused policies, unbound groups |
| 6.3 Enhanced Least-Privilege | Risk scoring 1-10, `--min-severity` filter, unused/duplicate detection |
| 6.4 TTL Cache | Generic in-memory cache with `sync.RWMutex`, HTTP middleware for GET caching |
| 6.5 Cache Commands | Make `cache stats/clear` functional, add `cache keys/set-ttl/reset-stats` |

---

## Phase 7 — Zones (Legacy)

**Goal**: Management zone support via Dynatrace entities API. Legacy feature for Python CLI parity.

**Prerequisites**: Phase 2 (done)

| Task | Description |
|------|-------------|
| 7.1 Zone Handler | `ZoneHandler` using `/api/v2/entities?entitySelector=type("MANAGEMENT_ZONE")` |
| 7.2 Zone Commands | `get zones [NAME]`, `zones export`, `zones compare-groups` |

**Note**: This is a legacy feature. Management zones are being superseded by Grail-based access control. Implement only if there is user demand.

---

## Phase 8 — Polish & v2.0.0 Release

**Goal**: Final quality pass, documentation sweep, version bump to 2.0.0.

**Prerequisites**: All prior phases complete (or consciously deferred)

| Task | Description |
|------|-------------|
| 8.1 Test Coverage | Target >70% overall, add tests for any new Phase 3-7 features |
| 8.2 Documentation Sweep | Update CLAUDE.md, COMMANDS.md, README.md, ARCHITECTURE.md for all new features |
| 8.3 Version Bump | `pkg/version/version.go` → 2.0.0, git tag `v2.0.0`, GitHub release |
| 8.4 Final Verification | `make build && make test && make lint`, smoke test all commands |

---

## Current Implementation Summary

**Fully implemented (70+ subcommands, 13 resource handlers, 770+ tests):**
- All core CRUD: groups, users, service users, policies, bindings, boundaries
- Resource types: environments, limits, subscriptions, platform tokens, apps, schemas
- Bulk ops: add/remove users, create groups, create bindings, export group members
- Export: all resources, single group, single policy (with --as-template), per-resource exports (environments, users, bindings, boundaries, service-users)
- Analyze: user/group permissions, permissions matrix, policy analysis, least-privilege, effective permissions via API
- Account: limits, subscriptions, capacity check, forecast, capabilities
- User: add/remove/replace groups, list groups, create, info
- Config: multi-context, multi-credential, XDG paths
- Output: table, wide, JSON, YAML, CSV with --plain machine mode

**Stub/placeholder:**
- Cache commands (no-op, documented as non-persistent)

**Not yet implemented (~4% of Python CLI parity):**
- Templates & apply (Phase 5)
- Zones (Phase 7)
- Permission diff/gaps (Phase 6)
