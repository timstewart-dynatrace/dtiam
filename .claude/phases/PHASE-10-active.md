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

## Step 2 — 3.1.0 API currency — NEXT

- [ ] Notifications: `POST /v1/.../notifications` -> `GET /v2/.../notifications` (deprecated, removal 2027-01-11)
- [ ] Usage: `/sub/v2/.../environments/usage` -> `/sub/v3/...` (deprecated, removal 2027-01-11)
- [ ] Remove unused `PolicyHandler.Validate` / `ValidateUpdate` (validation endpoints deprecated)
- [ ] Platform token `PUT /{id}/expiration-date` and `PUT /{id}/status`
- [ ] `group update` command (PUT /groups/{uuid} now in the handler)
- [ ] Config subcommand to set `api-url` / `environment-url` / `environment-token`

## Step 3 — 3.2.0 dtctl parity (proposed, needs approval)

Context safety levels; `--agent` envelope; `whoami` / `auth can-i`; `commands`
catalog; `edit`; integration test suite behind `//go:build integration`.

## Step 4 — 3.3.0 new surfaces (proposed, needs approval)

WIF trust policies (Preview), IP allowlist, policy-level limits, env service users.

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
- `account forecast` reports "subscription not found" when the API says the
  subscription has no budget.
