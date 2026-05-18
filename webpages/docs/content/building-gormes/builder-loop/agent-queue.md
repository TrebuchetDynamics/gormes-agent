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
## 1. Profile workspace allow-list enforcement policy

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Make `agents.defaults.workspaces` the Gormes-owned profile workspace allow-list, not just setup metadata. With an empty list, the default project workspace is the operator home. With a non-empty list, model-facing project read/write access is restricted to the normalized listed roots. Runtime internals may access the active profile root (`GORMES_HOME`) for config, auth, sessions, memory, skills, logs, cron, and gateway state, but model-facing tools must not treat the whole profile root as a project workspace. Model-facing profile edits are limited to explicit profile-owned content: identity files (`SOUL.md`, `IDENTITY.md` when present) and the active profile `skills/` directory. Profile-local `home/` is subprocess HOME/runtime state, not a broad project workspace. Sibling profiles, arbitrary operator-home paths, `.env`, `auth.json`, session/memory databases, logs, and other runtime state are denied as project paths. File tools, local/project execute_code, and coding-agent delegation must share one resolver. Local terminal must use a tested sandbox-capable backend for allow-listed roots or fail closed; merely setting cwd is not accepted as confinement.
- Trust class: operator, system
- Ready when: The completed `gormes setup profiles — section scaffold + per-profile workspace list` row persists the selected profile's workspace list as a TOML array and config.Load round-trips it., The builder can add a single profile workspace policy resolver and inject it from cmd/gormes/registry.go into path-aware tools without hand-parsing config files., Terminal behavior is decided before coding: either a real confinement backend is in scope, or local terminal fails closed under a non-empty allow-list with typed evidence.
- Not ready when: The change only updates docs/setup text and leaves runtime tools unconstrained., The change claims local terminal is sandboxed only because its cwd starts inside an allowed root., The resolver allows `..`, symlink, deleted-cwd, or prefix-sibling escapes; grants sibling profile roots; or silently falls back to unrestricted operator home when a non-empty workspace list is invalid., The active profile root is treated as an unrestricted model-facing project workspace instead of a runtime-owned state root with a narrow editable-content allow-list., The row is merged into `Profile-local subprocess HOME parity`; subprocess HOME and workspace access policy are separate behaviors.
- Degraded mode: If `agents.defaults.workspaces` is empty, the project workspace policy defaults to the operator home for compatibility. If the list is non-empty and the local terminal backend cannot provide real confinement, terminal commands fail closed with `profile_workspace_scope_violation` instead of pretending cwd is a sandbox; path-aware tools still enforce the project allow-list plus explicit profile-owned editable content.
- Fixture: `Temp GORMES_HOME with a named profile containing `agents.defaults.workspaces = ["<project1>", "<project2>"]`, plus active-profile SOUL.md/IDENTITY.md/skills fixtures, profile secret/runtime-state fixtures, sibling-profile fixtures, and outside-root fixtures.`
- Write scope: `internal/config/agents.go`, `internal/config/agents_test.go`, `cmd/gormes/registry.go`, `cmd/gormes/registry_test.go`, `internal/tools/filesystem_scope.go`, `internal/tools/file_task_tools.go`, `internal/tools/file_task_tools_test.go`, `internal/tools/execute_code.go`, `internal/tools/execute_code_test.go`, `internal/tools/terminal_tool.go`, `internal/tools/terminal_tool_test.go`, `internal/codingagents/workspace.go`, `internal/codingagents/workspace_test.go`, `webpages/docs/content/cli/profile.md`, `webpages/docs/content/recipes/profiles.md`, `webpages/docs/profile_docs_test.go`
- Test commands: `go test ./internal/config ./internal/tools ./internal/codingagents -run 'Profile\|Workspace\|Scope\|Filesystem\|ExecuteCode\|Terminal' -count=1`, `go test ./cmd/gormes -run 'Registry\|Profile\|Workspace' -count=1`, `go test ./webpages/docs -run 'Profile\|DocsContent' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Builder reports the shared resolver API, the exact denied-path evidence code, and fixtures proving project1/project2 pass while outside roots and sibling profiles fail; terminal either shows tested confinement or typed fail-closed behavior before shell spawn.
- Acceptance: With no configured workspace list, the profile workspace policy resolves the operator home as the default project read/write root while preserving explicit `agents.defaults.workspace`, `terminal.cwd`, or per-agent workspace overrides where those are intentionally configured., With `agents.defaults.workspaces = [project1, project2]`, read/write/edit/search/code paths inside either project succeed and paths outside both roots fail with stable `profile_workspace_scope_violation` evidence., Runtime internals keep access to the active profile root for profile state, while model-facing file/edit/search/code access is limited to configured project roots plus explicit profile-owned editable content: `SOUL.md`, `IDENTITY.md` when present, and `skills/**`., Profile secrets and runtime state (`.env`, `auth.json`, session/memory databases, logs, gateway state, sibling profiles, and arbitrary operator-home paths) are denied as model-facing project paths under a non-empty workspace list., File tools, execute_code local/project mode, and coding-agent workspace resolution use the same normalized root list and produce matching pass/fail decisions for absolute, relative, symlink, and prefix-sibling paths., Terminal commands under a non-empty allow-list either run through a tested confinement backend rooted in the allow-list or return a typed fail-closed result before spawning a local shell; tests prove cwd-only confinement is rejected., Docs state the distinction between current shipped behavior and the planned Gormes-owned allow-list sandbox policy, including the operator-home default, Hermes' non-sandbox upstream behavior, runtime-owned profile state, and the narrow model-facing profile-owned editable content allowance.
- Source refs: internal/config/agents.go:AgentDefaultsCfg.Workspaces, cmd/gormes/setup.go:runSetupProfilesInteractive writes agents.defaults.workspaces, cmd/gormes/registry.go:buildDefaultRegistry registers file, execute_code, terminal tools, internal/agenttemplate/default_templates.go:SOUL.md and IDENTITY.md identity files, internal/tools/filesystem_scope.go:NewFilesystemScope, internal/tools/file_task_tools.go:FileTaskToolConfig / resolveWorkspacePathFromBase, internal/tools/terminal_tool.go:TerminalTool.Execute / terminalWorkdir, internal/tools/execute_code.go:LocalCodeSandbox.Execute, internal/codingagents/workspace.go:WorkspaceGuard.Resolve, hermes-agent/website/docs/user-guide/profiles.md:Profiles vs workspaces vs sandboxing (upstream says profiles do not sandbox; this row is Gormes-owned)
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. ACP setup-browser bootstrap parity

- Phase: 5 / 5.H
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: `gormes acp --setup-browser` ports Hermes' ACP browser-tool bootstrap behavior with platform-specific command planning, dry-run/report output, and browser harness dependency checks while keeping actual installs explicit and operator-approved.
- Trust class: -
- Ready when: ACP server/client rows are complete and command planning can be tested without installing browser tools.
- Not ready when: The slice downloads browsers or runs package managers during tests., The slice changes ACP JSON-RPC session behavior instead of only adding bootstrap planning.
- Degraded mode: -
- Fixture: `cmd/gormes acp setup-browser dry-run fixtures`
- Write scope: `cmd/gormes/acp.go`, `internal/acp`, `internal/tools`
- Test commands: `go test ./cmd/gormes ./internal/acp -run 'ACP.*SetupBrowser\|ACP.*Bootstrap' -count=1`, `go run ./cmd/progress validate`
- Done signal: ACP setup-browser dry-run and approval fixtures prove platform planning without live downloads.
- Acceptance: Linux/macOS and Windows plans match Hermes script intent and surface missing prerequisites., Dry-run output is deterministic and secret-free., Non-dry-run execution requires explicit operator approval and reports each step outcome.
- Source refs: ../hermes-agent/acp_adapter/bootstrap/bootstrap_browser_tools.sh, ../hermes-agent/acp_adapter/bootstrap/bootstrap_browser_tools.ps1, ../hermes-agent/acp_adapter/entry.py, cmd/gormes/acp.go, internal/acp
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Hermes LSP write-time semantic diagnostics

- Phase: 5 / 5.L
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: After `write_file` or `patch`, Gormes runs a language-server diagnostic pass equivalent to Hermes' write-time LSP surface, shifts baseline ranges through edits, and returns new semantic errors to the agent without blocking unsupported languages.
- Trust class: -
- Ready when: Native write/patch tools and lint-delta rows are complete., A fake diagnostic service can be injected without launching real language servers.
- Not ready when: The slice shells out to real language servers in unit tests., Unsupported languages fail the file operation instead of returning degraded diagnostic evidence.
- Degraded mode: -
- Fixture: `internal/tools LSP diagnostic fake-server fixtures`
- Write scope: `internal/tools`, `internal/lsp`
- Test commands: `go test ./internal/tools -run 'Test.*LSP\|Test.*Diagnostic\|TestWrite\|TestPatch' -count=1`, `go test ./internal/lsp -count=1`, `go run ./cmd/progress validate`
- Done signal: File write/patch fixtures prove LSP diagnostic deltas, range shifting, and graceful unsupported-language degradation.
- Acceptance: Post-write diagnostics report only new or shifted errors relevant to the edited file., Range-shift fixtures cover insert/delete/move edits and preserve baseline diagnostic identity., Unsupported or missing LSP backends return degraded evidence without failing successful file writes.
- Source refs: ../hermes-agent/agent/lsp/manager.py, ../hermes-agent/agent/lsp/range_shift.py, ../hermes-agent/tests/agent/lsp/test_delta_key.py, ../hermes-agent/tests/agent/lsp/test_service.py, internal/tools/file_task_tools.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Hermes x_search tool and auth surface

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Expose Hermes' first-class `x_search` tool in Gormes with a descriptor, OAuth/API-key auth status, query/result envelope, rate-limit/degraded errors, and registry/toolset visibility without requiring live X credentials in tests.
- Trust class: -
- Ready when: Tool registry and auth status helpers can be exercised with fake HTTP and temp config.
- Not ready when: The slice requires a live X OAuth/API-key credential., The slice hides x_search from tool descriptors while adding only a CLI helper.
- Degraded mode: -
- Fixture: `internal/tools x_search fake transport fixtures`
- Write scope: `internal/tools`, `internal/config`, `cmd/gormes/registry.go`
- Test commands: `go test ./internal/tools -run 'TestXSearch\|TestToolRegistry' -count=1`, `go test ./internal/config -run 'TestXSearch\|TestAuth' -count=1`, `go run ./cmd/progress validate`
- Done signal: x_search descriptor, auth status, fake-result normalization, and degraded errors are proven without live X credentials.
- Acceptance: `x_search` appears in the registry with Hermes-compatible schema and toolset availability., OAuth and API-key auth modes produce redacted status and missing-auth diagnostics., Fake search results normalize into a bounded model-visible result envelope; rate-limit and auth failures degrade explicitly.
- Source refs: ../hermes-agent/tools/x_search_tool.py, ../hermes-agent/tools/xai_http.py, ../hermes-agent/tests/tools/test_x_search_tool.py, ../hermes-agent/website/docs/user-guide/features/x-search.md, internal/tools, internal/config
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Hermes send command stdin/file payload parity

- Phase: 5 / 5.O
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: `gormes send` preserves Hermes `hermes send` behavior for stdin/file payload decoding, binary/invalid-text rejection, newline preservation, session targeting, dry/no-agent modes, and TUI resume safety without leaking raw control sequences into terminal output.
- Trust class: -
- Ready when: Hermes send_cmd.py and test_send_cmd.py are available in the in-repo Hermes checkout., The Gormes CLI command tree has a send/chat scripted-entry seam to bind without changing provider runtime behavior.
- Not ready when: The slice changes provider transport, session persistence, or TUI rendering beyond the send command input/output boundary., The implementation accepts undecodable bytes as model-visible text instead of returning bounded operator guidance.
- Degraded mode: -
- Fixture: `cmd/gormes send command tests against Hermes send_cmd fixtures`
- Write scope: `cmd/gormes`, `internal/cli`
- Test commands: `go test ./cmd/gormes -run 'TestSend\|TestHermesSend\|TestTUIResume' -count=1`, `go run ./cmd/progress validate`
- Done signal: Focused CLI fixtures prove send stdin/file decoding, session targeting, TUI resume safety, and sanitized errors without live provider credentials.
- Acceptance: Stdin and file payload paths preserve text and reject undecodable data with a stable, redacted error., Session target and resume behavior match Hermes tests without starting live providers., Terminal control bytes are sanitized before any human-mode output.
- Source refs: ../hermes-agent/hermes_cli/send_cmd.py, ../hermes-agent/tests/hermes_cli/test_send_cmd.py, ../hermes-agent/tests/hermes_cli/test_tui_resume_flow.py, cmd/gormes
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Hermes session recap command surface

- Phase: 5 / 5.O
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Port Hermes' session recap command as a Gormes-native read-only session summarizer over local session/transcript storage, preserving output modes, missing-session diagnostics, and provider-free degraded behavior.
- Trust class: -
- Ready when: Gormes session list/export storage helpers are available for hermetic temp-store tests.
- Not ready when: The slice calls a live model to summarize instead of first proving the read-only recap envelope and degraded provider-free path., The slice mutates session history while generating a recap.
- Degraded mode: -
- Fixture: `cmd/gormes session recap fixtures`
- Write scope: `cmd/gormes`, `internal/session`, `internal/store`
- Test commands: `go test ./cmd/gormes -run 'TestSessionRecap\|TestSession' -count=1`, `go test ./internal/session ./internal/store -run Recap -count=1`, `go run ./cmd/progress validate`
- Done signal: Session recap fixtures prove read-only transcript loading, bounded output, missing-session diagnostics, and no live provider dependency.
- Acceptance: Known session transcripts render a deterministic recap envelope in human and JSON modes., Missing or empty sessions return explicit diagnostics without panics or live provider calls., Long transcripts are bounded with visible truncation evidence.
- Source refs: ../hermes-agent/hermes_cli/session_recap.py, ../hermes-agent/hermes_cli/main.py, internal/session, internal/store
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Long-term plan: profile fleet supervisor and single control-plane gateway

- Phase: 5 / 5.O
- Owner: `orchestrator`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Define Gormes' long-term profile-fleet runtime so operators get one control surface for all named profiles while preserving Hermes-compatible profile state separation. The near-term per-profile gateway services remain a compatibility bridge; the target is a fleet supervisor that can enumerate configured profiles, start/stop/restart profile-scoped workers or a proven profile-scoped in-process equivalent, validate token ownership, surface per-profile health, and coordinate update/restart-all flows without sharing config, auth, sessions, memory, tool state, or kernels across profiles.
- Trust class: operator, gateway, system
- Ready when: The current per-profile gateway-service bridge is documented as migration/runtime compatibility, not the final operator model., Gormes-owned profile workspace/channel config and token-scoped gateway locks are available as inputs., The implementation shape chooses either isolated worker processes or a tested profile-scoped in-process runtime, with the same operator-facing fleet contract.
- Not ready when: The design treats a single gateway process as permission to reuse one GORMES_HOME, one auth store, one session DB, one memory DB, or one kernel across multiple named profiles., The slice deletes or disables the per-profile service bridge before the fleet supervisor can prove profile/token isolation and restart-all behavior., Tests require live Telegram tokens, live systemd units, or Juan's real profile directories.
- Degraded mode: If fleet supervision is unavailable, Gormes must keep the Hermes-compatible per-profile service/process bridge and report exact per-profile service state instead of collapsing profiles into the default GORMES_HOME.
- Fixture: `internal/gateway/fleet_supervisor_test.go; cmd/gormes/gateway_fleet_test.go`
- Write scope: `cmd/gormes/gateway.go`, `cmd/gormes/gateway_fleet_test.go`, `internal/gateway/fleet_supervisor.go`, `internal/gateway/fleet_supervisor_test.go`, `internal/config/agents.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`, `webpages/docs/content/building-gormes/modules/profiles.md`
- Test commands: `go test ./internal/gateway -run 'TestFleetSupervisor\|TestGatewayFleet' -count=1`, `go test ./cmd/gormes -run 'TestGatewayFleet' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: The profiles module documents one operator-facing fleet gateway/supervisor target, preserves profile isolation as non-negotiable, and makes the current per-profile services an explicit compatibility bridge rather than silent architecture drift.
- Acceptance: Fleet status JSON lists every configured profile with desired channels, runtime owner, version, health, last error, and token-lock evidence., Start/stop/restart-all paths operate on all configured profiles while preserving isolated GORMES_HOME, config, auth, session, memory, and tool state per profile., A duplicate Telegram token across profiles is detected and reported as a per-profile conflict rather than racing two pollers., Update/release restart hooks can target the fleet through one operator-facing command or service instead of requiring hand-managed unit names., Regression tests use fake profile roots and fake supervisors only; no live systemd, Telegram, or provider credentials are required.
- Source refs: webpages/docs/content/upstream-hermes/developer-guide/architecture.md:Profile isolation, webpages/docs/content/upstream-hermes/developer-guide/gateway-internals.md:profile-scoped process tracking, webpages/docs/content/upstream-hermes/reference/cli-commands.md:gateway --all, webpages/docs/content/upstream-hermes/reference/faq.md:multiple profiles and bot tokens, cmd/gormes/gateway.go:gatewayManagerConfig, internal/config/agents.go:AgentDefaultsCfg, internal/gateway/manager.go:ManagerConfig.ContextFilesProfile
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Native TUI Terminal.app truecolor and ANSI sanitizer parity

- Phase: 5 / 5.Q
- Owner: `tui`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Port Hermes Ink TUI Terminal.app/truecolor and ANSI sanitizer behavior into the native Gormes TUI so renderer output keeps cursor/source-of-truth stability, strips malformed CSI safely, and preserves readable color behavior across modern terminals.
- Trust class: -
- Ready when: Native TUI text rendering and input fast-echo helpers can be tested without launching an interactive terminal.
- Not ready when: The slice requires a live Terminal.app session or snapshots from a developer machine., The slice changes TUI layout or slash dispatch outside text/color/input sanitizer behavior.
- Degraded mode: -
- Fixture: `internal/tui Terminal.app/ANSI sanitizer fixtures`
- Write scope: `internal/tui`, `internal/tuigateway`, `cmd/gormes`
- Test commands: `go test ./internal/tui ./internal/tuigateway ./cmd/gormes -run 'Truecolor\|ANSI\|Terminal\|TextInput\|Resume' -count=1`, `go run ./cmd/progress validate`
- Done signal: Native TUI fixtures prove truecolor environment handling, ANSI sanitizer safety, and fast-echo cursor stability without a live terminal.
- Acceptance: Malformed or dangling ANSI/CSI sequences are stripped or bounded exactly by fixture expectations., Truecolor forcing/degradation is deterministic from injected terminal environment facts., Fast-echo cursor source-of-truth does not drift after sanitized writes.
- Source refs: ../hermes-agent/ui-tui/src/lib/forceTruecolor.ts, ../hermes-agent/ui-tui/src/lib/text.ts, ../hermes-agent/ui-tui/src/components/textInput.tsx, ../hermes-agent/ui-tui/src/__tests__/forceTruecolor.test.ts, ../hermes-agent/ui-tui/src/__tests__/text.test.ts, ../hermes-agent/ui-tui/src/__tests__/textInputFastEcho.test.ts, internal/tui
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Hermes v0.14 optional skill catalog refresh

- Phase: 6 / 6.C
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Refresh the Gormes skill catalog and metadata compatibility checks against Hermes v0.14 optional skills, including devops/pinggy-tunnel, research/darwinian-evolver, research/osint-investigation, and the updated Notion skill, without blindly copying unsupported Python scripts into runtime packages.
- Trust class: -
- Ready when: Skill metadata parser and hub registry fixtures exist.
- Not ready when: The slice vendors Hermes optional-skill scripts as trusted Go runtime code., The slice marks skills enabled by default without platform/dependency guards.
- Degraded mode: -
- Fixture: `internal/skills optional skill catalog fixtures`
- Write scope: `internal/skills`, `docs/development-skills`, `docs/content/building-gormes/architecture_plan`
- Test commands: `go test ./internal/skills -run 'Test.*Skill.*Catalog\|Test.*Optional' -count=1`, `go run ./cmd/progress validate`
- Done signal: Optional skill fixtures prove v0.14 metadata/catalog visibility and guarded unsupported-script handling.
- Acceptance: New optional skills parse with frontmatter, loaded/when metadata, references, and script/template inventories., Unsupported scripts remain catalog evidence with explicit dependency/degraded status., Skill hub/search output surfaces these skills with category and safety metadata.
- Source refs: ../hermes-agent/optional-skills/devops/pinggy-tunnel/SKILL.md, ../hermes-agent/optional-skills/research/darwinian-evolver/SKILL.md, ../hermes-agent/optional-skills/research/osint-investigation/SKILL.md, ../hermes-agent/skills/productivity/notion/SKILL.md, internal/skills
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. SimpleX Chat platform plugin parity

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Port Hermes' SimpleX Chat platform plugin into Gormes behind the shared channel adapter contract: local daemon/WebSocket configuration, allowlist admission, opaque contact IDs, DM pairing, outbound delivery, command routing, and status/degraded evidence.
- Trust class: -
- Ready when: Gateway platform manifest already classifies SimpleX as row-backed., Shared channel adapter fixtures can run without a live SimpleX daemon.
- Not ready when: The slice requires a real SimpleX account, daemon, or network socket in tests., The slice bypasses shared gateway admission/delivery abstractions.
- Degraded mode: -
- Fixture: `internal/channels/simplex fake WebSocket fixtures`
- Write scope: `internal/channels/simplex`, `internal/gateway`, `cmd/gormes/gateway.go`
- Test commands: `go test ./internal/channels/simplex ./internal/gateway -run 'SimpleX\|PlatformManifest\|Connected' -count=1`, `go run ./cmd/progress validate`
- Done signal: SimpleX fake-daemon fixtures prove config/status, inbound admission, outbound delivery, DM pairing, and command routing without live credentials.
- Acceptance: Config/status checks distinguish disabled, missing ws_url, unauthorized, and connected fake-daemon states., Inbound fake events produce normalized PlatformEvent values with opaque contact identity preserved., Outbound fake delivery and DM pairing preserve Hermes-visible SimpleX behavior and degraded errors.
- Source refs: ../hermes-agent/plugins/platforms/simplex/plugin.yaml, ../hermes-agent/plugins/platforms/simplex/adapter.py, internal/gateway/platform_manifest.go, internal/gateway/platform_connected_checkers.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
