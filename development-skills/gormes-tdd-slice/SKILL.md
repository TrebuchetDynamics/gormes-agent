---
name: gormes-tdd-slice
description: Use when building or fixing a Gormes feature with tests, when a parity evidence atom needs implementation, when the user asks for TDD or red-green-refactor, or when implementing Goncho/Hermes parity behavior one vertical slice at a time.
---

# Gormes TDD Slice

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Ship one narrow Gormes behavior with a red-green-refactor loop. Tests must verify public behavior, not implementation details.

For Hermes parity bugs, the RED test must encode the upstream-visible behavior
or artifact, including negative visual/output details. "It works" is not enough
when the bug is duplicate output, stale labels, boxed chrome, hidden tool-call
noise, or platform-visible formatting.

Hermes Agent is the Python upstream/father implementation for Gormes. Prefer
the in-repo checkout at `./hermes-agent`; fall back to `../hermes-agent` only
when absent. Resolve it as `$HERMES_SRC` before writing parity tests.

## Workflow

### 1. Select One Behavior

Use the selected atom from `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md` or choose one that is `missing` or `partial`. State:

- public interface under test;
- feature-map target or upstream concept;
- behavior to prove;
- Hermes source ref (`HERMES` column in the atom);
- Gormes target package (`GORMES` column in the atom);
- allowed write scope.

If the atom is too broad, split it and update the parity evidence doc before coding.

For UI/TUI/channel parity, identify the active upstream implementation before
writing the test. Current full-screen TUI UX comes from
`$HERMES_SRC/ui-tui`, while older `cli.py` prompt_toolkit code is only a
legacy/detail donor when current Ink does not cover the behavior.

### 2. RED

Write one failing test for one observable behavior. Prefer:

- command output or exit behavior for CLI;
- request/response behavior for API/gateway;
- public Go package behavior for provider/tools/memory/session;
- compatibility request/response for Goncho/Honcho;
- hermetic fixtures over live network/provider calls.

The failing test should prove the feature-map behavior through a public
contract. Avoid private helper tests unless the progress row explicitly makes
that helper the exported contract for the slice.

Run the focused test and confirm it fails for the right reason.

For visual/output regressions, prefer explicit artifact tests:

- absence of stale product labels such as `Hermes` where Gormes owns the label;
- absence of duplicate assistant messages or history+draft double-rendering;
- absence of debug/status noise in idle chrome;
- absence of old boxes/rules/prompt rows when current Hermes no longer renders
  them;
- presence and order of user-visible rows, not private helper state.

For Telegram or other channel bugs, turn the user's transcript into a fixture.
Assert the outbound operation sequence: status sends, edits/deletes, tool-call
visibility, final-message count, and final text. A correct final answer is not
green if an extra hourglass, leaked tool progress, or duplicate final reply was
also sent.

For gateway tool-progress bugs, first decide which Hermes surface defines the
artifact. Emoji/snake_case lines such as `📚 skill_view: "plan"` are gateway
progress from `$HERMES_SRC/gateway/run.py` plus
`agent.display.build_tool_preview`; current Ink TUI shelves render Title Case
tool calls such as `Read File("x")`. A regression test must encode the right
surface, including `new` vs `all`, preview truncation, `(×N)` collapse,
`todo merge=true` wording, and unknown-tool degradation.

For tool-loop regressions, test the boundary that leaked to the user. Kernel
budget bugs belong in `internal/kernel` and should prove Hermes' 90-turn
default plus toolless summary fallback. Channel bugs belong in gateway/channel
tests and should prove raw budget errors, tool-call XML, or provider repair
diagnostics do not become final user messages.

For dashboard/API security parity, encode both sides of the boundary in the RED
test: public read-only allowlist routes (for example `/api/status`) must remain
reachable, while sensitive dashboard `/api/` routes must reject missing session
tokens and accept the active Hermes header such as `X-Hermes-Session-Token`
plus any legacy Bearer compatibility the upstream source still supports. When a
new guard changes existing contract tests, update those fixtures to pass the
same token instead of weakening the guard or treating the old unauthenticated
expectation as product behavior.

For runtime-lock or installer-adjacent tests, isolate state with
`GORMES_HOME` temp dirs unless the behavior under test is the real persisted
home. Never make a test pass by depending on the developer's live
`workspace-gormes`, `workspace-mineru`, `~/.gormes`, or `~/.hermes` paths.

### 3. GREEN

Write the smallest implementation that passes this test. Stay inside write scope. Do not add speculative future behavior.

### 4. Repeat Vertically

Add the next behavior only after the prior test is green. Do not write a batch of imagined tests first.

For broad builder rows that enumerate several equivalent adapters or providers, a
cron-sized TDD pass may ship one source-backed adapter/provider as the vertical
slice when the row contract is otherwise too large for one safe run. In that
case:

- keep the progress row planned unless every acceptance bullet is now satisfied;
- update the row note with the exact partial evidence and remaining adapters;
- add a search/aggregation-level regression when the slice introduces a new
  typed degraded condition, not only a provider-local test;
- do not mark umbrella or multi-provider rows complete from one provider's green
  tests.

### 5. Refactor While Green

Only refactor after tests pass. Prefer deep modules: small interface, substantial hidden implementation, clear locality.

### 6. Verify

Run focused package tests, then `go test ./... -count=1` for the affected packages, and the gates in `references/gates.md`.

If the working tree contains unrelated user or parallel-agent changes, do not
revert them. Run the focused tests for the selected behavior first, then broaden
only as far as the current tree permits. If unrelated failures block a full
gate, report them separately with file paths and failure commands.

## Final Report

Report red-green cycles, feature-map target, behavior shipped, tests run, the parity status before/after, and the `HERMES` source ref used to verify the behavior.
