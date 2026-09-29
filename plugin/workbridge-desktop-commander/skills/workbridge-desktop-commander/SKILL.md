---
name: workbridge-desktop-commander
description: Use when operating Lappy through the exact DesktopCommanderMCP duplicate transported by VeraMesh. Preserve upstream Desktop Commander tool names and semantics; do not translate calls into bounded WorkBridge or VeraPort APIs.
---

# WorkBridge Desktop Commander

This plugin is the single-device Lappy experiment for the exact Desktop Commander duplicate.

## Authority chain

- DesktopCommanderMCP owns tool semantics.
- WorkBridgeMCP owns the exact upstream pin, packaging, install manifest, and qualification.
- VeraMesh owns authenticated remote transport.
- This plugin exposes the remote MCP endpoint to ChatGPT without renaming or reinterpreting upstream tools.

Exact upstream subject: `wonderwhy-er/DesktopCommanderMCP@550a0b3e31da18b7cf25e87ed840e3d953b6da42`, package `0.2.51`.

## Expected native tools

The live tool list must include the names in `tool-contract.json`, including `start_process`, `read_process_output`, `interact_with_process`, `force_terminate`, filesystem/edit/search tools, OS process tools, configuration, usage, recent history, feedback, and prompts.

`start_process` must retain upstream's required `command` string argument. Do not replace it with argv-only process execution, named executable grants, or VeraPort process calls.

## Runtime verification

Do not infer runtime installation from repository state. Before claiming a functioning connection, verify the remote endpoint initializes, exposes the exact required tool names, and successfully executes a harmless marker through `start_process` with output readback.

## Scope

This V1 plugin represents one machine: Lappy. It deliberately does not implement RDC-style device discovery or route to DS216. Multi-device routing is a later, separate contract after this exact path passes end-to-end.

## Secrets

Never expose or commit the Secure MCP Tunnel capability URL, tunnel credentials, runtime keys, or private keys.
