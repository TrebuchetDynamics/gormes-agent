---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 2 / 2.A | Coding-agent delegation: Phase 1 scaffold (internal/codingagents) | Shared internal/codingagents package providing the CodingAgent interface, CodingAgentRequest/Result, mode constants, binary availability detection, workspace guard with default deny list, git snapshot/diff helper, and prompt wrapper. No tools are registered in this slice; adapters and registry exposure land in later phases. | operator, system | `internal/codingagents` | Already active; contract metadata keeps execution bounded. |
| 1 / 5.X | Termux storage and path safety audit | Audit and test Gormes path selection under synthetic Termux env so config, dotenv secrets, sessions, gateway state, SQLite/Goncho, browser temp dirs, and generated files land only under configured GORMES_HOME/XDG/HOME locations while install publication remains $PREFIX/bin/gormes. No runtime code may hardcode desktop workspace paths such as /home/xel or workspace-mineru. | operator, system | `internal/config Termux path fixtures plus cmd/gormes doctor/config/goncho smoke fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux gateway foreground tmux lifecycle | Gateway lifecycle commands and docs present a Termux-specific foreground/tmux model: Telegram/Discord/Slack gateways are supported from a foreground shell or tmux session, systemd/Windows service assumptions are not advertised, and doctor/status guidance names termux-wake-lock plus Android battery settings as best-effort survival aids. The implementation must preserve the same gateway command names and JSON contracts as desktop Linux. | operator, gateway, system | `cmd/gormes gateway/doctor fixtures under synthetic Termux env` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux notification bridge via termux-api | Add an optional Termux notification adapter that shells out to termux-notification only when Termux and the command are detected. Gateway/long-run status can emit Android notifications through this adapter, while non-Termux hosts and Termux hosts without Termux:API degrade to structured no-op/WARN evidence. The adapter must redact secrets and never make termux-api a hard dependency. | operator, gateway, system | `internal/gateway or internal/tools Termux notification adapter tests with fake exec runner` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux real-device smoke evidence | Capture a dated real-device no-root Android Termux smoke record for the current release: install via repo-root install.sh release asset, run gormes version, gormes doctor --offline --json, gormes config check, initialize SQLite/Goncho state, and run a provider-backed gormes chat -q "hello from Termux" when a test credential is available. The evidence must record Android/Termux versions, device arch, install method, and any caveats without leaking credentials. | operator, system | `webpages/docs/content/install/termux-smoke.md or release evidence note` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux remote execution guidance | Document and, where useful, add setup/status guidance for using Termux Gormes as the mobile operator/controller while SSHing to stronger machines for heavy builds, Docker, local browser automation, and GPU/local model inference. The guidance must preserve PC-like local Gormes CLI behavior while making remote execution the credible path for workstation/server workloads. | operator, system | `webpages/docs/content/install/ Termux remote-execution docs` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
