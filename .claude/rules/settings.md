# settings.json Configuration Guide

---

## 1. Overview

`.claude/settings.json` is the **project configuration source of truth**. It defines:

- Project metadata and repository information
- Code quality standards and enforcement thresholds
- Language and environment configuration
- Design patterns and framework choices
- Reflection Protocol instructions

## 2. File Location [MUST]

```
.claude/settings.json       ← Project metadata and standards
.claude/settings.local.json ← Local permissions (not committed if user-specific)
```

`settings.json` is committed to the repo. `settings.local.json` is for local Claude Code permissions only.

## 3. Schema [MUST]

### Project Metadata

```json
{
  "projectName": "Human-readable project name",
  "projectType": "go",
  "version": "2.0.0",
  "description": "What this project solves"
}
```

- `projectName` — Short, human-readable name
- `projectType` — One of: `python`, `go`, `typescript`, `dynatrace-app`, `dynatrace-workflow`, `general`
- `version` — Current semantic version (must match `pkg/version/version.go`)
- `description` — One sentence describing the project's purpose

### Repository

```json
{
  "repository": {
    "owner": "github-username-or-org",
    "name": "repo-name",
    "url": "https://github.com/owner/repo-name"
  }
}
```

### Standards

```json
{
  "standards": {
    "semanticVersioning": true,
    "branchingStrategy": "feature/fix",
    "requiresDocumentation": true,
    "requiresTests": true,
    "requiresChangeLog": true
  }
}
```

All fields are boolean flags that indicate mandatory project requirements.

### Code Quality

```json
{
  "minimumCodeQuality": {
    "linting": "golangci-lint",
    "formatter": "gofmt",
    "testing": "go test",
    "coverage": 80,
    "errorHandling": "explicit"
  }
}
```

- `coverage` — Minimum test coverage percentage (signal, not hard gate)
- `errorHandling` — `"explicit"` means all errors must be checked and wrapped

### Go Environment

```json
{
  "goEnvironment": {
    "version": "1.23+",
    "modules": true,
    "buildTool": "make"
  }
}
```

### Design

```json
{
  "design": {
    "pattern": "resource-oriented",
    "cliFramework": "cobra",
    "configFramework": "viper",
    "httpClient": "resty",
    "logging": "logrus"
  }
}
```

Documents the core architectural choices. See `DECISIONS.md` for reasoning behind each.

## 4. Reflection Protocol [SHOULD]

The `customInstructions` field defines the Reflection Protocol — automatic actions performed when a task completes:

```json
{
  "customInstructions": "When a task is complete, perform the 'Reflection Protocol': 1. Update MEMORY.md with non-obvious quirks discovered. 2. Log design decisions to DECISIONS.md. 3. Update active phase file with next step. 4. Do automatically before summarizing."
}
```

### Automatic Actions

1. **Update MEMORY.md** — Non-obvious technical quirks, integration patterns, pitfalls
2. **Update DECISIONS.md** — Why a choice was made, alternatives rejected, trade-offs
3. **Update phase file** — Mark progress, identify next step, note blockers

## 5. When to Update [MUST]

Update `settings.json` when:
- Version changes (keep in sync with `pkg/version/version.go`)
- Code quality standards change
- New framework or design pattern adopted
- Repository ownership changes

Do NOT use `settings.json` for:
- Sensitive credentials (use `.env` or config contexts)
- Runtime configuration (use environment variables or `~/.config/dtiam/config`)
- Temporary debugging settings

## 6. Validation [SHOULD]

Before committing, verify JSON is valid:

```bash
python3 -m json.tool .claude/settings.json > /dev/null
```

## 7. Integration with Rules

Other rule files reference `settings.json`:

| Rule File | Uses |
|-----------|------|
| `core.md` | `standards.*` for enforcement gates |
| `deployment.md` | `version` for release tagging |
| `development.md` | `goEnvironment` and `minimumCodeQuality` |
| `testing.md` | `minimumCodeQuality.coverage` threshold |
| `CLAUDE.md` | `version` and `repository` fields |
