package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginsListCommand_RendersDiscoveredPlugins(t *testing.T) {
	dataHome := t.TempDir()
	configHome := filepath.Join(dataHome, "config")
	cwd := t.TempDir()

	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HERMES_ENABLE_PROJECT_PLUGINS", "true")
	t.Setenv("PWD", cwd)
	t.Chdir(cwd)

	if err := os.MkdirAll(filepath.Join(configHome, "gormes"), 0o755); err != nil {
		t.Fatalf("MkdirAll(config): %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "gormes", "config.toml"), []byte(`
[plugins]
disabled = ["team-tools"]

[memory]
provider = "memx"

[context]
engine = "compressy"
`), 0o644); err != nil {
		t.Fatalf("WriteFile(config.toml): %v", err)
	}

	writePluginManifestCLI(t, filepath.Join(cwd, "plugins", "calculator"), `
name: calculator
version: 1.0.0
`)
	writePluginManifestCLI(t, filepath.Join(dataHome, "gormes", "plugins", "weather"), `
name: weather
version: 1.0.0
requires_env:
  - WEATHER_API_KEY
`)
	writePluginManifestCLI(t, filepath.Join(cwd, ".hermes", "plugins", "team-tools"), `
name: team-tools
version: 1.0.0
`)
	writePluginManifestCLI(t, filepath.Join(cwd, "plugins", "memory", "memx"), `
name: memx
version: 1.0.0
`)
	writePluginManifestCLI(t, filepath.Join(cwd, "plugins", "context_engine", "compressy"), `
name: compressy
version: 1.0.0
`)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"plugins", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"name\tkind\tsource\tstate\tdetails",
		"calculator\tgeneral\tbundled\tenabled\tready",
		"weather\tgeneral\tuser\tenabled\tmissing env: WEATHER_API_KEY",
		"team-tools\tgeneral\tproject\tdisabled\tready",
		"memx\tmemory_provider\tbundled\tselected\tready",
		"compressy\tcontext_engine\tbundled\tselected\tready",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want substring %q", out, want)
		}
	}
}

func writePluginManifestCLI(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
