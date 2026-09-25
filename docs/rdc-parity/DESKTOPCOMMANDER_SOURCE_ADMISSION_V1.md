# DesktopCommanderMCP Source Admission V1

Status: `ADAPT_WITH_PROVENANCE`

## Source

Repository: `wonderwhy-er/DesktopCommanderMCP`

Exact source head:

`550a0b3e31da18b7cf25e87ed840e3d953b6da42`

License: MIT.

This admission is limited to the open-source workstation agent and its local process/session/device mechanics. The proprietary hosted Remote Desktop Commander relay/service is **not** admitted.

## Why this source matters

The open-source agent contains concrete implementations and regression tests for the workstation behaviors WorkBridge still lacks relative to the Remote Desktop Commander surface:

- persistent interactive child-process sessions;
- stdin interaction with a running process;
- retained output after process completion;
- offset/tail pagination over retained process output;
- bounded output retention;
- active-session enumeration;
- OS process enumeration;
- process termination;
- local tool-call history;
- remote-device readiness/reconnect semantics.

The useful design lesson is not “copy Desktop Commander wholesale.” WorkBridge already has a stricter authority model: bounded roots, explicit process enablement, SHA-256-pinned executable grants, bounded runtime/output, and loopback transport. The admitted pattern is therefore:

`DesktopCommander session semantics + WorkBridge authority model + VeraMesh transport/readiness`

## Exact admitted source blobs

- `src/terminal-manager.ts` — `528e2872754315751dc1559922bbf9764cdcd702`
- `src/tools/improved-process-tools.ts` — `92bad49daf49230a60483fcc6a798802218d2e36`
- `src/tools/process.ts` — `c91bcc7c831fbb9466847c2a4ebe159a1d6be74e`
- `src/handlers/terminal-handlers.ts` — `1f6930fdd6dcf28aebecd99036eab2585edaac23`
- `src/handlers/process-handlers.ts` — `fef82042b75324d6ccf7ab566f737daad5c09487`
- `src/handlers/history-handlers.ts` — `1c65a6c2eded665afbea2c54b60a98ef4ead43d4`
- `src/utils/toolHistory.ts` — `6c9d548812316eb4c32ef7f010c66100af902a34`
- `src/remote-device/device.ts` — `0b3354662cef7a561eefb4a4c353482f7d11d762`
- `src/remote-device/remote-channel.ts` — `45b81c976d64a5aba1c7cfaee683faf1a8c8e802`
- `src/remote-device/README.md` — `5f5fc0a8979fe0fa55b7f0431990541be8ff134b`
- `test/test-process-pagination.js` — `30da2c029cb95b4ac7777e856f27ce046fe39e7a`
- `test/test-read-completed-process.js` — `ef412331227a0e54971f54c23dafe1789ca4a9d2`
- `test/test-list-processes.js` — `5c799cb3cc6efabf7f70f4e13de2ebb49358a09e`
- `test/test-remote-device-readiness.js` — `5e05264c4f2662a00021443479100742c05535fc`
- `LICENSE` — `cc6ce4fa68f26cd5db3dc1ff83c8163094f28451`

## Adaptation rules

1. Preserve WorkBridge's explicit process capability gate.
2. Preserve named executable grants with exact executable SHA-256.
3. Preserve configured working-root confinement.
4. Do not add an unrestricted shell-string execution surface.
5. Do not inherit Desktop Commander's hosted relay, telemetry, usage/accounting, onboarding, or proprietary service behavior.
6. Session handles must be WorkBridge-issued opaque IDs; OS PIDs may be reported as evidence but must not be the sole authority token for session mutation.
7. Output retention must be bounded.
8. Completed-session output must remain readable for a bounded retention interval/count.
9. Session interaction must reject closed/completed sessions.
10. Device readiness/reconnect remains a VeraMesh responsibility rather than being duplicated in WorkBridge.
11. Any copied/adapted code must preserve MIT attribution/provenance.
12. Source admission is not independent corroboration.

## Rejected carryovers

The following RDC behaviors are intentionally not parity requirements:

- changing security configuration through the same general-purpose tool surface;
- “empty allowedDirectories means full filesystem access”;
- unrestricted command strings/default-shell execution;
- hosted usage metering/accounting;
- feedback/onboarding tools;
- proprietary device relay/database coupling.

## Claim ceiling

`RDC_WORKSTATION_MECHANISM_SOURCE_ADMISSION_ONLY`

This admission says the open-source agent is a valid implementation reference. It does not claim WorkBridge currently has parity or that any runtime/process authority has been enabled on Lappy.
