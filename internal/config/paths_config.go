package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"

func XDGConfigHome() string {
	return paths.XDGConfigHome()
}

// GormesHome returns the native Gormes state/config root. GORMES_HOME wins;
// otherwise Gormes uses ~/.gormes so it never needs to share Hermes runtime
// state such as ~/.hermes/auth.json or ~/.hermes/gateway_state.json.
func GormesHome() string {
	return paths.GormesHome()
}

// GormesBaseHome returns the base Gormes home that owns profile roots. When a
// process is already scoped to a named profile (`.../.gormes/profiles/<name>`),
// profile-management commands still need the parent `.../.gormes` so they do
// not create nested profile trees under the active profile.
func GormesBaseHome() string {
	return paths.GormesBaseHome()
}

// GormesBaseHomeFor returns the parent Gormes home for a named profile root,
// otherwise it returns current unchanged.
func GormesBaseHomeFor(current string) string {
	return paths.GormesBaseHomeFor(current)
}

// SubprocessHome returns the Hermes-compatible subprocess HOME for the active
// Gormes home. The directory must already exist; this keeps legacy/default
// profiles from silently redirecting shell tools before profile creation or
// migration has materialized the profile-local home tree.
func SubprocessHome() (string, bool) {
	return paths.SubprocessHome()
}

// SubprocessHomeFor returns <gormesHome>/home when it exists as a directory.
func SubprocessHomeFor(gormesHome string) (string, bool) {
	return paths.SubprocessHomeFor(gormesHome)
}

// ConfigPath returns the Gormes TOML config file path.
func ConfigPath() string {
	return paths.ConfigPath()
}

// YAMLConfigPath returns the YAML variant of the Gormes config file path.
// This is used as a fallback when config.toml doesn't exist, allowing Hermes
// users to copy their config.yaml directly without converting to TOML.
func YAMLConfigPath() string {
	return paths.YAMLConfigPath()
}

// LogPath returns the default path for the Gormes log file.
func LogPath() string {
	return paths.LogPath()
}

// CrashLogDir returns the directory where TUI panic dumps are written.
func CrashLogDir() string {
	return paths.CrashLogDir()
}

// SessionDBPath returns the default location of the bbolt sessions map.
// When a profiles/main/ directory exists under GormesHome the path is
// under that profile root; otherwise the legacy root location is returned.
func SessionDBPath() string {
	return CurrentProfileStorageContract().SessionDBPath
}

// SessionIndexMirrorPath returns the default location of the read-only YAML
// mirror for the bbolt session map.
func SessionIndexMirrorPath() string {
	return paths.SessionIndexMirrorPath()
}

// MemoryDBPath returns the default location of the Phase-3.A SQLite
// memory database. When a profiles/main/ directory exists under GormesHome
// the path is under that profile root; otherwise the legacy root location
// is returned.
func MemoryDBPath() string {
	return CurrentProfileStorageContract().MemoryDBPath
}

// KanbanDBPath returns the native Gormes Kanban SQLite database path.
// Gormes intentionally ignores Hermes runtime env vars here; Hermes state is
// only read by explicit migrate commands.
func KanbanDBPath() string {
	return paths.KanbanDBPath()
}

// KanbanHome returns the root directory for the Kanban board registry.
// Named board databases live under <KanbanHome>/kanban/boards/<slug>/kanban.db
// while the legacy default board lives at <KanbanHome>/kanban.db.
func KanbanHome() string {
	return paths.KanbanHome()
}

// CronMirrorPath returns the resolved CRON.md path — either
// cfg.Cron.MirrorPath (explicit override) or the Gormes home default.
func (c Config) CronMirrorPath() string {
	return paths.CronMirrorPath(c.Cron.MirrorPath)
}

// HooksRoot returns the root directory for gateway HOOK.yaml hook directories.
func HooksRoot() string {
	return paths.HooksRoot()
}

// GatewayRuntimeStatusPath returns the shared gateway_state.json read-model
// path for live gateway lifecycle status.
func GatewayRuntimeStatusPath() string {
	return paths.GatewayRuntimeStatusPath()
}

// GatewayLockDir returns the machine-local directory for token-scoped gateway
// credential locks.
func GatewayLockDir() string {
	return paths.GatewayLockDir()
}

// BootPath returns the BOOT.md path used by the built-in gateway startup hook.
func BootPath() string {
	return paths.BootPath()
}

// ToolAuditLogPath returns the append-only JSONL path for tool execution audit records.
func ToolAuditLogPath() string {
	return paths.ToolAuditLogPath()
}

// ResolvedRunLogPath returns the JSONL path for append-only subagent run logs.
// An explicit TOML override wins; otherwise Gormes writes under GormesHome.
func (d DelegationCfg) ResolvedRunLogPath() string {
	return paths.DelegationRunLogPath(d.RunLogPath)
}
