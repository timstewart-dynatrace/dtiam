# Architecture

## Project Structure

```
dtiam/
├── cmd/dtiam/main.go                 # Entry point, command registration
├── internal/                         # CLI wiring -- not importable
│   ├── cli/
│   │   ├── root.go                   # Root command, global flags, ErrSilentExit
│   │   ├── state.go                  # Global state (context, output, verbose, agent)
│   │   └── agentdetect.go            # Coding-agent / CI detection
│   └── commands/                     # One package per verb, verb_resource.go files
│       ├── common/                   # Shared command utilities (CreateClient)
│       ├── config/                   # Config, credentials, keyring commands
│       ├── get/                      # List/retrieve, incl. --watch
│       ├── describe/ create/ delete/ # Single-resource operations
│       ├── user/ serviceuser/ group/ # Identity lifecycle
│       ├── token/                    # Platform token activate/deactivate/set-expiration
│       ├── boundary/ account/        # Boundaries; limits, subscriptions, usage
│       ├── bulk/ export/ apply/      # File-driven operations
│       ├── diff/                     # Preview what apply would change
│       ├── analyze/                  # Permission analysis
│       ├── doctor/                   # Diagnostic health checks
│       ├── template/ cache/          # Templates; cache management
│       └── ...
├── pkg/                              # Importable library
│   ├── auth/
│   │   ├── auth.go                   # TokenProvider interface
│   │   ├── oauth.go                  # OAuth2 token manager, DefaultScopeList
│   │   └── bearer.go                 # Static bearer token
│   ├── client/
│   │   ├── client.go                 # HTTP client with retry
│   │   ├── errors.go                 # APIError type
│   │   ├── urls.go                   # Centralized API URL constants
│   │   └── pagination.go             # Per-endpoint paging configs
│   ├── config/
│   │   ├── config.go                 # Config structs, effective-value helpers
│   │   ├── loader.go                 # Config load/save, XDG paths
│   │   └── keyring.go                # OS keyring secret storage
│   ├── resources/
│   │   ├── handler.go                # Handler interfaces + BaseHandler
│   │   ├── types.go                  # Typed response structs with table tags
│   │   ├── groups.go users.go        # Core identity handlers
│   │   ├── policies.go bindings.go boundaries.go
│   │   ├── environments.go serviceusers.go limits.go subscriptions.go
│   │   ├── tokens.go apps.go schemas.go
│   │   ├── audit.go                  # Account audit logs
│   │   ├── reference.go              # Reference data (grantable permissions)
│   │   ├── grouppermissions.go       # Role-style permission grants
│   │   ├── notifications.go          # Account notifications
│   │   └── orglevels.go              # Environment-level Platform IAM
│   ├── output/                       # Printer, formats, columns, tables
│   ├── diff/                         # Spec-vs-live comparison
│   ├── watch/                        # Change-detecting poller
│   ├── prompt/                       # Confirmation prompts
│   ├── diagnostic/                   # Errors with exit codes and suggestions
│   ├── logging/                      # Structured logging (logrus)
│   ├── suggest/                      # Levenshtein command/flag suggestions
│   ├── template/                     # Template engine, store, built-ins
│   ├── utils/                        # Permissions calculator, safe map helpers
│   └── version/version.go            # Version info
├── go.mod
├── Makefile
├── .golangci.yml
└── .goreleaser.yaml
```

**Why this split:** `pkg/` holds everything a caller could reasonably want to
reuse — the API client, resource handlers, auth, config, output, and the diff and
watch logic. `internal/cli` and `internal/commands` stay internal because they are
cobra wiring; exposing them would invite callers to depend on command plumbing
rather than on the client and handlers. The split was clean because none of the
`pkg/` packages referenced either one.

## Key Components

### CLI Layer (`internal/cli/`) — internal
- Root command with global flags (`--context`, `--output`, `--verbose`, `--plain`, `--dry-run`)
- GlobalState singleton accessed by all commands
- Printer factory method on GlobalState

### Command Layer (`internal/commands/`) — internal
- Verb-noun pattern: `get groups`, `create group`, `delete policy`
- Each verb is a package with a single exported `Cmd`
- Commands use `common.CreateClient()` for account APIs, and
  `common.CreateEnvironmentClient(auth.<X>Scopes)` for environment-served APIs
  (apps, schemas, env-users, env-groups) -- see DECISIONS.md 2026-10-02
- All follow `command-standards.md`

### Resource Layer (`pkg/resources/`)
- `BaseHandler` provides generic CRUD via HTTP methods
- Concrete handlers embed BaseHandler and override as needed
- Handler interface: `List()`, `Get()`, `Create()`, `Update()`, `Delete()`

### Output Layer (`pkg/output/`)
- Unified Printer supports table, wide, JSON, YAML, CSV
- `--plain` mode forces JSON for machine consumption
- Column definitions per resource type

### Auth Layer (`pkg/auth/`)
- `TokenProvider` interface with OAuth2 and Bearer implementations
- OAuth2 auto-refreshes expired tokens
- Bearer is static (no refresh)

## Data Flow

```
CLI Command → common.CreateClient() → Auth (OAuth2/Bearer)
    → Resource Handler → HTTP Client (Resty) → Dynatrace API
    → Response → Printer (table/json/yaml/csv) → stdout
```

## API Endpoints

Base URL: `https://api.dynatrace.com/iam/v1/accounts/{account_uuid}`

All shapes below were verified against a live account on 2026-10-01. **The
published documentation does not match the API for several of these** — it states
`items` for endpoints that return `data`, `results`, or `content`. Verify against
a real account before trusting a documented shape; an unmatched key yields an
empty list, not an error.

| Resource | Path | Response shape | Paginated | ID field |
|----------|------|----------------|-----------|----------|
| Groups | `/groups` | `{count, items}` | no | `uuid` |
| Users | `/users` | `{count, items}` | no | `uid` |
| Service Users | `/service-users` | `{results, nextPageKey, totalCount}` | cursor | `uid` |
| Platform Tokens | `/platform-tokens` | `{pageSize, pageNumber, total, results}` | page number | `tokenId` |
| Limits | `/limits` | `{pageSize, pageNumber, total, results}` | page number | `limitType` |
| Policies | `/repo/{level_type}/{level_id}/policies` | `{policies}` | no | `uuid` |
| Bindings | `/repo/{level_type}/{level_id}/bindings` | `{policyBindings}` | no | `policyUuid` |
| Boundaries | `/repo/account/{uuid}/boundaries` | `{pageSize, pageNumber, totalCount, content}` | page number | `uuid` |
| Environments | `/env/v2/accounts/{uuid}/environments` | `{data}` | no | `id` |
| Subscriptions | `/sub/v2/accounts/{uuid}/subscriptions` | `{data}` | no | `uuid` |
| Audit logs | `/audit/v1/accounts/{uuid}` | `{audits, warnings}` | no | `eventId` |
| Reference data | `/ref/v1/account/permissions` | bare array | no | `id` |

Field-name traps confirmed live: limits use `limitType`/`currentValue`/`limitValue`
(not `name`/`current`/`max`); platform tokens use `tokenId`/`expirationDate`/`scope`
(not `id`/`expiresIn`/`scopes`); environments use `active`/`url` (not `state`/`trial`).
The audit API's default projection returns only timestamp, eventType, user,
resource, resourceName, eventProvider and eventId — anything else needs
`--add-fields`.

Key resolution lives solely in `BaseHandler.extractList`/`extractPage`. Handlers
must not override them: three handlers used to, each with its own hardcoded key
list, which is how six commands came to return empty results.

Pagination is declared per handler via `BaseHandler.Pagination`
(`pkg/client/pagination.go`). A nil value means the endpoint returns its
whole collection in one response. `BaseHandler.List` follows all pages before
returning, so command code never sees a partial result.

`BaseHandler.NoSingleGet` marks collections with no GET-by-ID endpoint (groups,
environments, platform tokens); `Get` then resolves from `List`. Users are
addressed by email only -- `UserHandler` resolves a UID first. Live API shapes
that differ from the docs are listed in `docs/dev/API_BEHAVIORS.md`.

**Environment API**: `https://api.dynatrace.com/env/v2/accounts/{uuid}/environments`

**Subscription API**: `https://api.dynatrace.com/sub/v2/accounts/{uuid}/subscriptions`

**Resolution API** (effective permissions):
`https://api.dynatrace.com/iam/v1/resolution/{level_type}/{level_id}/effectivepermissions`

**App Engine Registry API**:
`https://{environment-id}.apps.dynatrace.com/platform/app-engine/registry/v1/apps`

**Audit API**: `https://api.dynatrace.com/audit/v1/accounts/{uuid}`
(response `{audits, warnings}`; warnings signal a partial result)

**Reference Data API**: `https://api.dynatrace.com/ref/v1/account/permissions`
(bare JSON array, not account-scoped)

**Notifications API**: `https://api.dynatrace.com/v1/accounts/{uuid}/notifications`
(POST with the filter in the body; note the unprefixed `/v1` path)

**Permission Management API**:
`https://api.dynatrace.com/iam/v1/accounts/{uuid}/groups/{group}/permissions`
(GET/POST/PUT/DELETE; role-style grants that coexist with IAM policies)

**Subscription API v3**:
`https://api.dynatrace.com/sub/v3/accounts/{uuid}/subscriptions/{sub}/environments/cost`
(cost moved to v3; listing, usage and forecast remain on v2)

**Environment-level Platform IAM API**:
`https://{environment-id}.apps.dynatrace.com/platform/iam/v1/organizational-levels/{level_type}/{level_id}/{users,groups,service-users}`
(served from the environment, not api.dynatrace.com; needs `iam:users:read`
granted on the environment. Organizational level types are `account` and
`environment` only -- `global` is not valid here.)

Level types: `account`, `environment`, `global`

## API Coverage

### Implemented

| Endpoint | Operation | Handler Method |
|----------|-----------|----------------|
| `GET /groups` | List groups | `GroupHandler.List()` |
| `GET /groups/{uuid}` | Get group | `GroupHandler.Get()` |
| `POST /groups` | Create group | `GroupHandler.Create()` |
| `PUT /groups/{uuid}` | Update group | `GroupHandler.Update()` |
| `DELETE /groups/{uuid}` | Delete group | `GroupHandler.Delete()` |
| `GET /users` | List users | `UserHandler.List()` |
| `GET /users/{uid}` | Get user | `UserHandler.Get()` |
| `POST /users` | Create user | `UserHandler.Create()` |
| `DELETE /users/{uid}` | Delete user | `UserHandler.Delete()` |
| `PUT /users/{email}/groups` | Replace user's groups | `UserHandler.ReplaceGroups()` |
| `DELETE /users/{email}/groups` | Remove from groups | `UserHandler.RemoveFromGroups()` |
| `POST /users/{email}` | Add to multiple groups | `UserHandler.AddToGroups()` |
| `GET /service-users` | List service users | `ServiceUserHandler.List()` |
| `POST /service-users` | Create service user | `ServiceUserHandler.Create()` |
| `DELETE /service-users/{uid}` | Delete service user | `ServiceUserHandler.Delete()` |
| `GET /policies` | List policies | `PolicyHandler.List()` |
| `POST /policies` | Create policy | `PolicyHandler.Create()` |
| `DELETE /policies/{uuid}` | Delete policy | `PolicyHandler.Delete()` |
| `GET /bindings` | List bindings | `BindingHandler.List()` |
| `POST /bindings` | Create binding | `BindingHandler.Create()` |
| `DELETE /bindings` | Delete binding | `BindingHandler.Delete()` |
| `GET /boundaries` | List boundaries | `BoundaryHandler.List()` |
| `POST /boundaries` | Create boundary | `BoundaryHandler.Create()` |
| `DELETE /boundaries/{uuid}` | Delete boundary | `BoundaryHandler.Delete()` |
| `GET /limits` | List limits | `LimitsHandler.List()` |
| `GET /subscriptions` | List subscriptions | `SubscriptionHandler.List()` |
| `GET /environments` | List environments | `EnvironmentHandler.List()` |

### Bulk Operations

| Command | Description |
|---------|-------------|
| `bulk add-users-to-group` | Add users from file |
| `bulk remove-users-from-group` | Remove users from file |
| `bulk create-groups` | Create groups from file |
| `bulk create-bindings` | Create bindings from file |
| `bulk export-group-members` | Export group members |

### Export Operations

| Command | Description |
|---------|-------------|
| `export all` | Export all resources |
| `export group` | Export single group |
| `export policy` | Export single policy (with --as-template) |
| `export environments` | Export all environments |
| `export users` | Export all users (with --detailed enrichment) |
| `export bindings` | Export all bindings (with --detailed enrichment) |
| `export boundaries` | Export all boundaries (with --detailed enrichment) |
| `export service-users` | Export all service users |

### Analyze Operations

| Command | Description |
|---------|-------------|
| `analyze user-permissions` | Calculate user permissions |
| `analyze group-permissions` | Calculate group permissions |
| `analyze permissions-matrix` | Generate permissions matrix |
| `analyze policy` | Analyze policy permissions |
| `analyze least-privilege` | Least privilege compliance |
| `analyze effective-user` | Get user permissions via API |
| `analyze effective-group` | Get group permissions via API |

### Advanced Group Operations

| Command | Description |
|---------|-------------|
| `group clone SOURCE` | Clone group with optional members and policy bindings |
| `group setup` | One-step group provisioning from YAML/JSON policies file |

### Boundary Helpers

| Command | Description |
|---------|-------------|
| `boundary create-app-boundary NAME` | Create boundary scoped to app IDs (`shared:app-id IN/NOT IN`) |
| `boundary create-schema-boundary NAME` | Create boundary scoped to schema IDs (`settings:schemaId IN/NOT IN`) |

### Template Operations

| Command | Description |
|---------|-------------|
| `template list` | List built-in and custom templates |
| `template show NAME` | Display template content and variables |
| `template render NAME` | Render template with `--set` variables |
| `template apply NAME` | Render and create resource |
| `template save NAME` | Save custom template from file |
| `template delete NAME` | Delete custom template |
| `template path` | Show templates directory |

### Apply Command

| Command | Description |
|---------|-------------|
| `apply -f FILE` | Create resources from YAML/JSON with auto-detect kind, `--set` variables, multi-document support |

### Deferred (Not Planned)

| Feature | Description | Status |
|---------|-------------|--------|
| Caching | In-memory caching with TTL | Deferred — not needed for parity |
| Zones | Management zone listing via entities API | Deferred — legacy feature |
| Permission diff/gaps | Advanced analysis enhancements | Deferred — nice-to-have |
