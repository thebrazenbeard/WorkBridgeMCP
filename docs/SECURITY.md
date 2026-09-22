# WorkBridgeMCP security model

## Default posture

The intended default profile is:

- stdio transport;
- read roots explicitly enumerated;
- no write roots;
- process execution disabled;
- no network listener.

HTTP mode is optional and configuration validation permits only literal loopback
addresses. If `http.bearer_token_env` is configured, startup fails when that environment
variable is empty.

## Filesystem boundaries

Paths must be absolute and remain beneath configured roots after symlink evaluation.

Read operations:

- reject paths outside read roots;
- bound file size;
- accept UTF-8 text only for `workspace_read_text`;
- bound directory entry count.

Write operations:

- are not registered unless write roots exist;
- bound write size;
- create new files with exclusive creation;
- refuse to overwrite symbolic-link leaf nodes;
- require `overwrite=true` for existing files;
- create only one directory level at a time.

### Current claim ceiling

The v0.1 source substantially reduces ordinary traversal and symlink-escape risk, but it
does **not** claim resistance to a hostile local actor racing directory junctions,
reparse points, mount changes, or file replacement between admission and the final OS
operation.

Before WorkBridge is qualified for mutually hostile local users, filesystem mutation
must gain handle-relative/no-follow or equivalent final-handle identity enforcement on
the target OS and hostile race tests.

For a single-owner workstation, keep write roots narrow and do not expose HTTP beyond
loopback.

## Process boundaries

Process execution is absent unless explicitly enabled.

Each executable grant contains:

- an operator-chosen stable name;
- an absolute executable path;
- an exact lowercase SHA-256.

At startup WorkBridge resolves the path, verifies it is a regular file, and hashes it.
Immediately before execution it resolves and hashes the file again. Immediately after
execution it verifies the hash again.

The MCP caller supplies only the grant name and literal argument array. WorkBridge does
not route process requests through a shell.

Working directories are separately bounded by `process.working_roots`.
Runtime, output bytes, and argument count are bounded.

### Current claim ceiling

Hash checks greatly reduce accidental executable substitution but are not a proof
against an attacker who can race replacement between the final pre-spawn hash and the
operating system's executable open. High-assurance deployment should add an OS-specific
execution identity mechanism or place admitted executables in operator-controlled,
non-writable locations.

## Secrets

Do not put credentials, bearer tokens, API keys, or NAS credentials in:

- repository files;
- service-wrapper XML;
- committed JSON configuration;
- tool arguments unless the called local executable explicitly owns that secret flow.

HTTP bearer values are loaded through an environment-variable indirection.

## Transport

Stdio is preferred for desktop MCP clients.

Loopback Streamable HTTP is useful for a persistent local process. WorkBridge v0.1
intentionally refuses non-loopback listen addresses. Public/LAN exposure requires a new
reviewed transport/authentication design; changing a bind string is not sufficient.

## Effect semantics

Source presence is not installation.
A passing unit test is not a workstation effect.
A successful tool call is evidence of that call, not standing authority for another one.
