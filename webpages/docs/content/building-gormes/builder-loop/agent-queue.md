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
## 1. Shared Bubble Tea wizard step chassis under internal/tui/wizard

- Phase: 1 / 1.E
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: internal/tui/wizard exposes a small Wizard interface (Run(ctx, steps...) (Result, error)) that drives a sequence of Step values — Text, MultiLine, Password, Pick (single-select), Confirm — under a Bubble Tea program. The chassis owns: (a) TTY detection (refuse to start when stdin is not a terminal, return a typed ErrRequiresTTY so callers emit *_requires_tty evidence), (b) bypass-when-fully-specified (callers compose 'if all inputs already supplied via flags, do not run the wizard'), (c) Ctrl-C / escape returning ErrAbort, (d) golden-snapshot testability via charmbracelet/x/exp/teatest. The chassis must not import any cmd/gormes package; admin-TUI screens (1.E.3+) compose it from their screen models, and stand-alone command callers can compose it independently if needed.
- Trust class: operator
- Ready when: go.mod already has bubbletea/bubbles/lipgloss/teatest; no new dependency is required for this slice., The chassis interface fits inside internal/tui/wizard/ as a sibling package and does not edit existing internal/tui/model.go or hermes_chrome.go behavior., Callers can compose the chassis without importing cmd/gormes — the API works for any package that has a *cobra.Command (or just stdio handles).
- Not ready when: The slice edits the main Bubble Tea TUI (internal/tui/model.go, hermes_chrome.go, etc.) or the legacy setup/onboard readline prompts., The slice persists wizard state to disk or shares storage with sessions/memory., The slice adds new third-party dependencies; the chassis must run on the pre-existing Bubble Tea version pin.
- Degraded mode: Without the chassis, every admin-TUI screen would reinvent step UI, TTY detection, and golden testing; the admin-shell rows (1.E.2-1.E.5) cannot land cleanly.
- Fixture: `internal/tui/wizard/wizard_test.go teatest scripts`
- Write scope: `internal/tui/wizard/wizard.go`, `internal/tui/wizard/steps.go`, `internal/tui/wizard/wizard_test.go`, `internal/tui/wizard/testdata/ (teatest golden frames)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui/wizard -run '^TestWizardChassis_' -count=1`, `go test ./internal/tui/wizard -count=1`, `go run ./cmd/progress validate`
- Done signal: internal/tui/wizard package ships the Wizard + Step interfaces with five named tests green; no edits to internal/tui/model.go or cmd/gormes in this row.
- Acceptance: TestWizardChassis_TextStepCapturesInput drives a single-step text wizard through teatest and asserts the captured string and clean exit., TestWizardChassis_PickerStepReturnsSelection drives a picker step with three options through teatest and asserts the selected ID is returned., TestWizardChassis_ConfirmStepHonorsKeybindings drives a confirm step and asserts both yes/no paths return the expected bool., TestWizardChassis_NonInteractiveReturnsErrRequiresTTY asserts Run on a non-TTY stdin returns errors.Is(err, wizard.ErrRequiresTTY) without prompting., TestWizardChassis_AbortReturnsErrAbort asserts Ctrl-C / escape during a step returns errors.Is(err, wizard.ErrAbort).
- Source refs: internal/tui/model.go (existing Bubble Tea root model — reuse style conventions), internal/tui/model_picker.go (existing single-select picker — extract reusable Step), github.com/charmbracelet/bubbletea v1.3.10 (already in go.mod), github.com/charmbracelet/bubbles v1.0.0 (already in go.mod — textinput, list), github.com/charmbracelet/x/exp/teatest (already in go.mod — golden-script harness), cmd/gormes/navivox.go runNavivoxSetupHostApply (existing *_requires_tty evidence pattern)
- Unblocks: Unified admin TUI shell with tab navigation, Admin TUI: Setup health screen with missing-config callouts, Admin TUI: Chat tab with keybinding to jump in from any screen, Admin TUI: Agents screen wired to the 2.H dynamic registry
- Why now: Unblocks Unified admin TUI shell with tab navigation, Admin TUI: Setup health screen with missing-config callouts, Admin TUI: Chat tab with keybinding to jump in from any screen, Admin TUI: Agents screen wired to the 2.H dynamic registry.

## 2. TD engineering blog scaffolded and live

- Phase: 8 / 8.A
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post.
- Trust class: operator
- Ready when: Hosting choice and blog framework are decided (operator decision; not loop-driven)., A subdomain or path on an existing TD-controlled domain is available.
- Not ready when: The blog is private, password-protected, or behind authentication., There is no Atom/RSS feed at a stable URL., The first post is empty or placeholder text rather than the writeup #1 draft or a real introduction.
- Degraded mode: Without a publication outlet, every loop commit is invisible in the reputation market; the strategy described in success-plan.md cannot start.
- Fixture: `webpages/blog/ (or chosen blog repo path)`
- Write scope: `webpages/blog/ (or external blog repo path)`, `DNS / Cloudflare / hosting config (operator-only)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Documentation/research/planning row — automated tests not applicable
- Done signal: Public blog URL + feed URL recorded in success-plan.md and README.md.
- Acceptance: Blog is reachable at a public URL with at least one real (non-placeholder) post., An Atom or RSS feed exists at a stable, discoverable URL., Publishing a new post is a markdown-commit-and-merge operation; no console click-through required., An /about page exists that names TrebuchetDynamics and points at gormes-agent + agentic-porting-kit.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/landing/
- Unblocks: Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline
- Why now: Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline.

## 3. Agentic-porting-kit repo scaffold

- Phase: 8 / 8.E
- Owner: `skills`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: The gormes-* skill set (gormes-planner, gormes-builder, gormes-tdd-slice, gormes-parity-auditor, gormes-references, gormes-skill-manager) is extracted into a separate public TrebuchetDynamics repo (`agentic-porting-kit` or equivalent), with a README that frames the kit as a generic Python→Go porting toolkit, a worked example using a small non-Hermes target, and a clear license. The kit must work standalone — its rows must be loadable by Codex or Claude Code in any repo, not just Gormes.
- Trust class: operator
- Ready when: All listed skills have a README of their own that does not assume the Gormes repo layout., Skills' references that hard-code Gormes paths have been parameterized or generalized.
- Not ready when: Skills still hard-code paths under docs/content/building-gormes/., The extracted kit cannot be tested without cloning Gormes.
- Degraded mode: Without extraction, the methodology is invisible to other teams; "the loop is the product" cannot be substantiated externally.
- Fixture: `(separate repo: TrebuchetDynamics/agentic-porting-kit)`
- Write scope: `(separate repo)`, `webpages/docs/development-skills/ (de-Gormes-fy paths)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Documentation/research/planning row — automated tests not applicable
- Done signal: Repo URL recorded in success-plan.md and README.md; star count tracked monthly.
- Acceptance: Public repo TrebuchetDynamics/agentic-porting-kit exists with the listed skills., Repo README explains the kit independent of Gormes/Hermes., A worked example demonstrates the kit on a non-Hermes target (any small Python project being ported to Go)., Skills can be loaded into a fresh Codex or Claude Code session and successfully plan-and-execute one row in the example target.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
