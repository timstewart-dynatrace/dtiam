# Phase 07 — Polish, Testing, and v2.0.0 Release
Status: PENDING

## Goal
Comprehensive test coverage >70%, complete documentation, version bump to 2.0.0.

## Prerequisites
- All prior phases complete

## Tasks

### 7.1 Comprehensive Test Coverage
- [ ] Target >70% overall coverage
- [ ] Create `internal/client/testutil_test.go` — mock HTTP server for handler tests
- [ ] Add tests for every command package:
  - Flag parsing tests (verify required flags, default values)
  - Dry-run tests (verify no API calls made)
  - Error case tests (missing args, invalid input)
- [ ] Add tests for:
  - Template engine (rendering, variables, defaults, missing vars)
  - Cache (TTL, concurrent access, stats)
  - Prompt package (force=true skips, force=false blocks)
  - Suggest package (Levenshtein, closest match)
  - Format detection (JSON vs YAML, conversion)

### 7.2 Documentation Update
- [ ] Update `CLAUDE.md`:
  - New project structure (add prompt, suggest, template, cache, format, logging packages)
  - New API endpoints (platform tokens, apps, schemas)
  - New command patterns
  - Reference .claude/rules/command-standards.md
- [ ] Update `docs/COMMANDS.md`:
  - All new commands with full reference
  - template, apply, new get resources, account capabilities
  - Group clone, parameterized bindings, app/schema boundaries
  - Cache management commands
  - Analyze enhancements (diff-permissions, min-severity)
- [ ] Update `README.md`:
  - Features table (add template system, caching, app boundaries, etc.)
  - Getting started updates
  - AI/automation section (--plain, --force, -o json patterns)
- [ ] Update `docs/ARCHITECTURE.md`:
  - New architecture: resty, viper, logrus, struct tags
  - Cache layer diagram
  - Template system design
- [ ] Update `CHANGELOG.md`:
  - All changes organized by phase
  - Breaking changes noted (struct-tag output, new deps)

### 7.3 Version Bump
- [ ] Update `pkg/version/version.go` → `2.0.0`
  - Major bump justified by: new dependency surface, struct-tag output replaces column system, architecture changes
- [ ] Create git tag: `git tag -a v2.0.0 -m "Release version 2.0.0"`
- [ ] Create GitHub release with changelog

### 7.4 Final Verification
- [ ] `make build` — binary compiles for all platforms
- [ ] `make test` — all tests pass, >70% coverage
- [ ] `make lint` — no lint errors
- [ ] `dtiam version` shows 2.0.0
- [ ] Manual smoke test all major commands
- [ ] Verify backward compatibility: existing scripts using dtiam still work

## Key Files
- CREATE: Test files across all packages
- MODIFY: CLAUDE.md, README.md, docs/COMMANDS.md, docs/ARCHITECTURE.md, CHANGELOG.md
- MODIFY: pkg/version/version.go

## Acceptance Criteria
- [ ] `go test ./... -coverprofile` shows >70%
- [ ] All documentation updated and complete
- [ ] `make build && make test && make lint` all pass
- [ ] `dtiam version` shows 2.0.0
- [ ] All Phase 1-6 acceptance criteria still met (no regressions)

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
