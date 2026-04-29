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

<!-- PROGRESS:START kind=agent-queue -->
## 1. Token vault

- Phase: 4 / 4.G
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Gormes owns an XDG-scoped, Hermes-compatible credential file vault that resolves declared relative credential paths without path traversal, missing-file leaks, or cross-profile bleed.
- Trust class: operator, system
- Ready when: The builder restates that Hermes is the parity contract only and no hermes-agent runtime services may be imported, launched, or called., The slice can use temp HOME, HERMES_HOME, XDG_CONFIG_HOME, and XDG_DATA_HOME fixtures only; no live profile, keychain, provider credential, or network access is required., The public seam is a small Go helper under internal/config that callers can later use for provider auth, skill required_credential_files, and remote terminal mounts.
- Not ready when: The slice reads Juan's live ~/.hermes credential files or prints any token/API key bytes in test output., The implementation accepts absolute paths, .. traversal, or symlink escapes outside the resolved Hermes profile home., The slice wires provider adapters, Google/Anthropic/Codex OAuth flows, remote terminal backends, or MCP transports instead of only creating the credential-vault helper.
- Degraded mode: Missing, absolute, traversal, symlink-escaped, or unreadable credential files return structured redacted evidence and do not mount or expose host paths outside the active profile.
- Fixture: `internal/config/token_vault_test.go`
- Write scope: `internal/config/token_vault.go`, `internal/config/token_vault_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/config -run '^TestTokenVault' -count=1`, `go test ./internal/config -count=1`, `go run ./cmd/progress validate`
- Done signal: Token vault fixtures prove safe relative credential resolution, unsafe-path rejection, session isolation, config credential-file loading, dedupe, clear semantics, redacted evidence, and no live credential access.
- Acceptance: TestTokenVaultRegisterCredentialFile accepts an existing relative file below a temp Hermes home and returns a mount with host_path inside that home and container_path under /root/.hermes., TestTokenVaultRejectsUnsafePaths proves absolute paths, .. traversal, and symlink escapes are rejected with redacted reason codes and no host_path in external evidence., TestTokenVaultSessionIsolation proves independent vault instances do not share registered files across sessions or profiles., TestTokenVaultConfigCredentialFiles loads terminal.credential_files from a temp Hermes config.yaml, skips missing files, rejects unsafe entries, and deduplicates skill-registered and config-declared mounts by container path., TestTokenVaultClear removes only the current vault's registered files and preserves deterministic config-derived mounts.
- Source refs: ../hermes-agent/tools/credential_files.py:register_credential_file,register_credential_files,get_credential_file_mounts,clear_credential_files, ../hermes-agent/tools/path_security.py:validate_within_dir, ../hermes-agent/hermes_constants.py:get_hermes_home, internal/config/config.go:hermesConfigPath,xdgConfigHome,xdgDataHome
- Why now: P0 handoff; needs contract proof before closeout.

<!-- PROGRESS:END -->
