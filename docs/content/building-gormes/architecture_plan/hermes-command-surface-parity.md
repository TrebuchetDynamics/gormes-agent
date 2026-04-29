---
title: "Hermes Command Surface Parity Matrix"
description: "Operator-visible Hermes CLI command surface, Gormes implementation state, and progress.json ownership for parity backlog planning."
weight: 96
---

# Hermes Command Surface Parity Matrix

This page records the operator-visible Hermes CLI command surface that Gormes
must preserve while remaining a native Go runtime. It complements the narrower
[Hermes Auth CLI Parity Manifest](../hermes-auth-cli-parity/) and does not
replace the canonical backlog.

## Canonical backlog and proof files

- Canonical backlog: `docs/content/building-gormes/architecture_plan/progress.json`.
- CLI parity backlog home: Phase `5`, subphase `5.O`, `Hermes CLI Parity`.
- Provider/auth runtime backlog home: Phase `4`, especially subphases `4.A`,
  `4.G`, and `4.H` for provider bindings, token vault/auth, and provider-error
  behavior.
- Executable CLI parity manifest: `cmd/gormes/hermes_cli_parity.go`.
- Executable CLI parity tests: `cmd/gormes/hermes_cli_parity_test.go`.
- Auth command manifest: `docs/content/building-gormes/architecture_plan/hermes-auth-cli-parity.md`.
- Feature map rule: `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md` says missing P0/P1 Hermes behavior must become or refine a `progress.json` row, not live as a side-channel TODO.

If this page disagrees with `progress.json`, fix `progress.json` first and then
update or regenerate derived docs. This page is an operator-readable matrix; the
machine-readable work queue remains `progress.json`.

## Audit answer: do we have every command?

For source-backed inventory, yes: the executable manifest now captures every
current Hermes top-level argparse command, nested parser command, nested alias,
gateway message handler, slash command/alias, dynamic plugin command class, and
Gormes-owned divergence that this audit found. This does **not** mean Gormes
implements every handler. Unsupported commands remain visible as `Row-backed`,
`Excluded/deprecated`, or `Gormes-owned` so builders can ship them deliberately.

The drift gate is now explicit in `cmd/gormes/hermes_cli_parity_test.go`:
`TestHermesCLIParityManifestNestedParserInventoryMatchesHermes` locks the
current nested parser inventory from `../hermes-agent/hermes_cli/main.py` and
fails if stale paths such as `gateway reset`, `cron enable`, `webhook serve`,
`mcp call`, `profile set`, `auth login`, or `auth refresh` are advertised as
active parser commands.

## Source capture

Hermes source paths inspected for this matrix:

- `../hermes-agent/hermes_cli/main.py`
- `../hermes-agent/hermes_cli/auth.py`
- `../hermes-agent/hermes_cli/auth_commands.py`
- `../hermes-agent/agent/credential_pool.py`
- `../hermes-agent/gateway/run.py`

Gormes source and runtime probes inspected for this matrix:

- `cmd/gormes/hermes_cli_parity.go`
- `cmd/gormes/hermes_cli_parity_test.go`
- `go run ./cmd/gormes --help`
- `go run ./cmd/gormes auth --help`
- `go run ./cmd/gormes gateway --help`
- `go run ./cmd/progress validate`

Gormes-reference donor anchors consulted for provider/channel/tool planning:

- `../references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md`
- `../references/go-agent-os/goclaw/internal/oauth/openai.go`
- `../references/go-agent-os/goclaw/internal/oauth/token.go`
- `../references/go-agent-os/goclaw/internal/oauth/openai_quota_transport.go`
- `../references/go-agent-os/nanobot/pkg/tools/service.go`
- `../references/go-agent-os/nanobot/pkg/tools/flows.go`
- `../references/go-agent-os/nanobot/pkg/agents/truncate.go`
- `../references/go-agent-os/trpc-agent-go/agent/callbacks.go`
- `../references/go-agent-os/trpc-agent-go/model/callbacks.go`
- `../references/go-agent-os/engram/internal/mcp/write_queue.go`

## Provider, channel, and tool-calling guardrails

Command parity is not enough by itself. Gormes also needs source-backed
inventories for provider IDs/auth modes, gateway platform IDs, and raw
tool-call parser families so unsupported behavior remains visible.

| Surface | Current upstream source | Current planner answer | Progress action |
|---|---|---|---|
| Provider IDs and aliases | `../hermes-agent/hermes_cli/providers.py:HERMES_OVERLAYS,ALIASES`, `../hermes-agent/agent/models_dev.py:PROVIDER_TO_MODELS_DEV`, `../hermes-agent/hermes_cli/main.py:--provider choices` | Implemented as a source-backed Go manifest in `internal/hermes/provider_registry_manifest.go`, with drift tests covering overlays, aliases, models.dev mappings, provider prefixes, auth registry names, and `_PROVIDER_MODELS`. Unsupported or not-yet-bound providers remain visible as `row_backed` instead of disappearing from `gormes model`, auth planning, status, or fallback work. | Phase `4.A` validated row: `Hermes provider registry and alias manifest`. |
| Provider auth commands | `../hermes-agent/hermes_cli/main.py:auth_subparsers`, `../hermes-agent/hermes_cli/auth_commands.py` | Current non-deprecated provider login is `auth add <provider> --type oauth`; top-level `login`, `auth login`, and `auth refresh` are not implementation targets. OpenAI Codex is the first native vertical; Anthropic, Nous, Google Gemini CLI, Qwen OAuth, and Spotify remain fixture-ready follow-up adapters. | Phase `5.O` fixture-ready row: `Hermes auth OAuth provider adapters`; existing auth command rows remain the command contract. |
| Gateway platform IDs | `../hermes-agent/gateway/config.py:Platform`, `../hermes-agent/gateway/platforms/*.py` | Implemented as a source-backed Go manifest in `internal/gateway/platform_manifest.go`, with drift tests covering every current Platform enum value and connector file. Unsupported or paused channels remain visible as `row_backed` or `partial`, while `local`, `api_server`, `webhook`, and `wecom_callback` are classified as local/runtime/webhook surfaces instead of being dropped. | Phase `2.B.12` validated row: `Hermes gateway platform registry manifest`. |
| Raw tool-call parsers | `../hermes-agent/environments/tool_call_parsers/*.py` | Validated manifest tracks the current 11 parser files: `deepseek_v3_1`, `deepseek_v3`, `glm45`, `glm47`, `hermes`, `kimi_k2`, `llama`, `longcat`, `mistral`, `qwen3_coder`, and `qwen`. Parser execution and provider-specific malformed-output behavior stay row-backed per family. | Phase `5.B` validated row: `Raw tool-call parser fixture matrix`. |

## Current Gormes visible command surface

The current Gormes root help exposes these top-level commands:

`auth`, `completion`, `config`, `doctor`, `gateway`, `goncho`, `login`,
`memory`, `migrate`, `session`, `telegram`, `usage`, `version`.

Current implemented or stubbed subcommand highlights:

- `gormes auth`: `add`, `list`, `logout`, `remove`, `reset`, `status`.
- `gormes gateway`: `status` is read-model backed; `install`, `restart`,
  `start`, `stop`, and `uninstall` are explicit unavailable stubs until service
  restart support lands.
- `gormes config`: `check`, `edit`, `env-path`, `migrate`, `path`, `set`, `show`.
- `gormes session`: `export`.
- `gormes memory`: `status`.
- `gormes goncho`: `doctor`.
- `gormes migrate`: `hermes`, `openclaw`.
- `gormes telegram`: native Telegram bot adapter entry point.
- `gormes usage`: provider account usage read model flags.
- `gormes login`: compatibility shim; it must remain a deprecation redirect, not
  a new OAuth implementation surface.

Known absent root commands from current Gormes help include `model`, `fallback`,
`setup`, `whatsapp`, `slack`, `logout`, `status`, `cron`, `webhook`, `hooks`,
`dump`, `debug`, `backup`, `import`, `pairing`, `skills`, `plugins`, `tools`,
`mcp`, `sessions`, `insights`, `claw`, `update`, `uninstall`, `acp`, `profile`,
`dashboard`, and `logs`. These absences are not ignored; they are row-backed
below.

## Current upstream nested parser inventory

This table is the planner guardrail for the new nested-manifest refresh row.
It distinguishes real `hermes <group> <subcommand>` parser paths from gateway
message handlers and dynamic plugin commands.

| Hermes parser group | Current upstream nested commands / aliases | Current manifest state | Progress action |
|---|---|---|---|
| `fallback` | `list`/`ls`, `add`, `remove`/`rm`, `clear` | Captured. | Handler work remains under fallback command rows. |
| `gateway` | `run`, `start`, `stop`, `restart`, `status`, `install`, `uninstall`, `setup`, `migrate-legacy` | Captured; gateway message handlers stay separate as `gateway-handler` entries. | Management handlers remain in `Gateway, platform, webhook, and cron management CLI`. |
| `slack` | `manifest` | Captured. | Platform handler work stays row-backed. |
| `auth` | `add`, `list`, `remove`, `reset`, `status`, `logout`, `spotify` | Captured; stale `auth login` and `auth refresh` are excluded. | Keep non-deprecated provider login through `auth add <provider> --type oauth`. |
| `cron` | `list`, `create`/`add`, `edit`, `pause`, `resume`, `run`, `remove`/`rm`/`delete`, `status`, `tick` | Captured. | Handler work remains under `Gateway, platform, webhook, and cron management CLI`. |
| `webhook` | `subscribe`/`add`, `list`/`ls`, `remove`/`rm`, `test` | Captured; stale `serve` is not active. | Platform/webhook behavior remains row-backed. |
| `hooks` | `list`/`ls`, `test`, `revoke`/`remove`/`rm`, `doctor` | Captured; stale `run` is not active. | Hook behavior remains row-backed. |
| `debug` | `share`, `delete` | Captured; stale `doctor`, `paste`, and `sweep` are not active parser commands. | Diagnostics rows own behavior. |
| `config` | `show`, `edit`, `set`, `path`, `env-path`, `check`, `migrate` | Captured. | Config behavior rows own handlers. |
| `pairing` | `list`, `approve`, `revoke`, `clear-pending` | Captured; stale `deny`/`reset` are not active. | Platform pairing behavior remains row-backed. |
| `skills` | `browse`, `search`, `install`, `inspect`, `list`, `check`, `update`, `audit`, `uninstall`, `reset`, `publish`, `snapshot export`, `snapshot import`, `tap list`, `tap add`, `tap remove`, `config` | Captured including nested `snapshot` and `tap` subcommands. | Skill manager/runtime rows own behavior. |
| `plugins` | `install`, `update`, `remove`/`rm`/`uninstall`, `list`/`ls`, `enable`, `disable` | Captured; stale `doctor` is not active. | Plugin SDK owns behavior. |
| `memory` | `setup`, `status`, `off`, `reset` | Captured; plugin-style memory command discovery remains a dynamic plugin class. | Memory/Goncho rows own implementation. |
| `tools` | `list`, `disable`, `enable`; bare `tools` opens interactive config; `--summary` prints summary | Captured. | Tool runtime/security rows own behavior. |
| `mcp` | `serve`, `add`, `remove`/`rm`, `list`/`ls`, `test`, `configure`/`config`, `login` | Captured; stale `call` and `auth` are not active parser commands. | ACP/MCP rows own behavior. |
| `sessions` | `list`, `export`, `delete`, `prune`, `stats`, `rename`, `browse` | Captured; stale `resume` is not active as a parser command. | Session rows own handlers. |
| `claw` | `migrate`, `cleanup`/`clean` | Captured. | OpenClaw migration rows own behavior. |
| `profile` | `list`, `use`, `create`, `delete`, `show`, `alias`, `rename`, `export`, `import` | Captured; stale `set` is not active. | Config/profile rows own behavior. |
| `logs` | No subparser; optional `log_name` argument accepts `agent`, `errors`, `gateway`, or `list`. | Top-level row-backed only. | Diagnostics rows own log viewing/filtering behavior. |

## Hermes top-level parity matrix

Status values:

- `Implemented`: visible in Gormes or covered by an equivalent native command.
- `Partial`: visible but missing important Hermes behavior.
- `Row-backed`: missing or incomplete, with a named `progress.json` row.
- `Gormes-owned`: intentional Go/Goncho extension, not upstream Hermes.
- `Excluded/deprecated`: should not perform legacy behavior; keep compatibility
  or a safe redirect only.

| Hermes command | Gormes state | Backlog owner / proof | Notes |
|---|---|---|---|
| `chat` / root interactive | Partial | `cmd/gormes root TUI/oneshot`; Phase `5.O` root flags rows | Native TUI/oneshot exists; full Hermes chat UX is still broader than root help parity. |
| `model` | Row-backed | Phase `5.O`: `Gormes model interactive provider/model picker` | Must select provider/model and run needed OAuth flows. |
| `fallback` | Row-backed | Phase `5.O`: fallback provider chain rows; CLI manifest rows | Current Hermes subcommands are `list`/`ls`, `add`, `remove`/`rm`, and `clear`. |
| `gateway` | Partial | Phase `5.O`: `Gateway, platform, webhook, and cron management CLI` plus `Gateway management CLI read-model closeout` | `status` exists; mutating service commands are explicit unavailable stubs. |
| `setup` | Row-backed | Phase `5.O`: `Gormes setup minimal sectioned wizard slice`; `Gormes config command surface` | Must preserve Hermes wizard semantics where relevant. |
| `whatsapp` | Row-backed | Phase `5.O`: `Gateway, platform, webhook, and cron management CLI` | Platform management surface not yet visible as a root command. |
| `slack` | Row-backed | Phase `5.O`: `Gateway, platform, webhook, and cron management CLI` | Current Hermes nested command is `slack manifest`. |
| `login` | Excluded/deprecated | Phase `5.O`: `Gormes login deprecated-redirect contract` | Must print deprecation guidance and exit cleanly; do not run OAuth work here. |
| `logout` | Row-backed | Phase `5.O`: `Gormes top-level logout provider shortcut`; auth rows | `gormes auth logout` exists; top-level shortcut remains separate parity work. |
| `auth` | Partial | Phase `5.O`: auth command rows; `hermes-auth-cli-parity.md` | API-key and pool operations exist; Codex device-code OAuth is in progress/native, while Anthropic, Nous, Google Gemini CLI, Qwen, and Spotify remain row-backed. |
| `status` | Partial | Phase `5.O`: `Diagnostics, backup, logs, and status CLI`; gateway status rows | Current equivalent is `gormes gateway status`, not full Hermes root status. |
| `cron` | Row-backed | Phase `5.O`: `Gateway, platform, webhook, and cron management CLI` | Runtime cron exists elsewhere; CLI management parity remains planned. |
| `webhook` | Row-backed | Phase `5.O`: `Gateway, platform, webhook, and cron management CLI` | CLI management surface remains planned. |
| `doctor` | Implemented | `cmd/gormes doctor`; Phase `5.O`: doctor readiness rows | Current command has `--offline`; parity gaps should become diagnostics rows. |
| `dump` | Row-backed | Phase `5.O`: `Diagnostics, backup, logs, and status CLI`; `CLI dump support-summary helper` | Helper exists; command surface remains planned. |
| `debug` | Row-backed | Phase `5.O`: `Diagnostics, backup, logs, and status CLI` | Share/paste/sweep/doctor helpers remain planned. |
| `backup` | Row-backed | Phase `5.O`: `Backup/update opt-in and exclusion policy` | Must keep destructive/update behavior explicit and opt-in. |
| `import` | Row-backed | Phase `5.O`: `Hermes config migration dry-run manifest` | Current Gormes surface is `migrate hermes`; preserve operator expectations. |
| `config` | Implemented/partial | Phase `5.O`: `Gormes config command surface`; config closeout rows | Root command exists with show/set/check/edit/migrate/path/env-path. |
| `pairing` | Row-backed | Phase `5.O`: `Gateway, platform, webhook, and cron management CLI` | Pairing management CLI remains planned. |
| `skills` | Row-backed | Phase `5`: skills/tooling rows and CLI manifest | Not visible in current root help. |
| `plugins` | Row-backed | Phase `5`: `Plugin SDK` rows and CLI manifest | Not visible in current root help. |
| `honcho` | Gormes-owned replacement | Goncho/Gormes memory rows | Gormes exposes `goncho`, not `honcho`; keep Honcho-compatible interfaces but internal branding is Goncho. |
| `memory` | Partial | Phase `3` memory rows; Phase `5` command surface rows | Current visible command has `status`; Hermes plugin-style search/add/delete/export parity remains row-backed. |
| `tools` | Row-backed | Phase `5`: tool/runtime/security rows | Not visible in current root help. |
| `mcp` | Row-backed | Phase `5.O`: `Gormes mcp login OAuth re-auth bridge`; ACP/MCP rows | Not visible in current root help. |
| `sessions` | Row-backed | Phase `5`: session rows | Gormes exposes singular `session export`; plural Hermes surface remains broader. |
| `insights` | Row-backed | Phase `4`: `Self-monitoring telemetry` plus diagnostics rows | Not visible in current root help. |
| `claw` | Row-backed | Phase `5.O`: OpenClaw migration rows | Current equivalent is `gormes migrate openclaw`. |
| `version` | Implemented | `cmd/gormes version` | Visible and help-backed. |
| `update` | Row-backed | Phase `5.O`: `Backup/update opt-in and exclusion policy` | Self-update must remain safe/opt-in. |
| `uninstall` | Row-backed | Phase `5.O`: `Gormes uninstall dry-run command contract` | Destructive behavior must have dry-run/confirmation semantics. |
| `acp` | Row-backed | Phase `5`: `ACP server side` | Not visible in current root help. |
| `profile` | Row-backed | Phase `5.O`: profile resolver/store rows | Not visible in current root help. |
| `completion` | Implemented | Cobra completion command; Phase `5.O`: command-tree manifest | Visible in current root help. |
| `dashboard` | Row-backed | Dashboard API/client rows | Not visible in current root help. |
| `logs` | Row-backed | Phase `5.O`: `Diagnostics, backup, logs, and status CLI`; log redactor rows | Not visible in current root help; redactor/snapshot helpers exist as rows. |

## Provider/auth parity matrix

The supported operator recipe is:

```sh
hermes auth add openai-codex
hermes auth list openai-codex
hermes chat -q 'Reply with exactly: ok' --provider openai-codex --model gpt-5.5
```

The Gormes parity target is:

```sh
gormes auth add openai-codex
gormes auth list openai-codex
gormes -z 'Reply with exactly: ok' --provider openai-codex --model gpt-5.5
```

Do not document manual JSON editing or ad-hoc token copying as the normal path.
Codex CLI token import may exist only as an explicitly labeled emergency bridge.

| Surface | Hermes behavior | Current Gormes state | Backlog owner |
|---|---|---|---|
| `auth add <provider>` API-key path | Securely stores manual pooled credential; redacts secrets. | Visible in `gormes auth add`; API-key path is present. | Phase `5.O`: `Hermes auth credential-pool command surface`. |
| `auth add openai-codex` | Fresh Hermes-owned OAuth device-code flow, stored in `~/.hermes/auth.json`; separate from Codex CLI / VS Code tokens. | Native device-code adapter and credential-pool persistence are in progress; remaining provider OAuth adapters stay row-backed. | Phase `5.O`: `Hermes auth OAuth provider adapters`; `Gormes auth add openai-codex strict isolation contract`. |
| `auth list [provider]` | Lists redacted credential-pool entries and current selection markers. | Implemented as `gormes auth list`. | Phase `5.O`: auth command surface rows. |
| `auth status <provider>` | Provider-specific logged-in/logged-out metadata. | Implemented read model; provider-specific OAuth expansions remain adapter work. | Phase `5.O`: `Gormes auth status per-provider aggregator`. |
| `auth remove <provider> <target>` | Removes by index/id/label and runs source cleanup/suppression. | Implemented for native pool removal; source-specific cleanup gaps must remain explicit. | Phase `5.O`: auth command surface rows. |
| `auth reset <provider>` | Clears credential exhaustion/cooldown/auth-failure state. | Implemented. | Phase `5.O`: auth command surface rows. |
| `auth logout <provider>` | Clears provider auth and resets matching model provider config. | Implemented for native credential pool; top-level shortcut remains row-backed. | Phase `5.O`: `Gormes top-level logout provider shortcut`. |
| `auth spotify` | Separate Spotify control-plane PKCE, not inference provider selection. | Planned. | Phase `5.O`: `Hermes auth Spotify service-provider subcommand`. |
| `model` | Interactive provider/model picker; invokes provider login as needed. | Planned. | Phase `5.O`: `Gormes model interactive provider/model picker`. |
| `setup model` | Wizard path into provider/model setup. | Planned. | Phase `5.O`: `Gormes setup minimal sectioned wizard slice`. |
| `mcp login <name>` | OAuth re-auth for OAuth MCP servers only. | Planned. | Phase `5.O`: `Gormes mcp login OAuth re-auth bridge`. |
| top-level `login` | Deprecated shim; prints guidance to use `auth`, `model`, or `setup`; exits `0`. | Visible as `gormes login`; must stay a redirect shim. | Phase `5.O`: `Gormes login deprecated-redirect contract`. |

## Runtime parity notes from live Telegram dogfood

Recent live Telegram dogfood found two runtime issues that are now part of the
provider/operator parity evidence:

- Provider errors must be safe for Telegram/operator display. Raw HTML upstream
  bodies are sanitized to `provider returned HTML error body` in provider and
  gateway formatting paths.
- Provider `OpenStream` setup failures must return the kernel to idle so the
  next Telegram turn is admitted. The regression row is covered by
  `internal/kernel/provider_failure_admission_test.go`.

The next provider-auth slices should preserve the Hermes path above: native
`gormes auth add <provider> --type oauth` behavior with Hermes-compatible
credential storage, not Python Hermes runtime delegation. The OpenAI Codex
device-code vertical is the first native adapter; any remaining `Forbidden:
provider returned HTML error body` result should be triaged as
auth/entitlement/relogin evidence, not as an admission-control or Telegram
wedging blocker.

## Validation commands for parity-doc changes

Use this minimum validation set when changing this page or the parity backlog:

```sh
go run ./cmd/progress validate
go test ./cmd/gormes -run HermesCLIParity -count=1
go test ./docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1
git diff --check
```

If `progress.json` changes, regenerate derived progress surfaces deliberately
with the repo's progress writer before staging any generated file.
