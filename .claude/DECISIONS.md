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
