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
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh
less install.sh
sh install.sh
```

The installer manages a user-scoped Gormes checkout and binary. Re-run it to update the managed install.

## Requirements

The current module declares Go 1.25.0. Use the Go version required by the branch you are building and prefer `./bin/gormes` while validating a source checkout so PATH shadowing cannot hide an older binary.

Precompiled release artifacts and package-manager installs should be documented only after signed artifacts, checksums, and release URLs are verified for the current release.
