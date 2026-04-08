# Phase 08 — Retroactive Test Coverage
Status: PENDING

## Goal
Comprehensive test coverage for ALL features that existed before Phase 2. Every resource handler, command, output format, and utility must have tests. Target: establish a regression safety net so Phase 2+ refactoring doesn't break existing behavior.

## Prerequisites
- None — can run in parallel with or after Phase 2
- Should be done BEFORE Phase 2 modifies core internals (to catch regressions)

## Reference
- dtctl-1 test patterns: /Users/Shared/GitHub/dtctl-1/pkg/output/golden_test.go (golden/snapshot tests)
- dtctl-1 test utils: /Users/Shared/GitHub/dtctl-1/cmd/testutil/golden.go (AssertGolden helper)

## Tasks

### 8.1 Test Infrastructure
- [ ] Create `internal/client/testutil_test.go`:
  - `NewMockServer(t *testing.T) (*httptest.Server, *client.Client)` — returns mock HTTP server + configured client
  - `RegisterHandler(method, path string, statusCode int, responseBody string)` — register mock responses
  - `AssertRequestCount(path string, expected int)` — verify API call counts
- [ ] Create `internal/testutil/golden.go`:
  - `AssertGolden(t *testing.T, name string, actual string)` — compare against golden file
  - `UpdateGolden(t *testing.T, name string, content string)` — update golden file with `-update` flag
  - Golden files stored in `internal/testutil/testdata/golden/`

### 8.2 Resource Handler Tests
Each handler needs tests for List, Get, GetByName, Create, Delete (where applicable).

- [ ] `internal/resources/groups_test.go`:
  - TestGroupHandler_List — mock /groups, verify parsed response
  - TestGroupHandler_Get — mock /groups/{uuid}, verify fields
  - TestGroupHandler_GetByName — name resolution via list
  - TestGroupHandler_Create — verify POST body
  - TestGroupHandler_Delete — verify DELETE call
  - TestGroupHandler_GetMembers — mock /groups/{uuid}/users
  - TestGroupHandler_AddMember — verify POST
  - TestGroupHandler_RemoveMember — verify DELETE
- [ ] `internal/resources/users_test.go`:
  - TestUserHandler_List, Get, GetByEmail, Create, Delete
  - TestUserHandler_AddToGroups, RemoveFromGroups, ReplaceGroups
- [ ] `internal/resources/policies_test.go`:
  - TestPolicyHandler_List, Get, GetByName, Create, Delete
  - TestPolicyHandler_ListAllLevels — verify multi-level aggregation
- [ ] `internal/resources/bindings_test.go`:
  - TestBindingHandler_List, Create, Delete
  - TestBindingHandler_GetForGroup, GetForPolicy
  - TestBindingHandler_AddBoundary, RemoveBoundary
- [ ] `internal/resources/boundaries_test.go`:
  - TestBoundaryHandler_List, Get, GetByName, Create, Delete
  - TestBoundaryHandler_GetAttachedPolicies
  - TestBoundaryHandler_buildZoneQuery — verify management zone query format
- [ ] `internal/resources/environments_test.go`:
  - TestEnvironmentHandler_List, Get, GetByName
- [ ] `internal/resources/serviceusers_test.go`:
  - TestServiceUserHandler_List, Get, GetByName, Create, Update, Delete
  - TestServiceUserHandler_AddToGroup, RemoveFromGroup, GetGroups
- [ ] `internal/resources/limits_test.go`:
  - TestLimitsHandler_List, GetSummary, CheckCapacity
- [ ] `internal/resources/subscriptions_test.go`:
  - TestSubscriptionHandler_List, Get, GetForecast
- [ ] `internal/resources/handler_test.go`:
  - TestBaseHandler_List, Get, Create, Delete
  - TestGetOrResolve — UUID vs name fallback logic

### 8.3 Command Tests (Flag Parsing & Dry-Run)
Test each command's argument validation, required flags, and dry-run behavior.

- [ ] `internal/commands/get/get_test.go`:
  - TestGetGroups_FlagParsing — --name, -o json, --plain
  - TestGetPolicies_LevelFlags — --level, --all-levels
  - TestGetBindings_FilterFlags — --group, --policy
- [ ] `internal/commands/describe/describe_test.go`:
  - TestDescribeGroup_RequiresArg
  - TestDescribeUser_RequiresArg
- [ ] `internal/commands/create/create_test.go`:
  - TestCreateGroup_RequiredFlags — --name required
  - TestCreateGroup_DryRun — no API call on --dry-run
  - TestCreatePolicy_RequiredFlags — --name, --statement
  - TestCreateBinding_RequiredFlags — --group, --policy
  - TestCreateBoundary_RequiredFlags — --name + (--zone or --query)
- [ ] `internal/commands/delete/delete_test.go`:
  - TestDeleteGroup_RequiresArg
  - TestDeleteGroup_DryRun
  - TestDeleteGroup_Force — skips confirmation
- [ ] `internal/commands/user/user_test.go`:
  - TestUserCreate_RequiredFlags
  - TestUserAddToGroups_RequiredFlags
- [ ] `internal/commands/serviceuser/serviceuser_test.go`:
  - TestServiceUserCreate_RequiredFlags
- [ ] `internal/commands/group/group_test.go`:
  - TestGroupMembers_RequiresArg
  - TestGroupAddMember_RequiredFlags
- [ ] `internal/commands/boundary/boundary_test.go`:
  - TestBoundaryAttach_RequiredFlags
  - TestBoundaryDetach_RequiredFlags
- [ ] `internal/commands/account/account_test.go`:
  - TestAccountLimits_NoArgs
  - TestAccountCheckCapacity_RequiresArg
- [ ] `internal/commands/bulk/bulk_test.go`:
  - TestBulkAddUsers_RequiredFlags
  - TestBulkAddUsers_CSVParsing — verify CSV/JSON/YAML detection
  - TestBulkCreateGroups_FileFormat
- [ ] `internal/commands/export/export_test.go`:
  - TestExportAll_DefaultFlags
  - TestExportGroup_RequiresArg
- [ ] `internal/commands/analyze/analyze_test.go`:
  - TestAnalyzeUserPermissions_RequiresArg
  - TestAnalyzePermissionsMatrix_ScopeFlag

### 8.4 Output Tests
- [ ] `internal/output/printer_test.go`:
  - TestPrinter_PrintTable — verify table formatting
  - TestPrinter_PrintJSON — verify valid JSON output
  - TestPrinter_PrintYAML — verify valid YAML output
  - TestPrinter_PrintCSV — verify CSV with headers
  - TestPrinter_PrintWide — verify extra columns
  - TestPrinter_PlainMode — verify --plain forces JSON
- [ ] `internal/output/columns_test.go`:
  - TestGroupColumns — correct headers and keys
  - TestUserColumns, PolicyColumns, etc. — all column sets
- [ ] `internal/output/table_test.go`:
  - TestTableFormatter — alignment, truncation, empty data

### 8.5 Auth Tests
- [ ] `internal/auth/oauth_test.go`:
  - TestOAuthTokenManager_GetToken — mock SSO endpoint
  - TestOAuthTokenManager_RefreshExpired — verify auto-refresh
  - TestOAuthTokenManager_ExtractClientID — dt0s01 format parsing
- [ ] `internal/auth/bearer_test.go`:
  - TestStaticTokenManager_GetToken
  - TestStaticTokenManager_IsValid

### 8.6 Config Tests (Expand Existing)
- [ ] `internal/config/config_test.go` (expand):
  - TestConfig_SaveAndLoad — round-trip YAML
  - TestConfig_EnvironmentOverrides — DTIAM_* env vars
  - TestConfig_MaskSecret — secret display masking
  - TestConfig_Validate — missing required fields

### 8.7 Prompt Tests
- [ ] `internal/prompt/confirm_test.go`:
  - TestConfirm_ForceSkips — skip=true returns true
  - TestConfirmDelete_ForceSkips
  - TestConfirmDelete_MessageFormat — verify resource type and name in output

### 8.8 Utils Tests (Expand Existing)
- [ ] `internal/utils/permissions_test.go` (expand):
  - TestPermissionsCalculator — aggregate from multiple policies
  - TestPermissionsMatrix — cross-tab generation
  - TestEffectivePermissionsAPI — mock resolution endpoint
  - TestParseStatementQuery_ComplexQueries — multi-action, conditions
- [ ] `internal/utils/safemap_test.go` (expand):
  - TestFloat64From, TestMapFrom edge cases

## Key Files
- CREATE: `internal/client/testutil_test.go`, `internal/testutil/golden.go`
- CREATE: All `*_test.go` files listed above
- EXPAND: Existing test files in config, output, utils

## Acceptance Criteria
- [ ] Every resource handler has at least 5 tests (CRUD + special ops)
- [ ] Every command package has flag parsing + dry-run tests
- [ ] Output formats verified for at least 3 resource types
- [ ] Auth token refresh tested with mock SSO
- [ ] `go test ./...` passes with no failures
- [ ] Coverage report shows significant improvement across all packages
- [ ] No test uses real Dynatrace API endpoints (all mocked)

## MANDATORY: Follow .claude/rules/command-standards.md for all new code
