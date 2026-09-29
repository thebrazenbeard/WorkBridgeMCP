# Concurrent Process Workers V1 Implementation Plan

> **For agentic workers:** Use the host's available task-by-task implementation workflow. Steps use checkbox syntax for tracking.

**Goal:** Add configurable four-way bounded process concurrency to WorkBridge while preserving its single MCP endpoint and all existing process-policy guarantees.

**Architecture:** Extend ProcessConfig with a validated max_concurrent value defaulting to four. Runner owns a cancellable semaphore that gates only process launch; every invocation keeps independent context, buffers, working directory, timeout, and executable-identity checks. No extra listeners, paid services, or MCP API changes are introduced.

**Tech Stack:** Go, standard library contexts/channels/exec, existing WorkBridge config/policy/runner packages, Go test and race detector.

## Global Constraints

- One existing MCP HTTP endpoint; no additional exposed ports.
- Default process concurrency is 4 and remains locally configurable.
- Preserve executable grants, SHA-256 admission/readback, working-root policy, argument limits, sanitized environment, timeout, and output caps per invocation.
- Capacity waits honor cancellation and never launch after cancellation.
- Every acquired slot releases on all exits.
- Existing clients and MCP response schemas remain unchanged.
- Zero paid dependencies or infrastructure.

---

### Task 1: Configuration contract

Files: modify internal/config/config.go, internal/config/config_test.go, config.example.json.
Interface: ProcessConfig.MaxConcurrent int; default 4; valid range 1 through 32 after defaults.

- [ ] Add failing tests for default 4, explicit 1 and 4, negative rejection, and above-32 rejection.
- [ ] Run go test ./internal/config and observe the missing behavior.
- [ ] Add JSON field, default, and validation without coupling it to process.enabled.
- [ ] Run go test ./internal/config; expect pass.
- [ ] Update config.example.json with max_concurrent 4.
- [ ] Commit feat: configure bounded process concurrency.

### Task 2: Cancellable runner admission gate

Files: modify internal/runner/runner.go and internal/runner/runner_test.go.
Interfaces: Runner owns slots chan struct{} sized from MaxConcurrent; unexported acquire(ctx) and release(); Run acquires immediately before child process launch and defers release.

- [ ] Add deterministic tests: concurrency 1 blocks second launch; concurrency 4 admits four; fifth waits; cancellation while waiting returns context cancellation without launch.
- [ ] Run go test ./internal/runner and observe failures before implementation.
- [ ] Implement channel semaphore admission while preserving all existing pre-launch policy checks.
- [ ] Verify timeout and child failure release capacity and concurrent output/working directories remain isolated.
- [ ] Run go test ./internal/runner; expect pass.
- [ ] Commit feat: bound concurrent process execution.

### Task 3: Full qualification and packaging

Files: update existing process documentation; modify binary smoke script only if its contract supports a concurrency assertion.
Interfaces: Windows workbridge-mcp.exe built from exact qualified head; installed config max_concurrent 4.

- [ ] Run go test ./...; all packages green.
- [ ] Run go test -race ./... where supported; any new race blocks deployment.
- [ ] Run go vet ./...; expect clean.
- [ ] Build candidate and run existing binary/HTTP smoke tests against temporary config/alternate loopback port, leaving live bridge untouched.
- [ ] Record SHA-256 of installed and candidate binaries; preserve rollback copy.
- [ ] Commit qualification/docs and push exact head.

### Task 4: Rollback-safe live deployment and proof

Files: replace C:/Program Files/WorkBridgeMCP/workbridge-mcp.exe only after qualification; update C:/ProgramData/WorkBridgeMCP/config.json max_concurrent to 4 while preserving unrelated fields.
Interface: endpoint remains 127.0.0.1:8765/mcp and bearer-token contract is unchanged.

- [ ] Identify supported WorkBridge restart mechanism.
- [ ] Stop only WorkBridge, install qualified candidate, update max_concurrent, restart existing transport/config.
- [ ] Verify endpoint health and ordinary calls.
- [ ] Launch four independent blocking test processes and verify overlap; fifth waits until capacity frees.
- [ ] Verify cancellation while waiting creates no child.
- [ ] On failure restore preserved binary/config and restart prior version.
- [ ] Record installed SHA-256, source head, config concurrency, and live proof in repository provenance/review documentation.

## Unresolved externally observable decisions

None. The approved design fixes default concurrency at four, retains the existing endpoint, and preserves API behavior.
