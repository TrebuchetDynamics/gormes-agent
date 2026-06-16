package telegram

import (
	"context"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_SendMediaUsesTelegramFileTypes(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	cases := []struct {
		name     string
		media    gateway.OutboundMedia
		wantType string
	}{
		{
			name:     "photo",
			media:    gateway.OutboundMedia{Path: writeTelegramTestMedia(t, "photo.png"), Kind: gateway.OutboundMediaKindImage},
			wantType: "photo",
		},
		{
			name:     "document",
			media:    gateway.OutboundMedia{Path: writeTelegramTestMedia(t, "report.pdf"), Kind: gateway.OutboundMediaKindDocument},
			wantType: "document",
		},
		{
			name:     "video",
			media:    gateway.OutboundMedia{Path: writeTelegramTestMedia(t, "clip.mp4"), Kind: gateway.OutboundMediaKindVideo},
			wantType: "video",
		},
	}
	for _, tc := range cases {
		if _, err := b.SendMedia(context.Background(), "42", "99", tc.media); err != nil {
			t.Fatalf("SendMedia %s: %v", tc.name, err)
		}
	}

	sent := mc.sentMessages()
	if len(sent) != len(cases) {
		t.Fatalf("sent count = %d, want %d", len(sent), len(cases))
	}
	if cfg, ok := sent[0].(tgbotapi.PhotoConfig); !ok || cfg.ReplyToMessageID != 99 {
		t.Fatalf("sent[0] = %T/%+v, want PhotoConfig with reply_to 99", sent[0], sent[0])
	}
	if cfg, ok := sent[1].(tgbotapi.DocumentConfig); !ok || cfg.ReplyToMessageID != 99 {
		t.Fatalf("sent[1] = %T/%+v, want DocumentConfig with reply_to 99", sent[1], sent[1])
	}
	if cfg, ok := sent[2].(tgbotapi.VideoConfig); !ok || cfg.ReplyToMessageID != 99 {
		t.Fatalf("sent[2] = %T/%+v, want VideoConfig with reply_to 99", sent[2], sent[2])
	}
}

func TestBot_SendMediaRejectsInvalidReplyID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	_, err := b.SendMedia(context.Background(), "42", "abc", gateway.OutboundMedia{
		Path: writeTelegramTestMedia(t, "photo.png"),
		Kind: gateway.OutboundMediaKindImage,
	})
	if err == nil || !strings.Contains(err.Error(), `invalid reply msgID "abc"`) {
		t.Fatalf("SendMedia err = %v, want invalid reply msgID", err)
	}
}

func TestBot_SendMediaIncludesTelegramThreadID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	msgID, err := b.SendMedia(context.Background(), "42", "99", gateway.OutboundMedia{
		Path:     writeTelegramTestMedia(t, "photo.png"),
		Kind:     gateway.OutboundMediaKindImage,
		ThreadID: "777",
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if msgID == "" || msgID == "0" {
		t.Fatalf("msgID = %q, want upload response message ID", msgID)
	}

	uploads := mc.uploadRequests()
	if len(uploads) != 1 {
		t.Fatalf("uploads = %+v, want one upload", uploads)
	}
	upload := uploads[0]
	if upload.Endpoint != "sendPhoto" {
		t.Fatalf("endpoint = %q, want sendPhoto", upload.Endpoint)
	}
	if upload.Params["chat_id"] != "42" || upload.Params["reply_to_message_id"] != "99" || upload.Params["message_thread_id"] != "777" {
		t.Fatalf("params = %+v, want chat_id/reply_to/message_thread_id", upload.Params)
	}
	if len(upload.Files) != 1 || upload.Files[0].Name != "photo" {
		t.Fatalf("files = %+v, want Telegram photo file field", upload.Files)
	}
}
