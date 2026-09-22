# Source provenance

This build intentionally separates design input from copied implementation.

## Filamind

Reference repository:

- `filamind-app/filamind-ai`
- observed source head: `620b1c74fd57e8059983e2519d4f8c3db0f476c7`
- upstream license: Apache-2.0

WorkBridge used Filamind as engineering evidence for:

- treating an announced feature as incomplete until the actual executable path is wired
  and tested;
- keeping tool loops bounded;
- explicit tool allowlists;
- source-only packaging versus actual installed/runtime state;
- workstation/WSL-to-device build and packaging discipline;
- honest device/runtime qualification.

No Filamind source file was copied into WorkBridge by this implementation branch.

Filamind's Synology LLM runtime, chat UI, model runtime, daemon implementation, and
device-specific AI binaries are outside WorkBridge's scope.

## MCP Go SDK

WorkBridge uses the official Go SDK as a dependency:

- `github.com/modelcontextprotocol/go-sdk`
- configured dependency: v1.8.0

The WorkBridge server follows the SDK's public server/tool/stdio/Streamable HTTP APIs.

## Other Synology MCP research

Additional Synology MCP repositories were reviewed as architecture references for
permission tiers, DSM API breadth, authentication, and session recovery. WorkBridge does
not vendor or copy their source in this branch.

If third-party source is later copied or substantially derived, its exact source commit
and license obligations must be added here and to the appropriate third-party notice
file before merge.
