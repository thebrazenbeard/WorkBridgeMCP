# WorkBridge black-box evaluations

These cases are release gates for the capabilities WorkBridge actually exposes.

They are deliberately adversarial. A tool being present is not evidence that its boundary
is safe, and a source-level test is not an installed workstation effect.

The machine-readable case list is `cases.json`.

## Required classes

- MCP initialization and tools/list;
- absolute-root containment;
- traversal/symlink denial;
- mutation-tool absence in read-only profiles;
- explicit executable-grant identity;
- process working-root containment;
- literal-loopback HTTP;
- optional bearer rejection;
- malformed-request survival.

## Claim ceiling

The v0.1 suite does not prove resistance to a malicious local actor racing Windows
junctions/reparse points or executable replacement between final pre-spawn verification
and OS execution. The documentation must continue to say so until an OS-handle-bound
implementation and hostile race harness exist.

## Suggested execution profiles

1. **read-only stdio** — no write roots, process disabled.
2. **write-enabled stdio** — one disposable temp write root.
3. **process-enabled stdio** — one pinned benign executable and disposable working root.
4. **loopback HTTP** — bearer enabled and disabled variants.

Run every destructive/mutation case only against disposable directories and test
executables.
