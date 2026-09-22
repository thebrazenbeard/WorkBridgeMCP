# WorkBridgeMCP black-box evals

These evals intentionally treat the WorkBridge binary as an external subject.
They do not import the Go implementation or depend on internal package names.

That makes them suitable for:
- release qualification;
- independent hostile review;
- catching regressions after packaging;
- verifying that public behavior matches the documented security contract.

## Required cases

The machine-readable registry is `cases.json`.

A runner should distinguish:
- **PASS** — expected externally observable behavior occurred;
- **FAIL** — observable contract violation;
- **UNSUPPORTED** — feature is intentionally absent in this build;
- **INCONCLUSIVE** — the harness could not establish the result.

An inconclusive or unsupported case must never be promoted to PASS.

## High-value hostile cases

The first release-quality suite should cover:

1. MCP initialize + tools/list over stdio.
2. Configured filesystem root cannot be escaped with `..`.
3. Absolute path outside the configured root is rejected.
4. Symlink/junction traversal cannot escape the root.
5. Read-only mode rejects writes.
6. Process execution disabled by policy fails closed.
7. Process command allow/deny policy cannot be bypassed through shell metacharacters.
8. Process output/session handles cannot cross sessions.
9. Unknown/stale process handles fail closed.
10. Loopback HTTP mode does not bind a public interface unless explicitly configured.
11. A malformed JSON-RPC/MCP request returns an error without crashing the server.
12. Tool argument schema rejects unknown/incorrectly typed fields where the contract requires strictness.

The black-box harness is evidence of observable behavior only. It does not prove the
host OS prevented every possible escape or that an external dependency is bug-free.
