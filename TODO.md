# TODO.md — Gormes operational blockers

[BLOCKED] Publish follow-up release for Termux latest installer — 2026-05-21 20:14:15 CST
  blocker: live public `v0.2.20` remains the latest release and affected Termux/Android installs can still fail publish verification, then report `unknown command /data/data/com.termux/files/usr/bin/gormes for gormes`; the recovery fix is committed on `development` but not released.
  evidence: `gh release view v0.2.20` returned published release https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.2.20; `refs/tags/v0.2.20` is `c27835f25d32`; Termux recovery fix commit `72b4ee248475` is on `origin/development`; `git ls-remote --tags origin refs/tags/v0.2.21` returned zero lines.
  unblocks when: a `development` -> `main` PR is merged through the required release lane and a follow-up tag/release, expected `v0.2.21` unless superseded, publishes fixed Termux installer artifacts.
  owner: Gormes release lane / operator with release approval.
  workaround/pivot: keep public docs explicit that `v0.2.20` Termux latest install is affected; use the committed `development` fix only for source/local validation until release is approved.
  next check: next release-lane iteration or explicit publish request.

## Resolved blocker receipts

[RESOLVED] Gormes Telegram provider-native client gap — resolved 2026-05-23 10:24:05 CST
  original blocker: `@gormes_bot` could receive Telegram messages, but Gormes-native model execution lacked a production native provider client/endpoint for `openai-codex`; local probes previously failed with `Post "/v1/responses": unsupported protocol scheme ""` or attempted the old `127.0.0.1:8642` bridge.
  resolution evidence: progress rows now complete/validated for `Channel-neutral native runtime turn adapter`, `Codex Responses HTTP client binding`, and `Provider endpoint/API-key root flags + runtime resolution`; source refs show `internal/runtime/binding.go` resolves native-provider clients without implicit localhost, `internal/hermes` routes `openai-codex` through `/v1/responses`, and gateway/Telegram fixtures pass configured provider context.
  validation: `go test ./internal/runtime -run 'TestResolveNativeRuntimeBinding' -count=1`; `go test ./internal/hermes -run 'TestProviderTranscriptHarness_OpenAICodexUsesResponsesAPI|TestCodexProviderStatusReportsResponsesRuntime' -count=1`; `go test ./cmd/gormes -run 'TestBuildGatewayConfig|TestTelegram.*Provider|TestGateway.*Provider' -count=1`.
  remaining live-smoke caveat: this resolves the code/progress blocker only; live `@gormes_bot` provider smoke still depends on operator credentials, configured OAuth/provider state, and service deployment outside this repo.
