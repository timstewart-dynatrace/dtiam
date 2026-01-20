# CLAUDE.md

This file provides guidance for AI agents working with the dtiam codebase.

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**.

## Development Workflow - MANDATORY

**ALL development work MUST follow this workflow:**

### Branching Requirements

1. **NEVER commit features directly to main**
   - ALL new features, enhancements, and non-trivial changes MUST be developed in a feature branch
   - Branch naming convention: `feature/descriptive-name` or `fix/descriptive-name`
   - Only documentation fixes and critical hotfixes may be committed directly to main (with approval)

2. **Feature Branch Workflow**
   ```bash
   # Create feature branch from main
   git checkout main
   git pull
   git checkout -b feature/my-feature

   # Develop and commit
   git add <files>
   git commit -m "feat: description"

   # Push feature branch
   git push -u origin feature/my-feature
   ```

3. **Documentation Requirements - MANDATORY**
   - **ALL features MUST be documented BEFORE merging to main**
   - Documentation checklist (ALL must be completed):
     - [ ] [CLAUDE.md](CLAUDE.md) - Add to project structure, patterns, or API endpoints
     - [ ] [docs/COMMANDS.md](docs/COMMANDS.md) - Full command reference with examples
     - [ ] [README.md](README.md) - Update quick start or features section
     - [ ] [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - Update if architecture changes
     - [ ] [examples/](examples/) - Add sample files if applicable
     - [ ] Code comments for new functions/types

4. **Merge Process**
   ```bash
   # Before merging: verify ALL documentation is complete
   git checkout main
   git merge feature/my-feature --no-ff

   # If documentation is missing, DO NOT MERGE
   # Create documentation commits in the feature branch first
   ```

5. **Verification Before Merge**
   - Run tests: `make test`
   - Run linter: `make lint`
   - Verify command help: `dtiam <new-command> --help`
   - Check all documentation files are updated
   - Ensure examples are provided
   - Verify CLAUDE.md includes new patterns/endpoints

### Why This Matters

- **Prevents incomplete features in main**: Feature branches isolate work-in-progress
- **Ensures documentation completeness**: No undocumented features reach users
- **Enables easy rollback**: Feature branches can be deleted if not needed
- **Maintains clean history**: Main branch only contains complete, documented features
- **Facilitates collaboration**: Multiple features can be developed in parallel

### Example: Adding a New Resource

```bash
# 1. Create feature branch
git checkout -b feature/add-apps-resource

# 2. Implement feature
# - Add internal/resources/apps.go
# - Add command in internal/commands/get/get.go
# - Add output columns in internal/output/columns.go

# 3. Test implementation
make build
./bin/dtiam get apps --help

# 4. Document EVERYTHING
# - Update CLAUDE.md (project structure)
# - Update docs/COMMANDS.md (command reference)
# - Update README.md (add to resources table)
# - Update docs/ARCHITECTURE.md (add to resource handlers)
# - Add examples/apps/ directory with samples

# 5. Commit feature and documentation together
git add .
git commit -m "feat: add apps resource for App Engine Registry

- Add AppHandler for App Engine Registry API
- Add get apps command with --environment option
- Add AppColumns() for table output
- Document in CLAUDE.md, COMMANDS.md, README.md, ARCHITECTURE.md
- Add usage examples"

# 6. Push feature branch
git push -u origin feature/add-apps-resource

# 7. Merge to main (only after ALL documentation complete)
git checkout main
git merge feature/add-apps-resource --no-ff
git push
```

**REMEMBER: Documentation is NOT optional. It is MANDATORY before merge.**

### Version Management - MANDATORY

**ALL merges to main that add features or fixes MUST increment the version number.**

Current version: **1.0.0** (defined in `pkg/version/version.go` and set via ldflags)

#### Semantic Versioning (SemVer)

We follow [Semantic Versioning 2.0.0](https://semver.org/):

**Format:** `MAJOR.MINOR.PATCH` (e.g., 1.0.0)

1. **MAJOR version** (X.0.0) - Incompatible API changes
   - Breaking changes to CLI commands
   - Removal of commands or options
   - Changes that break existing scripts/workflows
   - Example: Removing `--zone` flag, changing command structure

2. **MINOR version** (1.X.0) - New features (backwards-compatible)
   - New commands (e.g., `get apps`)
   - New options to existing commands
   - New resource handlers
   - Example: Adding `bulk create-groups-with-policies`

3. **PATCH version** (1.0.X) - Bug fixes (backwards-compatible)
   - Bug fixes
   - Documentation updates
   - Performance improvements
   - Example: Fixing error handling, updating help text

#### When to Increment

**Before merging to main:**

```bash
# For new features (MINOR)
# 1.0.0 -> 1.1.0
git checkout feature/my-feature
# Edit pkg/version/version.go: Version = "1.1.0"
git add pkg/version/version.go
git commit -m "chore: bump version to 1.1.0"

# For bug fixes (PATCH)
# 1.0.0 -> 1.0.1
git checkout fix/my-bugfix
# Edit pkg/version/version.go: Version = "1.0.1"
git add pkg/version/version.go
git commit -m "chore: bump version to 1.0.1"

# Then merge to main
git checkout main
git merge feature/my-feature --no-ff
```

#### Version Bump Checklist

Before merging to main, ensure:
- [ ] Version incremented in `pkg/version/version.go`
- [ ] CHANGELOG.md updated with changes
- [ ] Correct increment type (MAJOR/MINOR/PATCH)
- [ ] Version bump committed in feature branch before merge

#### Version Display

Users can check the version:
```bash
dtiam --version
# Output: dtiam version 1.0.0
```

**REMEMBER: Version increments are MANDATORY for all feature and fix merges to main.**

### CHANGELOG Management - MANDATORY

**ALL changes MUST be documented in CHANGELOG.md**

We follow [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

#### CHANGELOG Structure

```markdown
## [Unreleased]

### Added
- New features go here

### Changed
- Changes to existing functionality

### Deprecated
- Features marked for removal

### Removed
- Removed features

### Fixed
- Bug fixes

### Security
- Security fixes

## [1.1.0] - 2025-01-20

### Added
- Actual released features
...
```

#### When to Update CHANGELOG

**In your feature branch, BEFORE merging:**

1. **For new features** - Add to `## [Unreleased]` → `### Added` section
2. **For changes** - Add to `## [Unreleased]` → `### Changed` section
3. **For bug fixes** - Add to `## [Unreleased]` → `### Fixed` section
4. **For documentation** - Add to `## [Unreleased]` → `### Documentation` section (optional)

#### CHANGELOG Checklist

Before merging to main:
- [ ] CHANGELOG.md updated with your changes
- [ ] Changes in appropriate section (Added/Changed/Fixed/etc)
- [ ] [Unreleased] section moved to version section
- [ ] Version number matches pkg/version/version.go
- [ ] Comparison links updated at bottom
- [ ] Date added to version heading (YYYY-MM-DD)

### Creating GitHub Releases

**After merging to main with version bump:**

```bash
# 1. Create git tag
git tag -a v1.1.0 -m "Release version 1.1.0"
git push origin v1.1.0

# 2. Create GitHub Release (web UI or CLI)
gh release create v1.1.0 \
  --title "v1.1.0" \
  --notes-file <(sed -n '/## \[1.1.0\]/,/## \[1.0.0\]/p' CHANGELOG.md | head -n -1)
```

**REMEMBER: CHANGELOG updates are MANDATORY for all merges to main.**

## Project Overview

**dtiam** is a kubectl-inspired CLI for managing Dynatrace Identity and Access Management (IAM) resources. It provides a consistent interface for managing groups, users, policies, bindings, boundaries, environments, and service users.

**Language:** Go 1.22+

## Quick Reference

### Build & Run

```bash
# Build the CLI
make build

# Install locally
make install

# Run CLI
./bin/dtiam --help
dtiam --help  # if installed

# Run with verbose output
dtiam -v get groups

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Project Structure

```
dtiam/
├── cmd/dtiam/main.go                 # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                   # Root command, global flags
│   │   └── state.go                  # Global state (context, output, verbose)
│   ├── commands/
│   │   ├── common/                   # Shared command utilities
│   │   ├── config/                   # Config management commands
│   │   ├── get/                      # List/retrieve resources
│   │   ├── describe/                 # Detailed resource views
│   │   ├── create/                   # Create resources
│   │   ├── delete/                   # Delete resources
│   │   ├── user/                     # User lifecycle commands
│   │   ├── serviceuser/              # OAuth client management
│   │   ├── group/                    # Advanced group ops
│   │   ├── boundary/                 # Boundary attach/detach
│   │   ├── account/                  # Limits and subscriptions
│   │   └── cache/                    # Cache management
│   ├── config/
│   │   ├── config.go                 # Config structs
│   │   └── loader.go                 # Config load/save, XDG paths
│   ├── client/
│   │   ├── client.go                 # HTTP client with retry
│   │   └── errors.go                 # APIError type
│   ├── auth/
│   │   ├── auth.go                   # TokenProvider interface
│   │   ├── oauth.go                  # OAuth2 token manager
│   │   └── bearer.go                 # Static bearer token
│   ├── resources/
│   │   ├── handler.go                # Handler interfaces
│   │   ├── groups.go                 # GroupHandler
│   │   ├── users.go                  # UserHandler
│   │   ├── policies.go               # PolicyHandler
│   │   ├── bindings.go               # BindingHandler
│   │   ├── boundaries.go             # BoundaryHandler
│   │   ├── environments.go           # EnvironmentHandler
│   │   ├── serviceusers.go           # ServiceUserHandler
│   │   ├── limits.go                 # LimitsHandler
│   │   └── subscriptions.go          # SubscriptionHandler
│   ├── output/
│   │   ├── format.go                 # Format enum
│   │   ├── printer.go                # Unified Printer
│   │   ├── table.go                  # Table formatter
│   │   └── columns.go                # Column definitions
│   └── utils/
│       └── resolver.go               # Name-to-UUID resolution
├── pkg/version/version.go            # Version info
├── go.mod
├── Makefile
└── .goreleaser.yaml
```

## Authentication

dtiam supports two authentication methods:

### OAuth2 (Recommended)
- Auto-refreshes tokens when expired
- Best for automation, CI/CD, long-running processes
- Requires `DTIAM_CLIENT_ID`, `DTIAM_CLIENT_SECRET`, `DTIAM_ACCOUNT_UUID`

### Bearer Token (Static)
- Does NOT auto-refresh (fails when token expires)
- Best for quick testing, debugging, one-off operations
- Requires `DTIAM_BEARER_TOKEN`, `DTIAM_ACCOUNT_UUID`

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DTIAM_BEARER_TOKEN` | Static bearer token (alternative to OAuth2) |
| `DTIAM_CLIENT_ID` | OAuth2 client ID (optional - auto-extracted from secret) |
| `DTIAM_CLIENT_SECRET` | OAuth2 client secret (format: dt0s01.CLIENTID.SECRET) |
| `DTIAM_ACCOUNT_UUID` | Dynatrace account UUID |
| `DTIAM_CONTEXT` | Override current context |
| `DTIAM_ENVIRONMENT_URL` | Environment URL for App Engine Registry (e.g., abc12345.apps.dynatrace.com) |

**Note:** `DTIAM_CLIENT_ID` is optional. If not set, it will be automatically extracted from
`DTIAM_CLIENT_SECRET` since Dynatrace secrets follow the format `dt0s01.CLIENTID.SECRETPART`.

## Key Patterns

### Filtering Resources

All `get` commands support **partial text matching** via `--name` (or `--email` for users).

**Filter Implementation Pattern:**

```go
// In get commands, apply client-side filtering after fetching list
results, _ := handler.List(ctx, nil)
if name != "" {
    filtered := make([]map[string]any, 0)
    for _, r := range results {
        if n, ok := r["name"].(string); ok && strings.Contains(strings.ToLower(n), strings.ToLower(name)) {
            filtered = append(filtered, r)
        }
    }
    results = filtered
}
printer.Print(results, columns)
```

**Filter Options by Command:**

| Command | Filter | Description |
|---------|--------|-------------|
| `get groups` | `--name` | Filter by group name |
| `get users` | `--email` | Filter by email address |
| `get policies` | `--name` | Filter by policy name |
| `get boundaries` | `--name` | Filter by boundary name |
| `get environments` | `--name` | Filter by environment name |
| `get service-users` | `--name` | Filter by service user name |

**Filter Behavior:**
- **Case-insensitive**: `--name prod` matches "Production", "NonProd"
- **Substring match**: `--name LOB` matches "LOB5", "LOB6", "MyLOBTeam"
- **Client-side**: Filters are applied after fetching full list from API

### Adding a New Command

1. Create command file in `internal/commands/<name>/<name>.go`:
```go
package newfeature

import (
    "context"
    "github.com/spf13/cobra"
    "github.com/jtimothystewart/dtiam/internal/cli"
    "github.com/jtimothystewart/dtiam/internal/commands/common"
)

var Cmd = &cobra.Command{
    Use:   "new-feature",
    Short: "New feature operations",
}

func init() {
    Cmd.AddCommand(doSomethingCmd)
}

var doSomethingCmd = &cobra.Command{
    Use:   "do-something NAME",
    Short: "Do something useful",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        c, err := common.CreateClient()
        if err != nil {
            return err
        }
        defer c.Close()

        printer := cli.GlobalState.NewPrinter()
        ctx := context.Background()

        // Implementation here
        return nil
    },
}
```

2. Register in `cmd/dtiam/main.go`:
```go
import "github.com/jtimothystewart/dtiam/internal/commands/newfeature"

func main() {
    cli.AddCommand(newfeature.Cmd)
    // ...
}
```

### Adding a New Resource Handler

1. Create handler in `internal/resources/<name>.go`:
```go
package resources

import "github.com/jtimothystewart/dtiam/internal/client"

type NewResourceHandler struct {
    BaseHandler
}

func NewNewResourceHandler(c *client.Client) *NewResourceHandler {
    return &NewResourceHandler{
        BaseHandler: BaseHandler{
            Client:    c,
            Name:      "new-resource",
            Path:      "/new-resources",
            ListKey:   "items",
            IDField:   "uuid",
            NameField: "name",
        },
    }
}
```

2. Add columns in `internal/output/columns.go`:
```go
func NewResourceColumns() []Column {
    return []Column{
        {Key: "uuid", Header: "UUID"},
        {Key: "name", Header: "NAME"},
        {Key: "description", Header: "DESCRIPTION"},
    }
}
```

### Boundary Query Format

Boundaries use the following Dynatrace-compliant format:

**Management Zone Boundaries:**
```
# Single zone
environment:management-zone IN ("Production");
storage:dt.security_context IN ("Production");
settings:dt.security_context IN ("Production")

# Multiple zones
environment:management-zone IN ("Production", "Staging");
storage:dt.security_context IN ("Production", "Staging");
settings:dt.security_context IN ("Production", "Staging")
```

**App ID Boundaries:**
```
# Allow specific apps only (IN)
shared:app-id IN ("dynatrace.dashboards", "dynatrace.logs", "dynatrace.notebooks");

# Exclude specific apps (NOT IN)
shared:app-id NOT IN ("dynatrace.classic.smartscape", "dynatrace.classic.custom.applications");
```

**Schema ID Boundaries:**
```
# Allow specific schemas only (IN)
settings:schemaId IN ("builtin:alerting.profile", "builtin:alerting.maintenance-window");

# Exclude specific schemas (NOT IN)
settings:schemaId NOT IN ("builtin:span-attribute", "builtin:span-capture-rule");
```

### Global State Access

Commands access global CLI state through the `cli` package:
```go
import "github.com/jtimothystewart/dtiam/internal/cli"

// Available:
cli.GlobalState.Context   // string - context override
cli.GlobalState.Output    // output.Format - output format
cli.GlobalState.Verbose   // bool - verbose mode
cli.GlobalState.Plain     // bool - plain mode (no colors)
cli.GlobalState.DryRun    // bool - dry-run mode

// Create printer with current settings
printer := cli.GlobalState.NewPrinter()
```

### HTTP Client Usage

Always close the client when done:
```go
c, err := common.CreateClient()
if err != nil {
    return err
}
defer c.Close()

ctx := context.Background()
body, err := c.Get(ctx, "/groups", nil)
```

### Output Formatting

Use the Printer for consistent output:
```go
printer := cli.GlobalState.NewPrinter()
printer.Print(data, output.GroupColumns())      // List
printer.PrintSingle(data, output.GroupColumns()) // Single item
printer.PrintDetail(data)                        // Key-value pairs
printer.PrintSuccess("Operation completed")      // Success message
printer.PrintWarning("Warning message")          // Warning message
```

## API Endpoints

Base URL: `https://api.dynatrace.com/iam/v1/accounts/{account_uuid}`

| Resource | Path |
|----------|------|
| Groups | `/groups` |
| Users | `/users` |
| Service Users | `/service-users` |
| Limits | `/limits` |
| Policies | `/repo/{level_type}/{level_id}/policies` |
| Bindings | `/repo/{level_type}/{level_id}/bindings` |
| Boundaries | `/repo/account/{uuid}/boundaries` |

**Environment API**: `https://api.dynatrace.com/env/v2/accounts/{uuid}/environments`

**Subscription API**: `https://api.dynatrace.com/sub/v2/accounts/{uuid}/subscriptions`

**Resolution API** (for effective permissions):
Base URL: `https://api.dynatrace.com/iam/v1`

| Resource | Path |
|----------|------|
| Effective Permissions | `/resolution/{level_type}/{level_id}/effectivepermissions` |

**App Engine Registry API**:
Base URL: `https://{environment-id}.apps.dynatrace.com/platform/app-engine/registry/v1`

| Resource | Path |
|----------|------|
| Apps | `/apps` |
| App Details | `/apps/{id}` |

Level types: `account`, `environment`, `global`

## API Coverage & Missing Operations

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

### Not Yet Implemented (from Python version)

| Feature | Description | Priority |
|---------|-------------|----------|
| `bulk` commands | Bulk operations from CSV/YAML | High |
| `template` commands | Template-based resource creation | Medium |
| `analyze` commands | Permissions analysis | Medium |
| `export` commands | Export resources to files | Medium |
| `get apps` | App Engine Registry integration | Low |
| `get schemas` | Settings schema listing | Low |
| Caching | In-memory caching with TTL | Low |
| Permissions calculation | Effective permissions for users/groups | Medium |

## Configuration

Config file: `~/.config/dtiam/config` (YAML)

```yaml
api-version: v1
kind: Config
current-context: production
contexts:
  - name: production
    context:
      account-uuid: abc-123
      credentials-ref: prod-creds
credentials:
  - name: prod-creds
    credential:
      client-id: dt0s01.XXX
      client-secret: dt0s01.XXX.YYY
```

Environment variable overrides:
- `DTIAM_CONTEXT` - context name
- `DTIAM_OUTPUT` - output format
- `DTIAM_VERBOSE` - verbose mode
- `DTIAM_CLIENT_ID` - OAuth2 client ID
- `DTIAM_CLIENT_SECRET` - OAuth2 client secret
- `DTIAM_ACCOUNT_UUID` - account UUID

## Common Tasks

### Build

```bash
make build           # Build binary to bin/dtiam
make build-all       # Build for all platforms
make install         # Install to $GOPATH/bin
```

### Test

```bash
make test            # Run all tests
make test-coverage   # Run tests with coverage report
```

### Lint

```bash
make lint           # Run golangci-lint
make fmt            # Format code
make vet            # Run go vet
```

### Debug Authentication

```bash
# Verbose mode shows HTTP requests
dtiam -v get groups
```

Check `~/.config/dtiam/config` for credential configuration.

## Code Style

- Use Go idioms and conventions
- Error handling: return errors, don't panic
- Use `context.Context` for cancellation
- Close resources with defer
- Use interfaces for testability
- Keep packages focused and minimal
- All exported types and functions must have comments
- Use meaningful variable names

## Dependencies

```
github.com/spf13/cobra      # CLI framework
github.com/olekukonko/tablewriter  # Table output
golang.org/x/oauth2         # OAuth2 support
gopkg.in/yaml.v3            # YAML parsing
```

## Troubleshooting

### "No context configured"

Run:
```bash
dtiam config set-credentials NAME --client-id XXX --client-secret YYY
dtiam config set-context NAME --account-uuid UUID --credentials-ref NAME
dtiam config use-context NAME
```

### "Permission denied"

OAuth2 client needs appropriate scopes:
- `account-idm-read` / `account-idm-write`
- `iam-policies-management`
- `account-env-read`
- `iam:effective-permissions:read` (for effective permissions API)
- `app-engine:apps:run` (for App Engine Registry API)

### Build Errors

Ensure Go 1.22+ is installed:
```bash
go version
go mod tidy
```

## Documentation

- [README.md](README.md) - Overview and quick start
- [docs/QUICK_START.md](docs/QUICK_START.md) - Detailed getting started
- [docs/COMMANDS.md](docs/COMMANDS.md) - Full command reference
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - Technical design
- [docs/API_REFERENCE.md](docs/API_REFERENCE.md) - Programmatic usage
- [examples/README.md](examples/README.md) - Sample configurations and scripts
