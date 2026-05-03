package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/pelletier/go-toml/v2"
)

func TestSetupToolsChecklistShowsCLIConfigurableToolsets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HASS_TOKEN", "")

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "tools", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}

	for _, want := range []string{
		"Tools for CLI",
		"web", "Web Search & Scraping",
		"browser", "Browser Automation",
		"terminal", "Terminal & Processes",
		"file", "File Operations",
		"code_execution", "Code Execution",
		"vision", "Vision / Image Analysis",
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
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupToolsChecklistPreselectsDefaultsMinusDefaultOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HASS_TOKEN", "")

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "tools", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}

	for _, want := range []string{
		"[x] Web Search & Scraping",
		"[x] Browser Automation",
		"[x] Terminal & Processes",
		"[x] File Operations",
		"[x] Text-to-Speech",
		"[ ] Mixture of Agents",
		"[ ] RL Training",
		"[ ] Home Assistant",
		"[ ] Spotify",
		"[ ] Discord (read/participate)",
		"[ ] Discord Server Admin",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("non-interactive setup tools mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestSetupToolsPersistsViaPlatformToolsetConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HASS_TOKEN", "")
	writeSetupToolsFixtureConfig(t, `
platform_toolsets = { cli = ["terminal", "custom-mcp-server", "no_mcp"] }
`)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "web,browser,discord,no_mcp\n", "tools")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Saved CLI tool configuration:",
		"browser",
		"custom-mcp-server",
		"web",
		"setup_tools_issue: kind=restricted_toolset toolset=discord",
		"setup_tools_issue: kind=no_mcp_suppression toolset=no_mcp",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}

	got := readCLIPlatformToolsets(t)
	for _, want := range []string{"browser", "custom-mcp-server", "web"} {
		if !containsString(got, want) {
			t.Fatalf("persisted toolsets = %v, missing %q", got, want)
		}
	}
	for _, unwanted := range []string{"discord", "no_mcp"} {
		if containsString(got, unwanted) {
			t.Fatalf("persisted toolsets = %v, should not contain %q", got, unwanted)
		}
	}
}

func TestSetupToolsProviderRowsAreRowBacked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HASS_TOKEN", "test-hass-token")

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "web,browser,image_gen,rl,tts,skills,memory,homeassistant\n", "tools")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
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
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"sk-", "access_token", "refresh_token"} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("setup tools output leaked secret-shaped text %q:\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
}

func writeSetupToolsFixtureConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func readCLIPlatformToolsets(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(config.ConfigPath())
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
