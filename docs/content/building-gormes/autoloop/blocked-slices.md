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
| 2 / 2.B.5 | BlueBubbles iMessage session-context prompt guidance | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound formatting splits blank-line paragraphs into separate iMessage sends, so prompt guidance has a matching delivery contract. | - |
| 2 / 2.B.11 | Discord SessionSource guild/parent/message evidence | Discord forum channel ingress + thread lifecycle | Discord forum ingress and thread lifecycle fixtures are green on top of the shared gateway InboundEvent envelope. | Discord forum media + polish parity, Manager remember-source hook, Discord tool split + platform-scoped toolsets |
| 2 / 2.F.3 | Pairing approval + rate-limit semantics | Pairing read-model schema + atomic persistence | Pairing read-model schema + atomic persistence is complete and exposes fake-clock-friendly pending/approved state., Telegram DM events can be represented through gateway.InboundEvent or a tiny gateway-owned resolver fixture without importing python-telegram-bot. | Unauthorized DM pairing response contract, `gormes gateway status` read-only command |
| 2 / 2.F.3 | Unauthorized DM pairing response contract | Pairing approval + rate-limit semantics | Pairing approval, rate limiting, allowlist checks, and Telegram missing-from_user fallback are fixture-locked in internal/gateway/pairing_approval_test.go. | - |
| 2 / 2.F.5 | Steer slash command registry + queue fallback | 2.E.2 | 2.E.2 is complete and the shared CommandDef registry is stable for gateway commands. | Mid-run steer injection between tool calls, Gateway-handled slash commands bypass active-session guard |
| 4 / 4.A | Bedrock stale-client eviction + retry classification | Bedrock SigV4 + credential seam | A Bedrock client/cache seam exists behind the provider adapter and can be exercised without live AWS credentials. | - |
| 4 / 4.A | Codex OAuth state + stale-token relogin | Token vault, Multi-account auth, Codex Responses pure conversion harness | Gormes has an XDG-scoped token vault and account-selection seam for provider credentials. | - |
| 4 / 4.G | Anthropic OAuth/keychain credential discovery | Token vault | Token vault owns XDG-scoped credential files and can expose provider auth status without live credentials. | - |
| 4 / 4.H | Unsupported temperature retry + Codex flush guard | Classified provider-error taxonomy, Provider interface + stream fixture harness | Classified provider-error taxonomy and provider transcript fixtures are validated., Gormes has ChatRequest.Temperature and can exercise fake HTTP providers without live credentials. | OpenRouter, Codex OAuth state + stale-token relogin, Provider rate guard + budget telemetry |
| 5 / 5.A | Tool parity manifest refresh for Hermes b35d692f | Tool registry inventory + schema parity harness | Tool registry inventory + schema parity harness is validated and can load the embedded manifest without porting handlers. | Discord tool split + platform-scoped toolsets, Cron context_from output chaining, Platform toolset config persistence + MCP sentinel |
| 5 / 5.A | Discord tool split + platform-scoped toolsets | Tool parity manifest refresh for Hermes b35d692f, Discord SessionSource guild/parent/message evidence | Tool parity manifest refresh for Hermes b35d692f is complete., Discord SessionSource guild/parent/message evidence is fixture-locked so current-message IDs can feed Discord actions later. | - |
| 5 / 5.J | Subagent dangerous-command non-interactive approval policy | Dangerous-command detector + blocked-result schema, Approval mode config normalization | Dangerous-command detection and approval-mode config normalization are fixture-locked for local tools. | - |
| 5 / 5.N | Cron context_from output chaining | Cronjob tool API + schedule parser parity | Cronjob tool API + schedule parser parity has a create/update/list surface over the Go cron store, or this slice owns the minimal ContextFrom field and prompt-builder fixture without exposing a public tool yet. | - |
| 5 / 5.O | Busy command guard for compression and long CLI actions | CLI command registry parity + active-turn busy policy | The CLI command registry has a shared active-turn/busy policy surface. | - |
| 5 / 5.O | Platform toolset config persistence + MCP sentinel | Tool parity manifest refresh for Hermes b35d692f | Tool parity manifest refresh for Hermes b35d692f captures current donor toolset names and platform restrictions., Config, profile, auth, and setup command surfaces has established the config file read/write seam or this slice owns a pure helper under internal/cli without command wiring. | - |
| 5 / 5.P | Unix installer root/FHS layout policy | Unix installer (install.sh) source-backed update flow | Unix installer (install.sh) source-backed update flow has canonical scripts under scripts/ and a byte-equal served site copy. | Installer site asset/route coverage |
| 5 / 5.Q | Dashboard PTY chat sidecar contract | PTY bridge protocol adapter, SSE streaming to Bubble Tea TUI | PTY bridge behavior and TUI gateway event streaming are each fixture-locked. | - |
| 6 / 6.C | Portable SKILL.md format | Phase 2.G skills runtime | Phase 2.G skills runtime is complete and the parser/store seam is stable enough for versioned metadata. | LLM-assisted pattern distillation, Hybrid lexical + semantic lookup, Skill effectiveness scoring |
| 7 / 7.C | Matrix E2EE device-id crypto-store binding | Matrix real client/bootstrap layer | Matrix real client/bootstrap layer has auth, sync/invite handling, room-kind policy, and a fakeable E2EE bootstrap seam. | - |
<!-- PROGRESS:END -->
