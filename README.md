# WorkBridgeMCP

WorkBridgeMCP is a least-authority MCP bridge for a workstation. It exposes only the local filesystem and process capabilities explicitly granted in its configuration.

Current V1 core:
- official Go MCP SDK;
- stdio transport for local MCP clients;
- stateless Streamable HTTP on a literal loopback address only;
- traversal-resistant configured read/write roots backed by Go `os.Root`;
- bounded text/binary reads and directory listing;
- explicitly gated write and directory-creation operations;
- process execution disabled by default;
- when enabled, process execution requires an exact absolute executable allow-list, allowed working roots, runtime ceiling, and output ceiling;
- Go 1.25.12+ runtime floor for the patched `os.Root` security baseline;
- Linux and Windows CI plus Windows amd64 cross-build.

The HTTP transport is intentionally loopback-only. Public authentication, reverse proxying, VPN exposure, installation, and service management are separate deployment concerns and are not performed by this repository's core runtime.

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/workbridgemcp
```

Validate a configuration without starting a transport:

```sh
workbridgemcp -config workbridge.local.json -check-config
```

Run as a local stdio MCP server:

```sh
workbridgemcp -config workbridge.local.json -transport stdio
```

Run Streamable HTTP on the configured loopback listener:

```sh
workbridgemcp -config workbridge.local.json -transport http
```

Copy `config.example.json` to the ignored `workbridge.local.json` and replace the example root before running it. Process execution remains off until deliberately enabled.
