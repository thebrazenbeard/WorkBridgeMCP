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

## Existing Lappy installer

`scripts/Install-WorkBridgeLappy.ps1` is a separate elevated installation
packet. Running it builds an exact pinned source commit, registers a persistent
SYSTEM scheduled task, and starts authenticated loopback HTTP. It configures
read and write roots from the existing VeraPort `allowed_roots`; process
execution stays disabled. Merely building or packaging this repository does
not run that packet.

The default source pin is the code-bearing repair commit
`613c3df0e7bd1d43b123d249e3aaca4366852546`. Check the source and local
policy before a later operator invokes the installer.

The installer waits for any prior managed listener to release its port before
replacing the binary. After startup it verifies that the listener belongs to
the installed executable, the scheduled task is running, and the existing
VeraPort configuration and identity files did not change. A foreign listener
or ambiguous prior task fails installation rather than being reported as a
healthy WorkBridge runtime.

## Security boundary

For workstation use, prefer:
1. stdio when the MCP client launches WorkBridge directly;
2. loopback-only HTTP when a persistent local service is required.

Do not expose the HTTP transport to a non-loopback interface merely because the
service wrapper can keep the process alive.
