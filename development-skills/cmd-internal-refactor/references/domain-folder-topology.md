# cmd/gormes Internal App Topology

Use this reference after selecting one command domain for `cmd-internal-refactor`.

## Ownership table

| Owner | Belongs there | Does not belong there |
|---|---|---|
| root `cmd/gormes` | Cobra command construction, flags, args, help text, command registration, root exit-code wiring, CLI/golden tests | behavior-heavy helpers, reusable service logic, domain behavior tests |
| `internal/app/<domain>` | command-domain options/results, orchestration, formatting helpers, filesystem/env decisions already owned by the command, behavior tests | gateway/channel/provider/session/persistence/TUI/tool runtime internals |
| existing deeper `internal/` package | reusable runtime services, gateway/channel/provider/session/persistence/TUI/tool behavior | command-only Cobra wiring or one-off CLI formatting |

## Package rules

- Use `internal/app/<domain>` with a valid lowercase Go package name such as
  `uninstall`, `logs`, `admin`, or `authcodex`.
- Record command-to-package mapping when command words do not equal a valid Go
  identifier, e.g. `auth codex` -> `authcodex`.
- The app subpackage must not import the root `cmd/gormes` package. Root
  wiring may import the app subpackage.
- Do not pull behavior out of deeper runtime packages into `internal/app/<domain>`.
- Existing `cmd/gormes/<domain>` packages from earlier extraction passes are
  migration candidates. Move one back to `internal/app/<domain>` only when that
  exact domain is selected and the migration is the one bounded refactor.

## Many-file domains

For files like `uninstall.go`, `uninstall_paths.go`, `uninstall_prompt.go`, and
`uninstall_test.go`, do not bulk-move by filename. Classify each file:

1. **Root wiring**: Cobra command construction, flags, help text, args, and CLI
   contract tests stay in root `cmd/gormes`.
2. **Command-local behavior**: pure orchestration, path/env decisions already
   owned by the command, output formatting helpers, and behavior tests move to
   `internal/app/<domain>`.
3. **Reusable runtime behavior**: gateway, provider, channel, persistence,
   session, TUI, and tool internals stay in their deeper `internal/` packages.
4. **Ambiguous or dirty files**: stop, report the risk, or skip the domain unless
   the user explicitly asked to continue that exact dirty domain.

## CLI boundary checklist

Before and after the move, preserve:

- command names, aliases, flags, defaults, args, help text, and shell completion;
- env vars, config paths, profile/home resolution, and filesystem side effects;
- stdout/stderr text, JSON shapes, ordering, colors, and prompts;
- exit codes and error wording;
- test fixtures and golden output.
