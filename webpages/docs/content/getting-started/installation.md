---
title: "Installation"
description: "Build or install Gormes with an inspectable path."
weight: 10
---

# Installation

## Method 1: Build From Source

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
gormes version
gormes doctor --offline
```

This is the primary trust path while Gormes is moving quickly: inspect the
source tree, build the command, put the fresh build first on `PATH`, and run
offline diagnostics before adding credentials.

## Method 2: install.sh

```bash
curl -fsSLO https://gormes.ai/install.sh
less install.sh
sh install.sh
gormes doctor --offline
```

The installer mirrors Hermes' source-backed user flow for Gormes: it clones or
updates a managed checkout, builds `gormes`, publishes the command, verifies
`gormes version`, runs `gormes doctor --offline`, and starts `gormes setup`
when a terminal is available. Rerun the same command to update.

If you intentionally want the one-line convenience form:

```bash
curl -fsSL https://gormes.ai/install.sh | bash
```

To defer setup, run `sh install.sh --skip-setup` after the inspectable download
or `curl -fsSL https://gormes.ai/install.sh | bash -s -- --skip-setup`. By
default the installer keeps code under `~/.gormes/gormes-agent`, publishes to
`~/.local/bin/gormes`, and uses `/usr/local/lib/gormes-agent` plus
`/usr/local/bin/gormes` for new root Linux installs. If no terminal is
available, the setup wizard is skipped with guidance to run `gormes setup`
later.

## Requirements

Source builds need Git and Go 1.26+. The installer can download a managed Go
toolchain when local Go is missing. Put the freshly built source checkout first
on `PATH` while validating development work so PATH shadowing cannot hide an
older installed command.
