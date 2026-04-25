# install.ps1 - source-backed Windows installer for Gormes.
#
# Usage:
#   irm https://gormes.ai/install.ps1 | iex
#
# Environment overrides:
#   GORMES_BRANCH        target branch (default: main)
#   GORMES_INSTALL_HOME  managed install home (default: $env:LOCALAPPDATA\gormes)
#   GORMES_INSTALL_DIR   managed checkout directory (default: $InstallHome\gormes-agent)
#   GORMES_BIN_DIR       published command directory (default: $InstallHome\bin)
#   GORMES_GO_VERSION    managed Go fallback version (default: 1.25.0)
#
# This installer mirrors the Unix install.sh contract on Windows:
#   * managed checkout under a Hermes-analogy install home
#   * rerun-as-update with autostash for local edits
#   * stable global gormes.exe under the published bin directory
#   * winget -> choco -> managed go.dev download fallback for Go
#
# Tested against Windows PowerShell 5.1 and PowerShell 7+.

$ErrorActionPreference = 'Stop'

# Force TLS 1.2 for older Windows PowerShell hosts that still default to TLS 1.0/1.1.
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.ServicePointManager]::SecurityProtocol
} catch {
    # Best-effort; PowerShell 7+ on .NET Core does not need this.
}

$Script:GormesBranch      = if ($env:GORMES_BRANCH)       { $env:GORMES_BRANCH }       else { 'main' }
$Script:GormesGoVersion   = if ($env:GORMES_GO_VERSION)   { $env:GORMES_GO_VERSION }   else { '1.25.0' }
$Script:GormesRepoHttps   = if ($env:GORMES_REPO_URL_HTTPS) { $env:GORMES_REPO_URL_HTTPS } else { 'https://github.com/TrebuchetDynamics/gormes-agent.git' }
$Script:GormesInstallHome = if ($env:GORMES_INSTALL_HOME) { $env:GORMES_INSTALL_HOME } else { Join-Path $env:LOCALAPPDATA 'gormes' }
$Script:GormesInstallDir  = if ($env:GORMES_INSTALL_DIR)  { $env:GORMES_INSTALL_DIR }  else { Join-Path $Script:GormesInstallHome 'gormes-agent' }
$Script:GormesBinDir      = if ($env:GORMES_BIN_DIR)      { $env:GORMES_BIN_DIR }      else { Join-Path $Script:GormesInstallHome 'bin' }

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
        & go build -trimpath -ldflags '-s -w' -o $buildBin .\cmd\gormes
        if ($LASTEXITCODE -ne 0) { Stop-GormesWithError 'go build failed' }
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
    if (Test-Path $tmp) { Remove-Item -Force $tmp }
    Copy-Item -Path $buildBin -Destination $tmp -Force
    Move-Item -Path $tmp -Destination $publishedBin -Force
    if (-not (Test-Path $publishedBin)) { Stop-GormesWithError "could not publish $publishedBin" }
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

function Show-InstallSummary([bool]$PathUpdated) {
    $binDir = Get-PublishedBinDir
    $publishedBin = Join-Path $binDir 'gormes.exe'
    Write-GormesLog 'Core install: succeeded'
    Write-GormesLog "Managed checkout: $(Get-ManagedCheckoutDir)"
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

function Invoke-Main {
    Ensure-Git
    Ensure-Go
    Install-Repository
    Build-Gormes
    Publish-Gormes
    $pathUpdated = Ensure-UserPathContainsBin
    Verify-Install
    Show-InstallSummary -PathUpdated $pathUpdated
}

if ($env:GORMES_INSTALL_TEST_MODE -ne '1') {
    Invoke-Main
}
