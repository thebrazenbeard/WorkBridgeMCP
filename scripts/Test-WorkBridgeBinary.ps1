[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$Binary,
    [string]$ProtocolVersion = "2026-07-28",
    [int]$TimeoutSeconds = 10
)

$ErrorActionPreference = "Stop"
$binaryPath = (Resolve-Path -LiteralPath $Binary).Path
$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("workbridge-smoke-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null
$configPath = Join-Path $tempRoot "config.json"

$config = [ordered]@{
    schema = "WORKBRIDGE_CONFIG_V1"
    read_roots = @($tempRoot)
    write_roots = @()
    limits = @{}
    process = @{
        enabled = $false
        allowed_executables = @()
        working_roots = @()
    }
    http = @{
        listen = "127.0.0.1:8765"
        path = "/mcp"
        bearer_token_env = "WORKBRIDGE_HTTP_TOKEN"
    }
}
$config | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $configPath -Encoding UTF8

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $binaryPath
$psi.UseShellExecute = $false
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.CreateNoWindow = $true
$psi.Environment["WORKBRIDGE_CONFIG"] = $configPath

$process = New-Object System.Diagnostics.Process
$process.StartInfo = $psi

function Read-JsonRpcResponse {
    param(
        [Parameter(Mandatory=$true)]$Process,
        [Parameter(Mandatory=$true)][int]$Id,
        [Parameter(Mandatory=$true)][DateTime]$Deadline
    )
    while ([DateTime]::UtcNow -lt $Deadline) {
        if ($Process.HasExited) {
            $stderr = $Process.StandardError.ReadToEnd()
            throw "WorkBridge exited before JSON-RPC response id=$Id. exit=$($Process.ExitCode) stderr=$stderr"
        }
        if ($Process.StandardOutput.Peek() -ge 0) {
            $line = $Process.StandardOutput.ReadLine()
            if ([string]::IsNullOrWhiteSpace($line)) {
                continue
            }
            $candidate = $line | ConvertFrom-Json
            if ($candidate.id -eq $Id) {
                return $candidate
            }
            continue
        }
        Start-Sleep -Milliseconds 25
    }
    throw "No JSON-RPC response id=$Id received within timeout"
}

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
    $response = Read-JsonRpcResponse -Process $process -Id 1 -Deadline $deadline
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

    $toolsResponse = Read-JsonRpcResponse -Process $process -Id 2 -Deadline $deadline
    if ($null -ne $toolsResponse.error) {
        throw ("tools/list returned JSON-RPC error: " + ($toolsResponse.error | ConvertTo-Json -Compress))
    }

    $toolNames = @($toolsResponse.result.tools | ForEach-Object { $_.name })
    $required = @("workbridge_health", "workspace_list", "workspace_stat", "workspace_read_text")
    foreach ($name in $required) {
        if ($toolNames -notcontains $name) {
            throw "tools/list omitted required read-only tool: $name"
        }
    }
    foreach ($forbidden in @("workspace_write_text", "workspace_mkdir", "process_run")) {
        if ($toolNames -contains $forbidden) {
            throw "least-authority smoke config unexpectedly exposed tool: $forbidden"
        }
    }

    Write-Output ([ordered]@{
        initialize = "PASS"
        toolsList = "PASS"
        negotiatedProtocolVersion = $response.result.protocolVersion
        serverInfo = $response.result.serverInfo
        tools = $toolNames
    } | ConvertTo-Json -Depth 8)
}
finally {
    if ($process -and -not $process.HasExited) {
        try { $process.Kill($true) } catch { }
    }
    if ($process) { $process.Dispose() }
    if (Test-Path -LiteralPath $tempRoot) {
        Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
