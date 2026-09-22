# Windows packaging

This directory contains **source-only packaging artifacts** for running a built
WorkBridgeMCP binary on Windows.

Nothing in this directory installs a service, changes the firewall, downloads a
wrapper, edits credentials, or exposes a listener by itself.

## Intended layout

A release directory can contain:

- `workbridge-mcp.exe` — the built WorkBridgeMCP binary;
- `README.md` and `THIRD_PARTY_NOTICES.md`;
- `LICENSE` only when the repository actually contains one;
- `WorkBridgeMCP.xml.template` — service-wrapper configuration source;
- operator-owned configuration outside the release archive.

The repository currently has no project `LICENSE` file. The release script therefore
does not invent one or fail solely because it is absent.

## Service wrapper model

`WorkBridgeMCP.xml.template` targets a WinSW-compatible wrapper because Windows
Service Control Manager cannot directly supervise an arbitrary console/stdio
program as a service.

The template deliberately does **not**:
- download WinSW;
- assume where WinSW is installed;
- inject credentials;
- enable public network access;
- choose an HTTP bind address.

A release/install workflow must supply those choices explicitly.

## Security boundary

For workstation use, prefer:
1. stdio when the MCP client launches WorkBridge directly;
2. loopback-only HTTP when a persistent local service is required.

Do not expose the HTTP transport to a non-loopback interface merely because the
service wrapper can keep the process alive.
