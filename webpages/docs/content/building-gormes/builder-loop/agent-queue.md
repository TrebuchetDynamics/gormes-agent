---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Coding-agent delegation: Phase 1 scaffold (internal/codingagents)

- Phase: 2 / 2.A
- Owner: `tools`
- Size: `medium`
- Status: `in_progress`
- Priority: `P1`
- Contract: Shared internal/codingagents package providing the CodingAgent interface, CodingAgentRequest/Result, mode constants, binary availability detection, workspace guard with default deny list, git snapshot/diff helper, and prompt wrapper. No tools are registered in this slice; adapters and registry exposure land in later phases.
- Trust class: operator, system
- Ready when: Shared CodingAgent interface and CodingAgentRequest/Result cover workspace, prompt, mode, edit permissions, timeout, files-changed, stdout/stderr, and git diff., Availability checks detect codex, claude/claude-code, and opencode binaries and report unavailable cleanly., Workspace guard refuses empty, ambiguous, denied, and outside-allowed inputs and accepts paths under an allowed root., Git snapshot/diff helper captures HEAD/branch/dirty/files for a real repo and returns ErrNotAGitRepo on a non-git dir., Prompt wrapper restates workspace/mode/task and injects gormes-repo rules when the workspace is a gormes-agent checkout.
- Not ready when: Adapters or tool descriptors register coding_agent / codex_run / claude_code_run / opencode_run before the umbrella's later phases., Results omit files_changed or git_diff across adapters., Workspace identifiers bypass the guard via raw-path voice input.
- Degraded mode: Without the scaffold, later phases cannot compile codex/claude-code/opencode adapters against a shared contract; doctor cannot probe coding-agent binaries.
- Fixture: `internal/codingagents`
- Write scope: `internal/codingagents/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/codingagents/... -count=1`, `go vet ./internal/codingagents/...`, `go run ./cmd/progress validate`
- Done signal: go test ./internal/codingagents/... -count=1 passes locally with the scaffold, availability probe, workspace guard, git snapshot, and prompt wrap covered by unit tests.
- Acceptance: internal/codingagents compiles and tests pass on stdlib only., WorkspaceGuard returns typed sentinels (ErrWorkspaceEmpty/Ambiguous/OutsideAllowed/Denied) and refuses $HOME, /, and ~/.ssh by default., DetectAll returns availability entries for codex, claude, claude-code, and opencode., TakeSnapshot + DiffBetween capture HEAD, dirty status, and a unified diff with file list on a temp repo; non-git dirs raise ErrNotAGitRepo.
- Source refs: User design: 2026-05-13 coding-agent delegation plan, internal/codingagents/codingagents.go, internal/codingagents/workspace.go, internal/codingagents/git_snapshot.go
- Why now: Already active; contract metadata keeps execution bounded.

## 2. Navivox VPN host enumeration helper

- Phase: 9 / 9.E
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Contract: An owned `internal/network/vpnhost` package enumerates active VPN interfaces — Tailscale (via `tailscale ip -4/-6`), WireGuard (via `ip -j link show type wireguard`), and tun-class OpenVPN/IPSec devices — returning `{iface, kind, ipv4, ipv6}` tuples ordered Tailscale → WireGuard → other VPN. Both `cmd/gormes/navivox` (SSH pair) and the navivox HTTP gateway channel consume it instead of detecting Tailscale ad-hoc.
- Trust class: -
- Ready when: Existing tailscale-only host detection in cmd/gormes/navivox.go (navivoxPairHostResolver, withNavivoxPairHostResolverTestSeams) is identified as the migration source., internal/channels/navivox already builds and `go test ./... -count=1` is green on development.
- Not ready when: The slice also rebinds the running gateway or edits install.sh., The slice adds VPN providers beyond Tailscale and WireGuard without test fixtures.
- Degraded mode: -
- Fixture: `internal/network/vpnhost/vpnhost_test.go`
- Write scope: `internal/network/vpnhost/`, `cmd/gormes/navivox.go`, `cmd/gormes/navivox_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/network/vpnhost -count=1`, `go test ./cmd/gormes -run NavivoxPairHost -count=1`
- Done signal: `go test ./internal/network/vpnhost ./cmd/gormes -count=1` passes; cmd/gormes/navivox.go no longer shells out to `tailscale` directly.
- Acceptance: internal/network/vpnhost.List(ctx) returns at least one entry on a host with Tailscale or WireGuard up, with kind ∈ {tailscale, wireguard, tun-other}., cmd/gormes/navivox.go consumes vpnhost.List for pair host resolution (Tailscale entry first), and existing TestNavivoxPairHost* tests still pass via seamed fakes., internal/network/vpnhost has zero imports from internal/channels or cmd/ so both can import it.
- Source refs: cmd/gormes/navivox.go:navivoxPairHostResolver, cmd/gormes/navivox.go:withNavivoxPairHostResolverTestSeams, cmd/gormes/navivox_test.go:TestNavivoxPairHostAutoDetectFallsBackToLAN
- Unblocks: Navivox HTTP gateway mandatory-VPN bind, Navivox HTTP gateway connect-info command
- Why now: Unblocks Navivox HTTP gateway mandatory-VPN bind, Navivox HTTP gateway connect-info command.

## 3. Agentic-porting-kit public repo scaffold

- Phase: 8 / 8.E
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout.
- Trust class: operator
- Ready when: Agentic-porting-kit extraction spec is complete., GitHub authentication can create or push to TrebuchetDynamics/agentic-porting-kit, or the operator has created the empty repo., The public repo name is confirmed as agentic-porting-kit or an equivalent name before the first push.
- Not ready when: No authenticated path exists to create or update the public TrebuchetDynamics repo., The builder plans to edit Gormes' repo-local skills in place instead of copied kit skills., The standalone example still requires cloning Gormes or running cmd/progress.
- Degraded mode: Without the public scaffold, the methodology remains inspectable only inside Gormes and cannot be cited or reused by other teams.
- Fixture: `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json`
- Write scope: `(separate repo) README.md`, `(separate repo) LICENSE`, `(separate repo) schemas/progress.schema.json`, `(separate repo) scripts/validate-example.sh`, `(separate repo) skills/`, `(separate repo) examples/python-greeter-to-go/`, `README.md`, `docs/content/building-gormes/strategy/success-plan.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `cd ${AGENTIC_PORTING_KIT_REPO:-../agentic-porting-kit} && ./scripts/validate-example.sh`, `go run ./cmd/progress validate`, `go test ./webpages/docs -count=1`
- Done signal: Public repo URL, standalone validation output, and Gormes backlink updates are recorded in the completed row note.
- Acceptance: Public repo exists with README.md, LICENSE, schemas/progress.schema.json, scripts/validate-example.sh, skills/, and examples/python-greeter-to-go/., README.md explains the kit independent of Gormes/Hermes and includes Codex plus Claude Code loading instructions., Each copied skill uses the porting-* name from the extraction spec and replaces hard-coded Gormes paths with target-repo variables., scripts/validate-example.sh validates the example progress file and runs the example tests without cloning Gormes., Gormes README.md and success-plan.md record the public repo URL after the repo is reachable.
- Source refs: docs/content/building-gormes/strategy/agentic-porting-kit.md, docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
