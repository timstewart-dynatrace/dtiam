# Phase 04 — Advanced Operations
Status: DONE

## Goal
Group cloning, app/schema boundary helpers, group setup, parameterized policies.

## Tasks

### 4.1 Group Clone ✅
- [x] `group clone SOURCE --name NEW [--include-members] [--include-policies]`
- [x] Resolves source by name/UUID, creates new group, copies members/bindings
- [x] Supports --dry-run, verbose progress to stderr
- [x] Tests for subcommand, flags, required --name

### 4.2 App Boundaries ✅
- [x] `boundary create-app-boundary NAME --app-ids ...`
- [x] Builds `shared:app-id IN/NOT IN` query
- [x] Optional `--environment` for validation via AppHandler.Get
- [x] Tests for subcommand, flags, buildBoundaryQuery helper

### 4.3 Schema Boundaries ✅
- [x] `boundary create-schema-boundary NAME --schema-ids ...`
- [x] Builds `settings:schemaId IN/NOT IN` query
- [x] Optional `--environment` for validation via SchemaHandler.Get
- [x] Tests for subcommand, flags

### 4.4 Group Setup ✅
- [x] `group setup --name NAME --policies-file FILE`
- [x] Loads YAML/JSON with policy definitions, creates group, binds policies
- [x] Tests for subcommand, flags, required flags

### 4.5 Parameterized Policies ✅
- [x] Added `--param key=value` repeatable flag to `create binding`
- [x] Extended `BindingHandler.Create()` to accept optional `parameters map[string]string`
- [x] Updated all callers (create.go, bulk.go, bindings_test.go)
- [x] Tests for --param flag

## Acceptance Criteria
- [x] All 5 new commands have Example help text
- [x] All 26 packages pass tests
- [x] All new commands support --dry-run where applicable
