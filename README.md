# WorkBridgeMCP

WorkBridgeMCP is a small local Model Context Protocol server that gives an MCP client
bounded access to a workstation without turning the workstation into an unrestricted
remote shell.

The server is stdio-first. Optional Streamable HTTP is restricted by configuration to
literal loopback addresses.

## Current v0.1 source capabilities

Always available when at least one read root is configured:

- `workbridge_health`
- `workspace_list`
- `workspace_stat`
- `workspace_read_text`

Registered only when `write_roots` is non-empty:

- `workspace_write_text`
- `workspace_mkdir`

Registered only when `process.enabled=true` and every executable grant passes startup
identity verification:

- `process_run`

Process tools accept a configured **grant name**, not an executable path from the model.
Every enabled executable is pinned by SHA-256 and rechecked immediately before and after
execution. Arguments are passed directly to the executable; WorkBridge does not insert a
shell.

## Build

Requires Go 1.25+.

```bash
go test ./...
go build ./cmd/workbridge-mcp
```

On Windows:

```powershell
go test ./...
go build -o workbridge-mcp.exe ./cmd/workbridge-mcp
```

## Configure

Copy `config.example.json` to an operator-owned location and edit it.

The default example is read-only and process-disabled.

Important rules:

- all roots are absolute;
- HTTP is literal-loopback only;
- writes exist only when `write_roots` is populated;
- process execution exists only when explicitly enabled;
- enabled executables require a stable name, absolute path, and lowercase SHA-256;
- process working directories must stay inside configured process roots;
- file, directory, process-output, runtime, and argument counts are bounded.

Generate a Windows executable hash with:

```powershell
(Get-FileHash -Algorithm SHA256 C:\path\to\tool.exe).Hash.ToLower()
```

## Run with stdio

```powershell
.\workbridge-mcp.exe --config C:\Users\you\.workbridge\config.json
```

or set `WORKBRIDGE_CONFIG` and omit `--config`.

Example MCP client entry:

```json
{
  "mcpServers": {
    "workbridge": {
      "command": "C:\\Tools\\WorkBridgeMCP\\workbridge-mcp.exe",
      "args": ["--config", "C:\\Users\\you\\.workbridge\\config.json"]
    }
  }
}
```

## Run with loopback HTTP

```powershell
$env:WORKBRIDGE_HTTP_TOKEN = "use-a-secret-from-your-secret-store"
.\workbridge-mcp.exe --transport http --config C:\Users\you\.workbridge\config.json
```

Set `http.bearer_token_env` to `WORKBRIDGE_HTTP_TOKEN` to require the bearer token.
The token value is read from the environment; it is not stored in the repository or JSON
configuration.

WorkBridge rejects non-loopback HTTP listen addresses.

## Security model

Configuration is authority.

A client gets only the capabilities represented by the exact local configuration loaded
at process start. A tool is not registered when its capability is disabled.

Read `docs/SECURITY.md` before enabling writes or processes.

This source does **not** claim deployment, installation, hostile-local-filesystem race
qualification, or workstation effect merely because the repository builds.

## Packaging and evaluation

Windows packaging source lives under `packaging/windows/`.
Release/test helpers live under `scripts/`.
Black-box cases live under `evals/blackbox/`.

Those artifacts are source-only. They do not install a service, alter a firewall, change
credentials, or expose a listener by themselves.

## Source provenance

See `docs/SOURCE_PROVENANCE.md`.

WorkBridge was informed by external MCP and Synology projects, including Filamind, but
this implementation is original WorkBridge source rather than copied Filamind code.
