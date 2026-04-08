# Core Workflow Rules — MANDATORY

---

## 1. Branching [MUST]

- **NEVER commit features directly to main**
- ALL new features, enhancements, and non-trivial changes MUST use a feature branch
- Branch naming: `feature/descriptive-name` or `fix/descriptive-name`
- Only documentation fixes and critical hotfixes may go directly to main (with approval)

**Feature Branch Workflow:**
```bash
git checkout main && git pull
git checkout -b feature/my-feature
# ... develop and commit ...
git push -u origin feature/my-feature
```

---

## 2. Documentation [MUST]

**ALL features MUST be documented BEFORE merging to main.**

Documentation checklist (ALL must be completed):
- [ ] `CLAUDE.md` / `.claude/architecture.md` — Update if structure changes
- [ ] `docs/COMMANDS.md` — Full command reference with examples
- [ ] `README.md` — Update quick start or features section
- [ ] `docs/ARCHITECTURE.md` — Update if architecture changes
- [ ] `examples/` — Add sample files if applicable
- [ ] Code comments for new functions/types

---

## 3. Merge Process [MUST]

```bash
# Before merging: verify ALL documentation is complete
git checkout main
git merge feature/my-feature --no-ff
```

**Verification Before Merge:**
- Run tests: `make test`
- Run linter: `make lint`
- Verify command help: `dtiam <new-command> --help`
- Check all documentation files are updated

---

## 4. Version Management [MUST]

**ALL merges to main that add features or fixes MUST increment the version number.**

Current version: **1.5.0** (defined in `pkg/version/version.go`)

**Semantic Versioning:**
- **MAJOR** (X.0.0) — Breaking CLI changes, removed commands/options
- **MINOR** (1.X.0) — New commands, new options, new resource handlers
- **PATCH** (1.0.X) — Bug fixes, documentation, performance improvements

**Version Bump Checklist:**
- [ ] Version incremented in `pkg/version/version.go`
- [ ] CHANGELOG.md updated with changes
- [ ] Correct increment type (MAJOR/MINOR/PATCH)
- [ ] Version bump committed in feature branch before merge

---

## 5. CHANGELOG Management [MUST]

Follow [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

**Update in your feature branch, BEFORE merging:**
1. New features → `## [Unreleased]` → `### Added`
2. Changes → `### Changed`
3. Bug fixes → `### Fixed`

**CHANGELOG Checklist:**
- [ ] Changes in appropriate section (Added/Changed/Fixed/etc)
- [ ] [Unreleased] section moved to version section on release
- [ ] Version number matches `pkg/version/version.go`
- [ ] Date added to version heading (YYYY-MM-DD)

---

## 6. Pre-Push Checklist [MUST]

**BEFORE EVERY `git push` to main:**

```
□ 1. VERSION UPDATE — Increment in pkg/version/version.go
□ 2. CHANGELOG UPDATE — Add changes under [Unreleased]
□ 3. DOCUMENTATION UPDATE — docs/COMMANDS.md, README.md, etc.
□ 4. VALIDATION — make build && make test
□ 5. COMMIT MESSAGE — feat:/fix:/docs:/chore: + Co-Authored-By
```

**FAILURE TO FOLLOW THIS CHECKLIST WILL RESULT IN INCOMPLETE RELEASES.**

---

## 7. GitHub Releases [SHOULD]

```bash
git tag -a v1.1.0 -m "Release version 1.1.0"
git push origin v1.1.0
gh release create v1.1.0 --title "v1.1.0" \
  --notes-file <(sed -n '/## \[1.1.0\]/,/## \[1.0.0\]/p' CHANGELOG.md | head -n -1)
```

---

## 8. Phase Management [MUST]

Phases tracked in `.claude/phases/`:
- One active phase at a time
- Re-read active phase file when resuming work
- Never start next phase without user approval
- Rename `PHASE-XX-active.md` → `PHASE-XX-done.md` when complete
