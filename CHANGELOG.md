# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [3.0.2] - 2026-10-02

### Fixed

- **The one-line installers could not find any release.** `install.sh`,
  `install.ps1`, the README install commands and releases link, and the
  security-report links pointed at `github.com/jtimothystewart/dtiam`, a
  repository that does not exist. They now point at
  `github.com/timstewart-dynatrace/dtiam`. Verified by installing v3.0.1 with
  both scripts, pinned and latest, with checksum verification passing.
- `.goreleaser.yaml` publishes releases (and would publish the Homebrew formula
  and deb/rpm metadata) under `timstewart-dynatrace` instead of the same
  nonexistent owner.

The Go module path is unchanged (`github.com/jtimothystewart/dtiam`); renaming it
would break library importers and is tracked separately.

## [3.0.1] - 2026-10-02

Every Account Management call was re-checked against the live OpenAPI spec
(`api.dynatrace.com/spec-json`) and against a live account. The previous
revalidation compared against a local spec folder that does not contain the
Account Management API, so these were missed. All fixes were verified live with
throwaway objects that were then deleted.

### Fixed — security-relevant

- **`boundary detach` removed every boundary on the binding, and `boundary attach`
  replaced them all.** The binding was read from the top level of a response that
  wraps it in `policyBindings`, so the current boundaries were always seen as
  empty. Detaching one boundary therefore *widened* access to the full policy.
  Template parameters on the binding were dropped as well. Now only the named
  boundary is added or removed, and parameters are kept. Bindings that exist
  several times for one pair (parameterized policies) are refused rather than
  edited arbitrarily.
- **`delete binding` rewrote every binding at the level** with a read-modify-PUT
  of the whole collection (an operation the API does not document), losing any
  concurrent change. It now uses `DELETE .../bindings/{policy}/{group}`.

### Fixed

- **Every Account Management error printed a blank message.** The API sends
  `"error": true`, and decoding that into a string field discarded the message.
  Errors now show the API's text ("payload.map is not a function", "Expected email
  to be email", ...).
- **`create group` always failed** (HTTP 500): `POST /groups` takes an array.
  This also broke `apply`, `bulk create-groups`, `group clone` and `group setup`.
- **Group policies always came back empty.** `GET .../bindings/groups/{uuid}`
  returns `{policyUuids, bindingsDetails}`, not `policyBindings`, and
  `GroupHandler.GetPolicies` also used a path that 404ed. Affected `group
  bindings`, `get bindings --group`, `describe group`, `export group`, `diff` on
  bindings, the local `analyze` commands, and **`group clone --include-policies`,
  which silently copied no policies**.
- **Lookup by ID failed for groups, environments and platform tokens**, whose
  APIs have no single-item GET. Fixed `get groups ID`, `get environments ID`,
  `describe environment`, `get tokens ID`, `analyze group-permissions`,
  `analyze effective-group` and `export group`.
- **Lookup by name failed for policies and boundaries** (`get policies NAME`,
  `describe boundary NAME`): the API answers a name with 400, not 404.
- **`user list-groups` and `analyze user-permissions` failed**: they called
  `/users/{uid}/groups` and `/users/{uid}`, neither of which exists. Users are
  addressed by email; a UID is now resolved first. `delete user UID` had the
  same problem.
- **Group membership used undocumented endpoints.** `group add-member` /
  `remove-member` and `user remove-from-groups` now use `POST /users/{email}` and
  `DELETE /users/{email}/groups?group-uuid=...`. `group remove-member --user`
  accepts an email or a UID, including a service user's.
- **Service-user group commands did not work.** `service-user list-groups`
  always showed none, and `add-to-group` / `remove-from-group` / `create --groups`
  sent a `groups` field the API ignores. They now go through the user endpoints.
  `service-user update --description` without `--name` no longer fails (PUT
  requires the name).
- **`apply` never updated anything** despite "Create or update" in its help. It
  now creates missing resources, updates changed ones (keeping fields the spec
  omits), and reports matching ones as unchanged, so re-applying is safe.
- **`diff` always reported a policy's statement as changed**: the policy list
  omits `statementQuery`. Both `apply` and `diff` now read the full policy.
- **`get env-groups` always failed**: the API filters groups on
  `partialGroupName`, not `partialString`, and requires it. `--search` (3+
  characters) is now required. Its table showed blank names; columns now match
  the response (`groupName`, `type`).
- **`get env-users` / `get env-groups` were denied** because the account token
  never carried `iam:users:read` / `iam:groups:read`. Environment commands
  (`get apps`, `get schemas`, `get env-users`, `get env-groups`, app and schema
  boundary validation) now request just the scopes they need, so a client
  lacking them fails only that command.
- **`DTIAM_API_URL`, the credential's `api-url`, `environment-url` and
  `environment-token`, and `DTIAM_ENVIRONMENT_TOKEN` were read but never used.**
  The API URL now moves every Account Management call to the given host; the
  environment URL and token feed the environment commands.
- `group clone` reported "Copied N" even when copies failed; it now reports
  how many succeeded.

### Changed

- `resources.BindingHandler.UpdateGroupBindings` now takes `[]string` policy
  UUIDs, matching the API's `{"policyUuids": [...]}` body. The previous signature
  could not produce a valid request.
- `resources.GroupHandler.RemoveMember` accepts an email or a UID.
- Handler errors now wrap the underlying `*client.APIError`, so `errors.As`
  reaches the status code.

## [3.0.0] - 2026-10-01

### Changed — BREAKING

Thirteen packages moved from `internal/` to `pkg/`, making dtiam usable as a Go
library rather than only a binary:

```
internal/{auth,client,config,diagnostic,diff,logging,output,
          prompt,resources,suggest,template,utils,watch}
  ->  pkg/{...}
```

`internal/cli` and `internal/commands` **stay internal** — they are the cobra
wiring, and exposing them would invite callers to depend on command plumbing
rather than on the API client and resource handlers. The move was possible
because none of the thirteen referenced either one.

**Impact:** only code importing these paths. The CLI is unaffected: the complete
`--help` tree for every command and subcommand is byte-identical before and after
(4,110 lines diffed). An external module importing `pkg/auth`, `pkg/client`,
`pkg/config`, `pkg/diff`, `pkg/output`, `pkg/resources`, `pkg/version` and
`pkg/watch` was built and run to confirm the packages resolve from outside the
module.

**Migration:** replace `internal/` with `pkg/` in your imports.

```go
// before
import "github.com/jtimothystewart/dtiam/internal/resources"
// after
import "github.com/jtimothystewart/dtiam/pkg/resources"
```

## [2.6.0] - 2026-10-01

### Added

- `dtiam diff -f FILE` — shows what `apply` would change, without changing
  anything. Fetches each live resource and compares field by field. Only fields
  present in the file are compared, since the API returns server-managed fields
  (`uuid`, `createdAt`, `owner`) that a spec never mentions. Lists compare
  order-insensitively, and `5` from YAML compares equal to `5.0` from the API —
  otherwise every numeric and list field would look modified on every run.
  Exits 1 on drift so it can gate a pipeline; `--exit-zero` disables that.
- **Agent auto-detection.** Running under a coding agent (Claude Code, Cursor,
  Copilot, Aider and others) now implies `--plain`, because an agent has no
  terminal to answer a confirmation prompt at and no use for ANSI colors. An
  explicit `--plain=false` still wins, and `DTIAM_NO_AGENT_DETECT=1` opts out.
  `-v` explains which variable triggered it. A specific agent is reported in
  preference to the generic `AI_AGENT` fallback.
- `--watch` / `-w` on `get groups`, `get users`, `get policies`, and
  `get bindings`, with `--watch-interval`. Reprints only when the result actually
  changes: the fingerprint sorts items first, so the API's unstable list order is
  not mistaken for a change. A failed poll is reported and the watch continues,
  rather than a transient API error ending a session someone left running.
  Refused with `--plain`, where the output would be an unparseable JSON stream.
- `cli.ErrSilentExit` — lets a command exit non-zero without an `Error:` line,
  for cases like `diff` where the non-zero exit is a result rather than a failure.

### Changed

- `internal/diff` and `internal/watch` are standalone packages, so the comparison
  and polling logic is testable without a command or a network.

## [2.5.0] - 2026-10-01

### Added

- Client secrets are now stored in the **OS keyring** when one is available.
  `dtiam config set-credentials` writes the secret to the keyring (service name
  `dtiam`) and records only a reference in the config file. Existing plaintext
  configs keep working unchanged — the stored value is resolved either way, so
  there is no forced migration.
- `dtiam config migrate-secrets` moves existing plaintext secrets into the
  keyring. Idempotent, and supports `--dry-run`.
- `dtiam config keyring-status` shows, per credential, whether its secret lives
  in the keyring or the config file — so you can confirm no plaintext remains.
- `--require-keyring` on `set-credentials` fails rather than ever writing a
  plaintext secret; `--no-keyring` forces file storage. `DTIAM_DISABLE_KEYRING`
  opts out entirely, which also avoids a keyring probe on headless hosts.
- `dtiam doctor` gained a **secret storage** check that reports plaintext secrets
  and names the command that fixes them.
- `dtiam config delete-credentials` now removes the keyring entry too, rather
  than leaving an orphaned secret behind, and supports `--dry-run`.

### Changed

- `config delete-credentials` and `set-credentials` route their output through
  the printer instead of `fmt.Printf`, so `--plain` and `-o json` behave.

### Security

- Plaintext credential storage is now a fallback rather than the only option.
  Where a secret is stored is always reported, never silent. See SECURITY.md.

## [2.4.0] - 2026-10-01

### Fixed

Validated every list endpoint against a live account. **Six commands were
silently returning empty results** because each handler read a response key the
API does not send. An unmatched key yields an empty slice rather than an error,
so these failed quietly and the test fixtures — written from documentation that
does not match the API — passed.

| Command | Read | API actually returns |
|---------|------|----------------------|
| `get environments` | `tenants` | `data` |
| `get boundaries` | `boundaries` | `content` (a page envelope) |
| `account limits` | `items` | `results` (a page envelope) |
| `account subscriptions` | `items` | `data` |
| `service-user list` | `items` | `results` |
| `get tokens` | `items` | `results` |

Root cause: `EnvironmentHandler`, `LimitsHandler`, and `SubscriptionHandler` each
shadowed `BaseHandler.List` and `extractList` with their own hardcoded key lists,
so the embedded `ListKey` was never consulted. Those overrides are now deleted
and key resolution lives in one place.

Also fixed, same root cause:

- `get boundaries` and `account limits` are paginated (the docs do not say so),
  and returned only the first page.
- `account check-capacity` reported **every limit as "not found"**, and computed
  0/0 for capacity. The limit's identity field is `limitType`, not `name`, and
  its values are `currentValue`/`limitValue`, not `current`/`max`. A capacity
  check that silently answers "no capacity" is worse than an error, since it
  reads as a real answer.
- `get tokens` showed a blank ID column and could not resolve a token by ID:
  the field is `tokenId`, not `id`. Expiry is `expirationDate`, not `expiresIn`;
  scopes is `scope`, not `scopes`.
- `get environments` showed blank STATE/TRIAL columns; the fields are `active`
  and `url`.
- `account limits --summary` renamed fields to `name`/`current`/`max` when
  building its output, so every column rendered blank.
- `get audit-logs` listed `eventOutcome` as a default column, but the API's
  default projection does not include it; it now sits in the wide set with the
  other `--add-fields` values.
- `account limits --summary` wrote status lines to stdout with `fmt.Printf`,
  which `command-standards.md` forbids; they now go to stderr.

### Added

- `internal/resources/response_shapes_test.go` — pins the live response shape and
  declared `ListKey`/`IDField`/pagination for every list endpoint, so this bug
  class cannot silently return.
- Single-resource responses are recognized by the handler's own identity fields
  and wrapped, in both the paginated and unpaginated paths.

### Changed

- `client.BoundaryPagination()` and `client.AccountLimitPagination()`.
- Test helpers now build handlers through their real constructors instead of
  hand-rolling a `BaseHandler`, which is what allowed a wrong `ListKey` in
  production code to pass its own tests.

## [2.3.0] - 2026-10-01

### Added

- `dtiam doctor` — diagnoses configuration, credentials, scopes, token
  retrieval, and API connectivity. Checks run cheapest-first and later checks
  report `skip` rather than `fail` when an earlier one makes them meaningless, so
  the output distinguishes "could not test" from "broken". Exits non-zero on any
  failure, so it works as a CI readiness gate. `--offline` skips the network
  checks.
- `.golangci.yml` — the linter config was never committed, so `make lint` ran
  with defaults. Now pinned, with the exclusions annotated. Lint is clean.
- `AGENTS.md` — agent-facing guidance: the `--plain` stream contract, the
  surprises worth knowing (two coexisting permission models, two different user
  APIs, partial audit results returning HTTP 200), and safety rules.
- `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `NOTICE`.
  SECURITY.md documents that credentials are stored in **plaintext** and that
  `-v` dumps Authorization headers.
- `install.sh` and `install.ps1` — platform detection, checksum verification,
  and PATH guidance.

### Fixed

- Replaced the deprecated `reflect.Ptr` with `reflect.Pointer` in the struct
  printer (7 occurrences), the only issues the newly-pinned linter reported.
- Corrected the README install instructions, which pointed at a `GO-dtiam`
  repository that does not exist.

### Changed

- Split the oversized per-verb command files into dtctl-style
  `verb_resource.go` files. The largest file drops from 954 lines
  (`analyze.go`, 7 subcommands) to 309, across 70 files.
  Behavior-preserving: the full `--help` tree for every command and
  subcommand is byte-identical before and after.

## [2.2.0] - 2026-10-01

### Added

Six Dynatrace API groups that were documented but unused:

- **Account audit logs** — `dtiam get audit-logs`. Who changed what in the
  account, with `--start`/`--end`, `--event-type`, `--user`, a raw `--filter`
  escape hatch, and `--scan-limit-gb`/`--result-limit-mb` to bound expensive
  queries. The API returns partial results with warnings rather than an error, so
  warnings are surfaced on stderr instead of being dropped.
- **Reference data** — `dtiam get available-permissions`. The authoritative list
  of grantable permission names, for validating a grant before attempting it.
- **Permission management** — `dtiam group permissions`,
  `group grant-permission`, `group revoke-permission`. These are role-style
  grants that coexist with IAM policies; a group's effective access is the union
  of its policy bindings and these direct grants, so `group bindings` alone
  understates what a group can do.
- **Account notifications** — `dtiam account notifications`, with type and
  severity filters validated locally.
- **Per-environment subscription usage and cost** —
  `dtiam account environment-usage` and `account environment-cost`. The cost
  endpoint lives on the v3 Subscription API while listing, usage and forecast
  remain on v2; dtiam handles the version difference internally.
- **Environment-level Platform IAM** — `dtiam get env-users`,
  `get env-groups`. A different API from `get users`/`get groups`: served from
  the environment rather than from `api.dynatrace.com`, reporting who is visible
  at an organizational level.

### Fixed

- `DTIAM_SCOPES` and the per-credential `scopes` config field had no effect. The
  value was parsed into the config struct but never passed to the OAuth token
  manager, so the documented escape hatch for scope problems was dead code.

### Changed

- `ReferenceHandler`, `NotificationHandler`, and `SubscriptionHandler` expose
  overridable base URLs. These endpoints sit outside the client's account-scoped
  base URL, and without the override their tests could only have run against
  the live API.

## [2.1.0] - 2026-10-01

### Added

- Automatic pagination for paginated Account Management endpoints. `List` now
  follows every page to completion, so callers always receive the full
  collection. Two paging styles are supported, matching the documented API
  behavior: cursor-based (`page-key`/`nextPageKey`, used by service users) and
  1-based page numbers (`page`/`size` plus `total`, used by platform tokens).
- `client.PaginationConfig` with presets `ServiceUserPagination()`,
  `PlatformTokenPagination()`, and `OrganizationalLevelPagination()`.

### Fixed

- `dtiam get service-users` returned an empty list against the live API. The
  service user endpoint responds with `{results, nextPageKey, totalCount}`, but
  the handler read `items`. The existing tests mocked the `items` shape, so they
  passed while the command was broken.
- `dtiam get platform-tokens` had the same defect: the endpoint responds with
  `{pageSize, pageNumber, total, results}` and the handler read `items`.
- Both endpoints also silently truncated to a single page, since no paging
  parameters were ever sent.

### Changed

- The list-key fallback chain now includes `results` alongside `items`, so
  paginated response shapes resolve even on handlers without an explicit key.

## [2.0.3] - 2026-10-01

### Fixed

- OAuth2 token requests now include the `account-uac-read` scope, so
  `dtiam account subscriptions` and `dtiam account forecast` no longer fail
  with HTTP 403.
- OAuth2 token requests now include the `platform-token:tokens:manage` scope,
  so `dtiam get platform-tokens` and platform token create/delete no longer
  fail with HTTP 403. The required scope was documented in the token handler
  but never requested.
- OAuth2 token requests now include the `account-audit-logs-read` scope, in
  preparation for account audit log support.
- Synchronized the version string across `pkg/version/version.go`,
  `.claude/settings.json`, `.claude/CLAUDE.md`, and `.claude/rules/core.md`,
  which had drifted to 2.0.0 while the release was 2.0.2.

### Changed

- `defaultScopes` is now derived from the exported `auth.DefaultScopeList`,
  with each scope annotated with the API group that requires it. A regression
  test asserts every API group dtiam calls has its scope requested.

## [2.0.2] - 2026-04-16

### Added

- **Policies-with-boundaries doc** — `docs/POLICIES_WITH_BOUNDARIES.md` ported from Python-dtiam; documents the six rules for resolving effective permissions when boundaries apply, with seven worked examples and a dtiam-specific command table
- **Command hierarchy diagram** — `images/04-command-hierarchy_930x500.svg` embedded in `docs/COMMANDS.md`; visualizes the four tiers (core CRUD verbs, resource-scoped commands, specialized workflows, resource vocabulary) with a `MARKDOWN_TABLE_ALTERNATIVE` fallback
- **README/ARCHITECTURE cross-links** — both now reference `POLICIES_WITH_BOUNDARIES.md`

## [2.0.1] - 2026-04-16

### Added

- **Architecture diagrams** — three SVG diagrams (`images/01-architecture_930x500.svg`, `02-two-tier-bindings_930x500.svg`, `03-oauth2-flow_930x500.svg`) embedded in `README.md`, `docs/ARCHITECTURE.md`, and `docs/QUICK_START.md`, with `MARKDOWN_TABLE_ALTERNATIVE` text fallbacks for environments that strip images

## [2.0.0] - 2026-04-08

### Added

- **Template engine** — `internal/template/` package with Go `text/template` renderer, XDG-based template store, and 5 built-in templates (group-team, policy-readonly, policy-admin, binding-simple, boundary-mz)
- **Template commands** — `template list/show/render/apply/save/delete/path` for managing and using resource templates
- **Apply command** — `dtiam apply -f resource.yaml` declarative resource creation with auto-detect kind, `--set` template variables, `--dry-run`, and multi-document YAML support
- **Bulk groups+policies** — `bulk create-groups-with-policies --file FILE` creates groups and binds policies from CSV/YAML/JSON in one step
- **Export template enhancement** — `export policy --as-template` now uses Go template syntax (`{{.name}}`) compatible with `dtiam template apply`

## [1.5.0] - 2026-04-08

### Added

- **Group clone** — `group clone SOURCE --name NEW [--include-members] [--include-policies]` clones a group with optional members and policy bindings
- **App boundaries** — `boundary create-app-boundary NAME --app-ids ...` creates boundaries scoped to specific Dynatrace apps with optional validation
- **Schema boundaries** — `boundary create-schema-boundary NAME --schema-ids ...` creates boundaries scoped to specific Settings 2.0 schemas with optional validation
- **Group setup** — `group setup --name NAME --policies-file FILE` one-step group provisioning from YAML/JSON policy definitions
- **Parameterized policies** — `create binding --param key=value` passes bind parameters for `${bindParam:name}` substitution in policy statements

## [1.4.0] - 2026-04-08

### Added

- **Account capabilities** — `account capabilities [SUBSCRIPTION]` command to list capability flags from subscriptions
- **Per-resource exports** — `export environments`, `export users`, `export bindings`, `export boundaries`, `export service-users` subcommands with `--detailed` enrichment
- **User info** — `user info IDENTIFIER` command as alias for `describe user` (Python CLI parity)

## [1.3.0] - 2026-04-08

### Added

- **Platform tokens** — `get tokens`, `create token`, `delete token` commands for IAM platform token management
- **App Engine Registry** — `get apps` command with `--environment` flag for listing apps
- **Settings schemas** — `get schemas` command with `--environment` and `--name` filter for Settings 2.0 schemas
- **Resty HTTP client** — replaced stdlib `net/http` with `go-resty/resty/v2` for built-in retry, debug mode, and request hooks
- **Viper config** — automatic `DTIAM_*` env var binding via `spf13/viper`, XDG paths via `adrg/xdg`
- **Structured logging** — `internal/logging/` package with `sirupsen/logrus` (verbosity levels, HTTP request logging)
- **Diagnostic errors** — `internal/diagnostic/` package with exit codes (auth=3, not-found=4, forbidden=5) and troubleshooting suggestions
- **Command suggestions** — `internal/suggest/` Levenshtein engine for typo correction
- **Struct-tag output** — `internal/output/structprinter.go` reads `table` struct tags via reflection for type-safe column rendering
- **Typed resource structs** — `internal/resources/types.go` with `json` + `table` tags for all 12 resource types
- **Credential enhancements** — `api-url`, `scopes`, `environment-url`, `environment-token` fields per credential
- **New env vars** — `DTIAM_API_URL`, `DTIAM_SCOPES`, `DTIAM_ENVIRONMENT_URL`, `DTIAM_ENVIRONMENT_TOKEN`, `DTIAM_OUTPUT`, `DTIAM_VERBOSE`
- **Comprehensive test coverage** — 737 tests across 26 packages (resource handlers, commands, output, auth, prompt, CLI state)

### Fixed

- **Bulk flag conflict** — removed duplicate `-f` shorthand on `remove-users-from-group` (was `--file` and `--force` both using `-f`)

## [1.2.1] - 2026-04-07

### Fixed

- **Dead code removal** — removed unused `detailColumns()` function and stale import in `describe` package
- **Loop simplification** — simplified append loop to `append(fields, result.Permissions...)` in `analyze` package
- **Boundary cleanup** — removed unused `columns` variable and dead loop in `boundary` package
- **Error handling** — check all `SetContext()`/`UseContext()` error returns in config tests

## [1.2.0] - 2026-04-06

### Added

- **Centralized confirmation prompts** — new `internal/prompt` package with `Confirm()` and `ConfirmDelete()` functions; replaces inline `bufio.NewReader`/`fmt.Scanln` implementations across all destructive commands
- **Safe type assertion helpers** — new `internal/utils/safemap.go` with `StringFrom`, `IntFrom`, `BoolFrom`, `SliceFrom`, `MapFrom`, `StringSliceFrom` for safely extracting values from `map[string]any` API responses without panic risk
- **API URL constants** — centralized all Dynatrace API base URLs and paths in `internal/client/urls.go`; eliminates hardcoded URLs scattered across 6 resource handlers
- **Example help text** — added Cobra `Example` fields to all ~50 CLI subcommands with real-world usage patterns including `--dry-run`, `--force`, `--plain`, and `-o json` examples
- **Command standards** — new `.claude/rules/command-standards.md` defining mandatory patterns for all commands (output through printer, `--plain` behavior, dry-run, confirmation, error handling)
- **Phase planning** — v2.0.0 refactor phase docs in `.claude/phases/`
- **Tests** — unit tests for safemap utilities and URL constants

### Changed

- **`--plain` mode JSON override** — `--plain` flag now forces JSON output when table/wide format is selected, ensuring machine-consumable output for AI agents and scripts
- **Standardized confirmation flow** — all destructive operations now use `prompt.ConfirmDelete()` with consistent `--force` flag behavior instead of ad-hoc implementations
- **Consistent dry-run output** — all dry-run messages use `printer.PrintWarning()` instead of raw `fmt.Printf`
- **Bulk force flag** — standardized to lowercase `-f` (was `-F`) for consistency with other commands
- **Client consolidation** — merged duplicate `tokenProviderAdapter` (3 copies) and `createClient()` (2 copies) into single implementations in `common/client.go`
- **Safe type assertions** — replaced ~40 unsafe bare `.(string)` assertions with `StringFrom()` helpers, preventing panics on unexpected API response shapes

### Fixed

- **Panic prevention** — eliminated potential panics from unguarded type assertions on API response maps throughout all resource handlers
- **Bulk confirmation** — `bulk remove-users-from-group` was using `fmt.Scanln` instead of the centralized confirmation prompt

## [1.1.1] - 2025-01-21

### Fixed

- **API URL correction** — fixed policies, bindings, and boundaries handlers to use correct `/repo/` endpoint path (`https://api.dynatrace.com/iam/v1/repo/...` instead of `/accounts/{uuid}/repo/...`); affected `policies.go`, `bindings.go`, `boundaries.go`
- **Name resolution fallback** — fixed `GetOrResolve` in `handler.go` to properly fall back to list search when direct GET returns 404; now searches by UUID fields (`uuid`, `uid`, `id`) then by name; fixes `describe group` failures on valid groups

## [1.1.0] - 2025-01-21

### Added

- **Bulk operations** — process multiple resources from CSV/YAML/JSON files:
  - `bulk add-users-to-group` / `bulk remove-users-from-group` — manage group membership at scale
  - `bulk create-groups` — create multiple groups from file
  - `bulk create-bindings` — create policy bindings from file
  - `bulk export-group-members` — export group membership to file
- **Export commands** — backup and migration support:
  - `export all` — export all IAM resources (groups, policies, bindings, boundaries) to files
  - `export group` — export single group with members and policy bindings
  - `export policy` — export single policy, optionally as a reusable template
- **Permissions analysis** — comprehensive IAM analysis tools:
  - `analyze user-permissions` / `analyze group-permissions` — calculate effective permissions from policy bindings
  - `analyze permissions-matrix` — generate cross-reference matrix of permissions by policy or group
  - `analyze policy` — analyze a policy's permission statements and binding usage
  - `analyze least-privilege` — identify policies granting excessive permissions
  - `analyze effective-user` / `analyze effective-group` — query Dynatrace resolution API for effective permissions
- **Permissions utilities** — new `internal/utils/permissions.go` with `ParseStatementQuery()`, `PermissionsCalculator`, `PermissionsMatrix`, and `EffectivePermissionsAPI`
- **Account enhancements** — `account check-capacity` for pre-flight capacity checks; `account limits --summary` for usage percentages
- **Validation** — comprehensive validation script (`scripts/validate.sh`) and `make validate` target
- **Unit tests** — added tests for permissions parsing, output formatting, configuration management, and HTTP client error handling

### Changed

- **Boundary query syntax** — updated to modern Dynatrace format: `environment:management-zone IN ("Zone")` with `storage:dt.security_context` and `settings:dt.security_context` queries (replaces legacy `managementZone.name = "Zone"`)
- **Pre-push checklist** — enhanced CLAUDE.md with mandatory version management checklist

### Documentation

- Full documentation for bulk, export, and analyze commands in COMMANDS.md
- Updated README.md with new command groups and resources table

## [1.0.0] - 2025-01-20

### Added

- **Initial release** — Go implementation of dtiam CLI (converted from Python)
- **kubectl-style commands** — `get`, `describe`, `create`, `delete` with consistent verb-noun syntax
- **Multi-context configuration** — named contexts with separate credentials, XDG Base Directory support
- **Dual authentication** — OAuth2 with automatic token refresh (recommended) and static bearer token
- **Resource handlers** — full CRUD for groups, users, service users, policies, bindings, boundaries, environments, limits, and subscriptions
- **Output formats** — table, wide, JSON, YAML, CSV, and plain modes
- **Safety features** — dry-run mode, confirmation prompts for destructive operations, verbose debugging
- **HTTP client** — exponential backoff retry logic with rate limit handling (429 responses)
- **Name resolution** — user-friendly name-to-UUID resolution for all resource identifiers
- **Cross-platform** — single binary for Linux, macOS (Intel + Apple Silicon), and Windows
- **Build system** — Makefile with build/test/lint targets and goreleaser for multi-platform releases

### Documentation

- CLAUDE.md with development workflow standards and mandatory pre-push checklist
- README.md with installation, authentication, and usage guide
- docs/QUICK_START.md, docs/COMMANDS.md, docs/ARCHITECTURE.md, docs/API_REFERENCE.md

[Unreleased]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/timstewart-dynatrace/GO-dtiam/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/timstewart-dynatrace/GO-dtiam/releases/tag/v1.0.0
