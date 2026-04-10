# dtiam Porting Reference for dtctl

Two design proposals — [IAM Integration](https://github.com/dynatrace-oss/dtctl/blob/docs/iam-integration-design/docs/dev/IAM_INTEGRATION_DESIGN.md) and [Account Namespace](https://github.com/dynatrace-oss/dtctl/blob/docs/account-namespace-design/docs/dev/ACCOUNT_NAMESPACE_DESIGN.md) — describe integrating dtiam's IAM functionality into dtctl as `dtctl iam` and `dtctl account` subcommand namespaces. Both designs draw from dtiam as the prototype, porting logic and design patterns rather than forking the codebase.

This directory captures what the design docs don't: the algorithms behind dtiam's analysis commands, the undocumented API behaviors discovered through 737 tests and production usage, and the internal data formats that define compatibility. These are things that would otherwise require reading dtiam's source to learn.

**This is a reference branch. It should not be merged to main.**

---

## Documents

| Document | What it covers |
|----------|---------------|
| [DESIGN_REVIEW.md](DESIGN_REVIEW.md) | Evaluation of both design proposals from dtiam's perspective. Maps dtiam features to proposed dtctl equivalents, identifies gaps (missing export system, group clone, 4 analyze subcommands, 2 bulk operations), and flags the shared PKCE scope blocker. |
| [ANALYZE_ALGORITHMS.md](ANALYZE_ALGORITHMS.md) | The 7 permission analysis algorithms: statement parsing, user/group effective permissions, policy and group matrices, least-privilege heuristics, and the Resolution API integration. **Not recommended for implementation** — these filled a niche in dtiam but the client-side algorithms are complex to maintain and produce approximations. Only the Resolution API commands (`effective-user`, `effective-group`) are worth considering. |
| [API_BEHAVIORS.md](API_BEHAVIORS.md) | Empirically discovered API behaviors not in official docs: response format inconsistencies across endpoints, null vs absent vs empty handling, pagination patterns (three different styles), error shapes, OAuth resource URN requirement, and rate limiting behavior. |
| [EXPORT_FORMAT.md](EXPORT_FORMAT.md) | YAML/JSON structure produced by each `export` subcommand. Field names, enrichment options, CSV flattening rules, and the `writeData()` format contract. Compatibility target if `dtctl iam export` is implemented. |
| [BOUNDARY_QUERY_FORMAT.md](BOUNDARY_QUERY_FORMAT.md) | How boundary query strings are constructed for app-id and schema-id boundaries. Covers the `buildBoundaryQuery()` function, quoting rules, validation against environment APIs, and the three-line management zone pattern. |
| [GROUP_CLONE.md](GROUP_CLONE.md) | The compound `group clone` operation: create group, copy members, copy bindings. Documents the best-effort/no-rollback error strategy, API call counts, and the related `group setup` command for declarative provisioning. |

## Context

The design docs describe *what* to build. These docs describe *how dtiam built it* and *what the API actually does*.
