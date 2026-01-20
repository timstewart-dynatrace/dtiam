# Policy Examples

This directory contains example IAM policies for common use cases.

## Policy Statement Syntax

Dynatrace policies use a simple statement syntax:

```
ALLOW <service>:<scope>:<permission>;
```

### Common Services

| Service | Description |
|---------|-------------|
| `settings` | Settings API (schemas, objects) |
| `storage` | Metrics, logs, traces storage |
| `environment` | Environment-level operations |
| `shared` | Shared resources (apps, dashboards) |

### Common Scopes

| Scope | Description |
|-------|-------------|
| `objects` | Settings objects |
| `schemas` | Settings schemas |
| `events` | Event data |
| `metrics` | Metric data |
| `logs` | Log data |

### Common Permissions

| Permission | Description |
|------------|-------------|
| `read` | Read access |
| `write` | Write/modify access |
| `delete` | Delete access |

## Examples

### Read-Only Policy

```yaml
# See viewer-policy.yaml
statement: |
  ALLOW settings:objects:read;
  ALLOW settings:schemas:read;
```

### DevOps Policy

```yaml
# See devops-policy.yaml
statement: |
  ALLOW settings:objects:read;
  ALLOW settings:objects:write;
  ALLOW settings:schemas:read;
```

### Schema-Restricted Policy

```yaml
# See alerting-only.yaml
# Restrict to specific schemas using conditions
statement: |
  ALLOW settings:objects:read WHERE settings:schemaId = "builtin:alerting.profile";
  ALLOW settings:objects:write WHERE settings:schemaId = "builtin:alerting.profile";
```

## Creating Policies

```bash
# Create a simple read-only policy
dtiam create policy \
  --name "viewer-policy" \
  --statement "ALLOW settings:objects:read; ALLOW settings:schemas:read;" \
  --description "Read-only access to settings"

# Create a policy with multiple permissions
dtiam create policy \
  --name "devops-policy" \
  --statement "ALLOW settings:objects:read; ALLOW settings:objects:write; ALLOW settings:schemas:read;" \
  --description "DevOps team permissions"
```

## Best Practices

1. **Start minimal**: Begin with the least permissions needed
2. **Use descriptive names**: Policy names should indicate their purpose
3. **Add descriptions**: Document what the policy allows
4. **Test first**: Use `--dry-run` to preview changes
5. **Use boundaries**: Combine with boundaries for scope restriction

## See Also

- [Dynatrace IAM Documentation](https://docs.dynatrace.com/docs/manage/access-control)
- [Command Reference](../../docs/COMMANDS.md)
