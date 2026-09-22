[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$Binary,
    [Parameter(Mandatory=$true)][string]$OutputDirectory,
    [Parameter(Mandatory=$true)][ValidatePattern('^[0-9a-fA-F]{40,64}

$ErrorActionPreference = "Stop"

$binaryPath = (Resolve-Path -LiteralPath $Binary).Path
if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
    throw "Binary does not exist: $Binary"
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$readme = Join-Path $repoRoot "README.md"
$thirdParty = Join-Path $repoRoot "THIRD_PARTY_NOTICES.md"
$template = Join-Path $repoRoot "packaging\windows\WorkBridgeMCP.xml.template"
$license = Join-Path $repoRoot "LICENSE"

foreach ($required in @($readme, $thirdParty, $template)) {
    if (-not (Test-Path -LiteralPath $required -PathType Leaf)) {
        throw "Required release input missing: $required"
    }
}

$goVersion = & go version -m $binaryPath 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "Unable to inspect Go build metadata for $binaryPath"
}
$revision = $null
$modified = $null
foreach ($line in $goVersion) {
    if ($line -match 'vcs\.revision=(?<revision>[0-9a-fA-F]+)') {
        $revision = $Matches.revision.ToLowerInvariant()
    }
    if ($line -match 'vcs\.modified=(?<modified>true|false)') {
        $modified = $Matches.modified
    }
}
if ([string]::IsNullOrWhiteSpace($revision)) {
    throw "Binary is missing embedded vcs.revision build metadata"
}
if ($revision -ne $SourceCommit.ToLowerInvariant()) {
    throw "Binary vcs.revision $revision does not match SourceCommit $SourceCommit"
}
if ($modified -ne "false") {
    throw "Binary was built from a modified source tree"
}

$resolvedOutput = [System.IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force -Path $resolvedOutput | Out-Null

$stage = Join-Path $resolvedOutput ("workbridge-mcp-" + $Version)
if (Test-Path -LiteralPath $stage) {
    Remove-Item -LiteralPath $stage -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $stage | Out-Null

Copy-Item -LiteralPath $binaryPath -Destination (Join-Path $stage "workbridge-mcp.exe")
Copy-Item -LiteralPath $readme -Destination (Join-Path $stage "README.md")
Copy-Item -LiteralPath $thirdParty -Destination (Join-Path $stage "THIRD_PARTY_NOTICES.md")
Copy-Item -LiteralPath $template -Destination (Join-Path $stage "WorkBridgeMCP.xml.template")
if (Test-Path -LiteralPath $license -PathType Leaf) {
    Copy-Item -LiteralPath $license -Destination (Join-Path $stage "LICENSE")
}

$manifest = [ordered]@{
    schema = "WORKBRIDGE_RELEASE_MANIFEST_V1"
    version = $Version
    source_commit = $revision
    binary_vcs_modified = $false
    files = @()
}

Get-ChildItem -LiteralPath $stage -File | Sort-Object Name | ForEach-Object {
    $manifest.files += [ordered]@{
        name = $_.Name
        sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        bytes = $_.Length
    }
}

$manifestPath = Join-Path $stage "release-manifest.json"
$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

$archive = Join-Path $resolvedOutput ("workbridge-mcp-" + $Version + "-windows-x64.zip")
if (Test-Path -LiteralPath $archive) {
    Remove-Item -LiteralPath $archive -Force
}
Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $archive -CompressionLevel Optimal

Write-Output $archive
)][string]$SourceCommit,
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"

$binaryPath = (Resolve-Path -LiteralPath $Binary).Path
if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
    throw "Binary does not exist: $Binary"
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$readme = Join-Path $repoRoot "README.md"
$thirdParty = Join-Path $repoRoot "THIRD_PARTY_NOTICES.md"
$template = Join-Path $repoRoot "packaging\windows\WorkBridgeMCP.xml.template"
$license = Join-Path $repoRoot "LICENSE"

foreach ($required in @($readme, $thirdParty, $template)) {
    if (-not (Test-Path -LiteralPath $required -PathType Leaf)) {
        throw "Required release input missing: $required"
    }
}

$resolvedOutput = [System.IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force -Path $resolvedOutput | Out-Null

$stage = Join-Path $resolvedOutput ("workbridge-mcp-" + $Version)
if (Test-Path -LiteralPath $stage) {
    Remove-Item -LiteralPath $stage -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $stage | Out-Null

Copy-Item -LiteralPath $binaryPath -Destination (Join-Path $stage "workbridge-mcp.exe")
Copy-Item -LiteralPath $readme -Destination (Join-Path $stage "README.md")
Copy-Item -LiteralPath $thirdParty -Destination (Join-Path $stage "THIRD_PARTY_NOTICES.md")
Copy-Item -LiteralPath $template -Destination (Join-Path $stage "WorkBridgeMCP.xml.template")
if (Test-Path -LiteralPath $license -PathType Leaf) {
    Copy-Item -LiteralPath $license -Destination (Join-Path $stage "LICENSE")
}

$manifest = [ordered]@{
    schema = "WORKBRIDGE_RELEASE_MANIFEST_V1"
    version = $Version
    files = @()
}

Get-ChildItem -LiteralPath $stage -File | Sort-Object Name | ForEach-Object {
    $manifest.files += [ordered]@{
        name = $_.Name
        sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        bytes = $_.Length
    }
}

$manifestPath = Join-Path $stage "release-manifest.json"
$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

$archive = Join-Path $resolvedOutput ("workbridge-mcp-" + $Version + "-windows-x64.zip")
if (Test-Path -LiteralPath $archive) {
    Remove-Item -LiteralPath $archive -Force
}
Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $archive -CompressionLevel Optimal

Write-Output $archive
