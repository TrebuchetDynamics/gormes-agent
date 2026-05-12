---
title: "Windows Native"
weight: 25
---

# Windows Native

Gormes has a native PowerShell install path for Windows 10 and Windows 11.
You do not need WSL, Python, Node, Docker, or admin rights for the core
installer path. The installer builds the Go binary from a managed checkout,
publishes `gormes.exe`, updates the user PATH, verifies the command, and can
restart an already-running gateway when requested.

WSL2 is still a useful option when you want Linux shell semantics or to share a
project with Linux-only tooling. Native Windows and WSL2 keep separate install
homes, so they can coexist.

## Quick Install

Open PowerShell or Windows Terminal:

```powershell
irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 | iex
```

No admin rights are required. By default, Gormes installs under
`%LOCALAPPDATA%\gormes` and publishes the command under
`%LOCALAPPDATA%\gormes\bin`. Open a new terminal after install so the updated
user PATH is visible.

## Inspect First

For the safer inspect-first path:

```powershell
irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
gormes doctor --offline
```

The script served from `https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1` is the same source as
`scripts/install.ps1` in the repository and the public site fixture pins that
copy.

## Options

Pass options with the scriptblock form:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1))) -DryRun -Branch main
```

| Option | Purpose |
|---|---|
| `-Branch <name>` | Clone or update a specific branch. |
| `-InstallHome <path>` | Override the managed install home. |
| `-InstallDir <path>` | Override the managed checkout directory. |
| `-BinDir <path>` | Override where `gormes.exe` is published. |
| `-Local` | Build from the current source checkout instead of cloning. |
| `-DryRun` | Print the plan without changing the machine. |
| `-RestartGateway auto|always|never` | Control gateway restart after upgrade. |
| `-NoRestart` | Shortcut for `-RestartGateway never`. |

The same controls are available as environment variables:
`GORMES_BRANCH`, `GORMES_INSTALL_HOME`, `GORMES_INSTALL_DIR`,
`GORMES_BIN_DIR`, `GORMES_GO_VERSION`, `GORMES_GO_SHA256`, and
`GORMES_RESTART_GATEWAY`.

## What The Installer Does

1. Chooses the managed install home, checkout directory, and published bin
   directory.
2. Acquires an install lock so two installers do not publish over each other.
3. Finds Go on PATH, tries `winget`, then tries `choco`, then downloads a
   managed Go toolchain from `go.dev` when needed.
4. Clones or updates `https://github.com/TrebuchetDynamics/gormes-agent.git`.
5. Builds `./cmd/gormes` with `go build -trimpath`.
6. Publishes `gormes.exe` atomically and refreshes any active command path.
7. Adds the published bin directory to the user PATH.
8. Runs `gormes version` and `gormes doctor --offline`.
9. Restarts the gateway only when the selected restart policy says to do so.
10. Appends a local install ledger entry.

Unlike Hermes' Python installer, this path does not create a Python environment
or install JavaScript packages. Provider credentials, channel tokens, and setup
choices remain explicit follow-up steps.

## Verify

Open a fresh terminal:

```powershell
Get-Command gormes
gormes version
gormes doctor --offline
gormes setup
```

`install.ps1` verifies the offline doctor during install. Run `gormes setup`
afterward when you are ready to configure providers, tools, agents, and channel
bindings.

## Update Or Remove

Rerun the installer to update the managed checkout and rebuild `gormes.exe`.
The update path uses `git fetch`, `git checkout`, and `git pull --ff-only`.

To remove Gormes, inspect first:

```powershell
gormes uninstall --dry-run
```

Then run the destructive path only when the dry-run output matches what you
intend to remove:

```powershell
gormes uninstall --yes
```

## Common Pitfalls

| Symptom | Fix |
|---|---|
| `gormes` is not found after install | Open a new terminal so the user PATH refreshes. |
| PowerShell blocks script execution | Use the inspect-first command with `-ExecutionPolicy Bypass` for that invocation. |
| Go is missing and package managers are unavailable | Let the installer use the managed `go.dev` fallback, or install Go manually and rerun. |
| A gateway was running during upgrade | Use `-RestartGateway always` to force restart, or `-NoRestart` to leave it alone. |
| You want an isolated test install | Set `GORMES_INSTALL_HOME` and `GORMES_BIN_DIR`, then run with `-DryRun` first. |
