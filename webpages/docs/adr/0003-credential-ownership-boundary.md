# Credential ownership boundary uses a shared pool with per-profile owner tags

The runtime credential pool (`auth.json`) lives at `$GORMES_BASE_HOME` so all profiles read from the same store. Each stored entry carries an `owner_profile` field. Provider API keys (OpenRouter, Anthropic, OpenAI) may be owned by `"main"` and shared across all profiles. Channel bot tokens (Telegram, Discord, Slack) are always owned by a specific profile id and never shared by default. The base `.env` holds all secrets using profile-scoped env var names (`GORMES_TULIN_TELEGRAM_BOT_TOKEN`). There is no per-profile `.env` or per-profile `auth.json`.

**Status**: accepted

**Considered options**: (a) per-profile `auth.json` (simple isolation but forces re-authentication per profile), (b) fully shared `auth.json` without ownership metadata (loses the provider-vs-channel distinction), (c) shared pool with `owner_profile` tagging (chosen — one store, profile-aware filtering, no duplication of OAuth flows).

**Consequences**: The credential pool reader must become profile-aware: `LoadCredentialPool` accepts an optional profile id and filters entries by `owner_profile`. The `auth.json` schema gains an optional `owner_profile` field; entries without it are implicitly shared (owned by `"main"`). The gateway already reads the base `.env` before profile startup, so env-var-based secrets are already correct.