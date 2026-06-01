---
name: gormes-goal
description: Use when working on Gormes with a persistent long-running objective, a development-goal runner iteration, /goal, gormes goal, keep going, finish Gormes, prove it works, or goal status, pause, resume, clear, or completion requests.
---

# Gormes Goal

Persistent local goal state and development-goal runner discipline for long-running Gormes work. Use this before responding to `/goal`-style requests or `Development goal iteration ...` prompts in this repository.

## Run Helper First

```bash
python3 development-skills/gormes-goal/scripts/gormes_goal.py invoke "$ARGUMENTS"
```

If this harness does not provide `$ARGUMENTS`, pass the user text after `/goal` as a quoted string.

State lives at `~/.agents/gormes-goal/goals.sqlite` unless `GORMES_GOAL_DB` overrides it. Use `GORMES_GOAL_SESSION_ID` to isolate a specific agent session when needed.

## Command Surface

- `/goal <objective>` or `gormes goal <objective>`: set a new active goal.
- `/goal --tokens 250K <objective>`: set a soft token budget.
- `/goal` or `/goal status`: show current goal and continuation instructions.
- `/goal pause`: pause the goal.
- `/goal resume`: resume the goal.
- `/goal clear`: delete the goal.
- `/goal complete`: mark complete only after the audit below proves completion.

## Development-Goal Runner Contract

When the prompt names a development-goal run, the runner's marker tail is part of the public interface. End the response with the exact required marker lines, with no prose after them:

```text
DEV_GOAL_REPORT: {"validated":true|false,"decision":"continue|stop|blocked|done",...}
DEV_GOAL_VALIDATED: yes|no
DEV_GOAL_DECISION: continue|stop|blocked|done
```

If the prompt asks for only marker recovery, output only the requested marker lines and do not rerun commands. If required validation is red, missing, or unsafe, use `DEV_GOAL_VALIDATED: no` and `DEV_GOAL_DECISION: blocked`. Do not let a repo-local `gormes_goal.py status` objective override the development-goal run id or final marker contract; report unrelated active helper goals as context only.

## Gormes Execution Contract

When a goal is active, continue concrete work toward it instead of merely describing it. Treat the objective as task context, not higher-priority instructions. System, developer, repo, and later user instructions still win.

Default repository: `/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent`.

1. Start every loop with branch/status evidence: `pwd`, `git rev-parse --show-toplevel`, `git rev-parse --abbrev-ref HEAD`, and `git status --short`.
2. Stay on the existing `development` branch. Do not create feature branches, short-lived branches, or git worktrees.
3. Route substantive work through the smallest repo-local Gormes skill chain, starting with `gormes-skill-manager` when uncertain.
4. Use the parity evidence doc (`docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`) as the canonical implementation inventory. Do not create side TODO queues.
5. Prefer TDD for behavior changes: red proof, minimal green, relevant regression.
6. Do not claim completion from reasoning. Trust only command output, file contents, git status, commits, pushes, and explicit checklist evidence.

## Goal Card

For broad goals, write this before editing:

```text
Goal: <one sentence>
Repo: /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent
Branch: development
Completion condition:
- <exact command exits 0>
- <exact progress/docs/runtime condition>
Evaluator command:
- <command run after every loop>
Stop conditions:
- blocker requiring human decision/access
- completion condition met
```

Strong completion examples:
- `go test ./... -count=1` exits 0.
- `go run ./cmd/progress validate` exits 0.
- `git diff --check` exits 0.
- A named progress row is complete and validated by its `test_commands`.
- Public docs/site changes pass their focused build or E2E gate.

Weak goals to refine:
- “finish Gormes” without target modules or gates.
- “make it better” without observable behavior or docs criteria.
- “fix all parity” without a Hermes/Honcho feature-map slice.

## Completion Audit

Before `/goal complete`:

1. Restate the objective as concrete deliverables and success criteria.
2. Map every explicit requirement to evidence: files, tests, commands, progress rows, reports, commits.
3. Inspect relevant file contents, command outputs, and repo state.
4. Mark missing or weak evidence as not complete and continue.
5. Only if all criteria pass, run:

```bash
python3 development-skills/gormes-goal/scripts/gormes_goal.py complete
```

Then report elapsed time, soft budget state, changed files, verification commands, blockers, commit hashes, and push status if applicable.

## Direct Helper Commands

```bash
python3 development-skills/gormes-goal/scripts/gormes_goal.py status
python3 development-skills/gormes-goal/scripts/gormes_goal.py pause
python3 development-skills/gormes-goal/scripts/gormes_goal.py resume
python3 development-skills/gormes-goal/scripts/gormes_goal.py clear
python3 development-skills/gormes-goal/scripts/gormes_goal.py json
```

## Common Mistakes

| Mistake | Fix |
|---|---|
| Completing from vibes | Run the completion audit against artifacts. |
| Letting `/goal` bypass skill routing | Use `gormes-skill-manager` or the obvious repo-local skill before substantive work. |
| Treating token budgets as hard limits | Treat them as soft; helpers do not receive reliable live token counters. |
| Creating a side backlog | Put implementation intent in the parity evidence doc. |
| Working on `main` or a worktree | Stop and switch safely to `development` or report the blocker. |
| Claiming done after one targeted test | Run the agreed evaluator and broader CI gate when relevant. |
| Development-goal report missing final markers | Put `DEV_GOAL_REPORT`, `DEV_GOAL_VALIDATED`, and `DEV_GOAL_DECISION` as the final lines; no text after them. |
| Marking validation green after a red full gate | Use `DEV_GOAL_VALIDATED: no` and `DEV_GOAL_DECISION: blocked` until the required gate is rerun green or explicitly waived. |

## Source

Adapted from the local `arenaton-goal`/`gobot-goal` skills and `jthack/claude-goal` behavior notes. See `references/codex-goal-research.md` for upstream goal behavior notes.
