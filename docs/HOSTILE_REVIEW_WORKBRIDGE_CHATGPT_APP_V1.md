# Hostile Review — WorkBridge ChatGPT Desktop Commander Idea

Status: REVIEWED / CORRECTIONS INCORPORATED

This memo preserves the adversarial review that changed the implementation direction.

> **Claim:** "The cleanest product is WorkBridge MCP as a clone of Remote Desktop Commander."

> **Hostile reviewer:** That conclusion is not established. The repository proves WorkBridge can own Desktop Commander source/build provenance and VeraMesh can transport it. It does not prove WorkBridge should own the remote broker, device directory, or every RDC-facing semantic.

Disposition: accepted. V1 is single-device and does not implement an RDC broker.

> **Claim:** "Desktop Commander gives us the Remote Desktop Commander surface."

> **Hostile reviewer:** Desktop Commander is fundamentally a workstation MCP server. RDC visibly adds broker/device behavior such as device listing and routing. Those layers are not equivalent.

Disposition: accepted. V1 exposes Desktop Commander exactly on one Lappy endpoint. Device discovery is deferred.

> **Claim:** "We can run the same exact Desktop Commander duplicate on DS216 if ARMv7 works."

> **Hostile reviewer:** ARMv7 CPU compatibility alone is insufficient. Desktop Commander depends on Node and native/platform-sensitive packages. The DS216 platform is substantially more constrained than Lappy.

Disposition: accepted. DS216 is removed from the V1 critical path. Native Go WorkBridge remains the stronger appliance candidate.

> **Claim:** "The live MCP tools/list will automatically become the ChatGPT action catalog."

> **Hostile reviewer:** Protocol truth, plugin package state, ChatGPT action scanning/approval, and the tool catalog loaded into an already-open conversation are separate evidence classes.

Disposition: accepted. ChatGPT PASS requires an explicit live action/tool scan and command round-trip. Repository source and backend capabilities are insufficient.

> **Claim:** "Lappy's exact Desktop Commander tree looked dirty because files differed from GitHub."

> **Hostile reviewer:** A Windows checkout can differ by CRLF line endings while remaining semantically identical. Raw text inequality is not enough to assert source modification.

Disposition: accepted. A normalized readback proved `src/server.ts` and `src/tools/schemas.ts` exactly equal to the pinned upstream content after newline normalization.

> **Claim:** "More adapters are safer."

> **Hostile reviewer:** Every semantic adapter creates another place to lose or alter tool behavior. If exact Desktop Commander semantics are the target, transparent transport is the falsifiable design. Put security around the server rather than silently replacing its API.

Disposition: accepted for Lappy duplicate mode. Bounded native WorkBridge remains a separate mode for environments that intentionally want narrower authority.

## Remaining hostile questions

> How do we prove the remote transport is actually transparent?

By comparing the remote `tools/list` schema against the exact upstream contract and exercising behaviorally significant tools, especially `start_process` plus interactive process flow.

> How do we prove a ChatGPT plugin is not still using an old/stale MCP endpoint?

By binding the rendered plugin to the managed tunnel subject, scanning the installed plugin/tool catalog, and executing a marker through the final route. A package version or plugin name alone is not evidence.

> How do we prevent "exact upstream" from drifting into "latest npm"?

By keeping the Git submodule pin, installer source pin, package version check, Node and entrypoint hashes, and source-derived tool-contract tests.

> How do we stop V1 from becoming another architecture project?

By holding the scope to one device, one exact upstream subject, one tunnel, one ChatGPT app, and one behavioral round trip.
