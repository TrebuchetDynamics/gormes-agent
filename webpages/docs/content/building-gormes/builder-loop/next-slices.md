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
| 1 / 5.X | Termux real-device smoke evidence | Capture a dated real-device no-root Android Termux smoke record for the current release: install via repo-root install.sh release asset, run gormes version, gormes doctor --offline --json, gormes config check, initialize SQLite/Goncho state, and run a provider-backed gormes chat -q "hello from Termux" when a test credential is available. The evidence must record Android/Termux versions, device arch, install method, and any caveats without leaking credentials. | operator, system | `webpages/docs/content/install/termux-smoke.md or release evidence note` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux remote execution guidance | Document and, where useful, add setup/status guidance for using Termux Gormes as the mobile operator/controller while SSHing to stronger machines for heavy builds, Docker, local browser automation, and GPU/local model inference. The guidance must preserve PC-like local Gormes CLI behavior while making remote execution the credible path for workstation/server workloads. | operator, system | `webpages/docs/content/install/ Termux remote-execution docs` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Gormes setup model step uses the dynamic provider-tracked model picker | The `gormes setup` Inference Provider section must present, for the operator's selected/active provider, the same dynamic per-provider callable-model list the `gormes model` and gateway/native-TUI `/model` pickers already use (internal/hermes.ListPickerProviders), instead of the legacy bare free-text prompt with at most five static suggestions. Two coupled defects: (1) the model prompt provider must equal the provider just selected/active in the section — the transcript witness shows Active provider "OpenAI Codex" but the prompt reads "Model for openrouter [gpt-5.5]", because runSetupInferenceProviderSection does not carry the chosen/active provider into runSetupActiveProviderModelPicker (it resolves to the provider-catalog default, openrouter); (2) cmd/gormes/model.go:promptModelChoice is fed defaultModelCatalogSuggestions(provider) = hermes.ProviderModelCatalogSuggestions(provider, nil) — the nil disables the live/dynamic catalog, so the operator never sees the provider's actual model set. The fix wires the existing ListPickerProviders-backed picker (the port already consumed by internal/gateway/model_picker.go and internal/tui/slash_model.go) into the setup model step with provider continuity, while preserving q/cancel, Enter-keeps-current, and `gormes setup model --non-interactive` default behavior. This supersedes the deliberately-static promptModelChoice seam introduced by the completed row at progress.json:19063; it must not fork a second picker or change the already-complete gateway/`gormes model`/TUI picker behavior. | - | `cmd/gormes/setup_model_picker_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
