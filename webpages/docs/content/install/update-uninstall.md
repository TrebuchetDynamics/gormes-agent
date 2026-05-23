---
title: "Update and uninstall"
description: "Update or remove a managed Gormes install and verify the result."
aliases:
  - /install/update/
  - /install/uninstall/
---

Use this page for guided update/removal workflows. Exact flags live in the
[CLI reference](../../cli/).

## Update

```bash
gormes update
gormes version
gormes doctor --offline
```

Installer-managed Unix installs can also rerun the release installer:

```bash
curl -fsSL https://gormes.ai/install.sh | bash
gormes doctor --offline
```

Windows installs use the PowerShell installer path from [Windows](../windows/).

## Uninstall

Review first:

```bash
gormes uninstall --dry-run
```

Then remove:

```bash
gormes uninstall
```

If you installed through `install.sh`, the script can delegate uninstall flags:

```bash
sh install.sh --uninstall --dry-run
sh install.sh --uninstall --yes
```

State under `$GORMES_HOME` may include config, secrets, profiles, sessions,
memory, logs, and backups. Review the dry-run before deleting local state.
