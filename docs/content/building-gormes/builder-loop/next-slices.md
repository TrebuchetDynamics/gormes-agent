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
| 4 / 4.E | Self-monitoring telemetry | Gormes bridges Hermes turn/provider/tool telemetry and Honcho telemetry/reasoning traces into local redacted telemetry, audit, and insights evidence through SelfMonitoringBridge, TelemetryEventMatrix, ReasoningTraceRecord, TelemetrySink, AuditSink, and InsightsRecorder interfaces without changing the local usage.jsonl schema until compatibility tests pass. | operator, system | `internal/telemetry/self_monitoring_test.go; internal/goncho/telemetry_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.B | Environment interface + file sync contract | Gormes ports Hermes sandbox environment and file-sync contracts into a Go Environment interface with path mapping, upload/download, timeout, cleanup, and parser-family inventory fixtures before backend-specific Docker/SSH/Modal/Daytona/Singularity execution lands. | operator, child-agent, system | `internal/tools/environment_contract_test.go; internal/hermes/tool_call_parser_manifest_test.go` | Unblocks Docker, Modal, Daytona, Singularity, Raw tool-call parser fixture matrix, Terminal snapshot source stdout suppression guard. |
| 5 / 5.H | ACP server side | Gormes maps Hermes ACP adapter entry/auth/session/tools/permissions/events into a Go-native manifest and stdio/server protocol fixture before editor integrations are advertised. | operator, system | `internal/acp/server_manifest_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Backup/update opt-in and exclusion policy | CLI backup/update policy defaults pre-update backups off unless explicitly requested, honors --no-backup over --backup, and excludes checkpoints plus SQLite WAL/SHM/journal sidecars from backup manifests | operator, system | `internal/cli/backup_policy_test.go::TestBackupPolicy_*` | Unblocks Backup manifest dry-run contract. |
| 5 / 5.P | OCI image | Gormes ships an OCI image contract that mirrors upstream Docker entrypoint/config volume operational behavior while proving the final image contains the Go binary and no required Python runtime path. | operator, system | `docs/install/oci_image_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.P | Homebrew | Gormes ports Hermes Homebrew/release artifact expectations into a Go-native formula fixture with version, checksum, binary install layout, and doctor smoke contract. | operator, system | `docs/install/homebrew_formula_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 7 / 7.E | Yuanbao protocol envelope + markdown fixtures | Gormes parses Yuanbao websocket/protobuf-style envelopes and Markdown message fragments into gateway-neutral events using fixture data only | gateway, system | `internal/channels/yuanbao/proto_test.go` | Unblocks Yuanbao media/sticker attachment normalization, Yuanbao gateway runtime + toolset registration. |
<!-- PROGRESS:END -->
