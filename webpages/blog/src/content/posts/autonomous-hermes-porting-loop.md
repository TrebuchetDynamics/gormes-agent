---
title: "Autonomous Hermes-porting loop"
date: "2026-05-13"
summary: "How TrebuchetDynamics turns Gormes progress rows into small, test-proven Hermes-parity slices."
---

# How an autonomous loop ships Hermes-parity slices

This is the field note I wanted before Gormes existed: not another claim that an
agent can write code, but a receipt for how an agent can keep shipping slices of
an existing system without turning the repository into a pile of unreviewable
magic. Gormes is the artifact. The deeper product is the loop that ports Hermes
behavior into a Go-native runtime while preserving a human-readable contract for
every step.

The loop is deliberately boring. It does not start with a blank prompt that says
"improve the codebase." It starts with a row in one canonical backlog,
`progress.json`. A planner sharpens that row until the interface is clear: what
behavior is missing, where Hermes or another source proves it, which Gormes
module owns the implementation, which files are allowed to change, and which
commands prove the row. A builder then takes one ready row, and a TDD slice
writes the smallest failing test that demonstrates the missing behavior. The
implementation is not allowed to count as shipped until the focused test, the
package gate, `go test ./... -count=1`, `go run ./cmd/progress validate`, and
`git diff --check` pass for the slice.

The useful invariant is:

```text
Intent → Oracle → Surface → Work package → Proof
```

Intent names the behavior the reader or operator should get. Oracle names the
thing that proves the behavior is real: upstream Hermes source, a local test, a
public command, a docs artifact, or an explicit owner decision. Surface names
where the behavior crosses into reality: CLI output, gateway events, installer
behavior, TUI state, or a public docs page. Work package names the files and the
allowed seam. Proof names the commands that somebody else can rerun.

## Architecture diagram

```text
                ┌─────────────────────────────────────────┐
                │ progress.json: one logical backlog       │
                │ row = contract + source refs + gates     │
                └────────────────────┬────────────────────┘
                                     │
                                     ▼
┌────────────────┐      ┌────────────────────┐      ┌────────────────┐
│ Planner pass   │─────▶│ Builder pass       │─────▶│ TDD slice      │
│ sharpen row    │      │ choose one row     │      │ RED → GREEN    │
│ exact refs      │      │ stay in scope      │      │ refactor       │
└───────┬────────┘      └─────────┬──────────┘      └───────┬────────┘
        │                         │                         │
        ▼                         ▼                         ▼
┌────────────────┐      ┌────────────────────┐      ┌────────────────┐
│ Generated docs │◀─────│ progress write     │◀─────│ validation     │
│ roadmap/status │      │ sync evidence      │      │ tests + diff   │
└────────────────┘      └────────────────────┘      └────────────────┘
```

The diagram is intentionally a loop, not a pipeline. Every shipped slice feeds
back into `progress.json` and the generated progress docs so the next agent can
start from repository state instead of chat memory. That is the main difference
between an autonomous demo and an autonomous engineering loop. The loop leaves a
map for the next worker.

## Walkthrough 1: a release-safety bug becomes a small installer seam

Commit `2e3a56a21` is a good example because it is not glamorous. The problem
was not that Gormes lacked a feature. The problem was that a Termux-style install
could reach outside its prefix and update an active `gormes` command elsewhere
on `PATH`. For a developer laptop this might look like a convenience. For a
phone or isolated prefix it is a safety bug: the install boundary is supposed to
mean something.

The loop did not solve that with a broad installer rewrite. It found the seam:
`update_active_command` in `install.sh`. The implementation added one guard:
when `is_termux` is true, skip the active PATH command update and log that the
installer is respecting `$PREFIX/bin`. The dry-run output changed from saying it
would adopt an existing command on PATH to saying the active command update is
skipped for Termux.

The proof lived next to the behavior. `internal/installtest/termux_path_safety_test.go`
kept the fixture focused on the install boundary, not on a live Android device.
That matters because the loop can run the same test on CI or a developer
machine. The row later broadened to full installer E2E, but the first slice was
small enough to reason about: if Termux is the runtime, the installer must not
mutate a command outside the prefix.

This is the pattern I want from the loop: identify one real operator failure,
place the fix behind the smallest interface that owns it, and make the proof
hermetic before touching public release claims.

## Walkthrough 2: a Navivox UX paper cut gets a compatibility alias

Commit `5cd4f870f` shows a different kind of slice. Navivox users needed a
visible `gormes navivox connect` command. The older command name,
`connect-info`, described an implementation detail. The user-facing surface was
connection: print the reachable HTTP URLs, health URL, WebSocket URL, and QR
payload for a mobile client.

The fix changed the command registration from one command to two adapters at the
same seam: `connect` as the public command and `connect-info` as a hidden legacy
alias. The test `TestNavivoxCommandHelpUsesConnectNotConnectInfo` encoded both
sides of the contract. Help output must advertise `connect`, must not advertise
`connect-info`, and the old alias must still resolve for scripts or operators
who already learned it.

That is an important detail. Autonomous loops are tempted to optimize for the
new happy path and forget compatibility. This slice kept compatibility explicit.
The implementation also changed error text from `navivox connect-info` to
`navivox connect`, because user-visible diagnostics are part of the interface.
If a blind operator is following terminal output, the command name in the error
message is not cosmetic. It is navigation.

The broader progress docs changed around the code, too: Navivox generated
roadmap pages and CLI docs moved with the feature. That is why the loop treats
`go run ./cmd/progress validate` as a real gate. The code and the status surface
must agree, or the next worker starts from a lie.

## Walkthrough 3: the loop also records non-feature receipts

Commit `c2a267dad` is intentionally less perfect. It recorded installer loop
progress in `.pi/development-loop/logs.jsonl`. There is no shiny runtime diff.
There is an operational receipt: what the loop attempted, which commands it ran,
which files changed, why push was not attempted, and what should happen next.

That is not a substitute for code. It is what makes code changes auditable when
there are many overlapping local slices. Gormes often has a dirty worktree
because several development-goal iterations are preserving unrelated work. The
loop needs to say, in machine-readable form, which files belong to which slice
and which validation commands were run. Otherwise an agent can accidentally
revert someone else's work, overclaim a row, or commit a mixed bundle that no
reviewer can untangle.

The tradeoff is obvious: logging progress does not move the product by itself.
But without receipts, autonomous work becomes folklore. A loop that only writes
features and never writes evidence is not a loop I would trust on a production
port.

## The validation gate is the product boundary

For Gormes, validation is not one command. It is a ladder.

1. A focused test proves the exact surface changed: a command, installer path,
   gateway event, TUI behavior, docs page, or public build artifact.
2. The package test proves neighboring behavior did not regress.
3. `go run ./cmd/progress validate` proves the single backlog and generated
   roadmap surfaces still parse and agree with the schema.
4. `git diff --check` catches whitespace and patch hygiene before the diff is
   handed to a human.
5. `go test ./... -count=1` is the broad local gate before a slice is called
   ready for integration.

The order matters. If the focused test is red, the loop should not hide behind a
big suite. If progress validation is red, the repository's own map is broken. If
`git diff --check` fails, the patch is not ready to hand off even if the feature
works. Each command exists because it catches a different class of failure.

This also explains why Gormes keeps `progress.json` as the only backlog. A side
TODO file might be convenient for one agent, but it creates another interface
that future agents must learn. The progress row is already the seam: it carries
source refs, acceptance, write scope, blockers, and test commands. Deepening
that seam gives every worker leverage.

## Cost-per-feature is not ready to publish yet

The strategy doc says the loop should publish cost-per-feature numbers. I agree,
but this draft should not invent them. The Phase 8.C row is still blocked until
loop $/iteration telemetry has at least one week of measured data and an
operator has reviewed the publication voice, date, and platform. That blocker is
part of the engineering story, not an embarrassment.

What we can say today is narrower and safer: the loop is structured so cost can
be measured per row, not guessed from vibes. Each run has an objective, a run id,
an iteration number, changed files, validation commands, and a decision. Once
token spend and provider billing are joined to that event stream, the
cost-per-feature table can be calculated from receipts instead of anecdotes.
Until then, the honest table is a placeholder:

| Metric | Publication status |
|---|---|
| Loop iterations per feature | Measurable from development-goal logs |
| Validation commands per feature | Measurable from delivery reports |
| Token/provider cost per feature | Blocked until one week of telemetry exists |
| Public submission result | Blocked until operator review and platform choice |

That is why this post is a local draft package, not the final public claim. The
operator review step is load-bearing. It checks tone, selects the publication
moment, and decides whether the piece belongs on Hacker News, Lobsters, Reddit,
or only the TrebuchetDynamics feed.

## What the loop is good at

The loop is strong when the behavior has a crisp public surface and a hermetic
oracle. Installer safety, command naming, generated docs, gateway redaction,
profile config, and progress validation are good targets. They have small seams
and tests that can fail for the right reason.

The loop is weaker when the oracle is social, economic, or live-infra dependent.
A public launch post cannot be validated only with `go test`. A cost table needs
billing data. A mobile voice smoke needs a responsive device. In those cases the
right move is not to fake the proof. The right move is to split the package: do
the local draft, add the dry-run or fixture, record the blocker, and leave the
row uncompleted until the live adapter exists.

This discipline is slower than declaring victory. It is also the reason the
repository remains useful after hundreds of agent-authored changes. The loop can
move fast because it is willing to say "not yet" in the same place it says
"green."

## Why this matters for Gormes

Gormes began as Hermes in Go: a single-binary runtime that can carry the Hermes
operator experience into places where Python, venvs, Docker, or Node are the
wrong dependency shape. That is still the engineering direction. But the more
interesting claim is methodological. If the loop can keep turning source-backed
rows into tested Go slices, then Gormes is evidence that large ports can be
worked as a queue of small, replayable contracts.

That does not remove humans. It changes where humans spend attention. Instead of
micromanaging every edit, the operator reviews row quality, public claims,
blockers, cost, and the final diff. The agent handles the repetitive motion:
read the row, write the test, implement the smallest slice, regenerate the docs,
run the gate, and leave a receipt.

The final writeup will need measured cost-per-feature numbers and publication
review before it becomes an external claim. This draft gives the evidence shape:
architecture, three real commit walkthroughs, validation gates, and the explicit
places where the loop still needs better telemetry. That is enough for the next
safe iteration to be smaller: wire the cost metric, run it for a week, and then
turn this local draft into a public receipt.
