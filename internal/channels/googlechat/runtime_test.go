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

func TestGoogleChatNativeChatAPIPubSubMessage(t *testing.T) {
	payload := []byte(`{
		"type": "MESSAGE",
		"space": {"name": "spaces/BBB", "spaceType": "GROUP_CHAT"},
		"message": {
			"name": "spaces/BBB/messages/msg-2",
			"text": "native chat event",
			"sender": {
				"name": "users/native",
				"email": "native@example.com",
				"displayName": "Native User",
				"type": "HUMAN"
			},
			"thread": {"name": "spaces/BBB/threads/thread-2"}
		}
	}`)

	ch := NewChannel(Config{}, nil, nil)
	ev, ok := ch.NormalizePubSubMessage(payload)
	if !ok {
		t.Fatal("NormalizePubSubMessage ok = false, want true")
	}
	if ev.ChatID != "spaces/BBB" || ev.ThreadID != "spaces/BBB/threads/thread-2" || ev.ChatType != "group" {
		t.Fatalf("native chat identity = %+v", ev)
	}
	if ev.UserID != "users/native" || ev.AccountID != "native@example.com" || ev.Text != "native chat event" {
		t.Fatalf("native sender/body = %+v", ev)
	}
}

func TestGoogleChatRelayFlatSenderTypeSelfFilter(t *testing.T) {
	payload := []byte(`{
		"event_type": "MESSAGE",
		"sender_email": "ada@example.com",
		"sender_display_name": "Ada Relay",
		"sender_type": "HUMAN",
		"text": "hello from relay",
		"space_name": "spaces/CCC",
		"space_type": "DIRECT_MESSAGE",
		"thread_name": "spaces/CCC/threads/thread-3",
		"message_name": "spaces/CCC/messages/msg-3"
	}`)

	ch := NewChannel(Config{}, nil, nil)
	ev, ok := ch.NormalizePubSubMessage(payload)
	if !ok {
		t.Fatal("NormalizePubSubMessage ok = false, want relay HUMAN event")
	}
	if ev.UserID != "users/relay-ada_at_example_com" || ev.UserName != "Ada Relay" || ev.AccountID != "ada@example.com" {
		t.Fatalf("relay sender = %+v", ev)
	}
	if ev.ChatID != "spaces/CCC" || ev.ThreadID != "spaces/CCC/threads/thread-3" || ev.MsgID != "spaces/CCC/messages/msg-3" {
		t.Fatalf("relay message identity = %+v", ev)
	}
	if ev.ChatType != "dm" || ev.Text != "hello from relay" {
		t.Fatalf("relay body = %+v", ev)
	}

	botPayload := []byte(`{
		"event_type": "MESSAGE",
		"sender_email": "bot@example.com",
		"sender_display_name": "Relay Bot",
		"sender_type": "BOT",
		"text": "loop me",
		"space_name": "spaces/CCC",
		"message_name": "spaces/CCC/messages/msg-bot"
	}`)
	if ev, ok := ch.NormalizePubSubMessage(botPayload); ok {
		t.Fatalf("relay BOT normalized as event = %+v, want self-filter drop", ev)
	}

	unknownTypePayload := []byte(`{
		"event_type": "MESSAGE",
		"sender_email": "service@example.com",
		"sender_display_name": "Service Relay",
		"sender_type": "SERVICE",
		"text": "fallback human",
		"space_name": "spaces/DDD",
		"message_name": "spaces/DDD/messages/msg-4"
	}`)
	ev, ok = ch.NormalizePubSubMessage(unknownTypePayload)
	if !ok {
		t.Fatal("NormalizePubSubMessage ok = false, want unknown sender_type to default to HUMAN")
	}
	if ev.UserID != "users/relay-service_at_example_com" || ev.Text != "fallback human" {
		t.Fatalf("unknown sender_type fallback = %+v", ev)
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
