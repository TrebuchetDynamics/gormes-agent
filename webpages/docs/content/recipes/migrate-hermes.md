---
title: "Migrate from Hermes or OpenClaw"
description: "Inspect a dry-run migration manifest, then apply it into the Gormes home."
difficulty: "M"
---

# Migrate from Hermes or OpenClaw

> **Outcome:** Your existing Hermes or OpenClaw config, env, and (optionally) memory + skills are copied into `~/.gormes`, reviewed first as a redacted manifest.
>
> **Prerequisites:** An existing `~/.hermes` or `~/.openclaw` directory. `gormes` installed.

## Steps

1. **Dry-run the migration**
   ```bash
   gormes migrate hermes --dry-run --source ~/.hermes
   ```
   For OpenClaw use either spelling — they map to the same code path:
   ```bash
   gormes migrate openclaw --dry-run --source ~/.openclaw
   gormes claw migrate --dry-run --source ~/.openclaw
   ```
   The manifest prints every source key, the destination it would write, and any conflicts. No files are written.

2. **Review conflicts**
   Look for rows flagged `conflict`. Decide whether to overwrite the existing Gormes value (`--overwrite`) or leave it alone.

3. **Apply the manifest**
   ```bash
   gormes migrate hermes --yes --source ~/.hermes
   ```
   For OpenClaw, add `--secrets` if you want to import secret env values (without it, secret rows are reported as `secret_skipped`):
   ```bash
   gormes migrate openclaw --yes --secrets --source ~/.openclaw
   ```

4. **Archive leftover OpenClaw directories (optional)**
   ```bash
   gormes migrate openclaw cleanup --dry-run
   gormes migrate openclaw cleanup --yes
   ```
   Old `~/.openclaw`, `~/.clawdbot`, and `~/.moltbot` directories are renamed to `.pre-migration` variants — not deleted.

## Verify

```bash
gormes config show
gormes setup --quick --non-interactive
```

Expected: the provider/model/endpoint from the source agent appear in `config show`; setup reports no missing required terminal setup.

## Troubleshooting

- **`--dry-run and --yes are mutually exclusive`** → Run one at a time.
- **Conflict rows blocking apply** → Re-run with `--overwrite` after confirming you want to replace existing Gormes values.
- **`--dest` ignored** → `--dest` defaults to `GormesHome`. Set it explicitly only when targeting an unusual location.

## See also

- [Switch profiles for client work](../../operate/profiles-client-work/) — keep migrated state in its own profile.
- [Diagnose a broken install](../../troubleshooting/diagnose/)
