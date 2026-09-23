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

- Commit and push this repair branch; verify the remote exact head and its CI result.
- Exercise the real WorkBridge binary through the VeraMesh client, then bind both exact heads in the result.
- Lappy is offline to the available Remote Desktop Commander connection at the initial check. Fresh-check before any read-only runtime qualification.
- Installation, service registration, credentials, and ChatGPT end-to-end effect remain separate protected effects.

The GitHub security-agent check on PR #5 at `e89a0b4` failed because its requested review model was unsupported. Its job log reached runner steps; this is neither a source test failure nor an independent security review pass.
