---
title: "Installation"
description: "Build or install Gormes with an inspectable path."
weight: 10
---

# Installation

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | bash
```

The installer mirrors Hermes' source-backed user flow for Gormes: it clones or
updates a managed checkout, builds `gormes`, publishes the command, verifies
`gormes version`, runs `gormes doctor --offline`, and starts `gormes setup`
when a terminal is available. Rerun the same command to update. To defer setup,
run `curl -fsSL ... | bash -s -- --skip-setup`.

## Source Build

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes version
./bin/gormes doctor --offline
```

This is the primary trust path while Gormes is moving quickly: inspect the source tree, build the binary, and run offline diagnostics before adding credentials.

## Inspectable Installer

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh
less install.sh
sh install.sh
```

Use the inspectable form when you want to read the script before running it. By
default it keeps code under `~/.gormes/gormes-agent`, publishes to
`~/.local/bin/gormes`, and uses `/usr/local/lib/gormes-agent` plus
`/usr/local/bin/gormes` for new root Linux installs. If no terminal is
available, the setup wizard is skipped with guidance to run `gormes setup`
later.

## Requirements

Installs need Git and Go 1.25+; the installer can download a managed Go
toolchain when local Go is missing. Prefer `./bin/gormes` while validating a
development checkout so PATH shadowing cannot hide an older installed binary.
