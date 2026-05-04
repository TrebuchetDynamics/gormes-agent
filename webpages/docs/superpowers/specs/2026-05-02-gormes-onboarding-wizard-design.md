# Gormes Onboarding Wizard Design

## Context

`gormes onboard` currently prints a truthful first-run status page: home path,
config path, runtime skills root, provider state, agents, bindings, learning
loop state, and next-step commands. That is useful for diagnosis, but it makes
a new user stitch together setup manually.

The existing roadmap row is `Interactive Onboarding` in Phase 5 / 5.O CLI
parity. Its contract is to promote `gormes onboard` into the full first-run
flow: model/provider selection, auth setup, gateway channel configuration,
browser/CDP checks, skill discovery, and dashboard launch. The current
`gormes setup` command has a partial sectioned wizard that should be reused
rather than replaced by a second setup path.

## Goal

Make the best first-run UX the default:

- `gormes onboard` starts a guided wizard when required setup is missing and
  stdin is interactive.
- Existing configured installs keep getting a concise status page, with an
  explicit way to re-run the wizard.
- Noninteractive environments never hang on prompts; they get status plus exact
  commands and environment variables to complete setup.
- Provider, model, and auth are presented as one joined step so a user finishes
  with a working `gormes --oneshot "hello"` path.

## Non-Goals

- Do not require live credentials in tests.
- Do not replace `gormes setup`; make it a compatibility route into the same
  setup logic.
- Do not implement persistent mutation for gateway bindings that the current
  config writer cannot safely express.
- Do not hide partial systems. Browser, gateway, skills, and dashboard steps
  must clearly report skipped or unavailable states.

## Command Behavior

### `gormes onboard`

If provider/model/auth setup is incomplete and stdin is a TTY, launch the
guided wizard immediately.

If the install is already configured, print the current status page with short
actions:

- `gormes onboard --wizard`
- `gormes doctor --offline`
- `gormes dashboard`
- `gormes gateway status`

If stdin is not a TTY, print the current status page and noninteractive setup
guidance. It must not prompt.

### `gormes onboard --wizard`

Always run the guided flow, prefilled from existing config and env state. This
is the explicit reconfigure path. Steps can be skipped, but skips are visible
in the summary.

### `gormes setup`

Keep `gormes setup [section]` for Hermes/OpenClaw compatibility. The full
setup entry should delegate to the same wizard primitives used by
`gormes onboard`, while section commands remain focused shortcuts.

## Wizard Flow

The wizard is linear and resumable by re-running it. Each step prints the
current value, the default action, and the resulting side effect.

1. Provider, model, and auth
   - Choose a known provider or custom OpenAI-compatible endpoint.
   - Pick or accept a default model for that provider.
   - Store API key material in the Gormes dotenv file, never in `config.toml`.
   - Write provider, endpoint, and model to `config.toml`.

2. Quick provider test
   - Offer a dry local config check first.
   - Offer a single provider-backed `hello` test only when credentials exist.
   - Failure reports the provider, endpoint, model, and redacted auth state.

3. Workspace and agents
   - Show the default agent and workspace.
   - Allow accepting defaults.
   - For multi-agent setup, print or apply only config shapes already supported
     by the existing config package.

4. Bindings and gateway channels
   - Show current bindings.
   - Offer guided channel route prompts.
   - If safe persistent binding writes are not yet supported, print the exact
     config block and mark the step as manual in the summary.

5. Browser/CDP check
   - Detect whether browser automation dependencies are available.
   - Report unavailable browser support as a skipped/degraded step with the
     next command to fix it.

6. Skills
   - Show runtime skills root and installed local/bundled counts.
   - Offer `gormes skills list`.
   - Do not confuse runtime skills with repo development skills.

7. Dashboard
   - Offer to launch or print `gormes dashboard`.
   - Do not require dashboard startup for onboarding success.

## Architecture

Introduce an internal onboarding runner that accepts injected IO and seams. The
CLI command owns flag parsing and status rendering; the runner owns step order,
prompt defaults, and summary state.

Suggested boundary:

- `cmd/gormes/onboard.go`: flags, TTY decision, status output, command wiring.
- `cmd/gormes/setup.go`: compatibility commands that call shared setup helpers.
- `internal/cli/onboard.go`: wizard runner, step summaries, prompt behavior.
- Existing config helpers: provider/model/env writes and reads.
- Existing skills helpers: runtime skill counts.

The runner should return a typed result with completed, skipped, manual, and
failed steps. Command handlers render that result for CLI users. Tests can
exercise the runner without a real terminal, network, browser, or credentials.

## Error Handling

- Missing TTY: never prompt; return status and noninteractive guidance.
- Missing API key: keep the provider/model config if chosen, mark auth missing,
  and show the exact command/env var to finish.
- Provider test failure: do not erase config; report a redacted diagnostic and
  continue to later local-only steps.
- Unsupported persistent config writes: print manual config and mark the step
  manual.
- Browser/gateway unavailable: report degraded status and continue.

## Testing

Focused tests should cover:

- Bare `gormes onboard` starts the wizard when provider/auth are missing and
  stdin is interactive.
- Bare `gormes onboard` keeps the status page when already configured.
- Non-TTY `gormes onboard` never prompts and prints noninteractive guidance.
- `gormes onboard --wizard` runs even when already configured and pre-fills
  provider/model defaults.
- Provider/model/auth writes keep secrets out of `config.toml` and redact all
  CLI output.
- Skipped browser, gateway, skills, and dashboard steps appear in the summary.
- `gormes setup` and `gormes onboard` share provider/model helper behavior.

Verification for the implementation slice:

```sh
go test ./cmd/gormes -run 'TestOnboard|TestSetup' -count=1
go test ./internal/cli -run TestOnboard -count=1
go run ./cmd/progress validate
git diff --check
```

## Progress Row Update

After implementation, update `Interactive Onboarding` evidence in
`docs/content/building-gormes/architecture_plan/progress.json` with the
implemented files and exact test commands. The row should remain `in_progress`
until all planned wizard steps are implemented with tests; partial slices must
name which steps are live and which remain manual/degraded.
