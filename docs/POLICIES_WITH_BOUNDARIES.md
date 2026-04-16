# Effective Policies with Boundaries — Resolution Rules

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**.

When a policy binding carries one or more boundaries, the *effective* set of ALLOW statements is derived by combining each policy statement with the boundary's conditions according to a fixed set of rules. This document describes those rules and walks through worked examples.

The rules below are authoritative per Dynatrace's policy-evaluation engine. Effective permissions as computed by `dtiam analyze user-permissions` and `dtiam analyze permissions-matrix` follow them verbatim.

## Resolution Rules

1. **Service-scoped conditions only apply to matching permissions.** Only conditions whose prefix matches the service configuration of a permission are applied to that permission. Exception: `global:` conditions apply to *every* permission in the policy.
2. **Repeated condition names multiply statements.** When a boundary lists the same condition name more than once, each repeat produces a separate effective statement for every applicable policy statement.
3. **Multiple boundaries evaluate independently.** When more than one boundary is attached to a binding, the effective statements are computed for each boundary separately, then unioned.
4. **Multi-permission statements are split.** A single policy statement that grants multiple permissions is decomposed into one effective statement per permission before boundary conditions are applied.
5. **Boundaries stack on existing `WHERE` clauses unconditionally.** A boundary is applied even when a policy statement already contains the same condition — the effective `WHERE` clause is the `AND` of both.
6. **Boundaries do not apply to `DENY` statements.** A `DENY` is copied through verbatim regardless of attached boundaries.

## Examples

### Boundary with conditions not found in service configuration

Boundary

```
settings:schemaId = "builtin:maintenance-windows";
settings:not-found-in-service-configuration = "1";
```

Policy

```
ALLOW settings:objects:read;
```

Effective policy

```
ALLOW settings:objects:read WHERE settings:schemaId = "builtin:maintenance-windows";
```

The `settings:not-found-in-service-configuration` condition is silently dropped because the `settings:*` permissions do not accept it per service configuration.

### Boundary with a global condition

Boundary

```
settings:schemaId = "builtin:maintenance-windows";
global:week-day = "Monday";
```

Policy

```
ALLOW settings:objects:read;
ALLOW app-engine:apps:run;
```

Effective policy

```
ALLOW settings:objects:read WHERE settings:schemaId = "builtin:maintenance-windows" AND global:week-day = "Monday";
ALLOW app-engine:apps:run WHERE global:week-day = "Monday";
```

`global:week-day` attaches to both permissions; `settings:schemaId` attaches only to the `settings:*` permission.

### Boundary contains repeated condition names

Boundary

```
settings:schemaId = "builtin:maintenance-windows";
settings:schemaId startsWith "custom";
settings:objectId = "1";
```

Policy

```
ALLOW settings:objects:read WHERE global:week-day = "Monday";
```

Effective policy

```
ALLOW settings:objects:read WHERE global:week-day = "Monday" AND settings:schemaId = "builtin:maintenance-windows" AND settings:objectId = "1";
ALLOW settings:objects:read WHERE global:week-day = "Monday" AND settings:schemaId startsWith "custom" AND settings:objectId = "1";
```

The repeated `settings:schemaId` produces two distinct effective statements.

### Many boundaries apply to a single policy

Boundary 1

```
settings:schemaId = "builtin:maintenance-windows";
settings:schemaId startsWith "custom";
settings:objectId = "1";
```

Boundary 2

```
settings:schemaId startsWith "dynatrace";
settings:objectId = "4";
```

Policy

```
ALLOW settings:objects:read WHERE global:week-day = "Monday";
ALLOW app-engine:apps:run;
ALLOW settings:objects:read;
```

Effective policy

```
// Statements calculated for Boundary 1
ALLOW settings:objects:read WHERE global:week-day = "Monday" AND settings:schemaId = "builtin:maintenance-windows" AND settings:objectId = "1";
ALLOW settings:objects:read WHERE global:week-day = "Monday" AND settings:schemaId startsWith "custom" AND settings:objectId = "1";
ALLOW settings:objects:read WHERE settings:schemaId = "builtin:maintenance-windows" AND settings:objectId = "1";
ALLOW settings:objects:read WHERE settings:schemaId startsWith "custom" AND settings:objectId = "1";

// Statements calculated for Boundary 2
ALLOW settings:objects:read WHERE global:week-day = "Monday" AND settings:schemaId startsWith "dynatrace" AND settings:objectId = "4";
ALLOW settings:objects:read WHERE settings:schemaId startsWith "dynatrace" AND settings:objectId = "4";

// app-engine:apps:run carries no service-matching conditions from either boundary
ALLOW app-engine:apps:run;
```

### Policy statement with multiple permissions

Boundary

```
settings:schemaId = "builtin:maintenance-windows";
settings:objectId = "4";
app-engine:appId = "application-id";
```

Policy

```
ALLOW settings:objects:read, settings:objects:write, app-engine:apps:run;
```

Effective policy

```
ALLOW settings:objects:read WHERE settings:schemaId = "builtin:maintenance-windows" AND settings:objectId = "4";
ALLOW settings:objects:write WHERE settings:schemaId = "builtin:maintenance-windows" AND settings:objectId = "4";
ALLOW app-engine:apps:run WHERE app-engine:appId = "application-id";
```

Each permission receives only the conditions whose service prefix matches it.

### Boundary has the same condition as in the policy

Boundary

```
settings:schemaId = "builtin:maintenance-windows";
```

Policy

```
ALLOW settings:objects:read WHERE settings:schemaId startsWith "test";
```

Effective policy

```
ALLOW settings:objects:read WHERE settings:schemaId startsWith "test" AND settings:schemaId = "builtin:maintenance-windows";
```

Both clauses are `AND`-combined — the policy's own condition is not replaced by the boundary's.

### Policy with a DENY statement

Boundary

```
settings:schemaId = "builtin:maintenance-windows";
```

Policy

```
DENY settings:objects:read;
```

Effective policy

```
DENY settings:objects:read;
```

`DENY` is copied through verbatim; boundaries never narrow a `DENY`.

## Working with Boundaries in dtiam

| Goal | Command |
|------|---------|
| List all boundaries at a level | `dtiam get boundaries --level account` |
| View a boundary's conditions | `dtiam describe boundary NAME` |
| Create a management-zone boundary | `dtiam boundary create-mz-boundary NAME --zones Zone1,Zone2` |
| Create an app-id boundary | `dtiam boundary create-app-boundary NAME --app-ids app.a,app.b` |
| Create a Settings-schema boundary | `dtiam boundary create-schema-boundary NAME --schema-ids builtin:...` |
| Attach a boundary to a binding | `dtiam boundary attach BOUNDARY --binding BINDING_ID` |
| Detach a boundary from a binding | `dtiam boundary detach BOUNDARY --binding BINDING_ID` |
| Calculate effective permissions for a user | `dtiam analyze user-permissions EMAIL` |
| Generate a permissions matrix | `dtiam analyze permissions-matrix` |

The `analyze` commands resolve effective policies using the rules above, so their output matches what the Dynatrace platform actually enforces.

## See Also

- [Command Reference](COMMANDS.md)
- [Architecture](ARCHITECTURE.md)
- [API Reference](API_REFERENCE.md)
