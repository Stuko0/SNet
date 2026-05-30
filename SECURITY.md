# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.x     | ✅ Active development |

## Reporting a Vulnerability

SNet interacts with NetworkManager via `nmcli` and does **not** make network connections on its own. However, if you discover a security vulnerability:

1. **Do not** open a public issue.
2. Email the maintainer directly or DM on GitHub.
3. Include a detailed description and steps to reproduce.

You can expect:
- **Acknowledgment** within 48 hours.
- **Update** on progress within 5 business days.
- **Credit** in release notes if a fix is published.

## Scope

This policy covers:
- The `snet` binary and its source code.
- Build and release pipelines.

Out of scope:
- NetworkManager itself (report to the NetworkManager team).
- The terminal emulator or compositor you run SNet under.
