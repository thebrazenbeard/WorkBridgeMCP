# WorkBridge Commander / Desktop Commander process-capacity trace — 2026-09-30

Status: LIVE OBSERVATION + SOURCE-BOUND ROOT CAUSE; FIX CANDIDATE NOT DEPLOYED

## Live subject

The connected WorkBridge Commander service on Lappy reported one device, generation 1, `executionCapacityPerDevice=8`, and `upstreamContextCapacity=64`. The current process environment exposes `WORKBRIDGE_EXECUTION_CAPACITY=8` and `WORKBRIDGE_UPSTREAM_CONTEXT_CAPACITY=64`. The Commander device agent launches the pinned Desktop Commander duplicate with the inherited process environment.

The installed duplicate is pinned to upstream DesktopCommanderMCP commit `550a0b3e31da18b7cf25e87ed840e3d953b6da42`, package version 0.2.51, plus the WorkBridge-owned bounded-process overlay.

No service restart, package replacement, token change, or permission change was performed during this trace.

## Full saturation precursor

The preceding full live saturation test established:

- 64 / 64 upstream work contexts simultaneously active;
- 8 / 8 Commander execution lanes simultaneously active;
- 56 effect-bearing requests queued behind those eight lanes;
- 64 / 64 requests completed successfully;
- terminal state drained to zero active / zero queued.

That test showed regular sub-waves inconsistent with the advertised eight-lane execution capacity, motivating stage tracing.

## Stage-trace method

A direct loopback MCP harness tagged each `start_process` request with a unique marker and requested Desktop Commander's existing `verbose_timing` output. The harness correlated those results with the duplicate's local `tool-history.jsonl` completion timestamp and duration.

Derived intervals:

1. client request start -> approximate Desktop Commander tool-handler receipt;
2. Desktop Commander whole-tool duration;
3. Desktop Commander terminal/process duration from `verbose_timing`;
4. pre-terminal wrapper/admission time = whole-tool duration minus terminal duration;
5. Desktop Commander completion -> client response.

This uses existing runtime telemetry and does not patch or restart the live service.

## One-context baseline

For a two-second PowerShell workload:

- client end-to-end: 2,304 ms;
- Desktop Commander whole-tool: 2,295 ms;
- terminal/process lifetime: 2,285 ms;
- approximate client-to-Desktop receipt: 6 ms;
- pre-terminal wrapper overhead: 10 ms;
- Desktop-complete-to-client: 3 ms.

At no load, the Commander bridge is not a material source of latency.

## Eight-context result

Eight simultaneous two-second PowerShell calls all reached Desktop Commander in roughly 170 ms, but only four immediately entered the terminal/process stage.

Fast half pre-terminal overhead: approximately 20–43 ms.
Slow half pre-terminal overhead: approximately 2,180–2,898 ms.

An immediate-output discriminator confirmed the delay occurs before Desktop Commander's terminal timer begins, not while waiting for the PowerShell process to start emitting output.

Zero-workload PowerShell and zero-workload `cmd.exe` tests retained the same four-fast / four-delayed shape, ruling out the two-second workload and PowerShell specifically.

A direct Node `child_process.spawn()` benchmark outside Desktop Commander launched eight `cmd.exe` children in approximately 79 ms total and eight PowerShell children in approximately 378 ms total, with individual synchronous `spawn()` calls approximately 3–12 ms. This falsified generic Windows/Node process creation as the multi-second cause.

## Sixty-four-context stage trace

For 64 simultaneous two-second process calls:

- 64 / 64 succeeded;
- peak upstream active: 64;
- peak execution active: 8;
- peak execution queued: 56;
- median client total: 19,422 ms;
- median client-to-Desktop receipt / Commander queue delay: 14,955 ms;
- median Desktop whole-tool duration: 4,422 ms;
- median pre-terminal wrapper overhead: 2,145 ms;
- median terminal/process lifetime: 2,276 ms;
- median Desktop-complete-to-client return: approximately 1 ms.

59 of 64 calls spent 1.5–3.0 seconds in the pre-terminal portion; only four were below 250 ms.

A telemetry-suppressed `origin=ui` experiment sharply reduced eight-way no-op latency, so Desktop Commander telemetry contributes under moderate concurrency. It did not remove the approximately 2.3-second pre-terminal delay under full 64-context saturation, so telemetry is not the root capacity limiter.

## Root cause from installed source

The installed WorkBridge-owned overlay contains:

```ts
export const workbridgeProcessAdmission = new ProcessAdmissionGate(4);
```

and the WorkBridge-derived `terminal-manager.ts` wraps the complete `executeCommand()` lifecycle in:

```ts
return workbridgeProcessAdmission.run(async () => {
  // ... shell resolution, spawn, process wait, completion ...
});
```

Therefore WorkBridge Commander admits eight workstation effects, but the qualified Desktop Commander duplicate admits only four process lifetimes. Requests five through eight wait inside the duplicate before the terminal timing interval begins. The hidden four-slot gate exactly explains the observed two sub-waves of four.

The fixed-four gate is not present in pinned upstream DesktopCommanderMCP; it is a WorkBridge overlay introduced by the earlier bounded-process V1 design.

## Repair bound

The V2 source candidate preserves the original standalone default of four but resolves the inner gate from `WORKBRIDGE_EXECUTION_CAPACITY` when explicitly supplied. The current Commander route already supplies 8 through inherited environment. Allowed values remain bounded 1 through 32; malformed or out-of-range values fail closed.

This source result is not deployment evidence. The installed live duplicate remains at the V1 fixed-four gate until an explicitly authorized rebuild/install/restart occurs.

## Hostile review

> **HOSTILE REVIEWER:** Matching the inner gate to eight could merely move the bottleneck downstream and increase resource pressure.

**ACCEPTED.** V2 removes a proven artificial mismatch; it does not prove eight concurrent process lifetimes are resource-safe or lower-latency on every workload. A qualified candidate must undergo an isolated eight-way process test and, only after an authorized installation, the live 8/64 saturation test again.

> **HOSTILE REVIEWER:** The `origin=ui` experiment suppresses telemetry and is not a valid production optimization.

**ACCEPTED.** It was used only as a diagnostic A/B. V2 does not alter origin semantics or suppress telemetry.

> **HOSTILE REVIEWER:** The environment variable is an implicit cross-repository contract.

**PARTIALLY ACCEPTED.** Commander already consumes and exposes `WORKBRIDGE_EXECUTION_CAPACITY`, and its device agent inherits the process environment into the duplicate. V2 records this binding in the duplicate manifest and CI. Future route changes must preserve or deliberately revise that contract.



## V2 candidate qualification

The supported Desktop Commander duplicate installer was run against a scratch install root, not the live `C:\ProgramData` installation, using the V2 overlay and `-RunTests`.

The installer completed successfully. Because the installer moves the staging tree into the requested install root only after its focused overlay test and full upstream Desktop Commander test suite succeed, the resulting scratch candidate establishes that both gates passed for the exact pinned upstream source plus V2 overlay.

The scratch manifest records:

- upstream commit `550a0b3e31da18b7cf25e87ed840e3d953b6da42`;
- upstream version `0.2.51`;
- `workbridge_process_concurrency_default=4`;
- `workbridge_process_concurrency_max=32`;
- `workbridge_process_concurrency_env=WORKBRIDGE_EXECUTION_CAPACITY`;
- overlay `bounded-process-concurrency-v2`.

The candidate acceptance probe passed with `WORKBRIDGE_EXECUTION_CAPACITY=8`, returning Desktop Commander 0.2.51 with the expected unrestricted command-string tool surface.

An isolated eight-way real-process probe against the scratch candidate used eight simultaneous PowerShell commands, each performing a two-second sleep and emitting a unique marker. All eight markers were observed, all calls exited via `process_exit`, and the full eight-call wall clock was 3,644 ms. Individual terminal lifetimes were 2,558–3,565 ms. This demonstrates eight overlapping real process lifetimes in the candidate and eliminates the V1 fixed-four two-wave behavior under the same class of workload.

This remains candidate qualification, not live deployment verification.


## CI qualification note

The first PR workflow attempt exposed an unrelated Windows-only dependency bootstrap failure before the WorkBridge overlay step: `@vscode/ripgrep` attempted unauthenticated release discovery through the GitHub API and received HTTP 403.

The supported WorkBridge duplicate installer already avoids that failure mode by downloading the exact pinned v15.0.0 Windows asset directly and verifying SHA-256 before `npm rebuild`. The PR workflow now primes the same verified cache before the Windows rebuild instead of weakening or skipping Windows qualification.


## Live V2 canary route qualification

The qualified V2 duplicate was copied byte-for-byte from the already-qualified scratch installation into a side-by-side live candidate root:

`C:\ProgramData\WorkBridgeMCP\DesktopCommanderMCP.v2.a135586`

The staged candidate preserved these verified hashes:

- manifest SHA-256: `8da114dd4c1aee247a5e28230b9bc319da51148229e1d5f68f188abaf5501fe6`;
- packaged Node SHA-256: `3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237`;
- built `dist/index.js` SHA-256: `a4145198dc75cd34e7c7452c2054b4cc0d29e199ae1459961a17e4da25a13500`;
- built V2 admission module SHA-256: `36af43eaefc53b5df8053923f0dc105ef24e77b1b5524d9062e6da39bec79408`.

Instead of interrupting unrelated active work on the existing `lappy` device, the V2 candidate was attached temporarily to the already-running Commander server as a second device identity, `lappy-v2-canary`, using the existing authorized device credential and the V2 manifest trust hash. The original `lappy` route remained live throughout. No credential was rotated or exposed.

Commander health reported two registered devices, each with execution capacity 8, before the canary load tests.

### Eight-way canary

Eight simultaneous two-second PowerShell workloads were sent through the live Commander MCP route to `lappy-v2-canary`.

Result:

- success: 8 / 8;
- failure: 0;
- p50 end-to-end: 2,859 ms;
- p95 / max end-to-end: 3,375 ms;
- all terminal lifetimes approximately 2.27–3.15 s.

The aggregate health peak showed 9 execution-active requests only because the harness itself was one active request on the original `lappy` route; the canary supplied the other eight.

Raw evidence:

- `docs/evidence/WORKBRIDGE_COMMANDER_V2_CANARY_8WAY_20260930.json`
- SHA-256 `9cef20ced307277c3bdc2487f9eefa3e26f1575f7a48ea4f6086cca8c923065d`

### Full 64-context / 8-execution canary

The harness was then detached before issuing 64 direct MCP requests to avoid consuming one of the shared upstream context slots.

Observed live peaks:

- upstream active: **64**;
- upstream queued: **0**;
- execution active: **8**;
- execution queued: **56**;
- success: **64 / 64**;
- failed: **0**;
- post-drain upstream active / queued: **0 / 0**;
- post-drain execution active / queued: **0 / 0**.

For the same two-second workload class used by the prior V1 stage trace:

| Metric | V1 fixed-four overlay | V2 canary | Improvement |
| --- | ---: | ---: | ---: |
| p50 | 19,422 ms | 12,424 ms | 36.0% |
| p95 | 40,058 ms | 21,252 ms | 46.9% |
| max | 45,556 ms | 22,063 ms | 51.6% |
| mean | 20,330 ms | 12,410 ms | 39.0% |

Raw evidence:

- `docs/evidence/WORKBRIDGE_COMMANDER_V2_CANARY_64X8_20260930.json`
- SHA-256 `0c11ae4b17dceeeb80775d47856e1b9005f07faf57de3049179341c6fa8a2b5c`

The canary was then stopped cleanly. Commander returned to one registered device (`lappy`).

### Live cutover gate

A rollback-safe cutover script was prepared at:

`D:\VERA\tools\WorkBridgeCommander\Cutover-DesktopCommanderV2.ps1`

It verifies old/new manifest hashes, candidate manifest semantics and packaged hashes, waits for the existing Desktop Commander child to become quiescent, preserves the old duplicate directory and launcher as rollback material, changes only the duplicate manifest trust anchor, restarts only the device-agent/duplicate path, performs a real MCP process probe, and automatically restores the old route if qualification fails.

Its preflight correctly refused cutover while unrelated active Desktop Commander sessions existed. A later Commander health observation also showed a separate 32-context workload in flight on the original route. Therefore no live cutover was attempted during those collisions.

This is a deliberate non-collision hold, not a failed V2 qualification.


## Authorized live cutover attempt — rollback defect found

Patrick explicitly authorized termination of the four unrelated long-running retrieval PIDs and the exact V1 -> V2 live cutover on Lappy. The four previously identified retrieval sessions were terminated and Desktop Commander reported no active sessions before cutover.

The prepared cutover was launched detached so it could survive the old WorkBridge device route shutting down. The attempt did **not** replace the live V1 duplicate. The result file recorded:

- status: `ROLLBACK_EXCEPTION`;
- primary failure: Windows refused to rename `C:\ProgramData\WorkBridgeMCP\DesktopCommanderMCP` because the directory was in use;
- rollback hit the same rename failure.

Out-of-band VeraPort readback after the failed attempt established:

- live manifest still reports `bounded-process-concurrency-v1` and fixed process concurrency 4;
- live manifest SHA-256 remains `0e2e80ae5acd26c04e0adb5ac21b453edbe5b7004505998d2692b7b948ae1e2f`;
- `Start-WorkBridgeCommander.ps1` still contains the old V1 trusted manifest hash;
- V2 candidate directory `C:\ProgramData\WorkBridgeMCP\DesktopCommanderMCP.v2.a135586` remains present and intact;
- Commander server startup log remains bound to `127.0.0.1:8787` at 8 execution / 64 upstream capacity;
- tunnel health URL file remains present;
- the WorkBridge device agent stopped when the old Desktop Commander child exited and did not recover because rollback failed before `Start-Device`.

The direct cause of the rename failure is a cutover-script design defect: the detached PowerShell process inherited a current working directory inside the live Desktop Commander install tree. Windows therefore held that directory in-use even after the old Desktop Commander process exited.

> **HOSTILE REVIEWER:** A rollback script that shares a working directory with the tree it must rename is not rollback-safe.

**ACCEPTED.** Future cutover logic must move to a neutral working directory before any process stop or install-root rename, and the recovery channel must be independent of the route being replaced.

No credential, launcher trust anchor, server configuration, tunnel configuration, or V2 candidate file was changed by this failed cutover. The live WorkBridge Commander tool route is currently unavailable only because its device agent is stopped; the repository fix and side-by-side V2 candidate remain valid.
