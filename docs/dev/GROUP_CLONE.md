# Group Clone Algorithm

This document describes the compound operation performed by `dtiam group clone`, which creates a copy of an existing group with optional member and policy binding duplication.

**Source:** `internal/commands/group/group.go`, `cloneCmd` (lines ~230-336)

---

## Command Signature

```
dtiam group clone SOURCE --name "New Name" [--description "..."] [--include-members] [--include-policies] [--dry-run]
```

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--name` | `-n` | Yes | Name for the new group |
| `--description` | `-d` | No | Description for the new group |
| `--include-members` | | No | Copy all members from source to new group |
| `--include-policies` | | No | Copy all policy bindings (with boundaries) from source to new group |

`SOURCE` can be a group UUID or name (resolved via `GetOrResolve`).

---

## Algorithm

### Step 0: Dry-Run Check

If `--dry-run` is set:
- Print what would be created (group name, source details)
- Print member count and binding count if `--include-members` / `--include-policies`
- Return without making any API calls (no client created)

### Step 1: Resolve Source Group

```
API: GetOrResolve(ctx, groupHandler, SOURCE)
    → tries GET /groups/{SOURCE} (if UUID-shaped)
    → falls back to GET /groups, then filter by name
```

Extract source UUID from resolved group.

### Step 2: Create New Group

```
API: POST /groups
Body: { "name": "{--name}", "description": "{--description or source description}" }
```

Extract new group UUID from response.

**Output:** `Group "New Name" created (UUID: xxx)`

### Step 3: Copy Members (if --include-members)

```
API: GET /groups/{sourceUUID}/users    → get member list
For each member:
    API: POST /groups/{newUUID}/users   → add member by email
```

**Error handling per member:**
- If adding a member fails: print warning to stderr, continue with next member
- Does NOT roll back the group or previously added members
- Reports total count at end: `Copied N member(s)`

**Verbose output:** Each successfully added member is logged to stderr.

### Step 4: Copy Policy Bindings (if --include-policies)

```
API: GET /bindings (filtered by groupUuid=sourceUUID)    → get source bindings
For each binding:
    Extract policyUuid and boundaries array
    API: POST /bindings
    Body: { "groupUuid": "{newUUID}", "policyUuid": "{policyUuid}", "boundaries": [...] }
```

**Error handling per binding:**
- If copying a binding fails: print warning to stderr, continue with next
- Does NOT roll back previously created bindings
- Reports total count at end: `Copied N binding(s)`

**Verbose output:** Each successfully copied binding is logged to stderr.

---

## Operation Sequence Diagram

```
                        clone "Source Group" --name "New Group"
                                        |
                                   [Dry Run?]
                                  /          \
                               Yes            No
                              /                \
                    Print preview         Resolve source group
                    Return nil                  |
                                      Create new group (API)
                                                |
                                    [--include-members?]
                                      /              \
                                   Yes                No
                                  /                    \
                       Get source members          [--include-policies?]
                       For each member:               /              \
                         Add to new group           Yes                No
                         (warn on failure)          /                   \
                                |           Get source bindings      Return
                    [--include-policies?]    For each binding:
                      /              \        Create for new group
                   Yes                No      (warn on failure)
                  /                    \            |
       Get source bindings          Return      Return
       For each binding:
         Create for new group
         (warn on failure)
               |
            Return
```

---

## Error Handling Strategy

The algorithm uses a **best-effort, no-rollback** approach:

| Failure Point | Behavior |
|--------------|----------|
| Source group not found | Return error, abort |
| New group creation fails | Return error, abort |
| Individual member add fails | Print warning, continue with next member |
| Individual binding copy fails | Print warning, continue with next binding |

**There is no rollback.** If the clone partially succeeds (group created, 3 of 5 members added, then fails), the partially-cloned group remains. This is intentional — partial results are more useful than atomic rollback for IAM operations where re-running is safe.

**Idempotency note:** Adding a member who is already in the group is a no-op (not an error). Creating a binding that already exists returns 409 Conflict. Neither of these should cause issues on retry.

---

## API Call Count

| Scenario | API Calls |
|----------|-----------|
| Clone with no flags | 2 (resolve source + create group) |
| Clone with --include-members | 2 + 1 + N (resolve + create + get members + N member adds) |
| Clone with --include-policies | 2 + 1 + M (resolve + create + get bindings + M binding creates) |
| Clone with both flags | 2 + 1 + N + 1 + M (all of the above) |

For a group with 50 members and 10 bindings, that's ~64 API calls.

---

## Group Setup Command (Related)

`dtiam group setup` is a related but different compound operation:

```
dtiam group setup --name "New Team" --policies-file policies.yaml
```

**Algorithm:**
1. Read YAML/JSON file containing policy definitions
2. Create new group
3. For each policy in file: resolve by name/UUID, then create binding

**File format:**

```yaml
policies:
  - name: "ReadOnly Policy"
    boundaries:
      - "boundary-uuid-1"
  - name: "Admin Policy"
  - uuid: "policy-uuid-direct"
```

Each policy can be specified by `name` (resolved at runtime) or `uuid` (used directly). Optional `boundaries` array attaches boundary UUIDs to the binding.

**Difference from clone:** Setup creates a group from a declarative file. Clone copies an existing group's actual state. Setup is reproducible across accounts; clone is specific to the source account.
