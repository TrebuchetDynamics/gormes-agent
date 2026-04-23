package plugins

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadManifest_ParsesManifestMetadata(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, `
name: calculator
version: 1.0.0
description: Math helper
author: Trebuchet Dynamics
provides_tools:
  - calculate
  - unit_convert
provides_hooks:
  - post_tool_call
provides_skills:
  - calculator-workflow
provides_commands:
  - calc
requires_env:
  - WEATHER_API_KEY
  - name: UNITS_API_KEY
    description: Needed for premium unit conversions
    url: https://example.com/units
    secret: true
`)

	got, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if got.Name != "calculator" {
		t.Fatalf("Name = %q, want %q", got.Name, "calculator")
	}
	if got.Version != "1.0.0" {
		t.Fatalf("Version = %q, want %q", got.Version, "1.0.0")
	}
	if got.Description != "Math helper" {
		t.Fatalf("Description = %q, want %q", got.Description, "Math helper")
	}
	if got.Author != "Trebuchet Dynamics" {
		t.Fatalf("Author = %q, want %q", got.Author, "Trebuchet Dynamics")
	}
	if got.Kind != KindGeneral {
		t.Fatalf("Kind = %q, want %q", got.Kind, KindGeneral)
	}
	if got.RootDir != dir {
		t.Fatalf("RootDir = %q, want %q", got.RootDir, dir)
	}
	if got.ManifestPath != filepath.Join(dir, "plugin.yaml") {
		t.Fatalf("ManifestPath = %q, want %q", got.ManifestPath, filepath.Join(dir, "plugin.yaml"))
	}

	if !reflect.DeepEqual(got.ProvidesTools, []string{"calculate", "unit_convert"}) {
		t.Fatalf("ProvidesTools = %#v", got.ProvidesTools)
	}
	if !reflect.DeepEqual(got.ProvidesHooks, []string{"post_tool_call"}) {
		t.Fatalf("ProvidesHooks = %#v", got.ProvidesHooks)
	}
	if !reflect.DeepEqual(got.ProvidesSkills, []string{"calculator-workflow"}) {
		t.Fatalf("ProvidesSkills = %#v", got.ProvidesSkills)
	}
	if !reflect.DeepEqual(got.ProvidesCommands, []string{"calc"}) {
		t.Fatalf("ProvidesCommands = %#v", got.ProvidesCommands)
	}

	wantEnv := []EnvRequirement{
		{Name: "WEATHER_API_KEY"},
		{
			Name:        "UNITS_API_KEY",
			Description: "Needed for premium unit conversions",
			URL:         "https://example.com/units",
			Secret:      true,
		},
	}
	if !reflect.DeepEqual(got.RequiresEnv, wantEnv) {
		t.Fatalf("RequiresEnv = %#v, want %#v", got.RequiresEnv, wantEnv)
	}
}

func TestLoadManifest_RejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, `
version: 1.0.0
requires_env:
  - name: ""
`)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("LoadManifest() error = nil, want invalid-manifest error")
	}
	for _, want := range []string{"name is required", "requires_env"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadManifest() error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestDiscover_ScansPluginRootsDeterministically(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, filepath.Join(root, "beta-dir"), `
name: beta
version: 2.0.0
`)
	writePluginManifest(t, filepath.Join(root, "alpha-dir"), `
name: alpha
version: 1.0.0
`)
	if err := os.WriteFile(filepath.Join(root, "README.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("WriteFile(ignore): %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "empty-dir"), 0o755); err != nil {
		t.Fatalf("MkdirAll(empty-dir): %v", err)
	}

	got, err := Discover(root, KindMemoryProvider)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Discover()) = %d, want 2", len(got))
	}
	if got[0].Name != "alpha" || got[1].Name != "beta" {
		t.Fatalf("Discover() names = [%q %q], want [alpha beta]", got[0].Name, got[1].Name)
	}
	for _, manifest := range got {
		if manifest.Kind != KindMemoryProvider {
			t.Fatalf("manifest.Kind = %q, want %q", manifest.Kind, KindMemoryProvider)
		}
		if manifest.RootDir == "" || manifest.ManifestPath == "" {
			t.Fatalf("discovered manifest missing paths: %#v", manifest)
		}
	}
}

func writePluginManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
