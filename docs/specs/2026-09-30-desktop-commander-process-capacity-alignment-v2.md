# Desktop Commander Process Capacity Alignment V2

Status: QUALIFIED CANDIDATE / NOT DEPLOYED

## Problem bound from live evidence

The live WorkBridge Commander service on Lappy reports `executionCapacityPerDevice=8` and `upstreamContextCapacity=64`, while the WorkBridge-owned Desktop Commander duplicate overlay still instantiates its inner `ProcessAdmissionGate` with a fixed capacity of 4.

A live 64-context saturation run therefore reached 64 upstream contexts and 8 Commander execution lanes, but the installed duplicate admitted only four `executeCommand()` lifetimes at once. Stage tracing showed the hidden inner gate as pre-terminal latency: requests waited inside Desktop Commander before the terminal timer began. The installed overlay source confirms that the gate wraps the complete `executeCommand()` promise.

This is an alignment defect, not evidence that upstream DesktopCommanderMCP itself has a four-process invariant. The fixed gate is a WorkBridge overlay.

## Successor contract

Preserve the standalone safety default of four while allowing an explicitly configured Commander route to bind the inner gate to the same execution capacity:

- environment absent/blank -> 4;
- `WORKBRIDGE_EXECUTION_CAPACITY=N` -> N;
- accepted range -> integer 1 through 32;
- malformed, zero, negative, fractional or above-32 -> fail closed;
- the gate still wraps the complete `executeCommand()` lifetime and releases capacity in `finally`;
- no extra listener, token, filesystem authority, process grant, or endpoint is introduced.

The existing Commander device agent already spawns the qualified Desktop Commander payload with `{ ...process.env }`, so the current Lappy value `WORKBRIDGE_EXECUTION_CAPACITY=8` propagates without another transport or credential change.

## Evidence ceiling

Source/tests can prove configuration resolution and admission behavior. They do not prove an installed eight-process runtime until a qualified duplicate is rebuilt, installed through the supported installer/restart path, and live 8-way plus 64-context tests are rerun.

The prior V1 design remains historical evidence for the original conservative four-worker default. This V2 supersedes only the assumption that the duplicate must always remain fixed at four when an authorized Commander route explicitly declares a different bounded execution capacity.

## Hostile review

> **HOSTILE REVIEWER:** Raising the inner gate to eight everywhere could double workstation process pressure and erase the original safety boundary.

**ACCEPTED.** The default remains four. Capacity widens only when the existing Commander execution-capacity environment explicitly requests it, and values above 32 fail closed.

> **HOSTILE REVIEWER:** Two schedulers with separately configured capacities can still drift.

**PARTIALLY ACCEPTED.** V2 removes the current hidden fixed-four mismatch by binding the duplicate to the same `WORKBRIDGE_EXECUTION_CAPACITY` environment already consumed by Commander. A future route that does not propagate that environment falls back to four rather than silently widening authority.
