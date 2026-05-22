# TODO.md — Gormes operational blockers

[BLOCKED] Publish follow-up release for Termux latest installer — 2026-05-21 20:14:15 CST
  blocker: live public `v0.2.20` remains the latest release and affected Termux/Android installs can still fail publish verification, then report `unknown command /data/data/com.termux/files/usr/bin/gormes for gormes`; the recovery fix is committed on `development` but not released.
  evidence: `gh release view v0.2.20` returned published release https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.2.20; `refs/tags/v0.2.20` is `c27835f25d32`; Termux recovery fix commit `72b4ee248475` is on `origin/development`; `git ls-remote --tags origin refs/tags/v0.2.21` returned zero lines.
  unblocks when: a `development` -> `main` PR is merged through the required release lane and a follow-up tag/release, expected `v0.2.21` unless superseded, publishes fixed Termux installer artifacts.
  owner: Gormes release lane / operator with release approval.
  workaround/pivot: keep public docs explicit that `v0.2.20` Termux latest install is affected; use the committed `development` fix only for source/local validation until release is approved.
  next check: next release-lane iteration or explicit publish request.

[BLOCKED] Gormes Telegram provider-native client gap — 2026-04-29 06:56:45 CST
  blocker: `@gormes_bot` can receive Telegram messages, but Gormes-native model execution has no production native provider client/endpoint for `openai-codex`; after removing the silent localhost default, local probe fails with `Post "/v1/responses": unsupported protocol scheme ""`.
  evidence: screenshot showed `Post "http://127.0.0.1:8642/v1/responses": dial tcp 127.0.0.1:8642: connect: connection refused` and repeated `admission: still processing previous turn`; source-run state after restart shows `gateway_state=running`, Telegram platform `running`, `active_agents=0`; commit `aa91fcff6` removed implicit localhost default and full `go test ./... -count=1` passed.
  unblocks when: Gormes has a production native provider client factory for `openai-codex` or an explicit operator-configured endpoint/proxy with valid credentials, without depending on Python Hermes runtime.
  owner: Gormes runtime/provider implementation
  workaround/pivot: keep gateway source-run up for ingress/status checks, but expect model turns to fail until provider-native execution lands; next code slice should wire provider-native client creation or produce structured `native_runtime_unavailable` channel replies instead of raw transport errors.
  next check: after provider-native client factory or explicit endpoint/proxy configuration lands
