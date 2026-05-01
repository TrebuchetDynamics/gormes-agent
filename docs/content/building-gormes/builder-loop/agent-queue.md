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
## 1. Tool descriptor layer (OperationSpec)

- Phase: 5 / 5.A
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Every tool in the registry carries a declarative descriptor (OperationSpec) that generates model schemas, CLI commands, gateway slash commands, doctor checks, and audit taxonomy from one source
- Trust class: operator, gateway, child-agent, system
- Ready when: Tool registry inventory + schema parity harness is complete., Hardline command pattern table + DetectHardline function is validated on main.
- Not ready when: The slice ports handler logic instead of adding descriptors around existing handlers., The slice changes the existing Tool interface contract., The slice wires descriptors into live prompt assembly or gateway dispatch before the descriptor schema is fixture-backed.
- Degraded mode: If descriptors are missing, doctor reports tool_descriptor_incomplete and the tool is hidden from gateway/child-agent callers until the descriptor is present.
- Fixture: `internal/tools/operation_spec_test.go`
- Write scope: `internal/tools/operation_spec.go`, `internal/tools/operation_spec_test.go`, `internal/tools/registry.go`, `internal/tools/registry_test.go`, `internal/tools/executor.go`, `internal/tools/executor_test.go`, `internal/doctor/tool_descriptors.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestOperationSpec\|TestTrustClass\|TestExecutor' -count=1`, `go test ./internal/tools -count=1`, `go test ./internal/doctor -run 'TestToolDescriptor' -count=1`, `go run ./cmd/progress validate`
- Done signal: OperationSpec fixtures prove descriptor validation, trust-class rejection, and doctor checks. Core tools carry descriptors. Executor rejects disallowed callers before handler entry.
- Acceptance: OperationSpec struct declares name, description, schema, mutating, idempotent, prompt_safe, allowed trust classes, timeout, and audit kind., Tool registry accepts tools with or without descriptors; tools without descriptors report descriptor_missing in doctor., Shared tool executor rejects disallowed trust classes before a handler runs, with explicit trust_class_denied evidence., Doctor validates every registered tool descriptor for required fields and schema validity., Default toolset assigns descriptors to all core tools (read_file, search_files, write_file, patch, terminal, todo, session_search)., Descriptor schema generation produces valid JSON Schema for model consumption without handler changes.
- Source refs: gbrain:src/core/operations.ts (contract-first operation catalog), mercury-agent:permission manifest (trust-class enforcement), docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md, docs/content/building-gormes/upstream-lessons.md
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Shell blocklist (36+ dangerous patterns)

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: The shared tool executor blocks 36+ dangerous shell patterns before execution, with category-specific evidence and override policies
- Trust class: operator, gateway, child-agent, system
- Ready when: Hardline command pattern table + DetectHardline function is validated on main., Recoverable dangerous patterns + blocked-result schema is complete.
- Not ready when: The slice ports patterns without categorization (destructive, network, privilege, crypto-mining, data-exfil)., The slice allows bypass through off/yolo/manual/smart mode without explicit evidence., The slice changes the existing GuardCommand or approval-mode infrastructure.
- Degraded mode: If the blocklist is incomplete, doctor reports shell_blocklist_partial with missing pattern count; blocked commands still fail closed.
- Fixture: `internal/tools/shell_blocklist_test.go`
- Write scope: `internal/tools/shell_blocklist.go`, `internal/tools/shell_blocklist_test.go`, `internal/tools/dangerous_command.go`, `internal/config/shell_blocklist.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestShellBlocklist' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Shell blocklist fixtures prove 36+ patterns across 5 categories, positive/negative matching, category-specific evidence, GuardCommand integration, and config-driven extensibility.
- Acceptance: ShellBlocklist covers 36+ patterns across 5 categories: destructive (rm -rf, mkfs, dd), network (curl/wget with suspicious flags), privilege (sudo/chown/system-wide), crypto-mining (xmrig, miner), data-exfil (scp/rsync with external targets)., Each blocked pattern returns category-specific evidence (shell_blocklist_destructive, shell_blocklist_network, etc.)., Blocklist integrates with GuardCommand so hardline commands remain non-bypassable and recoverable commands respect approval mode., Blocklist patterns are table-tested with positive and negative cases (e.g., 'rm file.txt' is allowed, 'rm -rf /' is blocked)., New patterns can be added via config without code changes., Doctor reports shell_blocklist_coverage with pattern count and category breakdown.
- Source refs: mercury-agent:src/tools/shellBlocklist.ts (36+ patterns), hermes-agent:tools/terminal_tool.py (dangerous command guards), docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Why now: P0 handoff; needs contract proof before closeout.

## 3. Filesystem scoping (folder-level read/write restrictions)

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: File tools enforce folder-level read/write scope restrictions so agents cannot access paths outside configured boundaries
- Trust class: operator, gateway, child-agent
- Ready when: Native file task tool surface (read_file, search_files, write_file, patch) is validated on main., Path security helpers exist for traversal prevention.
- Not ready when: The slice changes file tool signatures or breaks existing read_file/search_files/write_file/patch behavior., The slice requires interactive prompts for scope configuration., The slice implements path security without centralized normalization.
- Degraded mode: If scoping is unconfigured, doctor reports filesystem_scope_unconfigured and file tools run with cwd-only restriction as fallback.
- Fixture: `internal/tools/filesystem_scope_test.go`
- Write scope: `internal/tools/filesystem_scope.go`, `internal/tools/filesystem_scope_test.go`, `internal/tools/read_file.go`, `internal/tools/write_file.go`, `internal/tools/patch.go`, `internal/tools/search_files.go`, `internal/config/filesystem_scope.go`, `internal/doctor/filesystem_scope.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestFilesystemScope' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Filesystem scope fixtures prove path normalization, read/write rejection, CWD fallback, search exclusion, and doctor reporting without interactive configuration.
- Acceptance: FilesystemScope config declares allowed_read_paths and allowed_write_paths as ordered lists with CWD fallback., Path normalization resolves symlinks, cleans '..', rejects absolute paths outside scope, and records path_normalized evidence., read_file rejects paths outside allowed_read_paths with filesystem_read_scope_violation evidence., write_file and patch reject paths outside allowed_write_paths with filesystem_write_scope_violation evidence., search_files respects read scope and excludes paths outside allowed_read_paths., CWD-only mode (default when unconfigured) restricts all file ops to the current working directory., Doctor reports filesystem_scope_config with active read/write paths and fallback mode.
- Source refs: mercury-agent:src/tools/filesystemScope.ts (folder-level restrictions), hermes-agent:tools/path_security.py, space-agent:server/api/AGENTS.md (path normalization, batch preflight), docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Why now: P0 handoff; needs contract proof before closeout.

## 4. Permission approval UX (inline y/n/always)

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `large`
- Status: `planned`
- Priority: `P0`
- Contract: Dangerous actions trigger an inline approval prompt (y/n/always) with clear command preview, risk category, and persistent preference storage
- Trust class: operator, gateway
- Ready when: Approval mode config normalization is validated on main., Hardline command pattern table + DetectHardline function is validated on main., Recoverable dangerous patterns + blocked-result schema is complete., Shell blocklist (36+ dangerous patterns) is complete.
- Not ready when: The slice implements approval without the GuardCommand and blocked-result infrastructure., The slice changes approval mode semantics (off/manual/smart) without preserving existing behavior., The slice requires gateway/TUI/slash command wiring in the same change.
- Degraded mode: If approval UX is unavailable (non-interactive mode), dangerous commands fail closed with approval_ui_unavailable evidence instead of defaulting to allow.
- Fixture: `internal/tools/approval_ux_test.go`
- Write scope: `internal/tools/approval_ux.go`, `internal/tools/approval_ux_test.go`, `internal/tools/approval_prompt.go`, `internal/tools/approval_session.go`, `internal/audit/approval.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestApprovalUX' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Approval UX fixtures prove prompt rendering, y/n/always handling, session persistence, non-interactive fail-closed, audit logging, and doctor reporting.
- Acceptance: Approval prompt displays command preview, risk category (destructive/network/privilege), and affected paths., User can respond 'y' (once), 'n' (deny), or 'always' (persist for this session/category)., 'always' preference is stored per-session with category scope (e.g., always allow git commands, always deny rm -rf)., Non-interactive contexts (cron, subagent, API) fail closed with approval_ui_unavailable instead of prompting., Approval decisions are logged to audit with command, category, decision, and timestamp., Doctor reports approval_ux_status with interactive/non-interactive mode and persisted preference count.
- Source refs: mercury-agent:src/tools/approval.ts (inline y/n/always), hermes-agent:tools/approval.py, docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Why now: P0 handoff; needs contract proof before closeout.

## 5. Trust-class enforcement in shared tool executor

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: The shared tool executor rejects tool calls from disallowed trust classes before a handler runs, preventing gateway/child-agent callers from exercising operator-local tools
- Trust class: operator, gateway, child-agent, system
- Ready when: Tool descriptor layer (OperationSpec) is complete., Existing tool registry and executor interfaces are stable.
- Not ready when: The slice changes tool handler signatures or breaks existing tool execution paths., The slice implements trust classes without the OperationSpec descriptor layer., The slice wires trust enforcement into gateway or kernel before the executor-level enforcement is fixture-backed.
- Degraded mode: If trust-class enforcement is bypassed or misconfigured, doctor reports trust_class_enforcement_gap and the tool is hidden from disallowed callers.
- Fixture: `internal/tools/trust_class_test.go`
- Write scope: `internal/tools/trust_class.go`, `internal/tools/trust_class_test.go`, `internal/tools/executor.go`, `internal/tools/executor_test.go`, `internal/doctor/trust_class.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestTrustClass' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Trust-class fixtures prove operator/gateway/child-agent/system enforcement, schema visibility gating, and doctor reporting without changing handler signatures.
- Acceptance: TrustClass type defines operator, gateway, child-agent, and system constants., Tool executor checks the caller's trust class against the tool descriptor's allowed trust classes before invoking the handler., Disallowed trust classes return trust_class_denied with the tool name, requested class, and allowed classes., Operator-local tools (config edit, auth, setup) are invisible to gateway and child-agent callers even in tool schemas., Gateway-safe tools are visible to gateway callers but not child-agent callers when so configured., Doctor reports trust_class_enforcement_status with tool count per trust class and any enforcement gaps.
- Source refs: gbrain:src/core/operations.ts (OperationContext.remote trust boundaries), mercury-agent:permission manifest (trust-class based tool visibility), docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/upstream-lessons.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Why now: P0 handoff; needs contract proof before closeout.

## 6. 6 typed memory categories with confidence scoring

- Phase: 6 / 6.G
- Owner: `memory`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Identity, preference, goal, habit, episode, reflection with confidence/durability scoring and conflict resolution
- Trust class: operator, system
- Ready when: Goncho SQLite schema exists for extension
- Not ready when: Goncho schema is still in flux
- Degraded mode: Memory falls back to session-based storage; typed memory unavailable but session history preserved
- Fixture: `internal/goncho/typed_memory_test.go`
- Write scope: `internal/goncho/typed_memory.go`, `internal/goncho/conflict_resolution.go`
- Test commands: `go test ./internal/goncho -run TestTypedMemory -count=1`
- Done signal: Typed memory tests prove CRUD, scoring, conflict resolution, and pruning
- Acceptance: CRUD for 6 memory types, Confidence 0-1 and durability scoring, Conflict resolution: higher confidence wins, equal → newer, Auto-pruning: 21-day active stale, 120-day durable decay
- Source refs: mercury-agent/src/core/memory.ts, honcho/src/models.py, engram/internal/store/store.go
- Unblocks: Memory auto-extraction, Memory consolidation
- Why now: Unblocks Memory auto-extraction, Memory consolidation.

## 7. Regex-based auto-link extraction + brain-first lookup

- Phase: 6 / 6.I
- Owner: `memory`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Markdown links, wikilinks, qualified wikilinks auto-extracted; typed inference; brain-first 5-step lookup
- Trust class: operator, system
- Ready when: Goncho page storage exists
- Not ready when: No local page/slug storage in Goncho
- Degraded mode: Links not auto-extracted; lookup skips local DB/graph and goes directly to LLM/external API
- Fixture: `internal/goncho/auto_link_test.go`
- Write scope: `internal/goncho/auto_link.go`, `internal/goncho/brain_first.go`
- Test commands: `go test ./internal/goncho -run TestAutoLink -count=1`
- Done signal: Auto-link tests prove markdown/wikilink/typed extraction; brain-first tests prove 5-step lookup
- Acceptance: Markdown links [Name](path) auto-extracted, Wikilinks [[slug]] and [[source:dir/slug]] auto-extracted, Typed inference: FOUNDED, INVESTED, ADVISES, WORKS_AT, Brain-first lookup: local DB → graph → cache → LLM → external API
- Source refs: gbrain/src/core/link-extraction.ts, gbrain/src/core/search/hybrid.ts, hermes-agent/agent/context_references.py
- Unblocks: Compiled truth pattern, Tiered enrichment
- Why now: Unblocks Compiled truth pattern, Tiered enrichment.

## 8. xAI Grok provider adapter

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes can route tool-call turns through xAI Grok API with native request/response mapping, streaming, and error classification
- Trust class: system
- Ready when: Provider interface + stream fixture harness is validated on main., Hermes xAI transport sources are available for contract comparison.
- Not ready when: The slice changes provider-neutral event contracts or kernel routing., The slice requires live xAI credentials to prove behavior.
- Degraded mode: Provider status reports grok_unavailable until the adapter has request fixtures, stream decoding, and tool-call normalization.
- Fixture: `internal/hermes/grok_adapter_test.go`
- Write scope: `internal/hermes/grok_adapter.go`, `internal/hermes/grok_adapter_test.go`, `internal/hermes/testdata/grok_transcripts/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run 'TestGrok' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/progress validate`
- Done signal: Grok adapter fixtures prove request shaping, stream decoding, tool-call normalization, and error classification without live credentials.
- Acceptance: Grok adapter converts shared hermes.Message requests into xAI-compatible JSON payloads., Grok stream events decode into provider-neutral stream events (text, reasoning, tool_calls, finish_reason, usage)., Grok tool-call normalization produces hermes-compatible assistant messages and tool continuations., Grok error classification maps rate-limit, auth, and capacity errors into the shared provider-error taxonomy., All tests run without live xAI credentials using replay fixtures.
- Source refs: ../hermes-agent/agent/transports/xai_http.py, ../hermes-agent/agent/transports/xai_schema.py, docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-4-brain-transplant.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. LM Studio provider adapter

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes can route turns through LM Studio local inference server with OpenAI-compatible request/response mapping
- Trust class: system
- Ready when: Provider interface + stream fixture harness is validated on main., OpenAI-compatible chat completions adapter is complete (reuses existing seam).
- Not ready when: The slice invents a new protocol instead of using OpenAI-compatible /v1/chat/completions., The slice requires a live LM Studio server to prove behavior.
- Degraded mode: Provider status reports lmstudio_unavailable until the adapter has request fixtures and local-server discovery.
- Fixture: `internal/hermes/lmstudio_adapter_test.go`
- Write scope: `internal/hermes/lmstudio_adapter.go`, `internal/hermes/lmstudio_adapter_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run 'TestLMStudio' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/progress validate`
- Done signal: LM Studio adapter fixtures prove local OpenAI-compatible routing, model enumeration, and unreachable degradation without a live server.
- Acceptance: LM Studio adapter sends OpenAI-compatible requests to configurable local base URL (default http://localhost:1234/v1)., LM Studio adapter parses streaming SSE responses through existing OpenAI-compatible decoder., LM Studio adapter exposes local model enumeration via /v1/models with fixture evidence., LM Studio adapter reports lmstudio_unreachable when the local server is not responding.
- Source refs: ../hermes-agent/agent/transports/lmstudio_reasoning.py, docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-4-brain-transplant.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Resilient provider chain dispatch

- Phase: 4 / 4.K
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: DeepSeek → OpenAI → Anthropic → Grok → Ollama resilient routing with chain failure detection
- Trust class: operator, system
- Ready when: At least two provider adapters exist to form a chain
- Not ready when: Only one provider adapter exists
- Degraded mode: Single provider failure is surfaced but the chain continues; complete chain failure reports all attempted providers
- Fixture: `internal/hermes/fallback_chain_test.go`
- Write scope: `internal/hermes/fallback_chain.go`, `internal/hermes/error_classifier.go`
- Test commands: `go test ./internal/hermes -run TestFallback -count=1`, `go test ./internal/hermes -run TestErrorClassifier -count=1`
- Done signal: Fallback chain tests prove multi-provider resilient dispatch and error classification
- Acceptance: Chain dispatch tries each provider in order, Failure classification determines retry vs fallback, Degraded mode reports which providers failed
- Source refs: mercury-agent/src/core/providers.ts, picoclaw/pkg/providers/, hermes-agent/agent/error_classifier.py
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
