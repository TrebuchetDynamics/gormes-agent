---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Hermes send_message tool list and target contract

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Bring Gormes' existing `send_message` tool descriptor/handler up to the narrow Hermes contract that can be proven without live channel sends: the schema must expose `action` with `send\|list`, optional `target`, and optional `message`; `action=list` must return a typed list/unavailable envelope from an injected channel-directory provider; `action=send` must reject missing target/message with Hermes-style tool-error JSON, parse `platform[:chat[:thread]]` targets through the shared gateway delivery-target parser, and call an injected sender only after validation. This row must not start gateway services, contact Telegram/Discord/Slack, or implement media delivery.
- Trust class: operator, gateway, child-agent, system
- Ready when: Gormes already has a minimal `internal/tools/sendmessage` package and gateway delivery-target parsing, but the parity atom still says missing because the handler lacks Hermes `action=list`, optional schema, target guidance, and fail-closed no-backend evidence., The row can be tested with injected fake directory/sender implementations and JSON fixtures only; no live channel credentials, gateway process, or provider call is required., Top-level `gormes send` CLI payload parsing is already complete and remains out of scope except as source evidence for body/target boundaries.
- Not ready when: The builder wires live Telegram/Discord/Slack clients, starts gateway processes, implements media attachment delivery, or changes cron delivery behavior in this slice., The tool silently succeeds when no sender is configured, accepts malformed targets without evidence, or requires a target/message for `action=list`., The implementation changes model-visible names outside the existing `send_message` tool or exposes private channel directories/secrets in list output.
- Degraded mode: When no directory or sender is injected, `send_message` returns typed `send_message_directory_unavailable` or `send_message_backend_unavailable` evidence instead of silently succeeding. Unknown/ambiguous human channel names should tell the model to call `send_message(action="list")` before sending, preserving Hermes guidance without live network access.
- Fixture: `internal/tools/sendmessage/send_message_test.go::TestSendMessageToolListAndValidatedSendContract`
- Write scope: `internal/tools/sendmessage/send_message.go`, `internal/tools/sendmessage/send_message_test.go`, `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `docs/content/building-gormes/architecture_plan/progress.json/modules/tools.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/tools.json`
- Test commands: `go test ./internal/tools/sendmessage -run 'TestSendMessageTool' -count=1`, `go test ./internal/tools -run 'TestSendMessage' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: `send_message` is no longer a two-field optimistic stub: list/send action semantics, target validation, fail-closed unavailable evidence, and source-backed partial parity are all fixture-proven without live channel sends.
- Acceptance: Schema tests prove `action` enum includes `send` and `list`, and `target`/`message` are optional at schema level so `action=list` is valid., `action=list` with an injected fake directory returns deterministic JSON targets; without a directory it returns typed unavailable evidence and no panic., `action=send` rejects missing target/message with tool-error JSON, parses platform/chat/thread targets via shared gateway parsing, and calls the fake sender exactly once for a valid target., Unknown or unresolved friendly channel names return guidance to call `send_message(action="list")` and do not call the sender., The parity atom moves from stale missing to partial, naming the remaining live adapter/media-delivery gaps.
- Source refs: hermes-agent/tools/send_message_tool.py:SEND_MESSAGE_SCHEMA, hermes-agent/tools/send_message_tool.py:send_message_tool, hermes-agent/tools/send_message_tool.py:_handle_list, hermes-agent/tools/send_message_tool.py:_handle_send, hermes-agent/tools/send_message_tool.py:_parse_target_ref, internal/tools/sendmessage/send_message.go, internal/tools/sendmessage/send_message_test.go, internal/gateway/delivery.go:ParseDeliveryTarget, internal/automation/cron/delivery_plan.go:resolveCronDeliveryTarget, docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md:Send-message tool, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Operator tools
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
