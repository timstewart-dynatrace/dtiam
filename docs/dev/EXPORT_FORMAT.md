# Export YAML/JSON Schema

This document defines the structure of data produced by dtiam's `export` subcommands. If `dtctl iam export` is implemented, this schema should be the compatibility target.

**Source:** `internal/commands/export/export.go` (901 lines)

---

## Export Commands

| Command | Output | Format Options |
|---------|--------|----------------|
| `export all` | Timestamped directory with one file per resource type | json, yaml, csv |
| `export group NAME` | Single group with optional members and bindings | json, yaml |
| `export policy NAME` | Single policy, optionally as Go template | json, yaml |
| `export environments` | All environments | json, yaml, csv |
| `export users` | All users, optionally with group enrichment | json, yaml, csv |
| `export bindings` | All bindings, optionally with name enrichment | json, yaml, csv |
| `export boundaries` | All boundaries, optionally with policy details | json, yaml, csv |
| `export service-users` | All service users | json, yaml, csv |

---

## `export all` — Directory Structure

Creates a timestamped directory:

```
dtiam-export-YYYYMMDD-HHMMSS/
    environments.{format}
    groups.{format}
    users.{format}
    policies.{format}
    bindings.{format}
    boundaries.{format}
    service-users.{format}
```

The format is specified by `--format` flag (default: `yaml`).

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `yaml` | Output format: json, yaml, csv |
| `--dir` | `-d` | `dtiam-export-{timestamp}` | Custom output directory |
| `--detailed` | | `false` | Include enriched data (members, policy names) |

---

## Resource Schemas

Each resource is exported as an array of objects. The field names match the Dynatrace API response format.

### Groups

```yaml
- uuid: "8f6e5d4c-3b2a-1098-7654-321fedcba098"
  name: "Production Admins"
  description: "Full access to production"
  owner: "LOCAL"
  createdAt: "2026-01-15T10:30:00Z"
  updatedAt: "2026-03-20T14:22:00Z"
  # When --detailed:
  memberCount: 5
  members:
    - uid: "user-uid-1"
      email: "alice@example.com"
    - uid: "user-uid-2"
      email: "bob@example.com"
```

### Users

```yaml
- uid: "john-uid-123"
  email: "john@example.com"
  name: "John Smith"
  surname: "Smith"
  # When --detailed:
  groups:
    - uuid: "group-uuid-1"
      name: "Admins"
    - uuid: "group-uuid-2"
      name: "DevOps"
  groupCount: 2
```

### Policies

```yaml
- uuid: "policy-uuid-456"
  name: "ReadOnly Access"
  description: "Read-only access to settings"
  statementQuery: "ALLOW settings:objects:read, settings:schemas:read"
  _level_type: "account"
  _level_id: "account-uuid"
```

Note the underscore-prefixed level fields — these come directly from the API.

### Bindings

```yaml
- groupUuid: "group-uuid-1"
  policyUuid: "policy-uuid-456"
  boundaries: null               # or ["boundary-uuid-1"]
  # When --detailed:
  groupName: "Production Admins"
  policyName: "ReadOnly Access"
```

### Boundaries

```yaml
- uuid: "boundary-uuid-789"
  name: "Production Only"
  description: "Restrict to production management zone"
  boundaryQuery: "environment:management-zone IN (\"Production\")"
  # When --detailed:
  attachedPolicies:
    - policyUuid: "policy-uuid-456"
      policyName: "ReadOnly Access"
```

### Environments

```yaml
- id: "abc123"
  name: "Production"
  state: "ACTIVE"
```

### Service Users

```yaml
- uid: "dt0s01.XXXXXX"
  name: "CI Pipeline"
  description: "Automation service account"
  owner: "LOCAL"
  groups:
    - uuid: "group-uuid-1"
      name: "Automation"
```

---

## Single-Resource Export

### `export group NAME`

```yaml
uuid: "8f6e5d4c-3b2a-1098-7654-321fedcba098"
name: "Production Admins"
description: "Full access to production"
# With --include-members:
members:
  - uid: "user-uid-1"
    email: "alice@example.com"
# With --include-bindings:
bindings:
  - policyUuid: "policy-uuid-456"
    policyName: "ReadOnly Access"
    boundaries: ["boundary-uuid-789"]
```

### `export policy NAME`

Standard export:

```yaml
uuid: "policy-uuid-456"
name: "ReadOnly Access"
description: "Read-only access to settings"
statementQuery: "ALLOW settings:objects:read, settings:schemas:read"
```

With `--as-template`:

```yaml
name: "{{ .Name }}"
description: "{{ .Description }}"
statementQuery: "{{ .Statement }}"
```

This produces a Go template that can be rendered with dtiam's `template render` command using `--set` variables.

---

## CSV Format

For CSV exports, nested structures are flattened:

- Arrays are JSON-serialized: `["uuid-1","uuid-2"]`
- Maps are JSON-serialized: `{"key":"value"}`
- Scalar values are written directly

Header row is auto-generated from all unique keys across all records. Field order is non-deterministic (Go map iteration) — sort headers if stable output is required.

---

## `writeData` Helper

All export commands use a single `writeData()` function:

```go
func writeData(data []map[string]any, path, format string) error
```

- `json`: Uses `json.MarshalIndent` with 2-space indent
- `yaml`: Uses `yaml.Marshal`
- `csv`: Flattens nested structures, collects all unique fields for header, writes via `encoding/csv`

File permissions: `0644`

---

## Enrichment Logic

When `--detailed` is set, export commands make additional API calls to resolve UUIDs to human-readable names:

| Resource | Enrichment | Additional API Calls |
|----------|-----------|---------------------|
| Groups | Member list | `GET /groups/{uuid}/users` per group |
| Users | Group membership | `GET /users/{uid}` per user (extract groups) |
| Bindings | Group and policy names | `GET /groups/{uuid}` and `GET /policies/{uuid}` per binding |
| Boundaries | Attached policy info | `GET /bindings` then filter by boundary UUID |

Enrichment is optional because it significantly increases API call count and execution time for large accounts.
