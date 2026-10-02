# Testing Standards

---

## 1. Test Requirements [MUST]

- Every new package MUST have tests — no deferring
- All tests must pass before merge: `make test`
- New features should include tests
- Bug fixes must include a regression test

## 2. Test Structure [MUST]

Use table-driven tests for Go:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "should return value when input is valid",
            input:    "valid",
            expected: "result",
        },
        {
            name:    "should error when input is empty",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if tt.wantErr {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got != tt.expected {
                t.Errorf("got %q, want %q", got, tt.expected)
            }
        })
    }
}
```

## 3. Test Naming [SHOULD]

Test names describe behavior, not implementation:
```
should return null when user does not exist
should throw unauthorized when token is expired
should retry three times before failing
```

## 4. What to Test [MUST]

**Test:**
- Every public API or function with business logic
- Every bug fix (regression test)
- Edge cases mentioned in requirements
- Flag parsing and dry-run behavior for commands

**Do NOT test:**
- Framework boilerplate (getters/setters, auto-generated code)
- Third-party library behavior
- Implementation details that change without behavior changes

## 5. Mocking [SHOULD]

- Use mocks for external dependencies (HTTP, DB)
- Use real implementations for internal code when feasible
- Test files live next to their source: `foo.go` → `foo_test.go`

## 6. Coverage [SHOULD]

- Coverage is a signal, not a goal
- All critical paths must have tests before feature is complete
- Unhappy paths need at least as much coverage as happy paths
- New code should not reduce overall test coverage
- Current: 38 test packages, all passing via `make test`

## 7. Integration Suite [SHOULD]

- `test/integration/` (build tag `integration`) drives the real binary against
  a live account: `DTIAM_INTEGRATION=1 DTIAM_INTEGRATION_CONTEXT=<ctx> make test-integration`
- Never runs by default; refuses a context that is not `readwrite`
- Only creates `dtiam-it-<run>-*` objects, deletes them in `t.Cleanup`, and
  sweeps stale ones; never modifies anything else
- Every command that changes the account should have a lifecycle test there,
  asserting on the agent envelope (`-A`)
