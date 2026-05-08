package googlechat

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestGoogleChatPluginMetadataMatchesHermesPlugin(t *testing.T) {
	meta := PluginMetadata()
	if meta.Name != PlatformName || meta.Label != "Google Chat" {
		t.Fatalf("metadata identity = %+v", meta)
	}
	if !reflect.DeepEqual(meta.RequiredEnv, []string{"GOOGLE_CHAT_PROJECT_ID", "GOOGLE_CHAT_SUBSCRIPTION_NAME", "GOOGLE_CHAT_SERVICE_ACCOUNT_JSON"}) {
		t.Fatalf("RequiredEnv = %v", meta.RequiredEnv)
	}
	if meta.AllowedUsersEnv != "GOOGLE_CHAT_ALLOWED_USERS" || meta.AllowAllEnv != "GOOGLE_CHAT_ALLOW_ALL_USERS" {
		t.Fatalf("auth env metadata = %+v", meta)
	}
	if meta.InstallHint != "pip install 'hermes-agent[google_chat]'" {
		t.Fatalf("InstallHint = %q", meta.InstallHint)
	}
	if meta.MaxMessageLength != MaxMessageLength {
		t.Fatalf("MaxMessageLength = %d, want %d", meta.MaxMessageLength, MaxMessageLength)
	}
	if !meta.SetupAvailable || !strings.Contains(meta.PlatformHint, "Google Chat") {
		t.Fatalf("metadata missing setup/platform hint: %+v", meta)
	}
}

func TestGoogleChatNormalizePubSubMessage(t *testing.T) {
	payload := []byte(`{
		"chat": {
			"messagePayload": {
				"space": {"name": "spaces/AAA", "spaceType": "DIRECT_MESSAGE"},
				"message": {
					"name": "spaces/AAA/messages/msg-1",
					"text": "hello from chat",
					"argumentText": "hello from chat",
					"sender": {
						"name": "users/12345",
						"email": "ada@example.com",
						"displayName": "Ada Lovelace",
						"type": "HUMAN"
					},
					"thread": {"name": "spaces/AAA/threads/thread-1"},
					"space": {"name": "spaces/AAA", "spaceType": "DIRECT_MESSAGE"}
				}
			}
		}
	}`)

	ch := NewChannel(Config{}, nil, nil)
	ev, ok := ch.NormalizePubSubMessage(payload)
	if !ok {
		t.Fatal("NormalizePubSubMessage ok = false, want true")
	}
	if ev.Platform != PlatformName || ev.ChatID != "spaces/AAA" || ev.ThreadID != "spaces/AAA/threads/thread-1" {
		t.Fatalf("chat/thread identity = %+v", ev)
	}
	if ev.MsgID != "spaces/AAA/messages/msg-1" || ev.MessageID != "spaces/AAA/messages/msg-1" {
		t.Fatalf("message IDs = %+v", ev)
	}
	if ev.UserID != "users/12345" || ev.UserName != "Ada Lovelace" || ev.AccountID != "ada@example.com" {
		t.Fatalf("sender identity = %+v", ev)
	}
	if ev.ChatType != "dm" || ev.Kind != gateway.EventSubmit || ev.Text != "hello from chat" {
		t.Fatalf("normalized body = %+v", ev)
	}
}

func TestGoogleChatDeliveryWithoutTransport(t *testing.T) {
	ch := NewChannel(Config{}, nil, nil)
	if _, err := ch.Send(context.Background(), "spaces/AAA", "hello"); err == nil || err.Error() != "googlechat_transport_not_configured" {
		t.Fatalf("Send without transport error = %v", err)
	}
	if _, err := ch.SendThread(context.Background(), "spaces/AAA", "spaces/AAA/threads/thread-1", "hello"); err == nil || err.Error() != "googlechat_transport_not_configured" {
		t.Fatalf("SendThread without transport error = %v", err)
	}
}

func TestGoogleChatFollowsTeamsSeamPattern(t *testing.T) {
	var _ gateway.Channel = (*Channel)(nil)
	var _ gateway.ThreadSender = (*Channel)(nil)

	client := &fakeGoogleChatClient{events: make(chan []byte)}
	ch := NewChannel(Config{}, client, nil)
	msgID, err := ch.SendThread(context.Background(), "spaces/AAA", "spaces/AAA/threads/thread-1", "hello")
	if err != nil || msgID != "googlechat-msg-1" {
		t.Fatalf("SendThread = %q/%v", msgID, err)
	}
	if got, want := client.sent, []string{"spaces/AAA|spaces/AAA/threads/thread-1|hello"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sent = %v, want %v", got, want)
	}
}

type fakeGoogleChatClient struct {
	events chan []byte
	sent   []string
	closed bool
}

func (c *fakeGoogleChatClient) Events() <-chan []byte { return c.events }

func (c *fakeGoogleChatClient) SendMessage(_ context.Context, space, thread, text string) (string, error) {
	c.sent = append(c.sent, space+"|"+thread+"|"+text)
	return "googlechat-msg-1", nil
}

func (c *fakeGoogleChatClient) Close() error {
	c.closed = true
	return nil
}
