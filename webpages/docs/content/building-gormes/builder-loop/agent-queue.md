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
## 1. Gormes-owned session tree navigator over lineage and labels

- Phase: 4 / 4.B
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add a native `/tree` session navigator that projects Gormes' existing session lineage, fork, compression, and title metadata into an in-place tree view with search/filter modes and operator labels. Selecting a prior user turn should restore that prompt for editing when safe; selecting non-user entries should switch the visible leaf or report why the stored transcript cannot be replayed. The implementation must use Gormes session stores and lineage tables, not Pi JSONL files.
- Trust class: operator, system
- Ready when: The builder reuses existing session directory, lineage, fork, resume, and TUI panel seams with fake stores in tests., The first slice may omit LLM-generated branch summaries if it records typed not-yet-supported evidence and keeps labels/navigation functional.
- Not ready when: The implementation introduces a second session file format, writes Pi JSONL sessions, or bypasses internal/session and store abstractions., The navigator silently mutates live session state while the kernel is active or loses fork/compression lineage evidence.
- Degraded mode: -
- Fixture: `internal/tui/tree_selector_test.go`
- Write scope: `internal/session/`, `internal/tui/tree_selector.go`, `internal/tui/slash_tree.go`, `internal/tui/slash_dispatch.go`, `cmd/gormes/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/session ./internal/tui -run 'Test.*(Tree\|Label\|Lineage\|Resume\|Branch)' -count=1`, `go test ./cmd/gormes -run 'Test.*Session.*(Tree\|Resume\|Branch)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report seeded tree fixture output, label persistence evidence, replay/degraded cases, and progress validation.
- Acceptance: A seeded lineage fixture renders a tree with forks, compressed/continued sessions, titles, timestamps, and active leaf marker., Filter modes can show default, no-tools, user-only, labeled-only, and all-equivalent projections over Gormes transcript metadata where data exists., Labels/bookmarks can be set and cleared through a session metadata seam and appear in the tree selector., Selecting a prior user turn restores editable text or returns typed replay-unavailable evidence without corrupting the active session.
- Source refs: pi@fc8a155 packages/coding-agent/docs/sessions.md:/tree navigation, pi@fc8a155 packages/coding-agent/docs/session-format.md:labels, branch summaries, tree entries, pi@fc8a155 packages/coding-agent/src/modes/interactive/components/tree-selector.ts, internal/session/lineage.go:LineageKindFork and ResolveLineageTip, internal/session/directory.go:SessionDirectoryEntry, internal/tui/slash_sessions.go:/sessions and /resume picker, internal/tui/slash_branch.go:/branch fork seam
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Per-file mutation queue for native write edit and patch tools

- Phase: 5 / 5.L
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Serialize concurrent file mutations that target the same canonical path across native write, edit, patch, and custom file-task tools while preserving parallel execution for independent files. The queue must resolve symlink aliases for existing files, use cleaned absolute paths for new files, cover the full read-modify-write window, and compose with the existing file staleness registry and atomic writer helpers.
- Trust class: operator, system
- Ready when: The builder can prove behavior with in-memory/fake concurrent tools and temp files; no provider call is needed., The queue is scoped to file mutation paths only and does not serialise unrelated tool calls globally.
- Not ready when: The slice disables concurrent tool execution entirely, relies only on stale-read rejection, or queues only the final write rather than the full mutation window., Symlink aliases for an existing file can still run in parallel and clobber one another.
- Degraded mode: -
- Fixture: `internal/tools/file_mutation_queue_test.go`
- Write scope: `internal/tools/`, `internal/kernel/toolexec.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/kernel -run 'Test.*(MutationQueue\|FileState\|Atomic\|ToolExec)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report same-file concurrency, different-file concurrency, symlink alias evidence, and progress validation.
- Acceptance: Two concurrent mutations to the same existing file run in deterministic serial order and preserve both changes when each operation is valid., Concurrent mutations to different files overlap or are not forced through a global lock., Existing-file symlink aliases share one queue key; missing/new files use the resolved absolute path key., Staleness registry and atomic replace behavior remain covered by existing tests.
- Source refs: pi@fc8a155 packages/coding-agent/docs/extensions.md:withFileMutationQueue guidance, pi@fc8a155 packages/coding-agent/src/core/tools/file-mutation-queue.ts, internal/tools/file_state_registry.go:FileStateRegistry, internal/tools/atomic_replace.go:atomic file replace helper, internal/tools/file_task_tools.go:native file task tools, internal/kernel/toolexec.go:tool execution concurrency boundary
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Gormes JSONL RPC mode over agent runtime events

- Phase: 5 / 5.Q
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Expose a local `gormes` JSONL RPC run mode for language-agnostic embedding. The protocol should accept prompt, steer, follow_up, abort, get_state, get_messages, session stats, model/thinking controls where existing runtime seams support them, and stream agent/tool/queue/compaction events as newline-delimited JSON with strict LF framing. It should reuse Gormes kernel/API-server event models and must not require a web server, Pi subprocess, or live provider in tests.
- Trust class: operator, system
- Ready when: The builder starts with stdio JSONL and fake provider/kernel fixtures; no HTTP listener or live credentials are required., The protocol names Gormes-owned events and documents any unsupported Pi command as typed unavailable evidence rather than pretending parity.
- Not ready when: The slice starts a gateway server, opens network ports, depends on Pi RPC clients, or blocks on subscription/OAuth provider credentials., JSON records are split by anything other than LF, raw stderr contaminates stdout, or command responses cannot be correlated by id.
- Degraded mode: -
- Fixture: `cmd/gormes/rpc_mode_test.go`
- Write scope: `cmd/gormes/`, `internal/kernel/`, `internal/gateway/`, `internal/apiserver/`, `pkg/gormes/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes ./internal/kernel ./internal/gateway ./internal/apiserver -run 'Test.*(RPC\|JSONL\|EventStream\|Queue\|Abort)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report RPC fixture transcript, stdout-cleanliness proof, malformed-command evidence, and progress validation.
- Acceptance: `gormes --mode rpc --no-session` or the chosen subcommand starts a stdin/stdout JSONL loop with a session/header response and no startup chatter on stdout., A fake prompt command emits accepted response, agent/message/tool lifecycle events, and a final agent_end event in deterministic order., Steer/follow_up/abort commands update queue or cancellation state and emit structured responses with request ids., Malformed JSON, unknown commands, and unsupported model/session controls return structured errors without terminating the process unless stdin closes.
- Source refs: pi@fc8a155 packages/coding-agent/docs/rpc.md:Protocol Overview and Commands, pi@fc8a155 packages/coding-agent/docs/json.md:JSON Event Stream Mode, pi@fc8a155 packages/coding-agent/src/modes/rpc/rpc-types.ts, internal/kernel/frame.go:RenderFrame, internal/hermes/events.go:turn/run event types, internal/apiserver/runs.go:run inspection/event surfaces, cmd/gormes/main.go:root command mode selection
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Hermes integrations claim audit + source-backed plugin/skill parity map

- Phase: 8 / 8.C
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Turn the sanitized Reddit/WebAfterAI Hermes integrations post into a source-backed parity map without accepting marketing shorthand as fact: classify each named integration as first-party bundled skill, bundled plugin/backend, gateway/platform/tool, optional skill, indirect web/browser/MCP/scraping workflow, Gormes-owned candidate, or unsupported/excluded claim. The audit must explicitly handle cases where a workflow is achievable through generic web scraping, browser automation, MCP, or Firecrawl-style extraction without being a direct Hermes plugin or tool, and it must not create implementation rows for Reddit, Stripe, InsForge, Graphiti/Zep, or Fireflies unless exact current Hermes source appears.
- Trust class: operator, system
- Ready when: The audit uses only sanitized transcript text plus checked-in Hermes/Gormes source refs; no live private ~/.hermes, credentials, browser sessions, or external API accounts are read., Each of the 12 post items is classified with source refs or explicit unsupported/excluded evidence., Indirect capabilities are allowed as a separate class: generic web scraping, browser automation, Firecrawl extraction, MCP, or skill workflows may satisfy a use case without proving a direct Hermes plugin/tool exists.
- Not ready when: The row is used to implement all integrations in one pass instead of producing a bounded source-backed audit/map., Unsupported claims are copied into README/docs as if they were Hermes-native integrations., The audit treats `hermes plugins install reddit\|stripe\|insforge\|graphiti\|fireflies` as valid without exact current Hermes source or an external plugin repository URL., The audit reads live user config, token stores, memory databases, or private home directories as evidence.
- Degraded mode: Until the claims are source-classified, public roadmap and parity work can overstate Hermes/Gormes integration breadth by treating scraped workflows, optional skills, and unsupported social-post claims as native plugins.
- Fixture: `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md`
- Write scope: `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md`, `webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`, `webpages/docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`, `README.md`
- Test commands: `go run ./cmd/progress validate`, `go test ./webpages/docs -run 'TestUpstreamCoverageLedgerMatchesSourceClasses\|TestProgressCanonical' -count=1`, `git diff --check`
- Done signal: Report the 12-row classification table, exact Hermes source refs, unsupported/excluded claims, indirect scraping/browser/MCP classifications, and any newly-created follow-up progress row names.
- Acceptance: A checked-in audit document or architecture-plan section lists all 12 post items and classifies each as direct first-party skill/plugin/tool/gateway, optional skill, indirect scraping/browser/MCP workflow, Gormes-owned candidate, or unsupported/excluded., The audit explicitly notes that some user-visible workflows are not direct tools: e.g. competitor/site/reddit-style research may be covered by generic web search/extract/crawl, browser automation, or future MCP/web-scraping rows rather than a named Hermes Reddit plugin., Unsupported/excluded claims for Reddit, Stripe, InsForge, Graphiti/Zep, and Fireflies remain excluded or row-backed as discovery-only until source refs are found., Any follow-up implementation intent is routed into separate small progress rows by source class; this audit row does not broaden into a 12-integration implementation batch., Public messaging/docs are updated only with evidence-backed wording and avoid inflated integration counts.
- Source refs: sanitized user-provided Reddit/WebAfterAI transcript 2026-05-24: '12 Hermes Integrations That Actually Matter', hermes-agent/hermes_cli/plugins_cmd.py@43e566f77: `hermes plugins install` clones Git plugins into ~/.hermes/plugins and does not imply a built-in short-name registry for every social-post claim, hermes-agent/hermes_cli/plugins.py@43e566f77: bundled/user/project/pip plugin discovery and opt-in semantics, hermes-agent/skills/productivity/google-workspace/SKILL.md@43e566f77: first-party Gmail/Calendar/Drive/Docs/Sheets skill, hermes-agent/skills/note-taking/obsidian/SKILL.md@43e566f77: filesystem-first Obsidian vault skill, hermes-agent/plugins/web/firecrawl/plugin.yaml@43e566f77 and provider.py: bundled Firecrawl web backend with direct/gateway/self-hosted config, hermes-agent/tools/web_tools.py@43e566f77: generic web_search/web_extract/web_crawl dispatch; supports web-scraping/extraction workflows without naming them as native integrations, hermes-agent/skills/github/DESCRIPTION.md@43e566f77 and skills/github/*/SKILL.md: GitHub auth/repo/issues/PR/code-review skills, hermes-agent/skills/media/youtube-content/SKILL.md@43e566f77: YouTube transcript helper skill, hermes-agent/gateway/platforms/discord.py@43e566f77 and hermes-agent/tools/discord_tool.py@43e566f77: Discord gateway and Discord admin/core tools, hermes-agent/optional-skills/productivity/telephony/SKILL.md@43e566f77 and scripts/telephony.py: Twilio, Bland.ai, and Vapi optional telephony skill, hermes-agent/gateway/platforms/sms.py@43e566f77: Twilio-backed SMS gateway contract, repository search 2026-05-24: no first-party Hermes refs found for reddit, stripe API plugin, insforge, graphiti/zep, or fireflies beyond incidental text
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Gormes-owned TUI queued-message widget and busy delivery modes

- Phase: 8 / 8.D
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Adapt Pi's visible steering/follow-up queue pattern into the native Gormes Bubble Tea chat TUI without changing Hermes-compatible slash command semantics. While a turn is active, plain Enter should honor the configured busy-input mode, queued or steering drafts should be visible in the bottom-pinned chrome, queued entries should drain after the kernel returns idle, and the UI must keep Alt/Shift+Enter newline behavior intact.
- Trust class: operator, system
- Ready when: The builder keeps the work in the local Bubble Tea TUI and kernel submit/cancel seams; gateway/channel follow-up behavior remains unchanged., The implementation uses fake frames and pure TUI tests; no provider, gateway process, or live terminal automation is required.
- Not ready when: The slice changes Hermes-compatible Enter, Alt+Enter, Shift+Enter, Ctrl+C, slash dispatch, or active-turn policy semantics instead of only wiring visible queue state., The slice stores queued drafts in a side file, hidden TODO list, or non-session backlog outside the TUI/kernel state path.
- Degraded mode: -
- Fixture: `internal/tui/queued_messages_test.go`
- Write scope: `internal/tui/queued_messages.go`, `internal/tui/update.go`, `internal/tui/view.go`, `internal/tui/hermes_chrome.go`, `internal/tui/*queued*test.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'Test.*Queued\|TestHermesKeybindings_EnterPlainTextHonorsBusyInputMode\|TestHermesChrome' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report focused TUI test output, a short before/after render fixture for queued rows, and progress validation.
- Acceptance: Active-turn drafts submitted in queue mode appear in a three-row queued-message widget above the status rule and do not immediately call Submit., Steer mode shows steering evidence and schedules the draft through the existing active-turn injection path without hiding queued text from the operator., When a frame transitions to idle/failed, queued follow-up drafts drain in FIFO order through the existing submit callback and the widget clears., Alt+Enter and Shift+Enter still insert newlines and never enqueue or submit drafts.
- Source refs: pi@fc8a155 packages/coding-agent/README.md:Message Queue, pi@fc8a155 packages/coding-agent/docs/settings.md:Message Delivery, pi@fc8a155 packages/coding-agent/docs/rpc.md:queue_update events, internal/tui/queued_messages.go:QueuedMessages, internal/tui/update.go:HermesBusyInputMode and ResolveHermesKey, internal/tui/hermes_chrome.go:HermesChromeInput.QueuedMessages, internal/tui/view.go:RenderHermesChrome call site
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Gormes-owned TUI extension status widget and footer seam

- Phase: 8 / 8.D
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Introduce a small Go-native TUI extension context that lets trusted in-process Gormes extensions add or clear footer status entries, widgets above or below the editor, and working-indicator text/frames. The seam should be typed, width-safe, scoped to the active session, and degrade to no-op evidence in non-interactive modes; it must not execute TypeScript or import Pi packages.
- Trust class: operator, system
- Ready when: The builder defines a Go interface or small adapter layer rather than a script runtime; extension callbacks are fakeable in tests., The first slice only covers TUI status/widget/footer/working indicator rendering, not general tool registration or package installation.
- Not ready when: The slice loads third-party executable extension code, adds npm/TypeScript dependencies, or changes Hermes plugin CLI behavior., The extension seam can write files, mutate provider requests, or bypass existing tool safety in this TUI-only slice.
- Degraded mode: -
- Fixture: `internal/tui/status_bar_ext_test.go`
- Write scope: `internal/tui/`, `internal/kernel/extensions.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'Test.*Extension.*(Status\|Widget\|Footer\|Working)\|TestHermesChrome' -count=1`, `go test ./internal/kernel -run TestExtension -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report fake-extension render tests, non-interactive degraded evidence, and progress validation.
- Acceptance: A fake extension can set, replace, and clear a status entry that renders in the Hermes status/footer area without corrupting width-bounded output., A fake extension can set a widget above or below the editor and the widget composes with todo/panel/status chrome ordering., Working-indicator customization applies during active-turn frames and restores the default when cleared., Non-interactive or nil-extension contexts return typed unavailable/no-op evidence instead of panicking.
- Source refs: pi@fc8a155 packages/coding-agent/docs/extensions.md:ctx.ui.setStatus/setWidget/setFooter/setWorkingIndicator, pi@fc8a155 packages/coding-agent/docs/tui.md:Patterns 4-6, pi@fc8a155 packages/coding-agent/examples/extensions/custom-footer.ts, pi@fc8a155 packages/coding-agent/examples/extensions/status-line.ts, internal/tui/status_bar_ext.go:RenderFaceTicker and RenderContextBar, internal/tui/hermes_chrome.go:HermesChromeInput, internal/kernel/extensions.go:ExtensionChain
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
