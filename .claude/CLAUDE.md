# dtiam — Dynatrace IAM CLI

**ALWAYS** ask clarifying questions and **ALWAYS** provide a plan **BEFORE** making changes to ensure the end result matches intent.

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**.

## Project Summary

**dtiam** is a kubectl-inspired CLI for managing Dynatrace Identity and Access Management (IAM) resources. It provides a consistent interface for managing groups, users, policies, bindings, boundaries, environments, and service users.

**Language:** Go 1.23+ | **Current Version:** 2.0.0

## Essential Commands

```bash
make build           # Build binary to bin/dtiam
make install         # Install to $GOPATH/bin
make test            # Run all tests
make lint            # Run golangci-lint
make fmt             # Format code
dtiam -v get groups  # Run with verbose output
```

## Architecture

See [architecture.md](architecture.md) for project structure, components, API endpoints, and data flow.

## Current Phase

Before starting work, check `.claude/phases/` for the active phase.
- Track decisions in `DECISIONS.md` as you go
- When done: rename `PHASE-XX-active.md` to `PHASE-XX-done.md`

## Skills (Domain Knowledge)
@/Users/Shared/GitHub/PROJECTS/VisualCode-AI-Template/SKILLS/dynatrace-iam/SKILL.md
@/Users/Shared/GitHub/PROJECTS/VisualCode-AI-Template/SKILLS/dynatrace-account-management/SKILL.md
@/Users/Shared/GitHub/PROJECTS/VisualCode-AI-Template/SKILLS/dynatrace-apis/SKILL.md

## Rules

### Always Active
@.claude/rules/core.md
@.claude/rules/command-standards.md
@.claude/rules/go.md
@.claude/rules/development.md
@.claude/rules/testing.md
@.claude/rules/deployment.md
@.claude/rules/existing-code.md
@.claude/rules/settings.md

### Debugging & Troubleshooting
@.claude/rules/debugging.md

## Decision Log

See `.claude/DECISIONS.md` for architectural and technical decisions.
