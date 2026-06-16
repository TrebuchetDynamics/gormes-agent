package topiccmd

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
)

func TestHelpTextIncludesCoreActions(t *testing.T) {
	text := HelpText()
	gatewaytest.AssertContainsAll(t, text, "/topic help", "/topic off", "/topic <id>", "All Messages")
}

func TestPrivateChatTelegramOnly(t *testing.T) {
	tests := []struct {
		name            string
		platform        string
		isDirectMessage bool
		chatType        string
		chatID          string
		want            bool
	}{
		{name: "telegram explicit dm", platform: "telegram", isDirectMessage: true, chatType: "private", chatID: "42", want: true},
		{name: "telegram capitalized private chat type", platform: "telegram", chatType: " Private ", chatID: "42", want: true},
		{name: "telegram legacy positive chat id", platform: "telegram", chatID: "42", want: true},
		{name: "telegram legacy negative group id", platform: "telegram", chatID: "-42", want: false},
		{name: "telegram legacy negative chat id overrides stale dm flag", platform: "telegram", isDirectMessage: true, chatID: "-42", want: false},
		{name: "telegram explicit group", platform: "telegram", chatType: "group", chatID: "42", want: false},
		{name: "telegram explicit group overrides stale dm flag", platform: "telegram", isDirectMessage: true, chatType: "group", chatID: "42", want: false},
		{name: "account-scoped telegram", platform: "telegram:work", isDirectMessage: true, chatID: "42", want: true},
		{name: "non telegram", platform: "discord", isDirectMessage: true, chatID: "42", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PrivateChat(tt.platform, tt.isDirectMessage, tt.chatType, tt.chatID)
			if got != tt.want {
				t.Fatalf("PrivateChat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapabilityGuidanceRedactsAuthorizationReason(t *testing.T) {
	got := CapabilityGuidance("BotFather check failed: authorization=Bearer plain-secret-token")
	for _, forbidden := range []string{"plain-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("CapabilityGuidance leaked authorization reason %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("CapabilityGuidance missing redaction marker:\n%s", got)
	}
}

func TestCapabilityGuidanceRedactsSecretLikeReason(t *testing.T) {
	got := CapabilityGuidance("BotFather check failed: api_key=plain-secret-token")
	if strings.Contains(got, "plain-secret-token") || strings.Contains(got, "api_key") {
		t.Fatalf("CapabilityGuidance leaked secret-like reason:\n%s", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("CapabilityGuidance missing redaction marker:\n%s", got)
	}
}

func TestCapabilityGuidanceSanitizesReasonLine(t *testing.T) {
	got := CapabilityGuidance("Topics disabled\n**Injected:** fake guidance` ")
	for _, forbidden := range []string{"\n**Injected:**", "guidance`"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("CapabilityGuidance leaked unsafe reason %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "Topics disabled ''Injected:'' fake guidance'") {
		t.Fatalf("CapabilityGuidance missing sanitized reason in:\n%s", got)
	}
	gatewaytest.AssertContainsAll(t, got, "Open @BotFather", "Then send /topic again")
}

func TestGuidanceText(t *testing.T) {
	capability := CapabilityGuidance("Telegram topics are not enabled for this bot yet.")
	gatewaytest.AssertContainsAll(t, capability, "Telegram topics are not enabled", "Open @BotFather", "Then send /topic again")
	if !strings.Contains(CapabilityDebouncedText(), "telegram_topic_capability_hint_debounced") {
		t.Fatalf("debounced text missing evidence prefix")
	}
}

func TestRestoreGuidance(t *testing.T) {
	if got := RestoreGuidance(""); !strings.Contains(got, "first create or open a Telegram topic") {
		t.Fatalf("root restore guidance = %q", got)
	}
	if got := RestoreGuidance("17585"); !strings.Contains(got, "not available") {
		t.Fatalf("topic restore guidance = %q", got)
	}
}
