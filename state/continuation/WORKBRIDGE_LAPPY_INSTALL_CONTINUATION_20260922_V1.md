# WORKBRIDGE LAPPY INSTALL CONTINUATION — 2026-09-22 V1

## Authority

Live user authorization:

> Authorize installing and starting WorkBridgeMCP on Lappy, preserving the existing VeraPort identity and keeping process execution disabled unless separately authorized.

This authority is limited to WorkBridgeMCP installation/start on Lappy. Do not merge unrelated PRs, alter VeraPort identity, enable process execution, widen firewall/Tailscale/public exposure, or replace VeraPort.

## Current canonical WorkBridge subject

Repository: `thebrazenbeard/WorkBridgeMCP`

Current repair PR: #5
Branch: `repair/workbridge-current-main-v1-20260922`
Exact head at checkpoint: `92c66159d9b610a94f59066ab2bb6416f4513093`
Base: `main@12bcbec3a4bda69e8b9361317feb8e9d9d47b6df`
PR state: open, review-ready, mergeable at checkpoint time.

Exact-head CI:
- workflow run: `35797931646`
- conclusion: PASS
- Windows native build/test/vet: PASS
- stdio MCP smoke: PASS
- HTTP MCP smoke under Windows PowerShell 5.1: PASS
- release packaging smoke: PASS
- Lappy installer parse under Windows PowerShell 5.1: PASS
- Windows amd64 cross-build: PASS

## Installer source binding

File:
`scripts/Install-WorkBridgeLappy.ps1`

At checkpoint head the installer defaults to:

`SourceCommit = a4bf4600c3bc75ef65ed91223c2d3c11073c9959`

That source commit is independently exact-head CI PASS:
- workflow run: `35797814620`
- commit message: `Split stdio and PowerShell 5.1 HTTP smoke gates`

The installer intentionally builds the qualified source commit rather than the continuation/checkpoint commit.

## Prior failed live install

An earlier installer invocation on Lappy built/tests successfully until the black-box smoke attempted to parse a temporary JSON config written by Windows PowerShell 5.1.

Observed failure:

`decode config: invalid character '├»' looking for beginning of value`

Root cause:
- Windows PowerShell 5.1 `Set-Content -Encoding UTF8` emitted a UTF-8 BOM.
- WorkBridge JSON decoding correctly rejected the BOM.

The failed attempt occurred before the installer copied the WorkBridge binary into the install location, registered the scheduled task, or changed VeraPort.

The repair lineage:
- smoke config now uses `[IO.File]::WriteAllText(..., UTF8Encoding(false))`;
- smoke cleanup uses PowerShell 5.1-compatible `Process.Kill()`;
- a separate HTTP smoke is exercised explicitly under `powershell.exe` 5.1;
- installer config/token/runner are BOM-free;
- installer explicitly checks the installed config for a BOM.

Do not reuse installer URLs/commits older than the current PR #5 head without fresh verification.

## Intended live WorkBridge installation

Lappy:
- binary target: `C:\Program Files\WorkBridgeMCP\workbridge-mcp.exe`
- data/config target: `C:\ProgramData\WorkBridgeMCP`
- startup mechanism: scheduled task `WorkBridgeMCP`, running as SYSTEM
- HTTP listen: `127.0.0.1:8765`
- MCP path: `/mcp`
- bearer token: stored locally in `C:\ProgramData\WorkBridgeMCP\http-token.txt`, never commit/token-content into Git or chat
- process execution: MUST remain disabled
- filesystem roots: reuse the existing VeraPort `allowed_roots`; do not broaden authority
- VeraPort identity/controller material: hash before/after and require unchanged
- firewall/Tailscale/public exposure: no changes

## VeraMesh / Lappy connection already qualified

Companion repository: `thebrazenbeard/vera-mesh`
Current PR: #25
Branch: `work/veramesh-rdc-replacement-v1-20260922`
Exact head at checkpoint: `60d233c8ccc25871b0666d00c53fa9be26c7cc74`

TheSimsVault -> VeraMesh edge -> Lappy VeraPort is live-qualified without RDC.

Observed live connection facts:
- edge target: `100.88.50.35:17444`
- NAS local edge: `127.0.0.1:17445`
- TLS: 1.3
- ALPN: `veraport/1`
- direct/edge certificate identity matched
- VeraPort application mutual authentication succeeded
- existing enrolled controller identity was reused
- Lappy workstation application signature verified
- granted capability classes: `fs.read`, `fs.write`
- `lane.list`: PASS, returned zero lanes
- RDC was not used
- no service/identity/process-policy mutation occurred

The NAS now has the required VeraPort controller private key and Lappy workstation public key under:
`/volume1/homes/psims85/.veramesh/lappy/`
Do not expose their contents.

## Immediate continuation

1. Fresh-read PR #5 and exact current head/CI before acting.
2. Fresh-read `scripts/Install-WorkBridgeLappy.ps1` from that exact head.
3. If the current head remains qualified and installer source binding remains valid, give the user the exact one-line elevated PowerShell command for the current installer.
4. User runs it on Lappy as Administrator.
5. Inspect the resulting `WORKBRIDGE_LAPPY_INSTALL_QUALIFICATION_V1` JSON.
6. If PASS, qualify:
   - task running,
   - loopback-only listener 127.0.0.1:8765,
   - authenticated health,
   - process disabled,
   - VeraPort identity/service unchanged.
7. Then connect VeraMesh to WorkBridge without exposing WorkBridge directly or widening firewall/Tailscale authority.
8. Preserve source/build/install/runtime/network/E2E as separate claims.

No merge is authorized by this checkpoint.
