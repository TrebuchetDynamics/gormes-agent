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
| 5 / 5.N | Hermes send_message tool list and target contract | Bring Gormes' existing `send_message` tool descriptor/handler up to the narrow Hermes contract that can be proven without live channel sends: the schema must expose `action` with `send\|list`, optional `target`, and optional `message`; `action=list` must return a typed list/unavailable envelope from an injected channel-directory provider; `action=send` must reject missing target/message with Hermes-style tool-error JSON, parse `platform[:chat[:thread]]` targets through the shared gateway delivery-target parser, and call an injected sender only after validation. This row must not start gateway services, contact Telegram/Discord/Slack, or implement media delivery. | operator, gateway, child-agent, system | `internal/tools/sendmessage/send_message_test.go::TestSendMessageToolListAndValidatedSendContract` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
