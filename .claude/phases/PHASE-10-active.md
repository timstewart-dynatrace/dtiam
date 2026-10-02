# PHASE 10 — API Currency, Correctness & dtctl Parity

**Status:** active
**Started:** 2026-10-02

## Goal

Re-validate dtiam against the live Account Management spec and a live account,
fix what is broken, move off deprecated endpoints before their 2027-01-11
removal, and close the most useful gaps relative to dtctl.

## Reference

- Live spec: `https://api.dynatrace.com/spec-json` (operations carry
  `x-token-scopes` and `deprecated`). The local `ARCHIVE/.../api-spec/` folder
  has no Account Management spec -- do not validate against it.
- Live behavior that contradicts or extends the spec is recorded in
  `docs/dev/API_BEHAVIORS.md`.

## Step 1 — 3.0.1 correctness fixes — DONE (branch `fix/api-revalidation-3.0.1`)

Scope approved: all root causes found, plus apply and config wiring. Every fix
verified live; write paths verified on throwaway `dtiam-probe-*` objects, all
deleted afterwards.

- [x] R1 no single-item GET (groups, environments, tokens) -> `NoSingleGet`
- [x] R2 group bindings shape `{policyUuids, bindingsDetails}` + wrong path
- [x] R3 users addressed by email; UID resolved (incl. service users)
- [x] R4 boundary attach/detach clobbered boundaries (security-relevant)
- [x] R5 bindings create/delete via per-pair endpoints
- [x] R6 membership via `/users/{email}` endpoints; DELETE uses query params
- [x] Service-user membership, create --groups, update without --name
- [x] `apply` create-or-update; `diff`/`apply` read full policy
- [x] `create group` (POST takes an array)
- [x] Blank error messages (boolean `error` field)
- [x] Name lookup on 400 for policies/boundaries
- [x] env-groups `partialGroupName` + columns; per-command environment scopes
- [x] `DTIAM_API_URL` / `api-url`, `environment-url`, `environment-token` wired
- [x] Tests rewritten to the live contract; new regression tests

## Step 2 — 3.2.0 API currency — DONE

- [x] Notifications -> `GET /v2/.../notifications` (paged; new filters/types; column fix)
- [x] Usage -> `/sub/v3/...` (paged); cost paged; flattened rows; ACTIVE default
- [x] `PolicyHandler.Validate` / `ValidateUpdate` marked Deprecated (remove in 4.0)
- [x] `token activate|deactivate|set-expiration`
- [x] `group update`
- [x] `config set-credentials --api-url --environment-url --environment-token` (keyring)
- [x] Found and fixed: `migrate-secrets` left plaintext secrets in the file
- [x] Found and fixed: `create token` request shape (never worked)
- [x] Verified live: notifications (91 records, filters), usage v3 == v2 totals,
      token lifecycle and group update on throwaway objects

## Step 3 — 3.3.0 dtctl parity, part 1 — DONE

- [x] Context safety levels (readonly / no-delete / readwrite), central operations
      table, enforcement in PersistentPreRunE, read-only scopes for readonly
- [x] `--agent` / `-A` envelope with error codes; stray stdout captured
- [x] `auth whoami`, `auth can-i`
- [x] `commands` catalog
- [x] Found and fixed: effective permissions truncated at 100; unknown
      subcommands exited 0

## Step 3b — 3.4.0 dtctl parity, part 2 (approved scope, not started)

- [ ] `edit group|policy|boundary NAME` via $EDITOR, diff, update
- [ ] Integration test suite behind `//go:build integration` + `make test-integration`

## Step 4 — 3.5.0 new surfaces (proposed, needs approval)

WIF trust policies (Preview), IP allowlist, policy-level limits, env service users.

## Step 1b — 3.0.2 installer repo paths — DONE

- [x] install.sh / install.ps1 / README / SECURITY / CODE_OF_CONDUCT / CONTRIBUTING
      / .goreleaser.yaml point at `timstewart-dynatrace/dtiam`
- [x] Both installers verified against the v3.0.1 release (pinned and latest)

## Step 1c — 3.1.0 module path & GoReleaser — DONE

- [x] Module path -> `github.com/timstewart-dynatrace/dtiam/v3` (99 files)
- [x] GoReleaser archive names match the installers; Windows tar.gz
- [x] Homebrew section removed (tap repo does not exist)
- [x] Verified: `goreleaser check`, local snapshot, external `/v3` import

## Known issues found, not yet fixed

- `export group|policy|all|...` and `bulk export-group-members` define a local
  `-o` meaning "output file", shadowing the global `-o` output format.
  `export group X -o json` writes a file named `json`.
- `analyze permissions-matrix` emits junk permissions such as `ALLOW:THE`,
  `ALLOW:GEN3` -- the statement parser picks up words outside statements.
- `.claude/rules/go.md` documents `--name` / `--email` filters on `get` commands
  that do not exist.
- `group clone` does not copy binding parameters.
- `get schemas` uses `{env}.live.dynatrace.com/api/v2` (classic API); with an
  OAuth token the platform path `{env}.apps.dynatrace.com/platform/classic/environment-api/v2`
  is likely required. Not verifiable with the current OAuth client (no settings scope).
- Handler tests on absolute URLs reach the real api.dynatrace.com if they forget
  to override the base URL (two usage tests did, failing on 401). Consider a
  test transport that refuses non-loopback hosts.
- `create token` could default `--user` to the caller (the access token's `sub`)
  since the API only allows that owner anyway.
- 151 direct `fmt.Print` calls to stdout in 21 command files bypass the printer
  (violates command-standards). Agent mode captures them; route them through the
  printer.
- `auth can-i` cannot evaluate Account Management access (group permissions).
- `account forecast` reports "subscription not found" when the API says the
  subscription has no budget.
