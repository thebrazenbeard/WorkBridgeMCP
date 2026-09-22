# Windows packaging

This directory contains **source-only packaging artifacts** for running a built
WorkBridgeMCP binary on Windows.

Nothing in this directory installs a service, changes the firewall, downloads a
wrapper, edits credentials, or exposes a listener by itself.

## Intended layout

A release directory can contain:

- `workbridge-mcp.exe` — the built WorkBridgeMCP binary;
- `WorkBridgeMCP.xml` — service-wrapper configuration rendered from the template;
- operator-owned configuration outside the release archive.

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
