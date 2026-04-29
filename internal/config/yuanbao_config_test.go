package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYuanbaoConfig_DisabledByDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Yuanbao.Enabled {
		t.Fatalf("Yuanbao.Enabled = true, want false (disabled by default)")
	}
	if cfg.Yuanbao.RuntimeEnabled() {
		t.Fatalf("Yuanbao.RuntimeEnabled() = true, want false (no creds)")
	}
}

func TestYuanbaoConfig_RequiresEnabledAndCredentials(t *testing.T) {
	cases := []struct {
		name string
		cfg  YuanbaoCfg
		want bool
	}{
		{
			name: "disabled with full credentials stays off",
			cfg:  YuanbaoCfg{Enabled: false, LoginToken: "tok", HySource: "src", AgentID: "agent"},
			want: false,
		},
		{
			name: "enabled without token stays off",
			cfg:  YuanbaoCfg{Enabled: true, HySource: "src", AgentID: "agent"},
			want: false,
		},
		{
			name: "enabled without hy_source stays off",
			cfg:  YuanbaoCfg{Enabled: true, LoginToken: "tok", AgentID: "agent"},
			want: false,
		},
		{
			name: "enabled without agent_id stays off",
			cfg:  YuanbaoCfg{Enabled: true, LoginToken: "tok", HySource: "src"},
			want: false,
		},
		{
			name: "enabled with all credentials runs",
			cfg:  YuanbaoCfg{Enabled: true, LoginToken: "tok", HySource: "src", AgentID: "agent"},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.RuntimeEnabled(); got != tc.want {
				t.Fatalf("RuntimeEnabled() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestYuanbaoConfig_StatusRedactsCredentialAndSessionFields(t *testing.T) {
	cfg := YuanbaoCfg{
		Enabled:               true,
		LoginToken:            "secret-login-token-1234",
		HySource:              "secret-hy-source",
		AgentID:               "secret-agent-id",
		AllowedConversationID: "conv-789",
	}

	status := cfg.RedactedStatus()

	for field, value := range map[string]string{
		"login_token": cfg.LoginToken,
		"hy_source":   cfg.HySource,
		"agent_id":    cfg.AgentID,
	} {
		if strings.Contains(status, value) {
			t.Fatalf("RedactedStatus exposed %s value %q in %q", field, value, status)
		}
	}
	for _, want := range []string{"yuanbao", "enabled=true", "allowed_conversation_id=conv-789"} {
		if !strings.Contains(status, want) {
			t.Fatalf("RedactedStatus missing %q in %q", want, status)
		}
	}
}

func TestYuanbaoConfig_LoadParsesDisabledByDefaultSectionFromTOML(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[yuanbao]
login_token = "fake-login-token"
hy_source = "fake-hy-source"
agent_id = "fake-agent"
allowed_conversation_id = "conv-1"
coalesce_ms = 750
first_run_discovery = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Yuanbao.Enabled {
		t.Fatalf("Yuanbao.Enabled = true, want false (omitted from TOML)")
	}
	if cfg.Yuanbao.LoginToken != "fake-login-token" {
		t.Fatalf("Yuanbao.LoginToken = %q", cfg.Yuanbao.LoginToken)
	}
	if cfg.Yuanbao.HySource != "fake-hy-source" || cfg.Yuanbao.AgentID != "fake-agent" {
		t.Fatalf("Yuanbao identity fields = %#v", cfg.Yuanbao)
	}
	if cfg.Yuanbao.AllowedConversationID != "conv-1" {
		t.Fatalf("Yuanbao.AllowedConversationID = %q", cfg.Yuanbao.AllowedConversationID)
	}
	if cfg.Yuanbao.CoalesceMs != 750 || !cfg.Yuanbao.FirstRunDiscovery {
		t.Fatalf("Yuanbao knobs = %#v", cfg.Yuanbao)
	}
	if cfg.Yuanbao.RuntimeEnabled() {
		t.Fatalf("RuntimeEnabled() = true, want false (Enabled flag missing)")
	}
}
