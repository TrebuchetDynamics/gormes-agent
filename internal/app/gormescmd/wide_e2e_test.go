package gormescmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWideE2E_FreshInstallReadOnlySurfaceDoesNotCreateRuntimeStores sweeps a
// broad operator read-only surface through one synthetic fresh install and pins
// the boundary between inspection commands and persistent runtime state. These
// commands may create config scaffolding, but they must not create chat,
// memory, or kanban stores just because an operator is probing a new install.
func TestWideE2E_FreshInstallReadOnlySurfaceDoesNotCreateRuntimeStores(t *testing.T) {
	root := freshInstallE2EHome(t)
	home := filepath.Join(root, "gormes-home")

	cases := [][]string{
		{"version"},
		{"config", "path"},
		{"config", "env-path"},
		{"config", "check"},
		{"profile", "list"},
		{"profile", "show"},
		{"auth", "list"},
		{"session", "list"},
		{"memory", "status"},
		{"channels", "capabilities"},
		{"checkpoints", "status"},
		{"gateway", "status"},
		{"gateway", "discover", "--timeout", "50"},
		{"plugins", "list"},
		{"restore", "--list"},
		{"doctor", "--offline", "--target", "terminal"},
	}

	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, args...)
			if code := commandExitCode(err); code != 0 {
				t.Fatalf("read-only command `%s` exited %d, want 0\nstdout=%s\nstderr=%s\nerr=%v",
					strings.Join(args, " "), code, stdout, stderr, err)
			}
			assertFreshInstallRuntimeStoresAbsent(t, home)
		})
	}
}

// TestWideE2E_JSONStdoutIsMachineOnlyAcrossSuccessAndErrorPaths widens the
// JSON conformance fence beyond individual command tests. Success and expected
// error paths must emit exactly one parseable JSON document on stdout, without
// human headers, usage blocks, or ANSI control bytes mixed into fleet logs.
func TestWideE2E_JSONStdoutIsMachineOnlyAcrossSuccessAndErrorPaths(t *testing.T) {
	freshInstallE2EHome(t)

	cases := []struct {
		name       string
		args       []string
		wantAction string
	}{
		{name: "version_success", args: []string{"version", "--json"}},
		{name: "config_check_success", args: []string{"config", "check", "--json"}},
		{name: "doctor_success", args: []string{"doctor", "--offline", "--target", "terminal", "--json"}},
		{name: "status_missing_progress_degraded", args: []string{"status", "--progress", filepath.Join("missing", "progress.json"), "--json"}},
		{name: "auth_missing_arg", args: []string{"auth", "status", "--json"}, wantAction: "missing_argument"},
		{name: "kanban_missing_subcommand", args: []string{"kanban", "--json"}, wantAction: "subcommand_required"},
		{name: "gateway_unknown_subcommand", args: []string{"gateway", "definitely-not-a-subcommand", "--json"}, wantAction: "unknown_subcommand"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, _ := executeRootCommandForTest(cmd, tc.args...)
			trimmed := strings.TrimSpace(stdout)
			if trimmed == "" {
				t.Fatalf("`gormes %s` --json wrote empty stdout; stderr=%s", strings.Join(tc.args, " "), stderr)
			}
			for _, reject := range []string{"\x1b[", "Usage:", "Gormes Agent Installer", "Gormes Doctor", "┌", "│"} {
				if strings.Contains(trimmed, reject) {
					t.Fatalf("`gormes %s` --json stdout leaked human chrome %q:\n%s", strings.Join(tc.args, " "), reject, stdout)
				}
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
				t.Fatalf("`gormes %s` --json stdout must be one parseable JSON object: %v\nstdout=%s\nstderr=%s",
					strings.Join(tc.args, " "), err, stdout, stderr)
			}
			if tc.wantAction != "" {
				if got, _ := parsed["action"].(string); got != tc.wantAction {
					t.Fatalf("`gormes %s` action = %q, want %q; stdout=%s", strings.Join(tc.args, " "), got, tc.wantAction, stdout)
				}
			}
		})
	}
}

// TestWideE2E_HermesFeatureFamiliesStayOnTheCommandSurface is a command-tree
// contract for the broad Hermes feature families operators expect after
// install: chat, setup, provider/auth, gateway, sessions, memory, tools,
// skills, MCP, cron, kanban, channels, security, migration, and ACP. It does
// not claim every row-backed feature is implemented; it proves the public
// entry points remain discoverable instead of disappearing from the binary.
func TestWideE2E_HermesFeatureFamiliesStayOnTheCommandSurface(t *testing.T) {
	freshInstallE2EHome(t)

	root := newRootCommandWithRuntime(rootRuntime{})
	paths := collectVisibleCommandPaths(root, nil)
	if len(paths) < 120 {
		t.Fatalf("command tree exposes only %d visible paths; wide Hermes surface coverage collapsed", len(paths))
	}

	required := [][]string{
		{"chat"},
		{"setup"},
		{"doctor"},
		{"auth", "list"},
		{"auth", "status"},
		{"config", "check"},
		{"gateway", "status"},
		{"gateway", "probe"},
		{"session", "list"},
		{"memory", "status"},
		{"tools", "list"},
		{"skills", "list"},
		{"skills", "sync"},
		{"mcp", "list"},
		{"cron", "list"},
		{"kanban", "list"},
		{"channels", "capabilities"},
		{"send"},
		{"logs"},
		{"dashboard"},
		{"curator", "status"},
		{"curator", "run"},
		{"curator", "effectiveness"},
		{"insights"},
		{"security", "audit"},
		{"migrate", "hermes"},
		{"claw", "migrate"},
		{"acp", "serve"},
		{"telegram"},
		{"whatsapp"},
		{"slack", "manifest"},
		{"navivox", "pair"},
	}
	for _, want := range required {
		if !visiblePathExists(paths, want) {
			t.Fatalf("required Hermes/Gormes feature path %q missing from visible command tree; available paths include %v", strings.Join(want, " "), firstVisiblePaths(paths, 20))
		}
	}
}

// TestWideE2E_UserBootstrapJourneyKeepsSecretsRedactedAndCoreFeaturesUsable
// drives the offline path a real user hits before their first provider call:
// configure model and API key, inspect config, run doctor, list tool/skill/MCP
// surfaces, initialize kanban, and verify session/memory inventories. The
// journey is intentionally credential-free but catches regressions that would
// make a freshly installed agent unusable before the LLM turn starts.
func TestWideE2E_UserBootstrapJourneyKeepsSecretsRedactedAndCoreFeaturesUsable(t *testing.T) {
	freshInstallE2EHome(t)
	secret := "sk-wide-e2e-secret-do-not-leak"

	steps := []struct {
		name     string
		args     []string
		exitCode int
		want     []string
	}{
		{name: "set_model", args: []string{"config", "set", "hermes.model", "wide-e2e-model", "--json"}, want: []string{"hermes.model"}},
		{name: "set_api_key", args: []string{"config", "set", "hermes.api_key", secret, "--json"}, want: []string{"GORMES_API_KEY", "secret"}},
		{name: "read_model", args: []string{"config", "get", "hermes.model", "--json"}, want: []string{"wide-e2e-model"}},
		{name: "read_secret_redacted", args: []string{"config", "get", "hermes.api_key", "--json"}, want: []string{"redacted"}},
		{name: "doctor_terminal", args: []string{"doctor", "--offline", "--target", "terminal", "--json"}, want: []string{"checks", "target"}},
		{name: "auth_inventory", args: []string{"auth", "list", "--json"}, want: []string{"credentials", "redacted"}},
		{name: "gateway_status", args: []string{"gateway", "status", "--json"}, want: []string{"channels", "build"}},
		{name: "gateway_probe", args: []string{"gateway", "probe", "--json"}, exitCode: 1, want: []string{"gateway_probe_unreachable", "build"}},
		{name: "channels_capabilities", args: []string{"channels", "capabilities", "--json"}, want: []string{"channels", "telegram", "slack"}},
		{name: "tools_inventory", args: []string{"tools", "list"}, want: []string{"Tools for CLI", "terminal"}},
		{name: "skills_inventory", args: []string{"skills", "list"}, want: []string{"builtin"}},
		{name: "learning_loop_status", args: []string{"curator", "status", "--json"}, want: []string{"state", "skills"}},
		{name: "learning_loop_dry_run", args: []string{"curator", "run", "--dry-run", "--sync", "--json"}, want: []string{"dry_run", "curator skipped"}},
		{name: "mcp_inventory", args: []string{"mcp", "list"}, exitCode: 2, want: []string{"row-backed in Gormes"}},
		{name: "kanban_init", args: []string{"kanban", "init", "--json"}, want: []string{"initialized", "kanban.db"}},
		{name: "kanban_list", args: []string{"kanban", "list", "--json"}, want: []string{"tasks"}},
		{name: "session_inventory", args: []string{"session", "list", "--json"}, want: []string{"sessions"}},
		{name: "memory_inventory", args: []string{"memory", "status", "--json"}, want: []string{"memory"}},
	}

	for _, step := range steps {
		step := step
		t.Run(step.name, func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, step.args...)
			if got := commandExitCode(err); got != step.exitCode {
				t.Fatalf("`gormes %s` exit code = %d, want %d\nstdout=%s\nstderr=%s\nerr=%v",
					strings.Join(step.args, " "), got, step.exitCode, stdout, stderr, err)
			}
			combined := stdout + "\n" + stderr
			if strings.Contains(combined, secret) {
				t.Fatalf("`gormes %s` leaked API key in output:\n%s", strings.Join(step.args, " "), combined)
			}
			for _, want := range step.want {
				if !strings.Contains(combined, want) {
					t.Fatalf("`gormes %s` output missing %q\nstdout=%s\nstderr=%s", strings.Join(step.args, " "), want, stdout, stderr)
				}
			}
		})
	}
}

func visiblePathExists(paths [][]string, want []string) bool {
	for _, path := range paths {
		if len(path) != len(want) {
			continue
		}
		match := true
		for i := range want {
			if path[i] != want[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func firstVisiblePaths(paths [][]string, limit int) []string {
	if len(paths) < limit {
		limit = len(paths)
	}
	out := make([]string, 0, limit)
	for _, path := range paths[:limit] {
		out = append(out, strings.Join(path, " "))
	}
	return out
}

func assertFreshInstallRuntimeStoresAbsent(t *testing.T, home string) {
	t.Helper()
	for _, rel := range []string{
		"sessions.db",
		"sessions.db-shm",
		"sessions.db-wal",
		"memory.db",
		"memory.db-shm",
		"memory.db-wal",
		"kanban.db",
		"kanban.db-shm",
		"kanban.db-wal",
		filepath.Join("cron", "jobs.db"),
		filepath.Join("gateway", "status.json"),
	} {
		path := filepath.Join(home, rel)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("read-only wide e2e unexpectedly created runtime store %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}
