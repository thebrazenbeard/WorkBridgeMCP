[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$Binary,
    [string]$ProtocolVersion = "2026-07-28",
    [int]$TimeoutSeconds = 10
)

$ErrorActionPreference = "Stop"
$binaryPath = (Resolve-Path -LiteralPath $Binary).Path

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $binaryPath
$psi.UseShellExecute = $false
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.CreateNoWindow = $true

$process = New-Object System.Diagnostics.Process
$process.StartInfo = $psi

try {
    if (-not $process.Start()) {
        throw "Failed to start WorkBridgeMCP binary"
    }

    $initialize = @{
        jsonrpc = "2.0"
        id = 1
        method = "initialize"
        params = @{
            protocolVersion = $ProtocolVersion
            capabilities = @{}
            clientInfo = @{
                name = "workbridge-blackbox-smoke"
                version = "1.0"
            }
        }
    } | ConvertTo-Json -Compress -Depth 8

    $process.StandardInput.WriteLine($initialize)
    $process.StandardInput.Flush()

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    $response = $null

    while ([DateTime]::UtcNow -lt $deadline -and -not $process.HasExited) {
        if ($process.StandardOutput.Peek() -ge 0) {
            $line = $process.StandardOutput.ReadLine()
            if (-not [string]::IsNullOrWhiteSpace($line)) {
                $candidate = $line | ConvertFrom-Json
                if ($candidate.id -eq 1) {
                    $response = $candidate
                    break
                }
            }
        } else {
            Start-Sleep -Milliseconds 50
        }
    }

    if ($null -eq $response) {
        throw "No initialize response received within timeout"
    }
    if ($null -ne $response.error) {
        throw ("Initialize returned JSON-RPC error: " + ($response.error | ConvertTo-Json -Compress))
    }
    if ($null -eq $response.result.protocolVersion) {
        throw "Initialize response omitted protocolVersion"
    }

    $initialized = @{
        jsonrpc = "2.0"
        method = "notifications/initialized"
        params = @{}
    } | ConvertTo-Json -Compress -Depth 4
    $process.StandardInput.WriteLine($initialized)

    $toolsList = @{
        jsonrpc = "2.0"
        id = 2
        method = "tools/list"
        params = @{}
    } | ConvertTo-Json -Compress -Depth 4
    $process.StandardInput.WriteLine($toolsList)
    $process.StandardInput.Flush()

    Write-Output ([ordered]@{
        initialize = "PASS"
        negotiatedProtocolVersion = $response.result.protocolVersion
        serverInfo = $response.result.serverInfo
    } | ConvertTo-Json -Depth 6)
}
finally {
    if ($process -and -not $process.HasExited) {
        try { $process.Kill($true) } catch { }
    }
    if ($process) { $process.Dispose() }
}
