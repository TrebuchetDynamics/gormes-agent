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

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 2 / 2.F.4 | Channel-neutral native runtime turn adapter | Telegram, Slack, Discord, WhatsApp, BlueBubbles, and future channels enter the same native Gormes turn adapter so provider/runtime fixes preserve Hermes channel parity instead of hard-coding Telegram behavior. | gateway, operator, system | `internal/gateway/channel_neutral_turn_adapter_test.go` | P0 handoff; needs contract proof before closeout. |
| 4 / 4.I | Native runtime provider gateway binding | Gormes gateway constructs a native Go runtime/provider binding from Hermes-compatible config when no explicit endpoint is configured, so live Telegram and other channel turns do not default to a dead localhost backend while explicit OpenAI-compatible endpoints remain supported. | gateway, operator, system | `internal/runtime/native_provider_gateway_binding_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.B | Environment interface + file sync contract | Gormes ports Hermes sandbox environment and file-sync contracts into a Go Environment interface with path mapping, upload/download, timeout, cleanup, and parser-family inventory fixtures before backend-specific Docker/SSH/Modal/Daytona/Singularity execution lands. | operator, child-agent, system | `internal/tools/environment_contract_test.go; internal/hermes/tool_call_parser_manifest_test.go` | Unblocks Docker, Modal, Daytona, Singularity, Raw tool-call parser fixture matrix, Terminal snapshot source stdout suppression guard. |
| 5 / 5.H | ACP server side | Gormes maps Hermes ACP adapter entry/auth/session/tools/permissions/events into a Go-native manifest and stdio/server protocol fixture before editor integrations are advertised. | operator, system | `internal/acp/server_manifest_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.P | OCI image | Gormes ships an OCI image contract that mirrors upstream Docker entrypoint/config volume operational behavior while proving the final image contains the Go binary and no required Python runtime path. | operator, system | `docs/install/oci_image_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.P | Homebrew | Gormes ports Hermes Homebrew/release artifact expectations into a Go-native formula fixture with version, checksum, binary install layout, and doctor smoke contract. | operator, system | `docs/install/homebrew_formula_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
