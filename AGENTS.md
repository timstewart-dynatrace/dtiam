# AGENTS.md — dtiam

Guidance for AI agents and automation working with **dtiam**, a kubectl-inspired
CLI for Dynatrace Identity and Access Management.

> **DISCLAIMER:** dtiam is an independent, community-developed tool. It is **not
> produced, endorsed, or supported by Dynatrace**. Use at your own risk.

## Agent-friendly mode

**dtiam detects coding agents automatically.** When `CLAUDECODE`, `CURSOR_AGENT`,
`AIDER`, `AI_AGENT` or similar is set, `--plain` is implied — you get JSON, no
colors, and no confirmation prompts without asking for it. `-v` reports which
variable triggered it.

Pass `--plain` explicitly anyway when you care: it is harmless, and it makes the
intent clear if the environment ever lacks the variable. `--plain=false` forces
interactive behavior, and `DTIAM_NO_AGENT_DETECT=1` disables detection entirely.

Explicitly passing `--plain` when consuming dtiam output programmatically:

```bash
dtiam get groups --plain
```

`--plain` guarantees:

- Table and wide output become **JSON** on stdout
- All ANSI color codes are stripped
- Interactive confirmation prompts are **skipped** (acts like `--force`)
- Progress spinners and bars are suppressed
- Only structured data goes to stdout; messages go to stderr

Combine with `-o json` or `-o yaml` when you need a specific format regardless
of mode. `--dry-run` previews any mutating command without applying it.

### Stream contract

| Stream | Carries |
|--------|---------|
| stdout | Structured data only (JSON in `--plain`) |
| stderr | Progress, warnings, errors |
| exit 0 | Success |
| exit 1 | Error; message on stderr |

Never parse stderr for data, and never assume stdout is empty on failure —
check the exit code.

## Setup

```bash
export DTIAM_ACCOUNT_UUID=<account-uuid>
export DTIAM_CLIENT_SECRET=dt0s01.CLIENTID.SECRET   # client ID is derived from this
dtiam doctor --plain                                 # verify before doing work
```

`dtiam doctor` is the readiness gate: it checks config, credentials, scopes,
token retrieval, and API connectivity, and exits non-zero if any check fails.
Run it first; it will tell you exactly which of those is wrong rather than
leaving you to interpret an HTTP 403.

Use `dtiam doctor --offline` for the local checks only, with no network calls.

## Command shape

```
dtiam [global-flags] <verb> [<resource>] [<identifier>] [local-flags]
```

Lists use plural resources (`get groups`); single-item operations use singular
nouns (`create group`, `delete policy`). Every plural has a singular alias.

| Verb | Purpose |
|------|---------|
| `get` | List or retrieve |
| `describe` | Detailed single-resource view |
| `create` / `delete` | Mutate a single resource |
| `apply` | Declarative create-or-update from a file |
| `export` | Write resources to files |
| `analyze` | Permission analysis |
| `bulk` | Multi-resource operations from a file |
| `doctor` | Diagnose configuration and connectivity |

## Things that will surprise you

**Two permission models coexist.** A group's effective access is the **union**
of its IAM policy bindings (`dtiam group bindings`) and its direct role-style
permission grants (`dtiam group permissions`). Reading only one understates what
a group can do. `analyze` does not yet merge them.

**`get users` and `get env-users` hit different APIs.** `get users` lists the
account's user records from `api.dynatrace.com`. `get env-users` reports who is
visible at an environment, served from the environment itself and needing
`iam:users:read` granted *on the environment*. They return different fields.

**Audit log results can be partial and still return HTTP 200.** When a scan or
result-size limit is hit the API returns what it has plus a warning. dtiam
prints those warnings to stderr — do not treat a successful exit as proof the
audit trail is complete.

**Subscription cost is on API v3, everything else on v2.** Handled internally;
mentioned because the version skew is real and surprising.

**Not every endpoint paginates, and the ones that do disagree on how.** dtiam
follows pagination to completion inside `List`, so you always receive the whole
collection. Do not add your own paging.

**Response keys are not what the documentation says.** Six commands once returned
empty lists because the handlers read `items` where the API sends `data`,
`results`, or `content`. The shapes are now pinned in
`pkg/resources/response_shapes_test.go` and tabulated in
`.claude/architecture.md`. If you add an endpoint, verify its shape against a
live account — an unmatched key yields an empty list, not an error.

**`dtiam diff -f FILE` exits 1 on drift.** That is a result, not a failure, and no
error line is printed. Use `--exit-zero` if you need exit 0 regardless.

**`--watch` is refused with `--plain`.** The output would be an unparseable JSON
stream. Poll the command on a timer instead.

## OAuth scopes

All scopes dtiam needs are requested by default (`auth.DefaultScopeList`).
Dynatrace grants the intersection of requested and granted scopes, so a client
missing one still gets a usable token for everything else — which is why a 403
on one command and success on another is normal.

Override with `DTIAM_SCOPES` (space-separated, as OAuth2 requires). An override
**replaces** the default set; `dtiam doctor` will name any default scope your
override omits.

## Safety rules for agents

1. **Run `dtiam doctor --plain` first.** Do not diagnose auth failures by trial.
2. **Use `--dry-run` before any mutation** you have not performed before.
3. **`--plain` skips confirmation prompts.** That is deliberate, since agents
   have no stdin — but it means destructive commands execute immediately. Pair
   with `--dry-run` first.
4. **`group grant-permission --replace` removes every other grant.** Prefer the
   default additive behavior unless replacement is explicitly intended.
5. **Never invent permission names.** Get the valid set from
   `dtiam get available-permissions --plain`.
6. **Check `len()` of a list before acting on `[0]`.** An empty list is a
   legitimate result.

## Repository conventions

Rules live in `.claude/rules/` and are authoritative:

| File | Covers |
|------|--------|
| `core.md` | Branching, versioning, CHANGELOG, pre-push checklist |
| `command-standards.md` | Command structure, flags, output, dry-run, confirmation |
| `go.md` | Go style, imports, error wrapping, adding commands and handlers |
| `testing.md` | Table-driven tests, what to test |
| `development.md` | Build, auth, configuration |
| `deployment.md` | Release process |
| `existing-code.md` | Reading before changing, safe refactoring |
| `debugging.md` | Common failures, required scopes |

Key points: feature branches always; version bump plus CHANGELOG entry before
merge; `make test` and `make lint` must pass; commands return errors rather than
calling `os.Exit`; all data output goes through the printer, never `fmt.Printf`.

Architecture: `.claude/architecture.md` (includes the full endpoint table).
Decisions and their rationale: `.claude/DECISIONS.md`.

## Verify your changes

```bash
make build && make test && make lint
./bin/dtiam doctor --offline
```
