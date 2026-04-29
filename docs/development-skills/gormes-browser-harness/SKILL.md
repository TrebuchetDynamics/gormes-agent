---
name: gormes-browser-harness
description: Use when planning, auditing, or implementing Gormes browser automation parity that involves Browser Use, browser-harness, Hermes /browser connect, CDP sessions, Browserbase/Firecrawl/Camofox provider routing, browser tool descriptors, or browser interaction skills.
---

# Gormes Browser Harness

## Mission

Make Gormes browser automation track Hermes behavior while using
`../browser-harness` as the Browser Use/CDP workflow reference. This skill is
for browser parity slices only; keep provider auth work in
`gormes-provider-parity` and generic tool registry work in `gormes-builder`.

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
3. Inspect browser-harness only for Browser Use/CDP operator workflow:
   - `../browser-harness/SKILL.md`
   - `../browser-harness/install.md`
   - `../browser-harness/agent-workspace/agent_helpers.py`
   - relevant `../browser-harness/interaction-skills/*.md`.
4. Use Context7 for Browser Use API or cloud behavior when current public docs
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
3. Keep browser-harness as an external runtime bridge until a Go-native CDP
   backend replaces it. The Go contract must still expose Hermes-style schemas,
   result envelopes, evidence, private-URL safety, timeout, and cleanup
   behavior.
4. Do not start Chrome, open tabs, call Browser Use, or hit Browserbase,
   Firecrawl, or Camofox in unit tests. Use fake command runners, fake HTTP
   transports, fake CDP payloads, and fixture transcripts first.
5. If a live smoke test is needed, mark it optional and keep it outside the
   required CI gate.

## Browser-Harness Mapping Rules

- `browser-harness -c` maps to a fakeable command runner; never shell-concatenate
  user input.
- First navigation in harness instructions is `new_tab(url)`, not `goto_url`,
  because `goto_url` can clobber a user's active tab.
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

For runtime browser-harness rows, add the row-specific test first. Typical
minimum commands:

```sh
go test ./internal/tools -run 'TestBrowserHarness|TestBrowserProvider|TestBrowserCDP|TestBrowserContract' -count=1
go test ./internal/tools -count=1
go run ./cmd/progress validate
```

## Guardrails

- Do not mark browser parity complete because `../browser-harness` exists.
  Gormes needs Go tests and Hermes-visible tool behavior.
- Do not import browser-harness Python packages into Gormes runtime code.
  External execution must be behind an interface.
- Do not add side TODO files. Missing behavior becomes a builder-ready
  `progress.json` row.
- Do not weaken private/LAN URL handling. Reuse `Browser hybrid private-URL
  local sidecar routing` and `Browser SSRF quoted-false guard`.
