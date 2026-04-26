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
autonomous implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared unattended-loop facts live in [Builder Loop Handoff](../builder-loop-handoff/):
the main entrypoint, orchestrator plan, candidate source, generated docs,
tests, and candidate policy. Keep those control-plane facts in
`meta.builder_loop`, and keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Watchdog checkpoint coalescing

- Phase: 1 / 1.C
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Pure helper internal/builderloop/watchdog_coalesce.go exposes type Decision string with constants DecisionFirst="first", DecisionAmend="amend", DecisionNoop="noop"; type CheckpointState struct{LastCheckpointAt time.Time; LastSubject string; WindowID string}; type CoalesceConfig struct{WindowSeconds int; Dirty bool; NextWindowID func() string} and one function DecideCheckpoint(now time.Time, st CheckpointState, cfg CoalesceConfig) (Decision, CheckpointState). Window math: windowSeconds <= 0 falls back to 600. Algorithm: when cfg.Dirty is false, return DecisionNoop and the input state unchanged. When cfg.Dirty is true and (st.WindowID == "" OR now.Sub(st.LastCheckpointAt) >= window), return DecisionFirst with state {LastCheckpointAt: now, LastSubject: st.LastSubject (caller fills), WindowID: cfg.NextWindowID()}. Otherwise (cfg.Dirty and inside the window) return DecisionAmend with state {LastCheckpointAt: now, LastSubject: st.LastSubject, WindowID: st.WindowID} (preserve the existing WindowID). No git invocation, no shell-script change, no live filesystem mutation in this slice; cfg.NextWindowID is the only source of new IDs and is supplied by the caller (typically a small monotonic counter or a uuid stub in tests).
- Trust class: operator, system
- Ready when: Watchdog dirty-worktree checkpointing is in place (commit ff96a5d94) and emits a record_run_health event we can key off of., Tests can use a fake clock and an in-memory git repo or a synthetic commit-recorder seam — no live system clock or systemd is required.
- Not ready when: The slice changes the watchdog stall threshold, the dead-process detection, or the planner cadence., The slice silently drops the dirty-worktree checkpoint when a stall is real — the first tick of every distinct stall window must still produce a single observable checkpoint.
- Degraded mode: Watchdog status reports checkpoint_coalesce_active, checkpoint_coalesce_window_seconds, and the existing dirty-worktree checkpoint commit ID instead of emitting a fresh commit on every tick.
- Fixture: `internal/builderloop/watchdog_coalesce_test.go`
- Write scope: `internal/builderloop/watchdog_coalesce.go`, `internal/builderloop/watchdog_coalesce_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/builderloop -run TestDecideCheckpoint -count=1`, `go test ./internal/builderloop -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/builderloop/watchdog_coalesce_test.go fixtures prove first/amend/first-after-rotation/noop decisions across a fake clock with no live git or systemd.
- Acceptance: TestDecideCheckpoint_FirstTickInFreshWindowReturnsFirst: empty state, cfg.Dirty=true, NextWindowID=()=>"w-1" returns (DecisionFirst, {LastCheckpointAt: now, WindowID: "w-1"})., TestDecideCheckpoint_LaterTickInsideWindowReturnsAmend: prior state {LastCheckpointAt: now-30s, WindowID: "w-1"}, cfg.Dirty=true, windowSeconds=600 returns (DecisionAmend, {LastCheckpointAt: now, WindowID: "w-1"}); NextWindowID is NOT called., TestDecideCheckpoint_LaterTickPastWindowReturnsFirst: prior state {LastCheckpointAt: now-601s, WindowID: "w-1"}, cfg.Dirty=true, windowSeconds=600, NextWindowID=()=>"w-2" returns (DecisionFirst, {LastCheckpointAt: now, WindowID: "w-2"})., TestDecideCheckpoint_NoopWhenNotDirty: any prior state, cfg.Dirty=false returns (DecisionNoop, prior state unchanged); NextWindowID is NOT called., TestDecideCheckpoint_DefaultWindowWhenZeroOrNegative: cfg.WindowSeconds=0 and cfg.WindowSeconds=-5 both behave like windowSeconds=600., Helper is a pure function with no time.Now, no git invocation, no os.* calls — caller passes both the clock (now) and the dirty flag (cfg.Dirty) and the WindowID generator (cfg.NextWindowID).
- Source refs: internal/builderloop/run.go:CheckpointDirtyWorktree,lastCommitIsBuilderLoopCheckpoint,isBuilderLoopCheckpointSubject, scripts/orchestrator/watchdog.sh:checkpoint_dirty, docs/superpowers/specs/2026-04-25-builder-owned-planner-cycle-design.md
- Unblocks: Watchdog dead-process vs slow-progress separation
- Why now: Unblocks Watchdog dead-process vs slow-progress separation.

## 2. Watchdog dead-process vs slow-progress separation

- Phase: 1 / 1.C
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Pure helper internal/builderloop/watchdog_state.go exposes type Verdict string with constants VerdictHealthy="healthy", VerdictSlow="slow", VerdictDead="dead" (zero value = VerdictHealthy is NOT used; the helper always sets one of the three explicitly); type WorkerVitals struct{PID int; LastCommitAt time.Time; PIDIsLive bool}; and one function Diagnose(now time.Time, v WorkerVitals, deadAfter, slowAfter time.Duration) Verdict. Algorithm (evaluated in this order — first match wins): (1) when v.PID == 0, return VerdictDead (zero PID is treated as missing, regardless of PIDIsLive or thresholds). (2) when v.PIDIsLive == false AND now.Sub(v.LastCommitAt) >= deadAfter, return VerdictDead. (3) when v.PIDIsLive == true AND now.Sub(v.LastCommitAt) >= slowAfter, return VerdictSlow. (4) otherwise return VerdictHealthy. Threshold preconditions enforced by tests: deadAfter and slowAfter are both > 0 and deadAfter <= slowAfter; the helper does NOT validate them — the orchestrator config layer does. No os.FindProcess, no signal delivery, no time.Now lookup, no goroutines. Caller injects v.PIDIsLive (typically a small shim around os.FindProcess + Signal(0) in production; a fixed bool in tests).
- Trust class: operator, system
- Ready when: Existing watchdog timer (commit f96a5d94) emits stall events at a single threshold; this slice carves the threshold into two independent ones., Watchdog checkpoint coalescing is fixture-ready or validated, so the dead-process tick does not amplify the commit storm.
- Not ready when: The slice changes how worker output is rejected or how dirty worktrees are committed — only worker liveness detection is in scope., The slice introduces process-group signal sending or container-aware death detection (those belong to a separate sandboxing row).
- Degraded mode: Watchdog status reports worker_state ∈ {alive_progressing, alive_silent, dead, unknown} and the threshold each one tripped; record_run_health carries the worker_state and which threshold (dead_after_seconds, silent_after_seconds) fired.
- Fixture: `internal/builderloop/watchdog_state_test.go`
- Write scope: `internal/builderloop/watchdog_state.go`, `internal/builderloop/watchdog_state_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/builderloop -run TestDiagnose -count=1`, `go test ./internal/builderloop -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/builderloop/watchdog_state_test.go fixtures prove healthy/slow/dead verdicts including the dead-vs-slow precedence rule with no os.FindProcess or signal calls.
- Acceptance: TestDiagnose_HealthyWhenRecentCommitAndAlive: v={PID:1234, LastCommitAt:now-5s, PIDIsLive:true}, deadAfter=120s, slowAfter=600s returns VerdictHealthy., TestDiagnose_SlowWhenAliveButOverSlowThreshold: v={PID:1234, LastCommitAt:now-700s, PIDIsLive:true} returns VerdictSlow., TestDiagnose_DeadWhenPIDNotLiveAndOverDeadThreshold: v={PID:1234, LastCommitAt:now-200s, PIDIsLive:false} returns VerdictDead., TestDiagnose_DeadWhenPIDIsZero: v={PID:0, LastCommitAt:now-1s, PIDIsLive:true} returns VerdictDead (zero PID short-circuits; thresholds and PIDIsLive are ignored)., TestDiagnose_DeadDoesNotDowngradeToSlow: v={PID:1234, LastCommitAt:now-700s, PIDIsLive:false} with deadAfter=120s, slowAfter=600s returns VerdictDead (dead wins over slow when both fire)., TestDiagnose_NotDeadWhenPIDLiveEvenIfSilent: v={PID:1234, LastCommitAt:now-99999s, PIDIsLive:true} returns VerdictSlow (never VerdictDead while the process answers Signal(0))., Helper is pure — caller injects the clock (now) and the PIDIsLive result.
- Source refs: internal/builderloop/run.go, internal/builderloop/run_health_test.go
- Unblocks: Builder-loop self-improvement vs user-feature ratio metric
- Why now: Unblocks Builder-loop self-improvement vs user-feature ratio metric.

## 3. Builder-loop self-improvement vs user-feature ratio metric

- Phase: 1 / 1.C
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Pure helper internal/builderloop/ship_ratio.go exposes ClassifySubphase(subphaseID string) RowKind (kinds: self_improvement, user_feature, unclassified) and ComputeShipRatio(events []ShippedRowEvent, window time.Duration, now time.Time) ShipRatio where ShipRatio counts each kind in the [now-window, now] band. The classifier table maps 1.C/control-plane/* and 5.O operator-tooling rows to self_improvement, and 4.*/6.*/7.* to user_feature, with everything else as unclassified. No file I/O, no record_run_health emission.
- Trust class: system
- Ready when: record_run_health is the canonical health signal already (commit 2653a7b6 etc.) and runs.jsonl carries shipped row evidence., A subphase classifier mapping (e.g., 1.C, 5.* operator-tools, etc. → self_improvement; 4.*, 6.*, 7.* → user_feature) can live in a small in-package table for now.
- Not ready when: The slice changes ship-detection or ledger-write semantics — only adds a derived counter., The slice depends on a yet-unbuilt classifier service or external store.
- Degraded mode: When the ship_ratio cannot be computed (insufficient history, classification ambiguous), record_run_health carries ship_ratio=null and reports ship_ratio_evidence with the reason instead of fabricating zero.
- Fixture: `internal/builderloop/ship_ratio_test.go`
- Write scope: `internal/builderloop/ship_ratio.go`, `internal/builderloop/ship_ratio_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/builderloop -run 'TestClassifySubphase\|TestComputeShipRatio' -count=1`, `go test ./internal/builderloop -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/builderloop/ship_ratio_test.go fixtures prove classifier coverage of self_improvement/user_feature/unclassified plus windowed aggregation with no record_run_health writes.
- Acceptance: TestClassifySubphase_KnownSelfImprovement covers '1.C', 'control-plane/backend', '5.O/CLI log snapshot reader' returning RowKindSelfImprovement., TestClassifySubphase_KnownUserFeature covers '4.A', '4.H', '6.A', '7.B' returning RowKindUserFeature., TestClassifySubphase_UnknownReturnsUnclassified (e.g., '99.X') returns RowKindUnclassified., TestComputeShipRatio_FiltersByWindow excludes events older than now-window., TestComputeShipRatio_CountsAllKinds returns SelfImprovement, UserFeature, Unclassified counts and a Total., Helper is a pure function — caller injects the events slice and the clock.
- Source refs: internal/builderloop/run.go, internal/builderloop/health_writer.go, internal/progress/health.go, internal/builderloop/ledger.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Azure Foundry probe — path sniffing

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Pure helper internal/hermes/azure_foundry_path_sniff.go exposes type AzureTransport string with three string-typed constants — AzureTransportAnthropic = "anthropic", AzureTransportOpenAI = "openai", AzureTransportUnknown = "" (zero value) — and one function ClassifyAzurePath(rawURL string) AzureTransport. Algorithm: parse rawURL with url.Parse; on parse error or empty result, return AzureTransportUnknown. Otherwise lowercase parsed.Path with strings.ToLower; if the lowercased path equals "/anthropic", ends with the suffix "/anthropic", or contains the substring "/anthropic/", return AzureTransportAnthropic. Every other input — including bare hosts, /openai/* paths, missing paths, or paths matching /anthropicx (no trailing slash) — returns AzureTransportUnknown. The OpenAI constant is declared so the follow-up /models classification slice can return it; this slice never returns AzureTransportOpenAI. No HTTP, no env reads, no config writes, no goroutines.
- Trust class: operator, system
- Ready when: internal/hermes already compiles and has no azure_foundry_path_sniff.go file yet — this row creates the file plus a sibling _test.go., No upstream gating: this is a pure URL inspector with synthetic input.
- Not ready when: The slice opens HTTP connections, performs a /models probe, reads AZURE_FOUNDRY_BASE_URL or AZURE_FOUNDRY_API_KEY, or mutates config., The slice introduces detection of any third transport family (Bedrock, Vertex, etc.).
- Degraded mode: Probe status reports azure_path_sniff_unknown when no path heuristic matches, and azure_path_sniff_evidence with detected scheme/host/path otherwise.
- Fixture: `internal/hermes/azure_foundry_path_sniff_test.go`
- Write scope: `internal/hermes/azure_foundry_path_sniff.go`, `internal/hermes/azure_foundry_path_sniff_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run TestClassifyAzurePath -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/hermes/azure_foundry_path_sniff_test.go fixtures prove anthropic-path classification across suffix, mid-path, and case variants without HTTP.
- Acceptance: TestClassifyAzurePath_AnthropicSuffix: https://x.openai.azure.com/openai/deployments/y/anthropic returns AzureTransportAnthropic., TestClassifyAzurePath_AnthropicMidPath: https://x/openai/anthropic/v1/messages returns AzureTransportAnthropic., TestClassifyAzurePath_CaseInsensitive: /AnthrOPic and /ANTHROPIC both return AzureTransportAnthropic., TestClassifyAzurePath_OpenAIDefault: https://x.openai.azure.com/openai/v1/chat/completions returns AzureTransportUnknown (never AzureTransportOpenAI in this slice)., TestClassifyAzurePath_MalformedReturnsUnknown: empty string, "::garbage::", and rawURL="http://%zz" return AzureTransportUnknown., TestClassifyAzurePath_NotASubstringFalsePositive: /anthropicx and /anthropic-mirror return AzureTransportUnknown (substring guard requires "/anthropic" with a trailing path separator or end-of-path).
- Source refs: ../hermes-agent/hermes_cli/azure_detect.py:_looks_like_anthropic_path:114
- Unblocks: Azure Foundry probe — /models classification + Anthropic fallback
- Why now: Unblocks Azure Foundry probe — /models classification + Anthropic fallback.

## 5. [SYSTEM:→[IMPORTANT: meta-instruction prefix rename for Azure content filter compatibility

- Phase: 4 / 4.C
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Single rename of the bracketed meta-instruction prefix used for skill-invocation and cron-heartbeat prompts: change every occurrence of the literal string "[SYSTEM:" to "[IMPORTANT:" in (a) internal/skills/commands.go (currently at line 94 inside the BuildSkill block string template) and (b) internal/cron/heartbeat.go's CronHeartbeatPrefix constant. The replacement is byte-for-byte over a 7→10 character literal; semantic meaning to the model is unchanged. Update the two existing assertions in internal/skills/preprocessing_commands_test.go (string literal `[SYSTEM: The user has invoked`) and the two assertions in internal/cron/heartbeat_test.go (HasPrefix and contained "[SYSTEM:") to match the new prefix. No public Go API renames, no struct shape changes, no provider/transport edits, no schema migrations.
- Trust class: system
- Ready when: Both internal/skills/commands.go and internal/cron/heartbeat.go currently use the literal "[SYSTEM:" prefix (verified 2026-04-26 by grep)., Both *_test.go files assert against the literal "[SYSTEM:" prefix and will need synchronized updates inside this slice's write_scope., No upstream gate beyond the upstream Hermes commit d7a34682 already merged.
- Not ready when: The slice changes the body content of the cron heartbeat or skill-invocation prompts — only the bracketed prefix token rename is in scope., The slice introduces a runtime feature flag, environment variable, or config knob to switch between [SYSTEM: and [IMPORTANT: — the rename is unconditional., The slice touches gateway prompt assembly, provider transports, or any runtime that does not currently emit the [SYSTEM: marker.
- Degraded mode: Cron-heartbeat and skill-invocation prompts continue to assemble in the same shape; only the leading bracketed marker token differs. Azure OpenAI content-filter (Default/DefaultV2) no longer rejects with HTTP 400 'prompt-injection attempt' when these markers appear at message head.
- Fixture: `internal/cron/heartbeat_test.go`
- Write scope: `internal/skills/commands.go`, `internal/cron/heartbeat.go`, `internal/skills/preprocessing_commands_test.go`, `internal/cron/heartbeat_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cron -run TestCronHeartbeat -count=1`, `go test ./internal/skills -run TestPreprocessingCommands -count=1`, `go test ./internal/skills ./internal/cron -count=1`, `go vet ./...`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/cron/heartbeat_test.go and internal/skills/preprocessing_commands_test.go assert the [IMPORTANT: prefix and the codebase contains zero remaining "[SYSTEM:" string literals under internal/skills and internal/cron.
- Acceptance: TestCronHeartbeatBuildPromptUsesImportantPrefix asserts BuildPrompt output starts with "[IMPORTANT:" and contains the existing scheduled-cron-job description body unchanged., TestCronHeartbeatPrefixConstantValue asserts CronHeartbeatPrefix is exactly the new "[IMPORTANT: You are running as a scheduled cron job. " prefix., TestPreprocessingCommandsRendersImportantMarker (updated existing test) asserts the rendered skill block contains "[IMPORTANT: The user has invoked" and no longer contains "[SYSTEM:" anywhere., TestSkillsCommandsBuildBlockNeverEmitsLegacyMarker (new) renders BuildSkillBlock against a synthetic active skill and asserts strings.Contains(out, "[SYSTEM:") == false., go vet ./... and go build ./... pass with no compile errors after the rename.
- Source refs: ../hermes-agent/agent/skill_commands.py@d7a34682, ../hermes-agent/cron/scheduler.py@d7a34682, internal/skills/commands.go:94, internal/cron/heartbeat.go:16, internal/skills/preprocessing_commands_test.go:181, internal/cron/heartbeat_test.go:11
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Provider rate guard — x-ratelimit header classification

- Phase: 4 / 4.H
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Pure helper internal/hermes/provider_rate_guard.go exposes type RateLimitClass string with constants RateLimitGenuineQuota, RateLimitUpstreamCapacity, RateLimitInsufficientEvidence (zero value = RateLimitInsufficientEvidence) and Classify429(headers http.Header) RateLimitClass. The function reads exactly four candidate buckets — x-ratelimit-remaining-1h, x-ratelimit-remaining-1m, x-ratelimit-remaining-requests, x-ratelimit-remaining-tokens — via headers.Get (Go-canonical case insensitive). A bucket counts as 'present' when headers.Values returns at least one entry whose strings.TrimSpace value parses as a base-10 int via strconv.Atoi. Returns RateLimitGenuineQuota if any present bucket parses to <=0; RateLimitUpstreamCapacity when at least one bucket is present and every present bucket parses to >0; RateLimitInsufficientEvidence when none of the four bucket headers is present (parse failures count as not-present, never as -1). No Bucket struct, no reset evidence, no time.Now, no shared/process state. Bucket/reset detail is the next slice.
- Trust class: system
- Ready when: internal/hermes already compiles; the row creates a new file and a sibling _test.go., No upstream gate; pure header parsing with synthetic http.Header values.
- Not ready when: The slice changes retry timing, provider routing, or model fallback policy., The slice writes process-global breaker state in unit tests or sleeps to simulate reset windows.
- Degraded mode: Provider status reports rate_guard_classified as one of {genuine_quota, upstream_capacity, insufficient_evidence}, plus reset-window evidence when present, instead of silently tripping a global breaker.
- Fixture: `internal/hermes/provider_rate_guard_classification_test.go`
- Write scope: `internal/hermes/provider_rate_guard.go`, `internal/hermes/provider_rate_guard_classification_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run TestClassify429 -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/hermes/provider_rate_guard_classification_test.go fixtures prove genuine_quota / upstream_capacity / insufficient_evidence classification with redacted reset windows under a fake clock.
- Acceptance: TestClassify429_GenuineQuotaWhenAnyBucketExhausted (X-RateLimit-Remaining-1h=0) returns RateLimitGenuineQuota even when other present buckets are >0., TestClassify429_UpstreamCapacityWhenAllBucketsHaveRemaining (any subset of the four buckets >0, none missing-and-parsed-as-zero) returns RateLimitUpstreamCapacity., TestClassify429_InsufficientEvidenceWhenNoRateHeaders returns RateLimitInsufficientEvidence and equals the RateLimitClass zero value., TestClassify429_IgnoresUnknownHeaders (Retry-After, X-Custom-Foo) preserves the classification driven solely by the four x-ratelimit-remaining-* buckets., TestClassify429_UnparseableBucketIsNotPresent (X-RateLimit-Remaining-Tokens="abc") with no other rate headers returns RateLimitInsufficientEvidence rather than treating the malformed value as zero.
- Source refs: ../hermes-agent/agent/nous_rate_guard.py:is_genuine_nous_rate_limit:191
- Unblocks: Provider rate guard — degraded-state + last-known-good evidence
- Why now: Unblocks Provider rate guard — degraded-state + last-known-good evidence.

## 7. Skills list — enabled/disabled status column + --enabled-only filter

- Phase: 5 / 5.F
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Contract: internal/skills/list.go exposes type SkillStatus string ("enabled", "disabled"), extends SkillRow with a Status field and adds ListOptions{Source string; EnabledOnly bool}. ListInstalledSkills(opts ListOptions, disabled map[string]struct{}) []SkillRow returns every installed skill annotated with Status from the disabled set; when opts.EnabledOnly is true, disabled rows are filtered out. The CLI surface (gormes skills list --source <s> --enabled-only) calls this helper and prints a table with a Status column plus a summary "N enabled, M disabled" (or "K enabled shown" when --enabled-only). No platform-aware override read in this slice — disabled set comes from the active profile only, mirroring upstream do_list semantics.
- Trust class: operator, system
- Ready when: internal/skills already lists installed skills (existing list.go or equivalent) and has a typed disabled-skill set the active-profile config exposes., CLI table rendering exists for skills already (status column is an additional column).
- Not ready when: The slice plumbs a HERMES_PLATFORM-style platform override into list — upstream test_do_list_platform_env_is_ignored asserts the platform arg stays nil here., The slice rewrites do_check, do_install, or do_search behavior.
- Degraded mode: Status column makes disabled skills visible without forcing the operator to read config; --enabled-only matches the upstream "what will load" introspection question.
- Fixture: `internal/skills/list_test.go`
- Write scope: `internal/skills/list.go`, `internal/skills/list_test.go`, `internal/cli/skills_command.go`, `internal/cli/skills_command_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/skills -run 'TestListInstalledSkills' -count=1`, `go test ./internal/cli -run 'TestSkillsListCommand' -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/skills/list_test.go and internal/cli/skills_command_test.go fixtures prove the Status column, --enabled-only filter, and platform-arg guard.
- Acceptance: TestListInstalledSkills_StatusColumnPopulated annotates every row with Status="enabled" when disabled is empty., TestListInstalledSkills_DisabledRowsCarryDisabledStatus marks rows whose name is in the disabled set as Status="disabled"., TestListInstalledSkills_EnabledOnlyFilter hides disabled rows when opts.EnabledOnly is true., TestListInstalledSkills_SourceFilterRespected restricts rows to the requested source ("hub"\|"builtin"\|"local")., TestSkillsListCommand_RendersStatusColumnAndSummary prints the Status column and "N enabled, M disabled" footer (or "K enabled shown" with --enabled-only)., TestSkillsListCommand_PlatformArgNotPropagated proves the disabled-set lookup does not pass a platform override.
- Source refs: ../hermes-agent/hermes_cli/skills_hub.py:do_list@0e2a53ea, ../hermes-agent/hermes_cli/main.py:skills_list_parser@0e2a53ea, ../hermes-agent/tests/hermes_cli/test_skills_hub.py:test_do_list_renders_status_column, ../hermes-agent/tests/hermes_cli/test_skills_hub.py:test_do_list_marks_disabled_skills, ../hermes-agent/tests/hermes_cli/test_skills_hub.py:test_do_list_enabled_only_hides_disabled, internal/skills/store.go, internal/skills/list.go
- Unblocks: Skills hub search read-model function over registry providers
- Why now: Unblocks Skills hub search read-model function over registry providers.

## 8. CLI log redactor for known secret shapes

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: internal/cli/log_redact.go exposes RedactLine(line []byte) ([]byte, int) where the int is the number of redactions applied. Matches and replaces with "[REDACTED]": (1) Bearer XXX in any header line, (2) api_key=VALUE or x-api-key: VALUE, (3) Telegram bot tokens NN:XXXXXXXX (digits + colon + >=20 alnum/_/-), (4) Slack xoxb-/xoxp-/xoxs- tokens, (5) OpenAI sk-* keys longer than 16 chars. Returns input unchanged with count=0 if no match. Pure: only regexp + bytes packages from stdlib.
- Trust class: operator, system
- Ready when: internal/cli already compiles; this row adds a sibling log_redact.go + _test.go., Tests use fixed []byte literals — no file I/O.
- Not ready when: The slice reads files, walks XDG paths, or uploads anywhere., The slice adds new secret shapes beyond the five listed.
- Degraded mode: Redactor counts replacements per line so the snapshot caller can attach a per-section Redacted field without re-scanning.
- Fixture: `internal/cli/log_redact_test.go`
- Write scope: `internal/cli/log_redact.go`, `internal/cli/log_redact_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -run 'TestRedactLine' -count=1`, `go test ./internal/cli -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/cli/log_redact_test.go fixtures prove redaction across the five secret shapes plus no-match preservation.
- Acceptance: TestRedactLine_BearerToken returns redacted line and count=1 for "Authorization: Bearer abc123def456"., TestRedactLine_ApiKeyEqualsValue covers "api_key=sk-prod-XYZ" and "x-api-key: sk-test-..."., TestRedactLine_TelegramBotToken redacts "12345:ABCDEFGHabcdefgh1234567890"., TestRedactLine_SlackTokens covers xoxb-, xoxp-, xoxs- tokens., TestRedactLine_OpenAIStyleKey redacts sk-* longer than 16 chars only., TestRedactLine_NoMatchPreservesInput returns input unchanged with count=0.
- Source refs: ../hermes-agent/hermes_cli/logs.py
- Unblocks: CLI log snapshot reader using shared redactor
- Why now: Unblocks CLI log snapshot reader using shared redactor.

## 9. BlueBubbles iMessage bubble formatting parity

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes
- Trust class: gateway, system
- Ready when: The first-pass BlueBubbles adapter already owns Send, markdown stripping, cached GUID resolution, and home-channel fallback in internal/channels/bluebubbles.
- Not ready when: The slice attempts to add live BlueBubbles HTTP/webhook registration, attachment download, reactions, typing indicators, or edit-message support.
- Degraded mode: BlueBubbles remains a usable first-pass adapter, but long replies may still arrive as one stripped text send until paragraph splitting and suffix-free chunking are fixture-locked.
- Fixture: `internal/channels/bluebubbles/bot_test.go`
- Write scope: `internal/channels/bluebubbles/bot.go`, `internal/channels/bluebubbles/bot_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/bluebubbles -count=1`
- Done signal: BlueBubbles adapter tests prove paragraph-to-bubble sends, suffix-free chunking, and no edit/placeholder capability.
- Acceptance: Send splits blank-line-separated paragraphs into separate SendText calls while preserving existing chat GUID resolution and home-channel fallback., Long paragraph chunks omit `(n/m)` pagination suffixes and concatenate back to the stripped original text., Bot does not implement gateway.MessageEditor or gateway.PlaceholderCapable, preserving non-editable iMessage semantics.
- Source refs: ../hermes-agent/gateway/platforms/bluebubbles.py@f731c2c2, ../hermes-agent/tests/gateway/test_bluebubbles.py@f731c2c2, internal/channels/bluebubbles/bot.go, internal/gateway/channel.go
- Unblocks: BlueBubbles iMessage session-context prompt guidance
- Why now: Unblocks BlueBubbles iMessage session-context prompt guidance.

## 10. CLI profile name validator

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: internal/cli adds a pure function `ValidateProfileName(name string) error` and an exported sentinel error set: ErrProfileNameEmpty, ErrProfileNameTooLong, ErrProfileNameInvalidChars, ErrProfileNameReserved; the function accepts names matching `^[a-z0-9][a-z0-9_-]{0,63}$`, treats 'default' as valid (special alias), and rejects the reserved subcommand names {'create','delete','list','use','export','import','show'}
- Trust class: operator, system
- Ready when: internal/cli already exposes pure helpers; adding one new file with one validator + sentinel errors compiles cleanly alongside them., This slice only defines validation; no path resolution, active-profile read/write, command wiring, alias wrapper, or env mutation is required.
- Not ready when: The slice resolves filesystem paths, creates wrapper scripts, mutates provider credentials, modifies internal/config, or registers a Cobra command., The slice modifies any other internal/cli file beyond the new profile_name.go and profile_name_test.go.
- Degraded mode: Callers report a typed sentinel error class instead of free-form text so the CLI can render uniform error messages later without re-parsing strings.
- Fixture: `internal/cli/profile_name_test.go`
- Write scope: `internal/cli/profile_name.go`, `internal/cli/profile_name_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -run 'TestValidateProfileName_' -count=1`, `go test ./internal/cli -count=1`, `go vet ./internal/cli`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/cli/profile_name.go declares ValidateProfileName plus the four sentinel errors; five named tests pass; no other internal/cli, internal/config, or cmd/gormes file is modified.
- Acceptance: TestValidateProfileName_AcceptsValid: ValidateProfileName each of {'default','coder','work-1','tier_2','a','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'} returns nil (the last is exactly 64 chars)., TestValidateProfileName_RejectsEmpty: ValidateProfileName('') and ValidateProfileName('   ') (after caller-side trim) both return ErrProfileNameEmpty., TestValidateProfileName_RejectsTooLong: ValidateProfileName(strings.Repeat('a', 65)) returns ErrProfileNameTooLong., TestValidateProfileName_RejectsInvalidChars: each of {'Coder','my profile','-leading','_leading','dot.name','slash/name','tab\tname'} returns ErrProfileNameInvalidChars., TestValidateProfileName_RejectsReserved: each of {'create','delete','list','use','export','import','show'} returns ErrProfileNameReserved (these collide with subcommand names).
- Source refs: ../hermes-agent/hermes_cli/profiles.py@edc78e25:_PROFILE_ID_RE, ../hermes-agent/hermes_cli/profiles.py@edc78e25:validate_profile_name, ../hermes-agent/tests/hermes_cli/test_profiles.py@edc78e25, internal/cli/banner.go
- Unblocks: CLI active-profile store, CLI profile root resolver
- Why now: Unblocks CLI active-profile store, CLI profile root resolver.

<!-- PROGRESS:END -->
