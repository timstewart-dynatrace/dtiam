# Phase 02 — Architecture Alignment with go-dtctl-main
Status: DONE

## Goal
Adopt go-dtctl-main patterns for HTTP client, config, logging, errors, and output. Replace internal plumbing without changing command behavior.

## Prerequisites
- Phase 1 and 1.5 complete
- Dependencies already in go.mod (resty, viper, xdg, logrus, cockroachdb/errors)
- Command standards documented in .claude/rules/command-standards.md

## Reference
- go-dtctl-main source: /Users/Shared/GitHub/go-dtctl-main
- Key pattern files: pkg/client/client.go, pkg/config/config.go, pkg/output/, pkg/resources/, pkg/suggest/

## Tasks

### 2.1 HTTP Client → Resty
- [ ] Rewrite `internal/client/client.go` to use `go-resty/resty/v2`
- Keep same `Client` struct API: `Get()`, `Post()`, `Put()`, `Delete()`, `GetJSON()`, `PostJSON()`
- Use resty's `.SetRetryCount(3)`, `.SetRetryWaitTime(1s)`, `.AddRetryCondition()` for 429/5xx
- Use resty's `.SetAuthToken()` via TokenProvider
- Use resty's `.SetDebug(verbose)` instead of manual fmt.Printf logging
- Update `internal/client/errors.go`: wrap with `cockroachdb/errors`

### 2.2 Config → Viper + XDG
- [ ] Rewrite `internal/config/loader.go`:
  - Use `adrg/xdg` for `xdg.ConfigHome` instead of manual `os.Getenv("XDG_CONFIG_HOME")`
  - Use `spf13/viper` for automatic `DTIAM_*` env var binding
  - Keep `config.go` struct definitions unchanged
- [ ] Update `internal/cli/root.go`: bind Viper to cobra persistent flags

### 2.3 Structured Logging → Logrus
- [ ] Create `internal/logging/logger.go`
  - Package-level logger: `logging.Log`
  - `Init(verbose bool)` sets level
  - Structured fields for HTTP request/response
- [ ] Replace all `fmt.Printf` debug output in client.go with `logging.Log.WithFields()`
- [ ] Verbose mode (`-v`) sets Debug; default is Warn
- [ ] Progress messages in bulk/export go to stderr via logrus

### 2.4 Error Handling → cockroachdb/errors
- [ ] Update all `fmt.Errorf` in `internal/resources/*.go` to `errors.Wrap`/`errors.Newf`
- [ ] Update `internal/cli/root.go` `Execute()`:
  - Inspect error types for structured exit codes
  - 1=general, 2=usage, 3=auth, 4=not-found, 5=permission

### 2.5 Output → Struct Tags
- [ ] Create typed response structs in each resource handler:
  ```go
  type Group struct {
      UUID        string `json:"uuid" table:"UUID"`
      Name        string `json:"name" table:"NAME"`
      Description string `json:"description" table:"DESCRIPTION"`
      Owner       string `json:"owner" table:"OWNER,wide"`
  }
  ```
- [ ] Do same for: Policy, User, Binding, Boundary, Environment, ServiceUser, Limit, Subscription
- [ ] Create `internal/output/structprinter.go`: reads `table` struct tags via reflection
- [ ] Printer interface gains `Print(v any)` and `PrintList(v any)` (go-dtctl-main pattern)
- [ ] Migrate get/describe commands to use typed structs (start with groups, policies, users)

### 2.6 Prompt Package Enhancement
- [ ] Already created in Phase 1.5
- [ ] Ensure `prompt.Confirm()` and `prompt.ConfirmDelete()` handle all edge cases

### 2.7 Levenshtein Suggestion Engine
- [ ] Create `internal/suggest/suggest.go`:
  - `EditDistance(a, b string) int`
  - `FindClosest(input string, candidates []string, maxDistance int) string`
  - `FindClosestN(input string, candidates []string, maxDistance, n int) []string`
- [ ] Wire into root command's `RunE`:
  - On "unknown command", suggest closest command name
  - On "unknown flag", suggest closest flag name
  - Max edit distance: 3

### 2.8 Config Enhancements (from Python-IAM-CLI v3.13.0)
- [ ] Add `--api-url` global flag to override IAM API base URL
- [ ] Add `api-url` field to credential config (for testing / private deployments)
- [ ] Add `scopes` field to credential config (custom scopes per credential)
- [ ] Add `environment-url` field to credential config (for apps/schemas commands)
- [ ] Add `environment-token` field to credential config (separate env API token)
- [ ] Bind all new fields to `DTIAM_API_URL`, `DTIAM_SCOPES`, `DTIAM_ENVIRONMENT_URL`, `DTIAM_ENVIRONMENT_TOKEN` env vars via Viper

## Key Files
- REWRITE: `internal/client/client.go`, `internal/config/loader.go`
- CREATE: `internal/logging/logger.go`, `internal/output/structprinter.go`, `internal/suggest/suggest.go`
- MODIFY: `internal/cli/root.go`, all `internal/resources/*.go`

## Acceptance Criteria
- [ ] All existing commands work identically (same CLI interface)
- [ ] `dtiam -v get groups` shows structured log output (timestamps, levels)
- [ ] Exit codes differ: auth=3, not-found=4, permission=5
- [ ] At least groups, policies, users use struct-tag output
- [ ] Typo in command name shows suggestion: `dtiam get grops` → "Did you mean 'groups'?"
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all pass
- [ ] Tests from Phase 1 still pass; new tests for resty client, Viper config, struct printer, suggest

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
