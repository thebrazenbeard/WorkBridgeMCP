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
- Its default source pin is now the code-bearing repair commit `613c3df0e7bd1d43b123d249e3aaca4366852546`, rather than an older pre-repair commit.
- It verifies the existing VeraPort config hash alongside the identity/controller file snapshot.
- Windows PowerShell 5.1 and PowerShell 7 listener-guard checks: PASS.
- Go 1.25.12 `go test ./...`, `go vet ./...`, and `go build ./cmd/workbridge-mcp`: PASS locally on Windows.

## Source acceptance and remaining gates

- Source repair is in draft PR #6 against PR #5's candidate branch. Before this record update, its remote head was `ab5ca2dd263bdb35e9fcecb297bf094e3a43017e` and CI run `35846554913` passed Ubuntu tests, Windows tests, and Windows binary build.
- The exact-head local binary had SHA-256 `3853be9cb3f0af1ad833e29e1eb022906a0979e3531534147573ae8daccfc62a`. Stdio MCP initialize/tools-list and authenticated HTTP checks passed; unauthorized HTTP health returned 401. The VeraMesh adapter's local real-binary integration suite passed against that binary.
- VeraMesh draft PR #33 added an exact WorkBridge checkout to its Windows real-HTTP integration job. Its initial CI, reference, and CodeQL runs passed. Refresh its WorkBridge source pin if this WorkBridge branch head moves.
- The separate GitHub security-agent job failed before review because its requested model was unsupported. This is neither a source test failure nor an independent security review pass.
- Lappy remained offline at the latest available Remote Desktop Commander check. There is no live Lappy install/runtime/ChatGPT effect readback. Installation, service registration, credentials, and ChatGPT registration remain separate protected effects.

## Recovery after a rate or context limit

1. Read PR #6 and PR #33 descriptions, then verify both remote heads and their exact-head checks. Do not treat the hashes in this checkpoint as current without readback.
2. Check that VeraMesh CI pins the current intended WorkBridge source commit. A WorkBridge documentation commit also moves the Git head and requires a corresponding pin update if exact-head binding is promised.
3. Confirm Lappy connectivity before attempting read-only runtime qualification. Do not infer installation or tool effect from source CI.
4. Resume only authorized, non-colliding work. Merge, installation, credentials, provider/network changes, and ChatGPT registration require their own exact authority.
