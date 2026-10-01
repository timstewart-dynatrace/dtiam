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

### Step 2 — Pagination (2.1.0) — DONE
- [x] `client.PaginationConfig` with cursor and page-number styles
- [x] `BaseHandler.List` follows all pages; nil config = single request
- [x] Applied to service users and platform tokens (the only paginated
      Account Management endpoints; users/groups/policies return the full
      collection with no paging parameters)
- [x] Fixed wrong `ListKey` on both: they return `results`, not `items`,
      so both commands returned empty lists against the live API
- [x] 8 handler paging tests + 7 config tests

**Correction to the original finding:** users and groups were listed as
truncation risks. They are not -- those endpoints expose no paging parameters
and return `{count, items}` whole. The real defect was narrower but worse:
wrong response key on the two endpoints that do paginate.

### Step 3 — New API groups (2.2.0) — DONE
- [x] Audit logs — `get audit-logs`, with warning surfacing for partial results
- [x] Reference data — `get available-permissions`
- [x] Permission management — `group permissions` / `grant-permission` / `revoke-permission`
- [x] Notifications — `account notifications`
- [x] DPS deeper — `account environment-usage` (v2), `account environment-cost` (v3)
- [x] Environment-level Platform IAM — `get env-users`, `get env-groups`
- [x] Columns, help text, examples, docs for all 9 new commands
- [x] Tests: audit, group permissions, reference, notifications, org levels,
      DPS env usage/cost, plus command-level helper tests

**Extra bug found and fixed:** `DTIAM_SCOPES` and the per-credential `scopes`
field were parsed into config but never passed to the OAuth token manager, so
the documented scope escape hatch was dead code. Now wired through
`config.GetEffectiveScopes` + `common.NewOAuthProviderWithScopes`.

**Deliberately not done:** `analyze` still does not merge policy bindings with
direct permission grants into one effective-access view. Logged in DECISIONS.md
as the obvious follow-up.

### Step 4 — dtctl format alignment (2.3.0) — DONE
- [x] Split oversized verb files into `verb_resource.go` (70 files; largest
      dropped from 954 to 309 lines). Verified behavior-preserving by diffing
      the complete `--help` tree: byte-identical.
- [x] Committed `.golangci.yml` (was missing entirely; `make lint` ran on
      defaults). Lint now clean at 0 issues after fixing 7 deprecated
      `reflect.Ptr` uses.
- [x] `dtiam doctor` — 8 checks, skip-vs-fail distinction, `--offline`,
      non-zero exit for CI
- [x] AGENTS.md, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md, NOTICE
- [x] install.sh / install.ps1 with checksum verification

**Found while building doctor:** running it against a live account exposed four
more commands silently returning empty lists due to wrong response keys. See
Step 4.5.

### Step 4.5 — Live-validated response key fixes
Running `dtiam doctor` against a real account made it possible to compare every
list endpoint's actual response against what the handlers expect. The docs are
not reliable for response shapes.

| Command | Code expected | API actually returns | Paginated |
|---------|---------------|----------------------|-----------|
| `get environments` | `tenants` | `data` | no |
| `get boundaries` | `boundaries` | `content` | yes (page-number) |
| `account limits` | `items` | `results` | yes (page-number) |
| `account subscriptions` | `items` | `data` | no |

- [ ] Fix the four response keys above
- [ ] Add page-number pagination to boundaries and limits
- [ ] Fix `TokenColumns`: ID is `tokenId` not `id`, expiry is `expirationDate`
      not `expiresIn`, scopes is `scope` not `scopes`
- [ ] Fix `TokenHandler.IDField`: `tokenId`, so get/delete by ID resolve
- [ ] Add `data` and `content` to the list-key fallback chain
- [ ] Re-audit every list command against the live account after fixing

### Step 5 — Credential & UX parity (2.4.0)
- [ ] OS keyring for client secrets (currently plaintext YAML)
- [ ] `diff`, `watch`, agent auto-detection

### Step 6 — internal/ -> pkg/ (3.0.0, BREAKING)
- [ ] Promote reusable packages to `pkg/` so dtiam is importable like dtctl

## Next Step

Step 4.5 — live-validated response key fixes.
