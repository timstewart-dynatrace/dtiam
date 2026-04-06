# Command Standards — MANDATORY

These standards are modeled after go-dtctl-main and kubectl conventions. Every command in dtiam MUST follow these patterns. Violations will cause inconsistent behavior for AI agents and automation scripts.

---

## 1. Command Structure [MUST]

```
dtiam [global-flags] <verb> [<resource>] [<identifier>] [local-flags]
```

**Verbs:** `get`, `describe`, `create`, `delete`, `apply`, `export`, `analyze`, `bulk`, `config`, `user`, `group`, `service-user`, `boundary`, `account`, `cache`, `template`

**Resource Naming:**
- List operations use **plural**: `get groups`, `get policies`
- Single-item get also uses **plural** with optional arg: `get groups [ID]`
- Create/delete/describe use **singular nouns**: `create group`, `delete policy`, `describe boundary`
- All plural commands MUST have singular aliases: `groups` → alias `group`

---

## 2. Global Flags [MUST]

Every command inherits these from `cli/root.go`:

| Flag | Short | Type | Default | Purpose |
|------|-------|------|---------|---------|
| `--context` | | string | "" | Override active context |
| `--output` | `-o` | string | "table" | Output format: table, wide, json, yaml, csv |
| `--verbose` | `-v` | bool | false | Debug logging |
| `--plain` | | bool | false | AI/machine mode: force JSON, no colors, skip prompts |
| `--dry-run` | | bool | false | Preview changes without executing |

**`--plain` behavior (AI consumption mode):**
- Table/wide output becomes JSON automatically
- All color codes stripped
- Interactive confirmation prompts skipped (acts like `--force`)
- Progress spinners/bars suppressed
- Only structured data written to stdout; messages to stderr

---

## 3. Output Patterns [MUST]

### All commands MUST use the printer

**Never use raw `fmt.Printf` to stdout for data output.** All data MUST flow through `cli.GlobalState.NewPrinter()` so that `-o json`, `-o yaml`, `--plain` work consistently.

| Operation | Printer Method | When |
|-----------|---------------|------|
| List resources | `printer.Print(data, columns)` | `get` commands returning lists |
| Single resource | `printer.PrintSingle(data, columns)` | `get` with identifier |
| Detailed view | `printer.PrintDetail(data)` | `describe` commands |
| Structured data | `printer.PrintAny(data)` | Analyze, export, any non-tabular data |
| Success message | `printer.PrintSuccess(fmt, args...)` | After create/delete/update |
| Warning | `printer.PrintWarning(fmt, args...)` | Non-fatal issues |
| Error | `printer.PrintError(fmt, args...)` | Error context (goes to stderr) |
| Progress/status | `fmt.Fprintf(os.Stderr, ...)` | Progress info during bulk ops |

**Rules:**
- Structured data → stdout (via printer)
- Status messages (progress, "Processing 5 of 10...") → stderr
- Errors → stderr (via `printer.PrintError` or `return err`)
- Success messages → stdout (via `printer.PrintSuccess`)

### Output format behavior

| Format | `--plain` false | `--plain` true |
|--------|-----------------|----------------|
| table | Colored table | JSON |
| wide | Extended table | JSON |
| json | Pretty JSON | Pretty JSON |
| yaml | YAML | YAML |
| csv | CSV | CSV |

---

## 4. Error Handling [MUST]

```go
// Pattern: return errors, let cobra handle display
RunE: func(cmd *cobra.Command, args []string) error {
    // ...
    if err != nil {
        return fmt.Errorf("failed to create group: %w", err)
    }
    return nil
}
```

**Rules:**
- Use `RunE` (not `Run`) for all commands
- Return errors — never `os.Exit()` inside a command
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Root command prints `Error: <message>` to stderr and exits 1
- Do NOT print errors inline AND return them (double printing)

---

## 5. Destructive Operations [MUST]

**ALL destructive operations MUST confirm before executing.**

Destructive = delete, remove, replace, bulk modify, anything that cannot be undone.

```go
import "github.com/jtimothystewart/dtiam/internal/prompt"

// Standard pattern:
if cli.GlobalState.IsDryRun() {
    printer.PrintWarning("Would delete %s: %s", resourceType, identifier)
    return nil
}

if !prompt.ConfirmDelete(resourceType, identifier, force || cli.GlobalState.IsPlain()) {
    printer.PrintMessage("Aborted.")
    return nil
}
```

**Rules:**
- Use `internal/prompt` package (never inline `bufio.NewReader` or `fmt.Scanln`)
- `--force` flag (lowercase `-f`) skips confirmation
- `--plain` mode skips confirmation (AI agents don't have stdin)
- `--dry-run` takes precedence: print what would happen, then return nil
- Order of checks: dry-run → confirm → execute

**Force flag naming:**
- Always `--force` with short `-f`
- Never `-F` (uppercase)
- Defined per-subcommand, not globally

---

## 6. Dry-Run Pattern [MUST]

**All mutating commands MUST support `--dry-run`.**

Mutating = create, delete, update, apply, bulk operations, boundary attach/detach.

```go
if cli.GlobalState.IsDryRun() {
    printer.PrintWarning("Dry run: would create group %q", name)
    // Optionally show what would be sent:
    return printer.PrintAny(payload)
}
```

**Read-only commands (get, describe, analyze, export) do NOT need dry-run.**

---

## 7. Help Text [MUST]

Every cobra.Command MUST have:
- `Use`: Usage pattern showing required/optional args
- `Short`: One-line description (< 80 chars)
- `Long`: Multi-paragraph description
- `Example`: Real-world usage examples with comments

```go
var groupCmd = &cobra.Command{
    Use:   "group IDENTIFIER",
    Short: "Delete a group by name or UUID",
    Long:  `Delete a group from the Dynatrace account. Requires confirmation unless --force is set.`,
    Example: `  # Delete a group by name
  dtiam delete group "My Group"

  # Delete by UUID without confirmation
  dtiam delete group abc-123 --force

  # Preview deletion
  dtiam delete group "My Group" --dry-run

  # Machine-friendly (skip prompts, JSON output)
  dtiam delete group "My Group" --force --plain`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error { ... },
}
```

**Example format:**
- Indent with 2 spaces
- Comment line (`#`) before each example
- Show common flags: `--force`, `--dry-run`, `--plain`, `-o json`
- Include AI/automation use case

---

## 8. Args Validation [MUST]

| Command Type | Validation | Pattern |
|-------------|-----------|---------|
| List/get all | `cobra.MaximumNArgs(1)` | Optional identifier → list or get |
| Get/describe single | `cobra.ExactArgs(1)` | Required identifier |
| Create (from flags) | `cobra.NoArgs` | All input via flags |
| Create (from file) | `cobra.NoArgs` | Input via `-f` flag |
| Delete | `cobra.ExactArgs(1)` | Required identifier |

**Required flags MUST use `cmd.MarkFlagRequired("name")`** instead of manual checks in RunE.

---

## 9. Client Lifecycle [MUST]

```go
c, err := common.CreateClient()
if err != nil {
    return err
}
defer c.Close()
```

- Always use `common.CreateClient()` — never local implementations
- Always `defer c.Close()`
- Create client AFTER dry-run check (don't need auth for preview)

---

## 10. Command Registration [MUST]

```go
// In cmd/dtiam/main.go:
cli.AddCommand(get.Cmd)

// In each command package:
var Cmd = &cobra.Command{
    Use:   "verb",
    Short: "Description",
}

func init() {
    Cmd.AddCommand(subCmd1)
    Cmd.AddCommand(subCmd2)
}
```

- One `Cmd` export per package
- Subcommands registered in `init()`
- Flags registered in `init()` blocks, not inline

---

## 11. Verbose Output [SHOULD]

Commands that make multiple API calls SHOULD log progress when `--verbose` is set:

```go
if cli.GlobalState.IsVerbose() {
    fmt.Fprintf(os.Stderr, "Fetching groups... (%d found)\n", len(groups))
}
```

Verbose output goes to **stderr** so it doesn't interfere with stdout data.

---

## 12. New Command Checklist

When adding any new command, verify ALL of the following:

- [ ] Uses `common.CreateClient()` for API access
- [ ] Uses `cli.GlobalState.NewPrinter()` for all data output
- [ ] Has `Use`, `Short`, `Long`, `Example` fields
- [ ] Mutating commands support `--dry-run`
- [ ] Destructive commands use `prompt.ConfirmDelete()` with `--force`
- [ ] Client created AFTER dry-run check
- [ ] Errors returned (not printed inline)
- [ ] Args validated with cobra validators
- [ ] Progress/status messages go to stderr
- [ ] `--plain` mode produces clean JSON output
- [ ] Tests exist for flag parsing and dry-run behavior
