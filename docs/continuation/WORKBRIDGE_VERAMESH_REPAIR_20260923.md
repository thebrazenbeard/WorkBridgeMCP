# WorkBridgeMCP / VeraMesh repair continuation — 2026-09-23

This is a resumable engineering record, not installation or runtime evidence.
Fresh-check refs, checks, and Lappy state before using it.

## Subject and scope

- Repository: `thebrazenbeard/WorkBridgeMCP`.
- Base candidate: PR #5 `repair/workbridge-current-main-v1-20260922` at `e89a0b43717a4c9a4aabfd1d7e1c967a375f65c4`.
- Repair branch: `fix/workbridge-veramesh-20260923`.
- Allowed work: source, tests, documentation, build checks, and reviewable Git commits.
- No merge, Lappy installation, credential or network change is established by this record.

## Repair and local evidence

- A profile without read roots no longer advertises unusable read tools; an MCP tools/list regression checks the health-only profile.
- The Lappy installer waits for a previous listener to exit and verifies that the post-start listener belongs to the installed binary and the scheduled task is running.
- It verifies the existing VeraPort config hash alongside the identity/controller file snapshot.
- Windows PowerShell 5.1 and PowerShell 7 listener-guard checks: PASS.
- Go 1.25.12 `go test ./...`, `go vet ./...`, and `go build ./cmd/workbridge-mcp`: PASS locally on Windows.

## Open acceptance gates

- Source repair commit `613c3df0e7bd1d43b123d249e3aaca4366852546` was pushed and read back on `fix/workbridge-veramesh-20260923`; draft PR #6 targets PR #5's candidate branch.
- PR #6 exact-source CI run `35845677264`: Ubuntu tests PASS, Windows tests PASS, Windows binary build PASS.
- A Windows binary built from the repair source had SHA-256 `66fddfd9368449189cd1d4e327ac9b5282f0a2e67dda1d250bd5bbf36145f0be`. A live local WorkBridge HTTP process served read/stat/list through the VeraMesh client; write was absent in the read-only profile. PASS.
- VeraMesh's repeatable real-binary integration test and exact-source CI binding are on its separate `fix/veramesh-workbridge-20260923` branch; that branch still requires its own remote readback and CI at this checkpoint.
- Lappy is offline to the available Remote Desktop Commander connection at the initial check. Fresh-check before any read-only runtime qualification.
- Installation, service registration, credentials, and ChatGPT end-to-end effect remain separate protected effects.

The GitHub security-agent check on PR #6 at `613c3df` failed because its requested review model was unsupported. Its job log reached runner steps; this is neither a source test failure nor an independent security review pass. Fresh-check CI after this documentation update because any head movement creates a new exact subject.
