# PHASE 09 — API Revalidation & dtctl Format Alignment

**Status:** done
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

### Step 4.5 — Live-validated response key fixes (2.4.0) — DONE

Running `dtiam doctor` against a real account made it possible to compare every
list endpoint's actual response against what the handlers expect. **The docs are
not reliable for response shapes.** Six commands were silently returning empty.

| Command | Read | API returns | Before | After |
|---------|------|-------------|--------|-------|
| `get environments` | `tenants` | `data` | 0 | 5 |
| `get boundaries` | `boundaries` | `content` | 0 | 37 |
| `account limits` | `items` | `results` | 0 | 8 |
| `account subscriptions` | `items` | `data` | 0 | 10 |
| `service-user list` | `items` | `results` | 0 | 22 |
| `get tokens` | `items` | `results` | 0 | 105 |

- [x] Fixed all six response keys
- [x] Added page-number pagination to boundaries and limits (undocumented)
- [x] Deleted the three `List`/`extractList` overrides that made the `ListKey`
      field dead — the root cause
- [x] Fixed `account check-capacity`, which reported every limit as not found
      and computed 0/0 capacity (`limitType`, `currentValue`, `limitValue`)
- [x] Fixed `TokenColumns` (`tokenId`/`expirationDate`/`scope`) and
      `TokenHandler.IDField`
- [x] Fixed `EnvironmentColumns` (`active`/`url`) and `LimitColumns`
- [x] Fixed `account limits --summary`, which renamed fields and printed status
      to stdout
- [x] Fixed `AuditColumns` to match the API's default projection
- [x] Single-resource wrapping generalized to the handler's own ID/name fields
- [x] `response_shapes_test.go` pins every live shape, ListKey, IDField and
      pagination setting
- [x] Test helpers now use production constructors
- [x] Re-audited live: all 12 list commands return data; doctor 8/8 ok

### Step 5 — Credential storage (2.5.0) — DONE
- [x] OS keyring for client secrets, with a reported plaintext fallback
- [x] `config migrate-secrets`, `config keyring-status`
- [x] `--require-keyring` / `--no-keyring` / `DTIAM_DISABLE_KEYRING`
- [x] `doctor` secret-storage check naming the fix
- [x] `delete-credentials` cleans up the keyring entry
- [x] SECURITY.md rewritten; plaintext is now a documented fallback, not the default

### Step 5b — dtctl UX parity (2.6.0) — DONE
- [x] `diff` — previews what `apply` would change; exits 1 on drift as a CI gate
- [x] `--watch` / `-w` on get groups/users/policies/bindings
- [x] Agent auto-detection implying `--plain`, with opt-out and `-v` explanation
- [x] `cli.ErrSilentExit` for a non-zero exit that is a result, not a failure
- [x] `pkg/diff` and `pkg/watch` (originally `internal/`) as standalone, testable packages
- [x] Live-verified: diff detects field drift and creates; watch renders once and
      does not re-render on unchanged polls; clean SIGINT shutdown

### Step 6 — internal/ -> pkg/ (3.0.0, BREAKING) — DONE
- [x] Moved 13 packages to `pkg/`: auth, client, config, diagnostic, diff,
      logging, output, prompt, resources, suggest, template, utils, watch
- [x] Kept `internal/cli` and `internal/commands` internal -- they are cobra
      wiring, and exposing them would invite dependence on command plumbing
- [x] Verified behavior-preserving: the complete `--help` tree is byte-identical
      before and after (4,110 lines diffed)
- [x] Verified the point of the move: an external module importing pkg/auth,
      pkg/client, pkg/config, pkg/diff, pkg/output, pkg/resources, pkg/version
      and pkg/watch was built and run successfully
- [x] All docs, rule files and the architecture tree updated
- [x] README gained a library-usage section

## Outcome

All six steps complete. Released 2.0.3 -> 3.0.0 across seven versions.

**The headline finding was not what the request anticipated.** Every API dtiam
called was still current -- nothing deprecated, nothing removed. But validating
against a live account (possible once `doctor` existed) revealed **six commands
silently returning empty lists** because the handlers read response keys the API
does not send, plus `account check-capacity` reporting every limit as not found.
The published documentation does not match the API for several endpoints, and the
test fixtures had been written from that documentation -- so they passed while the
commands were broken.

Every response shape is now pinned in `pkg/resources/response_shapes_test.go`
and tabulated in `.claude/architecture.md`.

## Follow-ups not done

- `analyze` does not merge policy bindings with direct permission grants into one
  effective-access view. Logged in DECISIONS.md.
- No live-account smoke test in CI; the shape fixtures are a snapshot of one
  account on 2026-10-01 and will need updating when the API legitimately changes.
- `dtctl`'s `exec`, `wait`, `safety` and `skills` packages have no dtiam
  counterpart. None had a clear IAM use case.
