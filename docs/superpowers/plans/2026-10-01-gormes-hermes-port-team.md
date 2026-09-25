# Gormes Hermes-Port Team Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. This plan configures Hermes operational state; it is not permission for live inference, worker dispatch, outbound messages, gateway restart, or release.

**Goal:** Provision a fresh native Hermes coordinator and eight responsibility-based Gormes specialists with the selected provider/model, a nonsecret team briefing, and verified root/board configuration where supported.

**Architecture:** Use nine flat sibling Hermes profiles: `gormes-agent-team` plus `kernel`, `providers`, `memory`, `tools`, `extensions`, `gateway`, `cli`, and `parity` specialists. Keep root human-facing; use the existing native Kanban/dispatcher only after explicit board and dispatcher ownership are verified. Defer Telegram credential setup until the exposed credential is revoked and a replacement is supplied through a secure supported prompt. Do not dispatch coding tasks under the repository's unresolved workspace-policy conflict.

**Tech Stack:** Installed Hermes Agent CLI v0.21.0 (upstream `24f5a60e`, local `5f05b724`), native profiles/configuration, native Kanban, Gormes Go repository.

**Spec:** `docs/superpowers/specs/2026-10-01-gormes-hermes-port-team-design.md`

## Global Constraints

- Target repository: `/home/xel/git/gormes/gormes-agent`.
- User explicitly authorizes development on `main` for this task; do not create or switch branches.
- Profiles are direct siblings beneath the profiles directory resolved by installed Hermes; never hard-code a home or nest profiles.
- Use provider `openai-codex` and model `gpt-6-luna` for every profile, with no overrides; do not silently substitute.
- Create fresh profiles only. Never clone profiles or copy credentials, memory, sessions, or auth state.
- Do not include any Telegram token in chat, plan, config outside the supported secret store, profile descriptions/SOUL, Kanban metadata, or verification output.
- No inference or paid/live model call, card creation, worker dispatch, gateway start/restart, Telegram send/handoff test, commit, push, PR, or release.
- No unattended coding task until independent ownership, bounded recovery, and a repository-safe workspace method are verified. The repo currently prohibits worktrees; a shared checkout is not an assumed safe parallel workspace.
- Preserve unrelated state. If an expected profile/name/path is occupied by an unrelated owner, stop before writing and ask for a collision decision.

## Review Focus

1. **Provider/model identity:** an unqualified model name or stale fallback may look configured but route through the wrong provider; verify the persisted provider/model pair on every profile.
2. **Telegram secret and account identity:** a bot token is not the human Telegram user ID; revoke the exposed token and obtain any needed numeric allowlist identity separately. Use a hidden secure prompt only.
3. **Shared-service blast radius:** a profile's board selection does not establish dispatcher ownership; do not edit host-wide dispatch/gateway settings or start a service.
4. **Profile creation side effects:** the installed CLI may seed skills and creates profile-local files; disable aliases and inspect creation behavior before writing, with no cloning.
5. **Workspace and acceptance:** Hermes scratch defaults and per-board concurrency do not provide safe integrated code work; create no cards or release workers until the repo workspace conflict and bounded recovery are resolved.

---

### Task 1: Reconfirm creation scope and safe preconditions

**Files:** Read only Hermes CLI source and the design/spec above; no profile state changes.

**Interfaces:**
- Consumes: accepted spec and provider/model choice.
- Produces: redacted preflight record for the exact IDs, resolved profiles root, installed CLI version, provider-qualified catalog evidence or a blocker, and any profile-name collisions.

- [x] **Step 1: Check native state and IDs**

Run `hermes profile list` and compare the exact nine proposed IDs. Resolve the real profiles root from the installed Hermes profile resolver; confirm that the root's existing `HERMES_HOME`/custom root rules do not redirect it. Do not inspect `.env`, `auth.json`, memory, or session files.

- [x] **Step 2: Verify provider-qualified model evidence without inference**

The inspected local `/home/xel/.hermes/models_dev_cache.json` is keyed by other providers and contains no `gpt-6-luna` entry; the generic profile-list output is not provider-qualified. Installed `hermes_cli/codex_models.py` defines an offline `openai-codex` fallback containing GPT-5.6 and older model IDs, but not `gpt-6-luna`; its source describes authenticated per-account catalog discovery as the path for current entitlement-specific models. Check the installed provider-specific catalog/cache for an actual `openai-codex` / `gpt-6-luna` entry. A match in another provider or curated fallback is not sufficient. Do not run authenticated catalog refresh, login, or inference without explicit authorization. If evidence is unavailable, stop before profile creation/configuration and report the exact required authorized catalog check; do not choose a replacement.

- [x] **Step 3: Resolve collisions before mutation**

If any exact profile ID already exists, inspect only its nonsecret owner/remit metadata. Do not overwrite, rename, or adopt it implicitly; ask the owner to resolve that collision. If all IDs are free, proceed.

**Validation:** Redacted names-only profile listing, Hermes version, resolved profile root source, and catalog evidence tied to `openai-codex`. No model calls.

### Task 2: Create the fresh flat profile set

**Files:** Nine new homes under the resolved Hermes profiles root; no Gormes source files.

**Interfaces:**
- Consumes: Task 1's collision-free IDs and resolved root.
- Produces: the coordinator and eight fresh specialist profile homes, with default model config and per-profile secrets files created by the installed CLI.

- [x] **Step 1: Recheck the installed creation command's side effects**

Use the already inspected `hermes profile create --help` and installed `hermes_cli/profiles.py` behavior. The reviewed CLI supports `--no-alias` and `--no-skills`; creation seeds a default SOUL, creates profile-local directories and a mode-0600 `.env`, and the CLI path may seed bundled skills. No clone flags are permitted. Confirm the installed command's skill-seeding path before running it; if that path downloads/installs unapproved content, stop and request a scoped decision rather than hiding the side effect.

- [x] **Step 2: Create each profile with no alias or profile clone**

After Task 1 passes, run the installed command once per row below. Its fresh-profile skill seeding reads the installed bundled-skill tree through `tools/skills_sync.py`; it does not clone another profile. Do not use `--clone`, `--clone-from`, or `--clone-all`. Allow only this bundled skill sync; do not invoke skill-hub installs or other dependency downloads.

```bash
hermes profile create gormes-agent-team --no-alias --description 'Human-facing Gormes Hermes-port coordinator'
hermes profile create gormes-agent-team-kernel --no-alias --description 'Gormes agent runtime and kernel parity specialist'
hermes profile create gormes-agent-team-providers --no-alias --description 'Gormes provider, auth, model routing, and usage parity specialist'
hermes profile create gormes-agent-team-memory --no-alias --description 'Gormes memory and Goncho/Honcho compatibility specialist'
hermes profile create gormes-agent-team-tools --no-alias --description 'Gormes tools, execution, and security parity specialist'
hermes profile create gormes-agent-team-extensions --no-alias --description 'Gormes skills, plugins, MCP, and ACP parity specialist'
hermes profile create gormes-agent-team-gateway --no-alias --description 'Gormes gateway and messaging channel parity specialist'
hermes profile create gormes-agent-team-cli --no-alias --description 'Gormes CLI, TUI, API, install, and packaging specialist'
hermes profile create gormes-agent-team-parity --no-alias --description 'Independent Hermes upstream parity research and review specialist'
```

- [x] **Step 3: Verify only owned new homes**

Run `hermes profile list`; verify all nine IDs and resolved direct-child homes. Confirm each has a fresh, non-cloned identity and no unrelated pre-existing files were replaced. If creation partially fails, stop: preserve successfully created homes and report exact completed/failed IDs; do not delete or silently retry.

**Validation:** Exact profile list and filesystem homes; no gateway started, no cards created, no inference.

### Task 3: Configure and verify the selected model per profile

**Files:** Only the nine new profile `config.yaml` files, edited through the installed profile-scoped Hermes config command.

**Interfaces:**
- Consumes: verified provider-qualified catalog entry and fresh profile homes.
- Produces: `model.provider: openai-codex` and `model.default: gpt-6-luna` on every profile, no overrides.

- [x] **Step 1: Verify config command semantics**

Use the installed `hermes config set --help` plus current source to confirm profile-scoped invocation and scalar/YAML serialization for `model.provider` and `model.default`. Do not use arbitrary shell `sed` edits or edit any shared/default profile configuration.

- [x] **Step 2: Set both fields explicitly for all nine profiles**

For each profile ID, run these supported profile-scoped commands:

```bash
hermes -p PROFILE config set model.provider openai-codex
hermes -p PROFILE config set model.default gpt-6-luna
```

Replace `PROFILE` with each exact ID from Task 2. Do not use interactive `hermes model` selection or alter any shared/default profile configuration.

- [x] **Step 3: Read back the effective pair**

For each profile, run:

```bash
hermes -p PROFILE config get model.provider
hermes -p PROFILE config get model.default
```

Confirm all nine profiles match the selected pair and that no session/task override or provider base URL is being asserted as tested. Stop on any mismatch; never silently downgrade.

**Validation:** Nine-row redacted provider/model table. Inference remains `not run`; authentication is not inferred from config.

### Task 4: Set role instructions and create the durable team briefing

**Files:** `SOUL.md` in each newly created profile home and `<coordinator-home>/TEAM.md`; do not touch project `AGENTS.md`, other Hermes homes, or credentials.

**Interfaces:**
- Consumes: verified profile homes, model pairs, role table, design spec.
- Produces: minimal role-specific identity files and one nonsecret briefing with an explicit revision.

- [x] **Step 1: Draft the coordinator and specialist remits**

Use the roles in the spec. Root owns human communication, planning, and acceptance evidence, but never implements its own dispatched task or reviews its own candidate. Specialists use Gormes' canonical progress workflow and must return exact changed-file/candidate/test evidence. `parity` may independently review work it did not author; assign an actual different profile for any parity-authored candidate. Integration ownership is task-specific, not a permanent role.

- [x] **Step 2: Write a single briefing and root pointer**

Create `<coordinator-home>/TEAM.md` containing repo path, revision, exact IDs/homes, role coverage, provider/model, board/dispatcher marked unresolved until verified, root interface status, repo workspace conflict, acceptance/recovery gates, auth/inference status, and approval boundaries. Never include credentials or private transcripts. Put the root SOUL pointer/instruction to read and reload this briefing at session start and revision changes. Each specialist SOUL should describe only its stable remit and point to the same briefing; do not duplicate different roster records.

- [x] **Step 3: Read back and validate paths/content**

Read each new SOUL and the one TEAM document; confirm no placeholders expose work as ready, no secret-like token was copied, roles do not duplicate or overclaim authority, and the root instruction does not claim the briefing was loaded when it was not.

**Validation:** Nonsecret file read-back and exact revision. Actual prompt loading remains `not checked` until an authorized root session is started.

### Task 5: Bind root to the correct board without changing shared dispatch

**Files:** Only a verified root-profile-scoped board selector if the installed schema supports one; no global config or database surgery.

**Interfaces:**
- Consumes: owner-selected/existing Gormes board, real dispatcher owner/scope, installed config and Kanban schemas.
- Produces: explicit root board binding evidence, or a precise blocker with no mutation.

- [ ] **Step 1: Inspect boards, dispatcher, and root tool schemas read-only**

Use native `hermes kanban boards list`, `hermes kanban diagnostics`, installed source/config schemas, and gateway/dispatcher status commands only after confirming their read-only behavior. Identify the Gormes board slug/database root, dispatcher owner and lock scope, existing consumers, and per-setting reload semantics. Inspect installed root tool selectors and top-level availability gates for both CLI and Telegram; record schemas and effective prompt capabilities as unchecked unless actually observed. Do not rely on the active host board pointer.

- [ ] **Step 2: Require owner resolution for a missing/ambiguous board**

If no unique existing Gormes board can be established, ask the owner whether to create a board with a named slug before creating one. Never choose a board or mutate other boards silently.

- [ ] **Step 3: Apply only explicit root-scoped binding when supported**

Set a board selector only if installed configuration/CLI demonstrates the supported profile-scoped mechanism. Read it back. Do not set global orchestrator/dispatcher policy, create cards, alter concurrency or auto-decomposition, or start/restart a gateway.

**Validation:** Board slug and database root recorded in TEAM revision; dispatcher owner and blast radius evidenced; otherwise remain unbound and report `not checked`.

### Task 6: Gate authentication, Telegram, and all task dispatch

**Files:** No credential changes until the user completes required rotation and enters a replacement using a supported secure prompt. No task cards/workspaces in this task.

**Interfaces:**
- Consumes: profile/model configuration, board and dispatcher evidence, secure human credential action if separately available.
- Produces: per-profile redacted auth/readiness table, Telegram configuration state, and explicit release gates.

- [ ] **Step 1: Keep model authentication separate**

Inspect the installed `openai-codex` status/add command behavior before using it. Do not copy auth files, collect OAuth codes/tokens in chat, or run auth refresh/login without explicit authorization. Record each profile as `unchecked` until its effective context is verified; do not claim inference readiness.

- [ ] **Step 2: Leave Telegram blocked until safe credential entry is possible**

The token already pasted into chat is compromised: require revocation and a newly issued token. The Telegram bot ID embedded in a token is not proof of the human owner's Telegram user ID. Use only an installed secure, hidden-input setup prompt under `hermes -p gormes-agent-team`; verify exactly where the installed wizard stores the value and the selected allowlist field before entering it. If this environment cannot provide an interactive secure prompt, stop and provide the user the exact profile-scoped command to run locally; never accept another token in chat. Do not start/restart/rescan the gateway, set a home destination, test handoff, or send a message without separate approval.

- [ ] **Step 3: Keep worker work unreleased**

Create no cards or tasks. Before any later dispatch, resolve the repo's prohibition on worktrees versus the team's need for one-writer isolated candidates, name an independent reviewer and per-task integration owner, and agree finite enforceable attempt/review/deadline/spend limits. Do not treat a profile, configured provider, or auth status as authorization to run.

**Validation:** Readiness table for all nine profiles distinguishing configured/authentication reported/inference tested/released. Telegram remains gated until rotation, secure setup, destination/allowlist verification, and separately authorized gateway operations. No live inference or outbound messaging.

## Final acceptance receipt

The provisioning operation is complete only if all nine profile IDs/homes and role files match the spec, every provider/model pair is read back, one current nonsecret TEAM revision exists, and all board/readiness evidence is accurately classified. Report remaining auth as unchecked, inference as not run, Telegram as gated/unconfigured if its secure entry prerequisites are unmet, no workers released, and any unresolved dispatcher/workspace/recovery blockers. `git diff --check` validates this plan artifact only; it does not certify live Hermes behavior.
