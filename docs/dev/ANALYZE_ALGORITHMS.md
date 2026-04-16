# Permission Analysis Algorithms

This document describes the client-side computation logic used by dtiam's `analyze` subcommands. These algorithms are the primary porting target for `dtctl iam analyze` — the logic must be reimplemented in dtctl's patterns, but the algorithms are identical.

**Source:** `internal/utils/permissions.go` (652 lines)

---

## Core Data Structures

### Permission

A single parsed permission from a policy `statementQuery`:

```
Permission {
    Effect:      "ALLOW" | "DENY"
    Action:      "settings:objects:read" | "account:users:write" | ...
    Description: Human-readable label (from hardcoded pattern map, or action string)
    Conditions:  Optional WHERE clause text
}
```

### EffectivePermission

An aggregated permission that tracks every source (group + policy) granting it:

```
EffectivePermission {
    Effect:      "ALLOW" | "DENY"
    Action:      "settings:objects:read"
    Description: "Read settings objects"
    Sources: [
        { Group: "Admins", Policy: "Full Access" },
        { Group: "DevOps", Policy: "Settings Reader" },
    ]
}
```

### PermissionSource

Where a permission came from:

```
PermissionSource {
    Group:    "Admins"           (empty for group-level analysis)
    Policy:   "Full Access"
    Boundary: "prod-boundary"    (empty if no boundary on binding)
}
```

---

## Algorithm 1: ParseStatementQuery

**Used by:** Every analyze command, every export with enrichment

**Input:** A policy's `statementQuery` string

**Output:** `[]Permission`

**Algorithm:**

1. Split the statement by `;` (policies can contain multiple semicolon-separated statements)
2. For each sub-statement, apply regex: `(?i)(ALLOW|DENY)\s+([^\s]+(?:\s*,\s*[^\s]+)*)\s*(?:WHERE\s+(.+))?`
3. Extract: effect (`ALLOW`/`DENY`), comma-separated action list, optional WHERE conditions
4. For each action, create a Permission:
   - Look up human-readable description from `permissionPatterns` map
   - If not found, use the raw action string as description
   - Attach conditions if present

**Example:**

```
Input:  "ALLOW settings:objects:read, settings:schemas:read WHERE environment = 'prod'; DENY account:users:write"

Output: [
    { Effect: "ALLOW", Action: "settings:objects:read",  Description: "Read settings objects", Conditions: "environment = 'prod'" },
    { Effect: "ALLOW", Action: "settings:schemas:read",  Description: "Read settings schemas", Conditions: "environment = 'prod'" },
    { Effect: "DENY",  Action: "account:users:write",    Description: "Write account users",   Conditions: "" },
]
```

**Known pattern map** (10 entries):

| Action | Description |
|--------|-------------|
| `settings:objects:read` | Read settings objects |
| `settings:objects:write` | Write settings objects |
| `settings:schemas:read` | Read settings schemas |
| `environment:roles:manage` | Manage environment roles |
| `account:users:read` | Read account users |
| `account:users:write` | Write account users |
| `account:groups:read` | Read account groups |
| `account:groups:write` | Write account groups |
| `account:policies:read` | Read account policies |
| `account:policies:write` | Write account policies |

This map is intentionally small. Most permissions use their action string as the description.

---

## Algorithm 2: User Effective Permissions (`analyze user-permissions`)

**Input:** User email or UID

**Output:** `UserEffectivePermissions` — user info, groups, bindings, deduplicated permissions with sources

**API calls made (in order):**

1. **Resolve user** — `GET /users/{uid}` or search by email via `GET /users` + filter
2. **Get user's groups** — `GET /users/{uid}` then extract `groups` array, OR separate groups API call
3. **For each group:** `GET /bindings` filtered by `groupUuid` (via `BindingHandler.GetForGroup`)
4. **For each binding:** `GET /policies/{policyUuid}` to fetch the policy's `statementQuery`

**Computation:**

```
For each group the user belongs to:
    For each binding attached to that group:
        Fetch the policy referenced by the binding
        Parse the policy's statementQuery into permissions
        For each permission:
            Key = "{Effect}:{Action}"
            If key already seen:
                Append this (group, policy) as an additional Source
            Else:
                Create new EffectivePermission with this source
```

**Deduplication key:** `"{Effect}:{Action}"` (e.g., `"ALLOW:settings:objects:read"`)

This means if two policies grant the same ALLOW on the same action through different groups, the permission appears once with two sources — not twice.

**Error handling:**
- If a binding's policy can't be fetched (deleted, permission denied): silently skip
- If a group's bindings can't be fetched: silently skip that group
- If user not found: return error "user not found: {id}"

**N+1 API call pattern:** This algorithm makes O(groups * bindings) API calls. For a user in 5 groups with 3 bindings each, that's ~20 API calls (5 group binding lookups + 15 policy fetches). There is no batching or caching.

---

## Algorithm 3: Group Effective Permissions (`analyze group-permissions`)

**Input:** Group UUID or name

**Output:** `GroupEffectivePermissions` — group info, bindings, deduplicated permissions with sources

**API calls made (in order):**

1. **Resolve group** — `GET /groups/{uuid}` or search by name via `GET /groups` + filter
2. **Get group bindings** — `GET /bindings` filtered by `groupUuid`
3. **For each binding:** `GET /policies/{policyUuid}`

**Computation:** Same deduplication as user permissions, but without the outer group loop. Sources track (policy, boundary) instead of (group, policy).

**This is a subset of Algorithm 2** — it's the inner loop extracted for a single group.

---

## Algorithm 4: Policy Permissions Matrix (`analyze permissions-matrix --type policies`)

**Input:** None (operates on all policies in the account)

**Output:** `MatrixResult` — list of all permissions, list of all policies, boolean matrix

**API calls made:**

1. **List all policies** — `GET /policies` (account level)
2. **For each policy:** `GET /policies/{uuid}` to get `statementQuery`

**Computation:**

```
allPermissions = set()
policyPermissions = map[policyName -> set(permissions)]

For each policy:
    Parse statementQuery into permissions
    For each permission:
        Key = "{Effect}:{Action}"
        Add to allPermissions
        Add to policyPermissions[policyName]

Build matrix:
    Rows = policies
    Columns = allPermissions
    Cell = true/false (does this policy grant this permission?)
```

**Output structure:**

```json
{
    "permissions": ["ALLOW:settings:objects:read", "ALLOW:account:users:write", ...],
    "policies": ["Full Access", "ReadOnly", ...],
    "matrix": [
        { "policy_name": "Full Access", "policy_uuid": "...", "ALLOW:settings:objects:read": true, "ALLOW:account:users:write": true },
        { "policy_name": "ReadOnly",    "policy_uuid": "...", "ALLOW:settings:objects:read": true, "ALLOW:account:users:write": false }
    ],
    "policy_count": 2,
    "permission_count": 2
}
```

This can be exported as CSV for spreadsheet analysis (`--export csv`).

---

## Algorithm 5: Group Permissions Matrix (`analyze permissions-matrix --type groups`)

**Input:** None (operates on all groups in the account)

**Output:** `MatrixResult` — same structure as policy matrix, but rows are groups

**API calls made:**

1. **List all groups** — `GET /groups`
2. **For each group:** `GET /bindings` filtered by groupUuid
3. **For each binding:** `GET /policies/{policyUuid}`

**Computation:** Same matrix construction as Algorithm 4, but:
- Rows = groups (not policies)
- Permissions are the union of all permissions from all policies bound to each group

**N+1 pattern:** This is the most API-intensive operation. For 20 groups with 5 bindings each, it's ~120 API calls (20 binding lookups + 100 policy fetches).

---

## Algorithm 6: Least-Privilege Analysis (`analyze least-privilege`)

**Input:** None (operates on all policies)

**Output:** Report of policies with broad permissions

**Note:** This is implemented in the command layer (`internal/commands/analyze/analyze.go`), not in `permissions.go`. The algorithm:

1. List all policies
2. Fetch each policy's `statementQuery`
3. Parse into permissions
4. Flag policies that have:
   - Wildcard permissions (actions containing `*`)
   - Unrestricted ALLOW statements (no WHERE clause)
   - Broad scope actions (e.g., `account:*:write`)

This is a client-side heuristic, not an API-provided analysis.

---

## Algorithm 7: Effective Permissions via API (`analyze effective-user`, `analyze effective-group`)

**Input:** User email/UID or group UUID/name, plus optional level-type, level-id, and service filters

**Output:** `APIEffectivePermissions` — what the Dynatrace platform actually computes

**This is NOT a client-side computation.** It calls the Dynatrace Resolution API:

```
GET /iam/v1/resolution/{levelType}/{levelId}/effectivepermissions
    ?entityId={uid/groupUuid}
    &entityType={user/group}
    &services={optional,comma,separated}
    &page={N}
    &size={100}
```

**Algorithm:**

1. Resolve identifier (email to UID, group name to UUID)
2. If `levelID` is empty, default to the account UUID
3. Paginate through all results (page-based pagination, 100 items per page)
4. Return aggregated list

**Pagination details:**
- The response includes a `total` field
- Loop increments `page` until `len(collected) >= total` or empty page
- Response can return items under either `effectivePermissions` or `items` key (the code checks both)

**Why this matters for dtctl:** Algorithms 2-5 are client-side approximations. Algorithm 7 shows what Dynatrace actually computes after applying all inheritance, overrides, and boundary scoping. Both should be offered — they serve different purposes (transparency vs authority).

---

## Key Implementation Notes for Porting

1. **No caching.** Every analyze command starts fresh. If a user asks for user-permissions and then permissions-matrix, the same policies are fetched twice. This is intentional — analysis results should reflect current state, not cached state.

2. **Error tolerance.** Algorithms 2-5 silently skip resources they can't fetch (permission denied, deleted mid-operation). This prevents one bad policy from breaking the entire analysis. dtctl should preserve this behavior.

3. **Deduplication key format.** The key is `"{Effect}:{Action}"`, not just the action. `ALLOW:settings:objects:read` and `DENY:settings:objects:read` are separate entries. This matters when policies both ALLOW and DENY the same action.

4. **No transitive group expansion.** Users belong to groups directly. There is no nested group membership in Dynatrace IAM. The algorithms don't need to handle group inheritance.

5. **Statement parsing is regex-based.** The parser handles `ALLOW`/`DENY` with comma-separated actions and optional `WHERE` clauses. It does NOT handle nested parentheses, subqueries, or complex logical operators in conditions — it captures the raw condition text as a string.

6. **Order instability.** The matrix and permission lists are built from Go maps, so iteration order is non-deterministic. Output order may vary between runs. If dtctl needs stable ordering, sort the results after collection.
