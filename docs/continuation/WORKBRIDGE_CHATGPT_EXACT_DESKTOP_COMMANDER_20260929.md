# WorkBridge ChatGPT Exact Desktop Commander — Continuation 2026-09-29

## Current objective

Prove the smallest complete path:

`ChatGPT -> WorkBridge Desktop Commander app -> VeraMesh Secure MCP Tunnel -> exact DesktopCommanderMCP -> Lappy`

Do not expand into a multi-device broker until this path passes.

## Canonical source subjects

WorkBridge canonical main at branch creation:

`8707a2e1eaf7de5ce2316567b5e6f1e805c0537b`

Active draft PR:

- PR #16: `Add exact Desktop Commander ChatGPT app contract`
- branch: `work/chatgpt-exact-desktop-commander-app-v1-20260929`

VeraMesh canonical main with merged exact Desktop Commander transport:

`98b74ff77981a5478e20a748bbb94565ad9140c8`

Pinned Desktop Commander upstream:

`wonderwhy-er/DesktopCommanderMCP@550a0b3e31da18b7cf25e87ed840e3d953b6da42`

## Implemented on PR #16

- direct ChatGPT plugin source under `plugin/workbridge-desktop-commander/`;
- secret-free MCP URL template;
- private renderer;
- source-derived exact public-tool contract;
- native `start_process(command: string)` requirement;
- CI integration.

## Live Lappy evidence

See `docs/rdc-parity/LAPPY_EXACT_DESKTOP_COMMANDER_LIVE_EVIDENCE_20260929.md`.

Important live facts:

- exact upstream source subject is present;
- installed manifest exists and records runtime hashes;
- checked upstream files match after newline normalization;
- current VeraPort backend grants filesystem and process capabilities;
- this session cannot read `C:\ProgramData\VeraMesh` through the current V2 root policy;
- final tunnel activation receipt and final ChatGPT round-trip therefore remain unproven.

## Next gates

1. Get PR #16 exact-head CI fully green.
2. Verify the managed VeraMesh tunnel is bound to the manifest-qualified Desktop Commander runtime.
3. Render/install the WorkBridge Desktop Commander plugin with the managed tunnel endpoint.
4. Refresh/scan ChatGPT actions.
5. Require exact 26-tool set.
6. Verify `start_process` schema preserves `command: string`.
7. Execute a harmless marker command from ChatGPT.
8. Read marker output through Desktop Commander's own process tools.
9. Record a runtime acceptance receipt.
10. Only then evaluate multi-device/RDC-like routing.

## Explicitly deferred

- DS216 Desktop Commander parity;
- RDC-style `list_devices`;
- universal WorkBridge router;
- replacing SSH;
- retiring existing connectors;
- merge of PR #16 without live user authority.
