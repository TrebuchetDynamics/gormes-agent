package main

import (
	"bytes"
	"encoding/json"
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

// TestPluginsCommandList_JSONEmitsStructuredArray proves
// `gormes plugins list --json` returns a parseable
// `{build, plugins: [{name, version, status, source, path, description}]}`
// document so fleet automation can audit plugin state across machines
// without scraping tab-separated columns.
func TestPluginsCommandList_JSONEmitsStructuredArray(t *testing.T) {
	root := t.TempDir()
	source := writeCommandPluginFixture(t, "audit-plugin", map[string]string{
		"plugin.yaml": "name: audit-plugin\nversion: 7.7.7\ndescription: For audit tests\n",
	})
	if err := os.Mkdir(filepath.Join(source, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager := plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &commandPluginRunner{cloneSource: source, pullOutput: "Already up to date."},
	})

	stdout, _, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "install", "owner/audit-plugin", "--enable")
	if err != nil {
		t.Fatalf("install: %v stdout=%s", err, stdout)
	}

	stdout, stderr, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "list", "--json")
	if err != nil {
		t.Fatalf("plugins list --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Plugins []struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Status      string `json:"status"`
			Source      string `json:"source"`
			Path        string `json:"path"`
			Description string `json:"description"`
		} `json:"plugins"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("plugins list --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if len(got.Plugins) != 1 {
		t.Fatalf("plugins len = %d, want 1; got %+v", len(got.Plugins), got.Plugins)
	}
	p := got.Plugins[0]
	if p.Name != "audit-plugin" {
		t.Errorf("name = %q, want audit-plugin", p.Name)
	}
	if p.Version != "7.7.7" {
		t.Errorf("version = %q, want 7.7.7", p.Version)
	}
	if p.Status != "enabled" {
		t.Errorf("status = %q, want enabled", p.Status)
	}
	if p.Path == "" {
		t.Errorf("path must be populated")
	}
}

// TestPluginsCommandList_JSONEmptyEmitsArray proves the empty-state
// JSON is `{build, plugins: []}` rather than null. Operator scripts
// iterating the array on every host need a stable empty-list shape.
func TestPluginsCommandList_JSONEmptyEmitsArray(t *testing.T) {
	root := t.TempDir()
	manager := plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &commandPluginRunner{},
	})
	stdout, _, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "list", "--json")
	if err != nil {
		t.Fatalf("plugins list --json (empty): %v", err)
	}
	var got struct {
		Plugins []struct{} `json:"plugins"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Plugins == nil {
		t.Errorf("plugins must be a JSON array (possibly empty), not null")
	}
	if !strings.Contains(stdout, "\"plugins\": []") {
		t.Errorf("empty plugins must marshal as `[]`, not null:\n%s", stdout)
	}
}

// TestPluginsCommand_InstallJSONEmitsStructuredOutcome proves
// `gormes plugins install <id> --json` returns
// `{build, action: "installed", name, path, enabled}` so fleet
// automation provisioning plugins across machines can record the
// resulting on-disk path AND the enable state in one parseable
// document. Without this, scripts have to scrape "installed X\n
// enabled Y" two-line prose.
func TestPluginsCommand_InstallJSONEmitsStructuredOutcome(t *testing.T) {
	root := t.TempDir()
	source := writeCommandPluginFixture(t, "install-json-plugin", map[string]string{
		"plugin.yaml": "name: install-json-plugin\nversion: 9.9.9\ndescription: Install JSON test\n",
	})
	if err := os.Mkdir(filepath.Join(source, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager := plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &commandPluginRunner{cloneSource: source},
	})

	stdout, stderr, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "install", "owner/install-json-plugin", "--enable", "--json")
	if err != nil {
		t.Fatalf("plugins install --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action  string `json:"action"`
		Name    string `json:"name"`
		Path    string `json:"path"`
		Enabled bool   `json:"enabled"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("plugins install --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "installed" {
		t.Errorf("action = %q, want %q", got.Action, "installed")
	}
	if got.Name != "install-json-plugin" {
		t.Errorf("name = %q, want install-json-plugin", got.Name)
	}
	if !got.Enabled {
		t.Errorf("enabled must be true when --enable was passed")
	}
	if got.Path == "" {
		t.Errorf("path must be populated")
	}
}

// TestPluginsCommand_LifecycleJSONShapeIsConsistent proves
// `gormes plugins enable/disable/update/remove --json` all return a
// `{build, action, name}` document with the same shape so fleet
// automation reconciling plugin state across machines can write one
// JSON parser path. `action` is the only field that varies between
// the four lifecycle verbs.
func TestPluginsCommand_LifecycleJSONShapeIsConsistent(t *testing.T) {
	root := t.TempDir()
	source := writeCommandPluginFixture(t, "lifecycle-plugin", map[string]string{
		"plugin.yaml": "name: lifecycle-plugin\nversion: 1.0.0\ndescription: Lifecycle JSON tests\n",
	})
	if err := os.Mkdir(filepath.Join(source, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager := plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &commandPluginRunner{cloneSource: source, pullOutput: "Already up to date."},
	})

	if _, _, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), "install", "owner/lifecycle-plugin"); err != nil {
		t.Fatalf("install: %v", err)
	}

	verbs := []struct {
		args   []string
		action string
	}{
		{[]string{"enable", "lifecycle-plugin", "--json"}, "enabled"},
		{[]string{"disable", "lifecycle-plugin", "--json"}, "disabled"},
		{[]string{"update", "lifecycle-plugin", "--json"}, "updated"},
		{[]string{"remove", "lifecycle-plugin", "--json"}, "removed"},
	}
	for _, tt := range verbs {
		t.Run(tt.action, func(t *testing.T) {
			stdout, stderr, err := executePluginsCommandForTest(newPluginsCommandWithManager(manager), tt.args...)
			if err != nil {
				t.Fatalf("plugins %v: %v\nstdout=%s\nstderr=%s", tt.args, err, stdout, stderr)
			}
			var got struct {
				Build struct {
					Version string `json:"version"`
				} `json:"build"`
				Action string `json:"action"`
				Name   string `json:"name"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
				t.Fatalf("plugins %v --json must be valid JSON: %v\nstdout=%s", tt.args, jsonErr, stdout)
			}
			if got.Build.Version != Version {
				t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
			}
			if got.Action != tt.action {
				t.Errorf("action = %q, want %q", got.Action, tt.action)
			}
			if got.Name != "lifecycle-plugin" {
				t.Errorf("name = %q, want lifecycle-plugin", got.Name)
			}
		})
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
