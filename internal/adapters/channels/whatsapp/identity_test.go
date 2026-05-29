package whatsapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDecideRuntime_IdentitySourceFollowsSelectedRuntime(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")

	bridgePlan, err := DecideRuntime(RuntimeConfig{
		StateRoot: stateRoot,
	})
	if err != nil {
		t.Fatalf("DecideRuntime(bridge) error = %v, want nil", err)
	}
	if bridgePlan.Identity.BotIdentitySource != IdentitySourceBridgeMessage {
		t.Fatalf("bridge BotIdentitySource = %q, want %q", bridgePlan.Identity.BotIdentitySource, IdentitySourceBridgeMessage)
	}

	nativePlan, err := DecideRuntime(RuntimeConfig{
		Preference: RuntimePreferenceNativeFirst,
		StateRoot:  stateRoot,
		Native: NativeRuntimeConfig{
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatalf("DecideRuntime(native) error = %v, want nil", err)
	}
	if nativePlan.Identity.BotIdentitySource != IdentitySourceNativeSession {
		t.Fatalf("native BotIdentitySource = %q, want %q", nativePlan.Identity.BotIdentitySource, IdentitySourceNativeSession)
	}
}

func TestNormalizeInboundWithIdentity_Fixtures(t *testing.T) {
	fixtures := loadIdentityFixtures(t)
	keys := make([]string, 0, len(fixtures))
	for key := range fixtures {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		tt := fixtures[key]
		t.Run(key, func(t *testing.T) {
			got := NormalizeInboundWithIdentity(tt.Message, tt.Context)

			if string(got.Decision) != tt.Want.Decision {
				t.Fatalf("Decision = %q, want %q", got.Decision, tt.Want.Decision)
			}
			if got.Status.Source != IdentitySource(tt.Want.BotIdentitySource) {
				t.Fatalf("Status.Source = %q, want %q", got.Status.Source, tt.Want.BotIdentitySource)
			}
			if got.Status.Resolved != tt.Want.BotResolved {
				t.Fatalf("Status.Resolved = %v, want %v", got.Status.Resolved, tt.Want.BotResolved)
			}
			if got.Status.BotID != tt.Want.BotID {
				t.Fatalf("Status.BotID = %q, want %q", got.Status.BotID, tt.Want.BotID)
			}
			if got.Status.Reason != tt.Want.StatusReason {
				t.Fatalf("Status.Reason = %q, want %q", got.Status.Reason, tt.Want.StatusReason)
			}

			if tt.Want.SuppressionReason != "" {
				if got.Suppression.Reason != SelfChatSuppressionReason(tt.Want.SuppressionReason) {
					t.Fatalf("Suppression.Reason = %q, want %q", got.Suppression.Reason, tt.Want.SuppressionReason)
				}
				if got.Routed() {
					t.Fatal("Routed() = true for suppressed message, want false")
				}
				return
			}

			if !got.Routed() {
				t.Fatal("Routed() = false, want true")
			}
			if got.Event.Kind.String() != tt.Want.Kind {
				t.Fatalf("Event.Kind = %q, want %q", got.Event.Kind.String(), tt.Want.Kind)
			}
			if got.Event.Text != tt.Want.Text {
				t.Fatalf("Event.Text = %q, want %q", got.Event.Text, tt.Want.Text)
			}
			if got.Event.ChatID != tt.Want.EventChatID {
				t.Fatalf("Event.ChatID = %q, want %q", got.Event.ChatID, tt.Want.EventChatID)
			}
			if got.Event.UserID != tt.Want.EventUserID {
				t.Fatalf("Event.UserID = %q, want %q", got.Event.UserID, tt.Want.EventUserID)
			}
			if got.Identity.ChatID != tt.Want.IdentityChatID {
				t.Fatalf("Identity.ChatID = %q, want %q", got.Identity.ChatID, tt.Want.IdentityChatID)
			}
			if got.Identity.UserID != tt.Want.IdentityUserID {
				t.Fatalf("Identity.UserID = %q, want %q", got.Identity.UserID, tt.Want.IdentityUserID)
			}
			if got.Identity.RawChatID != tt.Want.RawChatID {
				t.Fatalf("Identity.RawChatID = %q, want %q", got.Identity.RawChatID, tt.Want.RawChatID)
			}
			if got.Identity.RawUserID != tt.Want.RawUserID {
				t.Fatalf("Identity.RawUserID = %q, want %q", got.Identity.RawUserID, tt.Want.RawUserID)
			}
			if got.Reply.ChatID != tt.Want.ReplyChatID {
				t.Fatalf("Reply.ChatID = %q, want %q", got.Reply.ChatID, tt.Want.ReplyChatID)
			}
		})
	}
}

type identityFixture struct {
	Context IdentityContext `json:"Context"`
	Message InboundMessage  `json:"Message"`
	Want    identityWant    `json:"Want"`
}

type identityWant struct {
	Decision          string `json:"Decision"`
	Kind              string `json:"Kind"`
	Text              string `json:"Text"`
	EventChatID       string `json:"EventChatID"`
	EventUserID       string `json:"EventUserID"`
	IdentityChatID    string `json:"IdentityChatID"`
	IdentityUserID    string `json:"IdentityUserID"`
	RawChatID         string `json:"RawChatID"`
	RawUserID         string `json:"RawUserID"`
	ReplyChatID       string `json:"ReplyChatID"`
	BotIdentitySource string `json:"BotIdentitySource"`
	BotResolved       bool   `json:"BotResolved"`
	BotID             string `json:"BotID"`
	StatusReason      string `json:"StatusReason"`
	SuppressionReason string `json:"SuppressionReason"`
}

func loadIdentityFixtures(t *testing.T) map[string]identityFixture {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", "identity_contract.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixtures map[string]identityFixture
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return fixtures
}
