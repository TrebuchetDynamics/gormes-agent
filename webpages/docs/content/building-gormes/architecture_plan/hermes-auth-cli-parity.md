---
title: "Hermes Auth CLI Parity Manifest"
description: "Source-of-truth manifest of hermes auth, hermes model, hermes setup, and hermes mcp login as they exist in canonical Python Hermes today, for Gormes parity planning."
weight: 95
---

# Hermes Auth CLI Parity Manifest

Read-only audit of the operator-visible CLI surface for **provider auth and
provider lifecycle** in canonical Python Hermes
(`workspace-mineru/hermes-agent/`). Captured for downstream Gormes planner
splits - no Hermes or Gormes runtime code changed by this pass.

Scope: only commands that authenticate, configure, or remove an inference
provider (or the Spotify control-plane provider). Out of scope: chat, gateway,
cron, sessions, tools, mcp non-login subcommands, slack manifest, whatsapp,
backup, status, doctor.

All paths below are absolute Python Hermes paths.

## 1. Command-tree manifest (non-deprecated surface)

`hermes auth` is the canonical pooled-credential CLI; `hermes model`,
`hermes setup`, and `hermes mcp login` are the supported adjacent surfaces a
user reaches when picking a provider, running the full wizard, or repairing a
broken MCP OAuth session.

| Command form | Behavior summary | Source: file:lines |
|---|---|---|
| `hermes auth` | Bare invocation. Prints the credential pool (per provider, with current pick markers and exhaustion status), prints AWS Bedrock identity if boto3 resolves credentials, then enters an interactive menu: add / remove / reset cooldowns / set rotation strategy / exit. | `hermes_cli/auth_commands.py:449-507`, `hermes_cli/auth_commands.py:632-656`; parser at `hermes_cli/main.py:8245-8307` |
| `hermes auth add <provider> [--type oauth\|api-key] [--label LBL] [--api-key VAL] [--portal-url URL] [--inference-url URL] [--client-id ID] [--scope S] [--no-browser] [--timeout SEC] [--insecure] [--ca-bundle PATH]` | Add a pooled credential for `<provider>`. Default type is `oauth` for `anthropic|nous|openai-codex|qwen-oauth|google-gemini-cli` and `api-key` for everyone else (and for `custom:*` pools). API-key path: prompt securely (or use `--api-key`), label-or-prompt, write `PooledCredential(auth_type=api_key, source=manual, base_url=PROVIDER_REGISTRY[provider].inference_base_url)`. OAuth path dispatches per provider (see section 3). On any add, all suppressed credential sources for that provider are unsuppressed. | parser `hermes_cli/main.py:8250-8282`; impl `hermes_cli/auth_commands.py:161-336` |
| `hermes auth list [provider]` | List pooled credentials for one provider (filter) or every provider in `PROVIDER_REGISTRY + {openrouter} + list_custom_pool_providers()`. Marks the currently-selected entry with `<-`, shows label, auth type, source, and exhaustion status with retry-window hints. | parser `hermes_cli/main.py:8283-8284`; impl `hermes_cli/auth_commands.py:339-363` |
| `hermes auth remove <provider> <target>` | Remove a credential by index, entry id, or exact label. After pool removal, dispatches the unified `find_removal_step` from `agent.credential_sources` which performs source-specific cleanup (env vars, external OAuth files like `~/.codex/auth.json`, gh-cli copilot tokens, claude-code, qwen-cli, etc.) and may suppress that source so it does not get re-imported. | parser `hermes_cli/main.py:8285-8291`; impl `hermes_cli/auth_commands.py:366-401` |
| `hermes auth reset <provider>` | Clear exhaustion / rate-limit / auth-failure status on every credential for `<provider>` so they are eligible to be picked again on the next inference call. | parser `hermes_cli/main.py:8292-8295`; impl `hermes_cli/auth_commands.py:404-408` |
| `hermes auth status <provider>` | Print logged-in / logged-out for `<provider>` via `auth.get_auth_status(provider)`, including auth_type, client_id, redirect_uri, scope, expires_at, api_base_url where available. | parser `hermes_cli/main.py:8296-8297`; impl `hermes_cli/auth_commands.py:411-428`, `hermes_cli/auth.py:3419-3446` |
| `hermes auth logout <provider>` | Delegates to `auth.logout_command` with `provider=<provider>`. Validates against `is_known_auth_provider`, calls `clear_provider_auth(target)`, resets `model.provider` if config matched, and prints a follow-up hint depending on whether `OPENROUTER_API_KEY` is set. | parser `hermes_cli/main.py:8298-8299`; impl `hermes_cli/auth_commands.py:431-432`, `hermes_cli/auth.py:4334-4360` |
| `hermes auth spotify [login\|status\|logout] [--client-id ID] [--redirect-uri URI] [--scope S] [--no-browser] [--timeout SEC]` | Spotify control-plane auth (separate from inference providers). Default action is `login` (PKCE browser flow). `status` and `logout` route through `auth_status_command` / `auth_logout_command` with `provider="spotify"`. Login flow prompts an interactive setup wizard if no `client_id` is found in env, args, or stored state. | parser `hermes_cli/main.py:8300-8307`; impl `hermes_cli/auth_commands.py:435-446`, login flow `hermes_cli/auth.py:2018-2108` |
| `hermes model [--portal-url URL] [--inference-url URL] [--client-id ID] [--scope S] [--no-browser] [--timeout SEC] [--ca-bundle PATH] [--insecure]` | Interactive provider + model picker. Calls `select_provider_and_model(args)` which: shows current provider/model, presents the provider menu, runs the per-provider login flow if needed (e.g. Codex via `_login_openai_codex`, Nous via device code, anthropic OAuth via `anthropic_adapter.run_hermes_oauth_login_pure`, etc.), fetches the model catalog (provider-specific or via `model_catalog`), prompts model selection, and persists `model.default` + `model.provider` to config. Requires a TTY (`_require_tty("model")`). | parser `hermes_cli/main.py:7862-7902`; handler `hermes_cli/main.py:1445-1448` then `select_provider_and_model` at `hermes_cli/main.py:1451+` |
| `hermes setup [section] [--non-interactive] [--reset] [--reconfigure] [--quick]` | Interactive setup wizard. With no `section`, runs the full wizard (which on existing installs is implicitly `--reconfigure` mode). `section` constrains it to one block: `model` (calls `setup_model_provider`, the same provider+model picker as `hermes model`), `tts`, `terminal`, `gateway`, `tools`, `agent`. `--quick` only prompts items that are missing/unset; `--reset` clears config to defaults; `--non-interactive` uses defaults/env vars. | parser `hermes_cli/main.py:8089-8123`; handler `hermes_cli/main.py:1438-1442`; wizard `hermes_cli/setup.py:667+` (model section), `hermes_cli/setup.py:1165+` (tts), `1175+` (terminal), `1541+` (agent) |
| `hermes mcp login <name>` | Force re-authentication for an OAuth-based MCP server. Looks up the server in MCP config, refuses if `auth != "oauth"` or if no URL, then `MCPOAuthManager.remove(name)` to wipe disk + in-memory cache, and finally calls `_probe_single_server(name, server_config)` which re-runs the OAuth flow (browser redirect + callback capture) and reports the resulting tool count. | parser `hermes_cli/main.py:9278-9282`; handler `hermes_cli/mcp_config.py:582-636`; dispatch `hermes_cli/mcp_config.py:740-758` |

Notes:
- `hermes auth` and `hermes model` together are the operator-visible auth surface. `hermes setup model` reaches the same `select_provider_and_model` path as `hermes model`, but routes through the wizard's defaults/quick logic.
- `hermes logout` (top-level, not `hermes auth logout`) still exists and is **not** in the deprecation note below - it is supported but narrowly scoped (`--provider` choices: `nous`, `openai-codex`, `spotify`). It dispatches to the same `logout_command` as `hermes auth logout`. Parser at `hermes_cli/main.py:8232-8243`.
  - This is an open question for parity: see section 5.

## 2. Deprecation note - `hermes login`

`hermes login` is the historical entry point that the Gormes README must
**not** recommend. The parser is still registered (so `hermes login --help`
still prints), but the handler is a stub that prints a deprecation message
and exits cleanly.

Source: `hermes_cli/auth.py:3848-3853`

```python
def login_command(args) -> None:
    """Deprecated: use 'hermes model' or 'hermes setup' instead."""
    print("The 'hermes login' command has been removed.")
    print("Use 'hermes auth' to manage credentials,")
    print("'hermes model' to select a provider, or 'hermes setup' for full setup.")
    raise SystemExit(0)
```

Parser definition (still wired so flags do not break old scripts, but every
flag is ignored): `hermes_cli/main.py:8186-8227`. Flags accepted but no-op:
`--provider {nous,openai-codex}`, `--portal-url`, `--inference-url`,
`--client-id`, `--scope`, `--no-browser`, `--timeout`, `--ca-bundle`,
`--insecure`.

Redirect targets:
- `hermes auth` - pooled credentials.
- `hermes model` - provider + model picker (and the OAuth flow for that provider).
- `hermes setup` - full or sectioned wizard.

**Gormes parity rule.** Public Gormes auth docs, help, and catalogs use `gormes auth add <provider> --type oauth`. The top-level `gormes login` parser is removed; typing it returns secret-safe guidance to the canonical auth command without executing provider auth.

## 3. Per-provider `auth add` behavior matrix

`<provider>` is the positional arg to `hermes auth add`. `_OAUTH_CAPABLE_PROVIDERS = {"anthropic", "nous", "openai-codex", "qwen-oauth", "google-gemini-cli"}` (`hermes_cli/auth_commands.py:36`). For all other providers in `PROVIDER_REGISTRY`, `--type oauth` is rejected (raises `SystemExit("hermes auth add <provider> is not implemented for auth type oauth yet.")` via the catch-all at `auth_commands.py:336`); only `--type api-key` works.

| Provider id | Default flow | Source function | Stored to `~/.hermes/auth.json` (or pool) | Account-pool semantics | Notes |
|---|---|---|---|---|---|
| `anthropic` | OAuth (PKCE) | `agent.anthropic_adapter.run_hermes_oauth_login_pure` invoked from `auth_commands.py:221-245` | `PooledCredential(auth_type=oauth, source="manual:hermes_pkce", access_token, refresh_token, expires_at_ms, base_url="https://api.anthropic.com")` | Multiple OAuth credentials supported; pooled rotation per `credential_pool_strategies.anthropic`. | Also accepts `--type api-key`; env-var fallbacks `ANTHROPIC_API_KEY`, `ANTHROPIC_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN` (`auth.py:240`). |
| `nous` | OAuth device code | `auth.py:_nous_device_code_login` (`auth.py:4075+`), persisted via `persist_nous_credentials` (`auth.py:2891+`) | provider state under `providers.nous` plus mint-agent-key state; pool entry honors `--label`. | Pool entry is the persisted Nous state. Label propagated into `providers.nous` so `label_from_token` does not overwrite it on the next `load_pool("nous")`. | Honors `--portal-url`, `--inference-url`, `--client-id`, `--scope`, `--no-browser`, `--timeout`, `--insecure`, `--ca-bundle`, `--min-key-ttl-seconds` (default 5min). Defaults from `DEFAULT_NOUS_PORTAL_URL` / `DEFAULT_NOUS_INFERENCE_URL` / `DEFAULT_NOUS_CLIENT_ID` / `DEFAULT_NOUS_SCOPE` (`auth.py:135-143`). |
| `openai-codex` | OAuth device code | `auth.py:_codex_device_code_login` (`auth.py:3930+`); pool persistence at `auth_commands.py:270-293` | `PooledCredential(auth_type=oauth, source="manual:device_code", access_token, refresh_token, base_url="https://chatgpt.com/backend-api/codex"  / `DEFAULT_CODEX_BASE_URL`, last_refresh)` | Pre-add: `unsuppress_credential_source("openai-codex", "device_code")` so a re-link after `auth remove` is not skipped. | **Vendor CLI conflict.** Hermes maintains its own Codex OAuth session separate from the Codex CLI / VS Code extension to avoid refresh-token rotation collisions where one app's refresh invalidates the other (`auth.py:2120-2126`). The non-pooled `_login_openai_codex` (`auth.py:3856+`) prompts to import `~/.codex/auth.json` if found but warns "a separate login is recommended" and "Hermes will keep working independently with its own session" (`auth.py:3897-3909`). The pooled `auth add` path (`auth_commands.py:270-293`) **does not** reuse `~/.codex/`; it always runs a fresh device-code flow. Removal step (`agent.credential_sources` for `openai-codex`/`device_code`) wipes the Hermes-owned tokens and suppresses the source. |
| `qwen-oauth` | Vendor CLI import (no fresh OAuth) | `auth.py:resolve_qwen_runtime_credentials(refresh_if_expiring=False)` invoked from `auth_commands.py:316-334`; tokens read from Qwen-CLI's `~/.qwen/oauth_creds.json` via `_read_qwen_cli_tokens` (`auth.py:1278+`). | `PooledCredential(auth_type=oauth, source="manual:qwen_cli", access_token=<api_key>, base_url=<resolved>)` | Token refresh handled by Hermes via `_refresh_qwen_cli_tokens` (`auth.py:1321+`). | If Qwen CLI is not installed or `~/.qwen/oauth_creds.json` is missing, the call raises and `auth add qwen-oauth` fails. There is no Hermes-native Qwen OAuth flow. |
| `google-gemini-cli` | OAuth (browser PKCE) | `agent.google_oauth.run_gemini_oauth_login_pure` invoked from `auth_commands.py:295-314` | `PooledCredential(auth_type=oauth, source="manual:google_pkce", access_token, refresh_token)`; default label is the user's email if returned. | Status / refresh via `resolve_gemini_oauth_runtime_credentials` (`auth.py:1453+`) and `get_gemini_oauth_auth_status` (`auth.py:1495+`). | Endpoint: `DEFAULT_GEMINI_CLOUDCODE_BASE_URL` (Google CloudCode endpoint), distinct from `gemini` (Google AI Studio). |
| `spotify` | Browser PKCE (control-plane, not inference) | `auth.login_spotify_command` (`auth.py:2018-2108`) - entered via `hermes auth spotify [login]`, not `hermes auth add spotify`. | provider state under `providers.spotify` (NOT `set_active=True`). Stored: client_id, redirect_uri, scope list, accounts/api base URLs, access/refresh tokens, expiry. | Single Spotify session per Hermes home. `spotify` is not in `PROVIDER_REGISTRY`; trying `hermes auth add spotify` is rejected. | If `client_id` is not configured anywhere, prompts an interactive developer-app setup wizard (`_spotify_interactive_setup`, `auth.py:1957+`). Honors `HERMES_SPOTIFY_CLIENT_ID` env. Redirect URI must be allow-listed in the user's Spotify app. |
| `copilot` | API key | env-var resolution via `ProviderConfig.api_key_env_vars=("COPILOT_GITHUB_TOKEN","GH_TOKEN","GITHUB_TOKEN")`; `hermes auth add copilot --type api-key` writes a manual entry. | `PooledCredential(auth_type=api_key, source="manual", access_token=<paste>, base_url=DEFAULT_GITHUB_MODELS_BASE_URL)` | Standard pool. | `gh_cli` source is a separate auto-detected source with its own RemovalStep. |
| `copilot-acp` | external_process (vendor CLI) | `auth_type="external_process"` (`auth.py:170-176`). `auth add` of api-key is allowed but routes through API-key handler. | base URL `DEFAULT_COPILOT_ACP_BASE_URL` (override `COPILOT_ACP_BASE_URL`). | n/a - process-managed auth. | Treated as an API-key provider for pool purposes. |
| `gemini` | API key | env vars `GOOGLE_API_KEY`, `GEMINI_API_KEY`. | manual entry, base `https://generativelanguage.googleapis.com/v1beta`, override `GEMINI_BASE_URL`. | Standard pool. | Distinct from `google-gemini-cli`. |
| `zai` | API key | env vars `GLM_API_KEY`, `ZAI_API_KEY`, `Z_AI_API_KEY`. | manual entry, base `https://api.z.ai/api/paas/v4`, override `GLM_BASE_URL`. | Standard pool. | `detect_zai_endpoint` (`auth.py:514`) auto-rewrites base for some keys. |
| `kimi-coding` | API key | env vars `KIMI_API_KEY`, `KIMI_CODING_API_KEY`; base `https://api.moonshot.ai/v1`, override `KIMI_BASE_URL`. | manual entry. | Standard pool. | `_resolve_kimi_base_url` (`auth.py:409`) auto-redirects `sk-kimi-` keys to `api.kimi.com/coding`. |
| `kimi-coding-cn` | API key | env var `KIMI_CN_API_KEY`. | manual entry, base `https://api.moonshot.cn/v1`. | Standard pool. | Mainland China endpoint. |
| `stepfun` | API key | `STEPFUN_API_KEY`, base `STEPFUN_STEP_PLAN_INTL_BASE_URL`, override `STEPFUN_BASE_URL`. | manual entry. | Standard pool. | |
| `arcee` | API key | `ARCEEAI_API_KEY`, base `https://api.arcee.ai/api/v1`, override `ARCEE_BASE_URL`. | manual entry. | Standard pool. | |
| `minimax` | API key | `MINIMAX_API_KEY`, base `https://api.minimax.io/anthropic`, override `MINIMAX_BASE_URL`. | manual entry. | Standard pool. | Anthropic-shaped surface. |
| `minimax-cn` | API key | `MINIMAX_CN_API_KEY`, base `https://api.minimaxi.com/anthropic`, override `MINIMAX_CN_BASE_URL`. | manual entry. | Standard pool. | |
| `alibaba` | API key | `DASHSCOPE_API_KEY`, base `https://dashscope-intl.aliyuncs.com/compatible-mode/v1`, override `DASHSCOPE_BASE_URL`. | manual entry. | Standard pool. | |
| `alibaba-coding-plan` | API key | `ALIBABA_CODING_PLAN_API_KEY`, `DASHSCOPE_API_KEY`, base `https://coding-intl.dashscope.aliyuncs.com/v1`. | manual entry. | Standard pool. | |
| `deepseek` | API key | `DEEPSEEK_API_KEY`, base `https://api.deepseek.com/v1`. | manual entry. | Standard pool. | |
| `xai` | API key | `XAI_API_KEY`, base `https://api.x.ai/v1`. | manual entry. | Standard pool. | |
| `nvidia` | API key | `NVIDIA_API_KEY`, base `https://integrate.api.nvidia.com/v1`. | manual entry. | Standard pool. | |
| `ai-gateway` | API key | `AI_GATEWAY_API_KEY`, base `https://ai-gateway.vercel.sh/v1`. | manual entry. | Standard pool. | Vercel AI Gateway. |
| `opencode-zen` | API key | `OPENCODE_ZEN_API_KEY`, base `https://opencode.ai/zen/v1`. | manual entry. | Standard pool. | |
| `opencode-go` | API key | `OPENCODE_GO_API_KEY`, base `https://opencode.ai/zen/go/v1`. | manual entry. | Standard pool. | Mixed API surface (OpenAI-compat for GLM/Kimi, Anthropic for MiniMax) selected per-model. |
| `kilocode` | API key | `KILOCODE_API_KEY`, base `https://api.kilo.ai/api/gateway`. | manual entry. | Standard pool. | |
| `huggingface` | API key | `HF_TOKEN`, base `https://router.huggingface.co/v1`. | manual entry. | Standard pool. | |
| `xiaomi` | API key | `XIAOMI_API_KEY`, base `https://api.xiaomimimo.com/v1`. | manual entry. | Standard pool. | |
| `ollama-cloud` | API key | `OLLAMA_API_KEY`, base `DEFAULT_OLLAMA_CLOUD_BASE_URL`, override `OLLAMA_BASE_URL`. | manual entry. | Standard pool. | |
| `bedrock` | AWS SDK (boto3 chain) | `auth_type="aws_sdk"` (`auth.py:351-358`). `hermes auth add bedrock` is not handled - bedrock surfaces in the bare `hermes auth` summary via boto3 STS (`auth_commands.py:457-476`). | n/a - credentials come from boto3 chain (env vars, profiles, SSO, role). | n/a | No `--type api-key` path because pool dispatch falls through `auth_commands.py:336` for unrecognized OAuth and never gets here for AWS SDK. **Open question** in section 5. |
| `azure-foundry` | API key | `AZURE_FOUNDRY_API_KEY`, base set per-deployment via `AZURE_FOUNDRY_BASE_URL`. | manual entry. | Standard pool. | Inference base intentionally empty in registry - user must provide it. |
| `openrouter` (alias `or`, `open-router`) | API key | `auth_commands._normalize_provider` rewrites to `openrouter`. Not in `PROVIDER_REGISTRY`; base URL hardcoded to `OPENROUTER_BASE_URL` in `_provider_base_url`. | `PooledCredential(auth_type=api_key, source="manual", access_token, base_url=OPENROUTER_BASE_URL)`. | Standard pool. | Special-cased in `_normalize_provider`. |
| `custom:<name>` | API key | matches `custom_providers` config entries; resolved via `_resolve_custom_provider_input` and `_get_custom_provider_config`. | `PooledCredential(auth_type=api_key, source="manual", access_token, base_url=<from custom_providers>)`. | Standard pool. | Display name vs `provider_key` both accepted. |

## 4. `auth list` / `status` / `logout` / `reset` / `spotify` behavior

**`auth list [provider]`** - Resolves the filter via `_normalize_provider` (so
aliases like `or` become `openrouter`). With no filter, iterates over
`sorted(PROVIDER_REGISTRY.keys() + {"openrouter"} + list_custom_pool_providers())`.
For each provider with at least one entry, prints a header
`<provider> (N credentials):` and one line per entry showing index, label,
auth_type, source (with the `manual:` prefix stripped via `_display_source`),
exhaustion status (`_format_exhausted_status` distinguishes rate-limited /
auth-failed / exhausted with retry-window math), and a `<-` marker on the
currently-selected entry. Source: `hermes_cli/auth_commands.py:339-363`.

**`auth status <provider>`** - Required positional. Calls
`auth.get_auth_status(provider)` (`auth.py:3419-3446`) which dispatches to per-
provider status helpers (`get_nous_auth_status` `auth.py:3256+`,
`get_codex_auth_status` `auth.py:3309+`, `get_qwen_auth_status` `auth.py:1425+`,
`get_gemini_oauth_auth_status` `auth.py:1495+`, `get_spotify_auth_status`
`auth.py:1938+`, `get_api_key_provider_status` `auth.py:3358+`,
`get_external_process_provider_status` `auth.py:3389+`). Prints
`<provider>: logged in` plus indented metadata
(`auth_type`, `client_id`, `redirect_uri`, `scope`, `expires_at`, `api_base_url`)
or `<provider>: logged out (<reason>)` on failure. Source:
`hermes_cli/auth_commands.py:411-428`.

**`auth logout <provider>`** - Wraps `auth.logout_command` with a
`SimpleNamespace`. The underlying command (`auth.py:4334-4360`) validates
`is_known_auth_provider`, picks the target as `provider or active or config_provider`, calls
`clear_provider_auth(target)` to wipe `providers.<target>` from `auth.json` and
clears `active_provider` if it pointed there, resets `model.provider` in config
when it matched, and prints
`Logged out of <Display Name>.` followed by either
`Hermes will use OpenRouter for inference.` (if `OPENROUTER_API_KEY`) or
`Run 'hermes model' or configure an API key to use Hermes.`. Source:
`hermes_cli/auth_commands.py:431-432`.

**`auth reset <provider>`** - Calls `pool.reset_statuses()` on the named
provider's pool, which clears every entry's exhaustion / cooldown so all
credentials are eligible again on the next inference call. Prints
`Reset status on <N> <provider> credentials`. Source:
`hermes_cli/auth_commands.py:404-408`.

**`auth spotify [login|status|logout]`** - Default action is `login`
(no subcommand). `login` runs PKCE: builds an authorize URL, opens a browser
unless `--no-browser` or remote-session detected, waits for the localhost
callback (`_spotify_wait_for_callback`, default 180s), exchanges code+verifier
for tokens, and stores `providers.spotify` with `set_active=False` (Spotify is
**not** an inference provider so it must not steal `active_provider`). If no
client_id is supplied or stored, runs `_spotify_interactive_setup` to walk the
user through creating a Spotify developer app. `status` and `logout` route to
the generic `auth_status_command` / `auth_logout_command` with
`provider="spotify"`. Source: `hermes_cli/auth_commands.py:435-446`,
`hermes_cli/auth.py:2018-2108`.

## 5. Open questions for the planner

1. **`hermes logout` (top-level) status.** Parser at `main.py:8232-8243` is
   live, dispatches to the same `logout_command` as `hermes auth logout`, but
   only exposes provider choices `nous|openai-codex|spotify`. Is this command
   on a deprecation path (sibling of `hermes login`) or supported in parallel?
   The user's framing of the parity rule lists `hermes auth ... logout` as the
   non-deprecated shape; the planner should decide whether Gormes ships
   `gormes logout`, only `gormes auth logout`, or both.

2. **`auth_command` vs interactive bare `hermes auth`.** The bare invocation
   prints both pool state and an AWS Bedrock STS identity (`auth_commands.py:457-476`)
   that depends on optional `boto3`. Gormes parity needs a Go path that prints
   the same pool table; the AWS STS print can be a separate row gated on AWS
   SDK availability - confirm whether the planner wants AWS bedrock identity
   in the Goncho equivalent or carved off.

3. **`auth add bedrock`.** Source has no explicit handler. The catch-all at
   `auth_commands.py:336` rejects OAuth, and there is no bedrock-specific
   API-key path because `auth_type="aws_sdk"`. Confirm whether Gormes should
   match the gap (refuse and tell the user to set AWS env / profile) or close
   it (offer `gormes auth add bedrock --profile <name>`).

4. **`auth add openrouter` accepts only `--type api-key`.** Same catch-all at
   `auth_commands.py:336` as bedrock, but openrouter is silently allowed
   because `_normalize_provider` rewrites it and the API-key branch at
   `auth_commands.py:194-219` does not require `provider in PROVIDER_REGISTRY`
   (the gate at `auth_commands.py:163-164` explicitly allows `openrouter` and
   `custom:*`). Confirm Gormes treats openrouter as a first-class provider
   ID even though it has no `ProviderConfig`.

5. **Vendor-CLI race envelope (Codex CLI / VS Code).** The pooled
   `auth add openai-codex` path always runs a fresh device-code flow and
   stores tokens in `~/.hermes/auth.json`, rejecting reuse of `~/.codex/`.
   Only the legacy `_login_openai_codex` flow (still reachable via
   `hermes model` -> OpenAI Codex provider when no Hermes credentials exist)
   prompts to import `~/.codex/auth.json`. Gormes parity must replicate the
   isolation contract - but the planner needs to decide whether
   `gormes auth add openai-codex` keeps offering import (parity with
   `cmd_model`) or stays strict (parity with `auth add`). This is the surface
   the user flagged as the README error.

6. **OAuth-capable provider set.** `_OAUTH_CAPABLE_PROVIDERS = {"anthropic", "nous", "openai-codex", "qwen-oauth", "google-gemini-cli"}`
   is hard-coded in two places (`auth_commands.py:36` and again inline at
   `auth_commands.py:173`). If a future provider grows OAuth, both spots need
   updating. The Gormes equivalent should derive this from `ProviderConfig.auth_type`
   instead.

7. **Spotify out of `PROVIDER_REGISTRY`.** Spotify is reachable only via
   `auth spotify`, never via `auth add`. Parity question: is Spotify a
   first-class auth provider in Gormes (separate command tree), or should it
   ride the same `auth` subparser?

8. **`hermes mcp login` requires `auth == "oauth"`.** Header-auth and stdio
   MCP servers cannot be re-authed via this command; the handler tells the
   user to `hermes mcp remove` + `hermes mcp add` instead
   (`mcp_config.py:613-614`). Confirm whether the Gormes equivalent narrows
   the surface the same way.

9. **Top-level `hermes login` parser still registered.** Gormes keeps a hidden compatibility shim only; public help and docs point operators at `gormes auth add <provider> --type oauth`.
