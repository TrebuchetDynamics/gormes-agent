---
name: gormes-browser-harness
description: Use when planning, auditing, or implementing Gormes browser automation parity that involves Browser Use, go-browser-harness, legacy browser-harness, Hermes /browser connect, CDP sessions, Browserbase/Firecrawl/Camofox provider routing, browser tool descriptors, or browser interaction skills.
---

# Gormes Browser Harness

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Make Gormes browser automation track Hermes behavior while using Gormes'
in-process Go browser runtime. The old `../go-browser-harness` sibling repo is
not a runtime dependency and may be deleted. The Python
`browser-use/browser-harness` checkout is a behavior reference only, not the
default Gormes plugin dependency. This skill is for browser parity slices only;
keep provider auth work in `gormes-provider-parity` and generic tool registry
work in `gormes-builder`.

## Source Order

1. Inspect current Gormes rows in
   `docs/content/building-gormes/architecture_plan/progress.json`, Phase 5.C
   `Browser Automation`.
2. Inspect Hermes upstream first:
   - `../hermes-agent/tools/browser_tool.py`
   - `../hermes-agent/tools/browser_cdp_tool.py`
   - `../hermes-agent/tools/browser_supervisor.py`
   - `../hermes-agent/tools/browser_dialog_tool.py`
   - `../hermes-agent/tools/browser_providers/{base,browserbase,browser_use,firecrawl}.py`
   - `../hermes-agent/hermes_cli/browser_connect.py`
   - `../hermes-agent/cli.py:_handle_browser_command`
   - matching tests under `../hermes-agent/tests/tools/` and
     `../hermes-agent/tests/cli/`.
3. Inspect Gormes' internal Go runtime contract:
   - `internal/tools/browser_harness_backend.go`
   - `internal/tools/browser_harness_chromedp_transport.go`
   - `internal/tools/browser_harness_tools.go`
   - `internal/tools/browser_use_harness_bridge.go`
4. Inspect legacy browser-harness only for Browser Use/CDP operator workflow:
   - `../browser-harness/SKILL.md`
   - `../browser-harness/install.md`
   - `../browser-harness/agent-workspace/agent_helpers.py`
   - relevant `../browser-harness/interaction-skills/*.md`.
5. Use Context7 for Browser Use API or cloud behavior when current public docs
   matter. Do not paste docs into the repo; cite only the stable behavior in
   progress rows or tests.

## Workflow

1. Classify the requested browser work:
   - **Skill/agent workflow parity:** update this skill or related skill
     routing, not runtime code.
   - **Planning parity:** use `gormes-parity-auditor` then `gormes-planner`;
     edit `progress.json` only.
   - **Runtime implementation:** use `gormes-builder` plus
     `gormes-tdd-slice`; select exactly one Phase 5.C row.
2. Preserve Hermes public tool names:
   `browser_navigate`, `browser_snapshot`, `browser_click`, `browser_type`,
   `browser_scroll`, `browser_back`, `browser_press`, `browser_console`,
   `browser_get_images`, `browser_vision`, `browser_cdp`, and
   `browser_dialog`.
3. Keep Gormes' in-process `gormes.browser.action.v1` backend as the default
   runtime bridge. The legacy `browser-harness -c` path is explicit
   compatibility only and must not be required for normal Gormes browser tool
   operation.
4. The Go contract must still expose Hermes-style schemas, result envelopes,
   evidence, private-URL safety, timeout, and cleanup behavior.
5. Do not start Chrome, open tabs, call Browser Use, or hit Browserbase,
   Firecrawl, or Camofox in unit tests. Use fake command runners, fake HTTP
   transports, fake CDP payloads, and fixture transcripts first.
6. If a live smoke test is needed, mark it optional and keep it outside the
   required CI gate.

## Chromedp Backend (Default as of 2026-04-29)

`internal/tools/browser_harness_backend.go` and
`internal/tools/browser_harness_chromedp_transport.go` provide a live
`BrowserHarnessChromedpBackend` that dispatches browser actions through the
Chrome DevTools Protocol (CDP) using the `github.com/chromedp/chromedp` and
`github.com/chromedp/cdproto` libraries.

### Operator setup

1. Start Chrome with remote debugging enabled:
   ```sh
   google-chrome --remote-debugging-port=9222 --user-data-dir=/tmp/gormes-chrome
   ```
2. Export the endpoint:
   ```sh
   export CHROME_REMOTE_DEBUGGING_URL=http://localhost:9222
   ```
   A full websocket URL (`ws://localhost:9222/devtools/browser/...`) is also accepted.
3. Run a Gormes browser tool or CDP-backed `web_extract` path that emits
   `gormes.browser.action.v1` internally.

Gormes never auto-launches Chrome. If neither `CHROME_REMOTE_DEBUGGING_URL` nor
`BROWSER_CDP_URL` is set, the internal backend returns typed unavailable
evidence.

### New-tab semantics

`browser_navigate` MUST call `Target.createTarget` (new tab) then
`Target.activateTarget` before navigating. It never reuses the operator's active tab.
In the Go action JSON contract this is represented by `new_tab: true` on navigate.

This is enforced at the internal backend layer in
`internal/tools/browser_harness_backend.go:BrowserHarnessChromedpBackend.runNavigate`
and at the tool layer in `internal/tools/browser_harness_tools.go:buildActionRequest`
(case `browser_navigate`).

### Python browser-harness (legacy path only)

`browser-harness -c` is retained for explicit legacy compatibility only. It must be
selected by setting `Command: "browser-harness"` and `Protocol: "python-browser-harness"`
in `BrowserHarnessToolsConfig`. It is never the default.

## Browser-Harness Mapping Rules

- `gormes.browser.action.v1` is the default fakeable internal backend
  contract; never shell-concatenate user input.
- `browser-harness -c` is legacy compatibility only and must be selected
  explicitly in config/tests.
- First navigation in harness instructions is `new_tab(url)`, not `goto_url`,
  because `goto_url` can clobber a user's active tab. In the Go action JSON
  contract this is represented by `new_tab: true` on navigate.
- `BU_NAME` is the session namespace. Derive it from the Gormes task/session
  key and sanitize it before passing to the environment.
- Remote Browser Use mode requires `BROWSER_USE_API_KEY`; cloud sessions return
  CDP connection data and must be stopped/cleaned up deterministically.
- Profile reuse and cookie sync are operator-controlled state. Store only
  profile IDs/names or redacted evidence; never copy cookies, tokens, or
  browser profile contents into transcripts.
- Screenshot and large text output must flow through Gormes artifact/result
  budgeting instead of being embedded unbounded in model context.

## Validation

For planning or skill-only edits:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./docs -count=1
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -path '*/gormes-browser-harness/SKILL.md' -print
```

For runtime browser harness rows, add the row-specific test first. Typical
minimum commands:

```sh
go test ./internal/tools -run 'TestBrowserHarness|TestBrowserProvider|TestBrowserCDP|TestBrowserContract' -count=1
go test ./internal/tools -count=1
go run ./cmd/progress validate
```

## Guardrails

- Do not mark browser parity complete because `../browser-harness` exists or
  because a deleted sibling runtime used to exist. Gormes needs Go tests and
  Hermes-visible tool behavior.
- Do not import browser-harness Python packages into Gormes runtime code or
  require Python installation for the default browser plugin path. External
  execution must be behind an interface.
- Do not add side TODO files. Missing behavior becomes a builder-ready
  `progress.json` row.
- Do not weaken private/LAN URL handling. Reuse `Browser hybrid private-URL
  local sidecar routing` and `Browser SSRF quoted-false guard`.
