# Examples

This directory contains example configurations and scripts for dtiam.

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**.

## Directory Structure

```
examples/
├── README.md                    # This file
├── auth/                        # Authentication examples
│   └── .env.example             # Environment variable template
├── boundaries/                  # Policy boundary examples
│   ├── production-only.yaml     # Restrict to production zones
│   └── team-scoped.yaml         # Team-specific zone restrictions
├── bulk/                        # Bulk operation sample files
│   ├── sample_users.csv         # For bulk add-users-to-group
│   ├── sample_groups.yaml       # For bulk create-groups
│   └── sample_bindings.yaml     # For bulk create-bindings
├── config/                      # Configuration examples
│   └── multi-account.yaml       # Multi-account config template
├── groups/                      # Group configuration examples
│   ├── team-group.yaml          # Standard team group
│   ├── admin-group.yaml         # Administrator group
│   └── readonly-group.yaml      # Read-only access group
├── policies/                    # Policy examples
│   ├── README.md                # Policy documentation
│   ├── viewer-policy.yaml       # Read-only policy
│   ├── devops-policy.yaml       # DevOps permissions
│   └── alerting-only.yaml       # Schema-restricted policy
├── service-users/               # Service user (OAuth client) examples
│   ├── ci-pipeline.yaml         # CI/CD automation service user
│   └── monitoring-bot.yaml      # Read-only monitoring service user
├── templates/                   # Reusable templates
│   ├── group-team.yaml          # Team group template
│   ├── policy-readonly.yaml     # Read-only policy template
│   └── boundary-zone.yaml       # Zone boundary template
└── scripts/                     # Shell script examples
    └── common-workflows.sh      # Common workflow reference
```

## Quick Start

### Authentication Setup

1. Copy the environment template:
   ```bash
   cp examples/auth/.env.example .env
   ```

2. Edit `.env` with your credentials:
   ```bash
   DTIAM_CLIENT_ID=dt0s01.XXXXX
   DTIAM_CLIENT_SECRET=dt0s01.XXXXX.YYYYY
   DTIAM_ACCOUNT_UUID=abc-123-def
   ```

3. Source the environment:
   ```bash
   source .env
   ```

### Using Configuration Examples

The examples in this directory can be used as references for:

1. **Creating groups** with consistent naming and descriptions
2. **Creating policies** with proper permission statements
3. **Creating boundaries** with correct Dynatrace query format
4. **Setting up service users** for automation

### Example: Create a Team Group

```bash
# View the example
cat examples/groups/team-group.yaml

# Create the group
dtiam create group --name "Platform Team" --description "Platform engineering team"
```

### Example: Create a Policy Binding

```bash
# Create the binding
dtiam create binding --group "Platform Team" --policy "viewer-policy"
```

## File Formats

### YAML Examples

YAML files in this directory follow the pattern:

```yaml
# Resource type and metadata
kind: Group
name: team-name
description: Team description

# Resource-specific fields
# ...
```

### CSV Examples (for bulk operations)

CSV files use headers matching the required fields:

```csv
group_name,policy_name,level,level_id,management_zones,boundary_name,description
LOB5,Standard User,account,,,,LOB5 team
```

## Best Practices

1. **Use descriptive names**: Group and policy names should clearly indicate their purpose
2. **Add descriptions**: Always include descriptions for documentation
3. **Follow least privilege**: Start with minimal permissions and add as needed
4. **Use boundaries**: Restrict scope with management zone boundaries where appropriate
5. **Version control**: Keep your actual configurations in version control

## See Also

- [Command Reference](../docs/COMMANDS.md)
- [Quick Start Guide](../docs/QUICK_START.md)
- [Architecture](../docs/ARCHITECTURE.md)
