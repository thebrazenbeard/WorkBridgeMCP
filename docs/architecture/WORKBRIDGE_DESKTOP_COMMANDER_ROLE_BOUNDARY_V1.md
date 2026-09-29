# WorkBridge / Desktop Commander / VeraMesh Role Boundary V1

Status: ACTIVE DESIGN BOUNDARY

## Two WorkBridge runtime modes

WorkBridgeMCP now contains two intentionally different capability models.

### Mode A: bounded native WorkBridge

Implementation: Go WorkBridge server.

Purpose:

- constrained filesystem roots;
- optional bounded writes;
- named executable grants;
- executable SHA-256 verification;
- argv execution without implicit shell;
- reduced environment;
- loopback HTTP / stdio;
- low-dependency deployment, including ARMv7 qualification work.

This is the credible base for constrained appliances such as the DS216.

### Mode B: exact Desktop Commander duplicate

Implementation: actual pinned DesktopCommanderMCP TypeScript source.

Purpose:

- preserve upstream Desktop Commander workstation semantics;
- unrestricted `start_process(command=...)` according to upstream configuration;
- interactive sessions;
- process inspection/control;
- native file/search/edit/config/history/document tools.

Mode B must not be silently "secured" by substituting Mode A semantics. Doing so would stop being a duplicate.

## Ownership matrix

| Concern | Owner |
| --- | --- |
| Desktop tool names and schemas | DesktopCommanderMCP exact upstream subject |
| Desktop tool implementation | DesktopCommanderMCP exact upstream subject |
| Upstream source pin | WorkBridgeMCP |
| Build/install package | WorkBridgeMCP |
| Node + entrypoint hashes | WorkBridgeMCP manifest |
| Remote MCP transport | VeraMesh |
| Tunnel process persistence | VeraMesh |
| Transport authentication | VeraMesh / current OpenAI tunnel integration |
| ChatGPT app metadata | WorkBridgeMCP plugin source |
| Concrete tunnel capability URL | runtime-only secret state |
| Emergency/admin recovery | SSH operator plane, independent |
| DS216 bounded appliance backend | native Go WorkBridge candidate |
| Multi-device broker | NOT ASSIGNED in V1 |

## Semantic rule

For the Lappy duplicate route:

`semantic_translation = false`

A transport component may frame, relay, reconnect, authenticate, hash-check, supervise, or route bytes. It must not:

- rename Desktop Commander tools;
- replace schemas;
- rewrite command strings;
- convert Desktop Commander sessions into VeraPort sessions;
- drop tools merely because a bounded WorkBridge equivalent is narrower;
- replay a failed mutation against another backend.

## Security interpretation

"Exact duplicate" does not mean "no security."

Security belongs around the MCP server:

- authenticated tunnel;
- protected runtime files;
- exact binary/source provenance;
- private capability URL;
- service identity;
- explicit plugin installation and ChatGPT action approval;
- independent SSH recovery.

But adding an inner command allowlist to `start_process` would change the semantics and therefore belongs to bounded mode, not duplicate mode.

## Why WorkBridge is still useful when Desktop Commander owns the tools

Without WorkBridge, a remote Desktop Commander deployment could drift into "whatever npm latest happened to install."

WorkBridge turns it into an exact subject:

```text
upstream commit
 + locked build
 + private Node runtime
 + entrypoint hash
 + install manifest
 + qualification tests
 + app contract
```

That provenance function is distinct from implementing the tools.

## Why VeraMesh is still useful

DesktopCommanderMCP is fundamentally a workstation MCP server. It does not by itself establish the desired persistent authenticated ChatGPT-to-private-Lappy path.

VeraMesh already has the merged transport path that launches the exact duplicate through the Secure MCP Tunnel without parsing or translating MCP semantics.

## Why RDC is not the architecture authority

Remote Desktop Commander provides a useful observed user experience, especially multi-device selection. But its hosted service is not admitted as an implementation dependency.

V1 intentionally proves one device first. RDC-like device brokering is a later product/interface decision.
