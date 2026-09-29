# Lappy Exact Desktop Commander Evidence — 2026-09-29

Status: LIVE READBACK / PARTIAL RUNTIME EVIDENCE

This file records observations from the live Lappy connector on 2026-09-29. It is evidence, not a claim that the final ChatGPT exact-Desktop-Commander path is already active.

## Current Lappy VeraPort session

Live `machine_info` reported:

- authenticated: true;
- data-plane verified: true;
- selected loopback path on `127.0.0.1:17444`;
- requested and granted capabilities:
  - `fs.read`
  - `fs.write`
  - `process.exec`
  - `process.control`
  - `process.inspect`
  - `process.interact`

This proves backend/session authority exists. It does not prove this already-open ChatGPT conversation exposes all of those operations in its current plugin tool catalog.

## Installed Desktop Commander tree

Observed path:

`D:\VERA\DesktopCommanderMCP-concurrency-v1`

The tree contains:

- upstream Desktop Commander source;
- built `dist/`;
- `node_modules/`;
- private `workbridge-runtime/`;
- `workbridge-desktop-commander.manifest.json`.

The Git working tree HEAD file reported exactly:

`550a0b3e31da18b7cf25e87ed840e3d953b6da42`

and origin:

`https://github.com/wonderwhy-er/DesktopCommanderMCP.git`

## Live manifest readback

Observed manifest fields:

- schema: `WORKBRIDGE_DESKTOP_COMMANDER_DUPLICATE_V1`
- upstream repository: `https://github.com/wonderwhy-er/DesktopCommanderMCP.git`
- upstream commit: `550a0b3e31da18b7cf25e87ed840e3d953b6da42`
- upstream version: `0.2.51`
- Node relative path: `workbridge-runtime\node.exe`
- Node SHA-256: `3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237`
- entrypoint relative path: `dist\index.js`
- entrypoint SHA-256: `a4145198dc75cd34e7c7452c2054b4cc0d29e199ae1459961a17e4da25a13500`
- MCP args: `dist\index.js --no-onboarding`
- unrestricted command-string shell: true
- WorkBridge process concurrency metadata: 4
- overlay label: `bounded-process-concurrency-v1`

The last two fields are packaging/coordination metadata and must not be interpreted as proof that the upstream Desktop Commander source itself was patched.

## Source equality check

An initial raw text comparison of the live Windows checkout against GitHub showed different character counts for:

- `src/server.ts`
- `src/tools/schemas.ts`

That initially raised a dirty-tree concern.

A normalized comparison then proved both files equal to the exact upstream commit after converting CRLF/CR line endings to LF:

- `src/server.ts`: normalized_equal = true; live CRLF count = 1652; upstream CRLF count = 0.
- `src/tools/schemas.ts`: normalized_equal = true; live CRLF count = 277; upstream CRLF count = 0.

Therefore the observed difference was Windows line-ending conversion, not a semantic source modification.

This correction is deliberately preserved because raw byte/text inequality must not be promoted into "dirty source" without normalization/context.

## What is not yet live-proven

The current V2 ChatGPT connector could not read:

`C:\ProgramData\VeraMesh`

because that path is outside its currently admitted filesystem roots.

Therefore this session has not authoritatively read:

- `desktop-commander-duplicate-activation.json`;
- the production `tunnel-runtime.json`;
- the managed tunnel alias/ID;
- the tunnel service status from the host filesystem/service plane.

Repository source shows how the activation should work, but repository state is not runtime proof.

## Existing personal plugin state

Existing Lappy personal plugins were observed pointing at a temporary Cloudflare-style MCP endpoint rather than evidence of the managed exact Desktop Commander tunnel.

Concrete capability-bearing endpoint paths are intentionally not recorded here.

This means those existing plugins are not accepted as proof of the final V1 route.

## Claim ceiling

Current live evidence supports:

- exact Desktop Commander source/build is present on Lappy;
- manifest identifies the intended upstream subject;
- source files checked are semantically identical to upstream after line-ending normalization;
- Lappy VeraPort backend has broad filesystem/process capabilities.

It does not yet support:

- exact Desktop Commander tunnel activation is currently installed/running;
- ChatGPT is connected to that tunnel;
- ChatGPT currently sees the exact 26-tool upstream surface;
- a ChatGPT-originated arbitrary Desktop Commander command has executed through the final route.
