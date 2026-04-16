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

---

## Error Responses

### Standard Error Shape

Most 4xx/5xx responses follow this format:

```json
{
    "error": {
        "code": 404,
        "message": "Group not found"
    }
}
```

But some endpoints return:

```json
{
    "message": "Forbidden",
    "statusCode": 403
}
```

dtiam's `handleResponse()` checks `error.message` first, then falls back to `message`, then to the raw response body.

### 404 on Deleted Resources

Deleting a resource that was already deleted returns 404, not a success code. dtiam treats this as a non-error for idempotent delete operations.

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

### Deleting Bindings

The DELETE endpoint for bindings does not use a binding UUID. Instead, it requires the group UUID and policy UUID in the request body. This is because bindings are identified by their (group, policy) composite key, not by a unique ID.

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

### UID Format

Service user UIDs follow a different format than regular user UIDs. They look like OAuth client IDs: `dt0s01.XXXXXX`.

---

## Environment-Specific APIs

### App Engine Registry

- **Base URL:** `https://{environment-id}.apps.dynatrace.com/platform/app-engine/registry/v1/apps`
- Requires environment-level token with `app-engine:apps:run` scope
- This is a different base URL pattern than the account management API — it targets the environment directly, not `api.dynatrace.com`

### Settings Schemas

- **Base URL:** `https://{environment-id}.apps.dynatrace.com/platform/classic/environment-api/v2/settings/schemas`
- Same environment-level pattern
- Used for validating schema IDs in boundary creation

Both of these APIs require a separate token/scope from the account-level IAM token. dtiam uses `DTIAM_ENVIRONMENT_URL` and `DTIAM_ENVIRONMENT_TOKEN` environment variables for these.

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
