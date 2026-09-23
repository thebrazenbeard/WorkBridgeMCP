$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$installer = Join-Path $PSScriptRoot 'Install-WorkBridgeLappy.ps1'
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($installer, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) { throw 'Lappy installer has PowerShell parser errors.' }
$installerText = Get-Content -LiteralPath $installer -Raw
foreach ($required in @(
    '$process = Start-Process -FilePath "{0}"',
    '-RedirectStandardOutput "{2}\stdout.log"',
    '-RedirectStandardError "{2}\stderr.log"',
    'exit $process.ExitCode'
)) {
    if (-not $installerText.Contains($required)) {
        throw "Installer runner is missing native-process guard fragment: $required"
    }
}
if ($installerText.Contains('    & "{0}" --config "{1}" --transport http')) {
    throw 'Installer runner regressed to direct native invocation under ErrorActionPreference=Stop.'
}

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

Write-Host 'Lappy installer listener ownership guards: PASS'
