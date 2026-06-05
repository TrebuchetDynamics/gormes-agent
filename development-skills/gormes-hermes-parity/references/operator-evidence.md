# Operator Evidence

Use this when a live report names a visible bug, transcript, installer path,
runtime home, gateway process, channel artifact, or stale product label.

## Preserve The Artifact

Capture before reducing to a row:

- surface: Telegram, TUI, CLI, installer, gateway log, etc.;
- exact user input and visible output;
- duplicate messages, edits, deletes, typing/status/hourglass artifacts;
- whether final content was correct but surrounding UX was wrong;
- binary/home/source surface that was actually running.

Do not paraphrase duplicate-message, tool-loop, or hourglass bugs into
"tool calling failed". The message sequence is the future test contract.

## Repeated Failure Patterns

| Pattern | First check |
|---|---|
| Gormes asks for `hermes gateway start`, mentions Hermes `api_server`, or depends on `~/.hermes` | Route through `gormes-dev-runtime`; prove binary path, `GORMES_HOME`, and installed source before changing UX code. |
| `go run`, `./bin/gormes`, and installed `gormes` differ | Test all three with explicit paths/homes. Do not infer installed behavior from dirty source. |
| `sessions.db` locked while switching binaries | Use gateway status/stop or isolated `GORMES_HOME`; do not delete the database. |
| Extra hourglass/status, duplicate assistant reply, visible raw iteration-limit text, leaked tool-call text, stale Hermes label | Preserve transcript and build a channel/TUI fixture. Correct final content is not enough. |
| Persona/reset mismatch | Inspect Hermes default soul/prompt sources and Gormes `internal/agenttemplate` before planning or coding. |
| Installer confusion | Final-user installs clone/build a branch; development validation uses `go run ./cmd/gormes` or `./bin/gormes`. |

## Channel Tool Progress

For pasted channel tool-progress blocks, pin the contract to Hermes gateway
progress, not only the TUI. Start from:

- `$HERMES_SRC/gateway/run.py:progress_callback`
- `agent.display.build_tool_preview`
- `agent.display.get_tool_emoji`

Compare with:

- `internal/kernel/toolexec.go:toolCallPreview`
- `internal/gateway/render.go:FormatToolProgressPlain`

Common drift: `todo merge=true` should render updating work, Python-only
`execute_code` schemas should not leak directly, and model-visible
`read_file`/`search_files` behavior must not be solved by renderer labels only.
