# Contributing to dtiam

Thanks for your interest. dtiam is an independent, community-developed CLI for
Dynatrace IAM — **not** produced, endorsed, or supported by Dynatrace.

## Before you start

For anything beyond a typo fix, open an issue first so the approach can be
agreed before you write code. Large rewrites need explicit agreement.

## Development setup

```bash
git clone https://github.com/timstewart-dynatrace/dtiam.git
cd dtiam
make build          # -> bin/dtiam
make test
make lint
```

Requires Go 1.23+, `make`, and `golangci-lint` (installed automatically by
`make lint`).

## Workflow

```bash
git checkout main && git pull
git checkout -b feature/descriptive-name   # or fix/descriptive-name
# ... work ...
make build && make test && make lint
```

Never commit features directly to `main`. Branch names are
`feature/...` or `fix/...`.

**Separate refactors from features.** A commit that both moves code and changes
behavior cannot be reviewed or reverted cleanly.

## What a complete change includes

Taken from `.claude/rules/core.md`, which is authoritative:

- [ ] Code, following the conventions in `.claude/rules/go.md` and
      `.claude/rules/command-standards.md`
- [ ] Tests — every new package needs them; every bug fix needs a regression test
- [ ] `docs/COMMANDS.md` updated for any new or changed command
- [ ] `README.md` updated if the resource table or scopes change
- [ ] `.claude/architecture.md` updated if structure or endpoints change
- [ ] `CHANGELOG.md` entry under `[Unreleased]`
- [ ] Version bumped in `pkg/version/version.go`, kept in sync with
      `.claude/settings.json`
- [ ] `make test` and `make lint` both clean

## Command conventions

Every command must:

- Use `RunE` and **return** errors; never call `os.Exit` inside a command
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Send all data output through `cli.GlobalState.NewPrinter()`, never
  `fmt.Printf` — otherwise `-o json`, `-o yaml`, and `--plain` silently break
- Have `Use`, `Short`, `Long`, and `Example`
- Support `--dry-run` if it mutates anything
- Confirm via `pkg/prompt` if it is destructive, with `--force` to skip
- Send progress and status messages to **stderr**, data to **stdout**

See `.claude/rules/command-standards.md` for the full checklist.

## Working with the Dynatrace API

Two hard-won lessons, both of which have caused silently-broken commands:

**Verify response shapes against a live account, not just the documentation.**
Several endpoints return keys the docs do not state — `content`, `data`, and
`results` all appear where `items` was documented. A wrong key yields an empty
list and a green test suite, not an error.

**Do not write a test fixture from a guess.** A fixture that invents a response
shape makes the suite argue the code is correct while the command is broken. If
you cannot verify a shape, say so in the test comment.

## Tests

Table-driven, with names describing behavior:

```go
{name: "should return an error when the scope type is unknown", ...}
```

Unit tests (`make test`) must not make real network calls. Use `httptest` and
the helpers in `pkg/resources/testhelper_test.go`.

### Integration suite

`test/integration/` runs the real binary against a live account. It is behind
the `integration` build tag and refuses to start without both variables:

```bash
DTIAM_INTEGRATION=1 DTIAM_INTEGRATION_CONTEXT=my-test-context make test-integration
```

The context must be at safety level `readwrite`. Every object the suite creates
is named `dtiam-it-<run>-...` and deleted when its test ends; leftovers from a
crashed run are swept on the next one (after an hour). Use an account where
creating and deleting a few throwaway groups, policies, boundaries, a service
user and a short-lived token is acceptable. Tests drive the CLI in agent mode
(`-A`) and assert on the envelope, so they exercise output and exit codes too.
Add an integration test for any command that changes the account.

## Commit messages

Conventional prefixes — `feat:`, `fix:`, `docs:`, `refactor:`, `test:`,
`chore:`, `style:`. Explain *why* in the body when it is not obvious.

## Reporting bugs

Include `dtiam version`, the exact command, what you expected, what happened,
and `-v` output with credentials, account UUIDs, and emails redacted.

## Security

Do not open public issues for security problems — see [SECURITY.md](SECURITY.md).

## License

Contributions are licensed under the same terms as the project
([LICENSE](LICENSE)).
