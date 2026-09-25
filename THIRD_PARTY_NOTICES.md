# Third-party notices

WorkBridgeMCP depends on the official Model Context Protocol Go SDK and its transitive
Go modules as declared by `go.mod` / `go.sum`.

## Desktop Commander duplicate mode

The duplicate path intentionally consumes the upstream MIT-licensed project:

- DesktopCommanderMCP
- https://github.com/wonderwhy-er/DesktopCommanderMCP
- exact admitted commit: `550a0b3e31da18b7cf25e87ed840e3d953b6da42`
- copyright: Eduard Ruzga and Desktop Commander Contributors
- license: MIT

The upstream source is pinned as `upstream/DesktopCommanderMCP`. Its own LICENSE and
copyright notices remain authoritative for that source. Distribution of the duplicate
path must preserve the upstream MIT notice.

The bounded native WorkBridge implementation was also informed by external open-source
projects, including Filamind, but that code path does not copy Filamind implementation
source.
