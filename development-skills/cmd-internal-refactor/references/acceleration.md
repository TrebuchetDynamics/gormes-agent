# cmd/gormes Refactor Acceleration Guide

Use this when the operator says the pass is too slow, asks to refactor all remaining files, or supplies a root-file list. The goal is fewer micro-passes without crossing command-domain boundaries.

## Core rule

One pass still owns exactly one domain, but a domain can be a same-contract cluster rather than a single file.

Good domain clusters:

- `setup-first/quick`: first-run quick setup helpers, target picker facade, setup-first behavior tests, and marker tests that point to the moved owner.
- `setup gateway profiles`: profile/channel binding helpers plus their behavior tests and `gormescli` facade.
- `setup tools`: tool-section behavior, directly owned tests, and CLI/TUI facade.
- `setup profiles TUI`: profile-control-center TUI model, apply functions, directly owned TUI tests, and narrow root adapters if `setup.go` still owns other setup sections.

Bad clusters:

- `setup.go` wholesale when it still contains provider, model, gateway, profiles, tools, TTS, terminal, and migration subflows.
- A file list joined only by filename prefix when tests touch unrelated providers/gateway/TUI contracts.
- Moving package-spanning e2e tests before their root command factory seams are available in `gormescli`.

## Fast candidate triage

From repo root:

```bash
repo=$(git rev-parse --show-toplevel)
listed='setup_bubbletea_test.go setup_profiles_tui.go setup_profiles_tui_test.go'
for f in $listed; do test -e "$repo/cmd/gormes/$f" && echo "present $f" || echo "gone $f"; done
rg -n 'selectedSymbol|selectedFunction|selectedType' "$repo/cmd/gormes" "$repo/internal/app" "$repo/internal/platform/cli/gormescli"
```

Build a small table only for listed candidates and directly referenced helpers:

```text
domain cluster | listed files covered | required helpers | graph/local evidence | validation | decision
```

Pick the largest row that has:

- one user-visible command/section contract;
- no live credentials or external services;
- package tests that can run without full-repo green;
- a clear app package and optional `gormescli` facade;
- no app -> CLI import cycle.

## Acceleration patterns

### Move behavior plus tests together

Do not leave a root test file behind solely because it refers to unexported names. Either:

- move the behavior test to `internal/app/<domain>` and test exported app APIs;
- move CLI/golden tests to `internal/platform/cli/gormescli`; or
- keep a package-spanning root e2e test only when it exercises multiple domains and classify it as deferred.

### Use narrow root adapters for giant root files

If a selected subflow lives inside `setup.go`, add a narrow root adapter that maps existing `setupCommandSeams`/Cobra details into a `gormescli` facade. The behavior should live below:

```text
cmd/gormes/setup.go thin adapter -> internal/platform/cli/gormescli/<domain>.go -> internal/app/<domain>
```

Delete the standalone root file for the selected domain when the adapter replaces it.

### Validate once, then close out

Acceleration does not skip bug discovery: capture one baseline oracle for the whole selected cluster before moving it, then rerun the same oracle after extraction. A newly failing oracle is an introduced regression, not a reason to weaken expectations.

Run focused tests first, preferably in parallel:

```bash
go test ./internal/app/<domain> -count=1
go test ./internal/platform/cli/gormescli -run '<Domain|Command>' -count=1
go test ./cmd/gormes -run '<Domain|Command|legacy marker>' -count=1
```

After fixes, run package/topology closeout once:

```bash
go test ./cmd/gormes -count=1
go test ./cmd/gormes/... -count=1
go test ./internal/support/repochecks -run 'Cmd|Internal|Topology|Import' -count=1
go run ./cmd/progress validate
git diff --check -- <selected files>
```

Do not claim full-repo green when unrelated dirty work is active.

## Stop conditions

Stop and report a blocker when speed would require:

- merging two command contracts in one pass;
- changing CLI output, config shape, prompt wording, persistence, or gateway behavior;
- making an app package import `cmd/gormes`, `gormescli`, or platform CLI packages;
- relying on live credentials or operator state;
- deleting shared root helpers without accounting for all callers.
