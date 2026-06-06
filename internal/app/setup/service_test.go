package setup

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestParseToolSelectionResolvesIndexesKeysAndUnknownCustomToolsets(t *testing.T) {
	options := []ToolOption{
		{Key: "web", Label: "Web Search"},
		{Key: "browser", Label: "Browser Automation"},
	}

	got, err := ParseToolSelection("1,browser,custom-mcp-server", options, []string{"terminal"})
	if err != nil {
		t.Fatalf("ParseToolSelection: %v", err)
	}
	want := []string{"web", "browser", "custom-mcp-server"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("selection = %v, want %v", got, want)
	}
}

func TestParseToolSelectionInvalidIndex(t *testing.T) {
	_, err := ParseToolSelection("3", []ToolOption{{Key: "web"}}, nil)
	var invalid InvalidToolSelectionError
	if !errors.As(err, &invalid) || invalid.Token != "3" {
		t.Fatalf("err = %v, want InvalidToolSelectionError token 3", err)
	}
}

func TestWriteToolsConfigPreservesSymlink(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real-config.toml")
	linkPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(realPath, []byte("platform_toolsets = { cli = [\"terminal\"] }\n"), 0o644); err != nil {
		t.Fatalf("write real config: %v", err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err := WriteToolsConfig(linkPath, map[string]any{
		"platform_toolsets": map[string]any{"cli": []string{"web", "browser"}},
	})
	if err != nil {
		t.Fatalf("WriteToolsConfig: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat config link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config link was replaced with mode %v, want symlink preserved", info.Mode())
	}
	got, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read real config: %v", err)
	}
	if !strings.Contains(string(got), "web") || !strings.Contains(string(got), "browser") {
		t.Fatalf("real config was not updated through symlink:\n%s", got)
	}
}

func TestToolsProviderRowsAreStableAndFiltered(t *testing.T) {
	rows := ToolsProviderRows([]string{"memory", "web", "terminal"})
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.Toolset+":"+row.Kind+":"+row.Label)
	}
	want := []string{
		"web:web:Web search and extraction",
		"memory:honcho:Honcho/Goncho memory provider",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rows = %v, want %v", got, want)
	}
}

func TestRunToolsNonInteractiveListsCLIConfigurableToolsets(t *testing.T) {
	t.Setenv("HASS_TOKEN", "")
	var stdout bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := RunTools(ToolsOptions{Out: &stdout, ConfigPath: configPath, NonInteractive: true}); err != nil {
		t.Fatalf("RunTools: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Tools for CLI",
		"web", "Web Search & Scraping",
		"browser", "Browser Automation",
		"terminal", "Terminal & Processes",
		"file", "File Operations",
		"code_execution", "Code Execution",
		"vision", "Vision / Image Analysis",
		"video", "Video Analysis",
		"image_gen", "Image Generation",
		"moa", "Mixture of Agents",
		"tts", "Text-to-Speech",
		"skills", "Skills",
		"todo", "Task Planning",
		"memory", "Memory",
		"session_search", "Session Search",
		"clarify", "Clarifying Questions",
		"delegation", "Task Delegation",
		"cronjob", "Cron Jobs",
		"messaging", "Cross-Platform Messaging",
		"rl", "RL Training",
		"homeassistant", "Home Assistant",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, want := range []string{
		"[x] 🔍 Web Search & Scraping",
		"[x] 🌐 Browser Automation",
		"[x] 💻 Terminal & Processes",
		"[x] 📁 File Operations",
		"[ ] 🎬 Video Analysis",
		"[x] 🎨 Image Generation",
		"[x] 🔊 Text-to-Speech",
		"[ ] 🧠 Mixture of Agents",
		"[ ] 🧪 RL Training",
		"[ ] 🏠 Home Assistant",
		"[ ] 🎵 Spotify",
		"[ ] 💬 Discord (read/participate)",
		"[ ] 🛡️  Discord Server Admin",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing default marker %q:\n%s", want, stdout.String())
		}
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("non-interactive setup tools mutated config path %s: %v", configPath, err)
	}
}

func TestRunToolsPersistsViaPlatformToolsetConfig(t *testing.T) {
	t.Setenv("HASS_TOKEN", "")
	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeToolsFixtureConfig(t, configPath, `
platform_toolsets = { cli = ["terminal", "custom-mcp-server", "no_mcp"] }
`)
	var stdout bytes.Buffer
	if err := RunTools(ToolsOptions{
		Out:        &stdout,
		ConfigPath: configPath,
		PromptString: func(string, string) (string, error) {
			return "web,browser,discord,no_mcp", nil
		},
	}); err != nil {
		t.Fatalf("RunTools: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Saved CLI tool configuration:",
		"browser",
		"custom-mcp-server",
		"web",
		"setup_tools_issue: kind=restricted_toolset toolset=discord",
		"setup_tools_issue: kind=no_mcp_suppression toolset=no_mcp",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}

	got := readCLIPlatformToolsets(t, configPath)
	for _, want := range []string{"browser", "custom-mcp-server", "web"} {
		if !testContainsString(got, want) {
			t.Fatalf("persisted toolsets = %v, missing %q", got, want)
		}
	}
	for _, unwanted := range []string{"discord", "no_mcp"} {
		if testContainsString(got, unwanted) {
			t.Fatalf("persisted toolsets = %v, should not contain %q", got, unwanted)
		}
	}
}

func TestRunToolsProviderRowsAreRowBacked(t *testing.T) {
	t.Setenv("HASS_TOKEN", "test-hass-token")
	configPath := filepath.Join(t.TempDir(), "config.toml")
	var stdout bytes.Buffer
	if err := RunTools(ToolsOptions{
		Out:        &stdout,
		ConfigPath: configPath,
		PromptString: func(string, string) (string, error) {
			return "web,browser,image_gen,rl,tts,skills,memory,homeassistant", nil
		},
	}); err != nil {
		t.Fatalf("RunTools: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Provider/API key setup",
		"setup_tools_provider_row_backed: toolset=web provider=web",
		"setup_tools_provider_row_backed: toolset=browser provider=browser",
		"setup_tools_provider_row_backed: toolset=image_gen provider=image_gen",
		"setup_tools_provider_row_backed: toolset=rl provider=rl",
		"setup_tools_provider_row_backed: toolset=tts provider=tts",
		"setup_tools_provider_row_backed: toolset=skills provider=github_skills_hub",
		"setup_tools_provider_row_backed: toolset=memory provider=honcho",
		"setup_tools_provider_row_backed: toolset=homeassistant provider=homeassistant",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-", "access_token", "refresh_token"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("setup tools output leaked secret-shaped text %q:\nstdout=%s", forbidden, stdout.String())
		}
	}
}

func writeToolsFixtureConfig(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func readCLIPlatformToolsets(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse config: %v\n%s", err, string(data))
	}
	platformToolsets, _ := doc["platform_toolsets"].(map[string]any)
	raw, _ := platformToolsets["cli"].([]any)
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		out = append(out, value.(string))
	}
	return out
}

func testContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
