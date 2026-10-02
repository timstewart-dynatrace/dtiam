# Command Reference

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**.

Complete reference for all dtiam commands and their options.

![dtiam command hierarchy: core CRUD verbs, resource-scoped verbs, specialized workflows, and the resource vocabulary](../images/04-command-hierarchy_930x500.svg)

<!-- MARKDOWN_TABLE_ALTERNATIVE
| Tier | Verbs | Notes |
|------|-------|-------|
| Core verbs | `get`, `describe`, `create`, `delete`, `apply` | Universal CRUD plus declarative apply |
| Resource-scoped | `user`, `group`, `service-user`, `boundary`, `account`, `config` | Advanced operations grouped by the resource they act on |
| Specialized workflows | `bulk`, `template`, `analyze`, `export`, `cache` | Bulk processing, templates, analysis, backup, caching |
| Resource vocabulary | `groups`, `users`, `service-users`, `policies`, `bindings`, `boundaries`, `environments`, `subscriptions`, `tokens`, `apps`, `schemas`, `limits` | Plural for lists, singular for single items |

Global flags: `-c context`, `-o output`, `-v verbose`, `--plain`, `--dry-run`.
-->

## Table of Contents

- [Global Options](#global-options)
- [Environment Variables](#environment-variables)
- [config](#config) - Configuration management
- [get](#get) - List/retrieve resources
- [describe](#describe) - Detailed resource information
- [create](#create) - Create resources
- [delete](#delete) - Delete resources
- [user](#user) - User management
- [service-user](#service-user) - Service user (OAuth client) management
- [group](#group) - Advanced group operations
- [token](#token) - Platform token lifecycle
- [auth](#auth) - whoami and can-i
- [commands](#commands) - Machine-readable command catalog
- [edit](#edit) - Edit resources in $EDITOR
- [boundary](#boundary) - Boundary management
- [account](#account) - Account limits and subscriptions
- [cache](#cache) - Cache management
- [bulk](#bulk) - Bulk operations from files
- [export](#export) - Export resources for backup/migration
- [analyze](#analyze) - Permission analysis and compliance

---

## Global Options

These options apply to all commands:

```bash
dtiam [OPTIONS] COMMAND [ARGS]
```

| Option            | Short | Description                                                        |
| ----------------- | ----- | ------------------------------------------------------------------ |
| `--context TEXT`  |       | Override the current context                                       |
| `--output FORMAT` | `-o`  | Output format: table, wide, json, yaml, csv, plain                 |
| `--verbose`       | `-v`  | Enable verbose/debug output                                        |
| `--plain`         |       | Plain output mode (no colors, no prompts)                          |
| `--dry-run`       |       | Preview changes without applying them                              |
| `--agent`         | `-A`  | Agent mode: one JSON envelope on stdout (see [Agent mode](#agent-mode)) |
| `--help`          | `-h`  | Show help message                                                  |

Use `dtiam version` for the version.

### Safety levels

Each context has a safety level, set with `dtiam config set-context NAME --safety-level LEVEL`:

| Level       | Allows                                   | Blocks                                                                 |
| ----------- | ---------------------------------------- | ---------------------------------------------------------------------- |
| `readonly`  | read commands                            | every change; the OAuth token is also requested with read scopes only |
| `no-delete` | read, create, update                     | deletes and anything that removes access: members, bindings, boundary detach, permission revoke, `--replace` |
| `readwrite` | everything (default)                     | nothing                                                                |

A blocked command fails before any API call. `--dry-run` is allowed at every
level. `dtiam commands` shows each command's operation. In a `readonly` context
`get tokens` is unavailable, because listing tokens needs the token-management
scope, which can also create them.

### Agent mode

`--agent` (`-A`, or `DTIAM_AGENT=1`) makes every command write exactly one JSON
document to stdout, success or failure:

```json
{
  "ok": true,
  "result": [ ... ],
  "error": null,
  "context": {
    "command": "dtiam get groups", "operation": "read", "total": 138,
    "exit_code": 0, "duration": "452ms", "messages": [], "warnings": [], "version": "3.3.0"
  }
}
```

On failure `ok` is `false` and `error` has `code` (`safety_blocked`, `usage`,
`bad_request`, `auth_required`, `permission_denied`, `not_found`, `conflict`,
`rate_limited`, `server_error`, `error`), `message`, and sometimes
`status_code` and `suggestions`. A result that exits non-zero without failing,
such as `diff` finding drift or `auth can-i` answering no, is `ok: true` with
`exit_code: 1`. Agent mode implies `--plain`. A detected coding agent still gets
`--plain` automatically; the envelope is opt-in.

---

## Environment Variables

dtiam supports authentication and configuration via environment variables:

### Authentication Variables

| Variable              | Description            | Use Case                 |
| --------------------- | ---------------------- | ------------------------ |
| `DTIAM_BEARER_TOKEN`  | Static bearer token    | Quick testing, debugging |
| `DTIAM_CLIENT_ID`     | OAuth2 client ID       | Automation (recommended) |
| `DTIAM_CLIENT_SECRET` | OAuth2 client secret   | Automation (recommended) |
| `DTIAM_ACCOUNT_UUID`  | Dynatrace account UUID | Required for all methods |

### Configuration Variables

| Variable        | Description                   |
| --------------- | ----------------------------- |
| `DTIAM_CONTEXT` | Override current context name |
| `DTIAM_OUTPUT`  | Default output format         |
| `DTIAM_VERBOSE` | Enable verbose mode           |

### Authentication Priority

When multiple authentication methods are configured:

1. **Bearer Token** - `DTIAM_BEARER_TOKEN` + `DTIAM_ACCOUNT_UUID`
2. **OAuth2 (env)** - `DTIAM_CLIENT_ID` + `DTIAM_CLIENT_SECRET` + `DTIAM_ACCOUNT_UUID`
3. **Config file** - Context with OAuth2 credentials

### OAuth2 vs Bearer Token

| Feature          | OAuth2 (Recommended) | Bearer Token       |
| ---------------- | -------------------- | ------------------ |
| Auto-refresh     | ✅ Yes               | ❌ No              |
| Long-running     | ✅ Suitable          | ❌ Not recommended |
| Automation       | ✅ Recommended       | ❌ Not recommended |
| Quick testing    | ✅ Works             | ✅ Ideal           |
| Setup complexity | Medium               | Low                |

**Example: OAuth2 Authentication**

```bash
export DTIAM_CLIENT_ID="dt0s01.XXXXX"
export DTIAM_CLIENT_SECRET="dt0s01.XXXXX.YYYYY"
export DTIAM_ACCOUNT_UUID="abc-123-def"
dtiam get groups
```

**Example: Bearer Token Authentication**

```bash
# WARNING: Token will NOT auto-refresh!
export DTIAM_BEARER_TOKEN="dt0c01.XXXXX.YYYYY..."
export DTIAM_ACCOUNT_UUID="abc-123-def"
dtiam get groups
```

---

## config

Manage configuration contexts and credentials.

### config view

Display the current configuration.

```bash
dtiam config view
```

### config get-contexts

List all configured contexts.

```bash
dtiam config get-contexts
```

### config use-context

Switch to a different context.

```bash
dtiam config use-context NAME
```

| Argument | Description               |
| -------- | ------------------------- |
| `NAME`   | Context name to switch to |

### config set-context

Create or update a context.

```bash
dtiam config set-context NAME [OPTIONS]
```

| Argument/Option     | Description                                                     |
| ------------------- | --------------------------------------------------------------- |
| `NAME`              | Context name                                                    |
| `--account-uuid`    | Dynatrace account UUID                                          |
| `--credentials-ref` | Reference to a named credential                                 |
| `--safety-level`    | `readonly`, `no-delete` or `readwrite` (see [Safety levels](#safety-levels)) |

**Examples:**

```bash
dtiam config set-context prod --account-uuid abc-123 --credentials-ref prod-creds
dtiam config set-context prod --safety-level readonly
```

### config delete-context

Delete a context.

```bash
dtiam config delete-context NAME
```

### config set-credentials

Create or update a credential set.

A new credential needs both `--client-id` and `--client-secret`. An existing one
can be updated with any subset of flags; fields you do not pass keep their
values. Pass a flag with an empty value (e.g. `--environment-token ""`) to clear it.

```bash
dtiam config set-credentials NAME [OPTIONS]
```

| Argument/Option       | Description                                                                 |
| --------------------- | --------------------------------------------------------------------------- |
| `NAME`                | Credential name                                                             |
| `--client-id`         | OAuth2 client ID                                                            |
| `--client-secret`     | OAuth2 client secret (stored in the OS keyring)                             |
| `--api-url`           | Alternative host for all Account Management API calls                       |
| `--environment-url`   | Default environment for `get apps`, `get schemas`, `get env-users`, `get env-groups` |
| `--environment-token` | Token for those environment commands (stored in the OS keyring)             |
| `--no-keyring`        | Store secrets in the config file instead of the keyring                     |
| `--require-keyring`   | Fail rather than fall back to plaintext storage                             |

Secrets go to the OS keyring when one is available, and the config file holds
only a reference. Where each secret ended up is always reported.

**Examples:**

```bash
dtiam config set-credentials prod-creds --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY
dtiam config set-credentials prod-creds --environment-url abc12345 --environment-token dt0s16.XXX
dtiam config set-credentials dev-creds --api-url https://api.example.com
```

### config delete-credentials

Delete stored credentials.

```bash
dtiam config delete-credentials NAME
```

### config get-credentials

List all stored credentials.

```bash
dtiam config get-credentials
```

---

## get

List or retrieve IAM resources.

### get groups

List or get IAM groups.

```bash
dtiam get groups [IDENTIFIER] [OPTIONS]
```

| Argument/Option | Short | Description                   |
| --------------- | ----- | ----------------------------- |
| `IDENTIFIER`    |       | Group UUID or name (optional) |
| `--output`      | `-o`  | Output format                 |

### get users

List or get IAM users.

```bash
dtiam get users [IDENTIFIER] [OPTIONS]
```

| Argument/Option | Short | Description                  |
| --------------- | ----- | ---------------------------- |
| `IDENTIFIER`    |       | User UID or email (optional) |
| `--output`      | `-o`  | Output format                |

### get policies

List or get IAM policies.

```bash
dtiam get policies [IDENTIFIER] [OPTIONS]
```

| Argument/Option | Short | Description                             |
| --------------- | ----- | --------------------------------------- |
| `IDENTIFIER`    |       | Policy UUID or name (optional)          |
| `--level`       | `-l`  | Policy level: account (default), global |
| `--output`      | `-o`  | Output format                           |

### get bindings

List IAM policy bindings.

```bash
dtiam get bindings [OPTIONS]
```

| Option     | Short | Description          |
| ---------- | ----- | -------------------- |
| `--group`  | `-g`  | Filter by group UUID |
| `--output` | `-o`  | Output format        |

### get environments

List or get Dynatrace environments.

```bash
dtiam get environments [IDENTIFIER] [OPTIONS]
```

| Argument/Option | Short | Description                       |
| --------------- | ----- | --------------------------------- |
| `IDENTIFIER`    |       | Environment ID or name (optional) |
| `--output`      | `-o`  | Output format                     |

### get boundaries

List or get IAM policy boundaries.

```bash
dtiam get boundaries [IDENTIFIER] [OPTIONS]
```

| Argument/Option | Short | Description                      |
| --------------- | ----- | -------------------------------- |
| `IDENTIFIER`    |       | Boundary UUID or name (optional) |
| `--output`      | `-o`  | Output format                    |

### get tokens

List or get platform tokens.

```bash
dtiam get tokens [IDENTIFIER] [OPTIONS]
```

| Argument/Option | Short | Description                    |
| --------------- | ----- | ------------------------------ |
| `IDENTIFIER`    |       | Token ID (optional)            |
| `--output`      | `-o`  | Output format                  |

### get apps

List apps from the App Engine Registry. Requires `--environment`.

```bash
dtiam get apps [IDENTIFIER] --environment ENV [OPTIONS]
```

| Argument/Option | Short | Description                              |
| --------------- | ----- | ---------------------------------------- |
| `IDENTIFIER`    |       | App ID (optional)                        |
| `--environment` |       | Environment ID or URL (required)         |
| `--output`      | `-o`  | Output format                            |

### get schemas

List Settings 2.0 schemas from the Environment API. Requires `--environment`.

```bash
dtiam get schemas [IDENTIFIER] --environment ENV [OPTIONS]
```

| Argument/Option | Short | Description                              |
| --------------- | ----- | ---------------------------------------- |
| `IDENTIFIER`    |       | Schema ID (optional)                     |
| `--environment` |       | Environment ID or URL (required)         |
| `--name`        |       | Filter schemas by name pattern           |
| `--output`      | `-o`  | Output format                            |

### Watching a collection

`get groups`, `get users`, `get policies`, and `get bindings` accept `--watch`
(`-w`) to poll and reprint when the result changes.

| Option             | Short | Description                              |
| ------------------ | ----- | ---------------------------------------- |
| `--watch`          | `-w`  | Poll and reprint on change (Ctrl-C stops) |
| `--watch-interval` |       | Polling interval (default 10s, min 2s)   |

Output is reprinted **only when the result actually changes**: the comparison
sorts items first, so the API's unstable list order is not mistaken for a change.
A failed poll is reported on stderr and the watch continues, so a transient API
error does not end a session you left running.

Not supported with `--plain` — the output would be an unparseable JSON stream.
Poll the command on a timer instead.

```bash
# Watch group membership land during a migration
dtiam get groups --watch

# Slower polling
dtiam get bindings --watch --watch-interval 60s
```

### get audit-logs

List account audit log entries: who changed what in the account. Requires the
`account-audit-logs-read` scope.

The API enforces server-side scan and result-size limits. When a limit is hit it
returns a partial result with a warning rather than an error; dtiam prints those
warnings to stderr so a truncated audit trail is never shown as complete.

```bash
dtiam get audit-logs [OPTIONS]
```

| Argument/Option    | Short | Description                                                  |
| ------------------ | ----- | ------------------------------------------------------------ |
| `--start`          |       | Start of window (ISO-8601, epoch ms, or relative `now-24h`)   |
| `--end`            |       | End of window                                                |
| `--event-type`     |       | Filter by event type, e.g. `CREATE`, `UPDATE`, `DELETE`       |
| `--user`           |       | Filter by the user who performed the action                  |
| `--filter`         |       | Raw filter expression (overrides `--event-type` and `--user`) |
| `--limit`          |       | Maximum entries to return                                    |
| `--scan-limit-gb`  |       | Server-side scan limit in gigabytes                          |
| `--result-limit-mb`|       | Maximum result size in megabytes                             |
| `--add-fields`     |       | Additional audit fields to include                           |
| `--output`         | `-o`  | Output format                                                |

```bash
# Last 24 hours
dtiam get audit-logs --start now-24h

# Deletions only, most recent 50
dtiam get audit-logs --start now-7d --event-type DELETE --limit 50

# Everything one user did
dtiam get audit-logs --start now-7d --user alice@example.com
```

### get available-permissions

List every permission the account can grant. This is reference data describing
what the account *accepts*, not what is currently assigned — use
`dtiam group permissions GROUP` for that. Requires the `account-env-read` scope.

```bash
dtiam get available-permissions [OPTIONS]
```

| Argument/Option | Short | Description                                            |
| --------------- | ----- | ------------------------------------------------------ |
| `--name`        |       | Filter by ID or description (case-insensitive) |
| `--output`      | `-o`  | Output format                                          |

### get env-users

Search users through the environment-level Platform IAM API.

This is a **different API** from `get users`. That command lists the account's
user records; this one reports which users are actually visible and assigned at
an environment, served from `https://{env}.apps.dynatrace.com/platform/iam/v1`
rather than from `api.dynatrace.com`. Requires the `iam:users:read` scope. dtiam
requests it for this command only, so an OAuth client without it fails here and
nowhere else; an environment token (`DTIAM_ENVIRONMENT_TOKEN` or the credential's
`environment-token`) is used instead when configured.

The API will not enumerate all users — a search term or UUID is required.

```bash
dtiam get env-users [OPTIONS]
```

| Argument/Option | Short | Description                                             |
| --------------- | ----- | ------------------------------------------------------- |
| `--environment` |       | Environment ID or URL (defaults to `DTIAM_ENVIRONMENT_URL`, then the credential's `environment-url`) |
| `--search`      |       | Partial email or name to search for                     |
| `--uuid`        |       | User UUID to look up                                    |
| `--level`       |       | Organizational level: `account` or `environment`        |
| `--output`      | `-o`  | Output format                                           |

### get env-groups

List groups through the environment-level Platform IAM API. Same API as
`get env-users`. The API will not enumerate every group: `--search` is required
and must be at least 3 characters. dtiam requests the `iam:groups:read` scope for
this command only, or uses an environment token when one is configured.

```bash
dtiam get env-groups [OPTIONS]
```

| Argument/Option | Short | Description                                             |
| --------------- | ----- | ------------------------------------------------------- |
| `--environment` |       | Environment ID or URL (defaults to `DTIAM_ENVIRONMENT_URL`, then the credential's `environment-url`) |
| `--search`      |       | Partial group name, at least 3 characters (required)    |
| `--level`       |       | Organizational level: `account` or `environment`        |
| `--output`      | `-o`  | Output format                                           |

---

## describe

Show detailed resource information.

### describe group

Show detailed information about an IAM group.

```bash
dtiam describe group IDENTIFIER [--output FORMAT]
```

Displays: UUID, name, description, member count, members list, policy bindings.

### describe user

Show detailed information about an IAM user.

```bash
dtiam describe user IDENTIFIER [--output FORMAT]
```

Displays: UID, email, status, creation date, group memberships.

### describe policy

Show detailed information about an IAM policy.

```bash
dtiam describe policy IDENTIFIER [OPTIONS]
```

| Option     | Short | Description                             |
| ---------- | ----- | --------------------------------------- |
| `--level`  | `-l`  | Policy level: account (default), global |
| `--output` | `-o`  | Output format                           |

Displays: UUID, name, description, statement query, parsed permissions.

### describe environment

Show detailed information about a Dynatrace environment.

```bash
dtiam describe environment IDENTIFIER [--output FORMAT]
```

### describe boundary

Show detailed information about an IAM policy boundary.

```bash
dtiam describe boundary IDENTIFIER [--output FORMAT]
```

Displays: UUID, name, description, boundary query, attached policies count.

---

## create

Create IAM resources.

### create group

Create a new IAM group.

```bash
dtiam create group [OPTIONS]
```

| Option          | Short | Description           |
| --------------- | ----- | --------------------- |
| `--name`        | `-n`  | Group name (required) |
| `--description` | `-d`  | Group description     |
| `--output`      | `-o`  | Output format         |

**Example:**

```bash
dtiam create group --name "DevOps Team" --description "Platform engineering"
```

### create policy

Create a new IAM policy.

```bash
dtiam create policy [OPTIONS]
```

| Option          | Short | Description                       |
| --------------- | ----- | --------------------------------- |
| `--name`        | `-n`  | Policy name (required)            |
| `--statement`   | `-s`  | Policy statement query (required) |
| `--description` | `-d`  | Policy description                |
| `--output`      | `-o`  | Output format                     |

**Example:**

```bash
dtiam create policy --name "viewer" --statement "ALLOW settings:objects:read;"
```

### create binding

Create a policy binding (bind a policy to a group).

```bash
dtiam create binding [OPTIONS]
```

| Option       | Short | Description                                    |
| ------------ | ----- | ---------------------------------------------- |
| `--group`    | `-g`  | Group UUID or name (required)                  |
| `--policy`   | `-p`  | Policy UUID or name (required)                 |
| `--boundary` | `-b`  | Boundary UUID or name (optional)               |
| `--param`    |       | Bind parameter as key=value (repeatable)       |
| `--output`   | `-o`  | Output format                                  |

**Examples:**

```bash
dtiam create binding --group "DevOps Team" --policy "admin-policy"
dtiam create binding --group "DevOps Team" --policy "env-policy" --param env=production --param region=us-east
```

### create boundary

Create a new IAM policy boundary.

```bash
dtiam create boundary [OPTIONS]
```

| Option          | Short | Description                        |
| --------------- | ----- | ---------------------------------- |
| `--name`        | `-n`  | Boundary name (required)           |
| `--zones`       | `-z`  | Management zones (comma-separated) |
| `--query`       | `-q`  | Custom boundary query              |
| `--description` | `-d`  | Boundary description               |
| `--output`      | `-o`  | Output format                      |

Either `--zones` or `--query` must be provided.

**Example:**

```bash
dtiam create boundary --name "prod-only" --zones "Production,Staging"
```

### create token

Create a new platform token. The token value is only returned once during
creation -- save it immediately (with `-o json` it is in the `token` field).

The owner (`--user`) must be the identity dtiam authenticates as -- for an OAuth
client, the service user behind it. The API refuses (HTTP 403) to mint tokens for
anyone else.

```bash
dtiam create token --name NAME --user USER --scopes SCOPES [OPTIONS]
```

| Option          | Short | Description                                                          |
| --------------- | ----- | -------------------------------------------------------------------- |
| `--name`        | `-n`  | Token name (required)                                                |
| `--user`        |       | Owning user or service user, by email or UID (required)              |
| `--scopes`      |       | Comma-separated scopes (required)                                    |
| `--expires-in`  |       | Lifetime: `30d`, `2w`, `1y`, `12h` (default `30d`)                   |
| `--expires-at`  |       | Exact expiration, RFC 3339                                           |
| `--environment` |       | Limit to these environment IDs (repeatable); default is the account  |
| `--resource`    |       | Resource URN, e.g. `urn:dtaccount:UUID` (repeatable)                 |
| `--tag`         |       | Tag to attach (repeatable)                                           |

**Examples:**

```bash
dtiam create token --name "CI Token" --user ci-bot@example.com --scopes account-idm-read
dtiam create token --name "Logs" --user 1a2b3c4d-... --scopes storage:logs:read \
  --environment abc12345 --expires-at 2027-01-01T00:00:00Z
```

---

## delete

Delete IAM resources.

### delete group

Delete an IAM group.

```bash
dtiam delete group IDENTIFIER [--force]
```

| Option    | Short | Description       |
| --------- | ----- | ----------------- |
| `--force` | `-f`  | Skip confirmation |

### delete policy

Delete an IAM policy.

```bash
dtiam delete policy IDENTIFIER [OPTIONS]
```

| Option    | Short | Description       |
| --------- | ----- | ----------------- |
| `--force` | `-f`  | Skip confirmation |

### delete binding

Delete a policy binding.

```bash
dtiam delete binding [OPTIONS]
```

| Option     | Short | Description            |
| ---------- | ----- | ---------------------- |
| `--group`  | `-g`  | Group UUID (required)  |
| `--policy` | `-p`  | Policy UUID (required) |
| `--force`  | `-f`  | Skip confirmation      |

### delete boundary

Delete an IAM policy boundary.

```bash
dtiam delete boundary IDENTIFIER [--force]
```

### delete user

Delete an IAM user.

```bash
dtiam delete user IDENTIFIER [--force]
```

### delete service-user

Delete a service user.

```bash
dtiam delete service-user IDENTIFIER [--force]
```

### delete token

Delete a platform token.

```bash
dtiam delete token IDENTIFIER [--force]
```

---

## user

User management operations.

### user create

Create a new user in the account.

```bash
dtiam user create EMAIL [OPTIONS]
```

| Argument/Option | Short | Description                          |
| --------------- | ----- | ------------------------------------ |
| `EMAIL`         |       | User email address (required)        |
| `--first-name`  |       | User's first name                    |
| `--last-name`   |       | User's last name                     |
| `--groups`      | `-g`  | Comma-separated group UUIDs or names |
| `--output`      | `-o`  | Output format                        |

**Examples:**

```bash
dtiam user create user@example.com
dtiam user create user@example.com --first-name John --last-name Doe
dtiam user create user@example.com --groups "DevOps,Platform"
```

### user info

Show detailed information about a user. Equivalent to `describe user`.

```bash
dtiam user info IDENTIFIER
```

| Argument      | Description                      |
| ------------- | -------------------------------- |
| `IDENTIFIER`  | User UID or email address        |

**Examples:**

```bash
dtiam user info alice@example.com
dtiam user info alice@example.com -o json
```

### user add-to-groups

Add a user to multiple groups.

```bash
dtiam user add-to-groups EMAIL [OPTIONS]
```

| Argument/Option | Short | Description                                     |
| --------------- | ----- | ----------------------------------------------- |
| `EMAIL`         |       | User email address                              |
| `--groups`      | `-g`  | Comma-separated group UUIDs or names (required) |

**Example:**

```bash
dtiam user add-to-groups user@example.com --groups "DevOps,Platform"
```

### user remove-from-groups

Remove a user from multiple groups.

```bash
dtiam user remove-from-groups EMAIL [OPTIONS]
```

| Argument/Option | Short | Description                                     |
| --------------- | ----- | ----------------------------------------------- |
| `EMAIL`         |       | User email address                              |
| `--groups`      | `-g`  | Comma-separated group UUIDs or names (required) |

### user replace-groups

Replace all group memberships for a user.

```bash
dtiam user replace-groups EMAIL [OPTIONS]
```

| Argument/Option | Short | Description                          |
| --------------- | ----- | ------------------------------------ |
| `EMAIL`         |       | User email address                   |
| `--groups`      | `-g`  | Comma-separated group UUIDs or names |

**Example:**

```bash
dtiam user replace-groups user@example.com --groups "DevOps,Platform"
```

### user list-groups

List all groups a user belongs to.

```bash
dtiam user list-groups IDENTIFIER [--output FORMAT]
```

---

## service-user

Service user (OAuth client) management.

Service users are used for programmatic API access. When you create a service user, you receive OAuth client credentials that can be used to authenticate API requests.

### service-user list

List all service users in the account.

```bash
dtiam service-user list [--output FORMAT]
```

### service-user get

Get details of a service user.

```bash
dtiam service-user get USER [--output FORMAT]
```

| Argument | Description               |
| -------- | ------------------------- |
| `USER`   | Service user UUID or name |

### service-user create

Create a new service user (OAuth client).

**IMPORTANT:** Save the client secret immediately - it cannot be retrieved later!

```bash
dtiam service-user create [OPTIONS]
```

| Option          | Short | Description                          |
| --------------- | ----- | ------------------------------------ |
| `--name`        | `-n`  | Service user name (required)         |
| `--description` | `-d`  | Description                          |
| `--groups`      | `-g`  | Comma-separated group UUIDs or names |
| `--output`      | `-o`  | Output format                        |

**Examples:**

```bash
dtiam service-user create --name "CI Pipeline"
dtiam service-user create --name "CI Pipeline" --groups "DevOps,Automation"
```

### service-user update

Update a service user.

```bash
dtiam service-user update USER [OPTIONS]
```

| Argument/Option | Short | Description               |
| --------------- | ----- | ------------------------- |
| `USER`          |       | Service user UUID or name |
| `--name`        | `-n`  | New name                  |
| `--description` | `-d`  | New description           |
| `--output`      | `-o`  | Output format             |

### service-user delete

Delete a service user.

```bash
dtiam service-user delete USER [--force]
```

**Warning:** Deleting a service user will invalidate any OAuth tokens issued to it.

### service-user add-to-group

Add a service user to a group.

```bash
dtiam service-user add-to-group USER [OPTIONS]
```

| Argument/Option | Short | Description                   |
| --------------- | ----- | ----------------------------- |
| `USER`          |       | Service user UUID or name     |
| `--group`       | `-g`  | Group UUID or name (required) |

### service-user remove-from-group

Remove a service user from a group.

```bash
dtiam service-user remove-from-group USER [OPTIONS]
```

| Argument/Option | Short | Description                   |
| --------------- | ----- | ----------------------------- |
| `USER`          |       | Service user UUID or name     |
| `--group`       | `-g`  | Group UUID or name (required) |

### service-user list-groups

List all groups a service user belongs to.

```bash
dtiam service-user list-groups USER [--output FORMAT]
```

---

## group

Advanced group operations.

### group members

List all members of a group.

```bash
dtiam group members IDENTIFIER [--output FORMAT]
```

### group add-member

Add a user to a group.

```bash
dtiam group add-member IDENTIFIER [OPTIONS]
```

| Argument/Option | Short | Description                        |
| --------------- | ----- | ---------------------------------- |
| `IDENTIFIER`    |       | Group UUID or name                 |
| `--email`       | `-e`  | User email address to add (required) |

### group remove-member

Remove a user from a group.

```bash
dtiam group remove-member IDENTIFIER [OPTIONS]
```

| Argument/Option | Short | Description                  |
| --------------- | ----- | ---------------------------- |
| `IDENTIFIER`    |       | Group UUID or name           |
| `--user`        | `-u`  | User email or UID to remove (required) |

### group update

Rename a group or change its description. Fields you do not pass keep their
current values.

```bash
dtiam group update IDENTIFIER [--name NAME] [--description TEXT]
```

| Argument/Option | Short | Description                                  |
| --------------- | ----- | -------------------------------------------- |
| `IDENTIFIER`    |       | Group UUID or name                           |
| `--name`        | `-n`  | New group name                               |
| `--description` | `-d`  | New description (`""` clears it)             |

```bash
dtiam group update "Platform Team" --name "Platform Engineering"
dtiam group update "Platform Team" --description "Owns the platform" --dry-run
```

### group bindings

List all policy bindings for a group.

```bash
dtiam group bindings IDENTIFIER [--output FORMAT]
```

### group clone

Clone an existing group with optional members and policy bindings.

```bash
dtiam group clone SOURCE --name NEW_NAME [OPTIONS]
```

| Option               | Short | Description                                |
| -------------------- | ----- | ------------------------------------------ |
| `--name`             | `-n`  | Name for the new group (required)          |
| `--description`      | `-d`  | Description for the new group              |
| `--include-members`  |       | Copy group members to the new group        |
| `--include-policies` |       | Copy policy bindings to the new group      |

**Examples:**

```bash
dtiam group clone "Production Team" --name "Staging Team"
dtiam group clone "Production Team" --name "Staging Team" --include-members --include-policies
dtiam group clone "Production Team" --name "Staging Team" --dry-run
```

### group setup

One-step group provisioning from a YAML/JSON policies file.

```bash
dtiam group setup --name NAME --policies-file FILE [OPTIONS]
```

| Option             | Short | Description                                       |
| ------------------ | ----- | ------------------------------------------------- |
| `--name`           | `-n`  | Name for the new group (required)                 |
| `--description`    | `-d`  | Group description                                 |
| `--policies-file`  | `-f`  | YAML or JSON file with policy definitions (required) |

**Policies file format:**

```yaml
policies:
  - name: "ReadOnly Policy"
    boundaries:
      - "boundary-uuid-1"
  - name: "Admin Policy"
```

**Examples:**

```bash
dtiam group setup --name "New Team" --policies-file policies.yaml
dtiam group setup --name "New Team" --policies-file policies.yaml --dry-run
```

### group permissions

List the permissions granted **directly** to a group.

These are role-style grants that predate IAM policies and still coexist with
them. A group's effective access is the union of its policy bindings and these
direct grants, so reviewing only `group bindings` understates what a group can
actually do. Requires the `account-idm-read` scope.

```bash
dtiam group permissions IDENTIFIER [OPTIONS]
```

| Argument/Option | Short | Description                 |
| --------------- | ----- | --------------------------- |
| `IDENTIFIER`    |       | Group name or UUID (required) |
| `--output`      | `-o`  | Output format               |

### group grant-permission

Grant a direct permission to a group. Requires the `account-idm-write` scope.

```bash
dtiam group grant-permission IDENTIFIER --permission NAME --scope SCOPE [OPTIONS]
```

| Argument/Option | Description                                                      |
| --------------- | ---------------------------------------------------------------- |
| `IDENTIFIER`    | Group name or UUID (required)                                    |
| `--permission`  | Permission name (required) — see `get available-permissions`      |
| `--scope`       | Scope value (required)                                           |
| `--scope-type`  | `account`, `tenant`, or `management-zone` (default `tenant`)      |
| `--replace`     | Replace **all** existing grants instead of adding — destructive   |
| `--dry-run`     | Preview without applying                                         |

Scope values by scope type:

| `--scope-type`    | `--scope` value                        |
| ----------------- | -------------------------------------- |
| `account`         | The account UUID                       |
| `tenant`          | The environment ID                     |
| `management-zone` | `{environment-id}:{management-zone-id}` |

**Examples:**

```bash
# Environment viewer access
dtiam group grant-permission "Dev Team" --permission tenant-viewer --scope abc12345

# Account-level user management
dtiam group grant-permission "Admins" \
  --permission account-user-management --scope $DTIAM_ACCOUNT_UUID --scope-type account

# Scoped to one management zone
dtiam group grant-permission "Dev Team" \
  --permission tenant-viewer --scope "abc12345:-1234567890" --scope-type management-zone
```

### group revoke-permission

Revoke a direct permission grant from a group. The grant is identified by the
combination of permission name, scope, and scope type, because a grant has no
identifier of its own — all three must match exactly.

Requires confirmation unless `--force` or `--plain` is set.

```bash
dtiam group revoke-permission IDENTIFIER --permission NAME --scope SCOPE [OPTIONS]
```

| Argument/Option | Short | Description                                                 |
| --------------- | ----- | ----------------------------------------------------------- |
| `IDENTIFIER`    |       | Group name or UUID (required)                               |
| `--permission`  |       | Permission name (required)                                  |
| `--scope`       |       | Scope the permission was granted on (required)              |
| `--scope-type`  |       | `account`, `tenant`, or `management-zone` (default `tenant`) |
| `--force`       | `-f`  | Skip confirmation                                           |
| `--dry-run`     |       | Preview without applying                                    |

---

## token

Platform token lifecycle. Create, list and delete tokens with `create token`,
`get tokens` and `delete token`. Requires the `platform-token:tokens:manage` scope.

### token deactivate

Set a token's status to `INACTIVE`. Anything using it loses access immediately;
the token is kept and can be reactivated. Asks for confirmation unless `--force`
or `--plain` is set.

```bash
dtiam token deactivate TOKEN_ID [--force]
```

### token activate

Set a token's status back to `ACTIVE`.

```bash
dtiam token activate TOKEN_ID
```

### token set-expiration

Change when a token expires. The date is RFC 3339 and is validated before any
request, including under `--dry-run`.

```bash
dtiam token set-expiration TOKEN_ID --date 2027-06-30T00:00:00Z
```

---

## auth

### auth whoami

Show the identity dtiam authenticates as -- read from the access token -- with
its user record, groups, OAuth client, context and safety level. With an OAuth
client this is the user or service user behind the client.

```bash
dtiam auth whoami [-o json]
```

### auth can-i

Ask the effective-permissions API whether the caller, `--user` or `--group`
holds a permission. Answers `yes`, `no`, or `conditional` (granted only where
listed conditions hold, such as a boundary or a bound group).

```bash
dtiam auth can-i PERMISSION [--user USER | --group GROUP] [--environment ENV] [--strict]
```

| Argument/Option | Description                                            |
| --------------- | ------------------------------------------------------ |
| `PERMISSION`    | A policy permission, e.g. `storage:logs:read`          |
| `--user`        | Check this user (email or UID) instead of the caller  |
| `--group`       | Check this group (UUID or name) instead of the caller |
| `--environment` | Check at environment level instead of account level   |
| `--strict`      | Treat `conditional` as no                              |

Exit code 0 for yes/conditional, 1 for no. This covers platform permissions
granted by IAM policies. Account Management access (managing users, groups and
policies through `api.dynatrace.com`) comes from account permissions on groups,
which the API does not report -- see `group permissions`.

```bash
dtiam auth can-i iam:bindings:write
dtiam auth can-i storage:logs:read --user alice@example.com --environment abc12345
```

---

## commands

List every command with its usage, flags, and operation (read, create, update,
delete -- the classification safety levels enforce). Intended for agents:
`dtiam commands -o json` describes the CLI in one call.

```bash
dtiam commands [--brief] [-o json|yaml]
```

The table view is always brief.

---

## boundary

Boundary attach/detach operations.

### boundary attach

Attach a boundary to an existing binding.

```bash
dtiam boundary attach [OPTIONS]
```

| Option       | Short | Description                      |
| ------------ | ----- | -------------------------------- |
| `--group`    | `-g`  | Group UUID or name (required)    |
| `--policy`   | `-p`  | Policy UUID or name (required)   |
| `--boundary` | `-b`  | Boundary UUID or name (required) |

**Example:**

```bash
dtiam boundary attach --group "DevOps" --policy "admin-policy" --boundary "prod-boundary"
```

### boundary detach

Detach a boundary from a binding.

```bash
dtiam boundary detach [OPTIONS]
```

| Option       | Short | Description                      |
| ------------ | ----- | -------------------------------- |
| `--group`    | `-g`  | Group UUID or name (required)    |
| `--policy`   | `-p`  | Policy UUID or name (required)   |
| `--boundary` | `-b`  | Boundary UUID or name (required) |

### boundary list-attached

List all bindings that use a boundary.

```bash
dtiam boundary list-attached BOUNDARY [--output FORMAT]
```

### boundary create-app-boundary

Create a boundary scoped to specific Dynatrace app IDs.

```bash
dtiam boundary create-app-boundary NAME --app-ids ID1,ID2 [OPTIONS]
```

| Option              | Description                                    |
| ------------------- | ---------------------------------------------- |
| `--app-ids`         | Comma-separated app IDs (required)             |
| `--not-in`          | Use NOT IN (exclude apps instead of allow)     |
| `--environment`     | Environment ID for app validation              |
| `--description`     | Boundary description                           |
| `--skip-validation` | Skip app ID validation                         |

**Examples:**

```bash
dtiam boundary create-app-boundary "Dashboard Apps" --app-ids dynatrace.dashboards,dynatrace.notebooks
dtiam boundary create-app-boundary "No Classic" --app-ids dynatrace.classic.smartscape --not-in
dtiam boundary create-app-boundary "My Apps" --app-ids dynatrace.dashboards --environment abc12345
```

### boundary create-schema-boundary

Create a boundary scoped to specific Settings 2.0 schema IDs.

```bash
dtiam boundary create-schema-boundary NAME --schema-ids ID1,ID2 [OPTIONS]
```

| Option              | Description                                    |
| ------------------- | ---------------------------------------------- |
| `--schema-ids`      | Comma-separated schema IDs (required)          |
| `--not-in`          | Use NOT IN (exclude schemas instead of allow)  |
| `--environment`     | Environment ID for schema validation           |
| `--description`     | Boundary description                           |
| `--skip-validation` | Skip schema ID validation                      |

**Examples:**

```bash
dtiam boundary create-schema-boundary "Alerting Only" --schema-ids builtin:alerting.profile,builtin:alerting.maintenance-window
dtiam boundary create-schema-boundary "No Spans" --schema-ids builtin:span-attribute --not-in
```

---

## account

Account limits and subscription information.

### account limits

List account limits and quotas.

```bash
dtiam account limits [OPTIONS]
```

| Option      | Description                         |
| ----------- | ----------------------------------- |
| `--summary` | Show summary with usage percentages |
| `--output`  | Output format                       |

Shows current usage and maximum allowed values for account resources like users, groups, and environments.

**Examples:**

```bash
dtiam account limits
dtiam account limits --summary
```

### account check-capacity

Check if there is capacity for additional resources.

```bash
dtiam account check-capacity LIMIT_NAME [OPTIONS]
```

| Argument/Option | Description                                          |
| --------------- | ---------------------------------------------------- |
| `LIMIT_NAME`    | Name of the limit to check (required)                |
| `--additional`  | Number of additional resources to check (default: 1) |

**Examples:**

```bash
dtiam account check-capacity user-limit
dtiam account check-capacity group-limit --additional 5
```

### account subscriptions

List account subscriptions.

```bash
dtiam account subscriptions [--output FORMAT]
```

Shows all subscriptions including type, status, and time period.

### account forecast

Get usage forecast for subscriptions.

```bash
dtiam account forecast [--output FORMAT]
```

### account capabilities

List capability flags from subscriptions.

```bash
dtiam account capabilities [SUBSCRIPTION] [--output FORMAT]
```

When called without arguments, lists capabilities from all subscriptions.
When given a subscription UUID or name, lists capabilities for that subscription only.

```bash
dtiam account capabilities
dtiam account capabilities "Enterprise Plan"
dtiam account capabilities -o json
```

### account notifications

List account notifications, newest first: budget, cost, forecast,
bring-your-own-key, and environment upgrade/downgrade events. Every matching
notification is returned (dtiam follows the API's pages). Requires the
`account-uac-read` scope.

Filter values are validated locally, so a mistyped type or severity fails with a
clear message instead of silently matching nothing.

```bash
dtiam account notifications [OPTIONS]
```

| Option       | Description                                                             |
| ------------ | ----------------------------------------------------------------------- |
| `--start`    | Start of window (ISO-8601)                                              |
| `--end`      | End of window (ISO-8601)                                                |
| `--type`     | `FORECAST`, `BUDGET`, `COST`, `BYOK_REVOKED`, `BYOK_ACTIVATED`, `ENVIRONMENT_UPGRADE`, `ENVIRONMENT_DOWNGRADE` |
| `--severity` | `SEVERE`, `WARN`, `INFO`                                                |
| `--environment` | Filter by environment ID (repeatable)                                |
| `--capability`  | Filter by capability key, e.g. `FULLSTACK_MONITORING`                |
| `--output`   | Output format                                                           |

```bash
dtiam account notifications --severity SEVERE
dtiam account notifications --type BUDGET,COST \
  --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z
```

### account environment-usage

Show subscription usage broken down by monitoring environment.

This is distinct from `account subscriptions`, which reports only the usage
totals embedded in the subscription record. This command calls the dedicated
per-environment usage endpoint, so it can attribute consumption to individual
environments. One row is printed per environment, capability and period
(`-o wide` adds the end time, capability name and cluster). Requires the
`account-uac-read` scope.

```bash
dtiam account environment-usage [SUBSCRIPTION] --start TIME --end TIME [OPTIONS]
```

| Argument/Option | Description                                                        |
| --------------- | ------------------------------------------------------------------ |
| `SUBSCRIPTION`  | Subscription UUID or name (default: the account's only ACTIVE subscription) |
| `--start`       | Start of window, e.g. `2026-09-01T00:00:00Z` (required)             |
| `--end`         | End of window (required)                                           |
| `--environment` | Restrict to these environment IDs                                  |
| `--capability`  | Restrict to these capability keys                                  |
| `--output`      | Output format                                                      |

```bash
dtiam account environment-usage --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z
dtiam account environment-usage --start 2026-09-01T00:00:00Z --end 2026-10-01T00:00:00Z \
  --environment abc12345 --capability full_stack_monitoring
```

### account environment-cost

Show subscription cost broken down by monitoring environment.

> **Note:** per-environment cost and usage use the **v3** Subscription API, while
> subscription listing and forecast remain on v2. dtiam handles the version
> difference internally and follows every page. One row is printed per
> environment, capability and period.

```bash
dtiam account environment-cost [SUBSCRIPTION] --start TIME --end TIME [OPTIONS]
```

| Argument/Option | Description                                                     |
| --------------- | --------------------------------------------------------------- |
| `SUBSCRIPTION`  | Subscription UUID or name (default: the only ACTIVE subscription) |
| `--start`       | Start of window (required)                                      |
| `--end`         | End of window (required)                                        |
| `--output`      | Output format                                                   |

---

## cache

Cache management.

### cache stats

Show cache statistics.

```bash
dtiam cache stats
```

### cache clear

Clear cache entries.

```bash
dtiam cache clear
```

---

## bulk

Bulk operations for managing multiple IAM resources from files.

### bulk add-users-to-group

Add multiple users to a group from a file.

```bash
dtiam bulk add-users-to-group [OPTIONS]
```

| Option              | Short | Description                                    |
| ------------------- | ----- | ---------------------------------------------- |
| `--file`            | `-f`  | File with user emails (JSON, YAML, or CSV)     |
| `--group`           | `-g`  | Group UUID or name                             |
| `--email-field`     | `-e`  | Field name containing email addresses (default: "email") |
| `--continue-on-error` |     | Continue processing on errors                  |

**File Format Examples:**

JSON/YAML:
```json
[{"email": "user1@example.com"}, {"email": "user2@example.com"}]
```

CSV:
```csv
email
user1@example.com
user2@example.com
```

**Examples:**

```bash
dtiam bulk add-users-to-group --file users.csv --group "DevOps Team"
dtiam bulk add-users-to-group -f users.json -g DevOps --continue-on-error
```

### bulk remove-users-from-group

Remove multiple users from a group from a file.

```bash
dtiam bulk remove-users-from-group [OPTIONS]
```

| Option              | Short | Description                                    |
| ------------------- | ----- | ---------------------------------------------- |
| `--file`            | `-f`  | File with user emails/UIDs (JSON, YAML, or CSV)|
| `--group`           | `-g`  | Group UUID or name                             |
| `--user-field`      | `-u`  | Field name containing email or UID (default: "email") |
| `--continue-on-error` |     | Continue processing on errors                  |
| `--force`           | `-F`  | Skip confirmation prompt                       |

**Example:**

```bash
dtiam bulk remove-users-from-group --file users.csv --group "DevOps Team" --force
```

### bulk create-groups

Create multiple groups from a file.

```bash
dtiam bulk create-groups [OPTIONS]
```

| Option              | Short | Description                          |
| ------------------- | ----- | ------------------------------------ |
| `--file`            | `-f`  | File with group definitions (YAML)   |
| `--continue-on-error` |     | Continue processing on errors        |

**File Format:**

```yaml
groups:
  - name: "Group A"
    description: "Description for Group A"
  - name: "Group B"
    description: "Description for Group B"
```

**Example:**

```bash
dtiam bulk create-groups --file groups.yaml
dtiam bulk create-groups -f groups.yaml --dry-run
```

### bulk create-bindings

Create multiple policy bindings from a file.

```bash
dtiam bulk create-bindings [OPTIONS]
```

| Option              | Short | Description                            |
| ------------------- | ----- | -------------------------------------- |
| `--file`            | `-f`  | File with binding definitions (YAML)   |
| `--continue-on-error` |     | Continue processing on errors          |

**File Format:**

```yaml
bindings:
  - group: "group-uuid-or-name"
    policy: "policy-uuid-or-name"
    boundary: "optional-boundary-uuid"
  - group: "another-group"
    policy: "another-policy"
```

**Example:**

```bash
dtiam bulk create-bindings --file bindings.yaml
dtiam bulk create-bindings -f bindings.yaml --dry-run
```

### bulk export-group-members

Export group members to a file.

```bash
dtiam bulk export-group-members [OPTIONS]
```

| Option     | Short | Description                          |
| ---------- | ----- | ------------------------------------ |
| `--group`  | `-g`  | Group UUID or name (required)        |
| `--output` | `-o`  | Output file path                     |
| `--format` | `-F`  | Output format: csv, json, yaml (default: csv) |

**Examples:**

```bash
dtiam bulk export-group-members --group "DevOps Team" --output members.csv
dtiam bulk export-group-members -g DevOps -o members.json -F json
```

### bulk create-groups-with-policies

Create groups and bind policies from a CSV, JSON, or YAML file.

```bash
dtiam bulk create-groups-with-policies --file FILE [OPTIONS]
```

| Option               | Short | Description                          |
| -------------------- | ----- | ------------------------------------ |
| `--file`             | `-f`  | Input file (required)                |
| `--continue-on-error`|       | Continue processing on errors        |

CSV columns: `group_name`, `description`, `policy_name`, `boundary_name`

**Examples:**

```bash
dtiam bulk create-groups-with-policies --file groups-policies.csv
dtiam bulk create-groups-with-policies --file groups-policies.yaml --continue-on-error
dtiam bulk create-groups-with-policies --file groups-policies.csv --dry-run
```

---

## export

Export IAM resources for backup or migration.

### export all

Export all IAM resources to files.

```bash
dtiam export all [OPTIONS]
```

| Option           | Short | Description                                    |
| ---------------- | ----- | ---------------------------------------------- |
| `--output`       | `-o`  | Output directory (default: ".")                |
| `--format`       | `-f`  | Output format: csv, json, yaml (default: csv)  |
| `--prefix`       | `-p`  | File name prefix (default: "dtiam")            |
| `--include`      | `-i`  | Comma-separated list of exports to include     |
| `--detailed`     | `-d`  | Include detailed/enriched data                 |
| `--timestamp-dir`|       | Create timestamped subdirectory (default: true)|

Available exports: environments, groups, users, policies, bindings, boundaries

**Examples:**

```bash
dtiam export all                              # Export all to CSV in current dir
dtiam export all -o ./backup -f json          # Export as JSON to backup dir
dtiam export all --detailed                   # Include enriched data
dtiam export all -i groups,policies           # Only export groups and policies
```

### export group

Export a single group with its details.

```bash
dtiam export group IDENTIFIER [OPTIONS]
```

| Option             | Short | Description                      |
| ------------------ | ----- | -------------------------------- |
| `--output`         | `-o`  | Output file                      |
| `--format`         | `-f`  | Output format: yaml, json (default: yaml) |
| `--include-members`|       | Include member list (default: true) |
| `--include-policies`|      | Include policy bindings (default: true) |

**Examples:**

```bash
dtiam export group "DevOps Team"
dtiam export group DevOps -o devops.yaml
dtiam export group DevOps --include-members=false
```

### export policy

Export a single policy with its details.

```bash
dtiam export policy IDENTIFIER [OPTIONS]
```

| Option         | Short | Description                              |
| -------------- | ----- | ---------------------------------------- |
| `--output`     | `-o`  | Output file                              |
| `--format`     | `-f`  | Output format: yaml, json (default: yaml)|
| `--as-template`| `-t`  | Export as reusable template              |

**Examples:**

```bash
dtiam export policy "admin-policy"
dtiam export policy viewer -o viewer.yaml
dtiam export policy viewer --as-template -o viewer-template.yaml
```

### export environments

Export all environments to a file.

```bash
dtiam export environments [OPTIONS]
```

| Option       | Short | Description                                |
| ------------ | ----- | ------------------------------------------ |
| `--output`   | `-o`  | Output directory (default: `.`)            |
| `--format`   | `-f`  | Output format: csv, json, yaml (default: csv) |
| `--prefix`   | `-p`  | File name prefix (default: `dtiam`)        |
| `--detailed`  | `-d`  | Include detailed/enriched data             |

### export users

Export all users to a file. With `--detailed`, includes group membership counts.

```bash
dtiam export users [OPTIONS]
dtiam export users --detailed -f json
```

### export bindings

Export all policy bindings to a file. With `--detailed`, enriches with group/policy names.

```bash
dtiam export bindings [OPTIONS]
dtiam export bindings --detailed -f json
```

### export boundaries

Export all boundaries to a file. With `--detailed`, includes attached policy counts.

```bash
dtiam export boundaries [OPTIONS]
dtiam export boundaries --detailed -f json
```

### export service-users

Export all service users to a file.

```bash
dtiam export service-users [OPTIONS]
dtiam export service-users -f yaml -o ./backup
```

---

## template

Manage and use resource templates.

### template list

List all available templates (built-in and custom).

```bash
dtiam template list
```

### template show

Display a template's content and required variables.

```bash
dtiam template show NAME
```

### template render

Render a template with variable substitution to stdout.

```bash
dtiam template render NAME --set key=value [--set key2=value2]
```

**Examples:**

```bash
dtiam template render policy-readonly --set name=MyPolicy
dtiam template render group-team --set name=DevOps --set description="DevOps Team"
```

### template apply

Render a template and create the resulting resource.

```bash
dtiam template apply NAME --set key=value [--dry-run]
```

**Examples:**

```bash
dtiam template apply policy-readonly --set name=MyPolicy
dtiam template apply group-team --set name=DevOps --dry-run
```

### template save

Save a custom template from a file.

```bash
dtiam template save NAME --file TEMPLATE_FILE
```

### template delete

Delete a custom template.

```bash
dtiam template delete NAME [--force]
```

### template path

Show the filesystem path where custom templates are stored.

```bash
dtiam template path
```

---

## edit

Edit a group, policy or boundary in your editor (`$VISUAL`, then `$EDITOR`, then
`vi`), review the diff, and apply it.

```bash
dtiam edit group|policy|boundary IDENTIFIER [--force] [--from-file PATH] [--dry-run]
```

| Resource   | Editable fields                                   |
| ---------- | ------------------------------------------------- |
| `group`    | `name`, `description`                             |
| `policy`   | `name`, `description`, `statementQuery`, `tags`   |
| `boundary` | `name`, `boundaryQuery`                           |

| Option        | Short | Description                                                  |
| ------------- | ----- | ------------------------------------------------------------ |
| `--force`     | `-f`  | Apply without asking after showing the diff                  |
| `--from-file` |       | Start from previously saved edits instead of the live resource |
| `--dry-run`   |       | Edit and show the diff, but do not apply                     |

The file is YAML with `kind`, `metadata.uuid` and `spec`; read-only fields are
shown as comments. Saving without changes does nothing. If the edit cannot be
applied -- invalid YAML, a rejected change, or "no" at the prompt -- the file is
kept and the error shows the `--from-file` command to resume. `edit` refuses to
run with `--plain`, `--agent` or without a terminal; use `apply -f` for
automation. It counts as an update for safety levels.

```bash
dtiam edit policy "Read Only"
EDITOR="code --wait" dtiam edit group "Platform Team"
```

---

## apply

Create or update resources declaratively from YAML/JSON files.

```bash
dtiam apply -f FILE [--set key=value] [--dry-run]
```

| Option   | Short | Description                                |
| -------- | ----- | ------------------------------------------ |
| `--file` | `-f`  | Resource definition file (required)        |
| `--set`  |       | Template variable as key=value (repeatable)|

Supports `kind: Group|Policy|Boundary|Binding` with a `spec` section. Multiple documents supported via YAML `---` separators.

Resources are matched by `name` (bindings by `group` and `policy`). A missing resource is created; an existing one is updated to match the spec; one that already matches is reported as `unchanged`, so re-applying the same file is safe. Fields a spec omits keep their current values. Run `dtiam diff -f FILE` first to see which of the three each document will be.

**Examples:**

```bash
dtiam apply -f group.yaml
dtiam apply -f policy-template.yaml --set name=MyPolicy
dtiam apply -f all-resources.yaml --dry-run
```

---

## analyze

Analyze IAM permissions, effective permissions, and policy compliance.

### analyze user-permissions

Calculate effective permissions for a user based on group memberships and policy bindings.

```bash
dtiam analyze user-permissions USER [OPTIONS]
```

| Option     | Short | Description        |
| ---------- | ----- | ------------------ |
| `--export` | `-e`  | Export to file     |

**Examples:**

```bash
dtiam analyze user-permissions admin@example.com
dtiam analyze user-permissions admin@example.com -o json
dtiam analyze user-permissions admin@example.com --export perms.yaml
```

### analyze group-permissions

Calculate effective permissions for a group based on its policy bindings.

```bash
dtiam analyze group-permissions GROUP [OPTIONS]
```

| Option     | Short | Description        |
| ---------- | ----- | ------------------ |
| `--export` | `-e`  | Export to file     |

**Examples:**

```bash
dtiam analyze group-permissions "DevOps Team"
dtiam analyze group-permissions DevOps -o json
```

### analyze permissions-matrix

Generate a permissions matrix showing which permissions are granted by each policy or group.

```bash
dtiam analyze permissions-matrix [OPTIONS]
```

| Option     | Short | Description                            |
| ---------- | ----- | -------------------------------------- |
| `--scope`  | `-s`  | Scope: policies or groups (default: policies) |
| `--export` | `-e`  | Export to CSV file                     |

**Examples:**

```bash
dtiam analyze permissions-matrix
dtiam analyze permissions-matrix --scope groups
dtiam analyze permissions-matrix --export matrix.csv
```

### analyze policy

Analyze a policy's permissions and bindings.

```bash
dtiam analyze policy IDENTIFIER
```

Shows what permissions a policy grants and which groups it's bound to.

**Examples:**

```bash
dtiam analyze policy "admin-policy"
dtiam analyze policy viewer -o json
```

### analyze least-privilege

Analyze policies for least-privilege compliance. Identifies policies that may grant excessive permissions.

```bash
dtiam analyze least-privilege [OPTIONS]
```

| Option     | Short | Description              |
| ---------- | ----- | ------------------------ |
| `--export` | `-e`  | Export findings to file  |

Checks for:
- Wildcard permissions (`*`)
- Resource wildcards (`:*`)
- Write/manage/delete/admin access
- Permissions without conditions

**Examples:**

```bash
dtiam analyze least-privilege
dtiam analyze least-privilege -o json
dtiam analyze least-privilege --export findings.yaml
```

### analyze effective-user

Get effective permissions for a user via the Dynatrace Resolution API.

```bash
dtiam analyze effective-user USER [OPTIONS]
```

| Option       | Short | Description                                        |
| ------------ | ----- | -------------------------------------------------- |
| `--level`    | `-l`  | Level type: account, environment, global (default: account) |
| `--level-id` |       | Level ID (uses account UUID if not specified)      |
| `--services` | `-s`  | Comma-separated service filter                     |
| `--export`   | `-e`  | Export to file                                     |

**Examples:**

```bash
dtiam analyze effective-user admin@example.com
dtiam analyze effective-user admin@example.com --level environment --level-id env123
dtiam analyze effective-user admin@example.com --services settings,entities
```

### analyze effective-group

Get effective permissions for a group via the Dynatrace Resolution API.

```bash
dtiam analyze effective-group GROUP [OPTIONS]
```

| Option       | Short | Description                                        |
| ------------ | ----- | -------------------------------------------------- |
| `--level`    | `-l`  | Level type: account, environment, global (default: account) |
| `--level-id` |       | Level ID (uses account UUID if not specified)      |
| `--services` | `-s`  | Comma-separated service filter                     |
| `--export`   | `-e`  | Export to file                                     |

**Examples:**

```bash
dtiam analyze effective-group "DevOps Team"
dtiam analyze effective-group DevOps --level environment --level-id env123
```

---

## Exit Codes

| Code | Description                                                              |
| ---- | ------------------------------------------------------------------------ |
| 0    | Success; `auth can-i` answered yes or conditional                        |
| 1    | Error, including an unknown subcommand; `diff` found drift; `auth can-i` answered no |

### config migrate-secrets

Move plaintext client secrets and environment tokens from the config file into
the OS keyring.

```bash
dtiam config migrate-secrets [--dry-run]
```

Idempotent — credentials already referencing the keyring are skipped. If the
keyring is unavailable the command fails without changing anything, and a
credential that cannot be migrated keeps its plaintext secret, so the config is
never left in a broken state.

### config keyring-status

Show whether the OS keyring is in use and where each credential's client secret
and environment token live.

```bash
dtiam config keyring-status [--output FORMAT]
```

Use this to confirm no plaintext secrets remain after `config migrate-secrets`.

---

## diff

Show what `apply` would change, without changing anything.

```bash
dtiam diff -f FILE [OPTIONS]
```

| Option        | Short | Description                                        |
| ------------- | ----- | -------------------------------------------------- |
| `--file`      | `-f`  | Resource definition file (required)                |
| `--set`       |       | Template variable as `key=value` (repeatable)      |
| `--exit-zero` |       | Exit 0 even when there are changes                 |
| `--output`    | `-o`  | Output format                                      |

Read-only. For each resource in the file it fetches the live resource and
compares field by field.

**What is and is not compared:**

- Only fields present in the file. The API returns server-managed fields a spec
  never mentions (`uuid`, `createdAt`, `owner`); reporting those would bury the
  real changes.
- Lists compare without regard to order, because the API returns members, scopes
  and zones in an order the caller does not control.
- `5` from a YAML file compares equal to `5.0` from the API — otherwise every
  numeric field would look modified on every run.

**Exit codes:** 1 when there are changes, 0 when up to date. That makes it a
drift gate in CI. `--exit-zero` always exits 0.

```bash
# What would apply change?
dtiam diff -f resources.yaml

# Drift detection in CI
dtiam diff -f desired-state.yaml --plain || echo "drift detected"
```

---

## doctor

Diagnose configuration, credentials, and API connectivity.

```bash
dtiam doctor [OPTIONS]
```

| Option      | Description                                   |
| ----------- | --------------------------------------------- |
| `--offline` | Skip the checks that make network calls       |
| `--context` | Check a specific context                      |
| `--output`  | Output format                                 |

Checks performed, cheapest first:

1. dtiam version
2. Configuration file exists and parses
3. A current context is selected
4. Account UUID is resolvable
5. Credentials are configured (OAuth2 or bearer token)
6. OAuth scope set in use, naming any default scope an override omits
7. A token can be obtained from the SSO endpoint *(network)*
8. The account API answers an authenticated request *(network)*

Later checks are **skipped** rather than failed when an earlier one makes them
meaningless, so the output distinguishes "could not test" from "failed".

Exits non-zero if any check fails, which makes it usable as a CI readiness gate.

```bash
# Full diagnosis
dtiam doctor

# Local checks only
dtiam doctor --offline

# Machine-readable, for CI
dtiam doctor --plain
```

Statuses: `ok`, `warn` (works but worth knowing — e.g. a static bearer token that
cannot refresh), `fail`, `skip`.

---

## See Also

- [Quick Start Guide](QUICK_START.md)
- [Architecture](ARCHITECTURE.md)
- [API Reference](API_REFERENCE.md)
