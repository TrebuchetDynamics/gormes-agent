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
| 9 / 9.F | Navivox natural-language profile seed Flutter UI | Add the sibling Navivox Flutter profile seed UI that calls the Gormes backend profile-seed API, offers Create from seed in the chat/profile flow, renders the returned editable draft fields, requires explicit workspace path entry or confirmation, applies only through the backend, and then shows the new profile as a contact. The Flutter app must not write TOML or infer/grant workspace roots on its own. | operator, gateway, system | `../navivox-app/test/features/profiles/profile_seed_flow_test.dart` | Unblocks Navivox per-profile BYO voice profiles. |
| 9 / 9.F | Navivox per-profile BYO voice profiles Flutter UI | Add the sibling Navivox Flutter profile/config controls that consume the backend voice-profile contract without writing config files or storing raw secrets. | - | `../navivox-app/test/features/profiles/; ../navivox-app/test/features/config/` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 9 / 9.F | Navivox safe config admin Flutter UI | Render the Navivox config admin backend contract in the sibling Flutter app: schema-driven controls, redacted current values, diff/validate/apply confirmation, secret set/rotate/delete/test actions, and reload-or-pending-restart status. Flutter consumes backend schema and actions only; it never edits config.toml, .env, or raw secret values directly. | operator, gateway, system | `../navivox-app/test/features/config/config_screen_test.dart` | Unblocks Navivox per-profile BYO voice profiles. |
| 9 / 9.F | Navivox structured tool event cards Flutter UI | The sibling Navivox Flutter app consumes the Gormes backend structured tool-progress contract, upserts one durable ToolCallCard per tool_call_id for started/updated/finished states, renders redacted artifact rows, and never converts tool events into assistant prose. | operator, gateway, system | `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart; ../navivox-app/test/features/chat/` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 9 / 9.F | Navivox voice run records Flutter inspection UI | Render the sibling Flutter Navivox inspection surface for backend run records after the Gormes run-record API lands. The app should fetch or receive redacted run records, show text and voice transcript evidence, STT/TTS metadata, tool timeline cards, attachment/artifact refs, terminal status, and provider usage/cost with explicit unknown states. This row is intentionally cross-root and must not be selected during repo-only Gormes iterations. | operator, gateway, system | `../navivox-app/test/features/chat/navivox_run_records_test.dart` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
