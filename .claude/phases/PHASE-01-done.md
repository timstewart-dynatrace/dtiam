# Phase 01 — Foundation
Status: DONE

## Goal
Eliminate duplicated code, add safe type assertions, extract magic strings, add Phase 2 dependencies.

## Tasks
- [x] Consolidate token provider adapter (deleted get/helpers.go, describe/helpers.go)
- [x] Create safe type assertion helpers (internal/utils/safemap.go)
- [x] Replace ~40 unsafe bare .(string) assertions with StringFrom()
- [x] Extract API URL constants to internal/client/urls.go
- [x] Add Phase 2 dependencies to go.mod (resty, viper, xdg, logrus, cockroachdb/errors)
- [x] Add tests for safemap and URL constants

## Acceptance Criteria
- [x] Zero duplicate tokenProviderAdapter
- [x] Zero unguarded type assertions
- [x] go build, go vet, go test all pass

## Branch
`feature/phase1-foundation` — commit 8280e2f
