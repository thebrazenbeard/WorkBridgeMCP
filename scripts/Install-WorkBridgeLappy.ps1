[CmdletBinding()]
param(
    [string]$SourceCommit = "0b80fe050d8f03da54ee14123737af26740c1605",
    [string]$GoVersion = "go1.25.12",
    [string]$InstallRoot = "C:\Program Files\WorkBridgeMCP",
    [string]$DataRoot = "C:\ProgramData\WorkBridgeMCP",
    [string]$TaskName = "WorkBridgeMCP",
    [int]$Port = 8765
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Assert-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "Run this installer from an elevated Administrator PowerShell window."
    }
}

function Get-FileSnapshot {
    param([string[]]$Roots)
    $snapshot = [ordered]@{}
    foreach ($root in $Roots) {
        if (-not (Test-Path -LiteralPath $root -PathType Container)) { continue }
        Get-ChildItem -LiteralPath $root -File -Recurse | Sort-Object FullName | ForEach-Object {
            $snapshot[$_.FullName.ToLowerInvariant()] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    }
    return $snapshot
}

function Assert-SnapshotEqual {
    param($Before, $After, [string]$Label)
    if (($Before | ConvertTo-Json -Compress -Depth 8) -ne ($After | ConvertTo-Json -Compress -Depth 8)) {
        throw "$Label changed during WorkBridge installation."
    }
}

function Protect-PrivateDirectory {
    param([string]$Path)
    & icacls.exe $Path /inheritance:r /grant:r "SYSTEM:(OI)(CI)F" "BUILTIN\Administrators:(OI)(CI)F" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Failed to protect private directory: $Path" }
}

function Get-PortableGo {
    param([string]$Version, [string]$TempRoot)
    $fileName = "$Version.windows-amd64.zip"
    $metadata = Invoke-RestMethod -Uri "https://go.dev/dl/?mode=json&include=all"
    $release = @($metadata | Where-Object { $_.version -eq $Version }) | Select-Object -First 1
    if ($null -eq $release) { throw "Official Go metadata does not contain $Version" }
    $file = @($release.files | Where-Object { $_.filename -eq $fileName }) | Select-Object -First 1
    if ($null -eq $file -or [string]::IsNullOrWhiteSpace([string]$file.sha256)) {
        throw "Official Go metadata does not contain SHA-256 for $fileName"
    }
    $archive = Join-Path $TempRoot $fileName
    Invoke-WebRequest -UseBasicParsing -Uri ("https://go.dev/dl/" + $fileName) -OutFile $archive
    $actual = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    $expected = ([string]$file.sha256).ToLowerInvariant()
    if ($actual -ne $expected) { throw "Go toolchain SHA-256 mismatch" }
    $extract = Join-Path $TempRoot "go-toolchain"
    Expand-Archive -LiteralPath $archive -DestinationPath $extract
    $go = Join-Path $extract "go\bin\go.exe"
    if (-not (Test-Path -LiteralPath $go -PathType Leaf)) { throw "Portable Go executable missing after extraction" }
    return $go
}

function Get-ExactSource {
    param([string]$Commit, [string]$TempRoot)
    if ($Commit -notmatch '^[0-9a-f]{40}$') { throw "SourceCommit must be full lowercase 40-hex" }
    $archive = Join-Path $TempRoot "workbridge-source.zip"
    Invoke-WebRequest -UseBasicParsing -Uri ("https://github.com/thebrazenbeard/WorkBridgeMCP/archive/" + $Commit + ".zip") -OutFile $archive
    $extract = Join-Path $TempRoot "source"
    Expand-Archive -LiteralPath $archive -DestinationPath $extract
    $dirs = @(Get-ChildItem -LiteralPath $extract -Directory)
    if ($dirs.Count -ne 1) { throw "Unexpected WorkBridge source archive layout" }
    if (-not (Test-Path -LiteralPath (Join-Path $dirs[0].FullName "go.mod") -PathType Leaf)) {
        throw "Downloaded WorkBridge source does not contain go.mod"
    }
    return $dirs[0].FullName
}

Assert-Administrator

$veraPortConfigPath = "C:\ProgramData\VeraMesh\veraport.json"
if (-not (Test-Path -LiteralPath $veraPortConfigPath -PathType Leaf)) { throw "Existing VeraPort config not found" }
$veraPortConfig = Get-Content -LiteralPath $veraPortConfigPath -Raw | ConvertFrom-Json
if ($veraPortConfig.allow_process_exec -eq $true) {
    throw "Existing VeraPort unexpectedly has process execution enabled; refusing to change authority assumptions."
}
$roots = @($veraPortConfig.allowed_roots)
if ($roots.Count -lt 1) { throw "Existing VeraPort config has no allowed roots to preserve." }
foreach ($root in $roots) {
    if (-not [IO.Path]::IsPathRooted([string]$root) -or -not (Test-Path -LiteralPath $root -PathType Container)) {
        throw "Existing VeraPort allowed root is unavailable: $root"
    }
}

$veraPortServiceBefore = Get-CimInstance Win32_Service -Filter "Name='VeraPortAgent'" -ErrorAction Stop
$veraPortIdentityRoots = @("C:\ProgramData\VeraMesh\identity","C:\ProgramData\VeraMesh\controller")
$identityBefore = Get-FileSnapshot -Roots $veraPortIdentityRoots

$runnerPath = Join-Path $DataRoot "run-workbridge.ps1"
$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($null -ne $existingTask) {
    $actions = @($existingTask.Actions)
    if ($actions.Count -ne 1 -or $actions[0].Execute -notmatch '(?i)powershell(\.exe)?$' -or $actions[0].Arguments -notmatch [regex]::Escape($runnerPath)) {
        throw "A scheduled task named $TaskName already exists but is not the managed WorkBridge task."
    }
}

$listenerBefore = @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue)
if ($listenerBefore.Count -gt 0 -and $null -eq $existingTask) {
    throw "Port $Port is already listening before WorkBridge installation."
}

$tempRoot = Join-Path $env:TEMP ("workbridge-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null

try {
    $go = Get-PortableGo -Version $GoVersion -TempRoot $tempRoot
    $source = Get-ExactSource -Commit $SourceCommit -TempRoot $tempRoot
    $built = Join-Path $tempRoot "workbridge-mcp.exe"

    Push-Location $source
    try {
        $env:CGO_ENABLED = "0"
        & $go mod verify
        if ($LASTEXITCODE -ne 0) { throw "go mod verify failed" }
        & $go test ./...
        if ($LASTEXITCODE -ne 0) { throw "go test failed" }
        & $go build -trimpath -o $built ./cmd/workbridge-mcp
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $built -PathType Leaf)) { throw "WorkBridge build failed" }
        & (Join-Path $source "scripts\Test-WorkBridgeBinary.ps1") -Binary $built
        if ($LASTEXITCODE -ne 0) { throw "WorkBridge black-box stdio smoke failed" }
    }
    finally { Pop-Location }

    if ($null -ne $existingTask) {
        Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500
    }

    New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null
    New-Item -ItemType Directory -Force -Path $DataRoot | Out-Null
    Protect-PrivateDirectory -Path $DataRoot

    $binaryPath = Join-Path $InstallRoot "workbridge-mcp.exe"
    Copy-Item -LiteralPath $built -Destination $binaryPath -Force

    $tokenPath = Join-Path $DataRoot "http-token.txt"
    if (-not (Test-Path -LiteralPath $tokenPath -PathType Leaf)) {
        $bytes = New-Object byte[] 48
        $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
        try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
        $token = [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+','-').Replace('/','_')
        [IO.File]::WriteAllText($tokenPath, $token, (New-Object Text.UTF8Encoding($false)))
    }
    $token = [IO.File]::ReadAllText($tokenPath).Trim()
    if ($token.Length -lt 32 -or $token -match '\s') { throw "Stored WorkBridge HTTP token is invalid" }

    $configPath = Join-Path $DataRoot "config.json"
    $config = [ordered]@{
        schema = "WORKBRIDGE_CONFIG_V1"
        read_roots = @($roots)
        write_roots = @($roots)
        limits = [ordered]@{
            max_read_bytes = 1048576
            max_write_bytes = 1048576
            max_directory_entries = 500
        }
        process = [ordered]@{
            enabled = $false
            allowed_executables = @()
            working_roots = @()
            max_runtime_seconds = 60
            max_output_bytes = 1048576
            max_args = 64
        }
        http = [ordered]@{
            listen = "127.0.0.1:$Port"
            path = "/mcp"
            bearer_token_env = "WORKBRIDGE_HTTP_TOKEN"
        }
    }
    [IO.File]::WriteAllText($configPath,(($config | ConvertTo-Json -Depth 10)+[Environment]::NewLine),(New-Object Text.UTF8Encoding($false)))

    $runnerLines = @(
        '$ErrorActionPreference = "Stop"',
        ('$token = [IO.File]::ReadAllText("{0}").Trim()' -f $tokenPath),
        'if ($token.Length -lt 32 -or $token -match ''\s'') { throw "Invalid WorkBridge token" }',
        '$env:WORKBRIDGE_HTTP_TOKEN = $token',
        ('& "{0}" --config "{1}" --transport http 1>>"{2}\stdout.log" 2>>"{2}\stderr.log"' -f $binaryPath,$configPath,$DataRoot),
        'exit $LASTEXITCODE'
    )
    [IO.File]::WriteAllText($runnerPath,($runnerLines -join [Environment]::NewLine),(New-Object Text.UTF8Encoding($false)))
    Protect-PrivateDirectory -Path $DataRoot

    $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument ('-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + $runnerPath + '"')
    $trigger = New-ScheduledTaskTrigger -AtStartup
    $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    $settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -RestartCount 20 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit ([TimeSpan]::Zero)
    $taskParams = @{
        TaskName = $TaskName
        Action = $action
        Trigger = $trigger
        Principal = $principal
        Settings = $settings
        Description = "WorkBridgeMCP loopback-only workstation bridge; process execution disabled."
        Force = $true
    }
    Register-ScheduledTask @taskParams | Out-Null
    Start-ScheduledTask -TaskName $TaskName

    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    $health = $null
    do {
        Start-Sleep -Milliseconds 300
        try {
            $health = Invoke-RestMethod -Uri "http://127.0.0.1:$Port/mcp/healthz" -Headers @{ Authorization = "Bearer $token" } -TimeoutSec 2
        } catch { $health = $null }
    } while ($null -eq $health -and [DateTime]::UtcNow -lt $deadline)

    if ($null -eq $health -or $health.status -ne "ok") {
        $stderrPath = Join-Path $DataRoot "stderr.log"
        $tail = if (Test-Path -LiteralPath $stderrPath) { (Get-Content -LiteralPath $stderrPath -Tail 30) -join [Environment]::NewLine } else { "<no stderr log>" }
        throw "WorkBridge failed local health qualification. stderr tail: $tail"
    }

    $listeners = @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction Stop)
    $badListeners = @($listeners | Where-Object { $_.LocalAddress -notin @("127.0.0.1","::1") })
    if ($badListeners.Count -gt 0) { throw "WorkBridge has a non-loopback listener." }
    if (@($listeners | Where-Object { $_.LocalAddress -eq "127.0.0.1" }).Count -lt 1) {
        throw "Expected WorkBridge loopback listener 127.0.0.1:$Port is absent."
    }

    $installedTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction Stop
    $taskInfo = Get-ScheduledTaskInfo -TaskName $TaskName -ErrorAction Stop
    $identityAfter = Get-FileSnapshot -Roots $veraPortIdentityRoots
    Assert-SnapshotEqual -Before $identityBefore -After $identityAfter -Label "VeraPort identity/controller material"

    $veraPortServiceAfter = Get-CimInstance Win32_Service -Filter "Name='VeraPortAgent'" -ErrorAction Stop
    if ($veraPortServiceAfter.State -ne $veraPortServiceBefore.State -or
        $veraPortServiceAfter.StartMode -ne $veraPortServiceBefore.StartMode -or
        $veraPortServiceAfter.StartName -ne $veraPortServiceBefore.StartName) {
        throw "VeraPort service state/start mode/account changed during WorkBridge installation."
    }

    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        $tokenHash = [BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($token))).Replace("-","").ToLowerInvariant()
    } finally { $sha.Dispose() }

    [ordered]@{
        schema = "WORKBRIDGE_LAPPY_INSTALL_QUALIFICATION_V1"
        source_commit = $SourceCommit
        workbridge_version = $health.version
        binary = $binaryPath
        binary_sha256 = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
        config = $configPath
        task = [ordered]@{
            name = $TaskName
            state = [string]$installedTask.State
            last_task_result = $taskInfo.LastTaskResult
            run_as = "SYSTEM"
            startup_trigger = $true
        }
        http = [ordered]@{
            listen = "127.0.0.1:$Port"
            path = "/mcp"
            authenticated_health = "PASS"
            token_file = $tokenPath
            token_sha256 = $tokenHash
        }
        authority = [ordered]@{
            read_roots = @($roots)
            write_roots = @($roots)
            process_enabled = $false
            authority_source = "existing VeraPort allowed_roots; no broader filesystem roots"
        }
        preservation = [ordered]@{
            veraport_identity_hashes_unchanged = $true
            veraport_service_state_unchanged = $true
            veraport_process_exec_remained_disabled = $true
        }
        network = [ordered]@{
            loopback_only = $true
            tailscale_changed = $false
            firewall_changed = $false
            public_exposure_changed = $false
        }
    } | ConvertTo-Json -Depth 12
}
finally {
    if (Test-Path -LiteralPath $tempRoot) {
        Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
