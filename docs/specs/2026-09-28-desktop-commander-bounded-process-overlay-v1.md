# Desktop Commander Bounded Process Overlay V1

Status: APPROVED

The live Lappy Desktop Commander connector is the pinned DesktopCommanderMCP duplicate path, not WorkBridge's disabled native Go process runner. The fix therefore remains a WorkBridge-owned overlay applied after exact upstream checkout and dependency installation, before build.

The overlay preserves upstream commit 550a0b3e31da18b7cf25e87ed840e3d953b6da42 and package 0.2.51 as source provenance. It replaces terminal-manager.ts with the pinned-source-derived WorkBridge version and adds workbridge-process-admission.ts. A FIFO admission gate permits four executeCommand lifetimes concurrently; a fifth waits until capacity is released. The gate wraps the complete executeCommand promise, so capacity remains occupied while a spawned process is active or start_process is waiting on that process. Release occurs through finally.

The installer must apply the overlay deterministically, run the focused overlay concurrency test on every build, optionally run the full upstream suite, and record workbridge_process_concurrency=4 plus overlay identity in the emitted manifest.

No second listener, MCP endpoint, paid dependency, or upstream fork is introduced. Rollback is the previous installed duplicate tree/manifest.