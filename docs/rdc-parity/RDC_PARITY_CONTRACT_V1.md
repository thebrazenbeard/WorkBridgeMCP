# RDC-Class Workstation Parity Contract V1

Status: `IMPLEMENTATION_TARGET / CORE_RUNTIME_DELEGATED_TO_BT2`

Base WorkBridge subject:

`PR #8 @ 6f578307d71b13ab93a8ab9e48bdfc547437a0e5`

Reference source:

`wonderwhy-er/DesktopCommanderMCP@550a0b3e31da18b7cf25e87ed840e3d953b6da42`

## Goal

The target is not pixel-level remote desktop and not duplication of Desktop Commander's hosted service.

The target is **RDC-class build-workstation control** through the composed architecture:

`ChatGPT -> VeraMesh transport/device layer -> WorkBridge workstation API -> Lappy`

A build chat should be able to inspect a repository, search files, edit files, start a compiler/test process, keep an interactive session alive, read new or historical output, send stdin, inspect processes, terminate its own session safely, and recover enough recent tool history to resume work after chat/context loss.

## Responsibility split

### VeraMesh owns

- workstation identity;
- authenticated route establishment;
- reconnect/backoff;
- readiness advertisement;
- route health/latency;
- capability negotiation;
- tunnel transport;
- current-session admission.

### WorkBridge owns

- filesystem operations;
- streaming/background searches;
- bounded one-shot process execution;
- persistent interactive process sessions;
- process output retention/pagination;
- process/session termination;
- OS process observation/control when separately authorized;
- local tool-call history.

This split avoids importing Desktop Commander's proprietary hosted relay.

## Current WorkBridge V1 capabilities

Current PR #8 registers:

- `workbridge_health`
- `workspace_list`
- `workspace_stat`
- `workspace_read_text`
- `workspace_write_text`
- `workspace_mkdir`
- `process_run`

Current process execution is safer than RDC's general shell surface:

- globally gated by `process.enabled`;
- exact named executable grants;
- exact executable SHA-256;
- configured working roots;
- bounded runtime;
- bounded output;
- bounded argument count;
- reduced environment.

These invariants remain mandatory.

## Required P0 parity additions

### Persistent process sessions

Required API semantics:

- `process_session_start`
- `process_session_read`
- `process_session_write`
- `process_session_list`
- `process_session_terminate`

A session start must reference an existing named executable grant. Arbitrary shell command strings do not qualify.

A successful start returns:

- opaque `session_id`;
- observed OS PID;
- executable grant name;
- canonical working directory;
- start timestamp;
- state;
- initial output slice when available.

### Output retention and pagination

Each session maintains a bounded line-oriented output buffer.

Required read modes:

- default/new-output cursor;
- absolute nonnegative line offset;
- negative tail offset;
- bounded `length`.

Required metadata:

- retained total line count;
- read-from offset;
- read count;
- remaining retained lines;
- completed/running state;
- exit code when complete;
- evicted line count;
- runtime.

Unbounded output accumulation is forbidden.

Completed-session output remains available after exit under a bounded retention policy.

### Interactive stdin

`process_session_write` sends literal bytes/text to one WorkBridge session.

It must:

- require the opaque session ID;
- reject unknown/completed sessions;
- bound input size;
- never reinterpret the input as a new WorkBridge command;
- make newline behavior explicit instead of silently depending on shell conventions.

### Session lifecycle

`process_session_list` reports active and retained-completed sessions without exposing unrelated OS processes.

A session must have an explicit state machine such as:

`STARTING -> RUNNING -> WAITING_OR_IDLE -> COMPLETED | TERMINATED | FAILED`

Shutdown must cancel/terminate owned child processes according to declared policy and must not leave a session marked ready after the executor is gone.

### OS process observation/control

Separate from WorkBridge-owned sessions:

- `process_list`
- `process_kill`

These require a distinct configuration gate from session execution.

`process_list` is observational.

`process_kill` is mutating and must never become authorized merely because `process.enabled` is true. It requires an explicit `process.allow_external_control`-class gate or equivalent.

PID alone is not sufficient provenance for a WorkBridge-owned session mutation; use opaque session ID for session APIs.

## Required P1 parity additions

### Filesystem/search

Target additions:

- multi-file read;
- move/rename;
- bounded exact-text replacement;
- filename search;
- content search;
- asynchronous/streaming search sessions with read-more/stop/list operations where needed for large trees.

Existing root confinement remains authoritative.

### Local tool-call history

A bounded in-memory history surface should expose recent WorkBridge calls/results for continuation/debugging.

It must:

- redact bearer tokens and configured secret values;
- bound entry count and per-entry payload size;
- distinguish success/error;
- never become canonical project history;
- disappear on restart unless a separate durable-history feature is explicitly designed.

## Explicit non-goals

P0 parity does not require:

- PDF/DOCX/XLSX convenience transforms;
- browser/UI automation;
- unrestricted shell;
- modifying WorkBridge security config from the same tool surface;
- usage/billing stats;
- feedback/onboarding prompts;
- proprietary RDC relay behavior.

## Security invariants

1. `process.enabled=false` means no one-shot or persistent process can start.
2. Every started process resolves through an exact named executable grant.
3. The executable's current SHA-256 must still match its grant at start time.
4. Working directory must resolve within configured process working roots.
5. Session count is bounded.
6. Input size is bounded.
7. Retained output per session is bounded.
8. Completed-session retention count/time is bounded.
9. Session IDs are unguessable opaque identifiers.
10. A session ID cannot authorize external-process kill.
11. External OS-process kill is separately gated.
12. Process environment remains reduced/allowlisted.
13. Server shutdown cannot leave a child marked live/ready.
14. Tool history redacts configured secrets.
15. VeraMesh must advertise the workstation ready only while both transport and executor are usable.

## Acceptance definition

“RDC-class build parity” requires every P0 acceptance case in `evals/rdc-parity/acceptance_cases_v1.json` to PASS on:

- Windows/Lappy-class environment;
- Linux CI where semantics are portable;
- exact WorkBridge head;
- exact VeraMesh compatibility head for end-to-end transport cases.

A PASS does not authorize installation or process enablement on Lappy. Runtime authority remains a separate protected effect.

## Claim ceiling

`RDC_CLASS_BUILD_WORKSTATION_PARITY_WITHIN_DECLARED_WORKBRIDGE_VERAMESH_SURFACE`

This does not claim GUI remote desktop, unrestricted administrative control, or equivalence to every Desktop Commander convenience feature.
