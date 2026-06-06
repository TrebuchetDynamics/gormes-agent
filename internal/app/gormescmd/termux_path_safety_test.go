package gormescmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestTermuxRuntimeCommandsKeepStateUnderConfiguredHome(t *testing.T) {
	env := setupTermuxPathSafetyEnv(t)

	pathOut, pathErr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "config", "path", "--json")
	if err != nil {
		t.Fatalf("config path: %v\nstdout=%s\nstderr=%s", err, pathOut, pathErr)
	}
	assertTermuxCommandJSONPath(t, pathOut, env.gormesHome)

	envOut, envErr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "config", "env-path", "--json")
	if err != nil {
		t.Fatalf("config env-path: %v\nstdout=%s\nstderr=%s", err, envOut, envErr)
	}
	assertTermuxCommandJSONPath(t, envOut, env.gormesHome)

	modelOut, modelErr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "config", "set", "hermes.model", "termux-model", "--json")
	if err != nil {
		t.Fatalf("config set model: %v\nstdout=%s\nstderr=%s", err, modelOut, modelErr)
	}
	assertTermuxConfigSetPath(t, modelOut, "toml", env.gormesHome)

	secretOut, secretErr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "config", "set", "api_key", "sk-termux-redacted", "--json")
	if err != nil {
		t.Fatalf("config set api_key: %v\nstdout=%s\nstderr=%s", err, secretOut, secretErr)
	}
	assertTermuxConfigSetPath(t, secretOut, "dotenv", env.gormesHome)

	doctorOut, doctorErr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "doctor", "--offline", "--json")
	if err != nil {
		t.Fatalf("doctor --offline --json: %v\nstdout=%s\nstderr=%s", err, doctorOut, doctorErr)
	}
	assertTermuxDoctorCheckPresent(t, doctorOut)

	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("Close memory store: %v", err)
	}

	gonchoOut, gonchoErr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "goncho", "doctor", "--json", "--peer=termux-user", "--session=termux:1")
	if err != nil {
		t.Fatalf("goncho doctor --json: %v\nstdout=%s\nstderr=%s", err, gonchoOut, gonchoErr)
	}
	assertTermuxGonchoPaths(t, gonchoOut, env.gormesHome)

	for name, path := range map[string]string{
		"config.toml": config.ConfigPath(),
		".env":        config.EnvPath(),
		"memory.db":   config.MemoryDBPath(),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s should exist under configured GORMES_HOME at %q: %v", name, path, err)
		}
	}

	for label, out := range map[string]string{
		"config path stdout":    pathOut,
		"config env stdout":     envOut,
		"config set model out":  modelOut,
		"config set secret out": secretOut,
		"doctor stdout":         doctorOut,
		"doctor stderr":         doctorErr,
		"goncho stdout":         gonchoOut,
		"goncho stderr":         gonchoErr,
	} {
		assertNoDesktopPathMarkers(t, label, out)
	}
	assertTermuxPrefixUnchanged(t, env)
	assertTermuxCreatedFilesStayInAllowedRoots(t, env)
}

func TestTermuxDefaultRegistryPathsStayUnderConfiguredHome(t *testing.T) {
	env := setupTermuxPathSafetyEnv(t)

	paths := map[string]string{
		"browser artifacts": gormescli.DefaultBrowserArtifactDir(),
		"web auth store":    gormescli.DefaultWebAuthStorePath(),
		"memory tool dir":   gormescli.DefaultMemoryToolDir(),
		"audio cache":       gormescli.DefaultAudioCacheDir(),
		"whisper cache":     gormescli.DefaultTranscriptionCacheDir(),
	}
	for name, got := range paths {
		assertPathWithinRoot(t, name, got, env.gormesHome)
		assertPathOutsideRoot(t, name, got, env.prefix)
		assertNoDesktopPathMarkers(t, name, got)
	}
}

type termuxPathSafetyEnv struct {
	root        string
	home        string
	gormesHome  string
	xdgConfig   string
	xdgData     string
	xdgState    string
	hermesHome  string
	codexHome   string
	tmp         string
	termuxRoot  string
	prefix      string
	binDir      string
	prefixFiles []string
}

func setupTermuxPathSafetyEnv(t *testing.T) termuxPathSafetyEnv {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "gormes-termux-cmd-")
	if err != nil {
		t.Fatalf("create termux command fixture root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	env := termuxPathSafetyEnv{
		root:       root,
		home:       filepath.Join(root, "home"),
		gormesHome: filepath.Join(root, "gormes-home"),
		xdgConfig:  filepath.Join(root, "xdg-config"),
		xdgData:    filepath.Join(root, "xdg-data"),
		xdgState:   filepath.Join(root, "xdg-state"),
		hermesHome: filepath.Join(root, "hermes-home"),
		codexHome:  filepath.Join(root, "codex-home"),
		tmp:        filepath.Join(root, "tmp"),
		termuxRoot: filepath.Join(root, "com.termux"),
		prefix:     filepath.Join(root, "com.termux", "files", "usr"),
	}
	env.binDir = filepath.Join(env.prefix, "bin")
	for _, dir := range []string{env.home, env.gormesHome, env.xdgConfig, env.xdgData, env.xdgState, env.hermesHome, env.codexHome, env.tmp, env.binDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, name := range []string{"termux-wake-lock", "termux-notification"} {
		path := filepath.Join(env.binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		env.prefixFiles = append(env.prefixFiles, path)
	}
	sort.Strings(env.prefixFiles)

	t.Setenv("TERMUX_VERSION", "0.119.0")
	t.Setenv("PREFIX", env.prefix)
	t.Setenv("HOME", env.home)
	t.Setenv("GORMES_HOME", env.gormesHome)
	t.Setenv("XDG_CONFIG_HOME", env.xdgConfig)
	t.Setenv("XDG_DATA_HOME", env.xdgData)
	t.Setenv("XDG_STATE_HOME", env.xdgState)
	t.Setenv("HERMES_HOME", env.hermesHome)
	t.Setenv("CODEX_HOME", env.codexHome)
	t.Setenv("TMPDIR", env.tmp)
	t.Setenv("PATH", env.binDir)
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_INFERENCE_MODEL", "")
	t.Setenv("GORMES_INFERENCE_PROVIDER", "")
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")
	t.Setenv("GORMES_STT_CACHE_DIR", "")
	return env
}

func assertTermuxCommandJSONPath(t *testing.T, raw, root string) {
	t.Helper()
	var got struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("invalid path JSON: %v\n%s", err, raw)
	}
	assertPathWithinRoot(t, "json path", got.Path, root)
}

func assertTermuxConfigSetPath(t *testing.T, raw, wantTarget, root string) {
	t.Helper()
	var got struct {
		Target string `json:"target"`
		Path   string `json:"path"`
		Secret bool   `json:"secret"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("invalid config set JSON: %v\n%s", err, raw)
	}
	if got.Target != wantTarget {
		t.Fatalf("target = %q, want %q\n%s", got.Target, wantTarget, raw)
	}
	if wantTarget == "dotenv" && !got.Secret {
		t.Fatalf("dotenv config set should be marked secret:\n%s", raw)
	}
	assertPathWithinRoot(t, "config set path", got.Path, root)
}

func assertTermuxDoctorCheckPresent(t *testing.T, raw string) {
	t.Helper()
	var got struct {
		Checks []struct {
			Name string `json:"name"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("invalid doctor JSON: %v\n%s", err, raw)
	}
	for _, check := range got.Checks {
		if check.Name == "Termux runtime" {
			return
		}
	}
	t.Fatalf("doctor JSON missing Termux runtime check:\n%s", raw)
}

func assertTermuxGonchoPaths(t *testing.T, raw, root string) {
	t.Helper()
	var got struct {
		Config struct {
			ConfigPath    string `json:"config_path"`
			MemoryDBPath  string `json:"memory_db_path"`
			SessionDBPath string `json:"session_db_path"`
		} `json:"config"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("invalid goncho doctor JSON: %v\n%s", err, raw)
	}
	for name, path := range map[string]string{
		"goncho config path":     got.Config.ConfigPath,
		"goncho memory db path":  got.Config.MemoryDBPath,
		"goncho session db path": got.Config.SessionDBPath,
	} {
		assertPathWithinRoot(t, name, path, root)
	}
}

func assertTermuxPrefixUnchanged(t *testing.T, env termuxPathSafetyEnv) {
	t.Helper()
	var got []string
	if err := filepath.WalkDir(env.prefix, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		got = append(got, path)
		return nil
	}); err != nil {
		t.Fatalf("walk prefix: %v", err)
	}
	sort.Strings(got)
	if strings.Join(got, "\n") != strings.Join(env.prefixFiles, "\n") {
		t.Fatalf("runtime commands changed Termux prefix files\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(env.prefixFiles, "\n"))
	}
}

func assertTermuxCreatedFilesStayInAllowedRoots(t *testing.T, env termuxPathSafetyEnv) {
	t.Helper()
	allowed := []string{env.home, env.gormesHome, env.xdgConfig, env.xdgData, env.xdgState, env.hermesHome, env.codexHome, env.tmp, env.termuxRoot}
	if err := filepath.WalkDir(env.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == env.root {
			return nil
		}
		for _, root := range allowed {
			if sameOrUnderPath(path, root) {
				return nil
			}
		}
		return fmt.Errorf("created path outside allowed roots: %s", path)
	}); err != nil {
		t.Fatalf("walk fixture root: %v", err)
	}
}

func assertPathWithinRoot(t *testing.T, name, path, root string) {
	t.Helper()
	if !sameOrUnderPath(path, root) {
		t.Fatalf("%s path = %q, want under %q", name, path, root)
	}
}

func assertPathOutsideRoot(t *testing.T, name, path, root string) {
	t.Helper()
	if sameOrUnderPath(path, root) {
		t.Fatalf("%s path = %q, must not be under %q", name, path, root)
	}
}

func assertNoDesktopPathMarkers(t *testing.T, label, value string) {
	t.Helper()
	for _, forbidden := range []string{"/home/xel", "workspace-mineru", "workspace-gormes"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("%s contains desktop checkout marker %q:\n%s", label, forbidden, value)
		}
	}
}

func sameOrUnderPath(path, root string) bool {
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
