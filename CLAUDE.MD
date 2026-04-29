# CLAUDE.MD - gormes-agent

Read `AGENTS.md` before touching code, docs, roadmap data, or generated
progress surfaces in this repository. `AGENTS.md` is the canonical agent
contract.

## Branch and CI safety rule

Main must always stay green. This is a hard rule for Claude, Codex, opencode,
workspace-mineru, workspace-mimeru, and any other agent working here.

- Do not develop directly on `main`.
- Do development on `development`, or on a short-lived branch created from
  `development`.
- Merge to `main` only through a GitHub pull request.
- Before opening or updating a PR to `main`, run:
  `go test ./... -count=1`, `go run ./cmd/progress validate`, and
  `git diff --check`.
- If `main` is red, stop normal feature work and fix `main` first.
- GitHub rules for `main` must require pull requests and the required CI status
  check.

## Skill routing

Before substantive Gormes work, use a repo-local skill from
`docs/development-skills/`. If unsure, start with
`docs/development-skills/gormes-skill-manager/SKILL.md`.
