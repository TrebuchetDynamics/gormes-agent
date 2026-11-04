# TODO.md — Gormes operational blockers

## Planned

[PLANNED] Navivox interactive approval routing — Gormes Slice 1 — 2026-06-17
  problem: No production code injects `WithApprovalCallback` today; `SendApprovalRequired` is never called outside tests. Navivox turns are decoupled from tool execution (`handleTurn` → `enqueueTurn` drops a `gateway.InboundEvent{EventSubmit}` on the gateway inbox, and the gateway/agent runtime runs the turn elsewhere with no path back to the originating channel), so operator approve/deny cannot reach the waiting tool.
  evidence: `internal/tools/approval/callback/callback.go` implements a blocking `ApprovalCallback` that tools call and wait on; absence = noninteractive fail-closed. `internal/adapters/channels/navivox/channel.go` has `SendApprovalRequired` but it is never called in production. No channel (Telegram, Discord, Slack, or Navivox) wires interactive approval today. Full scope analysis: navivox-app TODO.md [PLANNED] Navivox approval response protocol 2026-06-17.
  acceptance: inject an `ApprovalCallback` into the gateway run path for channel-originated turns that routes back to the submitting channel; in the Navivox channel keep a pending-approval registry keyed by `approval_id`; have the callback emit `approval_required` and block awaiting the decision with a bounded timeout and ctx-cancel → fail-closed (deny); add HTTP resolve endpoint `POST /v1/navivox/approvals/{id}` (approve/deny), rejects stale/unknown IDs safely, never logs tool payloads/secrets; advertise the approve/deny endpoint in `/v1/navivox/capabilities`; tests for approve, deny, stale/missing ID, timeout fail-closed, and no-secret logging.
  owner: Gormes gateway owner / agent-runtime owner / Navivox app owner
  next check: design/brainstorm pass for gateway→channel interactive-approval routing before implementation (core-runtime + concurrency + safety scope).

## Resolved blocker receipts

[RESOLVED] Durable session commands, Gemini native transport, and Telegram reply fallback — resolved 2026-06-17
  result: /compress, /retry, and /undo are implemented as real gateway commands; Gemini providers route through a native transport; Telegram handles deleted reply targets gracefully; durable session history is SQL-backed.
  resolution evidence:
    - /compress: `handleCompressCommand` in `internal/gateway/command_dispatch.go` routes to `Kernel.ManualCompress` (new `PlatformEventManualCompress` in the idle-only run loop); `ErrCompressDuringTurn` and `ErrCompressionUnavailable` guard concurrent and unconfigured cases; `EventCompress` wired through `commandregistry`, `events/commands/kind.go`, and `events/kind.go`; carries body text as focus hint.
    - /retry + /undo: `handleRetryCommand` / `handleUndoCommand` in `command_dispatch.go` load durable history via `SessionHistoryStore`, rewind via `kernelSessionResumer.ResumeSession`, then resubmit; `sqlSessionHistoryStore` in `internal/gateway/session_history_store.go` backed by `transcript.LoadMessages` / `RewriteSessionHistory` / `RewindSessionHistory`; wired into `service.go` via `gateway.NewSQLSessionHistoryStore(mstore.DB())`.
    - Gemini native transport: `internal/llm/gemini_native.go` + `gemini_native_test.go`; `usesGeminiNativeTransport()` in `http_client.go` routes provider=gemini/google/google-gemini (non-OpenAI-compat URL) to native endpoint with `x-goog-api-key` auth.
    - Telegram reply-not-found fallback: `IsReplyNotFoundError` in `internal/adapters/channels/telegram/send/errors.go`; `bot.go` strips `reply_to_message_id` and retries on first occurrence; `send_text.go` does the same for direct sends; two new tests in `thread_fallback_test.go`.
    - Content dedup key: `turnContentDedupKey` in `Manager` prevents the same in-flight turn text from being enqueued as a duplicate follow-up.
    - Status polish: title field omitted from /status output when blank.
  validation: `go build ./...` clean; 3021 tests pass; 4 pre-existing failures (manifest coverage, repository source pairs) unchanged; `git diff --check` clean.

[RESOLVED] Full repo validation for setup-provider parity iteration — resolved 2026-05-23 19:09:23 CST
  original blocker: `go test ./... -count=1` failed twice in `github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin` while validating the setup-provider parity slice.
  original evidence: first run failed `TestAdminAgents_SpawnWizardCreatesRecord` with `WaitFor: condition not met after 5s`; targeted rerun of that test passed; retry failed `TestAdminAgents_BindWizardWritesBinding` with `WaitFor: condition not met after 5s`; targeted rerun of that test passed.
  resolution evidence: fresh `go test ./... -count=1` exited 0; `github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin` passed in 33.137s.
  validation: `go test ./... -count=1`; `go run ./cmd/progress validate`; `git diff --check`.


[RESOLVED] Publish follow-up release for Termux latest installer — resolved 2026-05-23 14:25:24 CST
  original blocker: live public `v0.2.20` was the latest release and affected Termux/Android installs could fail publish verification, then report `unknown command /data/data/com.termux/files/usr/bin/gormes for gormes`; the recovery fix was committed on `development` but not released.
  resolution evidence: public `v0.2.22` is now the latest GitHub release; PR #221 merged `development` to `main` at `b6afeae04fd57cc77aed1126cc3236e473ef5833`; Release Binaries run `26342488226` succeeded; `gh release view v0.2.22` showed `install.sh`, `install.ps1`, `gormes-0.2.22-android-arm64.tar.gz`, and its `.sha256`; the android-arm64 checksum matched; deployed `https://gormes.ai/install.sh` synthetic Termux aarch64 dry-run selected binary-fetch to `$PREFIX/bin/gormes` and skipped active PATH/service writes.
  validation: `gh pr checks 221 --json name,state,link,bucket`; `gh run list --branch main --limit 20 --json workflowName,headSha,status,conclusion,url`; `gh run view 26342488226 --json status,conclusion,jobs`; `tag=v0.2.22 version=0.2.22; gh release view ... | jq -e ...`; android-arm64 archive SHA-256 sidecar comparison; synthetic Termux `install.sh --dry-run` with `GORMES_INSTALL_TEST_UNAME_M=aarch64`.
  remaining live-smoke caveat: no physical Android/Termux device was used in this iteration; the release artifacts and installer plan are public and verified, but on-device execution remains optional external smoke evidence.

[RESOLVED] Gormes Telegram provider-native client gap — resolved 2026-05-23 10:24:05 CST
  original blocker: `@gormes_bot` could receive Telegram messages, but Gormes-native model execution lacked a production native provider client/endpoint for `openai-codex`; local probes previously failed with `Post "/v1/responses": unsupported protocol scheme ""` or attempted the old `127.0.0.1:8642` bridge.
  resolution evidence: progress rows now complete/validated for `Channel-neutral native runtime turn adapter`, `Codex Responses HTTP client binding`, and `Provider endpoint/API-key root flags + runtime resolution`; source refs show `internal/runtime/binding.go` resolves native-provider clients without implicit localhost, `internal/hermes` routes `openai-codex` through `/v1/responses`, and gateway/Telegram fixtures pass configured provider context.
  validation: `go test ./internal/runtime -run 'TestResolveNativeRuntimeBinding' -count=1`; `go test ./internal/hermes -run 'TestProviderTranscriptHarness_OpenAICodexUsesResponsesAPI|TestCodexProviderStatusReportsResponsesRuntime' -count=1`; `go test ./cmd/gormes -run 'TestBuildGatewayConfig|TestTelegram.*Provider|TestGateway.*Provider' -count=1`.
  remaining live-smoke caveat: this resolves the code/progress blocker only; live `@gormes_bot` provider smoke still depends on operator credentials, configured OAuth/provider state, and service deployment outside this repo.
