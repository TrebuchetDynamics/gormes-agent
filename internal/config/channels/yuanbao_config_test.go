package channels

import (
	"strings"
	"testing"
)

func TestYuanbaoConfigRequiresEnabledAndCredentials(t *testing.T) {
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

func TestYuanbaoConfigStatusRedactsCredentialAndSessionFields(t *testing.T) {
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
