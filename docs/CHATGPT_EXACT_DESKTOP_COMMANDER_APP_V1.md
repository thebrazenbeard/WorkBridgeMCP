# WorkBridge ChatGPT Exact Desktop Commander App V1

Status: DESIGN + SOURCE CONTRACT / SINGLE-DEVICE LAPPY EXPERIMENT

Date: 2026-09-29

This document captures the current WorkBridge/ChatGPT idea after adversarial review. It is deliberately narrower than an RDC replacement product.

## Thesis

The first complete path to prove is:

```text
ChatGPT
  -> WorkBridge Desktop Commander app
  -> VeraMesh Secure MCP Tunnel
  -> packaged Node runtime
  -> exact DesktopCommanderMCP 0.2.51
  -> Lappy
```

The app is a transport binding, not a semantic adapter.

DesktopCommanderMCP owns the workstation tool API and behavior. WorkBridgeMCP owns the exact upstream source pin, packaging, provenance, install manifest, qualification, and acceptance tests. VeraMesh owns authenticated remote transport. ChatGPT consumes the remote MCP endpoint.

V1 intentionally does not implement multi-device discovery, an RDC-style broker, DS216 routing, or a translated WorkBridge/VeraPort tool API.

## Exact upstream subject

- repository: `wonderwhy-er/DesktopCommanderMCP`
- commit: `550a0b3e31da18b7cf25e87ed840e3d953b6da42`
- npm package: `@wonderwhy-er/desktop-commander`
- package version: `0.2.51`
- upstream license: MIT

The upstream source is already pinned as `upstream/DesktopCommanderMCP`.

## Exact public tool surface at the pinned subject

The V1 acceptance contract requires these 26 upstream tools:

1. `get_config`
2. `set_config_value`
3. `read_file`
4. `read_multiple_files`
5. `write_file`
6. `write_pdf`
7. `create_directory`
8. `list_directory`
9. `move_file`
10. `start_search`
11. `get_more_search_results`
12. `stop_search`
13. `list_searches`
14. `get_file_info`
15. `edit_block`
16. `start_process`
17. `read_process_output`
18. `interact_with_process`
19. `force_terminate`
20. `list_sessions`
21. `list_processes`
22. `kill_process`
23. `get_usage_stats`
24. `get_recent_tool_calls`
25. `give_feedback_to_desktop_commander`
26. `get_prompts`

The source also defines `track_ui_event` in its internal schema map, but that name is not part of the normal public tool list at the pinned `server.ts` subject and is therefore not a V1 public-tool requirement.

The tool list is not hand-maintained as mere prose. `plugin/workbridge-desktop-commander/tool-contract.json` plus `tools/test_workbridge_desktop_commander_plugin.py` bind the contract back to the pinned upstream source.

## Process semantics

The defining process capability is upstream Desktop Commander's:

```text
start_process(command: string, timeout_ms: number, ...)
```

V1 must not replace this with:

- argv-only execution;
- named executable grants;
- SHA-pinned command allowlists;
- VeraPort `process.exec`;
- bounded native WorkBridge `process_run`;
- a different process/session API.

Those mechanisms remain valid for the bounded native WorkBridge mode. They are not the Desktop Commander duplicate.

## Why this path

WorkBridge originally explored a constrained reimplementation of Remote Desktop Commander-class behavior. That direction is retained as history in `docs/rdc-parity/RDC_PARITY_CONTRACT_V1.md`, but it is superseded for Lappy duplicate mode by `DESKTOP_COMMANDER_DUPLICATE_CONTRACT_V1.md`.

The exact duplicate path avoids a large semantic translation layer. A transparent transport is easier to falsify:

- if Desktop Commander lists a tool locally, the remote path should list the same tool;
- if Desktop Commander accepts a command string locally, the remote path should preserve it;
- if a response, notification, error, or resource changes across the transport, that is a transport defect rather than an accepted reinterpretation.

## WorkBridge's role

WorkBridgeMCP owns:

- the admitted upstream repository and exact commit;
- MIT provenance and license preservation;
- reproducible build/install;
- packaged Node + built Desktop Commander runtime;
- runtime hash manifest;
- source/build tests;
- ChatGPT app source;
- exact upstream tool contract;
- acceptance evidence.

WorkBridgeMCP does not need to own the remote tunnel protocol.

## VeraMesh's role

VeraMesh owns:

- authenticated remote transport;
- Secure MCP Tunnel runtime;
- service persistence;
- runtime health/readiness;
- hash verification before launch;
- optional relay/discovery/fallback mechanics;
- byte transport without MCP semantic translation on this path.

Canonical VeraMesh main already contains the merged exact duplicate tunnel path at:

`thebrazenbeard/vera-mesh@98b74ff77981a5478e20a748bbb94565ad9140c8`

The corresponding runtime documentation states:

```text
ChatGPT -> VeraMesh Secure MCP Tunnel -> private node.exe -> exact DesktopCommanderMCP dist/index.js -> workstation
```

## ChatGPT app role

The ChatGPT app source lives under:

`plugin/workbridge-desktop-commander/`

It should contain no concrete capability-bearing tunnel URL. The repository stores a sentinel template only. A private render step supplies the live HTTPS MCP endpoint and produces `mcp.json` / `.mcp.json`.

The app should have no dependency on the stale Lappy V2 Apps SDK action catalog and no dependence on Codex. Codex compatibility may exist as an optional package surface, but the normal route is ChatGPT direct MCP.

## V1 acceptance ladder

A PASS is subject-specific.

### Source PASS

Requires:

- exact upstream submodule pin;
- app manifest/source present;
- exact tool contract bound to upstream source;
- process command-string contract proven in source;
- secret-free repository;
- renderer tests pass.

### Build PASS

Requires:

- locked upstream dependency install;
- exact upstream build;
- upstream MCP initialize/tools/list;
- required tools visible;
- arbitrary harmless command marker executes through upstream `start_process`;
- packaged runtime hashes emitted.

### Install PASS

Requires:

- exact packaged duplicate installed on Lappy;
- manifest identifies upstream commit/version;
- Node and `dist/index.js` hashes match the installed files.

### Tunnel PASS

Requires:

- VeraMesh tunnel configured to launch the manifest-bound Node + entrypoint;
- tunnel service running and healthy;
- tunnel transport does not rename/filter/rewrite MCP;
- activation receipt or equivalent authoritative readback exists.

### ChatGPT PASS

Requires:

- app connected to the managed tunnel endpoint;
- ChatGPT action/tool scan contains every required public tool;
- `start_process` schema still requires `command: string`;
- a harmless marker command is issued from ChatGPT;
- its output is read back through Desktop Commander's own process tools.

Only this final state proves the intended user experience.

## Non-goals for V1

V1 does not include:

- `list_devices`;
- account/device broker state;
- RDC hosted-service cloning;
- DS216 in the same tool endpoint;
- automatic failover between Lappy and DS216;
- translated VeraPort tool names;
- replacing SSH;
- merging or retiring any existing access path merely because V1 works.

## Next architectural question

Only after the single-device path passes end-to-end should the project decide whether a multi-device layer belongs:

- above Desktop Commander as a thin broker;
- inside VeraMesh routing;
- as a WorkBridge-branded broker;
- or not at all.

That decision must be evidence-driven from the successful V1 path rather than inferred from product names.
