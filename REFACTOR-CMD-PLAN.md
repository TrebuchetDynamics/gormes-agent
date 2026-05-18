# cmd/gormes Refactor Completion Note

Date: 2026-05-17

Status: **complete for the planned command-construction refactor.**

This file is now a completion record, not an active handoff and not a second
backlog. The executable backlog is still
`docs/content/building-gormes/architecture_plan/progress.json`, accessed through
`internal/progress.Load` / `SaveProgress` or `cmd/progress`.

## Final State

Branch must stay `development`. Do not create feature branches or worktrees for
Gormes work.

All command-construction slices in this plan are implemented:

- `internal/app/gormescli/contracts.go` defines an importable CLI contract
  registry.
- `internal/app/gormescli/default_contracts.go` maps the current command,
  setup, and slash surfaces to approved feature modules.
- `internal/app/gormescli/contract_test.go` proves invalid modules,
  overlapping command ownership, unowned command paths, unowned setup sections,
  and unowned slash commands fail closed.
- `cmd/gormes/cli_contract_manifest_test.go` derives visible command paths from
  the live Cobra root and validates them against the registry.
- The completed progress row is
  `CLI module contract registry and manifest gate` under module `cli`.
- `internal/app/gormescli/modules/profiles/profile.go` now owns the
  `gormes profile` command implementation.
- `cmd/gormes/profile.go` is now a thin compatibility adapter that injects
  build provenance and keeps existing local seam names available to setup,
  doctor, skills, and profile tests.
- The completed progress row is
  `cmd/gormes profile command package extraction` under module `cli`.
- `internal/app/gormescli/setup_registry.go` now owns setup section
  registration and validates setup section module owners against the contract
  registry.
- `internal/app/gormescli/modules/profiles/setup.go` and
  `internal/app/gormescli/modules/providers/setup.go` now declare their feature
  setup sections.
- The completed progress row is
  `cmd/gormes setup section registry extraction` under module `cli`.
- `internal/app/gormescli/modules/providers/` now owns provider command
  construction for `auth`, `logout`, `model`, `fallback`, `usage`, and
  `insights`.
- `cmd/gormes` keeps thin adapters for provider build provenance and for
  deeper runtime seams that are not part of this command-surface slice:
  auth/OAuth storage behavior and shared model prompt helpers.
- The completed progress rows are
  `cmd/gormes provider usage command package extraction` and
  `cmd/gormes provider command surface package extraction` under module
  `providers`.
- `internal/app/gormescli/modules/gateway/rowbacked.go` now owns the
  row-backed `webhook`, `hooks`, and `pairing` command trees.
- The completed progress row is
  `cmd/gormes gateway row-backed command package extraction` under module
  `gateway`.
- `internal/app/gormescli/modules/channels/capabilities.go` now owns
  `gormes channels capabilities` command construction and rendering.
- The completed progress row is
  `cmd/gormes channels capabilities command package extraction` under module
  `channels`.
- `internal/app/gormescli/modules/gateway/` now owns live `gateway`,
  `dashboard`, and `agent` command construction.
- `cmd/gormes` keeps runtime seams for gateway lifecycle, dashboard serving,
  and dynamic-agent registry behavior.
- The completed progress row is
  `cmd/gormes live gateway command package extraction` under module `gateway`.
- `internal/app/gormescli/modules/channels/` now owns `slack`, `whatsapp`,
  and `telegram` command construction.
- `cmd/gormes` keeps runtime seams for Slack manifest rendering/writes,
  WhatsApp pairing/install behavior, and Telegram long-poll startup.
- The completed progress row is
  `cmd/gormes channel service command package extraction` under module
  `channels`.
- `internal/app/gormescli/root.go` now owns root Cobra assembly, root flags,
  root help text, finalizer hook execution, and canonical top-level command
  ordering.
- `cmd/gormes/main.go` now defaults runtime seams and supplies command
  factories while keeping process entry, panic dump, exit-code mapping, and
  ldflags/build values.
- The completed progress row is
  `cmd/gormes root command assembly extraction` under module `cli`.

Verified during the closeout pass:

```sh
go test ./... -count=1
go run ./cmd/progress validate
git diff --check
```

## Original Problem

At the start of this plan, `cmd/gormes` was too broad: binary package, root
command builder, setup wizard, feature command implementations, runtime seams,
and operator policy surface all lived in one package. That made ownership
unclear: profiles, providers, channels, gateway, setup, TUI, sessions, tools,
and install behavior could all drift inside one package.

The completed `gormescli` contract gate stops unowned drift for the migrated
command-construction layer. Profile, setup, provider, gateway, channel,
session/checkpoint, and root command construction now have feature-owned
package boundaries. Runtime seams intentionally remain in `cmd/gormes` where
they were declared out of scope for this plan.

## Completed Architecture

`cmd/gormes` is now process-entry and runtime glue for this refactor boundary:

```text
cmd/gormes/
  main.go              os.Args, panic dump, exit code mapping, ldflags version,
                       command factories, and runtime seam defaults
```

Command/control-plane assembly now lives in importable app packages:

```text
internal/app/gormescli/
  contracts.go         module ownership registry and manifest gate
  default_contracts.go temporary central contract map
  root.go              NewRootCommand, root flags, help, command ordering
  runtime.go           importable root option/factory types
  setup_registry.go    setup section registration and validation
  modules/
    profiles/          profile command + setup profiles section
    providers/         auth/logout/model/fallback/usage/insights/setup provider
    gateway/           gateway/webhook/hooks/pairing/dashboard/agent surfaces
    channels/          telegram/slack/whatsapp/channels surfaces
    sessions/          session/checkpoint command surfaces
    tools/             future command-construction extraction
    skills/            future command-construction extraction
    tui/               future command-construction extraction
```

Long term, each module package should expose a small registration surface:

```go
type Module interface {
	Contract() gormescli.ModuleContract
	Commands(ctx BuildContext) []*cobra.Command
	SetupSections(ctx BuildContext) []SetupSection
}
```

Do not force that abstraction from this completed plan. The current extracted
packages use typed constructors and seams; a generic `Module` interface remains
future work only if more modules converge on the same shape.

## Non-Negotiables

- Preserve all public command spelling, flags, JSON shapes, and exit codes
  unless a progress row explicitly changes them.
- Keep `cmd/gormes` tests green while moving code. Prefer moving tests with the
  package only after the package boundary exists.
- Do not introduce `cmd/gormes` imports from internal packages. Direction must
  be `cmd/gormes -> internal/app/gormescli -> feature packages`.
- Do not create a parallel TODO list. Before implementation, add or refine a
  progress row for the exact slice.
- Keep setup/profile/provider/channel changes hermetic: temp `GORMES_HOME`, no
  live providers, no network credentials.

## Finish Line Decisions

The finish line for this plan is **command construction extraction**, not
deeper runtime extraction.

In scope:

- Move live `gateway`, `dashboard`, and `agent` command construction into the
  gateway module.
- Move `slack`, `whatsapp`, and `telegram` command construction into the
  channels module, in that order.
- Move root command assembly into `internal/app/gormescli`, leaving
  `cmd/gormes/main.go` as process-entry glue.
- Keep temporary `cmd/gormes` compatibility shims only for build provenance,
  existing tests, and runtime seams that have not moved yet.

All in-scope items are complete.

Out of scope for this plan:

- Deeper auth/OAuth storage/runtime extraction.
- Gateway manager/kernel/channel runtime extraction.
- Model prompt helper and Bubble Tea picker extraction.
- Generic `Module` interface introduction.
- Release/install/commit/push work.

Root assembly ownership decision:

- `internal/app/gormescli` owns Cobra root assembly for now.
- Feature packages expose constructors and typed options/seams.
- Do not force a generic module-registration interface until several extracted
  modules expose the same shape naturally.

## Completed Slices

1. Extract setup section registration. **Done.**

   Replace the global `setupSections` list with a registry that can be
   validated by `gormescli.Registry`. Move the `profiles` setup section into
   the profile module. This is the slice that unlocks the rich
   `gormes setup profiles` TUI without making setup another monolith.

   Required behavior:

   - Existing setup section order stays stable.
   - Boxed setup chrome stays shared.
   - Missing section owner fails in tests.
   - `gormes setup profiles` remains Gormes-owned, not Hermes parity.

2. Move provider-owned command surfaces. **Done at command-construction layer.**

   `auth`, `logout`, `model`, `fallback`, `usage`, and `insights` command
   construction now lives in the providers module. Credential redaction,
   provider setup, usage, fallback, model picker, logout, and auth fixtures
   stay green through `cmd/gormes` adapters. The deeper auth/OAuth runtime and
   shared model prompt helpers remain injected seams and should be moved by
   separate runtime/TUI rows if needed.

3. Move gateway and channel surfaces separately. **Done at command-construction layer.**

   Gateway now owns row-backed `webhook`, `hooks`, and `pairing` plus live
   `gateway`, `dashboard`, and `agent` command construction. Channels owns
   `channels capabilities` plus `slack`, `whatsapp`, and `telegram` command
   construction.

   Runtime internals intentionally remain injected seams: gateway manager
   lifecycle, dashboard HTTP serving, dynamic-agent registry behavior, Slack
   manifest rendering/writes, WhatsApp pairing/install behavior, and Telegram
   long-poll startup still live in `cmd/gormes` until separate runtime rows
   move them.

4. Move session/checkpoint surfaces. **Done at command-construction layer.**

   Sessions now owns `session`/`sessions` plus `checkpoints` command tree
   construction. Transcript export, session directory mutation, checkpoint
   status/prune/clear behavior, confirmations, and JSON payload rendering stay
   in `cmd/gormes` runtime handlers behind injected seams.

5. Move root command assembly last. **Done at command-construction layer.**

   `internal/app/gormescli.NewRootCommand` now owns root flags, help text,
   top-level command order, and finalizer hook execution. `cmd/gormes/main.go`
   still owns `rootRuntime` defaulting, runtime callbacks, command factories,
   process entry, panic dump, exit-code mapping, and build values. A later
   runtime-extraction plan can decide whether `rootRuntime` becomes importable
   or remains binary-owned glue.

## No Remaining Work In This Plan

Do not keep extending this file with new slices. The planned command
construction extraction is closed.

Future work must be filed as progress rows or a new bounded plan when it has a
real target and tests. Likely future plans, if needed:

- tools/MCP/ACP command construction extraction;
- skills/plugins/curator command construction extraction;
- chat/admin/TUI setup command construction extraction;
- deeper auth/OAuth, gateway manager, dashboard server, Telegram/Slack/WhatsApp
  runtime seam extraction;
- a generic `Module` interface, only after repeated constructor shapes justify
  it.

Until then, `cmd/gormes` retaining runtime seams is intentional, not unfinished
work from this plan.

## Historical Progress Row Template

This is retained as historical context only. Do not execute it as another
slice from this plan.

```text
name: cmd/gormes setup section registry extraction
module: cli
execution_owner: tools
slice_size: small
contract_status: fixture_ready
contract: Replace the global cmd/gormes setup section list with an importable
  gormescli setup registry validated by the module contract gate while
  preserving section order, boxed chrome, and existing setup section behavior.
write_scope:
  - cmd/gormes/setup.go
  - cmd/gormes/setup_*_test.go
  - internal/app/gormescli/setup_registry.go
  - internal/app/gormescli/modules/profiles
  - internal/app/gormescli/default_contracts.go
test_commands:
  - go test ./cmd/gormes ./internal/app/gormescli ./internal/app/gormescli/modules/profiles -run 'Setup|Profile|CLIContract' -count=1
  - go test ./cmd/gormes ./internal/app/gormescli ./internal/app/gormescli/modules/profiles ./internal/cli -count=1
  - go run ./cmd/progress validate
  - git diff --check
```

## Historical Final Gate

These were the gates used for command-construction slices. Keep them as
reference only; this plan has no remaining slices.

```sh
go test ./cmd/gormes ./internal/app/gormescli ./internal/cli ./internal/progress -count=1
go run ./cmd/progress validate
git diff --check
```

`go test ./... -count=1` was run before this plan was marked complete.
