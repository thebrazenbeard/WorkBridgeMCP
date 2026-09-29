# WorkBridge Concurrent Process Workers V1 Design

Date: 2026-09-28
Status: approved design, written-spec review
Base: origin/main

## Goal

Allow WorkBridge/Lappy to execute up to four independent process tasks concurrently through the existing single MCP HTTP endpoint, without paid services, additional exposed ports, or weakened path/executable/process policy.

## Architecture

Keep the existing HTTP listener and MCP surface. Add bounded concurrency at the process-runner boundary using a local in-memory semaphore/worker admission gate. process.max_concurrent is configurable, defaults to 4, and is bounded to a conservative maximum. Each admitted invocation retains its own context, timeout, stdout/stderr buffers, working directory, and process identity. Requests beyond capacity wait for a worker slot and remain cancellable through their request context.

Do not add extra MCP listeners or route tasks by port. HTTP request concurrency and process concurrency remain separate concerns.

## Safety and isolation

All existing executable grants, SHA-256 admission/readback, working-root resolution, argument limits, sanitized environment, runtime timeout, and output caps remain mandatory per invocation. Concurrency never bypasses or caches those checks.

The concurrency gate is acquired only after cheap static admission checks and before process launch. Waiting for capacity honors context cancellation. A worker slot is always released with defer, including process errors, timeout, cancellation, and post-run identity failures.

No shared mutable stdout/stderr buffers are introduced. No command inherits another command's working directory or environment.

## Configuration

Extend process with max_concurrent: integer, default 4.

Validation rejects negative values and values above the implementation ceiling. Zero/missing resolves to the default for backward compatibility. Process capability itself remains controlled only by process.enabled.

The installed Lappy configuration will be updated to 4 only after the new binary passes tests.

## Compatibility

Existing MCP method names and response schemas remain unchanged. Existing clients require no port or connector changes. Serial behavior remains valid when configured to 1.

This change does not alter filesystem concurrency semantics or make compound shell commands safer. Callers should still prefer independent process calls for independent tasks.

## Testing

Add deterministic runner tests proving default concurrency is four; configured concurrency of one serializes blocked runs; configured concurrency of four permits four blocked runs concurrently; a fifth waits; cancellation while waiting does not launch; timeout/error paths release slots; per-run output and working directories remain isolated; existing executable identity and policy tests remain green.

Add config tests for default, valid override, invalid negative, and over-ceiling values. Run the full Go test suite and race detector if supported locally.

## Deployment

Build from the exact qualified source head. Preserve the installed executable as a rollback copy. Stop/restart only the WorkBridge service/process needed to replace the binary, retain the existing single endpoint/token configuration, then run a four-way concurrency smoke test plus fifth-task backpressure test.

If replacing the active executable cannot be done safely through the running bridge, produce the qualified binary and installation command rather than weakening permissions or self-modifying around OS protections.

## Success criteria

Four independent process tasks can overlap measurably through the existing connector; a fifth is bounded until capacity exists; cancellation and policy controls remain effective; no new paid dependency or network listener is introduced.
