package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/plugins"
	"github.com/spf13/cobra"
)

func TestPluginsCommandListInstallUpdateRemove(t *testing.T) {
	root := t.TempDir()
	source := writeCommandPluginFixture(t, "demo-plugin", map[string]string{
		"plugin.yaml":         "name: demo-plugin\nversion: 1.2.3\ndescription: Demo plugin\nrequires_env: [DEMO_API_KEY]\n",
		"config.yaml.example": "enabled: true\n",
	})
	if err := os.Mkdir(filepath.Join(source, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := &commandPluginEnv{values: map[string]string{}}
	manager := plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &commandPluginRunner{cloneSource: source, pullOutput: "Already up to date."},
		Env:      env,
		Prompt: func(name string) (string, bool, error) {
			if name != "DEMO_API_KEY" {
				t.Fatalf("prompted unexpected env %q", name)
			}
			return "demo-secret", true, nil
		},
	})

	stdout, stderr, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "list")
	if err != nil {
		t.Fatalf("list empty: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "No plugins installed") {
		t.Fatalf("empty list output = %q", stdout)
	}

	stdout, stderr, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "install", "owner/demo-plugin", "--enable")
	if err != nil {
		t.Fatalf("install: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "installed demo-plugin") || strings.Contains(stdout+stderr, "demo-secret") {
		t.Fatalf("install output missing install evidence or leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if env.values["DEMO_API_KEY"] != "demo-secret" {
		t.Fatalf("required env not saved: %+v", env.values)
	}

	stdout, stderr, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "ls")
	if err != nil {
		t.Fatalf("list installed: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "demo-plugin") || !strings.Contains(stdout, "enabled") || !strings.Contains(stdout, "1.2.3") {
		t.Fatalf("list output missing enabled plugin:\n%s", stdout)
	}

	stdout, stderr, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "update", "demo-plugin")
	if err != nil {
		t.Fatalf("update: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "updated demo-plugin") {
		t.Fatalf("update output = %q", stdout)
	}

	stdout, stderr, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "disable", "demo-plugin")
	if err != nil {
		t.Fatalf("disable: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "disabled demo-plugin") {
		t.Fatalf("disable output = %q", stdout)
	}

	stdout, stderr, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "enable", "demo-plugin")
	if err != nil {
		t.Fatalf("enable: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "enabled demo-plugin") {
		t.Fatalf("enable output = %q", stdout)
	}

	stdout, stderr, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "rm", "demo-plugin")
	if err != nil {
		t.Fatalf("remove: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "removed demo-plugin") {
		t.Fatalf("remove output = %q", stdout)
	}
}

func TestPluginsCommandIdentifierSafety(t *testing.T) {
	root := t.TempDir()
	badSource := writeCommandPluginFixture(t, "bad-plugin", map[string]string{
		"plugin.yaml": "name: .\nversion: 1.0.0\n",
	})
	manager := plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &commandPluginRunner{cloneSource: badSource},
	})
	stdout, stderr, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "install", "owner/bad-plugin", "--force")
	if err == nil {
		t.Fatalf("unsafe install returned nil error stdout=%s stderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error()+stdout+stderr, "plugin_cli_invalid_name") {
		t.Fatalf("unsafe install output missing typed evidence:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	_, _, err = executePluginsCommandForTest(newPluginsCommandWithManager(manager), "remove", "../escape")
	if err == nil || !strings.Contains(err.Error(), "plugin_cli_invalid_name") {
		t.Fatalf("unsafe remove err = %v, want plugin_cli_invalid_name", err)
	}
}

func executePluginsCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

type commandPluginRunner struct {
	cloneSource string
	pullOutput  string
}

func (r *commandPluginRunner) Clone(_, dst string) error {
	return copyCommandPluginDir(r.cloneSource, dst)
}

func (r *commandPluginRunner) Pull(string) (string, error) {
	return r.pullOutput, nil
}

type commandPluginEnv struct {
	values map[string]string
}

func (e *commandPluginEnv) Lookup(name string) (string, bool) {
	value, ok := e.values[name]
	return value, ok
}

func (e *commandPluginEnv) Save(name, value string) error {
	if e.values == nil {
		e.values = make(map[string]string)
	}
	e.values[name] = value
	return nil
}

func writeCommandPluginFixture(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func copyCommandPluginDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}
