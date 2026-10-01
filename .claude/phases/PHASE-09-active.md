# PHASE 09 — API Revalidation & dtctl Format Alignment

**Status:** active
**Started:** 2026-10-01

## Goal

Revalidate every Dynatrace API dtiam calls against current documentation, close
the gaps where documented API groups are unused, and align the project's
structure and conventions with `dtctl` (`/Users/Shared/GitHub/PROJECTS/DTCTL/dtctl`).

## Validation Baseline (2026-10-01)

All 14 endpoint patterns dtiam calls were verified current against
docs.dynatrace.com and the Dynatrace OpenAPI specs in
`ARCHIVE/go-dtctl-main/api-spec/`. Nothing deprecated or removed.

## Work Items

### Step 1 — OAuth scopes & version sync (2.0.3) — DONE
- [x] Add `account-uac-read`, `platform-token:tokens:manage`, `account-audit-logs-read`
- [x] Export `auth.DefaultScopeList`, annotate each scope with its API group
- [x] Regression test asserting scope coverage per API group
- [x] Sync version to 2.0.3 across version.go, settings.json, CLAUDE.md, core.md
- [x] Correct stale test-count claim in rules/testing.md

### Step 2 — Pagination (2.1.0)
- [ ] `page`/`size` support in client + BaseHandler, auto-follow all pages
- [ ] Apply to users, groups, service users, platform tokens
- [ ] Tests for multi-page, single-page, and empty responses

### Step 3 — New API groups (2.2.0)
- [ ] Audit logs — `GET /audit/v1/accounts/{uuid}`
- [ ] Reference data — `GET /ref/v1/account/permissions`
- [ ] Permission management — `{GET,POST,PUT,DELETE} /iam/v1/accounts/{u}/groups/{g}/permissions`
- [ ] Notifications — `POST /v1/accounts/{uuid}/notifications`
- [ ] DPS deeper: `subscriptions/{s}/environments/usage` (v2),
      `subscriptions/{s}/environments/cost` (v3), cost-monitors, cost-allocation
- [ ] Environment-level Platform IAM — `/platform/iam/v1/organizational-levels/...`

### Step 4 — dtctl format alignment (2.3.0)
- [ ] Split oversized verb files into `verb_resource.go` (analyze 954, export 900, bulk 872)
- [ ] Commit `.golangci.yml`
- [ ] Add `dtiam doctor`
- [ ] Governance files: AGENTS.md, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md, NOTICE
- [ ] install.sh / install.ps1

### Step 5 — Credential & UX parity (2.4.0)
- [ ] OS keyring for client secrets (currently plaintext YAML)
- [ ] `diff`, `watch`, agent auto-detection

### Step 6 — internal/ -> pkg/ (3.0.0, BREAKING)
- [ ] Promote reusable packages to `pkg/` so dtiam is importable like dtctl

## Next Step

Step 2 — pagination.
