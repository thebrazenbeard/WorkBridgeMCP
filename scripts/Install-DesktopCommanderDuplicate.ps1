param(
    [string]$InstallRoot = "C:\ProgramData\WorkBridgeMCP\DesktopCommanderMCP",
    [string]$NodeExe = "",
    [string]$NpmExe = "",
    [switch]$RunTests
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$UpstreamRepository = "https://github.com/wonderwhy-er/DesktopCommanderMCP.git"
$UpstreamCommit = "550a0b3e31da18b7cf25e87ed840e3d953b6da42"
$ExpectedVersion = "0.2.51"
$UpstreamArchive = "https://github.com/wonderwhy-er/DesktopCommanderMCP/archive/$UpstreamCommit.zip"

function Require-Command([string]$Name) {
    $cmd = Get-Command $Name -ErrorAction Stop
    return $cmd.Source
}

$gitCommand = Get-Command "git" -ErrorAction SilentlyContinue
$git = if ($gitCommand) { $gitCommand.Source } else { $null }

if ([string]::IsNullOrWhiteSpace($NodeExe)) {
    $NodeExe = Require-Command "node"
}
$NodeExe = (Resolve-Path $NodeExe).Path

if ([string]::IsNullOrWhiteSpace($NpmExe)) {
    $npmCommand = Get-Command "npm" -ErrorAction SilentlyContinue
    if ($npmCommand) {
        $NpmExe = $npmCommand.Source
    }
    else {
        $nodeAdjacentNpm = Join-Path (Split-Path -Parent $NodeExe) "npm.cmd"
        if (Test-Path -LiteralPath $nodeAdjacentNpm -PathType Leaf) {
            $NpmExe = $nodeAdjacentNpm
        }
        else {
            throw "npm was not found on PATH or next to NodeExe."
        }
    }
}
$NpmExe = (Resolve-Path $NpmExe).Path

$parent = Split-Path -Parent $InstallRoot
New-Item -ItemType Directory -Force -Path $parent | Out-Null
$staging = Join-Path $parent ("DesktopCommanderMCP.staging." + [Guid]::NewGuid().ToString("N"))
$backup = $null
$archiveStage = $null

try {
    if ($git) {
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
        }
        finally {
            Pop-Location
        }
    }
    else {
        $archiveStage = Join-Path $env:TEMP ("DesktopCommanderMCP.archive." + [Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Force -Path $archiveStage | Out-Null
        $archive = Join-Path $archiveStage "source.zip"
        Invoke-WebRequest -UseBasicParsing -Uri $UpstreamArchive -OutFile $archive
        $expanded = Join-Path $archiveStage "expanded"
        Expand-Archive -LiteralPath $archive -DestinationPath $expanded -Force
        $dirs = @(Get-ChildItem -LiteralPath $expanded -Directory)
        if ($dirs.Count -ne 1) {
            throw "unexpected DesktopCommander archive layout"
        }
        Move-Item -LiteralPath $dirs[0].FullName -Destination $staging
    }

    Push-Location $staging
    $originalPath = $env:PATH
    try {
        $nodeDir = Split-Path -Parent $NodeExe
        $pathParts = @($nodeDir)
        if (-not [string]::IsNullOrWhiteSpace($originalPath)) {
            $pathParts += $originalPath
        }
        $env:PATH = ($pathParts -join [IO.Path]::PathSeparator)

        $package = Get-Content -Raw -Encoding UTF8 "package.json" | ConvertFrom-Json
        if ($package.name -ne "@wonderwhy-er/desktop-commander") {
            throw "unexpected package name: $($package.name)"
        }
        if ($package.version -ne $ExpectedVersion) {
            throw "unexpected package version: $($package.version)"
        }

        & $NpmExe ci --ignore-scripts
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }

        & $NpmExe rebuild "@vscode/ripgrep"
        if ($LASTEXITCODE -ne 0) {
            throw "Desktop Commander ripgrep dependency rebuild failed"
        }

        & $NpmExe run build
        if ($LASTEXITCODE -ne 0) { throw "npm run build failed" }

        & $NodeExe "dist\npm-scripts\verify-ripgrep.js"
        if ($LASTEXITCODE -ne 0) {
            throw "Desktop Commander ripgrep verification failed"
        }

        if ($RunTests) {
            & $NpmExe test
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
        $env:PATH = $originalPath
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
    if ($archiveStage -and (Test-Path -LiteralPath $archiveStage)) {
        Remove-Item -LiteralPath $archiveStage -Recurse -Force -ErrorAction SilentlyContinue
    }
    if ($backup -and (Test-Path -LiteralPath $backup) -and -not (Test-Path -LiteralPath $InstallRoot)) {
        Move-Item -LiteralPath $backup -Destination $InstallRoot
    }
    throw
}
finally {
    if ($archiveStage -and (Test-Path -LiteralPath $archiveStage)) {
        Remove-Item -LiteralPath $archiveStage -Recurse -Force -ErrorAction SilentlyContinue
    }
}
