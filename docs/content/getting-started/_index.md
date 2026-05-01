---
title: "Getting Started"
description: "Install Gormes, verify the local runtime, and complete a first useful run."
weight: 10
---

# Getting Started

Start here when you want Gormes running, not explained from first principles.

1. [Install](installation/) from source or an inspectable installer.
2. [First run](first-run/) with offline diagnostics and a local TUI smoke test.
3. [Configure](configuration/) provider, gateway, and state paths.
4. [Troubleshoot](troubleshooting/) common setup and runtime failures.

The conservative path is to verify the local runtime before adding model or channel credentials:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes doctor --offline
./bin/gormes --offline
```
