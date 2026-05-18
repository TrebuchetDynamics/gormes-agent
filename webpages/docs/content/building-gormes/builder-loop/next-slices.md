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
| 5 / 5.O | Profile workspace allow-list enforcement policy | Make `agents.defaults.workspaces` the Gormes-owned profile workspace allow-list, not just setup metadata. With an empty list, the default project workspace is the operator home. With a non-empty list, model-facing project read/write access is restricted to the normalized listed roots. Runtime internals may access the active profile root (`GORMES_HOME`) for config, auth, sessions, memory, skills, logs, cron, and gateway state, but model-facing tools must not treat the whole profile root as a project workspace. Model-facing profile edits are limited to explicit profile-owned content: identity files (`SOUL.md`, `IDENTITY.md` when present) and the active profile `skills/` directory. Profile-local `home/` is subprocess HOME/runtime state, not a broad project workspace. Sibling profiles, arbitrary operator-home paths, `.env`, `auth.json`, session/memory databases, logs, and other runtime state are denied as project paths. File tools, local/project execute_code, and coding-agent delegation must share one resolver. Local terminal must use a tested sandbox-capable backend for allow-listed roots or fail closed; merely setting cwd is not accepted as confinement. | operator, system | `Temp GORMES_HOME with a named profile containing `agents.defaults.workspaces = ["<project1>", "<project2>"]`, plus active-profile SOUL.md/IDENTITY.md/skills fixtures, profile secret/runtime-state fixtures, sibling-profile fixtures, and outside-root fixtures.` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 2 / 2.B.5 | Gateway memory monitor pressure policy | Port Hermes gateway memory monitor behavior into Gormes as a typed pressure policy that samples process memory, reports WARN/CRITICAL evidence, and can request bounded shutdown/restart action without killing unrelated operator processes. | - | `internal/gateway memory monitor fake sampler fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.H | ACP setup-browser bootstrap parity | `gormes acp --setup-browser` ports Hermes' ACP browser-tool bootstrap behavior with platform-specific command planning, dry-run/report output, and browser harness dependency checks while keeping actual installs explicit and operator-approved. | - | `cmd/gormes acp setup-browser dry-run fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.L | Hermes LSP write-time semantic diagnostics | After `write_file` or `patch`, Gormes runs a language-server diagnostic pass equivalent to Hermes' write-time LSP surface, shifts baseline ranges through edits, and returns new semantic errors to the agent without blocking unsupported languages. | - | `internal/tools LSP diagnostic fake-server fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Hermes x_search tool and auth surface | Expose Hermes' first-class `x_search` tool in Gormes with a descriptor, OAuth/API-key auth status, query/result envelope, rate-limit/degraded errors, and registry/toolset visibility without requiring live X credentials in tests. | - | `internal/tools x_search fake transport fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Hermes send command stdin/file payload parity | `gormes send` preserves Hermes `hermes send` behavior for stdin/file payload decoding, binary/invalid-text rejection, newline preservation, session targeting, dry/no-agent modes, and TUI resume safety without leaking raw control sequences into terminal output. | - | `cmd/gormes send command tests against Hermes send_cmd fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Hermes session recap command surface | Port Hermes' session recap command as a Gormes-native read-only session summarizer over local session/transcript storage, preserving output modes, missing-session diagnostics, and provider-free degraded behavior. | - | `cmd/gormes session recap fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Long-term plan: profile fleet supervisor and single control-plane gateway | Define Gormes' long-term profile-fleet runtime so operators get one control surface for all named profiles while preserving Hermes-compatible profile state separation. The near-term per-profile gateway services remain a compatibility bridge; the target is a fleet supervisor that can enumerate configured profiles, start/stop/restart profile-scoped workers or a proven profile-scoped in-process equivalent, validate token ownership, surface per-profile health, and coordinate update/restart-all flows without sharing config, auth, sessions, memory, tool state, or kernels across profiles. | operator, gateway, system | `internal/gateway/fleet_supervisor_test.go; cmd/gormes/gateway_fleet_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | Native TUI Terminal.app truecolor and ANSI sanitizer parity | Port Hermes Ink TUI Terminal.app/truecolor and ANSI sanitizer behavior into the native Gormes TUI so renderer output keeps cursor/source-of-truth stability, strips malformed CSI safely, and preserves readable color behavior across modern terminals. | - | `internal/tui Terminal.app/ANSI sanitizer fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.C | Hermes v0.14 optional skill catalog refresh | Refresh the Gormes skill catalog and metadata compatibility checks against Hermes v0.14 optional skills, including devops/pinggy-tunnel, research/darwinian-evolver, research/osint-investigation, and the updated Notion skill, without blindly copying unsupported Python scripts into runtime packages. | - | `internal/skills optional skill catalog fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
