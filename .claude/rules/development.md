# Development Guide

---

## 1. Prerequisites

- Go 1.23+
- `golangci-lint` for linting
- `make` for build automation

## 2. Build & Run

```bash
make build           # Build binary to bin/dtiam
make build-all       # Build for all platforms
make install         # Install to $GOPATH/bin

./bin/dtiam --help   # Run locally
dtiam --help         # If installed
dtiam -v get groups  # Verbose output
```

## 3. Testing

```bash
make test            # Run all tests
make test-coverage   # Run tests with coverage report
go test ./... -v     # Verbose test output
```

## 4. Code Quality

```bash
make lint            # Run golangci-lint
make fmt             # Format code
make vet             # Run go vet
```

## 5. Authentication

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
| `DTIAM_CLIENT_ID` | OAuth2 client ID (optional — auto-extracted from secret) |
| `DTIAM_CLIENT_SECRET` | OAuth2 client secret (format: `dt0s01.CLIENTID.SECRET`) |
| `DTIAM_ACCOUNT_UUID` | Dynatrace account UUID |
| `DTIAM_CONTEXT` | Override current context |
| `DTIAM_OUTPUT` | Output format (table, json, yaml, csv) |
| `DTIAM_VERBOSE` | Enable verbose output |
| `DTIAM_ENVIRONMENT_URL` | Environment URL for apps/schemas |
| `DTIAM_ENVIRONMENT_TOKEN` | Separate environment API token (used by apps/schemas/env-users/env-groups) |
| `DTIAM_API_URL` | Alternative host for all Account Management APIs (replaces `https://api.dynatrace.com`) |
| `DTIAM_SCOPES` | Custom OAuth scopes (comma-separated) |

## 6. Configuration

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

## 7. Adding a New Resource (End-to-End Example)

```bash
# 1. Create feature branch
git checkout -b feature/add-apps-resource

# 2. Implement
# - pkg/resources/apps.go (handler)
# - internal/commands/get/get.go (command)
# - pkg/output/columns.go (columns)

# 3. Test
make build && ./bin/dtiam get apps --help

# 4. Document (MANDATORY)
# - CLAUDE.md / .claude/architecture.md
# - docs/COMMANDS.md
# - README.md
# - docs/ARCHITECTURE.md
# - examples/

# 5. Commit, push, merge
git add . && git commit -m "feat: add apps resource"
git push -u origin feature/add-apps-resource
git checkout main && git merge feature/add-apps-resource --no-ff
```
