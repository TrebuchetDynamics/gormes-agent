# Hermes-Aligned Internal and Command Topology Refactor Plan

Date: 2026-05-23

Status: **active architecture plan, implementation not started.** This plan
supersedes the closed `cmd/gormes` command-construction completion note and the
first internal folder-sprawl draft. It plans the next topology shape only; it is
not a side backlog and it does not authorize runtime behavior changes.

Branch rule: work only on `development`. Do not create feature branches or
worktrees.

## Backlog Rule

This file is **not** the executable backlog. The executable backlog remains
`docs/content/building-gormes/architecture_plan/progress.json`, accessed through
`internal/progress.Load` / `SaveProgress` or `cmd/progress`. Before implementing
any slice below, add or refine one progress row with the slice's exact write
scope, source refs, acceptance, and test commands. Do not keep adding unchecked
TODOs to this file.

This plan is a source-backed design map. A builder-ready row is the only work
packet.

## Goal Oracle

The long-term refactor is successful when all of these are true:

1. `cmd/gormes` is a thin binary adapter: process entry, version/build
   provenance, panic/exit-code mapping, root invocation, and command-factory
   injection only.
2. `internal/` top-level scan cost is lower because shallow companion packages
   live behind deeper owner modules.
3. The target module families resemble Hermes' durable source families without
   copying Hermes' Python monoliths one-for-one.
4. Public behavior is unchanged: command spelling, flags, JSON reports, exit
   codes, gateway behavior, channel transcript rendering, tool descriptors, and
   persistence formats stay stable unless an explicit progress row says
   otherwise.
5. Every move is protected by a red characterization/topology test before the
   import-path move and by focused package tests plus `go run ./cmd/progress
   validate` and `git diff --check` after the move.

## Why The Old Plan Is Closed

The previous `cmd/gormes` plan completed command-construction extraction:

- `internal/app/gormescli` owns the command-contract registry, setup registry,
  root Cobra assembly, and feature command constructors.
- `cmd/gormes` still owns process entry, build provenance, panic dumps,
  exit-code mapping, and runtime glue.
- Runtime seams such as auth/OAuth, gateway lifecycle, dashboard serving,
  Slack/Discord/Telegram transport internals, tool execution, and TUI setup
  were intentionally out of scope.

The new problem is different: `cmd/gormes` remains a high-import binary package
and `internal/` has too many top-level folders. Many folders are useful modules
but shallow top-level names. The next refactor should preserve those modules'
interfaces while moving them under the deeper owner seam.

## Source-Backed Snapshot

Commands run from `/home/xel/git/gormes/gormes-agent` on 2026-05-23:

```sh
find internal -mindepth 1 -maxdepth 1 -type d ! -name '.*' | wc -l
# 67

find cmd/gormes -maxdepth 1 -type f -name '*.go' | wc -l
# 303

go list ./internal/... | wc -l
# 107

go list -f '{{join .Imports "\n"}}' ./cmd/gormes \
  | rg '^github.com/TrebuchetDynamics/gormes-agent/internal' | wc -l
# 55
```

Current Gormes topology facts:

- `cmd/gormes` directly imports 55 internal paths, including
  `internal/app/gormescli`, `internal/app/gormescli/modules/*`,
  `internal/toolcompact`, `internal/gonchotools`, `internal/kanbantools`,
  `internal/sessionsearchtool`, `internal/slack`, and many deep runtime
  modules.
- `internal/app/gormescli/root.go` already centralizes root Cobra assembly and
  top-level command ordering, but it lives under a shallow `internal/app`
  folder with no sibling app packages.
- `internal/cli/command_registry.go` mirrors Hermes slash-command semantics
  from `hermes_cli/commands.py`, while `internal/app/gormescli/contracts.go`
  mirrors visible Cobra command ownership. These are related CLI surface
  contracts but split across two unrelated-looking top-level paths.
- `internal/gateway/platform_manifest.go` already points Discord at
  `internal/channels/discord`, but Slack still points at `internal/slack`.
- `internal/tools` imports `internal/toolcompact` and tests in `cmd/gormes`
  import `internal/toolcompact`; these helpers are tool implementation details,
  not top-level product modules.
- `cmd/progress` is a thin entrypoint over `internal/progressctl`, and
  `internal/progressctl` imports `internal/builderloop` plus
  `internal/progress`. `cmd/repoctl` is similarly a thin entrypoint over
  `internal/repoctl`.

Hermes reference facts from the in-repo checkout `./hermes-agent`:

- Hermes has durable source families at top level: `hermes_cli/`, `gateway/`,
  `agent/`, `tools/`, `providers/`, `skills/`, `cron/`, `plugins/`, and
  `ui-tui/`.
- `hermes_cli/_parser.py` owns top-level parser construction and the `chat`
  subparser. It intentionally keeps only parser/bootstrap mechanics there; most
  subcommands are registered by `hermes_cli/main.py`.
- `hermes_cli/main.py` owns `cmd_*` dispatch functions and a large argparse
  registration block. That is source evidence for command-surface ownership, but
  it is also a Python monolith Gormes should not copy directly.
- `hermes_cli/commands.py` is the central slash-command registry. CLI help,
  gateway dispatch, Telegram BotCommands, Slack subcommand mapping, and
  autocomplete derive from `COMMAND_REGISTRY`.
- `hermes_cli/config.py` owns config path/default/secret mechanics, while
  `hermes_cli/claw.py` owns OpenClaw migration. This supports keeping Gormes
  config and migration modules as domain owners instead of hiding them inside
  the binary package.
- `gateway/run.py` is a very large gateway runner with slash handlers,
  adapter lifecycle, delivery, shutdown, session replay, and background
  watchers. Gormes should resemble its domain seams (`gateway/` plus
  `channels/<platform>/`) while avoiding one giant Go package.
- Hermes `tools/` keeps tool adapters and tool helpers together. This supports
  moving Gormes `toolcompact`, `tooltrace`, `lsp`, `gonchotools`,
  `kanbantools`, and `sessionsearchtool` under `internal/tools/*`.

## Hermes Resemblance Rules

Resemble Hermes at the **module-family** level, not at the file-count or
monolith level.

| Hermes source family | Gormes long-term owner seam | Rule |
|---|---|---|
| `hermes_cli/_parser.py`, `hermes_cli/main.py` | `cmd/gormes` + `internal/cli/surface` + `internal/cli/commands/*` | Keep process entry in `cmd/gormes`; move reusable command construction and command-surface contracts under `internal/cli`. |
| `hermes_cli/commands.py` | `internal/cli/command_registry.go` plus `internal/cli/surface` manifests | One registry feeds slash semantics, TUI completion, gateway command policy, and visible command ownership. |
| `hermes_cli/config.py` | `internal/config` and config-facing `internal/cli` helpers | Config loading/secrets stay in `internal/config`; CLI presentation stays near `internal/cli`. |
| `hermes_cli/claw.py` | `internal/migrate/openclaw` and `cmd/gormes migrate/openclaw` adapters | Migration is its own domain, not generic CLI glue. |
| `gateway/run.py`, `gateway/platforms/*` | `internal/gateway` + `internal/channels/<platform>` | Gateway owns lifecycle/session/coalescing; channel adapters own platform transport and rendering. |
| `agent/*` | `internal/kernel`, `internal/hermes`, `internal/agent`, `internal/memory`, `internal/store` | Preserve Gormes' deeper Go seams; do not merge into one `agent` package just to match Python. |
| `tools/*` | `internal/tools/*` | Tool adapters/helpers live behind the tool registry seam. |
| `providers/*` and provider adapters in `agent/*` | `internal/hermes` + `internal/provider` | Provider wire/model behavior stays out of CLI and gateway modules. |
| `skills/*` | `internal/skills` | Skills remain a deep runtime module, with command presentation via CLI modules only. |
| Gormes-only progress delivery | `internal/progress/*` | This has no Hermes analog; keep it explicitly Gormes-owned and away from runtime agent modules. |

Anti-rule: do not move code just because a path name differs from Hermes. Move
only when locality and leverage improve.

## Target Shape

The target shape is intentionally staged. The short-term moves use low-risk
rehome paths; the long-term shape can rename subpackages only after import
cycles and tests are proven.

```text
cmd/
  gormes/                         thin binary adapter only
    main.go                       process entry + root invocation
    version.go                    build provenance
    exitcode.go                   exit-code mapping
    crash/panic helpers           if still package-local and process-only
  progress/                       thin entrypoint over internal/progress/ctl
  repoctl/                        thin entrypoint over internal/progress/repoctl

internal/
  cli/                            Hermes CLI analogue and operator surface
    command_registry.go           slash/active-turn policy from Hermes commands.py
    surface/                      long-term home for root assembly and manifests
      root.go                     moved from internal/app/gormescli/root.go
      contracts.go                moved from internal/app/gormescli/contracts.go
      setup_registry.go           moved from internal/app/gormescli/setup_registry.go
      rowbacked.go                moved from internal/app/gormescli/rowbacked.go
    commands/                     command construction modules, not runtime owners
      providers/                  auth/logout/model/fallback/usage/insights constructors
      profiles/                   profile command constructors
      channels/                   telegram/slack/whatsapp/capabilities constructors
      gateway/                    gateway/dashboard/agent/webhook/hooks/pairing constructors
      sessions/                   sessions/checkpoints constructors
    modelcatalog/                 optional: only if CLI is the only real caller
    profileseed/                  optional: only if profile CLI owns the lifecycle

  gateway/                        GatewayRunner analogue without Python monolith
    manager/coalescing/session/fleet/slash modules remain package-local files
    platform_manifest.go          all channel surfaces point to internal/channels/*

  channels/                       Hermes gateway/platforms analogue
    telegram/
    discord/                      merge top-level internal/discord here
    slack/                        move top-level internal/slack here
    whatsapp/
    ...

  tools/                          Hermes tools analogue and registry seam
    compact/                      moved from internal/toolcompact
    trace/                        moved from internal/tooltrace
    lsp/                          moved from internal/lsp
    goncho/                       moved from internal/gonchotools
    kanban/                       moved from internal/kanbantools
    sessionsearch/                moved from internal/sessionsearchtool
    whisper/                      optional move from internal/wasi/whisper after audio check

  progress/                       Gormes-owned delivery/control-plane module
    schema/model files            current internal/progress
    ctl/                          moved from internal/progressctl
    builderloop/                  moved from internal/builderloop
    plannerloop/                  moved from internal/plannerloop
    triggers/                     moved from internal/plannertriggers
    repoctl/                      moved from internal/repoctl
    fidelity/                     moved from internal/fidelity

  runtime/                        local process/runtime mechanics
    cmdrunner/                    moved from internal/cmdrunner
    bridge/                       optional move from internal/runtimebridge
    update/managed-source pieces  optional only when repeated process mechanics exist

  migrate/                        external state import/migration domain
    hermes/
    openclaw/

  kernel/ hermes/ provider/ config/ tui/ memory/ session/ store/
  goncho/ skills/ plugins/ cron/ kanban/ subagent/ apiserver/
```

Short-term compatible path: move `internal/app/gormescli` to
`internal/cli/gormescli` if that is the safest first row. Long-term preferred
path: rename it to `internal/cli/surface` and move its modules to
`internal/cli/commands/*` after a second characterization test proves no import
cycles and no command-manifest drift.

## Topology Budgets

Budgets are guardrails, not the product goal. Do not trade correctness for a
folder-count number.

Primary targets after the first consolidation pass:

- Reduce non-hidden `internal/` top-level directories from **67** to **45 or
  fewer**.
- Reduce `cmd/gormes` direct `internal/...` imports from **55** to **35 or
  fewer**.
- Keep `cmd/gormes` direct imports focused on process/runtime assembly only:
  `internal/cli/surface`, `internal/config`, `internal/runtime`, and the few
  modules needed to build production factories until command modules absorb
  them.

Stretch targets after the second pass:

- Reduce non-hidden `internal/` top-level directories to **38 or fewer**.
- Reduce `cmd/gormes` direct `internal/...` imports to **20 or fewer**.
- Reduce `cmd/gormes` Go files toward **80 or fewer** by moving command
  construction into `internal/cli/commands/*` and leaving runtime-heavy command
  bodies behind deeper modules.

## Non-Negotiable Constraints

- No public command spelling, flag, JSON shape, exit code, gateway behavior,
  channel transcript behavior, tool descriptor, or persistence format changes
  unless a progress row explicitly asks for that behavior change.
- Every code-moving slice starts with a RED characterization or topology test.
- No broad `go fmt ./...` or repo-wide rewrites in a dirty worktree.
- Update `codemap.md` files only after the code move is green.
- Do not create compatibility shim packages just to keep old import paths alive.
  In an `internal/` package move, updating imports is safer than preserving a
  stale shallow module.
- Do not move generated docs or progress rows by hand. Use `cmd/progress` and
  `internal/progress` tooling when the executable backlog changes.

## Topology Guard Contract

The first implementation row should add a small, explicit topology guard before
moving code. The guard is an **interface** for the architecture plan: builders
can change implementations and import paths, but the guard makes the intended
shape executable.

Recommended file:

```text
internal/internal_topology_test.go
```

Guard responsibilities:

1. **Budget reporting, not budget-only failure.** Report the non-hidden
   `internal/` top-level directory count and the `cmd/gormes` direct
   `internal/...` import count. Budget failures should be opt-in by migration
   phase so an unrelated dirty tree does not fail because the repository is
   still between planned waves.
2. **Forbidden legacy roots per active row.** Each migration row gets a table
   entry with `OldRoots`, `NewRoots`, and `OwnerModule`. The RED proof is
   turning on one entry before the package move; it must fail on the old root
   and pass only after imports and path literals move.
3. **Import-path and path-literal audit.** Search Go imports and important path
   literals for the old roots. For CLI surface, this catches both
   `github.com/.../internal/app/gormescli` imports and progress/test strings
   that still claim `internal/app/gormescli` is the owner.
4. **Manifest-sensitive checks.** For channel moves, assert implemented channel
   `GormesSurface` values in `internal/gateway/platform_manifest.go` resolve
   under `internal/channels/<platform>` unless the row explicitly documents an
   exception.
5. **No hard global allowlist in the first row.** A full allowed-directory set
   is too brittle while unrelated work is dirty. Start with forbidden roots,
   metrics, and active-row expectations; add global budgets after two or more
   moves prove the pattern.

Suggested migration-entry shape:

```text
owner_module: cli|tools|channels|progress|runtime
old_roots:
  - internal/app/gormescli
new_roots:
  - internal/cli/gormescli
source_refs:
  - hermes-agent/hermes_cli/_parser.py:build_top_level_parser
  - internal/app/gormescli/root.go:NewRootCommand
red_signal: topology guard fails while old root is still imported or present
```

## Package Move Playbook

Every package move should follow the same playbook so future builders do not
turn topology work into behavior work.

1. **Shape the executable row first.** Use `cmd/progress` or
   `internal/progress` tooling to refine the row. The row names one module,
   one package family, source refs, write scope, and focused tests.
2. **Add the RED guard/characterization.** The test must fail for the real
   reason: old import path still present, command manifest drift, channel
   manifest drift, descriptor/render behavior drift, or process-runner behavior
   drift.
3. **Move one package family.** Prefer `git mv` for the directory and update
   import paths in the allowed write scope only. Do not combine CLI, tools,
   channels, and progress moves in one builder row.
4. **Run targeted `gofmt`, not repo-wide formatting.** Format only changed Go
   files. This preserves unrelated dirty work and makes review possible.
5. **Audit old paths.** Run `rg '<old/root>|github.com/.*/<old/root>'` against
   `cmd`, `internal`, and progress/docs surfaces named in the row. Intentional
   historical mentions must be in comments or docs that explain the migration,
   not active contracts.
6. **Run focused tests, then progress validation, then `git diff --check`.**
   Only after the code is green should builders update codemaps or this plan's
   path examples.
7. **No compatibility shims by default.** An `internal/` import is not a public
   compatibility contract. Shims preserve shallow modules and defeat the
   deletion test unless a progress row explicitly requires temporary bridging.

## Owner Interface Targets

These are the small caller-facing interfaces each deepened module should expose.
They are not new Go interface types by default; they are the facts callers are
allowed to know.

| Owner module | Caller-facing interface after consolidation | Implementation hidden behind it |
|---|---|---|
| `internal/cli/surface` or short-term `internal/cli/gormescli` | root assembly, command ownership manifest, setup registry, row-backed command helpers | Cobra construction details, module command ordering, setup section ownership checks |
| `internal/cli/commands/*` | one command-family constructor per feature family, with explicit runtime seams passed in from `cmd/gormes` | provider/profile/channel/gateway command wiring and JSON presentation adapters |
| `internal/tools/*` | tool registry descriptors, execution helpers, compact/trace rendering helpers imported through tool-owned paths | descriptor generation, terminal/tool compaction, session search, Goncho/Kanban tool glue, optional LSP/whisper helpers |
| `internal/channels/<platform>` | platform adapter constructors, render/parsing helpers, platform-specific config validation | Slack/Discord/Telegram transport details, transcript formatting, command registration, media handling |
| `internal/progress/*` | progress schema, validator/writer, repo metadata commands, planner/builder evidence loops | filesystem layout, doc generation, loop ledgers, fidelity evidence, row seeding |
| `internal/runtime/*` | local process/runtime mechanics such as command running, restart/update support, runtime bridge helpers | subprocess options, timeouts, env shaping, OS-specific process behavior |

If a caller still needs to know a package's old folder name, filesystem layout,
or migration history after the move, the module is still too shallow.

## Architecture Candidates

### 1. CLI Surface Enclave — Strong (9/10)

Files/packages:

- `cmd/gormes/*`
- `internal/app/gormescli`
- `internal/app/gormescli/modules/{providers,profiles,channels,gateway,sessions}`
- `internal/cli/command_registry.go`
- `internal/channels/navivox/profile_contacts.go`

Current interface burden:

- Callers must know two command-surface homes:
  `internal/app/gormescli` for Cobra command construction and `internal/cli` for
  slash/active-turn policy and CLI helper mechanics.
- `cmd/gormes/main.go` imports `internal/app/gormescli` directly and still holds
  the factory map for dozens of command constructors.
- Feature command files in `cmd/gormes` still import `internal/app/gormescli`
  modules, so command construction looks partly migrated and partly binary-owned.
- Hermes keeps `hermes_cli/_parser.py`, `hermes_cli/main.py`, and
  `hermes_cli/commands.py` in one source family. Gormes should mirror that
  family as `internal/cli/*`, not as `internal/app/*` plus `internal/cli/*`.

Deepening move:

- Short-term: move `internal/app/gormescli` to `internal/cli/gormescli` with no
  package-name change, preserving command contracts and tests.
- Long-term: rename `internal/cli/gormescli` to `internal/cli/surface` and move
  its `modules/*` to `internal/cli/commands/*` after import-cycle checks.
- Keep low-level CLI primitives such as profile resolution, setup plans,
  update helpers, and command registry in `internal/cli` only when they are
  reused outside command construction.
- Move command factory ownership out of `cmd/gormes` one family at a time until
  `cmd/gormes` injects runtime adapters instead of constructing feature command
  trees itself.

Deletion test:

- Deleting `internal/app` should remove no behavior; its only job is to hold the
  already-useful `gormescli` command-surface module. The module earns its keep;
  the `app` top-level seam does not.

Two-adapter test:

- The command-surface seam is real: `cmd/gormes`, TUI completion,
  gateway/slash policy, setup manifests, Navivox profile contacts, tests, and
  progress rows all consume command ownership evidence.

First safe TDD slices:

1. RED: add or extend a topology test proving `internal/app` is forbidden after
   the active row and that the live Cobra manifest still matches the registry.
2. GREEN: move `internal/app/gormescli` to `internal/cli/gormescli`; update
   imports only.
3. REFACTOR: update codemaps and this plan's path names.
4. Later RED: add an import-cycle/topology test for the long-term rename to
   `internal/cli/surface` and `internal/cli/commands/*`.

Focused validation:

```sh
go test ./cmd/gormes ./internal/cli/... -run 'CLIContract|Root|Setup|Profile|Provider|Gateway|Channels|Session|Navivox|CommandRegistry' -count=1
go run ./cmd/progress validate
git diff --check
```

Expected folder/import impact:

- Top-level reduction: **1** immediately (`internal/app`).
- Import clarity: high. This aligns Gormes' command family with Hermes'
  `hermes_cli/*` source family.

### 2. Tool Adapter Enclave — Strong (8/10)

Files/packages:

- `internal/gonchotools`
- `internal/kanbantools`
- `internal/sessionsearchtool`
- `internal/toolcompact`
- `internal/tooltrace`
- `internal/lsp`
- `internal/wasi/whisper` and `internal/wasi/whisper/audio` after a separate
  whisper-specific check
- callers in `cmd/gormes`, `internal/tools`, `internal/kernel`,
  `internal/gateway`, `internal/tui`, `internal/slack`, and `internal/discord`

Current interface burden:

- Callers must know that some tool adapters live beside `internal/tools` rather
  than under it.
- Tool rendering helpers (`tooltrace`) and compaction helpers (`toolcompact`)
  look like top-level product modules even though they only support tool display
  or execution.
- `cmd/gormes/registry.go` imports several tool-adapter packages directly.
- Hermes keeps tool adapters/helpers in `tools/*`, making `internal/tools/*` the
  natural Gormes analogue.

Deepening move:

- Move companion packages under `internal/tools/*` and keep `internal/tools` as
  the public registry seam.
- Use package names that describe the adapter role, not the old top-level
  folder name: `compact`, `trace`, `lsp`, `goncho`, `kanban`, `sessionsearch`.

Deletion test:

- Deleting these helper packages today would spread descriptor, render, and
  execution logic into `cmd/gormes`, `internal/tools`, kernel tests, and channel
  packages. They earn their keep; the top-level folders do not.

Two-adapter test:

- Real seam for tools: `internal/tools` already aggregates many adapters.
- `trace` is a shared render helper used by multiple channel adapters and TUI.

First safe TDD slices:

1. RED: add `internal/internal_topology_test.go` assertion that old top-level
   tool helper paths are forbidden once the tool-enclave row is active.
2. GREEN: move `internal/toolcompact` to `internal/tools/compact` and update
   only its callers.
3. Repeat for `tooltrace`, `lsp`, `sessionsearchtool`, `kanbantools`, and
   `gonchotools` as separate rows if needed.
4. Defer `wasi/whisper` until audio build tags and optional dependencies are
   characterized.

Focused validation:

```sh
go test ./internal/tools ./cmd/gormes -run 'Compact|Terminal|ExecuteCode|Registry|SessionSearch|Kanban|Goncho' -count=1
go test ./internal/gateway ./internal/tui ./internal/channels/telegram ./internal/channels/discord -run 'Tool|Trace|Render|Preview' -count=1
go run ./cmd/progress validate
git diff --check
```

Expected folder impact:

- Top-level reduction: **5-7** folders, depending on whether `wasi/whisper` is
  moved in the first pass.

### 3. Channel Runtime Consolidation — Strong (8/10)

Files/packages:

- `internal/slack`
- `internal/discord`
- `internal/channels/discord`
- future `internal/channels/slack`
- `cmd/gormes/gateway.go`, `cmd/gormes/doctor.go`
- `internal/gateway/platform_manifest.go`

Current interface burden:

- Discord has two top-level-looking homes: `internal/channels/discord` and
  `internal/discord`, both package `discord`.
- Slack is the only major implemented channel still living at top level.
- The platform manifest points Discord at `internal/channels/discord` but Slack
  at `internal/slack`, encoding the inconsistency into user-facing evidence.
- Hermes' channel adapters live under `gateway/platforms/*`; Gormes' analogue is
  `internal/channels/<platform>` plus `internal/gateway`.

Deepening move:

- Move Slack to `internal/channels/slack`.
- Merge the top-level Discord runtime helpers into `internal/channels/discord`
  after transcript/render characterization tests pass.
- Update the platform manifest to point Slack at `internal/channels/slack`.

Deletion test:

- Deleting top-level channel folders should not remove channel behavior; it
  should only remove duplicate homes. Channel behavior belongs under
  `internal/channels/<platform>`.

Two-adapter test:

- Real seam: Telegram, WhatsApp, Discord, SimpleX, and other adapters already
  live under `internal/channels/<platform>`.

First safe TDD slices:

1. RED: add/extend manifest test proving Slack's Gormes surface is
   `internal/channels/slack` and every implemented messaging channel resolves
   under `internal/channels/<platform>`.
2. GREEN: move `internal/slack` to `internal/channels/slack` and update imports.
3. Separate row: merge `internal/discord` into `internal/channels/discord`.

Focused validation:

```sh
go test ./internal/channels/slack ./internal/channels/discord ./internal/gateway -run 'Slack|Discord|Platform|Render|Mention|Thread|Approval' -count=1
go test ./cmd/gormes -run 'Gateway|Slack|Discord|Doctor' -count=1
go run ./cmd/progress validate
git diff --check
```

Expected folder impact:

- Top-level reduction: **2** folders.
- Total directory reduction: **1** folder if Discord is merged into the existing
  `internal/channels/discord` directory and Slack creates one nested directory.

### 4. Progress Delivery Enclave — Strong (8/10)

Files/packages:

- `internal/builderloop`
- `internal/plannerloop`
- `internal/plannertriggers`
- `internal/progressctl`
- `internal/repoctl`
- `internal/fidelity`
- `internal/cmdrunner` moved separately to `internal/runtime/cmdrunner`
- entrypoints in `cmd/progress` and `cmd/repoctl`

Current interface burden:

- Planning/building/repo metadata packages are separate top-level folders even
  though they all serve progress-driven delivery.
- `cmd/progress` wraps `internal/progressctl`; `progressctl` wraps
  `internal/progress` and `internal/builderloop`.
- `repoctl` and `fidelity` are repo/progress evidence tools, not runtime agent
  modules.
- Hermes has no direct analogue for Gormes' self-development control plane, so
  this must be explicitly Gormes-owned rather than hidden under generic CLI or
  agent packages.

Deepening move:

- Keep `internal/progress` as the schema/model package.
- Move delivery tooling under `internal/progress/*` subpackages:
  `builderloop`, `plannerloop`, `triggers`, `ctl`, `repoctl`, and `fidelity`.
- Move generic command execution to `internal/runtime/cmdrunner` because ACP,
  planner, builder, and progress tooling all use process execution mechanics.

Deletion test:

- Deleting these packages would spread progress validation, row seeding,
  builder-loop evidence, and repo metadata logic across `cmd/` entrypoints.
  The modules are useful; the scattered top-level layout is not.

Two-adapter test:

- `cmd/progress` and `cmd/repoctl` are separate adapters over related
  progress/repo evidence mechanics.
- `plannerloop` and `builderloop` both use trigger/runner mechanics.

First safe TDD slices:

1. RED: add an import-path topology test forbidding new code from importing
   `internal/progressctl`, `internal/builderloop`, and `internal/plannerloop`
   once the active row lands.
2. GREEN slice 1: move `internal/cmdrunner` to `internal/runtime/cmdrunner` and
   update ACP/planner/builder callers.
3. GREEN slice 2: move `internal/progressctl` to `internal/progress/ctl` and
   update `cmd/progress`.
4. Later slices move `builderloop`, `plannerloop`, `plannertriggers`,
   `repoctl`, and `fidelity`.

Focused validation:

```sh
go test ./cmd/progress ./cmd/repoctl ./internal/progress/... ./internal/runtime/... -count=1
go test ./internal/acp ./cmd/gormes -run 'ACP|Progress|Repo|Fidelity' -count=1
go run ./cmd/progress validate
git diff --check
```

Expected folder impact:

- Top-level reduction: **7** folders.

### 5. Runtime Mechanics Enclave — Worth Exploring (6/10)

Files/packages:

- `internal/cmdrunner`
- `internal/runtimebridge`
- `internal/bridge`
- process/update/restart helpers currently split between `cmd/gormes`,
  `internal/cli`, `internal/gateway`, and `internal/runtime`

Current interface burden:

- Process execution and bridge mechanics are scattered. Some live under
  `internal/runtime`, some under `internal/cmdrunner`, some under
  `internal/runtimebridge`, and some under `internal/bridge`.
- The names do not tell callers whether they are dealing with local process
  control, provider/runtime config, ACP bridge, or gateway runtime state.

Deepening move:

- Move only repeated local process mechanics under `internal/runtime/*`.
- Do **not** merge domain bridges that have their own protocol semantics unless
  callers share the same process lifecycle interface.
- Start with `cmdrunner` because it is a clear generic process adapter used by
  multiple modules.

Deletion test:

- `cmdrunner` earns its keep because deleting it would spread process-runner
  setup into ACP/progress/planner code. `bridge` and `runtimebridge` need more
  evidence before moving.

Two-adapter test:

- `cmdrunner` has real adapter demand across ACP and progress tooling.
- `runtimebridge` and `bridge` may be single-domain modules; treat them as
  speculative until caller evidence says otherwise.

First safe TDD slice:

- RED: characterization test for command runner timeout/stdout/stderr behavior
  through current public callers.
- GREEN: move `internal/cmdrunner` to `internal/runtime/cmdrunner`.
- STOP: re-evaluate `bridge` and `runtimebridge` after the first move.

Focused validation:

```sh
go test ./internal/runtime/... ./internal/acp ./internal/plannerloop ./internal/builderloop -run 'Command|Runner|Timeout|ACP|Loop' -count=1
go run ./cmd/progress validate
git diff --check
```

Expected folder impact:

- Top-level reduction: **1** immediately, more only after evidence.

## Proposed Slice Order

1. **Architecture guard row**
   - Add a topology guard that can express active migrations without failing
     before the move lands.
   - Prefer forbidden legacy path sets over a hard allowed-directory budget for
     early rows. A budget-only test is too easy to game.
   - First guard should report: non-hidden top-level internal directory count,
     direct `cmd/gormes` internal import count, and forbidden old roots for the
     current row.

2. **CLI surface rehome**
   - Aligns the most visible mismatch with Hermes: `hermes_cli/*` should map to
     `internal/cli/*`, not `internal/app/*`.
   - Start with `internal/app/gormescli -> internal/cli/gormescli` because it is
     a pure path move with existing command-manifest tests.
   - Defer the long-term `surface`/`commands` rename until after the path move.

3. **Tool adapter enclave**
   - Highest folder-count reduction with low public-contract risk.
   - Move one package family at a time; start with `toolcompact` or
     `sessionsearchtool` before touching channel render paths.

4. **Channel runtime consolidation**
   - Move Slack first because there is no existing `internal/channels/slack`
     conflict.
   - Merge top-level Discord only after Slack proves the pattern.

5. **Progress delivery enclave**
   - Start with `cmdrunner -> runtime/cmdrunner` because it unblocks both ACP and
     progress tooling and has clear shared mechanics.
   - Move `progressctl` next, then builder/planner/repo evidence packages.

6. **Second-pass CLI surface deepening**
   - Rename `internal/cli/gormescli` to `internal/cli/surface` if the first pass
     is green.
   - Move command modules to `internal/cli/commands/*` only when the import-cycle
     test proves `internal/cli` primitives remain independent.
   - Move `modelcatalog` and `profileseed` under `internal/cli/*` only if source
     evidence says CLI/profile ownership is their primary reason to exist.

7. **Second-pass leaf triage**
   - Audit `contextrefs`, `loopcost`, `i18n`, `mcpserver`, `events`,
     `tuigateway`, `piextension`, `testutil`, and test-only packages such as
     `internal/e2e` and `internal/installtest`.
   - Only move them when a source-backed owner exists and a focused test can
     preserve behavior.

## TDD Rule For All Slices

Use red-green-refactor, even for package moves.

1. **RED**: add or extend a public characterization/topology test before moving
   files. Prefer one of:
   - `internal/*_test.go` architecture guard for forbidden old top-level paths;
   - command manifest tests for CLI imports and visible command ownership;
   - transcript/render tests for channel moves;
   - tool descriptor/execution tests for tool-adapter moves;
   - `go list`/package import budget tests when the behavior is topology.
2. **GREEN**: move one package family only, update imports, and keep behavior
   unchanged.
3. **REFACTOR**: clean package names, codemaps, and path strings only while
   tests are green.
4. **VERIFY**: run the focused package tests, `go run ./cmd/progress validate`,
   and `git diff --check`.

A topology test is not weak when it fails on the real requirement: a forbidden
old import path, forbidden top-level folder, command-manifest drift, or channel
manifest inconsistency.

## Progress-Ready Row Templates

Use these as row seeds, not as a side backlog. A planner must insert/refine the
actual row in `progress.json` before builder work starts.

```text
name: internal topology guard for package consolidation
module: progress
execution_owner: orchestrator
slice_size: small
contract_status: fixture_ready
contract: Add a topology guard that reports internal top-level directory count,
  cmd/gormes direct internal import count, and active-row forbidden legacy roots
  without changing runtime behavior. The guard must support row-scoped
  forbidden roots so each later package move can start with a meaningful RED
  proof instead of a brittle whole-repo directory allowlist.
write_scope:
  - internal/internal_topology_test.go
  - internal/REFACTOR-CMD-PLAN.md
source_refs:
  - internal/REFACTOR-CMD-PLAN.md:Topology Guard Contract
  - cmd/gormes/main.go:rootCommandFactories
  - internal/app/gormescli/root.go:NewRootCommand
  - internal/gateway/platform_manifest.go
ready_when:
  - test can be made RED by enabling one active migration entry for an old root
  - guard reports metrics without failing unrelated untouched directories
not_ready_when:
  - guard hard-codes a global allowed directory set before any move lands
  - guard requires generated docs or broad repo formatting
acceptance:
  - focused topology test passes with no active migration enabled
  - the next migration row can turn on one forbidden-root entry and get a RED
    signal before moving files
test_commands:
  - go test ./internal -run TestInternalTopology -count=1
  - go run ./cmd/progress validate
  - git diff --check
```

```text
name: internal CLI surface package rehome
module: cli
execution_owner: tools
slice_size: medium
contract_status: fixture_ready
contract: Move the completed command-surface assembly from
  internal/app/gormescli to internal/cli/gormescli while preserving the live
  Cobra command manifest, setup registry, slash-command ownership, root command
  ordering, and feature command constructors. This is the first Hermes-aligned
  step toward internal/cli/surface and internal/cli/commands/*.
write_scope:
  - internal/app/gormescli
  - internal/cli/gormescli
  - internal/cli/command_registry.go
  - cmd/gormes
  - internal/channels/navivox/profile_contacts.go
source_refs:
  - hermes-agent/hermes_cli/_parser.py:build_top_level_parser
  - hermes-agent/hermes_cli/main.py:main
  - hermes-agent/hermes_cli/commands.py:COMMAND_REGISTRY
  - internal/app/gormescli/root.go:NewRootCommand
  - internal/app/gormescli/contracts.go:ModuleContract
  - internal/cli/command_registry.go:CommandRegistry
test_commands:
  - go test ./cmd/gormes ./internal/cli/... -run 'CLIContract|Root|Setup|Profile|Provider|Gateway|Channels|Session|Navivox|CommandRegistry' -count=1
  - go run ./cmd/progress validate
  - git diff --check
```

```text
name: internal tool adapter package consolidation
module: tools
execution_owner: tools
slice_size: medium
contract_status: fixture_ready
contract: Move shallow tool helper/adapter packages under internal/tools while
  preserving tool descriptors, execution behavior, tool-call rendering, and CLI
  registry wiring.
write_scope:
  - internal/toolcompact
  - internal/tooltrace
  - internal/lsp
  - internal/gonchotools
  - internal/kanbantools
  - internal/sessionsearchtool
  - internal/tools
  - cmd/gormes/registry.go
source_refs:
  - hermes-agent/tools/registry.py
  - hermes-agent/tools/terminal_tool.py
  - hermes-agent/tools/kanban_tools.py
  - hermes-agent/tools/session_search_tool.py
  - internal/tools/builtin.go
  - cmd/gormes/registry.go
test_commands:
  - go test ./internal/tools ./cmd/gormes -run 'Compact|Terminal|ExecuteCode|Registry|SessionSearch|Kanban|Goncho' -count=1
  - go test ./internal/gateway ./internal/tui ./internal/channels/discord -run 'Tool|Trace|Render|Preview' -count=1
  - go run ./cmd/progress validate
  - git diff --check
```

```text
name: internal channel runtime package consolidation
module: channels
execution_owner: gateway
slice_size: medium
contract_status: fixture_ready
contract: Move Slack and duplicate Discord runtime packages under
  internal/channels/<platform> while preserving platform manifest evidence,
  gateway wiring, transcript rendering, approvals, and mention/thread behavior.
write_scope:
  - internal/slack
  - internal/discord
  - internal/channels/slack
  - internal/channels/discord
  - internal/gateway/platform_manifest.go
  - cmd/gormes/gateway.go
  - cmd/gormes/doctor.go
source_refs:
  - hermes-agent/gateway/run.py:GatewayRunner
  - hermes-agent/gateway/platforms/slack.py
  - hermes-agent/gateway/platforms/discord.py
  - internal/gateway/platform_manifest.go
  - internal/channels/discord
test_commands:
  - go test ./internal/channels/slack ./internal/channels/discord ./internal/gateway -run 'Slack|Discord|Platform|Render|Mention|Thread|Approval' -count=1
  - go test ./cmd/gormes -run 'Gateway|Slack|Discord|Doctor' -count=1
  - go run ./cmd/progress validate
  - git diff --check
```

```text
name: internal progress delivery package consolidation
module: progress
execution_owner: orchestrator
slice_size: large
contract_status: fixture_ready
contract: Move progress-driven builder/planner/repo evidence packages under
  internal/progress subpackages and move shared command execution under
  internal/runtime/cmdrunner while preserving progress validation, doc writing,
  repoctl behavior, and loop fixture behavior.
write_scope:
  - internal/builderloop
  - internal/plannerloop
  - internal/plannertriggers
  - internal/progressctl
  - internal/repoctl
  - internal/fidelity
  - internal/cmdrunner
  - internal/progress
  - internal/runtime
  - cmd/progress
  - cmd/repoctl
  - internal/acp
source_refs:
  - cmd/progress/main.go
  - internal/progressctl/progressctl.go
  - internal/progress
  - internal/builderloop
  - internal/plannerloop
  - cmd/repoctl/main.go
  - internal/repoctl
test_commands:
  - go test ./cmd/progress ./cmd/repoctl ./internal/progress/... ./internal/runtime/... -count=1
  - go test ./internal/acp ./cmd/gormes -run 'ACP|Progress|Repo|Fidelity' -count=1
  - go run ./cmd/progress validate
  - git diff --check
```

## Validation Matrix

| Slice type | Minimum RED proof | Green gate |
|---|---|---|
| CLI path rehome | command manifest/topology test fails on `internal/app` or old imports | `go test ./cmd/gormes ./internal/cli/... -run 'CLIContract|Root|Setup|Profile|Provider|Gateway|Channels|Session|Navivox|CommandRegistry' -count=1` |
| Tool helper move | descriptor/execution/render test fails when old helper path or behavior drifts | `go test ./internal/tools ./cmd/gormes -run 'Compact|Terminal|ExecuteCode|Registry|SessionSearch|Kanban|Goncho' -count=1` |
| Channel adapter move | platform manifest/transcript fixture fails on old channel surface | `go test ./internal/channels/<platform> ./internal/gateway ./cmd/gormes -run '<Platform>|Gateway|Doctor' -count=1` |
| Progress delivery move | progress command fixture fails on old import path or doc behavior | `go test ./cmd/progress ./cmd/repoctl ./internal/progress/... ./internal/runtime/... -count=1` |
| Runtime process move | public caller test proves timeout/stdout/stderr semantics before move | `go test ./internal/runtime/... ./internal/acp -run 'Command|Runner|Timeout|ACP' -count=1` |
| Any progress row edit | `go run ./cmd/progress validate` before edit | `go run ./cmd/progress write && go run ./cmd/progress validate` |
| Any slice | `git diff --check` | `git diff --check` |

## Anti-Candidates

Do **not** do these as part of the topology push:

- Do not merge `kernel`, `gateway`, `hermes`, `config`, `tui`, `memory`,
  `session`, `skills`, `plugins`, or `tools` into a mega-package. These are
  deep modules with public behavior and tests.
- Do not copy Hermes' Python monolith shape. `hermes_cli/main.py` and
  `gateway/run.py` are source evidence for domain ownership, not a template for
  giant Go files.
- Do not collapse all channel adapters into one `channels` package. The real
  seam is `internal/channels/<platform>`.
- Do not move packages only to satisfy a number. If the deletion test says the
  package is useful but the new home is unclear, leave it and document the
  blocker.
- Do not change command output, channel output, provider behavior, persisted
  file layout, or progress schema during package moves.
- Do not update generated docs or codemaps before the code move is green.
- Do not introduce broad compatibility aliases from old internal import paths;
  update imports and let topology tests prevent backsliding.
- Do not move `modelcatalog`, `profileseed`, `contextrefs`, `loopcost`, or
  `i18n` in the first pass. They need separate caller evidence.

## Plan Completion Audit

Use this audit before declaring the topology-planning goal complete or handing
implementation to builders:

- **Hermes resemblance evidence:** every top-level owner family in the target
  shape maps either to an exact Hermes source family (`hermes_cli`, `gateway`,
  `tools`, `agent`, `providers`, `skills`) or is explicitly marked Gormes-owned
  (`progress`, `runtime`, `goncho`).
- **Depth evidence:** every proposed move names the deeper owner module, the
  caller-facing interface, the implementation hidden behind it, and the deletion
  test result.
- **Safety evidence:** every first slice has a RED proof, focused validation,
  forbidden scope, and rollback-friendly write scope.
- **Backlog evidence:** implementation work exists only as progress-row seeds;
  no side TODO queue in this file is treated as executable.
- **Dirty-work evidence:** builders can implement one row without touching
  unrelated changes outside the row's write scope.
- **Validation evidence:** `go run ./cmd/progress validate` and
  `git diff --check` pass after plan edits; package tests are reserved for
  implementation rows.

If any item is missing, keep iterating on this plan. If all items are satisfied,
the next safe action is a planner pass that inserts the topology guard row into
`progress.json`, followed by a TDD builder row.

## Top Recommendation

Start with **CLI Surface Enclave** plus a topology guard, then **Tool Adapter
Enclave**. This order best satisfies the user goal of a long-term structure that
resembles Hermes:

1. Hermes' `hermes_cli/*` source family maps most directly to Gormes'
   command-surface split, and `internal/app/gormescli` is the clearest shallow
   top-level mismatch.
2. The command manifest tests already exist, so the first row can be safe and
   small.
3. Tool adapter consolidation then gives the largest folder-count win while
   following Hermes' `tools/*` family.
4. Channel consolidation follows because it fixes a user-visible manifest
   inconsistency and mirrors Hermes' `gateway/platforms/*` family.
5. Progress delivery consolidation should follow after smaller moves prove the
   topology guard because it touches the self-development control plane and many
   path literals.

The first builder packet should be:

```text
architecture_review_packet:
  area: cmd/gormes + internal CLI topology
  candidate: CLI Surface Enclave
  score: 9/10
  smell: shallow top-level package plus split command-surface contracts
  evidence_quality: B source, with existing command-manifest tests available
  preserve_contracts:
    - visible Cobra command paths and help text
    - setup section registry
    - slash command active-turn policy
    - JSON report shapes and build provenance
  characterization_test:
    name: internal topology guard forbids internal/app after active CLI row
    command: go test ./cmd/gormes ./internal/cli/... -run 'CLIContract|Root|Setup|Profile|Provider|Gateway|Channels|Session|Navivox|CommandRegistry' -count=1
  allowed_write_scope:
    - internal/app/gormescli
    - internal/cli/gormescli
    - internal/cli/command_registry.go
    - cmd/gormes imports only
    - internal/channels/navivox/profile_contacts.go imports only
  forbidden_scope:
    - command behavior changes
    - flag or JSON report changes
    - gateway/channel runtime behavior
    - progress schema changes
  next_skill: gormes-tdd-slice
```
