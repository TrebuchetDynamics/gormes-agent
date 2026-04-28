---
title: "Phase 7 — Paused Channel Backlog"
weight: 80
---

# Phase 7 — Paused Channel Backlog

**Status:** 🔨 paused/inventory

**Deliverable:** Long-tail gateway adapters and platform-specific polish after
the priority gateway set and shared delivery policy are stable.

**Completion lane:** Phase 7 belongs to [Lane 4 — Gateway, Channels, Cron, And Delivery](../lane-roadmap/#lane-4--gateway-channels-cron-and-delivery).
It must not outrun the shared gateway contracts in Phase 2: session context,
delivery routing, command policy, active-turn semantics, pairing, home channel,
status, restart, and cron handoff.

## Rule For Reopening A Paused Channel

A Phase 7 row may move back into active work only when:

1. the shared gateway surface it depends on is shipped;
2. the row names the exact upstream adapter behavior;
3. fake payload fixtures prove ingress, reply target, command handling, and
   degraded status without live platform credentials;
4. write scope is limited to the adapter package, config, and focused tests;
5. live transport/bootstrap work is split from pure normalization and policy
   helpers.

## Adapter Order

Prefer adapters that reuse already-shipped shared contracts:

1. webhook/API-style adapters with signed ingress and fake payloads;
2. thread-text adapters that use the shared canonical thread ID contract;
3. regional bot platforms with shared-bot ingress policy;
4. media-heavy or SDK-heavy adapters after delivery, cache, and status
   contracts are stable.

## Builder Handoff

Use:

- `gormes-parity-auditor` to compare the upstream adapter file against current
  `internal/channels/*` or missing packages;
- `gormes-planner` to split adapter work into pure ingress/reply/config/status
  rows before transport rows;
- `gormes-builder` and `gormes-tdd-slice` for one adapter behavior at a time.

Do not combine SDK bootstrap, credential UX, media upload, and session-context
policy in one row.
