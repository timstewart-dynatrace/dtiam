# Phase 01.5 — Command Standards
Status: DONE

## Goal
Standardize all commands to follow go-dtctl-main kubectl patterns. Document standards as mandatory rules.

## Tasks
- [x] Create .claude/rules/command-standards.md (mandatory for all future work)
- [x] Create internal/prompt/confirm.go (centralized confirmation)
- [x] Migrate delete commands to use prompt package
- [x] Fix --plain mode to force JSON output for AI consumption
- [x] Fix bulk.go fmt.Scanln → prompt.Confirm
- [x] Fix bulk.go -F → -f (force flag naming)
- [x] Add Example help text to ALL commands (get, describe, create, delete, user, group, serviceuser, boundary, account, config, cache)
- [x] Standardize dry-run output to use printer.PrintWarning instead of fmt.Printf
- [x] Create phase docs for Phases 2-7

## Acceptance Criteria
- [x] Zero fmt.Printf for dry-run messages
- [x] Zero fmt.Scanln or inline bufio confirmation
- [x] All commands have Example field
- [x] --plain forces JSON output
- [x] go build, go vet, go test all pass

## Branch
`feature/phase1.5-command-standards`
