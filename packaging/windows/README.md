# Windows packaging

This directory contains **source-only packaging artifacts** for running a built
WorkBridgeMCP binary on Windows.

Nothing in this directory installs a service, changes the firewall, downloads a
wrapper, edits credentials, or exposes a listener by itself.

## Intended layout

A release directory can contain:

- `workbridge-mcp.exe` — the built WorkBridgeMCP binary;
- `WorkBridgeMCP.xml.template` — source template for service-wrapper configuration;
- `README.md`, `THIRD_PARTY_NOTICES.md`, and `release-manifest.json`;
- `LICENSE` only when the repository actually contains one;
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


## Qualification

The CI Windows release job builds the executable natively, launches it under a generated
least-authority configuration, completes MCP initialization, verifies the read-only
`tools/list` surface, packages the release, verifies required archive members, and uploads
the zip as a CI artifact. This qualifies the artifact build path; it does not install or
start a persistent service.


The release helper requires an exact source commit and checks the built executable with
`go version -m`. Packaging fails if the embedded `vcs.revision` differs or if Go reports
`vcs.modified=true`. The release manifest records that verified source commit.
