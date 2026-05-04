# install.ps1 - source-backed Windows installer for Gormes.
#
# Usage:
#   Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
#   Get-Content .\install.ps1
#   powershell -ExecutionPolicy Bypass -File .\install.ps1
#
# Environment overrides:
#   GORMES_BRANCH        target branch (default: main)
#   GORMES_INSTALL_HOME  managed install home (default: $env:LOCALAPPDATA\gormes)
#   GORMES_INSTALL_DIR   managed checkout directory (default: $InstallHome\gormes-agent)
#   GORMES_BIN_DIR       published command directory (default: $InstallHome\bin)
#   GORMES_GO_VERSION    managed Go fallback version (default: 1.25.0)
#   GORMES_RESTART_GATEWAY restart policy: auto, always, never (default: auto)
#   GORMES_GO_SHA256     optional expected SHA-256 for managed Go download
#
# This installer mirrors the Unix install.sh contract on Windows:
#   * managed checkout under the Gormes install home
#   * rerun-as-update with autostash for local edits
#   * stable global gormes.exe under the published bin directory
#   * winget -> choco -> managed go.dev download fallback for Go
#
# Tested against Windows PowerShell 5.1 and PowerShell 7+.

param(
    [string]$Branch,
    [Alias('Home')]
    [string]$InstallHome,
    [Alias('Dir')]
    [string]$InstallDir,
    [string]$BinDir,
    [switch]$Local,
    [switch]$DryRun,
    [ValidateSet('auto', 'always', 'never')]
    [string]$RestartGateway,
    [switch]$NoRestart
)

$ErrorActionPreference = 'Stop'

# Force TLS 1.2 for older Windows PowerShell hosts that still default to TLS 1.0/1.1.
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.ServicePointManager]::SecurityProtocol
} catch {
    # Best-effort; PowerShell 7+ on .NET Core does not need this.
}

$Script:GormesBranch      = if ($Branch) { $Branch } elseif ($env:GORMES_BRANCH) { $env:GORMES_BRANCH } else { 'main' }
$Script:GormesGoVersion   = if ($env:GORMES_GO_VERSION) { $env:GORMES_GO_VERSION } else { '1.25.0' }
$Script:GormesRepoHttps   = if ($env:GORMES_REPO_URL_HTTPS) { $env:GORMES_REPO_URL_HTTPS } else { 'https://github.com/TrebuchetDynamics/gormes-agent.git' }
$Script:GormesInstallHome = if ($InstallHome) { $InstallHome } elseif ($env:GORMES_INSTALL_HOME) { $env:GORMES_INSTALL_HOME } else { Join-Path $env:LOCALAPPDATA 'gormes' }
$Script:GormesInstallDir  = if ($InstallDir) { $InstallDir } elseif ($env:GORMES_INSTALL_DIR) { $env:GORMES_INSTALL_DIR } else { Join-Path $Script:GormesInstallHome 'gormes-agent' }
$Script:GormesBinDir      = if ($BinDir) { $BinDir } elseif ($env:GORMES_BIN_DIR) { $env:GORMES_BIN_DIR } else { Join-Path $Script:GormesInstallHome 'bin' }
$Script:GormesGoSha256    = if ($env:GORMES_GO_SHA256) { $env:GORMES_GO_SHA256.ToLowerInvariant() } else { '' }
$Script:RestartGateway    = if ($NoRestart) { 'never' } elseif ($RestartGateway) { $RestartGateway } elseif ($env:GORMES_RESTART_GATEWAY) { $env:GORMES_RESTART_GATEWAY } else { 'auto' }
$Script:DryRun            = [bool]$DryRun
$Script:LocalSourceDir    = if ($Local) { (Get-Location).Path } else { '' }
$Script:InstallLockDir    = ''
$Script:OldBuildTag       = ''
$Script:BuildTag          = ''
$Script:PreviousGatewayPid = $null
$Script:NewGatewayPid      = $null

if (@('auto', 'always', 'never') -notcontains $Script:RestartGateway) {
    [Console]::Error.WriteLine('[gormes] error: GORMES_RESTART_GATEWAY must be auto, always, or never')
    throw 'GORMES_RESTART_GATEWAY must be auto, always, or never'
}

function Write-GormesLog([string]$Message) {
    [Console]::Error.WriteLine("[gormes] $Message")
}

function Stop-GormesWithError([string]$Message) {
    [Console]::Error.WriteLine("[gormes] error: $Message")
    throw $Message
}

function Get-ManagedHome { $Script:GormesInstallHome }
function Get-ManagedCheckoutDir { $Script:GormesInstallDir }
function Get-PublishedBinDir { $Script:GormesBinDir }
function Get-ManagedBuildBin { Join-Path (Join-Path $Script:GormesInstallHome 'bin-build') 'gormes.exe' }

function Test-CommandExists([string]$Name) {
    [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Get-FileSha256([string]$Path) {
    if (-not (Test-Path $Path)) { return '' }
    return (Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLowerInvariant()
}

function Test-SameBinary([string]$Left, [string]$Right) {
    if (-not $Left -or -not $Right) { return $false }
    if (-not (Test-Path $Left) -or -not (Test-Path $Right)) { return $false }
    try {
        $leftResolved = (Resolve-Path -LiteralPath $Left).Path
        $rightResolved = (Resolve-Path -LiteralPath $Right).Path
        if ($leftResolved -eq $rightResolved) { return $true }
    } catch {
        # Fall through to content hash comparison.
    }
    $leftHash = Get-FileSha256 $Left
    $rightHash = Get-FileSha256 $Right
    return ($leftHash -and ($leftHash -eq $rightHash))
}

function Get-ActiveCommandPath {
    $cmd = Get-Command 'gormes.exe' -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($cmd) { return $cmd.Source }
    $cmd = Get-Command 'gormes' -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($cmd) { return $cmd.Source }
    return $null
}

function Get-AllCommandPaths {
    $paths = @()
    $commands = @(Get-Command 'gormes.exe' -All -ErrorAction SilentlyContinue) + @(Get-Command 'gormes' -All -ErrorAction SilentlyContinue)
    foreach ($cmd in $commands) {
        if ($cmd.Source -and ($paths -notcontains $cmd.Source)) { $paths += $cmd.Source }
    }
    $published = Join-Path (Get-PublishedBinDir) 'gormes.exe'
    if ((Test-Path $published) -and ($paths -notcontains $published)) { $paths += $published }
    return $paths
}

function Get-GoVersionString {
    try {
        $version = (& go env GOVERSION 2>$null)
        if (-not $version) {
            $line = (& go version 2>$null)
            if ($line -match '(go\d+\.\d+(\.\d+)?)') { $version = $Matches[1] }
        }
        return ($version | Select-Object -First 1)
    } catch {
        return $null
    }
}

function Test-GoVersionSupported([string]$Version) {
    if (-not $Version) { return $false }
    return ($Version -match '^go1\.(2[5-9]|[3-9][0-9])') -or ($Version -match '^go[2-9]')
}

function Invoke-WinGet([string[]]$Arguments) {
    if (-not (Test-CommandExists 'winget')) { return $false }
    try {
        & winget @Arguments | Out-Null
        return ($LASTEXITCODE -eq 0)
    } catch {
        return $false
    }
}

function Invoke-Choco([string[]]$Arguments) {
    if (-not (Test-CommandExists 'choco')) { return $false }
    try {
        & choco @Arguments | Out-Null
        return ($LASTEXITCODE -eq 0)
    } catch {
        return $false
    }
}

function Refresh-PathFromEnvironment {
    $machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $user    = [Environment]::GetEnvironmentVariable('Path', 'User')
    $combined = @($machine, $user) | Where-Object { $_ } | ForEach-Object { $_.TrimEnd(';') }
    $env:Path = ($combined -join ';')
}

function Ensure-Git {
    if (Test-CommandExists 'git') { return }

    Write-GormesLog 'git not found; attempting to install via winget'
    if (Invoke-WinGet @('install', '--id', 'Git.Git', '--exact', '--silent', '--accept-package-agreements', '--accept-source-agreements')) {
        Refresh-PathFromEnvironment
    } elseif (Invoke-Choco @('install', 'git', '-y', '--no-progress')) {
        Refresh-PathFromEnvironment
    }

    if (-not (Test-CommandExists 'git')) {
        Stop-GormesWithError 'git is required and could not be installed automatically; install Git for Windows manually then rerun this script'
    }
}

function Verify-ManagedGoDownload([string]$ZipPath) {
    if (-not $Script:GormesGoSha256) {
        Write-GormesLog 'Go download sha256 verification skipped; set GORMES_GO_SHA256 to enforce it'
        return
    }
    Write-GormesLog 'verifying Go download sha256'
    $actual = Get-FileSha256 $ZipPath
    if (-not $actual) {
        Stop-GormesWithError "could not compute sha256 for $ZipPath"
    }
    if ($actual -ne $Script:GormesGoSha256) {
        Stop-GormesWithError "Go download sha256 mismatch: expected $($Script:GormesGoSha256), got $actual"
    }
    Write-GormesLog 'Go download sha256 verified'
}

function Install-ManagedGo {
    $home = Get-ManagedHome
    $managedRoot = Join-Path $home 'go'
    $managedBin  = Join-Path $managedRoot 'bin'
    $managedGo   = Join-Path $managedBin 'go.exe'

    if (Test-Path $managedGo) {
        $env:Path = "$managedBin;$env:Path"
        $version = Get-GoVersionString
        if (Test-GoVersionSupported $version) {
            Write-GormesLog "using managed $version"
            return
        }
    }

    $arch = switch -Wildcard ($env:PROCESSOR_ARCHITECTURE) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        'x86'   { '386' }
        default { Stop-GormesWithError "managed Go download is not supported for architecture: $($env:PROCESSOR_ARCHITECTURE)" }
    }

    $version = $Script:GormesGoVersion
    $tarball = "go$version.windows-$arch.zip"
    $url     = "https://go.dev/dl/$tarball"
    $tmpDir  = Join-Path $home 'tmp'
    $tmpZip  = Join-Path $tmpDir $tarball

    Write-GormesLog "downloading Go $version for windows/$arch"
    New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null
    Invoke-WebRequest -Uri $url -OutFile $tmpZip -UseBasicParsing
    Verify-ManagedGoDownload $tmpZip

    if (Test-Path $managedRoot) { Remove-Item -Recurse -Force $managedRoot }
    Expand-Archive -Path $tmpZip -DestinationPath $home -Force

    if (-not (Test-Path $managedGo)) {
        Stop-GormesWithError "managed Go install completed but $managedGo was not created"
    }

    $env:Path = "$managedBin;$env:Path"
    Write-GormesLog "installed managed Go $version under $managedRoot"
}

function Ensure-Go {
    if (Test-CommandExists 'go') {
        $version = Get-GoVersionString
        if (Test-GoVersionSupported $version) { return }
        Write-GormesLog "found $version; installing managed Go $($Script:GormesGoVersion)"
    } else {
        Write-GormesLog 'go not found; attempting to install via winget'
        if (Invoke-WinGet @('install', '--id', 'GoLang.Go', '--exact', '--silent', '--accept-package-agreements', '--accept-source-agreements')) {
            Refresh-PathFromEnvironment
            $version = Get-GoVersionString
            if (Test-GoVersionSupported $version) { return }
        } elseif (Invoke-Choco @('install', 'golang', '-y', '--no-progress')) {
            Refresh-PathFromEnvironment
            $version = Get-GoVersionString
            if (Test-GoVersionSupported $version) { return }
        }
    }

    Install-ManagedGo

    $version = Get-GoVersionString
    if (-not (Test-GoVersionSupported $version)) {
        Stop-GormesWithError "Go 1.25+ required; found $version"
    }
}

function Get-BuildRoot {
    if ($Script:LocalSourceDir) {
        if ((Test-Path (Join-Path $Script:LocalSourceDir 'go.mod')) -and (Test-Path (Join-Path $Script:LocalSourceDir 'cmd\gormes'))) {
            return $Script:LocalSourceDir
        }
        Stop-GormesWithError '-Local must be run from a Gormes source checkout'
    }

    $checkout = Get-ManagedCheckoutDir
    if ((Test-Path (Join-Path $checkout 'go.mod')) -and (Test-Path (Join-Path $checkout 'cmd\gormes'))) {
        return $checkout
    }
    $sub = Join-Path $checkout 'gormes'
    if ((Test-Path (Join-Path $sub 'go.mod')) -and (Test-Path (Join-Path $sub 'cmd\gormes'))) {
        return $sub
    }
    Stop-GormesWithError "could not find a Gormes Go module under $checkout"
}

function Install-Repository {
    if ($Script:LocalSourceDir) {
        Write-GormesLog "using local source checkout $($Script:LocalSourceDir)"
        return
    }

    $checkout = Get-ManagedCheckoutDir
    $parent   = Split-Path -Parent $checkout
    New-Item -ItemType Directory -Force -Path $parent | Out-Null

    if (Test-Path (Join-Path $checkout '.git')) {
        Write-GormesLog "updating managed checkout $checkout"
        Push-Location $checkout
        try {
            $stashed = $false
            $status = (& git status --porcelain) -join "`n"
            if ($status.Trim()) {
                Write-GormesLog 'local changes detected; stashing before update'
                & git stash push --include-untracked -m 'gormes installer autostash' | Out-Null
                if ($LASTEXITCODE -ne 0) { Stop-GormesWithError "could not stash local changes in $checkout" }
                $stashed = $true
            }

            & git fetch origin $Script:GormesBranch
            if ($LASTEXITCODE -ne 0) { Stop-GormesWithError "could not fetch origin/$($Script:GormesBranch)" }
            & git checkout $Script:GormesBranch
            if ($LASTEXITCODE -ne 0) { Stop-GormesWithError "could not checkout $($Script:GormesBranch)" }
            & git pull --ff-only origin $Script:GormesBranch
            if ($LASTEXITCODE -ne 0) { Stop-GormesWithError "could not fast-forward $($Script:GormesBranch)" }

            if ($stashed) {
                & git stash pop | Out-Null
                if ($LASTEXITCODE -ne 0) {
                    Stop-GormesWithError "updated checkout but could not reapply stashed changes; inspect: cd $checkout && git stash list"
                }
                Write-GormesLog 'local changes restored after update'
            }
        } finally {
            Pop-Location
        }
        return
    }

    if (Test-Path $checkout) {
        Stop-GormesWithError "$checkout exists but is not a git checkout; remove it or rerun with GORMES_INSTALL_DIR"
    }

    Write-GormesLog "cloning Gormes into $checkout"
    & git clone --branch $Script:GormesBranch $Script:GormesRepoHttps $checkout
    if ($LASTEXITCODE -ne 0) { Stop-GormesWithError "could not clone Gormes from $($Script:GormesRepoHttps)" }
}

function Build-Gormes {
    $buildRoot = Get-BuildRoot
    $buildBin  = Get-ManagedBuildBin
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $buildBin) | Out-Null

    Write-GormesLog "building gormes from $buildRoot"
    Push-Location $buildRoot
    try {
        try {
            $Script:OldBuildTag = if (Test-Path "$buildBin.build-tag") { Get-Content "$buildBin.build-tag" -Raw } else { '' }
            $Script:OldBuildTag = $Script:OldBuildTag.Trim()
            $Script:BuildTag = (& git rev-parse --short HEAD 2>$null)
            if (-not $Script:BuildTag) { $Script:BuildTag = 'unknown' }
        } catch {
            $Script:BuildTag = 'unknown'
        }
        & go build -trimpath -ldflags '-s -w' -o $buildBin .\cmd\gormes
        if ($LASTEXITCODE -ne 0) { Stop-GormesWithError 'go build failed' }
        Set-Content -Path "$buildBin.build-tag" -Value $Script:BuildTag -Encoding ASCII
    } finally {
        Pop-Location
    }

    if (-not (Test-Path $buildBin)) { Stop-GormesWithError "build completed but $buildBin was not created" }
}

function Publish-Gormes {
    $buildBin    = Get-ManagedBuildBin
    $binDir      = Get-PublishedBinDir
    $publishedBin = Join-Path $binDir 'gormes.exe'

    New-Item -ItemType Directory -Force -Path $binDir | Out-Null

    $tmp = "$publishedBin.tmp.$PID"
    $backup = "$publishedBin.rollback.$PID"
    if (Test-Path $tmp) { Remove-Item -Force $tmp }
    if (Test-Path $backup) { Remove-Item -Force $backup }
    $hadPrevious = Test-Path $publishedBin
    if ($hadPrevious) {
        Copy-Item -Path $publishedBin -Destination $backup -Force
    }
    Copy-Item -Path $buildBin -Destination $tmp -Force
    Move-Item -Path $tmp -Destination $publishedBin -Force
    if (-not (Test-Path $publishedBin)) { Stop-GormesWithError "could not publish $publishedBin" }
    & $publishedBin version | Out-Null
    if ($LASTEXITCODE -ne 0) {
        if ($hadPrevious -and (Test-Path $backup)) {
            Move-Item -Path $backup -Destination $publishedBin -Force
        } elseif (Test-Path $publishedBin) {
            Remove-Item -Force $publishedBin
        }
        Stop-GormesWithError "published command verification failed for $publishedBin; rolled back"
    }
    if (Test-Path $backup) { Remove-Item -Force $backup }
}

function Update-ActiveCommand {
    $buildBin = Get-ManagedBuildBin
    $publishedBin = Join-Path (Get-PublishedBinDir) 'gormes.exe'
    foreach ($candidate in Get-AllCommandPaths) {
        if (-not $candidate) { continue }
        if ($candidate -eq $publishedBin) { continue }
        if ($candidate -eq $buildBin) { continue }
        if (Test-SameBinary $candidate $buildBin) { continue }
        Write-GormesLog "updating active PATH command $candidate"
        Copy-Item -Path $buildBin -Destination $candidate -Force
    }
}

function Ensure-UserPathContainsBin {
    $binDir = Get-PublishedBinDir
    try {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        if (-not $userPath) { $userPath = '' }
        $segments = $userPath.Split(';') | Where-Object { $_ }
        if ($segments -notcontains $binDir) {
            $newPath = (@($binDir) + $segments) -join ';'
            [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        }
        # Also update the current session so the user can run gormes without restarting.
        if ((';' + $env:Path + ';') -notlike "*;$binDir;*") {
            $env:Path = "$binDir;$env:Path"
        }
        return $true
    } catch {
        Write-GormesLog "PATH update skipped: $($_.Exception.Message)"
        return $false
    }
}

function Get-GatewayStatusOutput([string]$BinPath) {
    if (-not $BinPath -or -not (Test-Path $BinPath)) { return '' }
    try {
        $json = (& $BinPath gateway status --json 2>$null) -join "`n"
        if ($LASTEXITCODE -eq 0 -and $json) { return $json }
    } catch {
        # Fall back to the human status output below.
    }
    try {
        return ((& $BinPath gateway status 2>$null) -join "`n")
    } catch {
        return ''
    }
}

function Get-RunningGatewayPidFromStatus([string]$Status) {
    if (-not $Status) { return $null }
    try {
        $parsed = $Status | ConvertFrom-Json -ErrorAction Stop
        if ($parsed.runtime.gateway_state -eq 'running' -and $parsed.runtime.pid) {
            return [int]$parsed.runtime.pid
        }
        if ($parsed.validation.live -and $parsed.validation.pid) {
            return [int]$parsed.validation.pid
        }
    } catch {
        # Human status fallback.
    }
    if ($Status -match 'runtime:\s+running\s+\(pid=(\d+)') {
        return [int]$Matches[1]
    }
    return $null
}

function Get-RunningGatewayPid {
    $active = Get-ActiveCommandPath
    $status = Get-GatewayStatusOutput $active
    $pid = Get-RunningGatewayPidFromStatus $status
    if ($pid) { return $pid }

    $published = Join-Path (Get-PublishedBinDir) 'gormes.exe'
    $status = Get-GatewayStatusOutput $published
    return (Get-RunningGatewayPidFromStatus $status)
}

function Stop-GatewayForRestart([int]$OldPid) {
    $bin = Get-ActiveCommandPath
    if (-not $bin) { $bin = Join-Path (Get-PublishedBinDir) 'gormes.exe' }
    if (-not (Test-Path $bin)) { return $false }
    try {
        & $bin gateway stop | Out-Null
    } catch {
        return $false
    }
    for ($i = 0; $i -lt 10; $i++) {
        try {
            $process = Get-Process -Id $OldPid -ErrorAction SilentlyContinue
            if (-not $process) { return $true }
        } catch {
            return $true
        }
        Start-Sleep -Milliseconds 500
    }
    return $true
}

function Start-GatewayForRestart {
    $bin = Join-Path (Get-PublishedBinDir) 'gormes.exe'
    if (-not (Test-Path $bin)) { return $false }
    try {
        Start-Process -FilePath $bin -ArgumentList @('gateway') -WindowStyle Hidden | Out-Null
        return $true
    } catch {
        Write-GormesLog "gateway restart failed: $($_.Exception.Message)"
        return $false
    }
}

function Wait-ForGatewayRestart($OldPid) {
    $bin = Join-Path (Get-PublishedBinDir) 'gormes.exe'
    for ($i = 0; $i -lt 8; $i++) {
        $status = Get-GatewayStatusOutput $bin
        $pid = Get-RunningGatewayPidFromStatus $status
        if ($pid -and (($null -eq $OldPid) -or ($pid -ne $OldPid))) {
            return $pid
        }
        Start-Sleep -Seconds 1
    }
    return $null
}

function Restart-GatewayIfRunning($OldPid) {
    if ($Script:RestartGateway -eq 'never') {
        Write-GormesLog 'gateway restart skipped by policy=never'
        return
    }
    if ($Script:RestartGateway -eq 'auto' -and (-not $OldPid)) { return }

    if ($OldPid) {
        Write-GormesLog "restarting live gateway pid=$OldPid"
        if (-not (Stop-GatewayForRestart $OldPid)) {
            Write-GormesLog "gateway restart skipped: could not stop pid=$OldPid"
            return
        }
    } else {
        Write-GormesLog 'starting gateway by policy=always'
    }

    if (-not (Start-GatewayForRestart)) {
        Write-GormesLog "gateway restart failed: could not start $(Join-Path (Get-PublishedBinDir) 'gormes.exe') gateway"
        return
    }
    $newPid = Wait-ForGatewayRestart $OldPid
    $Script:NewGatewayPid = $newPid
    if ($newPid) {
        if ($OldPid) {
            Write-GormesLog "gateway restarted pid=$OldPid -> $newPid"
        } else {
            Write-GormesLog "gateway started pid=$newPid"
        }
    } else {
        Write-GormesLog 'gateway restart requested; status did not report a new live pid yet'
    }
}

function Verify-Install {
    $publishedBin = Join-Path (Get-PublishedBinDir) 'gormes.exe'
    if (-not (Test-Path $publishedBin)) { Stop-GormesWithError "published command is not executable: $publishedBin" }
    & $publishedBin version | Out-Null
    if ($LASTEXITCODE -ne 0) { Stop-GormesWithError "verification failed: $publishedBin version" }

    & $publishedBin doctor --offline 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-GormesLog 'offline doctor passed'
    } else {
        Write-GormesLog 'note: offline doctor did not pass; core version smoke check succeeded'
    }
}

function Get-InstallLedgerPath {
    Join-Path (Get-ManagedHome) 'install.log.jsonl'
}

function Append-InstallLedger {
    $buildBin = Get-ManagedBuildBin
    $hash = Get-FileSha256 $buildBin
    $entry = [ordered]@{
        event = 'install'
        timestamp = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
        source = (Get-BuildRoot)
        branch = $Script:GormesBranch
        old_commit = $Script:OldBuildTag
        new_commit = $Script:BuildTag
        binary_sha256 = $hash
        restart_gateway = $Script:RestartGateway
    }
    if ($Script:PreviousGatewayPid) { $entry.old_gateway_pid = $Script:PreviousGatewayPid }
    if ($Script:NewGatewayPid) { $entry.new_gateway_pid = $Script:NewGatewayPid }
    $ledger = Get-InstallLedgerPath
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $ledger) | Out-Null
    ($entry | ConvertTo-Json -Compress) | Add-Content -Encoding UTF8 -Path $ledger
}

function Show-InstallSummary([bool]$PathUpdated) {
    $binDir = Get-PublishedBinDir
    $publishedBin = Join-Path $binDir 'gormes.exe'
    Write-GormesLog 'Core install: succeeded'
    Write-GormesLog "Source: $(Get-BuildRoot)"
    Write-GormesLog "Published command: $publishedBin"
    Write-GormesLog 'Verification: succeeded'

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -and ($userPath.Split(';') -contains $binDir)) {
        Write-GormesLog "PATH: $binDir is on your user PATH"
    } elseif ($PathUpdated) {
        Write-GormesLog "PATH: added $binDir to your user PATH (restart your shell to pick it up)"
    } else {
        Write-GormesLog "PATH: add manually:  setx PATH `"$binDir;%PATH%`""
    }

    Write-GormesLog 'Update: rerun this installer to update Gormes'
}

function Show-DryRun {
    Write-GormesLog 'dry run'
    Write-GormesLog "  branch: $($Script:GormesBranch)"
    if ($Script:LocalSourceDir) {
        Write-GormesLog "  source: $($Script:LocalSourceDir)"
    } else {
        Write-GormesLog "  source: $(Get-ManagedCheckoutDir)"
    }
    Write-GormesLog "  install_home: $(Get-ManagedHome)"
    Write-GormesLog "  managed_binary: $(Get-ManagedBuildBin)"
    Write-GormesLog "  published_binary: $(Join-Path (Get-PublishedBinDir) 'gormes.exe')"
    Write-GormesLog "  restart_gateway: $($Script:RestartGateway)"
}

function Acquire-InstallLock {
    $home = Get-ManagedHome
    $lock = Join-Path $home 'install.lock'
    New-Item -ItemType Directory -Force -Path $home | Out-Null
    try {
        New-Item -ItemType Directory -Path $lock -ErrorAction Stop | Out-Null
        $Script:InstallLockDir = $lock
        Set-Content -Path (Join-Path $lock 'pid') -Value $PID -Encoding ASCII
    } catch {
        Stop-GormesWithError "another install is already running; remove $lock if it is stale"
    }
}

function Release-InstallLock {
    if ($Script:InstallLockDir -and (Test-Path $Script:InstallLockDir)) {
        Remove-Item -Recurse -Force $Script:InstallLockDir
        $Script:InstallLockDir = ''
    }
}

function Invoke-Main {
    if ($Script:DryRun) {
        Show-DryRun
        return
    }

    Acquire-InstallLock
    try {
        $Script:PreviousGatewayPid = Get-RunningGatewayPid
        Ensure-Git
        Ensure-Go
        Install-Repository
        Build-Gormes
        Publish-Gormes
        Update-ActiveCommand
        $pathUpdated = Ensure-UserPathContainsBin
        Verify-Install
        Restart-GatewayIfRunning $Script:PreviousGatewayPid
        Append-InstallLedger
        Show-InstallSummary -PathUpdated $pathUpdated
    } finally {
        Release-InstallLock
    }
}

if ($env:GORMES_INSTALL_TEST_MODE -ne '1') {
    Invoke-Main
}
