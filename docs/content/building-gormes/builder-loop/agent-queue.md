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
## 1. Watchdog dead-process vs slow-progress separation

- Phase: 1 / 1.C
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: TDD packet for a missing pure helper; create exactly two files and do not wire them into the watchdog loop. STEP 1: cd into the repo root and run `ls internal/builderloop/watchdog_state*.go` — both files must be absent before the worker writes them. STEP 2: write internal/builderloop/watchdog_state_test.go in package builderloop with one TestDiagnose function holding a t.Run-driven table. The subtests are named exactly zero_pid_dead, pid_not_live_after_dead_threshold, live_after_slow_threshold, dead_wins_when_both_thresholds_fire, healthy_recent_live, and pid_live_silent_for_a_year_is_slow_not_dead. Use a fixed time anchor `now := time.Date(2026,4,26,18,0,0,0,time.UTC)` and pass deadAfter=120*time.Second, slowAfter=600*time.Second. STEP 3: write internal/builderloop/watchdog_state.go exposing type Verdict string with constants VerdictHealthy="healthy", VerdictSlow="slow", VerdictDead="dead"; type WorkerVitals struct{PID int; LastCommitAt time.Time; PIDIsLive bool}; and func Diagnose(now time.Time, v WorkerVitals, deadAfter, slowAfter time.Duration) Verdict. Verdict precedence (evaluate in this order): if v.PID==0 return VerdictDead; let elapsed := now.Sub(v.LastCommitAt); if !v.PIDIsLive && elapsed>=deadAfter return VerdictDead; if v.PIDIsLive && elapsed>=slowAfter return VerdictSlow; otherwise return VerdictHealthy. STEP 4: the helper uses only caller-injected now and PIDIsLive — no os.FindProcess, signal delivery, time.Now, goroutines, watchdog wiring, config validation, or imports beyond `time`.
- Trust class: operator, system
- Ready when: Existing watchdog timer (commit f96a5d94) emits stall events at a single threshold; this slice carves the threshold into two independent ones., Watchdog checkpoint coalescing is fixture-ready or validated, so the dead-process tick does not amplify the commit storm., Both target files are absent on main; the worker's first edit is the focused failing table test, then the helper., If either target file already exists in the worker checkout, the worker should run the focused test and update this progress row instead of creating a duplicate helper.
- Not ready when: The slice changes how worker output is rejected or how dirty worktrees are committed — only worker liveness detection is in scope., The slice introduces process-group signal sending or container-aware death detection (those belong to a separate sandboxing row)., The worker needs to edit internal/builderloop/run.go, watchdog timers, backend prompts, ledger writing, or process-kill behavior to make the test pass.
- Degraded mode: Watchdog status reports worker_state ∈ {alive_progressing, alive_silent, dead, unknown} and the threshold each one tripped; record_run_health carries the worker_state and which threshold (dead_after_seconds, silent_after_seconds) fired.
- Fixture: `internal/builderloop/watchdog_state_test.go (new file)::TestDiagnose/zero_pid_dead+dead_wins_when_both_thresholds_fire`
- Write scope: `internal/builderloop/watchdog_state.go`, `internal/builderloop/watchdog_state_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/builderloop -run '^TestDiagnose$' -count=1`, `go test ./internal/builderloop -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/builderloop/watchdog_state_test.go fixtures prove healthy/slow/dead verdicts including the dead-vs-slow precedence rule with no os.FindProcess or signal calls., internal/builderloop/watchdog_state.go contains exactly the new Verdict, WorkerVitals, and Diagnose helper API; no existing builderloop runtime file changes are required.
- Acceptance: TestDiagnose/healthy_recent_live: v={PID:1234, LastCommitAt:now-5*time.Second, PIDIsLive:true}, deadAfter=120s, slowAfter=600s returns VerdictHealthy., TestDiagnose/live_after_slow_threshold: v={PID:1234, LastCommitAt:now-700*time.Second, PIDIsLive:true} returns VerdictSlow., TestDiagnose/pid_not_live_after_dead_threshold: v={PID:1234, LastCommitAt:now-200*time.Second, PIDIsLive:false} returns VerdictDead., TestDiagnose/zero_pid_dead: v={PID:0, LastCommitAt:now-1*time.Second, PIDIsLive:true} returns VerdictDead (zero PID short-circuits; thresholds and PIDIsLive are ignored)., TestDiagnose/dead_wins_when_both_thresholds_fire: v={PID:1234, LastCommitAt:now-700*time.Second, PIDIsLive:false} with deadAfter=120s, slowAfter=600s returns VerdictDead (dead wins over slow when both fire)., TestDiagnose/pid_live_silent_for_a_year_is_slow_not_dead: v={PID:1234, LastCommitAt:now-99999*time.Second, PIDIsLive:true} returns VerdictSlow (never VerdictDead while the process answers Signal(0))., Helper is pure — caller injects the clock (now) and the PIDIsLive result; the test file imports only `testing` and `time` from stdlib.
- Source refs: internal/builderloop/run.go, internal/builderloop/run_health_test.go
- Unblocks: Builder-loop self-improvement vs user-feature ratio metric
- Why now: Unblocks Builder-loop self-improvement vs user-feature ratio metric.

## 2. Azure Foundry probe — path sniffing

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: TDD packet for a missing pure helper; create exactly two files and do not edit the already validated /models probe. First add internal/hermes/azure_foundry_path_sniff_test.go in package hermes with TestClassifyAzurePath table subtests named exact_anthropic, trailing_anthropic, nested_anthropic, mixed_case, bare_host_unknown, openai_path_unknown, substring_false_positives, and empty_and_malformed_unknown. Then add internal/hermes/azure_foundry_path_sniff.go exposing ClassifyAzurePath(rawURL string) AzureTransport that reuses the AzureTransport type/constants from internal/hermes/azure_foundry_models_probe.go. Algorithm: url.Parse; empty path or parse error => AzureTransportUnknown; lowercase and strings.TrimRight(parsed.Path, "/"); return AzureTransportAnthropic only when the path equals "/anthropic", ends with "/anthropic", or contains "/anthropic/". This slice never returns AzureTransportOpenAI and never opens HTTP, reads env/config, writes files, starts goroutines, or changes ProbeAzureFoundry.
- Trust class: operator, system
- Ready when: internal/hermes/azure_foundry_models_probe.go is complete and owns the AzureTransport type/constant values; this row only adds a path-sniff helper file plus a sibling _test.go., No upstream gating: this is a pure URL inspector with synthetic input., internal/hermes/azure_foundry_path_sniff.go and azure_foundry_path_sniff_test.go are absent on main; the worker's first edit is the focused failing test file., If ClassifyAzurePath already exists in the worker checkout, the worker should run the focused test and update this progress row instead of adding a second detector.
- Not ready when: The slice opens HTTP connections, performs a /models probe, reads AZURE_FOUNDRY_BASE_URL or AZURE_FOUNDRY_API_KEY, or mutates config., The slice introduces detection of any third transport family (Bedrock, Vertex, etc.)., The worker needs to edit internal/hermes/azure_foundry_models_probe.go or any provider request builder to make the path-sniff tests pass.
- Degraded mode: Probe status reports azure_path_sniff_unknown when no path heuristic matches, and azure_path_sniff_evidence with detected scheme/host/path otherwise.
- Fixture: `internal/hermes/azure_foundry_path_sniff_test.go (new file)::TestClassifyAzurePath/substr_false_positive`
- Write scope: `internal/hermes/azure_foundry_path_sniff.go`, `internal/hermes/azure_foundry_path_sniff_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run '^TestClassifyAzurePath$' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/hermes/azure_foundry_path_sniff_test.go fixtures prove anthropic-path classification across suffix, mid-path, case variants, bare hosts, malformed URLs, and false-positive substrings without HTTP., internal/hermes/azure_foundry_models_probe.go remains untouched; ClassifyAzurePath only imports standard URL/string helpers and returns existing AzureTransport constants.
- Acceptance: TestClassifyAzurePath_AnthropicSuffix: https://x.openai.azure.com/openai/deployments/y/anthropic returns AzureTransportAnthropic., TestClassifyAzurePath_AnthropicMidPath: https://x/openai/anthropic/v1/messages returns AzureTransportAnthropic., TestClassifyAzurePath_CaseInsensitive: /AnthrOPic and /ANTHROPIC both return AzureTransportAnthropic., TestClassifyAzurePath_OpenAIDefault: https://x.openai.azure.com/openai/v1/chat/completions returns AzureTransportUnknown (never AzureTransportOpenAI in this slice)., TestClassifyAzurePath_MalformedReturnsUnknown: empty string, "::garbage::", and rawURL="http://%zz" return AzureTransportUnknown., TestClassifyAzurePath_NotASubstringFalsePositive: /anthropicx and /anthropic-mirror return AzureTransportUnknown (substring guard requires "/anthropic" with a trailing path separator or end-of-path).
- Source refs: ../hermes-agent/hermes_cli/azure_detect.py@9be83728:_looks_like_anthropic_path, internal/hermes/azure_foundry_models_probe.go:AzureTransport
- Unblocks: Azure Foundry transport probe read model
- Why now: Unblocks Azure Foundry transport probe read model.

## 3. Provider rate guard — x-ratelimit header classification

- Phase: 4 / 4.H
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: TDD packet for a missing pure helper that matches Hermes 192e7eb2 reset-window semantics; create exactly two files and do not wire a provider breaker. STEP 1: cd into the repo root and run `ls internal/hermes/provider_rate_guard*.go` — both files must be absent before the worker writes them; if Classify429 already exists, the worker should run the focused test and update this row. STEP 2: write internal/hermes/provider_rate_guard_classification_test.go in package hermes with one TestClassify429 function holding a t.Run-driven table with named subtests genuine_quota_1h_reset, short_reset_upstream_capacity, healthy_remaining_upstream_capacity, missing_headers_insufficient, unknown_headers_ignored, malformed_values_ignored, and three_buckets_with_remaining_one_missing_returns_upstream_capacity. Each subtest builds an http.Header via h := http.Header{}; h.Set(...); h.Set(...). STEP 3: write internal/hermes/provider_rate_guard.go exposing type RateLimitClass string with constants RateLimitGenuineQuota="genuine_quota", RateLimitUpstreamCapacity="upstream_capacity", RateLimitInsufficientEvidence="insufficient_evidence" and func Classify429(headers http.Header) RateLimitClass. Algorithm: iterate the four Hermes Nous bucket tags ("requests", "requests-1h", "tokens", "tokens-1h") via the paired x-ratelimit-remaining-{tag} and x-ratelimit-reset-{tag} headers; use headers.Get, strings.TrimSpace, strconv.Atoi for remaining, strconv.ParseFloat for reset seconds. A bucket is exhausted only if remaining<=0 AND reset>=60 seconds (matches Hermes _MIN_RESET_FOR_BREAKER_SECONDS). If any bucket is exhausted, return RateLimitGenuineQuota. If at least one bucket parsed successfully and none are exhausted, return RateLimitUpstreamCapacity. Otherwise return RateLimitInsufficientEvidence. STEP 4: no shared breaker state, no time.Now, no sleeps, no retry policy, and no provider routing changes; imports are limited to net/http, strconv, and strings (plus testing in the test file).
- Trust class: system
- Ready when: internal/hermes already compiles; the row creates a new file and a sibling _test.go., No upstream gate; pure header parsing with synthetic http.Header values., internal/hermes/provider_rate_guard.go and provider_rate_guard_classification_test.go are absent on main; the worker's first edit is the focused failing test file., If Classify429 or RateLimitClass already exists in the worker checkout, the worker should run the focused test and update this row instead of creating duplicate constants.
- Not ready when: The slice changes retry timing, provider routing, or model fallback policy., The slice writes process-global breaker state in unit tests or sleeps to simulate reset windows., The worker needs to edit internal/hermes/client.go, internal/hermes/errors.go, internal/kernel/retry.go, or any provider routing code to make the tests pass.
- Degraded mode: Provider status reports rate_guard_classified as one of {genuine_quota, upstream_capacity, insufficient_evidence}; reset-window evidence is parsed for the decision but detailed budget telemetry waits for the dependent row.
- Fixture: `internal/hermes/provider_rate_guard_classification_test.go (new file)::TestClassify429/genuine_quota_1h_reset`
- Write scope: `internal/hermes/provider_rate_guard.go`, `internal/hermes/provider_rate_guard_classification_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run '^TestClassify429$' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/hermes/provider_rate_guard_classification_test.go fixtures prove genuine_quota / upstream_capacity / insufficient_evidence classification from the four current Hermes Nous bucket tags, including the >=60s reset-window guard., internal/hermes/provider_rate_guard.go contains only RateLimitClass constants plus Classify429; no provider client, retry, or routing file changes are required.
- Acceptance: TestClassify429/genuine_quota_1h_reset (X-RateLimit-Remaining-Requests-1h=0 and X-RateLimit-Reset-Requests-1h=300) returns RateLimitGenuineQuota even when other present buckets are >0., TestClassify429/short_reset_upstream_capacity (remaining=0 and reset=30) returns RateLimitUpstreamCapacity, mirroring Hermes' <60s transient-throttle rule., TestClassify429/healthy_remaining_upstream_capacity covers any subset of the four buckets with remaining>0 (and none missing-and-parsed-as-zero) returning RateLimitUpstreamCapacity., TestClassify429/missing_headers_insufficient with no x-ratelimit-* headers returns RateLimitInsufficientEvidence and the returned string is non-empty., TestClassify429/unknown_headers_ignored (Retry-After, X-Custom-Foo) preserves the classification driven solely by the four x-ratelimit-remaining-* buckets., TestClassify429/malformed_values_ignored (X-RateLimit-Remaining-Tokens="abc") with no other rate headers returns RateLimitInsufficientEvidence rather than treating the malformed value as zero., TestClassify429/three_buckets_with_remaining_one_missing_returns_upstream_capacity proves a partial header set without any exhausted bucket still classifies as RateLimitUpstreamCapacity (not InsufficientEvidence).
- Source refs: ../hermes-agent/agent/nous_rate_guard.py@192e7eb2:_MIN_RESET_FOR_BREAKER_SECONDS, ../hermes-agent/agent/nous_rate_guard.py@192e7eb2:_parse_buckets_from_headers, ../hermes-agent/agent/nous_rate_guard.py@192e7eb2:_has_exhausted_bucket, ../hermes-agent/tests/agent/test_nous_rate_guard.py@192e7eb2:TestIsGenuineNousRateLimit
- Unblocks: Provider rate guard — degraded-state + last-known-good evidence
- Why now: Unblocks Provider rate guard — degraded-state + last-known-good evidence.

## 4. TUI prompt-submit auto-title eligibility helper

- Phase: 5 / 5.Q
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Pure helper internal/tui/auto_title.go exposes type AutoTitleInput struct{SessionKey string; FallbackSessionID string; Status string; UserText string; AssistantText string; Interrupted bool; HistoryCount int}, type AutoTitleRequest struct{SessionID string; UserText string; AssistantText string; HistoryCount int}, and BuildAutoTitleRequest(in AutoTitleInput) (AutoTitleRequest, bool). It returns ok=true only when Status=="complete", Interrupted is false, strings.TrimSpace(UserText) and strings.TrimSpace(AssistantText) are non-empty, and the resolved session ID is non-empty. Resolution prefers strings.TrimSpace(SessionKey) and falls back to strings.TrimSpace(FallbackSessionID). The returned request preserves the original UserText/AssistantText bytes and HistoryCount. No title generation, provider call, DB write, goroutine, clock lookup, or TUI transport change in this slice.
- Trust class: operator, gateway, system
- Ready when: internal/tui already owns pure helper files and tests; this row adds one helper without touching Bubble Tea update flow., The worker can table-test eligibility with synthetic strings and history counts; no title model, database, or live TUI session is required.
- Not ready when: The slice calls an LLM/title generator, writes session metadata, starts goroutines, or changes kernel/TUI submit behavior., The slice ports Hermes Python session_key storage directly instead of adapting to Gormes SessionID metadata.
- Degraded mode: TUI/session status can report auto_title_skipped with reason interrupted, empty_prompt, empty_response, non_complete, or missing_session before a later row wires title generation.
- Fixture: `internal/tui/auto_title_test.go::TestBuildAutoTitleRequest`
- Write scope: `internal/tui/auto_title.go`, `internal/tui/auto_title_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run TestBuildAutoTitleRequest -count=1`, `go test ./internal/tui -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/tui/auto_title_test.go fixtures prove complete-turn eligibility, interrupted/empty/non-complete skips, session-key fallback, and zero title generation side effects.
- Acceptance: TestBuildAutoTitleRequest_CompletePromptReturnsRequest: status complete with session_key, non-empty user text, non-empty assistant text, and HistoryCount=2 returns ok=true and preserves the original texts., TestBuildAutoTitleRequest_FallbackSessionID: empty SessionKey with FallbackSessionID="sid" returns request.SessionID="sid"., TestBuildAutoTitleRequest_SkipsInterrupted: Interrupted=true returns ok=false even when texts are non-empty., TestBuildAutoTitleRequest_SkipsEmptyPromptOrResponse: whitespace-only UserText or AssistantText returns ok=false., TestBuildAutoTitleRequest_SkipsNonCompleteOrMissingSession: non-complete status or empty resolved session returns ok=false.
- Source refs: ../hermes-agent/tui_gateway/server.py@9662e321:prompt.submit, ../hermes-agent/tests/test_tui_gateway_server.py@9662e321:test_prompt_submit_auto_titles_session_on_complete, ../hermes-agent/tests/test_tui_gateway_server.py@9662e321:test_prompt_submit_skips_auto_title_when_interrupted, ../hermes-agent/tests/test_tui_gateway_server.py@9662e321:test_prompt_submit_skips_auto_title_when_response_empty, internal/tui/update.go, internal/session/directory.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. BlueBubbles iMessage bubble formatting parity

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

## 6. CLI profile name validator

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

## 7. doctorCustomEndpointReadiness check function

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: cmd/gormes adds a pure function `doctorCustomEndpointReadiness(cfg config.Config) doctor.CheckResult` that returns Name='Custom endpoint', Status=Pass when Hermes.Endpoint and Hermes.APIKey and Hermes.Model are all non-empty, Status=Warn when any one is missing (with itemized evidence), and Status=Fail when Endpoint is set but Model is empty; doctorCmd RunE invokes this function after the existing Goncho/Slack checks; --offline still skips network probes elsewhere
- Trust class: operator, system
- Ready when: cmd/gormes/doctor.go already calls doctorGonchoConfig(cfg) and doctorSlackGatewayConfig(cfg, runtimeStatus) — adding a third helper alongside them is mechanical., internal/config/config.go declares HermesCfg{Endpoint, APIKey, Model} so the check has a stable typed input., internal/doctor/doctor.go already exposes CheckResult, ItemInfo, StatusPass/StatusWarn/StatusFail; this row only composes them.
- Not ready when: The slice changes config schema, adds new HermesCfg fields, modifies provider routing, or introduces a live /v1/models or auth lookup., The slice changes any other doctor check's behaviour., The slice ports Hermes Python config.yaml reading.
- Degraded mode: When endpoint is set but credentials or model are missing, the check emits Status=Warn with item-level notes (api_key=missing, model=missing) instead of exiting non-zero, so operators see precisely which field needs attention.
- Fixture: `cmd/gormes/doctor_custom_provider_test.go`
- Write scope: `cmd/gormes/doctor.go`, `cmd/gormes/doctor_custom_provider_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run 'TestDoctorCustomEndpoint\|TestDoctorCmdInvokesCustomEndpointReadiness' -count=1`, `go test ./cmd/gormes -count=1`, `go vet ./cmd/gormes`, `go run ./cmd/builder-loop progress validate`
- Done signal: doctorCustomEndpointReadiness is a pure function with five named tests; doctorCmd invokes it; no internal/config or internal/hermes files are modified.
- Acceptance: TestDoctorCustomEndpointAllSet: cfg with Endpoint='https://example.invalid', APIKey='secret', Model='m' returns CheckResult{Name='Custom endpoint', Status=StatusPass, Summary contains 'configured'} and no items are flagged Warn., TestDoctorCustomEndpointMissingAPIKey: cfg with Endpoint set, APIKey empty, Model='m' returns Status=StatusWarn with an item Name='api_key' Status=StatusWarn Note='missing'., TestDoctorCustomEndpointMissingModel: cfg with Endpoint set, APIKey set, Model empty returns Status=StatusFail with an item Name='model' Status=StatusFail Note='missing' (Hermes considers this a hard error since requests cannot route)., TestDoctorCustomEndpointAllEmpty: cfg with all three empty returns Status=StatusWarn Summary='disabled' so doctor stays useful even when no endpoint is configured., TestDoctorCmdInvokesCustomEndpointReadiness: running doctorCmd.RunE against an in-memory cfg with custom endpoint emits the new check's Format() block to stdout in --offline mode and exits 0 when Status<=Warn.
- Source refs: ../hermes-agent/hermes_cli/doctor.py@b2d3308f:_run_doctor, ../hermes-agent/tests/hermes_cli/test_doctor.py@b2d3308f:test_run_doctor_accepts_bare_custom_provider, cmd/gormes/doctor.go, cmd/gormes/goncho_doctor_test.go, internal/config/config.go:HermesCfg, internal/doctor/doctor.go:CheckResult
- Unblocks: CLI status summary over native stores
- Why now: Unblocks CLI status summary over native stores.

## 8. Custom provider model-switch credential preservation

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/cli adds a pure function `ResolveCustomProviderSecret(ref CustomProviderRef, env map[string]string) (CustomProviderResolution, error)` where CustomProviderRef has fields {Name string, BaseURL string, APIKey string, KeyEnv string} and CustomProviderResolution has fields {EffectiveSecret string, PersistAsRef string, Evidence string}; the function reads env-template `${VAR}` from APIKey via env, prefers KeyEnv when APIKey is empty, and never returns plaintext in PersistAsRef when the input was a reference
- Trust class: operator, system
- Ready when: internal/cli already exposes pure helpers (banner.go, output.go, parity.go) so adding a single new file with one exported function compiles cleanly., This slice only defines a pure resolver over Go map/struct literals; no config reader, /model command handler, fake catalog server, or TUI dispatch is required.
- Not ready when: The slice ports model_switch.py wholesale, opens a fake /v1/models server, modifies internal/config or internal/hermes, or wires the resolver into command handlers in the same change., The slice returns plaintext in CustomProviderResolution.PersistAsRef when the input APIKey was an env-template `${VAR}` reference or KeyEnv was set.
- Degraded mode: Resolution returns Evidence='credential_missing', 'secret_ref_preserved', 'plaintext_provided', or 'env_var_unset' so callers can distinguish persistable references from resolved secrets without writing plaintext to config.
- Fixture: `internal/cli/custom_provider_secret_test.go`
- Write scope: `internal/cli/custom_provider_secret.go`, `internal/cli/custom_provider_secret_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -run 'TestResolveCustomProviderSecret_' -count=1`, `go test ./internal/cli -count=1`, `go vet ./internal/cli`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/cli/custom_provider_secret.go declares ResolveCustomProviderSecret, CustomProviderRef, CustomProviderResolution, ErrCustomProviderEnvUnset, and ErrCustomProviderCredentialMissing; five named tests pass; no internal/config, internal/hermes, or cmd/gormes file is modified.
- Acceptance: TestResolveCustomProviderSecret_EnvTemplatePreserved: ref={Name:'acme',APIKey:'${ACME_KEY}'}, env={'ACME_KEY':'sk-real'} returns {EffectiveSecret:'sk-real', PersistAsRef:'${ACME_KEY}', Evidence:'secret_ref_preserved'}., TestResolveCustomProviderSecret_KeyEnvFallback: ref={Name:'acme',APIKey:'',KeyEnv:'ACME_KEY'}, env={'ACME_KEY':'sk-real'} returns {EffectiveSecret:'sk-real', PersistAsRef:'${ACME_KEY}', Evidence:'secret_ref_preserved'}., TestResolveCustomProviderSecret_PlaintextProvided: ref={Name:'acme',APIKey:'sk-plain'}, env={} returns {EffectiveSecret:'sk-plain', PersistAsRef:'sk-plain', Evidence:'plaintext_provided'} (the function does not invent a reference)., TestResolveCustomProviderSecret_EnvVarUnset: ref={Name:'acme',APIKey:'${ACME_KEY}'}, env={} returns {EffectiveSecret:'', PersistAsRef:'${ACME_KEY}', Evidence:'env_var_unset'} and a non-nil error of class ErrCustomProviderEnvUnset., TestResolveCustomProviderSecret_BothEmpty: ref={Name:'acme',APIKey:'',KeyEnv:''}, env={} returns {EffectiveSecret:'', PersistAsRef:'', Evidence:'credential_missing'} and a non-nil error of class ErrCustomProviderCredentialMissing.
- Source refs: ../hermes-agent/hermes_cli/main.py@1fdc31b2:_custom_provider_api_key_config_value, ../hermes-agent/hermes_cli/main.py@8bbeaea6:_named_custom_provider_map, ../hermes-agent/tests/hermes_cli/test_custom_provider_model_switch.py@8bbeaea6, internal/cli/banner.go, internal/cli/output.go
- Unblocks: CLI command registry parity + active-turn busy policy
- Why now: Unblocks CLI command registry parity + active-turn busy policy.

## 9. [IMPORTANT:] prompt prefix for cron and skill commands

- Phase: 5 / 5.F
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Contract: internal/cron.CronHeartbeatPrefix and internal/skills.BuildSkillSlashCommandMessage emit `[IMPORTANT:` instead of `[SYSTEM:` so Azure OpenAI Default/DefaultV2 content filters do not reject Gormes prompts as prompt-injection (HTTP 400) — same semantic meta-instruction, different bracketed marker; tests update in lockstep so the byte-match assertions still cover drift
- Trust class: operator, system
- Ready when: Upstream Hermes shipped this rename across two commits (d7a34682 + 20cb706e) on 2026-04-09 / 2026-04-26 with explicit cause (Azure content filter HTTP 400 on `[SYSTEM:` markers)., Gormes uses the same marker pattern in exactly two production code paths today: internal/cron/heartbeat.go (CronHeartbeatPrefix) and internal/skills/commands.go (BuildSkillSlashCommandMessage).
- Not ready when: The slice changes the `[SILENT]` token semantics, the skill body trimming, the cron prompt structure beyond the bracketed marker word, or the `Heartbeat [SYSTEM:] + [SILENT] delivery contract` row name in 2.D (that row name is a historical record; only the runtime constant + the byte-match tests change)., The slice introduces a new Azure provider adapter, a content-filter-detection layer, or a configurable marker word — the change is a hardcoded literal rename only., The slice updates internal/progress/progress_test.go literal assertions for the 2.D row name (the row name in progress.json must stay as `Heartbeat [SYSTEM:] + [SILENT] delivery contract` for historical accuracy).
- Degraded mode: Operator-visible prompt text changes from `[SYSTEM: ...]` to `[IMPORTANT: ...]`; behavior is otherwise identical, including the `[SILENT]` suppression contract and the skill body trimming.
- Fixture: `internal/cron/heartbeat_test.go`
- Write scope: `internal/cron/heartbeat.go`, `internal/cron/heartbeat_test.go`, `internal/skills/commands.go`, `internal/skills/preprocessing_commands_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cron ./internal/skills -count=1`, `go test ./internal/progress -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: go test ./internal/cron and ./internal/skills both pass after the marker rename; `grep -rn '\[SYSTEM:' internal/cron/ internal/skills/` returns no matches in production code; `grep -rn '\[IMPORTANT:' internal/cron/ internal/skills/` returns at least 4 matches (constant + tests in both packages).
- Acceptance: internal/cron/heartbeat.go:CronHeartbeatPrefix starts with `[IMPORTANT:` (replacing `[SYSTEM:`) and the load-bearing phrases (`DELIVERY:`, `SILENT:`, `[SILENT]`) are byte-identical to the prior version., internal/cron/heartbeat_test.go asserts `strings.HasPrefix(full, "[IMPORTANT:")` (not `[SYSTEM:`); the existing TestHeartbeatPrefix_ContainsLoadBearingPhrases load-bearing phrase set updates only its first member., internal/skills/commands.go:BuildSkillSlashCommandMessage emits `[IMPORTANT: The user has invoked the "<name>" skill, ...` (replacing `[SYSTEM:`)., internal/skills/preprocessing_commands_test.go updates its expected golden string to `[IMPORTANT:` for the affected fixtures., DetectSilent semantics in internal/cron/heartbeat.go are unchanged (the `[SILENT]` token is independent of the leading marker).
- Source refs: ../hermes-agent/cron/scheduler.py@d7a34682, ../hermes-agent/agent/skill_commands.py@d7a34682, ../hermes-agent/cli.py@20cb706e, ../hermes-agent/gateway/run.py@20cb706e, ../hermes-agent/tools/process_registry.py@20cb706e, internal/cron/heartbeat.go:CronHeartbeatPrefix, internal/cron/heartbeat_test.go, internal/skills/commands.go:BuildSkillSlashCommandMessage, internal/skills/preprocessing_commands_test.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. TUI TerminalNativeSelectionHelp constant + help-string fixture

- Phase: 5 / 5.Q
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Contract: internal/tui declares an exported string constant TerminalNativeSelectionHelp = 'Selection: use your terminal's native selection (Shift-drag in most terminals; iTerm Cmd-drag, tmux copy-mode). Gormes does not advertise an in-app copy hotkey.' and a pure helper SelectionHelpLine() that returns it; one fixture asserts the constant exists, mentions 'terminal' but not 'Cmd+C'/'Ctrl+C'/'Ctrl-Shift-C'/'OSC 52'/'clipboard hotkey'/'Ink', and another asserts no advertised copy shortcut leaks anywhere else in the package
- Trust class: operator
- Ready when: internal/tui already exposes Bubble Tea model/view/update files and a mouse tracking config; adding a single new file with one constant compiles cleanly alongside them., phase-5-final-purge.md already documents the terminal-native selection divergence, so this row is mechanical: lift that statement into a typed Go constant and a regression test.
- Not ready when: The slice ports Hermes Ink, calls OSC 52, adds clipboard libraries, modifies internal/tui/update.go input handling, or changes remote TUI transport., The slice introduces a Cobra command flag for copy mode or a configuration key., The slice modifies cmd/gormes/ files.
- Degraded mode: If a future row adds a real Go-native copy mode, it must replace this constant rather than extend it; until then, the help-string fixture prevents accidental advertising of unimplemented Ink shortcuts.
- Fixture: `internal/tui/selection_help_test.go`
- Write scope: `internal/tui/selection_help.go`, `internal/tui/selection_help_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'TestTerminalNativeSelectionHelpExists\|TestTerminalNativeSelectionHelpNoFakeShortcuts\|TestTUIPackageDoesNotAdvertiseCopyHotkey' -count=1`, `go test ./internal/tui -count=1`, `go vet ./internal/tui`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/tui/selection_help.go declares TerminalNativeSelectionHelp and SelectionHelpLine; three named tests pass; no other internal/tui or cmd/gormes file is modified.
- Acceptance: TestTerminalNativeSelectionHelpExists: TerminalNativeSelectionHelp is a non-empty string constant exported from internal/tui, contains the substring 'terminal', and SelectionHelpLine() returns the same value., TestTerminalNativeSelectionHelpNoFakeShortcuts: TerminalNativeSelectionHelp does not contain any of: 'Cmd+C', 'Ctrl+C', 'Ctrl-Shift-C', 'Cmd-Shift-C', 'OSC 52', 'clipboard hotkey', 'Ink' (case-insensitive)., TestTUIPackageDoesNotAdvertiseCopyHotkey: walking internal/tui/*.go files, no string literal in the package contains the same forbidden shortcuts above (test reads the package source via os.ReadFile, not a runtime check)., go vet ./internal/tui passes; no other package is imported by the new file beyond stdlib.
- Source refs: ../hermes-agent/ui-tui/packages/hermes-ink/src/ink/selection.ts@edc78e25, ../hermes-agent/ui-tui/packages/hermes-ink/src/ink/selection.ts@31d7f195, internal/tui/view.go, internal/tui/model.go, internal/tui/mouse_tracking.go, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
