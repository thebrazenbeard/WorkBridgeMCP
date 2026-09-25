$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$installer = Join-Path $PSScriptRoot 'Install-WorkBridgeLappy.ps1'
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($installer, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) { throw 'Lappy installer has PowerShell parser errors.' }
foreach ($name in @('Assert-PortUnoccupied', 'Assert-ListenerOwnedByBinary')) {
    $definition = $ast.Find({
        param($node)
        $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $name
    }, $true)
    if ($null -eq $definition) { throw "Installer guard $name is missing." }
    Invoke-Expression $definition.Extent.Text
}

$script:TestListeners = @()
$script:TestExecutable = ''
function Get-NetTCPConnection {
    param($State, $LocalPort, $ErrorAction)
    return $script:TestListeners
}
function Get-CimInstance {
    param($ClassName, $Filter, $ErrorAction)
    return [pscustomobject]@{ ExecutablePath = $script:TestExecutable }
}
function Start-Sleep { param($Milliseconds) }
function Assert-Throws {
    param([scriptblock]$Action, [string]$Label)
    $threw = $false
    try { & $Action } catch { $threw = $true }
    if (-not $threw) { throw "$Label was accepted." }
}

$binary = 'C:\Program Files\WorkBridgeMCP\workbridge-mcp.exe'
Assert-PortUnoccupied -LocalPort 8765 -TimeoutSeconds 0
$script:TestListeners = @([pscustomobject]@{ LocalAddress = '127.0.0.1'; OwningProcess = 1234 })
Assert-Throws { Assert-PortUnoccupied -LocalPort 8765 -TimeoutSeconds 0 } 'occupied port'
$script:TestExecutable = 'C:\Windows\System32\other.exe'
Assert-Throws { Assert-ListenerOwnedByBinary -LocalPort 8765 -BinaryPath $binary } 'foreign listener'
$script:TestExecutable = $binary
Assert-ListenerOwnedByBinary -LocalPort 8765 -BinaryPath $binary
$script:TestListeners = @([pscustomobject]@{ LocalAddress = '0.0.0.0'; OwningProcess = 1234 })
Assert-Throws { Assert-ListenerOwnedByBinary -LocalPort 8765 -BinaryPath $binary } 'non-loopback listener'


$installerText = [IO.File]::ReadAllText($installer)
if (-not $installerText.Contains('$process = Start-Process -FilePath')) {
    throw 'Installed runner does not isolate native stderr from PowerShell error handling.'
}
if (-not $installerText.Contains('-RedirectStandardOutput') -or -not $installerText.Contains('-RedirectStandardError')) {
    throw 'Installed runner does not redirect native stdout/stderr through Start-Process.'
}
if ($installerText.Contains('& "{0}" --config "{1}" --transport http')) {
    throw 'Installed runner still directly invokes WorkBridge through the PowerShell native pipeline.'
}

if (-not $installerText.Contains('read_roots = @($roots)')) {
    throw 'Lappy installer no longer preserves VeraPort roots as WorkBridge read roots.'
}
if ($installerText.Contains('write_roots = @($roots)')) {
    throw 'Lappy installer still promotes allowed roots into write authority.'
}
$emptyWriteRootAssignments = [regex]::Matches(
    $installerText,
    '(?m)^\s*write_roots\s*=\s*@\(\)\s*$'
).Count
if ($emptyWriteRootAssignments -lt 2) {
    throw 'Lappy installer must keep both installed config and qualification receipt write_roots empty.'
}
if (-not $installerText.Contains('process_enabled = $false')) {
    throw 'Lappy qualification receipt does not prove process execution remains disabled.'
}
if (-not $installerText.Contains('unauthenticated_health = "DENIED_401"')) {
    throw 'Lappy qualification receipt does not record unauthenticated health denial.'
}
if (-not $installerText.Contains('existing VeraPort allowed_roots establish read-only location admission only')) {
    throw 'Lappy qualification receipt does not bind roots to read-only location admission.'
}

Write-Host 'Lappy installer listener and least-authority guards: PASS'
