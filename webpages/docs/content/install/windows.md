---
title: "Windows native"
description: "Install Gormes on Windows 10 or Windows 11 with the release-first install.ps1 PowerShell bootstrap."
weight: 20
aliases:
  - /using-gormes/windows-native/
---

# Windows native

Gormes has a native PowerShell install path for Windows 10 and Windows 11. You do not need WSL, Python, Node, Docker, or admin rights for the core installer. `install.ps1` is release-first: by default it downloads the latest signed release archive for the host architecture, verifies its SHA-256, and publishes `gormes.exe` under `%LOCALAPPDATA%\gormes\bin`. It falls back to a managed source build when needed.

WSL2 is still a useful option when you want Linux shell semantics. Native Windows and WSL2 keep separate install homes, so both can coexist.

## 60-second install

Open PowerShell or Windows Terminal:

```powershell
irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 | iex
```

No admin rights are required. By default, Gormes installs under `%LOCALAPPDATA%\gormes` and publishes `gormes.exe` under `%LOCALAPPDATA%\gormes\bin`. Open a new terminal after install so the updated user PATH is visible.

After the installer finishes:

```powershell
gormes --version
gormes doctor --offline
```

## Inspect first

For the safer inspect-first path:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
gormes doctor --offline
```

The script served from `https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1` is the same source as `scripts/install.ps1` in the repository.

## Customize

`install.ps1` exposes operator controls as both PowerShell parameters and environment variables. Pass parameters with the scriptblock form when you are still using the one-liner:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1))) -DryRun -Branch main
```

| Parameter | Purpose |
|---|---|
| `-FromSource` | Build from source instead of fetching the release binary. Equivalent env var: `GORMES_INSTALL_FROM_SOURCE=1`. |
| `-Local` | Build from the current source checkout instead of cloning. |
| `-DryRun` | Print the resolved plan without changing the machine. |
| `-Branch <name>` | Target a non-default branch. Triggers a source build because release binaries are only published from `main`. |
| `-InstallHome <path>` | Override the managed install home (default: `%LOCALAPPDATA%\gormes`). |
| `-InstallDir <path>` | Override the managed checkout directory. |
| `-BinDir <path>` | Override the directory `gormes.exe` is published to. |
| `-RestartGateway auto\|always\|never` | Control whether the installer restarts a live gateway after upgrade. |
| `-NoRestart` | Shortcut for `-RestartGateway never`. |

The same controls are available as environment variables:
`GORMES_INSTALL_HOME`, `GORMES_INSTALL_DIR`, `GORMES_BIN_DIR`, `GORMES_BRANCH`, `GORMES_GO_VERSION`, `GORMES_GO_SHA256`, `GORMES_RESTART_GATEWAY`, and `GORMES_INSTALL_FROM_SOURCE`.

When the installer needs a source build it looks for Go on PATH, then tries `winget` (`GoLang.Go`), then `choco` (`golang`), and finally downloads a managed Go toolchain from `go.dev` into `%LOCALAPPDATA%\gormes\go`. Git is bootstrapped the same way via `winget` / `choco` if it is missing.

## Verify

Open a fresh terminal so PATH refreshes:

```powershell
Get-Command gormes
gormes --version
gormes doctor --offline
```

Once the offline doctor passes, run `gormes setup` to configure providers, channel credentials, and tool defaults.

Rerun the installer to update Gormes. The update path uses `git fetch`, `git checkout`, and `git pull --ff-only` when source-building, or a fresh release archive download when fetching the binary. To remove Gormes:

```powershell
gormes uninstall --dry-run
gormes uninstall --yes
```

## Common pitfalls

| Symptom | Fix |
|---|---|
| `gormes` is not found after install | Open a new terminal so the user PATH refreshes. |
| PowerShell blocks script execution | Use the inspect-first command with `-ExecutionPolicy Bypass` for that invocation. |
| Go is missing and `winget`, `choco` are unavailable | Let the installer use the managed `go.dev` fallback, or install Go manually and rerun. |
| A gateway was running during upgrade | Use `-RestartGateway always` to force restart, or `-NoRestart` to leave it alone. |
| You want an isolated test install | Set `GORMES_INSTALL_HOME` and `GORMES_BIN_DIR`, then run with `-DryRun` first. |
