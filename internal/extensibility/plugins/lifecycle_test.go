package plugins

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPluginLifecycleInstallListUpdateRemove(t *testing.T) {
	root := t.TempDir()
	source := writePluginFixture(t, "demo-plugin", map[string]string{
		"plugin.yaml": `name: demo-plugin
version: 1.2.3
description: Demo plugin
requires_env:
  - DEMO_API_KEY
optional_env:
  - DEMO_OPTIONAL
provides_tools:
  - demo_tool
`,
		"config.yaml.example": "enabled: true\n",
	})
	if err := os.Mkdir(filepath.Join(source, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	env := &fakeLifecycleEnv{values: map[string]string{
		"DEMO_OPTIONAL": "already optional",
	}}
	runner := &fakeLifecycleRunner{cloneSource: source, pullOutput: "Already up to date."}
	manager := NewLifecycleManager(LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   runner,
		Env:      env,
		Prompt: func(name string) (string, bool, error) {
			if name != "DEMO_API_KEY" {
				t.Fatalf("prompted unexpected env %q", name)
			}
			return "demo-secret", true, nil
		},
	})

	installed, err := manager.Install("owner/demo-plugin", InstallOptions{Enable: true})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if installed.Name != "demo-plugin" || installed.Path != filepath.Join(root, "plugins", "demo-plugin") {
		t.Fatalf("installed = %+v", installed)
	}
	if runner.cloneURL != "https://github.com/owner/demo-plugin.git" {
		t.Fatalf("clone URL = %q, want GitHub shorthand URL", runner.cloneURL)
	}
	if _, err := os.Stat(filepath.Join(installed.Path, "config.yaml")); err != nil {
		t.Fatalf("example file was not copied: %v", err)
	}
	if env.values["DEMO_API_KEY"] != "demo-secret" {
		t.Fatalf("required env not saved: %+v", env.values)
	}
	if env.values["DEMO_OPTIONAL"] != "already optional" {
		t.Fatalf("optional env was overwritten: %+v", env.values)
	}

	list, err := manager.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "demo-plugin" || list[0].Status != PluginLifecycleStatusEnabled {
		t.Fatalf("list = %+v, want enabled demo-plugin", list)
	}

	updated, err := manager.Update("demo-plugin")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated || runner.pullDir != installed.Path {
		t.Fatalf("update = %t pullDir=%q, want true and installed path", updated, runner.pullDir)
	}

	if err := manager.Disable("demo-plugin"); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if err := manager.Enable("demo-plugin"); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	sets, err := manager.ConfigSets()
	if err != nil {
		t.Fatalf("ConfigSets: %v", err)
	}
	if !slices.Equal(sets.Enabled, []string{"demo-plugin"}) || len(sets.Disabled) != 0 {
		t.Fatalf("sets = %+v, want demo-plugin enabled only", sets)
	}

	if err := manager.Remove("demo-plugin"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(installed.Path); !os.IsNotExist(err) {
		t.Fatalf("plugin path still exists after remove: %v", err)
	}
}

func TestPluginLifecycleRejectsUnsafeNamesAndManifestEscapes(t *testing.T) {
	root := t.TempDir()
	manager := NewLifecycleManager(LifecycleOptions{
		UserRoot: filepath.Join(root, "plugins"),
		Config:   filepath.Join(root, "config.toml"),
		Runner:   &fakeLifecycleRunner{cloneSource: root},
	})
	for _, name := range []string{"", ".", "..", "../escape", "nested/name", `nested\name`} {
		if _, err := manager.PluginPath(name); err == nil {
			t.Fatalf("PluginPath(%q) error = nil, want unsafe-name error", name)
		}
	}

	source := writePluginFixture(t, "bad-plugin", map[string]string{
		"plugin.yaml": "name: .\nversion: 1.0.0\n",
	})
	manager.runner = &fakeLifecycleRunner{cloneSource: source}
	if _, err := manager.Install("owner/bad-plugin", InstallOptions{Force: true}); err == nil || !strings.Contains(err.Error(), "plugin_cli_invalid_name") {
		t.Fatalf("Install manifest escape error = %v, want plugin_cli_invalid_name", err)
	}
	if _, err := os.Stat(filepath.Join(root, "plugins")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("plugins root was damaged: %v", err)
	}

	invalidSource := writePluginFixture(t, "invalid-plugin", map[string]string{
		"plugin.yaml": "name: invalid-plugin\n",
	})
	manager.runner = &fakeLifecycleRunner{cloneSource: invalidSource}
	if _, err := manager.Install("owner/invalid-plugin", InstallOptions{}); err == nil || !strings.Contains(err.Error(), "plugin_cli_invalid_manifest") {
		t.Fatalf("Install invalid manifest error = %v, want plugin_cli_invalid_manifest", err)
	}
	if _, err := os.Stat(filepath.Join(root, "plugins", "invalid-plugin")); !os.IsNotExist(err) {
		t.Fatalf("invalid plugin was installed despite manifest validation failure: %v", err)
	}
}

type fakeLifecycleRunner struct {
	cloneSource string
	cloneURL    string
	pullDir     string
	pullOutput  string
}

func (f *fakeLifecycleRunner) Clone(url, dst string) error {
	f.cloneURL = url
	return copyDirForLifecycleTest(f.cloneSource, dst)
}

func (f *fakeLifecycleRunner) Pull(dir string) (string, error) {
	f.pullDir = dir
	return f.pullOutput, nil
}

type fakeLifecycleEnv struct {
	values map[string]string
}

func (f *fakeLifecycleEnv) Lookup(name string) (string, bool) {
	value, ok := f.values[name]
	return value, ok
}

func (f *fakeLifecycleEnv) Save(name, value string) error {
	if f.values == nil {
		f.values = make(map[string]string)
	}
	f.values[name] = value
	return nil
}

func copyDirForLifecycleTest(src, dst string) error {
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
