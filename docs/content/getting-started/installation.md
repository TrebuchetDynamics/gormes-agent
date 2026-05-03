---
title: "Installation"
description: "Build or install Gormes with an inspectable path."
weight: 10
---

# Installation

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

The installer downloads the latest compatible release archive and publishes a
user-scoped `gormes` command. If no release exists yet, or no compatible Unix
archive is available, it clones the requested branch into a temporary directory,
builds from source, publishes the binary, and removes the temporary checkout.

## Requirements

Release installs need `curl` or `wget` plus `tar`. Source fallback needs Git and
Go 1.25+; the installer can download a managed Go toolchain when local Go is
missing. Prefer `./bin/gormes` while validating a source checkout so PATH
shadowing cannot hide an older binary.
