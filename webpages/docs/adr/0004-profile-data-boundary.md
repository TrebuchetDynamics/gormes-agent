# Profile data boundary scopes memory, sessions, workspace, cache, and runtime per profile

When a named profile is active, `memory/`, `sessions/`, `workspace/`, `cache/`, and `runtime/` live under `$GORMES_HOME` (which resolves to `$GORMES_BASE_HOME/profiles/<name>/`) and are never shared between profiles. The main profile uses `$GORMES_BASE_HOME/profiles/main/` like every other runnable profile. `auth.json`, `config.toml`, and `.env` stay at `$GORMES_BASE_HOME` and are shared. Workspace is a list; the profile-local `workspace/` is the implicit first entry, with additional external paths configured in profile config.

**Status**: accepted

**Considered options**: (a) every directory at the base home root regardless of profile (simple but no isolation), (b) everything per-profile including credentials (clean isolation but forces re-authentication per profile and duplicates whisper model downloads), (c) chosen boundary: data directories per-profile, config and credentials shared.

**Consequences**: Cache-like dirs (`cache/audio`, `cache/whisper`) are per-profile, which means whisper model files are downloaded once per profile. This is the deliberate trade-off: the user prefers complete profile independence over shared model storage. The `workspace/` dir is seeded with agent templates by the gateway for each profile independently.