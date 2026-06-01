# Gormes First-Run CLI/TUI UX Design

Date: 2026-05-14

## Goal

Improve the whole Gormes operator CLI/TUI experience, starting with the
fresh-install path through provider, auth, model, and channel setup.

The first implementation slice should stop plain `gormes` from dropping a new
operator into a broken chat screen. A fresh interactive launch should route the
operator through setup, prove that the selected target can work, and then hand
off to chat or gateway operation.

## Scope

In scope for the first slice:

- Plain `gormes` first-run routing.
- `gormes setup --quick` target-first flow.
- Shared readiness planning for `gormes`, `gormes setup`, `gormes onboard`, and
  `gormes doctor`.
- Target-aware setup for terminal chat and gateway channels such as Telegram,
  WhatsApp, Discord, and Slack.
- Non-interactive guidance that never prompts or hangs.
- A tiny provider-backed live test in Quick setup.

Out of scope for the first slice:

- Full redesign of the main chat Bubble Tea surface.
- Full redesign of the admin TUI command palette.
- Implementing every gateway platform-specific setup screen if a deeper row is
  still row-backed. The first slice may route to existing gateway setup and
  print exact commands where needed.
- Starting long-running gateway services without explicit operator choice.

## First-Run Flow

Plain `gormes` becomes a readiness-aware launcher.

When provider setup is missing and stdin is an interactive TTY, `gormes` opens a
first-run setup router instead of trying to start chat and failing later.

The router shows:

- Quick setup.
- Full setup.
- Migrate Hermes, only if Hermes state exists.
- Migrate OpenClaw, only if OpenClaw state exists.

Quick setup asks the target first:

- Terminal chat.
- Telegram.
- WhatsApp.
- Discord, Slack, or another supported gateway channel.

Then it configures only what the selected target needs:

1. Provider.
2. Credential.
3. Model.
4. Selected channel config when the target is not terminal chat.
5. Tiny live provider-backed test.
6. Target-specific handoff.

Terminal handoff opens the main Bubble Tea chat. Channel handoff shows or runs
the right gateway path after channel configuration is ready.

For non-TTY and CI, `gormes` never prompts. It prints a concise setup report with
the exact command to run, such as `gormes setup --quick`, and exits cleanly.

## Architecture

Add a small first-run planning layer in `internal/cli` rather than hard-coding
the routing only in `cmd/gormes/main.go`.

The planner owns pure data structures such as `SetupReadiness`,
`FirstRunPlan`, `SetupTarget`, and `SetupAction`.

Planner inputs:

- Loaded config.
- Provider, endpoint, model, and credential presence.
- Channel configuration state.
- Detected Hermes and OpenClaw migration sources.
- TTY versus non-TTY mode.
- Requested setup target, when known.

Planner outputs:

- Overall readiness.
- Available router options.
- Recommended default.
- Missing steps.
- Target-specific next command.
- Human summary text and JSON-friendly fields for command surfaces.

Command responsibilities:

- Plain `gormes` calls the planner before provider/TUI startup.
- If ready, it opens normal chat.
- If interactive and not ready, it launches the first-run router.
- If non-interactive and not ready, it prints the plan and exits without
  prompts.
- `gormes setup --quick` uses the same planner and asks for target first.
- `gormes onboard` renders the readable status and next-step view from the same
  plan.
- `gormes doctor` verifies the selected target and reports exact repair steps.
- `gormes setup gateway` remains the deeper channel configuration command.

This keeps the setup logic in one place and prevents `gormes`, `setup`,
`onboard`, and `doctor` from developing separate rules.

## Behavior Details

Quick setup is target-driven:

1. Ask target first.
2. Configure provider/auth/model.
3. Configure selected channel only if target is a gateway channel.
4. Run a tiny provider-backed live test automatically.
5. Hand off to chat or gateway operation.

Full setup is broader but still ordered:

1. Provider/auth/model.
2. Channel setup.
3. Workspace, agents, and bindings.
4. Tools and skills.
5. Dashboard/admin surfaces.

Doctor gains target-aware summary behavior. It should answer the operator's
real question: can the selected target work now? When not ready, it should show
the exact missing step. For example, a Telegram target with no bot token should
say that Telegram setup is missing and point to the relevant gateway setup
command.

Error handling rules:

- Never print raw API keys, bot tokens, app tokens, OAuth tokens, or generated
  channel secrets.
- Non-TTY never prompts.
- Live test failures return exact repair guidance and do not open chat.
- Gateway setup does not silently start long-running services unless the
  operator selected that handoff.
- JSON surfaces remain parseable and keep build provenance.

## Bubble Tea Implications

The first slice is mostly CLI routing, but the handoff affects the Bubble Tea
experience. The root command should preserve the existing Bubble Tea boundary:
the TUI model stays a `Model`/`Update`/`View` consumer of runtime state, while
first-run setup remains outside the chat model until the target is ready.

This means:

- Do not put provider setup prompts inside the main chat model for this slice.
- Do not open the chat TUI until Quick setup succeeds for the terminal target.
- Keep alternate-screen chat behavior unchanged for ready terminal chat.
- Later TUI polish can reuse the same planner to show readiness banners or admin
  repair actions without duplicating setup logic.

## Testing And Acceptance

Implementation starts with failing tests before production code.

Pure planner tests in `internal/cli`:

- Fresh install exposes Quick and Full.
- Hermes and OpenClaw migration options appear only when matching state exists.
- Quick setup asks target first.
- Non-TTY returns exact next commands instead of prompt actions.
- Channel targets include channel setup in missing steps.

Root command tests in `cmd/gormes`:

- Plain `gormes` with fresh config and TTY routes to first-run setup, not
  provider failure.
- Plain `gormes` with fresh config and non-TTY prints setup guidance and exits
  cleanly.
- Ready config still opens normal chat.

Setup command tests:

- `gormes setup --quick` asks target first.
- Terminal target runs provider/auth/model/live-test/chat handoff.
- Channel target delegates to selected gateway setup before handoff.
- Migration options are conditional on detected state.

Doctor and onboard tests:

- `onboard` and `doctor` use the same planner language.
- Target-aware missing channel config produces one exact command.
- JSON surfaces stay parseable and keep build provenance.

E2E:

- Extend the fresh-install e2e suite so a clean `GORMES_HOME` proves the
  first-run path.
- Keep install e2e separate from setup UX except for proving that an installed
  binary can run first-run commands.

Acceptance criteria:

- Fresh users are not dropped into a broken chat screen.
- Quick setup can get to a working first provider-backed chat.
- Channel users see Telegram, WhatsApp, and other gateway targets before they
  have to know the `gateway` command.
- Automation paths do not hang.
- Secrets remain redacted in human and JSON output.
