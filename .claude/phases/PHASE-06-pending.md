# Phase 06 — Caching and Analysis Enhancements
Status: PENDING

## Goal
Real in-memory caching, enhanced least-privilege analysis, permission diff.

## Prerequisites
- Phase 5 complete (all resource types available for caching)

## Reference
- Python-IAM-CLI: src/dtiam/utils/cache.py, src/dtiam/commands/analyze.py

## Tasks

### 6.1 In-Memory TTL Cache
- [ ] Create `internal/cache/cache.go`:
  - Generic TTL cache: `Cache` struct with `Get`, `Set`, `Delete`, `Clear`, `Keys`, `Stats`
  - Thread-safe with `sync.RWMutex`
  - Default TTL: 300 seconds (5 minutes)
  - Stats tracking: hits, misses, hit rate
  - `GlobalCache` singleton
- [ ] Create `internal/cache/middleware.go`:
  - HTTP middleware that caches GET responses by URL+params key
  - Configurable TTL per-endpoint
  - Skip cache for non-GET requests
- [ ] Integrate into `internal/client/client.go`:
  - Wrap GET requests with cache middleware
  - Cache active within single CLI invocation (useful for analyze, export --detailed)

### 6.2 Cache Commands (Make Functional)
- [ ] Rewrite `internal/commands/cache/cache.go`:
  - `cache stats` — display hit rate, entry count, TTL
  - `cache clear` — clear all entries (with --force, --expired-only)
  - `cache keys` — list cached keys (with --prefix, --limit)
  - `cache set-ttl SECONDS` — set default TTL
  - `cache reset-stats` — reset hit/miss counters

### 6.3 Enhanced Least-Privilege Analysis
- [ ] Enhance `analyze least-privilege` in analyze/analyze.go:
  - Add numeric risk scoring (1-10 per policy)
  - Add `--min-severity` filter flag (low, medium, high)
  - Detect additional patterns:
    - Unused policies (bound to 0 groups)
    - Duplicate policies (same permissions)
    - Overlapping permissions across policies
  - Output as structured report with summary
  - Support `--export FILE` for JSON/CSV report

### 6.4 Permission Diff
- [ ] Add `analyze diff-permissions ENTITY1 ENTITY2` subcommand
  - Compare effective permissions between two users or two groups
  - Show: permissions only in ENTITY1, only in ENTITY2, shared
  - Support `--format summary` that groups by service
  - Table output: permission | entity1 | entity2 | status (added/removed/shared)

## Key Files
- CREATE: `internal/cache/cache.go`, `internal/cache/middleware.go`
- REWRITE: `internal/commands/cache/cache.go`
- MODIFY: `internal/commands/analyze/analyze.go`, `internal/client/client.go`

## Acceptance Criteria
- [ ] `dtiam cache stats` shows real hit/miss statistics
- [ ] Running `dtiam export all --detailed` uses cache (fewer API calls)
- [ ] `dtiam analyze least-privilege --min-severity medium` filters findings
- [ ] `dtiam analyze diff-permissions user1@example.com user2@example.com` shows diff
- [ ] Cache tests: TTL expiration, concurrent access, stats accuracy
- [ ] All new commands have Example help text

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
