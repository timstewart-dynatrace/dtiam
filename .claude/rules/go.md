# Go Conventions

---

## 1. Code Style [MUST]

- Use Go idioms and conventions
- Error handling: return errors, don't panic
- Use `context.Context` for cancellation
- Close resources with `defer`
- Use interfaces for testability
- Keep packages focused and minimal
- All exported types and functions must have comments
- Use meaningful variable names

## 1a. Package Layout [MUST]

| Location | Holds | Importable |
|----------|-------|------------|
| `pkg/` | API client, auth, config, resource handlers, output, diff, watch, template, utils | yes |
| `internal/cli`, `internal/commands` | cobra wiring: root command, global state, all commands | no |

New reusable logic goes in `pkg/`. New commands go in `internal/commands/`.
A `pkg/` package MUST NOT import `internal/cli` or `internal/commands` — that
would make it unusable by external callers in practice and invert the dependency.

## 2. Import Ordering [MUST]

```go
import (
    // stdlib
    "context"
    "fmt"

    // external
    "github.com/spf13/cobra"

    // internal
    "github.com/jtimothystewart/dtiam/internal/cli"
)
```

## 3. Error Wrapping [MUST]

Always wrap errors with context:
```go
if err != nil {
    return fmt.Errorf("failed to create group: %w", err)
}
```

Never use `os.Exit()` inside a command. Return errors and let cobra handle display.

## 4. Key Patterns

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
cli.AddCommand(newfeature.Cmd)
```

### Adding a New Resource Handler

1. Create handler in `pkg/resources/<name>.go`:
```go
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

2. Add columns in `pkg/output/columns.go`:
```go
func NewResourceColumns() []Column {
    return []Column{
        {Key: "uuid", Header: "UUID"},
        {Key: "name", Header: "NAME"},
        {Key: "description", Header: "DESCRIPTION"},
    }
}
```

### Filtering Resources

All `get` commands support partial text matching via `--name` (or `--email` for users).

```go
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

| Command | Filter | Description |
|---------|--------|-------------|
| `get groups` | `--name` | Filter by group name |
| `get users` | `--email` | Filter by email address |
| `get policies` | `--name` | Filter by policy name |
| `get boundaries` | `--name` | Filter by boundary name |
| `get environments` | `--name` | Filter by environment name |
| `service-user list` | `--name` | Filter by service user name |

Filter behavior: case-insensitive, substring match, client-side.

### Global State Access

```go
import "github.com/jtimothystewart/dtiam/internal/cli"

cli.GlobalState.Context   // string - context override
cli.GlobalState.Output    // output.Format - output format
cli.GlobalState.Verbose   // bool - verbose mode
cli.GlobalState.Plain     // bool - plain mode (no colors)
cli.GlobalState.DryRun    // bool - dry-run mode

printer := cli.GlobalState.NewPrinter()
```

### HTTP Client Usage

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

```go
printer := cli.GlobalState.NewPrinter()
printer.Print(data, output.GroupColumns())      // List
printer.PrintSingle(data, output.GroupColumns()) // Single item
printer.PrintDetail(data)                        // Key-value pairs
printer.PrintSuccess("Operation completed")      // Success message
printer.PrintWarning("Warning message")          // Warning message
```

### Boundary Query Format

**Management Zone Boundaries:**
```
environment:management-zone IN ("Production");
storage:dt.security_context IN ("Production");
settings:dt.security_context IN ("Production")
```

**App ID Boundaries:**
```
shared:app-id IN ("dynatrace.dashboards", "dynatrace.logs");
shared:app-id NOT IN ("dynatrace.classic.smartscape");
```

**Schema ID Boundaries:**
```
settings:schemaId IN ("builtin:alerting.profile");
settings:schemaId NOT IN ("builtin:span-attribute");
```

## 5. Dependencies

```
github.com/spf13/cobra           # CLI framework
github.com/olekukonko/tablewriter # Table output
gopkg.in/yaml.v3                 # YAML parsing
github.com/go-resty/resty/v2     # HTTP client with retry
github.com/sirupsen/logrus       # Structured logging
github.com/spf13/viper           # Configuration management with env binding
github.com/adrg/xdg              # XDG base directory support
```

OAuth2 is implemented using the standard library (`net/http`, `net/url`) without external dependencies.
