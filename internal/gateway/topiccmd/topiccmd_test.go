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
		{name: "telegram legacy positive chat id", platform: "telegram", chatID: "42", want: true},
		{name: "telegram legacy negative group id", platform: "telegram", chatID: "-42", want: false},
		{name: "telegram explicit group", platform: "telegram", chatType: "group", chatID: "42", want: false},
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
