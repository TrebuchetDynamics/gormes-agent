---
title: "Blocked Slices"
weight: 40
aliases:
  - /building-gormes/blocked-slices/
---

# Blocked Slices

This page is generated from canonical `progress.json` rows that declare
`blocked_by`.

Use it to avoid assigning work before the dependency chain is ready.

<!-- PROGRESS:START kind=blocked-slices -->
| Phase | Slice | Blocked by | Ready when | Unblocks |
|---|---|---|---|---|
| 5 / 5.O | Gormes update bundled assets and skills sync | Gormes update verified binary swap and rollback | `Gormes update verified binary swap and rollback` is complete and exposes snapshot/rollback primitives for release installs., Release artifacts can include or point to a manifest with checksums for bundled assets and skills., Tests can fixture bundled skill roots, user-modified skill copies, removed manifest entries, asset checksum failures, and rollback without live network. | Gormes update managed service drain and restart |
| 5 / 5.O | Gormes update managed service drain and restart | Gormes update bundled assets and skills sync | `Gormes update bundled assets and skills sync` is complete and release updates can roll back binary/assets/skills coherently., Service restart and gateway status helpers are already tested through fake service/runtime seams., Tests can inject process tables, lock files, service states, drain timeouts, restart failures, health-check results, and rollback state without real systemd/launchd/sc.exe or live gateway. | Installed-runtime self-healing smoke, Gormes release operator trust report |
| 5 / 5.Q | Kernel cross-provider client swap for in-session model switch | Kernel in-session model-switch seam for the native TUI | Prerequisite row 'Kernel in-session model-switch seam for the native TUI' is complete (PlatformEventSetModel + k.sessionProvider exist)., A provider+model -> hermes.ModelRoute resolver path is identified in internal/hermes (BaseURL/APIMode/KeyEnv obtainable without a new catalog). | Native TUI /model slash command binding over the existing model picker |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all seven target GOOS/GOARCH combinations from CI or a Linux runner where cross-compilation is supported. | - |
| 8 / 8.D | Gormes streaming tool-trail status + spinner cadence wiring | Gormes-owned streaming feedback uplift | Gormes-owned streaming feedback uplift (R3) is complete (thinking indicator + collapsible output shipped). | - |
| 9 / 9.F | Navivox connect-and-talk first screen | Navivox HTTP/WS documentation refresh | `Navivox HTTP/WS documentation refresh` has landed so builders follow the current transport contract., `Navivox HTTP gateway connect-info command` is complete and prints token-redacted URLs., Flutter Navivox can build for web after the SSH key/drift dependency removal. | Navivox natural-language agent seed flow, Navivox structured tool event cards, Navivox safe config admin over HTTP, Navivox voice run records |
| 9 / 9.F | Navivox natural-language agent seed flow | Navivox connect-and-talk first screen | `Navivox connect-and-talk first screen` has landed so newly seeded agents can be tested immediately., Dynamic agent spawn/list/inspect/bind remains green and must not be replaced., The design preserves Gormes' server-authoritative config model; the Flutter app requests creation and renders/edit drafts, it does not write TOML. | Navivox per-agent BYO voice profiles |
| 9 / 9.F | Navivox structured tool event cards | Navivox connect-and-talk first screen | `Navivox connect-and-talk first screen` has landed so a streaming chat fixture exists., The gateway manager already has render-frame tool progress and channel-neutral fallback behavior., Flutter already has a basic NavivoxToolCall model and ToolCall body renderer to extend. | - |
| 9 / 9.F | Navivox safe config admin over HTTP | Navivox HTTP/WS documentation refresh, Navivox connect-and-talk first screen | `Navivox HTTP/WS documentation refresh` has landed so config admin docs describe HTTP/WS events, not stdio frames., `Navivox connect-and-talk first screen` has landed so the app has a live authenticated gateway connection., The config schema must be generated from Gormes config metadata or a typed allowlist; no free-form TOML editor. | Navivox per-agent BYO voice profiles |
| 9 / 9.F | Navivox voice run records | Navivox connect-and-talk first screen | `Navivox connect-and-talk first screen` has landed and produces session ids for normal turns., `Navivox structured tool event cards` is at least planned so tool timeline data has a typed event shape., The row uses existing session/store/apiserver surfaces where possible instead of creating a side database. | Navivox per-agent BYO voice profiles |
| 9 / 9.F | Navivox per-agent BYO voice profiles | Navivox natural-language agent seed flow, Navivox safe config admin over HTTP, Navivox voice run records | `Navivox natural-language agent seed flow` has landed so agents have editable draft/profile data., `Navivox safe config admin over HTTP` has landed so provider/secret changes use server-side validation and write-only secrets., `Navivox voice run records` has landed or is in progress so STT/TTS evidence can be audited. | - |
<!-- PROGRESS:END -->
