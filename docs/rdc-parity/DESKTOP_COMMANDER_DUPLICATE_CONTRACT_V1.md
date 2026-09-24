# Desktop Commander Duplicate Contract V1

Status: `IMPLEMENTATION_TARGET / EXACT_UPSTREAM_SOURCE`

Reference implementation:

`wonderwhy-er/DesktopCommanderMCP@550a0b3e31da18b7cf25e87ed840e3d953b6da42`

Upstream package version: `0.2.51`

## Goal

The target is a working duplicate of Desktop Commander over VeraMesh/VeraRelay/WorkBridgeMCP, not a capability-reduced imitation.

WorkBridgeMCP pins, builds, packages, and qualifies the exact upstream DesktopCommanderMCP source. VeraMesh transports that exact stdio MCP server remotely. VeraRelay may provide relay/discovery/fallback functions, but must forward Desktop Commander calls/results without renaming, filtering, narrowing, or reinterpreting the upstream tool surface.

## Required workstation semantics

The duplicate must preserve the upstream tool API and behavior that materially affects workstation control, including:

- unrestricted command strings through `start_process(command=...)`, subject only to Desktop Commander's own upstream behavior/configuration;
- interactive process sessions through `read_process_output`, `interact_with_process`, `force_terminate`, and `list_sessions`;
- OS process inspection/control through `list_processes` and `kill_process`;
- file read/write, multi-read, directory creation/listing, move/rename, metadata, and `edit_block`;
- progressive search through `start_search`, `get_more_search_results`, `stop_search`, and `list_searches`;
- local recent tool-call history;
- upstream document convenience tools and configuration tools when present at the pinned source subject.

Do not replace these semantics with named executable grants, SHA-pinned command allowlists, reduced environments, working-root confinement, or a different process API. Those belong to the bounded native WorkBridge server, not duplicate mode.

## Source ownership

`upstream/DesktopCommanderMCP` is a Git submodule pinned to the exact source commit above. Upstream MIT notices must be preserved.

WorkBridge duplicate mode builds the actual upstream TypeScript source. It does not port the tool implementation into Go.

## Runtime composition

`ChatGPT -> VeraMesh Secure MCP Tunnel -> packaged node.exe -> exact DesktopCommanderMCP dist/index.js -> workstation`

VeraMesh verifies the packaged Node runtime and Desktop Commander entrypoint hashes before launch. Integrity binding is transport/package provenance; it does not narrow Desktop Commander's command surface.

## Acceptance

A source/build PASS requires:

1. exact upstream Git pin;
2. locked dependency install succeeds;
3. exact upstream build succeeds on Windows and Linux;
4. live MCP initialize/tools-list succeeds;
5. required Desktop Commander tools are present;
6. `start_process` advertises a required `command` string argument;
7. a real arbitrary command string executes through the upstream MCP server and returns a marker;
8. VeraMesh tunnel command construction binds the exact packaged Node executable, exact Desktop Commander entrypoint, and arguments;
9. VeraMesh regression tests reject entrypoint tampering and command-argument injection.

Runtime installation and live ChatGPT-through-tunnel behavior remain separate effects from source/build acceptance.
