# dtiam - Dynatrace IAM CLI

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**. For official Dynatrace tools and support, please visit [dynatrace.com](https://www.dynatrace.com).

A kubectl-inspired command-line interface for managing Dynatrace Identity and Access Management resources.

## Features

- **kubectl-style commands** - Familiar syntax: `get`, `describe`, `create`, `delete`
- **Multi-context configuration** - Manage multiple Dynatrace accounts with named contexts
- **Rich output formats** - Table (default), JSON, YAML, CSV, and wide mode
- **Flexible authentication** - OAuth2 (recommended) or bearer token support
- **Bulk operations** - Process multiple resources from CSV/YAML files
- **Template system** - Variable substitution for manifests
- **Permissions analysis** - Calculate effective permissions for users and groups
- **Cross-platform** - Single binary for Linux, macOS, and Windows

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/timstewart-dynatrace/GO-dtiam.git
cd GO-dtiam

# Build the binary
make build

# Or install to $GOPATH/bin
make install

# Verify installation
dtiam --version
```

### From Releases

Download the latest release for your platform from the [releases page](https://github.com/timstewart-dynatrace/GO-dtiam/releases).

### Requirements

- Go 1.22+ (for building from source)
- Dynatrace Account with API access
- Authentication: OAuth2 client credentials (recommended) OR bearer token

## Authentication

dtiam supports two authentication methods. Choose based on your use case:

### Option 1: OAuth2 Client Credentials (Recommended)

**Best for:** Automation, scripts, long-running processes, CI/CD pipelines

**Advantages:**
- Tokens auto-refresh when expired
- Secure credential storage in config file
- Scoped permissions via OAuth2 client configuration

**Risks:**
- Client secret must be stored securely
- Requires creating an OAuth2 client in Dynatrace

```bash
# Add OAuth2 credentials
dtiam config set-credentials prod \
  --client-id YOUR_CLIENT_ID \
  --client-secret YOUR_CLIENT_SECRET

# Create a context
dtiam config set-context prod \
  --account-uuid YOUR_ACCOUNT_UUID \
  --credentials-ref prod

# Switch to the context
dtiam config use-context prod

# Or use environment variables
export DTIAM_CLIENT_ID="dt0s01.XXXXX"
export DTIAM_CLIENT_SECRET="dt0s01.XXXXX.YYYYY"
export DTIAM_ACCOUNT_UUID="abc-123-def"
```

### Option 2: Bearer Token (Static)

**Best for:** Quick testing, interactive sessions, debugging, integration with external token providers

**Advantages:**
- No OAuth2 client setup required
- Can use tokens from other systems
- Quick for one-off operations

**Risks:**
- ⚠️ **Tokens do NOT auto-refresh** - requests fail when token expires
- ⚠️ **Not suitable for automation** - requires manual token renewal
- Token expiration causes immediate failures with no recovery

```bash
# Set bearer token via environment variable
export DTIAM_BEARER_TOKEN="dt0c01.XXXXX.YYYYY..."
export DTIAM_ACCOUNT_UUID="abc-123-def"

# Run commands - token will be used until it expires
dtiam get groups
```

### Authentication Priority

When multiple authentication methods are configured, dtiam uses this priority:
1. `DTIAM_BEARER_TOKEN` + `DTIAM_ACCOUNT_UUID` (bearer token)
2. `DTIAM_CLIENT_ID` + `DTIAM_CLIENT_SECRET` + `DTIAM_ACCOUNT_UUID` (OAuth2 via env)
3. Config file context with OAuth2 credentials

## Quick Start

### 1. Set up credentials (OAuth2)

```bash
# Add OAuth2 credentials
dtiam config set-credentials prod \
  --client-id YOUR_CLIENT_ID \
  --client-secret YOUR_CLIENT_SECRET

# Create a context
dtiam config set-context prod \
  --account-uuid YOUR_ACCOUNT_UUID \
  --credentials-ref prod

# Switch to the context
dtiam config use-context prod

# Verify configuration
dtiam config view
```

### 2. List resources

```bash
# List all groups
dtiam get groups

# List policies
dtiam get policies

# List users
dtiam get users

# List environments
dtiam get environments
```

### 3. Get detailed information

```bash
# Describe a group (includes members and policies)
dtiam describe group "DevOps Team"

# Describe a policy (includes statements)
dtiam describe policy "admin-policy"

# Describe a user (includes group memberships)
dtiam describe user user@example.com
```

### 4. Create resources

```bash
# Create a group
dtiam create group --name "New Team" --description "A new team"

# Create a binding (assign policy to group)
dtiam create binding --group "New Team" --policy "viewer-policy"
```

## Commands

| Command | Description |
|---------|-------------|
| `config` | Manage configuration contexts and credentials |
| `get` | List/retrieve resources |
| `describe` | Show detailed resource information |
| `create` | Create resources |
| `delete` | Delete resources |
| `user` | User management operations |
| `service-user` | Service user (OAuth client) management |
| `account` | Account limits and subscriptions |
| `group` | Advanced group operations |
| `boundary` | Boundary attach/detach operations |
| `cache` | Cache management |
| `bulk` | Bulk operations from CSV/YAML/JSON files |
| `export` | Export resources for backup/migration |
| `analyze` | Permission analysis and compliance checks |

## Resources

| Resource | Description |
|----------|-------------|
| `groups` | IAM groups for organizing users |
| `policies` | Permission policies with statements |
| `users` | User accounts |
| `service-users` | Service users (OAuth clients) for automation |
| `bindings` | Policy-to-group assignments |
| `environments` | Dynatrace environments |
| `boundaries` | Scope restrictions for bindings |
| `limits` | Account limits and quotas |
| `subscriptions` | Account subscriptions |

## Global Options

```bash
dtiam [OPTIONS] COMMAND

Options:
  -c, --context TEXT    Override the current context
  -o, --output FORMAT   Output format: table, json, yaml, csv, wide
  -v, --verbose         Enable verbose/debug output
  --plain               Plain output mode (no colors, no prompts)
  --dry-run             Preview changes without applying them
  -V, --version         Show version and exit
  --help                Show help and exit
```

## Configuration

Configuration is stored at `~/.config/dtiam/config` (XDG Base Directory compliant).

```yaml
api-version: v1
kind: Config
current-context: production
contexts:
  - name: production
    context:
      account-uuid: abc-123-def
      credentials-ref: prod-creds
  - name: development
    context:
      account-uuid: xyz-789-uvw
      credentials-ref: dev-creds
credentials:
  - name: prod-creds
    credential:
      client-id: dt0s01.XXXXX
      client-secret: dt0s01.XXXXX.YYYYY
preferences:
  output: table
```

## Examples

### User Management

```bash
# Create a user
dtiam user create user@example.com --first-name John --last-name Doe

# Add user to groups
dtiam user add-to-groups user@example.com --groups "DevOps,Platform"

# List user's groups
dtiam user list-groups user@example.com

# Replace all group memberships
dtiam user replace-groups user@example.com --groups "NewTeam"
```

### Service User Management

```bash
# List service users
dtiam service-user list

# Create a service user (SAVE THE CREDENTIALS!)
dtiam service-user create --name "CI Pipeline" --description "CI/CD automation"

# Add to group
dtiam service-user add-to-group "CI Pipeline" --group DevOps

# List groups
dtiam service-user list-groups "CI Pipeline"
```

### Boundary Management

```bash
# Attach a boundary to a binding
dtiam boundary attach \
  --group "DevOps Team" \
  --policy "admin-policy" \
  --boundary "production-boundary"

# Detach a boundary
dtiam boundary detach \
  --group "DevOps Team" \
  --policy "admin-policy" \
  --boundary "production-boundary"

# List all bindings using a boundary
dtiam boundary list-attached "production-boundary"
```

### Account Information

```bash
# View account limits and quotas
dtiam account limits

# List subscriptions
dtiam account subscriptions

# Get usage forecast
dtiam account forecast
```

## Documentation

- [Quick Start Guide](docs/QUICK_START.md) - Detailed getting started guide
- [Command Reference](docs/COMMANDS.md) - Full command documentation
- [Architecture](docs/ARCHITECTURE.md) - Technical design and implementation
- [API Reference](docs/API_REFERENCE.md) - Programmatic usage

## Required OAuth2 Scopes

Your OAuth2 client needs specific scopes for each operation. Create your client at:
**Account Management → Identity & access management → OAuth clients**

### Scope Reference by Command

| Command | Operation | Required Scopes |
|---------|-----------|-----------------|
| **Groups** | | |
| `get groups` | List/get groups | `account-idm-read` |
| `create group` | Create group | `account-idm-write` |
| `delete group` | Delete group | `account-idm-write` |
| **Users** | | |
| `get users` | List/get users | `account-idm-read` |
| `user create` | Create user | `account-idm-write` |
| `user add-to-groups` | Add to groups | `account-idm-write` |
| **Service Users** | | |
| `service-user list` | List service users | `account-idm-read` |
| `service-user create` | Create service user | `account-idm-write` |
| `service-user delete` | Delete service user | `account-idm-write` |
| **Policies** | | |
| `get policies` | List/get policies | `iam-policies-management` or `iam:policies:read` |
| `create policy` | Create policy | `iam-policies-management` or `iam:policies:write` |
| `delete policy` | Delete policy | `iam-policies-management` or `iam:policies:write` |
| **Bindings** | | |
| `get bindings` | List bindings | `iam-policies-management` or `iam:bindings:read` |
| `create binding` | Create binding | `iam-policies-management` or `iam:bindings:write` |
| `delete binding` | Delete binding | `iam-policies-management` or `iam:bindings:write` |
| **Boundaries** | | |
| `get boundaries` | List/get boundaries | `iam-policies-management` or `iam:boundaries:read` |
| `create boundary` | Create boundary | `iam-policies-management` or `iam:boundaries:write` |
| `boundary attach/detach` | Modify bindings | `iam-policies-management` or `iam:bindings:write` |
| **Account** | | |
| `account limits` | Account limits | `account-idm-read` |
| `account subscriptions` | Subscriptions | Bearer token (auto) |
| **Environments** | | |
| `get environments` | List environments | `account-env-read` |

### Recommended Scope Sets

**Read-Only Access:**
```
account-idm-read
account-env-read
iam:policies:read
iam:bindings:read
iam:boundaries:read
```

**Full IAM Management:**
```
account-idm-read
account-idm-write
account-env-read
iam-policies-management
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DTIAM_BEARER_TOKEN` | Static bearer token (alternative to OAuth2) |
| `DTIAM_CLIENT_ID` | OAuth2 client ID |
| `DTIAM_CLIENT_SECRET` | OAuth2 client secret |
| `DTIAM_ACCOUNT_UUID` | Dynatrace account UUID |
| `DTIAM_CONTEXT` | Override current context name |
| `DTIAM_OUTPUT` | Default output format |
| `DTIAM_VERBOSE` | Enable verbose mode |

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint

# Install locally
make install

# Clean build artifacts
make clean
```

## Disclaimer

**USE AT YOUR OWN RISK.** This tool is provided "as-is" without any warranty of any kind, express or implied. The authors and contributors are not responsible for any damages or data loss that may result from using this tool.

**NOT PRODUCED BY DYNATRACE.** This is an independent, community-developed tool. It is not produced, endorsed, maintained, or supported by Dynatrace. For official Dynatrace products and support, visit [dynatrace.com](https://www.dynatrace.com).

**NO SUPPORT PROVIDED.** This tool is provided without support. Issues may be reported via GitHub, but there is no guarantee of response or resolution.

## License

MIT License - see LICENSE file for details.
