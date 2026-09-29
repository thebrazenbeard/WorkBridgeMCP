[CmdletBinding()]
param(
    [string]$SourceCommit = "d9e8881ca6ffa17ea5b98a6af8c0ef2b2d171f15",
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

function Assert-PortUnoccupied {
    param([int]$LocalPort, [int]$TimeoutSeconds = 20)
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        $listeners = @(Get-NetTCPConnection -State Listen -LocalPort $LocalPort -ErrorAction SilentlyContinue)
        if ($listeners.Count -eq 0) { return }
        Start-Sleep -Milliseconds 300
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Port $LocalPort remained occupied after stopping the prior WorkBridge task."
}

function Assert-ListenerOwnedByBinary {
    param([int]$LocalPort, [string]$BinaryPath)
    $expected = [IO.Path]::GetFullPath($BinaryPath)
    $listeners = @(Get-NetTCPConnection -State Listen -LocalPort $LocalPort -ErrorAction Stop)
    if ($listeners.Count -eq 0) { throw "No WorkBridge listener found on port $LocalPort." }
    foreach ($listener in $listeners) {
        if ($listener.LocalAddress -notin @("127.0.0.1","::1")) {
            throw "WorkBridge has a non-loopback listener."
        }
        $ownerPid = [int]$listener.OwningProcess
        $owner = Get-CimInstance Win32_Process -Filter "ProcessId=$ownerPid" -ErrorAction Stop
        if ($null -eq $owner -or [string]::IsNullOrWhiteSpace([string]$owner.ExecutablePath) -or
            -not [string]::Equals([IO.Path]::GetFullPath([string]$owner.ExecutablePath), $expected, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Port $LocalPort is owned by a process other than the installed WorkBridge binary."
        }
    }
    if (@($listeners | Where-Object { $_.LocalAddress -eq "127.0.0.1" }).Count -lt 1) {
        throw "Expected WorkBridge loopback listener 127.0.0.1:$LocalPort is absent."
    }
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
$veraPortConfigBefore = (Get-FileHash -LiteralPath $veraPortConfigPath -Algorithm SHA256).Hash.ToLowerInvariant()
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

if ($SourceCommit -notmatch '^[0-9a-f]{40}$') { throw "SourceCommit must be full lowercase 40-hex" }

$cacheRoot = Join-Path (Join-Path $DataRoot "build-cache") $SourceCommit
$cacheBinaryPath = Join-Path $cacheRoot "workbridge-mcp.exe"
$cacheManifestPath = Join-Path $cacheRoot "qualification.json"
$tempRoot = Join-Path $env:TEMP ("workbridge-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null

$qualifiedBinary = $null
$buildCacheStatus = "MISS"

try {
    New-Item -ItemType Directory -Force -Path $DataRoot | Out-Null
    Protect-PrivateDirectory -Path $DataRoot

    if ((Test-Path -LiteralPath $cacheBinaryPath -PathType Leaf) -and (Test-Path -LiteralPath $cacheManifestPath -PathType Leaf)) {
        try {
            $cacheManifest = Get-Content -LiteralPath $cacheManifestPath -Raw | ConvertFrom-Json
            $cachedHash = (Get-FileHash -LiteralPath $cacheBinaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($cacheManifest.schema -eq "WORKBRIDGE_LOCAL_BUILD_CACHE_V1" -and
                $cacheManifest.source_commit -eq $SourceCommit -and
                $cacheManifest.go_version -eq $GoVersion -and
                $cacheManifest.http_smoke -eq "PASS" -and
                $cacheManifest.binary_sha256 -eq $cachedHash) {
                $qualifiedBinary = $cacheBinaryPath
                $buildCacheStatus = "HIT"
            }
        } catch {
            $qualifiedBinary = $null
            $buildCacheStatus = "MISS"
        }
    }

    if ($null -eq $qualifiedBinary) {
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
            & (Join-Path $source "scripts\Test-WorkBridgeHttpBinary.ps1") -Binary $built
            if ($LASTEXITCODE -ne 0) { throw "WorkBridge PowerShell 5.1 HTTP smoke failed" }
        }
        finally { Pop-Location }

        $builtHash = (Get-FileHash -LiteralPath $built -Algorithm SHA256).Hash.ToLowerInvariant()
        New-Item -ItemType Directory -Force -Path $cacheRoot | Out-Null
        Copy-Item -LiteralPath $built -Destination $cacheBinaryPath -Force
        $cacheManifest = [ordered]@{
            schema = "WORKBRIDGE_LOCAL_BUILD_CACHE_V1"
            source_commit = $SourceCommit
            go_version = $GoVersion
            binary_sha256 = $builtHash
            http_smoke = "PASS"
        }
        $cacheUtf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
        [IO.File]::WriteAllText($cacheManifestPath, (($cacheManifest | ConvertTo-Json -Depth 5) + [Environment]::NewLine), $cacheUtf8NoBom)
        Protect-PrivateDirectory -Path $DataRoot
        $qualifiedBinary = $cacheBinaryPath
        $buildCacheStatus = "MISS_BUILT_AND_CACHED"
    }

    if ($null -ne $existingTask) {
        Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        Assert-PortUnoccupied -LocalPort $Port
    }

    New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null
    New-Item -ItemType Directory -Force -Path $DataRoot | Out-Null
    Protect-PrivateDirectory -Path $DataRoot

    $binaryPath = Join-Path $InstallRoot "workbridge-mcp.exe"
    Copy-Item -LiteralPath $qualifiedBinary -Destination $binaryPath -Force

    $utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
    $tokenPath = Join-Path $DataRoot "http-token.txt"
    if (-not (Test-Path -LiteralPath $tokenPath -PathType Leaf)) {
        $bytes = New-Object byte[] 48
        $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
        try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
        $token = [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+','-').Replace('/','_')
        [IO.File]::WriteAllText($tokenPath, $token, $utf8NoBom)
    }
    $token = [IO.File]::ReadAllText($tokenPath).Trim()
    if ($token.Length -lt 32 -or $token -match '\s') { throw "Stored WorkBridge HTTP token is invalid" }

    $configPath = Join-Path $DataRoot "config.json"
    $config = [ordered]@{
        schema = "WORKBRIDGE_CONFIG_V1"
        read_roots = @($roots)
        write_roots = @()
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
    $configJson = (($config | ConvertTo-Json -Depth 10) + [Environment]::NewLine)
    [IO.File]::WriteAllText($configPath, $configJson, $utf8NoBom)
    $configBytes = [IO.File]::ReadAllBytes($configPath)
    if ($configBytes.Length -ge 3 -and $configBytes[0] -eq 0xEF -and $configBytes[1] -eq 0xBB -and $configBytes[2] -eq 0xBF) {
        throw "Installed WorkBridge config unexpectedly contains a UTF-8 BOM"
    }

    $runnerErrorPath = Join-Path $DataRoot "runner-error.log"
    $runnerLines = @(
        '$ErrorActionPreference = "Stop"',
        ('$runnerError = "{0}"' -f $runnerErrorPath),
        'try {',
        ('    $token = [IO.File]::ReadAllText("{0}").Trim()' -f $tokenPath),
        '    if ($token.Length -lt 32 -or $token -match ''\s'') { throw "Invalid WorkBridge token" }',
        '    $env:WORKBRIDGE_HTTP_TOKEN = $token',
        ('    $process = Start-Process -FilePath "{0}" -ArgumentList @("--config","{1}","--transport","http") -NoNewWindow -Wait -PassThru -RedirectStandardOutput "{2}\stdout.log" -RedirectStandardError "{2}\stderr.log"' -f $binaryPath,$configPath,$DataRoot),
        '    exit $process.ExitCode',
        '} catch {',
        '    $_ | Out-String | Set-Content -LiteralPath $runnerError -Encoding UTF8',
        '    exit 1',
        '}'
    )
    $runnerText = ($runnerLines -join [Environment]::NewLine) + [Environment]::NewLine
    [IO.File]::WriteAllText($runnerPath, $runnerText, $utf8NoBom)
    Remove-Item -LiteralPath $runnerErrorPath -Force -ErrorAction SilentlyContinue
    Protect-PrivateDirectory -Path $DataRoot

    $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument ('-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + $runnerPath + '"')
    $trigger = New-ScheduledTaskTrigger -AtStartup
    $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    $settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -RestartCount 20 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit ([TimeSpan]::Zero) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
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
        $runnerErrorPath = Join-Path $DataRoot "runner-error.log"
        $tail = if (Test-Path -LiteralPath $stderrPath) { (Get-Content -LiteralPath $stderrPath -Tail 30) -join [Environment]::NewLine } else { "<no stderr log>" }
        $runnerTail = if (Test-Path -LiteralPath $runnerErrorPath) { (Get-Content -LiteralPath $runnerErrorPath -Tail 30) -join [Environment]::NewLine } else { "<no runner error log>" }
        throw "WorkBridge failed local health qualification. stderr tail: $tail runner-error tail: $runnerTail"
    }

    $unauthorized = $false
    try {
        Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$Port/mcp/healthz" -TimeoutSec 2 | Out-Null
    } catch {
        if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq 401) {
            $unauthorized = $true
        } else {
            throw
        }
    }
    if (-not $unauthorized) {
        throw "WorkBridge health unexpectedly succeeded without bearer authentication."
    }

    Assert-ListenerOwnedByBinary -LocalPort $Port -BinaryPath $binaryPath

    $installedTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction Stop
    if ([string]$installedTask.State -ne "Running") {
        throw "WorkBridge scheduled task is not running after authenticated health qualification."
    }
    $taskInfo = Get-ScheduledTaskInfo -TaskName $TaskName -ErrorAction Stop
    $identityAfter = Get-FileSnapshot -Roots $veraPortIdentityRoots
    Assert-SnapshotEqual -Before $identityBefore -After $identityAfter -Label "VeraPort identity/controller material"
    if ((Get-FileHash -LiteralPath $veraPortConfigPath -Algorithm SHA256).Hash.ToLowerInvariant() -ne $veraPortConfigBefore) {
        throw "VeraPort config changed during WorkBridge installation."
    }

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
        build_cache = [ordered]@{
            status = $buildCacheStatus
            manifest = $cacheManifestPath
            source_commit = $SourceCommit
            go_version = $GoVersion
        }
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
            unauthenticated_health = "DENIED_401"
            token_file = $tokenPath
            token_sha256 = $tokenHash
        }
        authority = [ordered]@{
            read_roots = @($roots)
            write_roots = @()
            process_enabled = $false
            authority_source = "existing VeraPort allowed_roots establish read-only location admission only; write authority disabled for bootstrap"
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
