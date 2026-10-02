# Dynatrace Account Management API — Empirical Quirks

These are behaviors discovered through building and testing dtiam that are not documented (or poorly documented) in the official Dynatrace API reference. They represent tribal knowledge that would otherwise require rediscovery.

**Source:** 737 tests + production usage against multiple Dynatrace accounts

---

## Authentication

### OAuth2 Client Credentials

- **Token endpoint:** `https://sso.dynatrace.com/sso/oauth2/token`
- **Grant type:** `client_credentials`
- **Resource URN:** `urn:dtaccount:{account-uuid}` — MUST be included in the token request body. Without it, the token is issued but has no account scope and every API call returns 403.
- **Client ID extraction:** The client secret format is `dt0s01.{CLIENT_ID}.{SECRET_PART}`. The client ID can be auto-extracted from the secret by splitting on `.` and taking the second segment. dtiam does this — users only need to provide the secret, not both ID and secret separately.
- **Token expiry buffer:** dtiam considers a token invalid 30 seconds before actual expiry to avoid race conditions on slow networks.
- **Scope format:** Space-separated in the token request: `"account-idm-read account-idm-write iam-policies-management account-env-read"`

### Bearer Tokens

- Static bearer tokens do NOT auto-refresh. When they expire, every API call returns 401 with no retry. dtiam detects this and shows a clear message directing users to OAuth2.
- API tokens created in the Dynatrace UI have a different format than OAuth2 access tokens. Both work as bearer tokens, but UI-created tokens have longer expiry (configurable) while OAuth2 tokens are typically 300 seconds.

---

## Response Format Inconsistencies

### List Responses

Different endpoints return lists in different wrapper formats. dtiam's `BaseHandler.extractList()` handles all three:

1. **Keyed list:** `{ "items": [...] }` or `{ "groups": [...] }` — most common
2. **Direct array:** `[...]` — some endpoints return a bare JSON array
3. **Single object in array:** Some endpoints return `{ "data": {...} }` when only one result exists, not `{ "data": [{...}] }`

The handler tries the configured `ListKey` first, then falls back to treating the entire response as an array.

### Field Name Inconsistencies

| Resource | Quirk |
|----------|-------|
| Users | `uid` (not `uuid`, not `id`) is the unique identifier |
| Users | Group membership returned as either `groups` array on the user object OR via separate `/users/{uid}/groups` endpoint — both exist, results may differ in format |
| Groups | Member count available as `memberCount` field on some responses, absent on others. dtiam falls back to `GET /groups/{uuid}/users` and counts the result |
| Policies | Level metadata returned with underscore-prefixed fields: `_level_type`, `_level_id` — not standard JSON naming |
| Bindings | The field is `policyUuid` (camelCase), not `policy_uuid` or `policy-uuid` |
| Bindings | Boundaries attached to bindings are in a `boundaries` array of UUIDs, but the field may be `null` (not absent, not empty array) when no boundaries exist |
| Subscriptions | The list endpoint returns items under `subscriptions` key, not `items` |
| Environments | The list endpoint returns under `environments` key, not `items` |

### Null vs Absent vs Empty

The API is inconsistent about representing "no value":

- **Groups without description:** Field is absent from response (not `null`, not `""`)
- **Bindings without boundaries:** Field is `null` (not absent, not `[]`)
- **Users without groups:** Can be `null`, `[]`, or absent depending on the endpoint

dtiam's `utils.StringFrom()` and `safemap` helpers handle all three cases uniformly. Any porting effort should replicate this defensive approach.

---

## Pagination

The account APIs page in at least four incompatible ways. Each rule below was
verified against a live account.

| API | Style | Page size | Later pages send |
|---|---|---|---|
| Notifications `GET /v2/.../notifications` | `page-key` / `nextPageKey`, `hasNextPage` | default 20, 500 accepted | **`page-key` alone** -- resending filters with it is a 400 |
| Subscription v3 `.../environments/usage` and `/cost` | `page-key` / `nextPageKey` (null on the last page) | **max 50** (more is a 400) | **`page-key` plus the full query** -- `page-key` alone is a 400 ("'endTime' must be provided") |
| Service users | `page-key` / `nextPageKey` | 500 | see `client.ServiceUserPagination` |
| Platform tokens, boundaries, limits | page number | 500 | page number |

Notifications v2 list filters (`types`, `severities`, `environments`,
`capabilities`) are **repeated** parameters (`types=A&types=B`); a
comma-separated value is a 400. The documented `totalRecordCount` field is not
sent. The Subscription v3 filters (`environmentIds`, `capabilityKeys`) are
comma-separated. v3 usage splits one environment's records across pages, so the
same `environmentId` appears in several entries; v3 totals match v2 exactly.

### Effective Permissions API

- Uses **page-based** pagination: `page=1&size=100`
- Response includes `total` field indicating total result count
- Items may be under `effectivePermissions` key OR `items` key — dtiam checks both
- Page numbering starts at 1, not 0

### Environments API

- **Not paginated.** Returns all environments in a single response. Accounts with 100+ environments still get a single response.

### Subscriptions API

- **Not paginated** for the list endpoint. Paginated for cost-per-environment (v3 endpoint uses cursor-based `page-key`).

### Bindings API

- **Not paginated.** Returns all bindings for the specified level in one response. This can be large for accounts with many policies.
- **The group view has its own shape.** `GET .../bindings/groups/{uuid}` returns `{"policyUuids": [...]}`, not the `policyBindings` envelope the other binding endpoints use. With `?details=true` it adds `bindingsDetails`, one `{policyUuid, groups, boundaries, levelType, levelId}` entry per binding. Reading `policyBindings` from it yields nothing, which is how `group bindings`, `group clone --include-policies` and the local `analyze` commands came back empty before 3.0.1.
- **The pair view is wrapped.** `GET .../bindings/{policy}/{group}` returns `{levelType, levelId, policyBindings: [...]}`, while `PUT` on the same path takes the bare `{boundaries, parameters, metadata}`. Read the binding from inside the envelope; reading `boundaries` from the top level finds none.

---

## Error Responses

### Standard Error Shape

The Account Management API (`api.dynatrace.com`) returns a **boolean** `error`:

```json
{ "error": true, "message": "Cannot get requested resource.", "payload": null }
```

The environment Platform APIs (`{env}.apps.dynatrace.com`) nest the message:

```json
{ "error": { "code": 400, "message": "Mandatory query param partialGroupName or uuid was not provided" } }
```

The SSO token endpoint uses OAuth's `{"error": "invalid_request", "error_description": "..."}`.

`extractErrorMessage()` in `pkg/client` reads `message`, then `error` as a string, then `error.message`, then `error_description`, then the raw body. Before 3.0.1 it decoded into a struct with a string `error` field; the boolean form made that decode fail and **every Account Management error printed with a blank message**.

### 400, Not 404, for a Name Where an ID Is Expected

`GET .../policies/{x}` and `GET .../boundaries/{x}` answer a non-UUID with `400 Validation failed (uuid is expected)`. Name resolution must treat that like 404 and fall back to the list, or `get policies NAME` fails.

### Endpoints That Have No Single-Item GET

| Collection | Path that does not exist | Resolve instead from |
|---|---|---|
| Groups | `GET /groups/{uuid}` (only PUT and DELETE) | `GET /groups` |
| Environments (v2) | `GET /env/v2/.../environments/{id}` | `GET .../environments` |
| Platform tokens | `GET /platform-tokens/{id}` (only DELETE) | `GET /platform-tokens` |
| Users by UID | `GET /users/{uid}` returns 400 "Expected email to be email" | `GET /users/{email}` |

Handlers mark the first three with `BaseHandler.NoSingleGet`, which makes `Get` resolve from `List`.

### 404 on Deleted Resources

Deleting a resource that was already deleted returns 404, not a success code. dtiam treats this as a non-error for idempotent delete operations.

### Request Body Shapes That Differ From the Obvious

- `POST /groups` takes and returns an **array** of groups. A bare object fails with `500 payload.map is not a function`.
- `POST /service-users` and `PUT /service-users/{uid}` accept only `name` and `description`; `name` is required on PUT. Group membership is not part of either.
- `PUT /groups/{uuid}` takes `{uuid, name, description, federatedAttributeValues}`.
- `PUT .../policies/{uuid}` requires `name`, `description` and `statementQuery`. The policy **list** omits `statementQuery`, so an update must start from `GET .../policies/{uuid}`.

### Platform Tokens

- `POST /platform-tokens` requires `name`, `scope`, `resource` (URNs such as
  `urn:dtaccount:{uuid}` or `urn:dtenvironment:{id}`), `tags`, `expirationDate`
  and `userUuid`. It returns `{name, tokenId, token}` once.
- The owner (`userUuid`) must be the calling identity. Minting a token for
  another service user returns 403 with no further detail. For an OAuth client,
  the caller is the service user in the access token's `sub` claim.
- `PUT /platform-tokens/{id}/status` takes `{"status": "ACTIVE"|"INACTIVE"}`;
  `PUT /platform-tokens/{id}/expiration-date` takes `{"expirationDate": RFC 3339}`.

### 409 on Duplicate Create

Creating a group/policy with a name that already exists returns 409 Conflict. The error message includes the existing resource's UUID, which can be parsed for "upsert" logic.

### Rate Limiting

- Returns HTTP 429 with `Retry-After` header (seconds)
- dtiam's Resty client handles this automatically with exponential backoff (3 retries, 1s/2s/4s)
- Rate limits are per-account, not per-token. Multiple tokens hitting the same account share the limit.

---

## Policy Statement Format

### Semicolons in Statements

The API stores and returns policy statements exactly as submitted, including trailing semicolons. A statement like `ALLOW settings:objects:read;` is valid and equivalent to `ALLOW settings:objects:read`. dtiam's parser handles both.

### Case Sensitivity

- `ALLOW` and `allow` are both accepted by the API
- Action strings are case-sensitive: `settings:objects:Read` will NOT match `settings:objects:read`
- dtiam normalizes effect to uppercase when parsing

### Multi-Statement Policies

A single policy can contain multiple semicolon-separated statements:

```
ALLOW settings:objects:read; ALLOW settings:schemas:read; DENY account:users:write
```

The API stores this as a single string. dtiam splits on `;` and parses each sub-statement independently.

---

## Boundary Queries

### Format

Boundary queries use a SQL-like syntax:

```
{scope}:{attribute} {IN|NOT IN} ({quoted, comma-separated values})
```

### Known Scopes and Attributes

| Scope:Attribute | Used For |
|----------------|----------|
| `environment:management-zone` | Management zone boundaries |
| `storage:dt.security_context` | Storage security context (often matches management zones) |
| `settings:dt.security_context` | Settings security context |
| `shared:app-id` | App Engine app boundaries |
| `settings:schemaId` | Settings 2.0 schema boundaries |

### Multi-Line Boundary Queries

Management zone boundaries typically combine three lines:

```
environment:management-zone IN ("Production");
storage:dt.security_context IN ("Production");
settings:dt.security_context IN ("Production")
```

All three lines must be present for the boundary to work correctly across all Dynatrace features. Omitting the `storage` or `settings` line creates a partial boundary that doesn't restrict settings or storage access.

### Quoting

Values MUST be double-quoted inside the parentheses: `IN ("value1", "value2")`. Single quotes are rejected by the API. dtiam uses Go's `fmt.Sprintf("%q", value)` which produces double-quoted output.

---

## Binding Semantics

### Three-Way Relationship

A binding connects: Group + Policy + optional Boundary(s)

- Group UUID is required
- Policy UUID is required
- Boundary UUIDs are optional (array, can be empty or null)
- A single group can be bound to the same policy multiple times with different boundaries

### Creating and Deleting Bindings

Bindings have no UUID; they are addressed by the (policy, group) pair in the path:

- Create: `POST .../bindings/{policy}/{group}` with `{boundaries, parameters, metadata}` (appends a binding).
- Delete: `DELETE .../bindings/{policy}/{group}`.
- Change boundaries: `PUT .../bindings/{policy}/{group}` with the bare binding.

The level-wide `.../bindings` collection has only `GET` and `DELETE` -- and that DELETE removes **every** binding at the level. Before 3.0.1 dtiam created bindings with `POST .../bindings` and deleted them by rewriting the whole level with `PUT .../bindings`; neither operation is in the spec.

A parameterized policy can bind the same pair more than once with different parameters. The per-pair endpoints then match on `query-params`; dtiam refuses boundary changes in that case rather than editing one binding arbitrarily.

### Level Scoping

Bindings exist at a specific level: `account`, `environment:{envId}`, or `global`. The level is part of the API path, not the binding payload:

```
/iam/v1/repo/{levelType}/{levelId}/bindings
```

A policy bound at the account level grants permissions account-wide. A policy bound at the environment level grants permissions only in that environment.

---

## Service Users

### Secret Return

`POST /service-users` returns the generated client secret in the response body. This is the ONLY time the secret is available — it cannot be retrieved again. If the user loses it, they must delete and recreate the service user.

### UID and Email

A service user's `uid` is a UUID, like a regular user's. Its `email` is `{uid}@service.sso.dynatrace.com`. The OAuth client ID (`dt0s02.XXXX`) is a separate value returned on create.

### Group Membership

`GET /service-users/{uid}` does **not** include groups. Service users are users for membership purposes: read groups from `GET /users/{email}` and change them with `POST /users/{email}` and `DELETE /users/{email}/groups?group-uuid=...`. A new service user is also given its own `service_user_group_{uid}` group automatically.

---

## User Group Membership

Membership is managed through the user, never through the group:

| Operation | Endpoint |
|---|---|
| List a user's groups | `GET /users/{email}` (the `groups` field; there is no `GET /users/{x}/groups`) |
| Add to groups | `POST /users/{email}` with a JSON array of group UUIDs |
| Remove from groups | `DELETE /users/{email}/groups?group-uuid=A&group-uuid=B` (query params, not a body) |
| Replace all | `PUT /users/{email}/groups` with a JSON array |
| List a group's members | `GET /groups/{uuid}/users` |

There is no `POST /groups/{uuid}/users` or `DELETE /groups/{uuid}/users/{uid}`. `GET /users` excludes service users unless `?service-users=true`, which returns both kinds.

---

## Environment-Specific APIs

### App Engine Registry

- **Base URL:** `https://{environment-id}.apps.dynatrace.com/platform/app-engine/registry/v1/apps`
- Requires environment-level token with `app-engine:apps:run` scope
- This is a different base URL pattern than the account management API — it targets the environment directly, not `api.dynatrace.com`

### Settings Schemas

- **Base URL used by dtiam:** `https://{environment-id}.live.dynatrace.com/api/v2/settings/schemas`
- Used for validating schema IDs in boundary creation

### Environment-Level Platform IAM

- **Base URL:** `https://{environment-id}.apps.dynatrace.com/platform/iam/v1/organizational-levels/{account|environment}/{id}`
- `users` requires `partialString` or `uuid`; `groups` requires `partialGroupName` (minimum 3 characters) or `uuid`. Neither will enumerate everything.
- Groups come back as `{uuid, groupName, type}`, not the account API's `{uuid, name, owner}`.
- An **account** OAuth token works here, provided it carries `iam:users:read` / `iam:groups:read`.

### Authenticating to Environment APIs

These scopes are deliberately not in the default set: requesting a scope the OAuth client was not granted fails the whole token request with HTTP 400, which would break every command. Instead each environment command requests only its own scopes (`auth.EnvironmentIAMScopes`, `auth.AppEngineScopes`, `auth.SettingsSchemaScopes`), so only that command fails if they are missing. An environment token (`DTIAM_ENVIRONMENT_TOKEN` or the credential's `environment-token`) takes precedence when set. The environment URL comes from `--environment`, `DTIAM_ENVIRONMENT_URL`, or the credential's `environment-url`.

---

## Retry and Timeout

### Resty Configuration

dtiam uses these defaults:

- **Retry count:** 3
- **Wait time:** 1 second (initial)
- **Max wait time:** 5 seconds
- **Retry conditions:** HTTP 429 (rate limited) and 5xx (server errors)
- **Request timeout:** 30 seconds

### Long-Running Operations

Bulk operations (adding 100+ users to a group) can trigger rate limiting mid-operation. dtiam handles this per-request via Resty's retry, but does not have global rate limiting awareness. If bulk operations consistently hit limits, the user sees intermittent "Warning: failed to add member" messages as individual retries eventually exhaust.
