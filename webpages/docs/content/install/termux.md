---
title: "Termux"
description: "Install and operate Gormes on Android with Termux as a no-root arm64 controller."
aliases:
  - /install/android/
  - /termux/
---

Termux is supported as a no-root Android arm64/aarch64 runtime for PC-like
Gormes operator workflows. Use it for setup, short CLI/TUI turns, gateway
control, notes, and operator review. Put heavy browser automation, GPU/local
model inference, Docker builds, and large `go test ./...` runs on a workstation
or server reached over SSH.

## Install

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
gormes version
gormes doctor --offline --json
gormes config check
```

On Termux, `install.sh` detects `TERMUX_VERSION` or the standard Termux
`$PREFIX` path and publishes the command to `$PREFIX/bin/gormes`. The release
asset is `android-arm64`, not `linux-arm64`.

Only source fallback or contributor builds need the build toolchain:

```bash
pkg update
pkg install git golang clang tmux openssh curl jq sqlite
```

## First smoke test

If provider credentials are configured:

```bash
gormes chat -q "hello from Termux"
```

For long gateway sessions, use the foreground/tmux model:

```bash
tmux new -s gormes-gateway
termux-wake-lock  # optional best-effort aid
gormes gateway
```

Use another Termux shell to inspect or stop the runtime:

```bash
gormes gateway status
gormes gateway stop
```

Android battery optimization can still stop background processes. Gormes does
not install or manage Android services automatically.

## Termux as controller

Set a shell shortcut for the host you use most:

```bash
export GORMES_REMOTE_HOST=workstation
ssh workstation 'gormes doctor --offline'
```

Use tmux on the remote machine for long-running builds and browser sessions:

```bash
ssh -t "$GORMES_REMOTE_HOST" 'tmux new -A -s gormes-build'
cd ~/code/gormes-agent
go test ./...
```

Run one-off remote agent turns from Termux with the existing scripted-chat
surface:

```bash
ssh "$GORMES_REMOTE_HOST" 'cd ~/code/gormes-agent && gormes chat -q "summarize the current failing tests"'
```
