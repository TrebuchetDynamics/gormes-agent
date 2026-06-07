package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTermuxPathHelpersStayUnderConfiguredGormesHome(t *testing.T) {
	root := newTermuxPathRoot(t)
	home := filepath.Join(root, "home")
	gormesHome := filepath.Join(root, "gormes-home")
	prefix := filepath.Join(root, "com.termux", "files", "usr")

	t.Setenv("TERMUX_VERSION", "0.119.0")
	t.Setenv("PREFIX", prefix)
	t.Setenv("HOME", home)
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")

	paths := map[string]string{
		"gormes_home":                 GormesHome(),
		"config":                      ConfigPath(),
		"yaml_config":                 YAMLConfigPath(),
		"dotenv":                      EnvPath(),
		"log":                         LogPath(),
		"crash_log_dir":               CrashLogDir(),
		"session_db":                  SessionDBPath(),
		"session_index_mirror":        SessionIndexMirrorPath(),
		"memory_db":                   MemoryDBPath(),
		"kanban_db":                   KanbanDBPath(),
		"kanban_home":                 KanbanHome(),
		"cron_mirror":                 (Config{}).CronMirrorPath(),
		"skills_root":                 (Config{}).SkillsRoot(),
		"skills_usage_log":            (Config{}).SkillsUsageLogPath(),
		"hooks_root":                  HooksRoot(),
		"gateway_runtime_status":      GatewayRuntimeStatusPath(),
		"gateway_lock_dir":            GatewayLockDir(),
		"boot":                        BootPath(),
		"tool_audit_log":              ToolAuditLogPath(),
		"delegation_resolved_run_log": (DelegationCfg{}).ResolvedRunLogPath(),
	}
	for name, got := range paths {
		assertTermuxPathUnder(t, name, got, gormesHome)
		assertTermuxPathNoDesktopCheckout(t, name, got)
		assertTermuxPathNotUnder(t, name, got, prefix)
	}
}

func TestTermuxPathHelpersDefaultToSyntheticHomeDotGormes(t *testing.T) {
	root := newTermuxPathRoot(t)
	home := filepath.Join(root, "home")
	prefix := filepath.Join(root, "com.termux", "files", "usr")
	wantHome := filepath.Join(home, ".gormes")

	t.Setenv("TERMUX_VERSION", "0.119.0")
	t.Setenv("PREFIX", prefix)
	t.Setenv("HOME", home)
	t.Setenv("GORMES_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")

	paths := map[string]string{
		"gormes_home":            GormesHome(),
		"config":                 ConfigPath(),
		"dotenv":                 EnvPath(),
		"session_db":             SessionDBPath(),
		"memory_db":              MemoryDBPath(),
		"gateway_runtime_status": GatewayRuntimeStatusPath(),
		"tool_audit_log":         ToolAuditLogPath(),
	}
	for name, got := range paths {
		assertTermuxPathUnder(t, name, got, wantHome)
		assertTermuxPathNoDesktopCheckout(t, name, got)
		assertTermuxPathNotUnder(t, name, got, prefix)
	}
}

func newTermuxPathRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "gormes-termux-path-")
	if err != nil {
		t.Fatalf("create termux fixture root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

func assertTermuxPathUnder(t *testing.T, name, got, root string) {
	t.Helper()
	if !termuxSameOrUnder(got, root) {
		t.Fatalf("%s path = %q, want under %q", name, got, root)
	}
}

func assertTermuxPathNotUnder(t *testing.T, name, got, root string) {
	t.Helper()
	if termuxSameOrUnder(got, root) {
		t.Fatalf("%s path = %q, must not target Termux install prefix %q", name, got, root)
	}
}

func assertTermuxPathNoDesktopCheckout(t *testing.T, name, got string) {
	t.Helper()
	for _, forbidden := range []string{"/home/xel", "workspace-mineru", "workspace-gormes"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("%s path = %q contains desktop checkout marker %q", name, got, forbidden)
		}
	}
}

func termuxSameOrUnder(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
