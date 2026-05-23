# TODO.md — Gormes operational blockers

No active operational blockers.

## Resolved blocker receipts

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
