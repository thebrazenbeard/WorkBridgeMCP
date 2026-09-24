param(
    [string]$InstallRoot = "C:\ProgramData\WorkBridgeMCP\DesktopCommanderMCP",
    [string]$NodeExe = "",
    [switch]$RunTests
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$UpstreamRepository = "https://github.com/wonderwhy-er/DesktopCommanderMCP.git"
$UpstreamCommit = "550a0b3e31da18b7cf25e87ed840e3d953b6da42"
$ExpectedVersion = "0.2.51"

function Require-Command([string]$Name) {
    $cmd = Get-Command $Name -ErrorAction Stop
    return $cmd.Source
}

$git = Require-Command "git"
$npm = Require-Command "npm"
if ([string]::IsNullOrWhiteSpace($NodeExe)) {
    $NodeExe = Require-Command "node"
}
$NodeExe = (Resolve-Path $NodeExe).Path

$parent = Split-Path -Parent $InstallRoot
New-Item -ItemType Directory -Force -Path $parent | Out-Null
$staging = Join-Path $parent ("DesktopCommanderMCP.staging." + [Guid]::NewGuid().ToString("N"))
$backup = $null

try {
    & $git clone --no-tags $UpstreamRepository $staging
    if ($LASTEXITCODE -ne 0) { throw "git clone failed" }

    Push-Location $staging
    try {
        & $git checkout --detach $UpstreamCommit
        if ($LASTEXITCODE -ne 0) { throw "git checkout failed" }
        $head = (& $git rev-parse HEAD).Trim()
        if ($head -ne $UpstreamCommit) {
            throw "source head mismatch: expected $UpstreamCommit got $head"
        }

        $package = Get-Content -Raw -Encoding UTF8 "package.json" | ConvertFrom-Json
        if ($package.name -ne "@wonderwhy-er/desktop-commander") {
            throw "unexpected package name: $($package.name)"
        }
        if ($package.version -ne $ExpectedVersion) {
            throw "unexpected package version: $($package.version)"
        }

        & $npm ci --ignore-scripts
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }

        & $npm run build
        if ($LASTEXITCODE -ne 0) { throw "npm run build failed" }

        if ($RunTests) {
            & $npm test
            if ($LASTEXITCODE -ne 0) { throw "DesktopCommander upstream test suite failed" }
        }

        $entrypoint = Join-Path $staging "dist\index.js"
        if (-not (Test-Path -LiteralPath $entrypoint -PathType Leaf)) {
            throw "built DesktopCommander entrypoint missing: $entrypoint"
        }

        $runtimeDir = Join-Path $staging "workbridge-runtime"
        New-Item -ItemType Directory -Force -Path $runtimeDir | Out-Null
        $runtimeNode = Join-Path $runtimeDir "node.exe"
        Copy-Item -LiteralPath $NodeExe -Destination $runtimeNode -Force

        $nodeHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $runtimeNode).Hash.ToLowerInvariant()
        $entryHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $entrypoint).Hash.ToLowerInvariant()

        $manifest = [ordered]@{
            schema = "WORKBRIDGE_DESKTOP_COMMANDER_DUPLICATE_V1"
            upstream_repository = $UpstreamRepository
            upstream_commit = $UpstreamCommit
            upstream_version = $ExpectedVersion
            node_executable_relative = "workbridge-runtime\node.exe"
            node_sha256 = $nodeHash
            entrypoint_relative = "dist\index.js"
            entrypoint_sha256 = $entryHash
            mcp_args = @("dist\index.js", "--no-onboarding")
            unrestricted_command_string_shell = $true
        }
        $manifest | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 "workbridge-desktop-commander.manifest.json"
    }
    finally {
        Pop-Location
    }

    if (Test-Path -LiteralPath $InstallRoot) {
        $backup = "$InstallRoot.backup.$([DateTime]::UtcNow.ToString('yyyyMMddHHmmss'))"
        Move-Item -LiteralPath $InstallRoot -Destination $backup
    }
    Move-Item -LiteralPath $staging -Destination $InstallRoot

    if ($backup -and (Test-Path -LiteralPath $backup)) {
        Remove-Item -LiteralPath $backup -Recurse -Force
    }

    $installedManifest = Join-Path $InstallRoot "workbridge-desktop-commander.manifest.json"
    $result = Get-Content -Raw -Encoding UTF8 $installedManifest | ConvertFrom-Json
    [pscustomobject]@{
        status = "installed"
        install_root = $InstallRoot
        upstream_commit = $result.upstream_commit
        node_executable = (Join-Path $InstallRoot $result.node_executable_relative)
        entrypoint = (Join-Path $InstallRoot $result.entrypoint_relative)
        unrestricted_command_string_shell = $result.unrestricted_command_string_shell
    } | ConvertTo-Json -Depth 4
}
catch {
    if (Test-Path -LiteralPath $staging) {
        Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
    }
    if ($backup -and (Test-Path -LiteralPath $backup) -and -not (Test-Path -LiteralPath $InstallRoot)) {
        Move-Item -LiteralPath $backup -Destination $InstallRoot
    }
    throw
}
