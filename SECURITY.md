# Security Policy

> **DISCLAIMER:** dtiam is an independent, community-developed tool. It is **not
> produced, endorsed, or supported by Dynatrace**. Report Dynatrace *platform*
> vulnerabilities to Dynatrace, not here.

## Reporting a vulnerability

Report security issues privately via
[GitHub Security Advisories](https://github.com/jtimothystewart/dtiam/security/advisories/new).

Please do **not** open a public issue for a security problem.

Include where practical: affected version (`dtiam version`), reproduction steps,
and the impact you believe it has. Please redact account UUIDs, client secrets,
tokens, and user email addresses from anything you attach.

You can expect an acknowledgement within 7 days and an assessment within 30.
Fixes ship as a patch release with the advisory published on merge.

## Supported versions

Fixes land on the latest minor release. There are no long-term support branches.

## How dtiam handles credentials

Understanding this matters for assessing your own exposure.

**Credentials are stored in plaintext.** `~/.config/dtiam/config` (or the
platform equivalent — `dtiam config path` prints it) holds `client-secret` in
plain YAML. dtiam does **not** currently use an OS keyring. Treat the config file
as a secret: keep it at `0600`, exclude it from backups and dotfile repos, and
prefer environment variables in CI.

**Credentials dtiam never writes to disk.** Values supplied through
`DTIAM_CLIENT_SECRET`, `DTIAM_BEARER_TOKEN`, or `DTIAM_ACCOUNT_UUID` are read at
runtime and not persisted.

**Access tokens are memory-only.** OAuth2 access tokens are held in memory for
their lifetime and never written to disk.

**Secrets in output.** `dtiam config view` masks secrets. `dtiam doctor` prints
the OAuth *client ID* but never the secret. A platform token's value is returned
only once, at creation, and cannot be retrieved afterwards — dtiam prints it to
stdout, so redirect carefully.

**Verbose mode prints requests and responses.** `-v` dumps HTTP traffic including
`Authorization` headers. Do not paste `-v` output into issues without redacting.

## Scope of this policy

In scope: credential handling, token leakage, output that discloses secrets,
dependency vulnerabilities, and anything causing dtiam to act on the wrong
account or escalate privileges unintentionally.

Out of scope: vulnerabilities in the Dynatrace platform or its APIs (report to
Dynatrace), and the plaintext credential storage described above, which is a
known limitation documented here rather than an undisclosed flaw.
