package gormescli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	mcpcatalog "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/catalog"
)

func TestMCPCatalogCommandJSONListsEmbeddedApprovedEntries(t *testing.T) {
	cmd := NewMCPCommand(testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "catalog", "--json")
	if err != nil {
		t.Fatalf("Execute catalog --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var report struct {
		Build   BuildProvenance `json:"build"`
		Action  string          `json:"action"`
		Count   int             `json:"count"`
		Entries []struct {
			Name      string `json:"name"`
			Transport string `json:"transport"`
			Auth      string `json:"auth"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("catalog JSON invalid: %v\n%s", err, stdout)
	}
	if report.Build.Version != testMCPVersion || report.Action != "mcp_catalog" || report.Count != 3 {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Entries) != 3 || report.Entries[0].Name != "linear" || report.Entries[1].Name != "n8n" || report.Entries[2].Name != "unreal-engine" {
		t.Fatalf("entries = %+v", report.Entries)
	}
	if report.Entries[0].Transport != "http" || report.Entries[0].Auth != "oauth" {
		t.Fatalf("linear entry = %+v", report.Entries[0])
	}
}

func TestMCPCatalogCommandTextUsesGormesGuidance(t *testing.T) {
	t.Setenv("HERMES_HOME", t.TempDir())
	t.Setenv("MCP_PRIVATE_TOKEN", "must-not-leak")
	cmd := NewMCPCommand(testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "catalog")
	if err != nil {
		t.Fatalf("Execute catalog: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Approved MCP catalog (3)", "linear", "n8n", "unreal-engine", "http", "stdio", "gormes mcp install <name>"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(strings.ToLower(stdout+stderr), "hermes") || strings.Contains(stdout+stderr, "must-not-leak") || strings.Contains(stdout, "gormes mcp add <name>") {
		t.Fatalf("catalog output contains stale branding/guidance or inherited secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestMCPCatalogCommandEmptyCatalogSucceedsExplicitly(t *testing.T) {
	opts := testMCPCommandOptions()
	opts.Catalog = func() mcpcatalog.Catalog { return mcpcatalog.Load(fstest.MapFS{}) }
	cmd := NewMCPCommand(opts)
	stdout, stderr, err := executeMCPCommandForTest(cmd, "catalog")
	if err != nil {
		t.Fatalf("Execute empty catalog: %v", err)
	}
	if strings.TrimSpace(stdout) != "No approved MCP catalog entries are available." || strings.TrimSpace(stderr) != "" {
		t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestMCPCatalogCommandSurfacesFutureManifestDiagnostic(t *testing.T) {
	opts := testMCPCommandOptions()
	opts.Catalog = func() mcpcatalog.Catalog {
		return mcpcatalog.Load(fstest.MapFS{
			"future/manifest.yaml": {Data: []byte(`
manifest_version: 99
name: future
description: Future
transport: {type: http, url: https://example.com/mcp}
`)},
		})
	}
	cmd := NewMCPCommand(opts)
	stdout, _, err := executeMCPCommandForTest(cmd, "catalog", "--json")
	if err != nil {
		t.Fatalf("Execute future catalog: %v", err)
	}
	var report struct {
		Count       int `json:"count"`
		Diagnostics []struct {
			Entry string `json:"entry"`
			Kind  string `json:"kind"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("JSON invalid: %v\n%s", err, stdout)
	}
	if report.Count != 0 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Entry != "future" || report.Diagnostics[0].Kind != "future_manifest" {
		t.Fatalf("report = %+v", report)
	}
}

func TestMCPInstallCatalogHTTPWritesNativeConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "profile", "config.toml")
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	cmd := NewMCPCommand(opts)

	stdout, stderr, err := executeMCPCommandForTest(cmd, "install", "linear", "--json")
	if err != nil {
		t.Fatalf("Execute install linear --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var report struct {
		Build     BuildProvenance `json:"build"`
		Action    string          `json:"action"`
		Name      string          `json:"name"`
		Evidence  string          `json:"evidence"`
		Transport string          `json:"transport"`
		Auth      string          `json:"auth"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("install JSON invalid: %v\n%s", err, stdout)
	}
	if report.Build.Version != testMCPVersion || report.Action != "mcp_install" || report.Name != "linear" || report.Evidence != "installed" || report.Transport != "http" || report.Auth != "oauth" {
		t.Fatalf("report = %+v", report)
	}
	doc, err := configwriter.ReadTOMLDoc(configPath)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	servers, ok := doc["mcp_servers"].(map[string]any)
	if !ok {
		t.Fatalf("mcp_servers = %#v", doc["mcp_servers"])
	}
	linear, ok := servers["linear"].(map[string]any)
	if !ok {
		t.Fatalf("linear = %#v", servers["linear"])
	}
	if linear["url"] != "https://mcp.linear.app/mcp" || linear["auth"] != "oauth" || linear["enabled"] != true {
		t.Fatalf("linear config = %#v", linear)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %o, want 600", got)
	}
}

func TestMCPInstallUnrealEngineWritesNoAuthMarker(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	cmd := NewMCPCommand(opts)
	stdout, stderr, err := executeMCPCommandForTest(cmd, "install", "official/unreal-engine", "--json")
	if err != nil {
		t.Fatalf("Execute install unreal-engine: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	doc, err := configwriter.ReadTOMLDoc(configPath)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	unreal, _ := servers["unreal-engine"].(map[string]any)
	if unreal["url"] != "http://127.0.0.1:8000/mcp" || unreal["enabled"] != true {
		t.Fatalf("unreal config = %#v", unreal)
	}
	if _, exists := unreal["auth"]; exists {
		t.Fatalf("unreal config unexpectedly has auth: %#v", unreal)
	}
}

func TestMCPInstallUnknownEntryDoesNotCreateConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	cmd := NewMCPCommand(opts)
	stdout, _, err := executeMCPCommandForTest(cmd, "install", "missing", "--json")
	if err == nil {
		t.Fatalf("install missing succeeded: %s", stdout)
	}
	var report struct {
		Evidence string `json:"evidence"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "not_found" {
		t.Fatalf("report=%+v jsonErr=%v stdout=%s", report, jsonErr, stdout)
	}
	if _, statErr := os.Stat(configPath); !os.IsNotExist(statErr) {
		t.Fatalf("config stat err = %v, want not-exist", statErr)
	}
}

func TestMCPInstallRejectsExtendedEntryWithoutConfigMutation(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	original := []byte("[hermes]\nmodel = 'keep-me'\n")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	cmd := NewMCPCommand(opts)

	stdout, stderr, err := executeMCPCommandForTest(cmd, "install", "n8n", "--json")
	if err == nil {
		t.Fatalf("Execute install n8n succeeded; stdout=%s stderr=%s", stdout, stderr)
	}
	var report struct {
		Name     string `json:"name"`
		Evidence string `json:"evidence"`
		Error    string `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("install rejection JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if report.Name != "n8n" || report.Evidence != "extended_install_required" || report.Error == "" {
		t.Fatalf("report = %+v", report)
	}
	after, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if string(after) != string(original) {
		t.Fatalf("config changed on rejected install:\n%s", after)
	}
	if strings.Contains(strings.ToLower(stdout+stderr+err.Error()), "hermes") {
		t.Fatalf("rejection leaked stale branding: stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestMCPInstallRejectsMalformedConfigWithoutLeak(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "private-profile", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("mcp_servers = 'private-secret'\n"), 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	cmd := NewMCPCommand(opts)
	stdout, stderr, err := executeMCPCommandForTest(cmd, "install", "linear", "--json")
	if err == nil {
		t.Fatalf("install succeeded: %s", stdout)
	}
	var report struct {
		Evidence string `json:"evidence"`
		Error    string `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if report.Evidence != "config_rejected" || report.Error != "MCP configuration could not be updated" {
		t.Fatalf("report = %+v", report)
	}
	combined := stdout + stderr + err.Error()
	if strings.Contains(combined, configPath) || strings.Contains(combined, "private-secret") {
		t.Fatalf("output leaked config detail: %s", combined)
	}
}

func TestMCPInstallReinstallPreservesToolSelectionAndOtherConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[hermes]
model = "keep-me"

[mcp_servers.linear]
url = "https://old.invalid/mcp"
enabled = false

[mcp_servers.linear.tools]
include = ["search", "create"]
`), 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	cmd := NewMCPCommand(opts)
	stdout, stderr, err := executeMCPCommandForTest(cmd, "install", "official/linear")
	if err != nil {
		t.Fatalf("Execute reinstall: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(strings.ToLower(stdout+stderr), "hermes") {
		t.Fatalf("output contains stale branding: stdout=%s stderr=%s", stdout, stderr)
	}
	doc, err := configwriter.ReadTOMLDoc(configPath)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	hermes, _ := doc["hermes"].(map[string]any)
	if hermes["model"] != "keep-me" {
		t.Fatalf("unrelated config changed: %#v", hermes)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	linear, _ := servers["linear"].(map[string]any)
	tools, _ := linear["tools"].(map[string]any)
	include, _ := tools["include"].([]any)
	if len(include) != 2 || include[0] != "search" || include[1] != "create" {
		t.Fatalf("tools.include = %#v", tools["include"])
	}
	if linear["url"] != "https://mcp.linear.app/mcp" || linear["enabled"] != true {
		t.Fatalf("linear config = %#v", linear)
	}
}

func TestMCPCommandRegistersConcreteCatalogSubcommand(t *testing.T) {
	cmd := NewMCPCommand(testMCPCommandOptions())
	child, _, err := cmd.Find([]string{"catalog"})
	if err != nil {
		t.Fatalf("Find catalog: %v", err)
	}
	if child == nil || child.Use != "catalog" {
		t.Fatalf("child = %#v, want concrete catalog command", child)
	}
}
