package paths

import (
	"os"
	"path/filepath"
	"strings"
)

func XDGConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

// GormesHome returns the native Gormes state/config root. GORMES_HOME wins;
// otherwise Gormes uses ~/.gormes so it never needs to share Hermes runtime
// state such as ~/.hermes/auth.json or ~/.hermes/gateway_state.json.
func GormesHome() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_HOME")); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gormes")
}

// GormesBaseHome returns the base Gormes home that owns profile roots. When a
// process is already scoped to a named profile (`.../.gormes/profiles/<name>`),
// profile-management commands still need the parent `.../.gormes` so they do
// not create nested profile trees under the active profile.
func GormesBaseHome() string {
	return GormesBaseHomeFor(GormesHome())
}

// GormesBaseHomeFor returns the parent Gormes home for a named profile root,
// otherwise it returns current unchanged.
func GormesBaseHomeFor(current string) string {
	clean := filepath.Clean(strings.TrimSpace(current))
	if clean == "." || clean == string(filepath.Separator) {
		return current
	}
	if filepath.Base(filepath.Dir(clean)) == "profiles" {
		return filepath.Dir(filepath.Dir(clean))
	}
	return current
}

// SubprocessHome returns the Hermes-compatible subprocess HOME for the active
// Gormes home. The directory must already exist; this keeps legacy/default
// profiles from silently redirecting shell tools before profile creation or
// migration has materialized the profile-local home tree.
func SubprocessHome() (string, bool) {
	return SubprocessHomeFor(GormesHome())
}

// SubprocessHomeFor returns <gormesHome>/home when it exists as a directory.
func SubprocessHomeFor(gormesHome string) (string, bool) {
	gormesHome = strings.TrimSpace(gormesHome)
	if gormesHome == "" {
		return "", false
	}
	candidate := filepath.Join(gormesHome, "home")
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return candidate, true
}

// ConfigPath returns the Gormes TOML config file path.
func ConfigPath() string {
	return filepath.Join(GormesHome(), "config.toml")
}

// YAMLConfigPath returns the YAML variant of the Gormes config file path.
func YAMLConfigPath() string {
	return filepath.Join(GormesHome(), "config.yaml")
}

// LogPath returns the default path for the Gormes log file.
func LogPath() string {
	return filepath.Join(GormesHome(), "gormes.log")
}

// CrashLogDir returns the directory where TUI panic dumps are written.
func CrashLogDir() string {
	return GormesHome()
}

// SessionIndexMirrorPath returns the default location of the read-only YAML
// mirror for the bbolt session map.
func SessionIndexMirrorPath() string {
	return filepath.Join(GormesHome(), "sessions", "index.yaml")
}

// KanbanDBPath returns the native Gormes Kanban SQLite database path.
// Gormes intentionally ignores Hermes runtime env vars here; Hermes state is
// only read by explicit migrate commands.
func KanbanDBPath() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_KANBAN_DB")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_KANBAN_HOME")); v != "" {
		return filepath.Join(v, "kanban.db")
	}
	return filepath.Join(GormesHome(), "kanban.db")
}

// KanbanHome returns the root directory for the Kanban board registry.
// Named board databases live under <KanbanHome>/kanban/boards/<slug>/kanban.db
// while the legacy default board lives at <KanbanHome>/kanban.db.
func KanbanHome() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_KANBAN_HOME")); v != "" {
		return v
	}
	return GormesHome()
}

// HooksRoot returns the root directory for gateway HOOK.yaml hook directories.
func HooksRoot() string {
	return filepath.Join(GormesHome(), "hooks")
}

// GatewayRuntimeStatusPath returns the shared gateway_state.json read-model
// path for live gateway lifecycle status.
func GatewayRuntimeStatusPath() string {
	return filepath.Join(GormesHome(), "runtime", "gateway_state.json")
}

// GatewayLockDir returns the machine-local directory for token-scoped gateway
// credential locks.
func GatewayLockDir() string {
	return filepath.Join(GormesHome(), "runtime", "gateway-locks")
}

// BootPath returns the BOOT.md path used by the built-in gateway startup hook.
func BootPath() string {
	return filepath.Join(GormesHome(), "BOOT.md")
}

// ToolAuditLogPath returns the append-only JSONL path for tool execution
// audit records.
func CronMirrorPath(configuredPath string) string {
	if configuredPath != "" {
		return configuredPath
	}
	return filepath.Join(GormesHome(), "cron", "CRON.md")
}

func DelegationRunLogPath(configuredPath string) string {
	if configuredPath != "" {
		return configuredPath
	}
	return filepath.Join(GormesHome(), "subagents", "runs.jsonl")
}

func ToolAuditLogPath() string {
	return filepath.Join(GormesHome(), "tools", "audit.jsonl")
}
