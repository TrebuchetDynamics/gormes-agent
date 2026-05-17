# cmd/gormes Refactor Continuation Plan

Date: 2026-05-17

This is a continuation handoff, not a second backlog. The executable backlog is
still `docs/content/building-gormes/architecture_plan/progress.json`, accessed
through `internal/progress.Load` / `SaveProgress` or `cmd/progress`.

## Current State

Branch must stay `development`. Do not create feature branches or worktrees for
Gormes work.

Six slices are implemented:

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

Verified before this handoff:

```sh
go test ./... -count=1
go run ./cmd/progress validate
git diff --check
```

## Problem

`cmd/gormes` is still too broad. It is currently a binary package, root command
builder, setup wizard, feature command implementations, runtime seams, and
operator policy surface all at once. That makes ownership unclear: profiles,
providers, channels, gateway, setup, TUI, sessions, tools, and install behavior
can all drift inside one package.

The new `gormescli` contract gate stops unowned drift, and the profile command
is the first extracted feature-owned command package. Setup, provider, gateway,
channel, session, tools, skills, TUI, and install surfaces still need the same
treatment.

## Target Architecture

Keep `cmd/gormes` as the thin process entrypoint:

```text
cmd/gormes/
  main.go              os.Args, panic dump, exit code mapping, ldflags version
```

Move command/control-plane assembly into importable app packages:

```text
internal/app/gormescli/
  contracts.go         module ownership registry and manifest gate
  default_contracts.go temporary central contract map
  app.go               NewRootCommand / Execute entrypoint after migration
  runtime.go           RootRuntime and top-level runtime seams
  setup_registry.go    setup section registration and validation
  modules/
    profiles/          profile command + setup profiles section
    providers/         auth/logout/model/fallback/usage/insights/setup provider
    gateway/           gateway/webhook/hooks/pairing/dashboard/agent surfaces
    channels/          telegram/slack/whatsapp/channels surfaces
    sessions/          session/checkpoints/chat resume surfaces
    tools/             tools/mcp/acp surfaces
    skills/            skills/plugins/curator surfaces
    tui/               chat/admin/terminal setup surfaces
```

Long term, each module package should expose a small registration surface:

```go
type Module interface {
	Contract() gormescli.ModuleContract
	Commands(ctx BuildContext) []*cobra.Command
	SetupSections(ctx BuildContext) []SetupSection
}
```

Do not force that abstraction too early. Extract one feature slice first and
let the interface harden around real needs.

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

## Recommended Next Slices

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

3. Move gateway and channel surfaces separately. **In progress.**

   Completed so far: gateway owns row-backed `webhook`, `hooks`, and `pairing`;
   channels owns `channels capabilities`.

   Remaining: gateway should own live `gateway`, `dashboard`, and `agent`
   command/runtime surfaces. Channels should own `telegram`, `slack`, and
   `whatsapp` service command surfaces. Do not mix these two modules just
   because they currently share gateway runtime helpers.

4. Move root command assembly last.

   Only after several feature packages exist, move `newRootCommandWithRuntime`
   and `rootRuntime` into `internal/app/gormescli`. `cmd/gormes/main.go` should
   then become a small wrapper around `gormescli.Execute`.

## Progress Row Template

Use this shape before executing the next slice:

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

## Final Gate For Any Slice

Run the row-local tests first, then:

```sh
go test ./cmd/gormes ./internal/app/gormescli ./internal/cli ./internal/progress -count=1
go run ./cmd/progress validate
git diff --check
```

Run `go test ./... -count=1` before handing off a broad package move or before
claiming the branch is ready to commit.
