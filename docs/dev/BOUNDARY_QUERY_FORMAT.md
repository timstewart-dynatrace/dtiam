# Boundary Query Construction

This document describes how dtiam constructs boundary query strings for the two convenience commands: `boundary create-app-boundary` and `boundary create-schema-boundary`.

**Source:** `internal/commands/boundary/boundary.go`, function `buildBoundaryQuery()`

---

## Query Format

```
{scope}:{attribute} {operator} ({quoted, comma-separated values})
```

### Components

| Component | Description | Examples |
|-----------|-------------|---------|
| Scope | The Dynatrace resource scope | `shared`, `settings`, `environment`, `storage` |
| Attribute | The specific attribute to filter | `app-id`, `schemaId`, `management-zone`, `dt.security_context` |
| Operator | `IN` (allow list) or `NOT IN` (deny list) | `IN`, `NOT IN` |
| Values | Double-quoted, comma-separated IDs | `("dynatrace.dashboards", "dynatrace.logs")` |

---

## buildBoundaryQuery Function

```go
func buildBoundaryQuery(prefix, operator string, ids []string) string {
    quoted := make([]string, len(ids))
    for i, id := range ids {
        quoted[i] = fmt.Sprintf("%q", id)
    }
    return fmt.Sprintf("%s %s (%s)", prefix, operator, strings.Join(quoted, ", "))
}
```

**Behavior:**
- Each ID is wrapped in double quotes using Go's `%q` format verb
- IDs are comma-separated with a space after each comma
- The result is a single-line string

**Example inputs/outputs:**

| prefix | operator | ids | output |
|--------|----------|-----|--------|
| `shared:app-id` | `IN` | `["dynatrace.dashboards"]` | `shared:app-id IN ("dynatrace.dashboards")` |
| `shared:app-id` | `NOT IN` | `["dynatrace.dashboards", "dynatrace.logs"]` | `shared:app-id NOT IN ("dynatrace.dashboards", "dynatrace.logs")` |
| `settings:schemaId` | `IN` | `["builtin:alerting.profile", "builtin:alerting.maintenance-window"]` | `settings:schemaId IN ("builtin:alerting.profile", "builtin:alerting.maintenance-window")` |

---

## App Boundary Command

**Command:** `dtiam boundary create-app-boundary NAME --app-ids <ids> [--not-in] [--environment <env>]`

### Construction

1. Parse `--app-ids` by splitting on commas and trimming whitespace
2. Set operator: `IN` (default) or `NOT IN` (if `--not-in` flag)
3. Call `buildBoundaryQuery("shared:app-id", operator, appIDs)`
4. Optionally validate each app ID against the App Engine Registry API
5. Create boundary with the constructed query

### Validation

If `--environment` is provided and `--skip-validation` is not set:
- Creates an `AppHandler` pointed at `https://{environment}.apps.dynatrace.com`
- Calls `GET /platform/app-engine/registry/v1/apps/{appId}` for each app ID
- Fails with error if any app ID is not found
- Requires `DTIAM_ENVIRONMENT_URL` and `DTIAM_ENVIRONMENT_TOKEN`

### Examples

```bash
# Allow only dashboards and logs apps
dtiam boundary create-app-boundary "Dashboard Only" \
    --app-ids "dynatrace.dashboards,dynatrace.logs"
# Query: shared:app-id IN ("dynatrace.dashboards", "dynatrace.logs")

# Exclude smartscape
dtiam boundary create-app-boundary "No Smartscape" \
    --app-ids "dynatrace.classic.smartscape" --not-in
# Query: shared:app-id NOT IN ("dynatrace.classic.smartscape")

# With validation against an environment
dtiam boundary create-app-boundary "My Apps" \
    --app-ids "dynatrace.dashboards" --environment abc12345
```

---

## Schema Boundary Command

**Command:** `dtiam boundary create-schema-boundary NAME --schema-ids <ids> [--not-in] [--environment <env>]`

### Construction

1. Parse `--schema-ids` by splitting on commas and trimming whitespace
2. Set operator: `IN` (default) or `NOT IN` (if `--not-in` flag)
3. Call `buildBoundaryQuery("settings:schemaId", operator, schemaIDs)`
4. Optionally validate each schema ID against the Settings API
5. Create boundary with the constructed query

### Validation

If `--environment` is provided and `--skip-validation` is not set:
- Creates a `SchemaHandler` pointed at `https://{environment}.apps.dynatrace.com`
- Calls `GET /platform/classic/environment-api/v2/settings/schemas/{schemaId}` for each schema ID
- Fails with error if any schema ID is not found

### Examples

```bash
# Allow only alerting schemas
dtiam boundary create-schema-boundary "Alerting Only" \
    --schema-ids "builtin:alerting.profile,builtin:alerting.maintenance-window"
# Query: settings:schemaId IN ("builtin:alerting.profile", "builtin:alerting.maintenance-window")

# Exclude span schemas
dtiam boundary create-schema-boundary "No Spans" \
    --schema-ids "builtin:span-attribute,builtin:span-capture-rule" --not-in
# Query: settings:schemaId NOT IN ("builtin:span-attribute", "builtin:span-capture-rule")
```

---

## Management Zone Boundaries (Manual)

dtiam does NOT have a convenience command for management zone boundaries. They must be created manually using `dtiam create boundary` with the full query string:

```bash
dtiam create boundary "Production Only" --query \
    'environment:management-zone IN ("Production"); storage:dt.security_context IN ("Production"); settings:dt.security_context IN ("Production")'
```

### Why Three Lines?

Management zone boundaries require three scope lines to be complete:

| Line | Scope | Controls |
|------|-------|----------|
| `environment:management-zone IN ("X")` | Environment data | Monitoring data, entities |
| `storage:dt.security_context IN ("X")` | Storage layer | Log storage, metrics storage |
| `settings:dt.security_context IN ("X")` | Settings layer | Settings objects scoped to the zone |

Omitting any line creates a boundary that restricts some access but not all. This is a common mistake — dtctl should consider a `create-zone-boundary` convenience command that generates all three lines.

---

## Boundary Creation API Call

All boundary creation goes through:

```go
boundaryHandler.Create(ctx, name, managementZones, &boundaryQuery, &description)
```

Where:
- `name` — Boundary display name (required)
- `managementZones` — `[]string` of zone names (for zone boundaries only, nil for query boundaries)
- `boundaryQuery` — `*string` containing the full query (nil for zone-only boundaries)
- `description` — `*string` optional description

The API endpoint is `POST /iam/v1/repo/account/{uuid}/boundaries`.
