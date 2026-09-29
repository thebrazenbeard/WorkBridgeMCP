# WorkBridge Desktop Commander ChatGPT app V1

This directory defines the smallest end-to-end experiment agreed after hostile review:

`ChatGPT -> VeraMesh Secure MCP Tunnel -> exact DesktopCommanderMCP -> Lappy`

WorkBridge does not translate Desktop Commander tools in this path. It owns provenance and qualification of the pinned upstream runtime.

## Source binding

- upstream: `wonderwhy-er/DesktopCommanderMCP`
- commit: `550a0b3e31da18b7cf25e87ed840e3d953b6da42`
- npm package: `@wonderwhy-er/desktop-commander`
- package version: `0.2.51`
- upstream license: MIT

The exact runtime tool contract is recorded in `tool-contract.json` and checked against the pinned upstream `src/server.ts` in CI.

## Rendering

The concrete VeraMesh Secure MCP Tunnel URL is capability-bearing runtime state and is not committed. Render a private install directory with:

```text
python tools/render_workbridge_desktop_commander_plugin.py \
  --source plugin/workbridge-desktop-commander \
  --output <private-output-directory> \
  --mcp-url <secure-mcp-tunnel-url>
```

The renderer creates both portable `mcp.json` and compatibility `.mcp.json`. It refuses to overwrite an existing output directory.

## Acceptance

Source/build acceptance is not runtime acceptance. The first live acceptance requires:

1. exact duplicate runtime installed and hash-verified on Lappy;
2. VeraMesh tunnel configured to launch that runtime;
3. ChatGPT plugin connected to that tunnel;
4. live tool scan includes every required tool;
5. `start_process` still requires a string `command`;
6. a harmless arbitrary command marker executes and is read back through Desktop Commander's own process tools.

No multi-device abstraction belongs in V1.
