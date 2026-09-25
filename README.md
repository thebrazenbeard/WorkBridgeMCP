> **License:** Source-visible, not open source. Original material is proprietary. Commercial use, redistribution, hosted-service use, and commercial derivative products require written permission. See [LICENSE](LICENSE) and [COMMERCIAL_LICENSE.md](COMMERCIAL_LICENSE.md). Separately identified third-party components retain their own licenses.

# WorkBridgeMCP

WorkBridgeMCP is a small local Model Context Protocol server that gives an MCP client
bounded access to a workstation without turning the workstation into an unrestricted
remote shell.

The server is stdio-first. Optional Streamable HTTP is restricted to literal loopback
addresses and requires an explicit bearer secret.

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
shell. Child processes receive a reduced environment rather than the server's full
environment.

Filesystem operations use Go 1.25.12 `os.Root` handles so operations execute relative to
already-open admitted roots rather than by validating one path string and later reopening
it globally.

## Build

Requires Go 1.25.12 or newer. The tested confinement/build floor is pinned to 1.25.12 in `go.mod` and CI.

```bash
go mod verify
go test ./...
go vet ./...
go build ./cmd/workbridge-mcp
```

On Windows:

```powershell
go mod verify
go test ./...
go vet ./...
go build -o workbridge-mcp.exe ./cmd/workbridge-mcp
```

CI also cross-builds a Windows amd64 executable from Linux.

## Configure

Copy `config.example.json` to an operator-owned location and edit it.

The default example is read-only and process-disabled.

Important rules:

- all roots are absolute;
- HTTP is literal-loopback only;
- HTTP requires `http.bearer_token_env` and a bearer secret of at least 32 bytes;
- writes exist only when `write_roots` is populated;
- process execution exists only when explicitly enabled;
- enabled executables require a stable name, absolute path, and lowercase SHA-256;
- process working directories must stay inside configured process roots;
- file, directory, process-output, runtime, argument count, and argument bytes are bounded.

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

Set `http.bearer_token_env` in the config, then supply the secret through that
environment variable:

```powershell
$env:WORKBRIDGE_HTTP_TOKEN = "replace-with-at-least-32-random-bytes"
.\workbridge-mcp.exe --transport http --config C:\Users\you\.workbridge\config.json
```

HTTP uses stateless Streamable MCP sessions, exact endpoint matching, bounded HTTP
timeouts/header size, and graceful shutdown.

The token value is read from the environment; it is not stored in repository or JSON
configuration. WorkBridge rejects HTTP startup when the token configuration is absent,
empty, too short, or contains whitespace.

## Security model

Configuration is authority.

A client gets only the capabilities represented by the exact local configuration loaded
at process start. A tool is not registered when its capability is disabled.

Read `docs/SECURITY.md` before enabling writes or processes.

This source does **not** claim deployment, installation, arbitrary local-adversary
resistance, or workstation effect merely because the repository builds.

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
