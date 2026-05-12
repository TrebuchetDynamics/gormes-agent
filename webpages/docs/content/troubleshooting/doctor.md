---
title: "Doctor"
description: "Verify Gormes runtime: provider readiness, built-in tools, gateway channels, memory, and browser harness."
aliases:
  - /using-gormes/wire-doctor/
---

# Doctor

`gormes doctor` validates the local stack before a live turn burns tokens. It checks build identity, secret runtime, provider readiness, the native TUI, the tool registry, web tools, the browser runtime, the ACP bridge, GitHub auth, Goncho memory config, and each configured gateway channel.

## Online run

```bash
gormes doctor
```

Adds a provider health probe (2-second timeout) against the configured endpoint on top of the offline checks. Exits non-zero on failure, so you can wire it into scripts.

## Offline run

```bash
gormes doctor --offline
```

Skips the provider health probe. Useful in CI, pre-flight, or any time you want to verify the local toolbox, gateway config, and Goncho state without a live backend.

## JSON output

```bash
gormes doctor --json
gormes doctor --offline --json
```

Emits a machine-readable `{build, failed, checks: [...]}` document suitable for fleet-health monitoring.

## Reading the output

Each line starts with a status tag:

- `[PASS]` — check succeeded.
- `[SKIP]` — check was skipped (for example, provider health under `--offline`, or a disabled channel).
- `[WARN]` — the runtime can keep going, but a follow-up is recommended.
- `[FAIL]` — a hard problem; doctor exits non-zero.

A typical offline run looks like:

```
[WARN] build identity: dirty build: version=0.2.8 commit=91b6a6950 — uncommitted source at build time
[PASS] SecretRef runtime: resolved=0 inactive=0 unavailable=0
[SKIP] provider health: skipped (--offline)
[PASS] Native TUI: available: Go-native Bubble Tea TUI compiled into gormes
[PASS] Toolbox: 35 tools registered (...)
[PASS] Web tools: backend=duckduckgo route=direct source=free evidence=web_ok
[WARN] Browser runtime: cdp_not_configured
[WARN] ACP bridge: server=ready client=ready remote=unsupported evidence=acp_bridge_unavailable
[WARN] GitHub auth: No GITHUB_TOKEN and gh auth status failed evidence=github_cli_status_failed
[PASS] Goncho config: enabled=true workspace=gormes observer_peer=gormes
[WARN] Gateway Slack: disabled
[WARN] Custom endpoint: configured provider=openai endpoint=... missing=api_key
[PASS] gateway/telegram: allowed_chat_id=... allowed_users=1
[SKIP] gateway/discord: disabled
```

A few specifics worth knowing:

- **build identity** is `WARN` whenever the binary was built from a dirty tree or without ldflags-injected commit metadata. A clean release artifact reports `PASS`.
- **Browser runtime** flags `WARN cdp_not_configured` until you point `BROWSER_CDP_URL` (or `CHROME_REMOTE_DEBUGGING_URL`) at a Chrome instance launched with `--remote-debugging-port=9222`.
- **GitHub auth** flags `WARN` until either `GITHUB_TOKEN`/`GH_TOKEN` is set or `gh auth status` succeeds.
- **Custom endpoint** flags `WARN` until the configured provider has an API key in `.env`.
- **gateway/<channel>** flags `SKIP disabled` for any channel that is not configured.

## Related diagnostics

```bash
gormes goncho doctor --json   # inspect local Goncho/Honcho memory storage
gormes gateway status         # confirm what is connected before treating a channel as live
gormes config check           # validate config schema
gormes config show            # show config with secrets redacted
```
