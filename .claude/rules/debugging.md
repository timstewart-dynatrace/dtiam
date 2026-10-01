# Debugging & Troubleshooting

---

## 1. Common Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| "No context configured" | Missing config | Run `dtiam config set-credentials`, `set-context`, `use-context` |
| "Permission denied" | Missing OAuth scopes | Add required scopes (see below) |
| Build errors | Wrong Go version | Ensure Go 1.23+, run `go mod tidy` |
| Token expired | Using bearer token | Switch to OAuth2 for auto-refresh |
| Empty results | Wrong account UUID | Verify `DTIAM_ACCOUNT_UUID` |

## 2. Required OAuth Scopes

Defined in `auth.DefaultScopeList` (`internal/auth/oauth.go`). All of these are
requested by default; override with `DTIAM_SCOPES`.

| Scope | Required For |
|-------|-------------|
| `account-idm-read` | List/get groups, users, service users, account limits |
| `account-idm-write` | Create/delete groups, users, service users |
| `account-env-read` | List environments, reference data |
| `account-uac-read` | DPS subscriptions, forecast, cost, notifications |
| `account-audit-logs-read` | Account audit logs |
| `platform-token:tokens:manage` | Platform token list/create/delete |
| `iam-policies-management` | Policies, bindings, boundaries |
| `iam:policies:read` / `iam:policies:write` | Get / create+delete policies |
| `iam:bindings:read` / `iam:bindings:write` | Get / attach+detach bindings |
| `iam:effective-permissions:read` | `analyze effective-user` / `effective-group` |

Not requested by default (needs a separate environment token or OAuth client):

| Scope | Required For |
|-------|-------------|
| `app-engine:apps:run` | App Engine Registry API (`get apps`) |
| `settings.read` | Settings 2.0 schemas (`get schemas`) |

## 3. Debug Tools

### Verbose Mode
```bash
dtiam -v get groups    # Shows HTTP requests/responses
```

### Config Verification
```bash
dtiam config view              # Show current config
dtiam config current-context   # Show active context
```

### Check Configuration
```bash
# Verify config file
cat ~/.config/dtiam/config

# Test with explicit env vars
DTIAM_ACCOUNT_UUID=xxx DTIAM_CLIENT_SECRET=yyy dtiam get groups
```

## 4. Debug Strategy

When debugging, follow this order:

1. **Read the full error output** — root cause is often buried in the middle
2. **Check verbose mode** (`-v`) — see actual HTTP requests/responses
3. **Verify authentication** — correct account UUID, valid credentials
4. **Check API endpoint** — correct level type (account/environment/global)
5. **Test with `--plain -o json`** — see raw structured output

## 5. Two-Attempt Rule

If the same problem persists after 2 fix attempts:
1. Stop
2. Re-read the full error and relevant code
3. State the problem precisely
4. State your assumptions
5. Form a hypothesis before attempt 3
