# Decisions

## 2025-01-01 — kubectl-style CLI Design
**Chosen:** Verb-noun command structure modeled after kubectl and go-dtctl-main
**Alternatives:** POSIX-style flags only, interactive TUI, REST-wrapper approach
**Why:** Target users are Dynatrace admins familiar with kubectl. Verb-noun is discoverable (`get groups`, `delete policy`) and scriptable. go-dtctl-main provided a proven reference implementation.
**Trade-offs:** More complex command registration than flat flag-based CLI
**Revisit if:** Target audience shifts away from kubectl-familiar users

---

## 2025-01-01 — BaseHandler Pattern for Resources
**Chosen:** Generic `BaseHandler` struct that all resource handlers embed, providing default CRUD via HTTP
**Alternatives:** Individual handler implementations, code generation from OpenAPI spec
**Why:** Dynatrace IAM API follows consistent patterns (list/get/create/delete). BaseHandler eliminates 80% of boilerplate while allowing per-resource overrides.
**Trade-offs:** Less type safety than generated clients; handlers return `map[string]any` instead of typed structs
**Revisit if:** API diverges significantly between resources, or typed response structs are needed

---

## 2025-01-01 — Resty over stdlib HTTP
**Chosen:** `go-resty/resty/v2` for HTTP client
**Alternatives:** Standard `net/http`, `hashicorp/go-retryablehttp`
**Why:** Built-in retry with backoff, request/response middleware, cleaner API for REST operations. Reduces boilerplate for auth header injection, error handling, and JSON marshaling.
**Trade-offs:** External dependency for something stdlib can do
**Revisit if:** Resty maintenance stalls or dependency footprint becomes a concern

---

## 2025-01-01 — Unified Printer Abstraction
**Chosen:** Single `Printer` type that handles table, wide, JSON, YAML, CSV output with `--plain` mode for machine consumption
**Alternatives:** Per-format output functions, template-based rendering
**Why:** Every command needs consistent output formatting. `--plain` mode forces JSON and strips colors so AI agents and scripts get clean structured data. Centralizing this prevents format drift across 30+ commands.
**Trade-offs:** Printer is a relatively large interface; all commands coupled to it
**Revisit if:** Output requirements diverge significantly between command categories

---

## 2025-01-01 — OAuth2 with stdlib (no external dep)
**Chosen:** Custom OAuth2 implementation using `net/http` and `net/url`
**Alternatives:** `golang.org/x/oauth2`, third-party OAuth libraries
**Why:** Dynatrace OAuth2 flow is straightforward client_credentials grant. Custom implementation avoids pulling in `golang.org/x/oauth2` and its transitive dependencies for a simple token exchange.
**Trade-offs:** Must maintain token refresh logic manually
**Revisit if:** Need to support additional OAuth2 flows (authorization code, PKCE)

---

## 2025-01-01 — Client-side Filtering
**Chosen:** Fetch full resource list from API, filter client-side with `--name`/`--email`
**Alternatives:** Server-side filtering (API query params), GraphQL
**Why:** Dynatrace IAM API does not support server-side filtering for most resources. Client-side substring matching provides consistent UX across all resource types.
**Trade-offs:** Fetches more data than needed; won't scale to very large accounts
**Revisit if:** API adds server-side filtering, or accounts exceed 10k resources

---

## 2026-04-08 — Modular .claude/ Rule Structure
**Chosen:** Break monolithic root CLAUDE.md into `.claude/CLAUDE.md` + modular rule files under `.claude/rules/`
**Alternatives:** Keep single CLAUDE.md, use only `@` includes from root
**Why:** 500+ line CLAUDE.md mixed workflow rules, code patterns, API docs, and architecture. Modular files align with PROJECT-TEMPLATES standard, improve maintainability, and allow rules to be updated independently.
**Trade-offs:** More files to maintain; must keep root CLAUDE.md in sync as pointer
**Revisit if:** Claude Code changes how it loads instructions and modular files become unnecessary

---

## 2026-10-01 — OAuth Scopes as an Annotated, Tested List
**Chosen:** Replace the hardcoded `defaultScopes` string with an exported `auth.DefaultScopeList`, one scope per line annotated with the API group that needs it, plus a test mapping every API group to its required scope.
**Alternatives:** Append the missing scopes to the existing string; request scopes lazily per command; let users set `DTIAM_SCOPES`.
**Why:** Two commands (`account subscriptions`, `get platform-tokens`) were shipping broken — the token request omitted `account-uac-read` and `platform-token:tokens:manage`, so every call returned HTTP 403. The scope for platform tokens was even documented in `tokens.go` but never requested, which is exactly the drift an annotated list plus a test prevents. Lazy per-command scoping would mean a token cache per scope set and more SSO round trips for no benefit, since Dynatrace grants only the scopes the OAuth client actually has.
**Trade-offs:** Requesting more scopes than a given command needs. Harmless — the SSO endpoint grants the intersection of requested and granted scopes, so an OAuth client without `account-uac-read` still gets a working token for everything else.
**Revisit if:** Dynatrace starts rejecting token requests that ask for scopes the client lacks, rather than returning the intersection. That would force per-command scope sets.

---

## 2026-10-01 — Pagination Declared Per Handler, Resolved Inside List
**Chosen:** A `*client.PaginationConfig` field on `BaseHandler` describing the endpoint's paging style, with `BaseHandler.List` looping internally until the collection is exhausted. Nil means unpaginated.
**Alternatives:** A separate `ListAll` method; paginate unconditionally on every endpoint; expose pages to callers and let commands loop.
**Why:** The Account Management API is not uniform — users/groups return `{count, items}` with no paging parameters at all, service users use a `page-key` cursor returning `{results, nextPageKey, totalCount}`, and platform tokens use 1-based `page`/`size` returning `{pageSize, pageNumber, total, results}`. Paginating unconditionally would send parameters that unpaginated endpoints ignore, which is harmless but misleading; worse, it invites assuming a uniformity that does not exist. Keeping the loop inside `List` means no command can accidentally render a truncated list, which is the failure mode that made `get service-users` and `get platform-tokens` return wrong results silently. A separate `ListAll` would have left the broken `List` as the easy default.
**Trade-offs:** Each new paginated endpoint needs its config wired up explicitly; forgetting it yields first-page-only results. Mitigated by asserting the config in each handler's constructor test.
**Revisit if:** Dynatrace unifies pagination across the Account Management API, at which point the style enum collapses to one case.

---

## 2026-10-01 — Trust Documented Response Shapes Over Existing Test Fixtures
**Chosen:** When a handler's `ListKey` disagreed with the documented response shape, treat the documentation as correct and rewrite the test fixture.
**Alternatives:** Preserve the fixtures and support both keys; leave as-is since tests were green.
**Why:** `serviceusers.go` and `tokens.go` both read `items`, but those endpoints return `results`. The tests mocked `items`, so a full green suite coexisted with two commands that returned an empty list against the live API. Green tests over a fabricated fixture are worse than no tests: they actively argue the code is correct. The fallback chain still accepts `items` for safety, but the fixtures now encode the documented shape so the tests would catch a regression.
**Trade-offs:** The fixtures are only as good as the documentation; neither was verified against a live account in this pass.
**Revisit if:** A live-account smoke test contradicts the documented shapes. That test is the real fix and is not yet written.

---

## 2026-10-01 — Permission Management Exposed Alongside Policies, Not Merged Into Them
**Chosen:** Surface the role-style permission grants as their own commands (`group permissions`, `grant-permission`, `revoke-permission`) rather than folding them into the existing policy/binding commands or the `analyze` output.
**Alternatives:** Merge direct grants into `group bindings`; include them in `analyze group-permissions`; skip the API as legacy.
**Why:** The two models are genuinely separate in the API and a group's effective access is their union, so hiding one inside the other would misrepresent both. Merging them into `group bindings` would conflate a binding (policy + boundary) with a grant (permission + scope), which have different shapes and different lifecycles. Skipping the API was tempting since it predates IAM policies, but the documentation still lists it as current as of September 2026, and a group carrying direct grants is invisible to every other dtiam command — exactly the blind spot an IAM tool must not have.
**Trade-offs:** Users must know to check both. `analyze` still does not merge the two into a single effective-access view, which is the obvious follow-up.
**Revisit if:** Dynatrace deprecates the permission management API, or `analyze` grows a unified effective-access view that should consume both.

---

## 2026-10-01 — Environment-Level IAM Kept Separate From Account-Level Commands
**Chosen:** Expose the environment-served Platform IAM API as distinct `get env-users` / `get env-groups` commands rather than as flags on `get users` / `get groups`.
**Alternatives:** An `--environment` flag on the existing commands that silently switches API; omit the API as dtctl's territory.
**Why:** They answer different questions against different hosts with different scopes. `get users` lists the account's user records from `api.dynatrace.com` with `account-idm-read`; this API reports who is visible at an organizational level from `{env}.apps.dynatrace.com` with `iam:users:read`, a scope granted on the environment rather than the account. A flag that switched between them would mean the same command returning different fields, different pagination, and failing with a different permission error depending on one flag. Separate commands make the distinction visible in `--help`, where it belongs.
**Trade-offs:** Two more commands, and some apparent duplication for users who do not care which API answers them.
**Revisit if:** Dynatrace consolidates the two into one API surface.

---

## 2026-10-01 — Overridable Base URLs on Handlers Outside the Account Scope
**Chosen:** Give `ReferenceHandler`, `NotificationHandler`, and `SubscriptionHandler` (for v3) exported, overridable base URL fields defaulting to the production constants.
**Alternatives:** Hardcode the constants; inject a full URL at construction; test through an HTTP transport shim.
**Why:** These endpoints do not hang off the client's account-scoped base URL — reference data is not account-scoped at all, notifications sit at an unprefixed `/v1`, and subscription cost moved to `sub/v3` while the rest of the handler stays on v2. With the constants hardcoded, the first versions of these tests either reached for the live API or degenerated into shims that re-implemented the method under test, which proves nothing. An overridable field is the smallest change that makes the real code path testable.
**Trade-offs:** Three more exported fields that callers could set to something wrong.
**Revisit if:** The client grows a general notion of multiple service base URLs, which would make these fields redundant.

---

## 2026-10-01 — Adopt dtctl's verb_resource.go File Layout
**Chosen:** Split each `internal/commands/<verb>/<verb>.go` into one file per resource (`get_groups.go`, `analyze_policy.go`, ...), keeping the existing package-per-verb structure.
**Alternatives:** Leave the large files alone; move to dtctl's flat `package cmd` with every command in one directory.
**Why:** Five files had passed 500 lines and `analyze.go` held 954 lines across 7 unrelated subcommands, so any change meant scrolling past six others and every concurrent edit touched the same file. dtctl's `verb_resource.go` convention solves exactly this and is already proven at 135 files. Adopting dtctl's *flat* layout as well would have meant collapsing 15 packages into one and renaming every symbol to avoid collisions — a much larger change for no benefit, since the package-per-verb split already prevents the name clashes that force dtctl to prefix everything.
**Trade-offs:** More files to navigate, and `init()` registration is now spread across them. Verified safe by diffing the complete `--help` tree for every command and subcommand before and after: byte-identical.
**Revisit if:** dtiam grows enough commands that package-per-verb starts producing its own collisions.

---

## 2026-10-01 — Live Account Is Authoritative for Response Shapes, Not Documentation
**Chosen:** Verify every list endpoint's response against a live account, treat what it returns as authoritative, and pin each observed shape in `response_shapes_test.go`.
**Alternatives:** Trust docs.dynatrace.com; trust the OpenAPI specs; keep the defensive multi-key fallback and move on.
**Why:** Six commands were silently returning empty lists — `get environments`, `get boundaries`, `account limits`, `account subscriptions`, `service-user list`, `get tokens`. The documentation states `items` for endpoints that actually return `data`, `results`, or `content`, and does not mention that boundaries and limits paginate. The failure mode is the problem: an unmatched key yields an empty slice, not an error, so every one of these looked like "this account has none of those" rather than "this code is broken." `account check-capacity` was the worst case — it reported every limit as not found and computed 0/0, which reads as a real capacity answer. A fallback chain alone would have papered over it without recording what is actually true.
**Trade-offs:** The pinned shapes are a snapshot of one account on one date, and a future API change will break these tests rather than silently degrading — which is the point, but it does mean the tests need updating when the API legitimately changes.
**Revisit if:** Dynatrace publishes OpenAPI specs for the Account Management API that match observed behavior, at which point generated types would beat hand-pinned fixtures.

---

## 2026-10-01 — One Place for List-Key Resolution
**Chosen:** Delete the `List`/`extractList` overrides on `EnvironmentHandler`, `LimitsHandler`, and `SubscriptionHandler`; resolve list keys only in `BaseHandler`.
**Alternatives:** Fix each override's key list in place; add the missing keys to every override.
**Why:** Each override re-implemented key resolution with its own hardcoded guess, so setting `ListKey` on the embedded `BaseHandler` had no effect at all — the fix looked applied and changed nothing. Three independent copies meant three places to be wrong and no single place to be right. Removing them deleted roughly 100 lines and made the shared fallback chain actually reachable. The only behavior worth keeping from the overrides was wrapping a single-resource response, which now lives in `BaseHandler` keyed off the handler's own `IDField`/`NameField` rather than a hardcoded `"name"` or `"uuid"`.
**Trade-offs:** `BaseHandler.extractList` now carries a longer fallback chain, which could in principle match an unintended key on a future endpoint. Guarded by asserting each handler's explicit `ListKey` in tests, so resolution never depends on the fallback.
**Revisit if:** An endpoint appears whose envelope uses one of these keys for something that is not the collection.

---

## 2026-10-01 — Test Helpers Must Use Production Constructors
**Chosen:** Build handlers in tests via `NewXHandler(client)` and override only the URL, rather than hand-rolling a `BaseHandler` literal.
**Alternatives:** Keep the hand-rolled literals; inject a test-only config struct.
**Why:** `newTestLimitsHandler` constructed its own `BaseHandler` with `ListKey: "items"` and `IDField: "name"` — a configuration the production code never used. The tests therefore validated the fixture against itself and could not fail when `NewLimitsHandler` carried the wrong keys. This is the same failure as the fabricated fixtures: the test agreed with itself and told us nothing about the shipped code.
**Trade-offs:** Tests are now coupled to constructor signatures, so a constructor change touches them. That coupling is the feature — it is what makes a misconfigured constructor fail.
**Revisit if:** A constructor starts requiring expensive setup that tests cannot reasonably provide.

---

## 2026-10-01 — Keyring With a Reported Plaintext Fallback
**Chosen:** Store client secrets in the OS keyring via `zalando/go-keyring`, writing a `keyring:dtiam` marker to the config file in place of the secret. Fall back to plaintext when no keyring exists, and always report which happened.
**Alternatives:** Require a keyring and fail without one; keep plaintext only; encrypt the config file with a passphrase.
**Why:** Plaintext secrets in a dotfile were the project's clearest security weakness, but requiring a keyring would break dtiam on exactly the hosts it is most used from — headless Linux boxes, containers, CI runners — none of which run a keyring daemon. A silent fallback would be worse than plaintext-only, since the user would believe their secret was protected. So the fallback exists but is never quiet: `set-credentials` names the destination, `doctor` warns every run, and `keyring-status` shows the state per credential. `--require-keyring` is there for anyone who wants the strict behavior. Passphrase encryption was rejected because it would prompt on every invocation, which breaks the non-interactive use `--plain` exists to serve.
**Trade-offs:** Three new indirect dependencies, and a `keyring:dtiam` marker that a third-party tool reading the config file would not understand. The marker is deliberately not a valid Dynatrace secret shape, so such a tool fails loudly rather than authenticating with nonsense.
**Revisit if:** A keyring backend proves unreliable enough that the probe cost or false negatives become a problem — `DTIAM_DISABLE_KEYRING` is the escape hatch in the meantime.

---

## 2026-10-01 — Agent Detection Implies --plain
**Chosen:** Detect coding-agent environment variables and enable `--plain` automatically unless `--plain` was passed explicitly or `DTIAM_NO_AGENT_DETECT` is set.
**Alternatives:** Require agents to pass `--plain`; detect only a TTY; detect CI as well as agents.
**Why:** Under an agent the interactive defaults are not merely unhelpful, they are wrong: ANSI colors become literal escape sequences in a transcript, and a confirmation prompt blocks forever because there is no stdin to answer it. Requiring `--plain` means every agent integration has one more thing to get right, and the failure is a hang rather than an error. A TTY check alone would also catch ordinary pipes (`dtiam get groups | grep`), where a human is still reading the output and the table is what they want. CI is detected separately and deliberately does *not* imply `--plain`, since CI output is usually read by a human in a log later, so the table is still the better format there.
**Trade-offs:** Output format now depends on the environment, which can surprise someone who did not expect it. Mitigated by `-v` naming the triggering variable, `--plain=false` overriding, and the opt-out env var.
**Revisit if:** A tool exports one of these variables in a context where a human is genuinely reading colored output.

---

## 2026-10-01 — Diff Compares Formatted Values, Not Go Values
**Chosen:** Compare spec and live fields by their formatted string representation, sorting list elements, rather than with `reflect.DeepEqual`.
**Alternatives:** `reflect.DeepEqual`; normalize both sides into typed structs first; compare marshalled JSON.
**Why:** The same logical value arrives with different Go types depending on source — a YAML file decodes `5` as `int`, the API returns it as `float64` — so `DeepEqual` would call them different and every numeric field would appear modified on every run. List order has the same problem: the API returns members, scopes and zones in an order the caller does not control, so an order-sensitive comparison reports changes that do not exist. A diff that cries wolf on every field is worse than no diff, because it trains the user to ignore it. Typed structs would fix the numeric case but require a schema per resource kind, which the generic `map[string]any` handler design does not have.
**Trade-offs:** Two values with different types but identical formatting compare equal — `"5"` and `5`, for instance. For IAM specs that is the desired behavior, since the API is loose about which it returns, but it would be wrong for a type-sensitive domain.
**Revisit if:** dtiam gains typed resource models, which would make structural comparison both possible and more precise.

---

## 2026-10-01 — Watch Fingerprints Sorted Encodings
**Chosen:** Detect change by hashing each item's JSON encoding, sorting the encodings, and comparing the hash — with no ID field required.
**Alternatives:** Compare lengths; sort by a configured ID field; diff item by item and report what changed.
**Why:** Length alone misses edits and simultaneous add/remove. Sorting by an ID field would need each watched resource to declare one, and the field differs per resource (`uuid`, `uid`, `tokenId`, `limitType`) — exactly the inconsistency that caused the response-key bugs. Sorting the encodings sidesteps the question entirely and is immune to both item order and field order, since marshalling a Go map already sorts keys. Reporting *what* changed would be nicer, but a reprint is what someone watching a migration actually wants, and the per-item diff machinery already exists in `pkg/diff` if that becomes worth building.
**Trade-offs:** The watch says "something changed" rather than "member X was added". Acceptable for the use case; the full collection is reprinted so the change is visible.
**Revisit if:** Users want change-only output, at which point `pkg/diff` can be applied between consecutive polls.

---

## 2026-10-01 — pkg/ for the Library, internal/ for the CLI Wiring
**Chosen:** Move 13 packages (`auth`, `client`, `config`, `diagnostic`, `diff`, `logging`, `output`, `prompt`, `resources`, `suggest`, `template`, `utils`, `watch`) from `internal/` to `pkg/`, and keep `internal/cli` and `internal/commands` internal.
**Alternatives:** Leave everything in `internal/`; mirror dtctl and move `commands` to `pkg/` as well; keep `internal/` and add a thin `pkg/` facade.
**Why:** Everything in `internal/` was unimportable, so anyone wanting dtiam's IAM client had to shell out to the binary and parse its output — which is what `--plain` exists for, but a poor substitute for calling `resources.NewGroupHandler(c).List(ctx, nil)`. The split line is the useful one: `pkg/` is what a caller would reuse, `internal/` is cobra wiring. Moving `commands` out too (as dtctl does) would invite callers to depend on command plumbing — flag registration, `RunE` closures, global state — rather than on the client and handlers, and that is a dependency we would then have to keep stable. A facade would have meant maintaining two surfaces for one implementation.
**Trade-offs:** A breaking change for anyone importing the old paths, hence 3.0.0. Mitigated by the migration being a mechanical `internal/` → `pkg/` substitution, and by the CLI being entirely unaffected: the complete `--help` tree is byte-identical across 4,110 lines, and an external module importing eight of the moved packages was built and run to confirm they resolve from outside the module.
**Revisit if:** A `pkg/` package needs something from `internal/cli` — that would be a sign the split line is in the wrong place, and the shared piece should move to `pkg/` rather than the import being added.

---

## 2026-10-02 — The Live Spec and a Live Account Are the References, Not the Docs Pages
**Chosen:** Validate every endpoint against `https://api.dynatrace.com/spec-json` (method, path, body schema, `x-token-scopes`, `deprecated`) and then against a live account, with write paths exercised on throwaway `dtiam-probe-*` objects that are deleted afterwards.
**Alternatives:** Docs pages; the local `ARCHIVE/go-dtctl-main/api-spec/` folder; mocked tests only.
**Why:** Phase 09 "verified all endpoints" against the local spec folder, which contains no Account Management spec at all, so the claim was unfalsifiable. The live spec immediately showed operations dtiam used that do not exist (`POST`/`PUT` on the level-wide bindings collection, `POST /groups/{uuid}/users`, `GET /groups/{uuid}`, `GET /users/{uid}`). But the spec alone is not enough either: it says nothing about `POST /groups` rejecting a bare object, the boolean `error` field, or 400-instead-of-404 for names. Those only showed up against a real account. Mock tests had passed throughout because each one encoded the same wrong assumption as the code it tested.
**Trade-offs:** Live write testing touches a real account. Confined to objects created for the test, with membership and binding phases kept separate so the probe group never held both a member and a policy.
**Revisit if:** A dedicated sandbox account becomes available, at which point this should become a `//go:build integration` suite like dtctl's.

---

## 2026-10-02 — Collections Without a Single-Item GET Resolve From List
**Chosen:** A `BaseHandler.NoSingleGet` flag that makes `Get` search `List` by `IDField`, set on groups, environments and platform tokens.
**Alternatives:** Per-handler `Get` overrides (what tokens and environments had); fall back from 404 to list in every caller.
**Why:** The per-handler overrides had drifted -- the token one compared the wrong field, the environment one called the missing endpoint. Callers falling back on 404 is what `GetOrResolve` did, but direct `handler.Get` callers (analyze, export, describe expansion) did not, which is why those commands failed while `get groups ID` worked. Putting the knowledge on the handler makes every caller correct and saves a guaranteed-404 round trip.
**Trade-offs:** A by-ID lookup now costs a full list. These collections are small (hundreds), and the alternative was a 404 followed by the same list.
**Revisit if:** The API adds single-item GETs, or a collection grows large enough that listing per lookup matters.

---

## 2026-10-02 — Environment Commands Request Their Own Scopes
**Chosen:** Environment-served commands build a client via `common.CreateEnvironmentClient(scopes)`, which uses a configured environment token if present, else requests only that command's scopes (`auth.EnvironmentIAMScopes`, `AppEngineScopes`, `SettingsSchemaScopes`) with the account OAuth client.
**Alternatives:** Add the scopes to `DefaultScopeList`; require an environment token for all environment commands.
**Why:** A token request naming a scope the OAuth client was not granted fails outright with HTTP 400 -- verified live -- so adding `app-engine:apps:run` to the default set would break *every* command for clients without it. Requiring an environment token was unnecessary: an account OAuth token with `iam:users:read` / `iam:groups:read` works against the environment Platform IAM API (verified live). Per-command scopes mean a missing grant fails exactly the command that needs it.
**Trade-offs:** Environment commands ignore `DTIAM_SCOPES`, since that override is shaped for the account APIs. An environment token is the escape hatch.
**Revisit if:** Users need to override environment scopes independently; add `DTIAM_ENVIRONMENT_SCOPES` then.

---

## 2026-10-02 — API Host Override Rewrites the Host, Not a Base Path
**Chosen:** `DTIAM_API_URL` / `api-url` is reduced to scheme+host and replaces `https://api.dynatrace.com` in every account-API URL, relative or absolute, at request time.
**Alternatives:** Override only the client's relative base URL; a per-API base-URL map.
**Why:** dtiam reaches nine account APIs (`/iam/v1`, `/env/v2`, `/sub/v2`, `/sub/v3`, `/audit/v1`, `/ref/v1`, `/v1`...) through absolute URLs, so overriding the relative base alone would have moved a fraction of the calls and silently left the rest on production -- worse than the setting doing nothing. All of them share one host, so a host rewrite moves them together with no per-API configuration. Environment URLs and the SSO endpoint are on other hosts and are untouched; a host that merely shares the prefix is not rewritten.
**Trade-offs:** The SSO token URL is not covered, so a dev stage with its own SSO still needs more work.
**Revisit if:** A target stage needs a different SSO endpoint, or the account APIs split across hosts.
