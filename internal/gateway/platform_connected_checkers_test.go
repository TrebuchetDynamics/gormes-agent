package gateway

import "testing"

func TestPlatformConnectedCheckersCoverManifest(t *testing.T) {
	missing := MissingPlatformConnectedCheckers(HermesGatewayPlatformManifest())
	if len(missing) > 0 {
		t.Fatalf("missing connected checkers for built-in platforms: %v", missing)
	}
}

func TestPlatformConnectedCheckersHandleMinimalConfig(t *testing.T) {
	for _, id := range PlatformConnectedCheckerIDs() {
		t.Run(id, func(t *testing.T) {
			_, ok := PlatformLooksConfigured(PlatformConnectionConfig{ID: id, Enabled: true})
			if !ok {
				t.Fatalf("checker %q did not return a bool result", id)
			}
		})
	}
}

func TestPlatformConnectedCheckersReturnTrueForSyntheticConfig(t *testing.T) {
	cases := []PlatformConnectionConfig{
		{ID: "telegram", Enabled: true, Token: "telegram-token"},
		{ID: "discord", Enabled: true, Token: "discord-token"},
		{ID: "slack", Enabled: true, Token: "slack-token"},
		{ID: "matrix", Enabled: true, Token: "matrix-token"},
		{ID: "mattermost", Enabled: true, Token: "mattermost-token"},
		{ID: "homeassistant", Enabled: true, APIKey: "hass-token"},
		{ID: "weixin", Enabled: true, Extra: map[string]string{"account_id": "acct", "token": "tok"}},
		{ID: "signal", Enabled: true, Extra: map[string]string{"http_url": "http://signal:8080"}},
		{ID: "email", Enabled: true, Extra: map[string]string{"address": "hermes@example.com"}},
		{ID: "sms", Enabled: true, Extra: map[string]string{"twilio_account_sid": "ACtest"}},
		{ID: "api_server", Enabled: true},
		{ID: "webhook", Enabled: true},
		{ID: "msgraph_webhook", Enabled: true, Extra: map[string]string{"client_state": "shared-secret"}},
		{ID: "whatsapp", Enabled: true},
		{ID: "feishu", Enabled: true, Extra: map[string]string{"app_id": "app"}},
		{ID: "google_chat", Enabled: true, Extra: map[string]string{"project_id": "project", "subscription_name": "subscription"}},
		{ID: "irc", Enabled: true, Extra: map[string]string{"server": "irc.example.net", "channel": "#gormes"}},
		{ID: "line", Enabled: true, Extra: map[string]string{"channel_access_token": "line-token", "channel_secret": "line-secret"}},
		{ID: "simplex", Enabled: true, Extra: map[string]string{"ws_url": "ws://127.0.0.1:5225"}},
		{ID: "wecom", Enabled: true, Extra: map[string]string{"bot_id": "bot"}},
		{ID: "wecom_callback", Enabled: true, Extra: map[string]string{"corp_id": "corp"}},
		{ID: "bluebubbles", Enabled: true, Extra: map[string]string{"server_url": "http://bb:1234", "password": "pw"}},
		{ID: "qqbot", Enabled: true, Extra: map[string]string{"app_id": "app", "client_secret": "secret"}},
		{ID: "yuanbao", Enabled: true, Extra: map[string]string{"app_id": "app", "app_secret": "secret"}},
		{ID: "dingtalk", Enabled: true, Extra: map[string]string{"client_id": "id", "client_secret": "secret"}},
		{ID: "teams", Enabled: true, Extra: map[string]string{"client_id": "id", "client_secret": "secret", "tenant_id": "tenant"}},
	}

	for _, tc := range cases {
		t.Run(tc.ID, func(t *testing.T) {
			configured, ok := PlatformLooksConfigured(tc)
			if !ok {
				t.Fatalf("checker %q missing", tc.ID)
			}
			if !configured {
				t.Fatalf("PlatformLooksConfigured(%+v) = false, want true", tc)
			}
		})
	}
}

func TestPlatformConnectedCheckersRequireMSGraphWebhookClientState(t *testing.T) {
	configured, ok := PlatformLooksConfigured(PlatformConnectionConfig{
		ID:      "msgraph_webhook",
		Enabled: true,
	})
	if !ok {
		t.Fatal("msgraph_webhook checker missing")
	}
	if configured {
		t.Fatal("msgraph_webhook without client_state should not look configured")
	}

	configured, ok = PlatformLooksConfigured(PlatformConnectionConfig{
		ID:      "msgraph_webhook",
		Enabled: true,
		Extra:   map[string]string{"client_state": "shared-secret"},
	})
	if !ok {
		t.Fatal("msgraph_webhook checker missing with client_state")
	}
	if !configured {
		t.Fatal("msgraph_webhook with client_state should look configured")
	}
}

func TestPlatformConnectedCheckersRequireBundledPluginFields(t *testing.T) {
	cases := []struct {
		id      string
		missing PlatformConnectionConfig
		present PlatformConnectionConfig
	}{
		{
			id:      "google_chat",
			missing: PlatformConnectionConfig{ID: "google_chat", Enabled: true, Extra: map[string]string{"project_id": "project"}},
			present: PlatformConnectionConfig{ID: "google_chat", Enabled: true, Extra: map[string]string{"project_id": "project", "subscription_name": "subscription"}},
		},
		{
			id:      "irc",
			missing: PlatformConnectionConfig{ID: "irc", Enabled: true, Extra: map[string]string{"server": "irc.example.net"}},
			present: PlatformConnectionConfig{ID: "irc", Enabled: true, Extra: map[string]string{"server": "irc.example.net", "channel": "#gormes"}},
		},
		{
			id:      "line",
			missing: PlatformConnectionConfig{ID: "line", Enabled: true, Extra: map[string]string{"channel_access_token": "line-token"}},
			present: PlatformConnectionConfig{ID: "line", Enabled: true, Extra: map[string]string{"channel_access_token": "line-token", "channel_secret": "line-secret"}},
		},
		{
			id:      "simplex",
			missing: PlatformConnectionConfig{ID: "simplex", Enabled: true},
			present: PlatformConnectionConfig{ID: "simplex", Enabled: true, Extra: map[string]string{"ws_url": "ws://127.0.0.1:5225"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			configured, ok := PlatformLooksConfigured(tc.missing)
			if !ok {
				t.Fatalf("%s checker missing", tc.id)
			}
			if configured {
				t.Fatalf("%s missing required field should not look configured", tc.id)
			}

			configured, ok = PlatformLooksConfigured(tc.present)
			if !ok {
				t.Fatalf("%s checker missing with required fields", tc.id)
			}
			if !configured {
				t.Fatalf("%s with required fields should look configured", tc.id)
			}
		})
	}
}
