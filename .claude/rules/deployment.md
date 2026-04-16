# Deployment & Release Checklist

---

## 1. Pre-Deployment [MUST]

Before any release, verify ALL of the following:

- [ ] All tests passing: `make test`
- [ ] Linter clean: `make lint`
- [ ] Code formatted: `make fmt`
- [ ] No uncommitted changes: `git status`
- [ ] No hardcoded credentials or secrets
- [ ] Documentation complete and reviewed
- [ ] CHANGELOG.md updated with version and date
- [ ] Version incremented in `pkg/version/version.go`
- [ ] Version in `pkg/version/version.go` matches CHANGELOG heading
- [ ] Command help text accurate: `dtiam --help`, `dtiam <command> --help`

## 2. Release Process [MUST]

### Tag the Release

```bash
# Ensure you're on main with all changes merged
git checkout main
git pull origin main

# Create annotated tag (never lightweight)
git tag -a vX.Y.Z -m "Release vX.Y.Z — brief description"
git push origin vX.Y.Z
```

### Build with GoReleaser

```bash
# Dry run first
goreleaser release --snapshot --clean

# Production release (triggered by tag push via CI, or manual)
goreleaser release --clean
```

GoReleaser configuration: `.goreleaser.yaml`
- Builds for: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
- Creates: tar.gz archives, checksums, GitHub release, Homebrew tap, deb/rpm packages

### GitHub Release

```bash
# Create release from tag with CHANGELOG excerpt
gh release create vX.Y.Z --title "vX.Y.Z" \
  --notes-file <(sed -n '/## \[X.Y.Z\]/,/## \[/p' CHANGELOG.md | head -n -1)
```

## 3. Post-Release Verification [SHOULD]

- [ ] GitHub release page has correct assets
- [ ] Binary downloads work: `curl -L <release-url> | tar xz`
- [ ] Version output correct: `dtiam version`
- [ ] Homebrew install works (if tap configured): `brew install dtiam`
- [ ] Key commands functional: `dtiam get groups --help`

## 4. Rollback Procedure [MUST]

If a release has critical issues:

```bash
# Option 1: Delete the tag and release (if caught immediately)
gh release delete vX.Y.Z --yes
git push origin --delete vX.Y.Z
git tag -d vX.Y.Z

# Option 2: Patch release (preferred for published releases)
git checkout -b fix/critical-issue
# ... fix the issue ...
# Bump patch version, update CHANGELOG
git checkout main && git merge fix/critical-issue --no-ff
git tag -a vX.Y.Z+1 -m "Release vX.Y.Z+1 — fix critical issue"
git push origin vX.Y.Z+1
```

Always prefer a patch release over deleting a published release.

## 5. Version Increment Guide [MUST]

| Change Type | Increment | Example |
|-------------|-----------|---------|
| Breaking CLI changes, removed commands/options | MAJOR | 2.0.0 -> 3.0.0 |
| New commands, new options, new resource handlers | MINOR | 2.0.0 -> 2.1.0 |
| Bug fixes, documentation, performance | PATCH | 2.0.0 -> 2.0.1 |

Version is defined in: `pkg/version/version.go`

## 6. Release Notes Template [SHOULD]

```markdown
# Release vX.Y.Z

**Release Date:** YYYY-MM-DD

## What's New

[2-3 line summary of major features/fixes]

## Upgrade Guide

[Any breaking changes or migration steps — omit if none]

## Changelog

[Link to or paste CHANGELOG section for this version]
```
