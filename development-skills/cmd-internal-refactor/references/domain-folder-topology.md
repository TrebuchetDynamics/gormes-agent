# cmd/gormes Internal App Topology

Use this reference after selecting one command domain for `cmd-internal-refactor`.

## Final topology

```text
cmd/gormes/
  main.go                         executable shim and root factories only
  # no other tracked root entries in final topology; rehome codemap.md/docs

internal/platform/cli/gormescli/
  <domain>.go                     Cobra command wiring and command seams
  <domain>_test.go                CLI contract/golden tests and command helpers

internal/app/<domain>/
  service.go                      command-local behavior/orchestration
  command.go                      optional helper when useful
  types.go                        public options/results
  *_test.go                       behavior/unit/integration tests

internal/<runtime-domain>/         reusable runtime packages; do not pull up
```

## Ownership table

| Owner | Belongs there | Does not belong there |
|---|---|---|
| root `cmd/gormes` | `main.go` executable shim only, preserving `go run ./cmd/gormes` | any other tracked root entry in the final topology, including `codemap.md`; Cobra command construction, flags, args, help text, command registration, root exit-code wiring, CLI/golden tests, behavior-heavy helpers |
| `internal/platform/cli/gormescli` | Cobra command construction, flags, args, help text, command registration, exit-code seams, CLI/golden tests, command-tree helpers, root-facing facades | command-domain behavior, reusable runtime internals |
| `internal/app/<domain>` | command-domain options/results, orchestration, command-local formatting/path/env helpers, behavior tests | Cobra command trees, root helpers, reusable gateway/channel/provider/session/persistence/TUI/tool internals |
| deeper `internal/` package | reusable runtime services and shared contracts | command-only Cobra wiring or one-off CLI formatting |

## Dependency direction

Allowed direction:

```text
cmd/gormes/main.go -> gormescli -> internal/app/<domain> -> deeper internal packages
```

Rules:

- App packages must not import `cmd/gormes`, `gormescli`, or other platform CLI packages.
- `gormescli` can adapt root seams and call `internal/app/<domain>`.
- Root `main.go` should depend on `gormescli` facades where possible to keep direct-internal import budget green.
- If extracting a helper would create an app→CLI cycle, move that helper down to `internal/app/<domain>` or deeper runtime instead.
- Existing `cmd/gormes/<domain>` packages from earlier passes are migration candidates. Move one only when that exact domain is selected.

## Root file classification

Classify every selected-domain root entry before moving it:

1. **Executable shim**: only `main.go` stays in root.
2. **CLI wiring/contract**: Cobra command construction, flags, args, help text, completion, command-tree helpers, root command tests, and CLI/golden tests move to `gormescli`.
3. **Command-local behavior**: orchestration, command-specific options/results, path/env decisions already owned by the command, formatting helpers, and behavior tests move to `internal/app/<domain>`.
4. **Reusable runtime behavior**: gateway, provider, channel, persistence, session, memory, TUI, and tool internals stay in or move to deeper runtime packages.
5. **Documentation/codemap**: `cmd/gormes/codemap.md` cannot remain under a hard `main.go`-only topology. Before final audit, rehome its content to the root codemap, `cmd/codemap.md`, or an internal CLI/platform codemap and update references.
6. **Ignored local state**: local harness state such as `.pi` may be out of scope, but must be classified as ignored/out-of-scope rather than silently skipped.
7. **Ambiguous/dirty file**: stop, report the risk, or mark it out of scope unless the user explicitly selected that exact domain.

Package-spanning characterization tests may remain outside the selected slice only when they are explicitly classified as unrelated dirty work or next candidates. Tests-only slices are valid when behavior/wiring already lives under `internal/` and the root burn-down is moving contract tests to their owner package.

## Dirty relocation handling

When the worktree already contains a broad root-file relocation:

- do not reset, checkout, or overwrite dirty files;
- select one bounded domain whose existing dirty state can be completed and validated;
- inspect `git diff --name-status -- <domain files>` before editing;
- avoid domains with overlapping helpers unless you can preserve them through a facade;
- report validation limits caused by unrelated dirty files instead of claiming full closeout.

## Same-domain acceleration

A refactor pass is one domain, not one file. When the operator supplies a file list or says the burn-down is too slow, pick the largest same-contract cluster that can validate together. Examples: setup-first/quick setup, setup gateway profile bindings, setup tools, or setup profiles TUI. Do not join unrelated setup sections only because they share the `setup_` prefix.

For giant root files, move the selected subflow through a narrow root adapter and delete standalone root files for that subflow. Leave package-spanning e2e tests in root only when they genuinely cover multiple domains; classify them as deferred candidates rather than pretending the selected domain is complete.

## Candidate and count commands

Useful discovery commands:

```bash
# Repo-root-safe preflight. Use this even when the agent cwd is cmd/gormes.
repo=$(git rev-parse --show-toplevel)
git -C "$repo" status --short --branch --untracked-files=all

# Root topology count and names: files plus subdirectories, not only Go files
find "$repo/cmd/gormes" -maxdepth 1 -mindepth 1 -printf '%f\n' | sort
find "$repo/cmd/gormes" -maxdepth 1 -mindepth 1 | wc -l

# Go-only burn-down can be reported separately
find "$repo/cmd/gormes" -maxdepth 1 -type f -name '*.go' -printf '%f\n' | sort
find "$repo/cmd/gormes" -maxdepth 1 -type f -name '*.go' | wc -l

# Fast prefix inventory for candidate-table seeding; confirm manually before selecting.
find "$repo/cmd/gormes" -maxdepth 1 -type f -printf '%f\n' \
  | sed -E 's/(_test)?\.go$//; s/_(test|e2e)$//; s/_.*$//' \
  | sort | uniq -c | sort -nr

# Existing app/CLI owners that may make a tests-only burn-down possible
find "$repo/internal/app" "$repo/internal/platform/cli/gormescli" -maxdepth 2 -type f 2>/dev/null \
  | sed "s#^$repo/##" | sort

# Current direct internal imports from root package
( cd "$repo" && go list -f '{{join .Imports "\n"}}' ./cmd/gormes ) \
  | grep '^github.com/TrebuchetDynamics/gormes-agent/internal' | sort

# Focused dirty state for a domain
git -C "$repo" diff --name-status -- cmd/gormes internal/platform/cli/gormescli internal/app/<domain> \
  | grep -i '<domain>\|<command>'
```

Use the folder-refactor tools when the harness provides them:

```text
folder_refactor_scan target=/absolute/path/to/repo/cmd/gormes      # before planning a topology claim
folder_refactor_audit target=/absolute/path/to/repo/cmd/gormes     # before claiming root topology complete
```

Avoid cwd-relative scan targets when the agent is already inside `cmd/gormes`; they can resolve as `cmd/gormes/cmd/gormes`.

## Hermes graph query shape

Prefer targeted graph queries over loading the full graph:

```bash
repo=$(git rev-parse --show-toplevel)
test -r "$repo/hermes-knowledge-graph.json"
pattern='<domain>|<command>'
jq -r --arg re "$pattern" '
  .nodes[]
  | select(((.id // "")|test($re; "i"))
        or ((.filePath // "")|test($re; "i"))
        or ((.summary // "")|test($re; "i")))
  | [.id, (.filePath // ""), (.summary // "")] | @tsv
' "$repo/hermes-knowledge-graph.json"
```

If the query has no relevant hits, do not invent Hermes refs. A no-Hermes-analogue classification is allowed only for Gormes-owned domains with explicit local evidence such as repocheck source refs, progress docs, command tests, or code ownership. Graph refs are navigation evidence only. Preserve current Gormes behavior during refactor; report behavior drift separately.

## CLI boundary checklist

Before and after the move, preserve:

- command names, aliases, flags, defaults, args, help text, and shell completion;
- env vars, config paths, profile/home resolution, and filesystem side effects;
- stdout/stderr text, JSON fields, ordering, colors, and prompts;
- exit codes and error wording;
- test fixtures, golden output, and command-constructor freshness.

Use the same before/after oracle for the selected domain so refactors can reveal bugs instead of only proving compilation. If the oracle fails before editing, classify a preexisting bug; if it fails only after the move, classify an introduced regression and fix it before closeout.

## Validation patterns

Focused first:

```bash
# Baseline/after-move bug oracle for behavior slices; use the same command both times.
go test ./cmd/gormes -run '<Domain|Command>' -count=1
GORMES_HOME="$(mktemp -d)" go run ./cmd/gormes <command> --help

# Tests-only/root burn-down slice when behavior and wiring already live under internal/.
go test ./internal/platform/cli/gormescli -run '<Domain|Command>' -count=1
go test ./cmd/gormes -run '<Domain|Command>' -count=1

# Behavior extraction slice.
go test ./internal/app/<domain> -count=1
go test ./internal/platform/cli/gormescli -run '<Domain|Command>' -count=1
go test ./cmd/gormes -run '<Domain|Command>' -count=1

# Package/topology closeout for any slice.
go test ./cmd/gormes -count=1
go test ./cmd/gormes/... -count=1
go test ./internal/support/repochecks -run 'Cmd|Internal|Topology|Import' -count=1
go run ./cmd/progress validate
git diff --check
```

Broad only when safe:

```bash
timeout 20m go test ./... -count=1
timeout 20m go vet ./...
```

Do not use an empty `-run` selector as behavior evidence. If a package has no selected tests, run the package test without `-run` or explain why compile-only evidence is sufficient.
