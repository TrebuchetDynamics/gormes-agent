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
| 3 / 3.G | Goncho webhook delivery retry worker contract | Goncho ports Honcho outbound webhook delivery as a local fake-HTTP worker contract through WebhookDeliveryWorker, WebhookDeliveryStore, WebhookHTTPClient, WebhookClock, WebhookDeliveryAttempt, and WebhookDeliveryResult types with queue-empty/test events, HMAC headers, retry/backoff/failure evidence, and workspace endpoint disablement without requiring Redis, hosted Honcho, or live network tests. | operator, system | `internal/goncho/webhook_delivery_test.go` | Unblocks Self-monitoring telemetry. |
| 4 / 4.B | ContextEngine compression-boundary callback vocabulary | internal/hermes defines a compression-boundary callback vocabulary on ContextEngine with stable lineage evidence and status fields, without binding kernel compression execution yet | operator, system | `internal/hermes/context_engine_boundary_test.go::TestContextEngineCompressionBoundaryVocabulary` | Unblocks ContextEngine compression-boundary notification. |
| 4 / 4.H | Provider account usage read model + renderer | Gormes exposes a Hermes-compatible provider account-usage read model with internal/hermes.AccountUsageSnapshot and AccountUsageWindow, fakeable Codex/Anthropic/OpenRouter fetchers, and a shared redacted renderer that later CLI/status and gateway `/usage` command rows can bind without rediscovering provider payload semantics. | operator, gateway, system | `internal/hermes/account_usage_test.go` | Unblocks Provider status command parity, Gateway /usage command binding over provider account usage, Self-monitoring telemetry. |
| 5 / 5.B | Environment interface + file sync contract | Gormes ports Hermes sandbox environment and file-sync contracts into a Go Environment interface with path mapping, upload/download, timeout, cleanup, and parser-family inventory fixtures before backend-specific Docker/SSH/Modal/Daytona/Singularity execution lands. | operator, child-agent, system | `internal/tools/environment_contract_test.go; internal/hermes/tool_call_parser_manifest_test.go` | Unblocks Docker, Modal, Daytona, Singularity, Raw tool-call parser fixture matrix, Terminal snapshot source stdout suppression guard. |
| 5 / 5.D | Image input mode router + native content parts | internal/hermes exposes a pure image input routing helper that resolves agent.image_input_mode auto/native/text from model vision capability and auxiliary vision override, then builds native provider content parts with text plus data-url image_url entries without invoking a live provider | operator, system | `internal/hermes/image_routing_test.go::TestImageInputRouting_*` | Unblocks Image-too-large shrink retry helper. |
| 5 / 5.H | ACP server side | Gormes maps Hermes ACP adapter entry/auth/session/tools/permissions/events into a Go-native manifest and stdio/server protocol fixture before editor integrations are advertised. | operator, system | `internal/acp/server_manifest_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Hermes CLI command-tree parity manifest | cmd/gormes owns a source-backed Hermes CLI compatibility manifest that enumerates every upstream top-level command, nested subcommand, global/root flag, slash command, gateway command handler, dynamic plugin-provided command, and Gormes-owned addition, then classifies each path as implemented, row-backed, owned, excluded, or not-yet-applicable before any handler work claims CLI parity. | operator, gateway, system | `cmd/gormes/hermes_cli_parity_test.go::TestHermesCLIParityManifest` | Unblocks Gormes config command surface, Gormes config edit/check/native schema-migrate closeout, Hermes config migration dry-run manifest, OpenClaw migration dry-run manifest, Gateway, platform, webhook, and cron management CLI, Diagnostics, backup, logs, and status CLI. |
| 5 / 5.O | Provider endpoint/API-key root flags + runtime resolution | cmd/gormes accepts --endpoint, --api-key, --model, and --provider as invocation-only overrides for oneshot and TUI startup; flag values win over env/config for the current process, --api-key is never persisted, and all status/error evidence redacts the secret value. | operator, system | `cmd/gormes/provider_flag_resolution_test.go` | Unblocks Gormes config command surface, Hermes config migration writer, OpenClaw migration writer and cleanup command. |
| 5 / 5.O | Backup/update opt-in and exclusion policy | CLI backup/update policy defaults pre-update backups off unless explicitly requested, honors --no-backup over --backup, and excludes checkpoints plus SQLite WAL/SHM/journal sidecars from backup manifests | operator, system | `internal/cli/backup_policy_test.go::TestBackupPolicy_*` | Unblocks Backup manifest dry-run contract. |
| 5 / 5.O | Custom provider model-switch key_env write guard | internal/cli exposes a pure model-switch patch helper that accepts an in-memory custom provider ref plus a target model and returns the config patch/evidence for default_model changes while preserving original credential storage: providers that relied on key_env and had no inline api_key/api_key_ref must not gain an api_key entry, while providers that already had inline plaintext or `${VAR}` api_key may keep that existing value without writing resolved plaintext | operator, system | `internal/cli/custom_provider_model_switch_test.go::TestCustomProviderModelSwitchPatch_*` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
