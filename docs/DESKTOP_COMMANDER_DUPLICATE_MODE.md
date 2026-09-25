# Desktop Commander duplicate mode

Status: SOURCE-BOUND DUPLICATE PATH / NOT INSTALLED BY REPOSITORY PRESENCE

WorkBridgeMCP now carries the exact upstream DesktopCommanderMCP source as a Git submodule:

- upstream: `wonderwhy-er/DesktopCommanderMCP`
- exact source commit: `550a0b3e31da18b7cf25e87ed840e3d953b6da42`
- upstream package version at that commit: `0.2.51`
- upstream license: MIT

This mode is intentionally different from the bounded native Go WorkBridge server. The duplicate mode builds and runs the actual Desktop Commander source and therefore preserves its real tool behavior, including the unrestricted command-string process surface exposed by `start_process(command=...)`.

## Build/install on Windows

Run:

```powershell
.\scripts\Install-DesktopCommanderDuplicate.ps1 -RunTests
```

The installer clones the exact upstream commit into a staging directory, verifies the commit and package identity, runs `npm ci --ignore-scripts`, builds the upstream source, optionally runs its test suite, records hashes for the Node executable and built MCP entrypoint, and atomically replaces the prior duplicate install.

Default install root:

`C:\ProgramData\WorkBridgeMCP\DesktopCommanderMCP`

The emitted `workbridge-desktop-commander.manifest.json` is the handoff contract for VeraMesh. It identifies the exact Node executable, its SHA-256, the built `dist/index.js` entrypoint and its SHA-256, and the arguments required to run the Desktop Commander stdio MCP server.

## Runtime composition

Target path:

`ChatGPT -> VeraMesh Secure MCP Tunnel -> node.exe -> exact DesktopCommanderMCP dist/index.js -> workstation`

VeraMesh provides the authenticated remote transport. DesktopCommanderMCP provides the workstation MCP behavior. WorkBridgeMCP owns the exact-source pin, build/install/verification packet, and interoperability tests.

VeraRelay remains available as a VeraMesh relay/currentness component, but it must not reinterpret or narrow Desktop Commander's tool semantics on this duplicate path.

## Non-effects

Repository source, the Git submodule pin, and installer source do not prove installation on Lappy, tunnel activation, or live ChatGPT tool calls. Those are separate runtime effects.
