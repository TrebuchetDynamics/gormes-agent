package doctor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// TermuxRuntimeOptions contains injectable probes for the Termux doctor check.
type TermuxRuntimeOptions struct {
	Env      map[string]string
	LookPath func(string) (string, error)
}

// CheckTermuxRuntime reports Android/Termux-specific runtime readiness without
// requiring network access, root, or live Android APIs.
func CheckTermuxRuntime(opts TermuxRuntimeOptions) CheckResult {
	env := opts.Env
	if env == nil {
		env = currentEnvMap()
	}
	if !IsTermuxEnvironment(env) {
		return CheckResult{
			Name:    "Termux runtime",
			Status:  StatusSkip,
			Summary: "not running under Termux",
		}
	}

	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	prefix := strings.TrimSpace(env["PREFIX"])
	home := strings.TrimSpace(env["HOME"])
	version := strings.TrimSpace(env["TERMUX_VERSION"])
	if version == "" {
		version = "unknown"
	}

	status := StatusPass
	items := []ItemInfo{{
		Name:   "environment",
		Status: StatusPass,
		Note:   "evidence=termux_detected version=" + version + compactEnvPathNote("prefix", prefix) + compactEnvPathNote("home", home),
	}}

	installDir := ""
	if prefix != "" {
		installDir = filepath.Join(prefix, "bin")
	}
	if installDir != "" && pathContainsDir(env["PATH"], installDir) {
		items = append(items, ItemInfo{
			Name:   "command_path",
			Status: StatusPass,
			Note:   "desktop-like command path ready install_dir=" + installDir,
		})
	} else {
		status = StatusWarn
		note := "PREFIX/bin missing from PATH; install.sh publishes gormes into $PREFIX/bin"
		if installDir != "" {
			note = "install_dir=" + installDir + " not present on PATH; install.sh publishes gormes there"
		}
		items = append(items, ItemInfo{Name: "command_path", Status: StatusWarn, Note: note})
	}

	if _, err := lookPath("tmux"); err == nil {
		items = append(items, ItemInfo{
			Name:   "tmux",
			Status: StatusPass,
			Note:   "tmux available for foreground gateway sessions",
		})
	} else {
		status = StatusWarn
		items = append(items, ItemInfo{
			Name:   "tmux",
			Status: StatusWarn,
			Note:   "tmux missing; gateway foreground mode still works from the current shell; run `pkg install tmux` for resilient foreground sessions",
		})
	}

	missingAPI := missingTermuxAPICommands(lookPath)
	if len(missingAPI) == 0 {
		items = append(items, ItemInfo{
			Name:   "termux_api",
			Status: StatusPass,
			Note:   "termux-api commands available: termux-wake-lock, termux-notification",
		})
	} else {
		status = StatusWarn
		items = append(items, ItemInfo{
			Name:   "termux_api",
			Status: StatusWarn,
			Note:   "optional termux-api commands missing: " + strings.Join(missingAPI, ",") + "; install Termux:API for wake-lock and notification integration",
		})
	}

	storageIssues := termuxStoragePathIssues(env)
	if len(storageIssues) > 0 {
		status = StatusWarn
		items = append(items, ItemInfo{
			Name:   "storage_paths",
			Status: StatusWarn,
			Note:   "paths may be on external storage with Android restrictions: " + strings.Join(storageIssues, "; ") + "; use internal Termux paths under $HOME or $PREFIX",
		})
	} else {
		items = append(items, ItemInfo{
			Name:   "storage_paths",
			Status: StatusPass,
			Note:   "runtime paths use internal Termux storage",
		})
	}

	status = StatusWarn
	items = append(items,
		ItemInfo{
			Name:   "android_lifecycle",
			Status: StatusWarn,
			Note:   "run long gateway sessions inside tmux; use termux-wake-lock and disable Android battery optimization for best-effort background survival",
		},
		ItemInfo{
			Name:   "local_boundaries",
			Status: StatusPass,
			Note:   "CLI/TUI, provider calls, SQLite/Goncho, and gateway foreground mode are local; Docker, heavy browser automation, GPU/local LLM, and large builds should be remote/degraded",
		},
	)

	return CheckResult{
		Name:    "Termux runtime",
		Status:  status,
		Summary: "Termux detected; desktop-like Gormes CLI supported with Android lifecycle caveats",
		Items:   items,
	}
}

func currentEnvMap() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

// IsTermuxEnvironment reports whether env carries standard Termux evidence. If
// env is nil, the current process environment is used.
func IsTermuxEnvironment(env map[string]string) bool {
	if env == nil {
		env = currentEnvMap()
	}
	return isTermuxEnv(env)
}

func isTermuxEnv(env map[string]string) bool {
	if textvalue.IsNonBlank(env["TERMUX_VERSION"]) {
		return true
	}
	for _, key := range []string{"PREFIX", "HOME"} {
		if strings.Contains(env[key], "com.termux/files") {
			return true
		}
	}
	return false
}

func compactEnvPathNote(key, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return " " + key + "=" + value
}

func pathContainsDir(pathValue, want string) bool {
	if strings.TrimSpace(want) == "" {
		return false
	}
	want = filepath.Clean(want)
	for _, entry := range filepath.SplitList(pathValue) {
		if filepath.Clean(entry) == want {
			return true
		}
	}
	return false
}

func missingTermuxAPICommands(lookPath func(string) (string, error)) []string {
	missing := []string{}
	for _, name := range []string{"termux-wake-lock", "termux-notification"} {
		if _, err := lookPath(name); err != nil {
			var execErr *exec.Error
			if errors.As(err, &execErr) || errors.Is(err, exec.ErrNotFound) {
				missing = append(missing, name)
				continue
			}
			missing = append(missing, name)
		}
	}
	return missing
}

func termuxStoragePathIssues(env map[string]string) []string {
	var issues []string
	check := func(label, path string) {
		lower := strings.ToLower(path)
		if lower == "/sdcard" || strings.HasPrefix(lower, "/sdcard/") ||
			lower == "/storage" || strings.HasPrefix(lower, "/storage/") {
			issues = append(issues, fmt.Sprintf("%s=%q", label, path))
		}
	}
	check("GORMES_HOME", env["GORMES_HOME"])
	check("HOME", env["HOME"])
	check("TMPDIR", env["TMPDIR"])
	return issues
}
