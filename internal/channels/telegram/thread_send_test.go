package telegram

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestBot_SendThreadUsesRawSendMessageThreadID(t *testing.T) {
	client := newMockClient()
	b := New(Config{}, client, nil)

	msgID, err := b.SendThread(context.Background(), "42", "777", "hello *thread*")
	if err != nil {
		t.Fatalf("SendThread: %v", err)
	}
	if msgID == "" {
		t.Fatalf("SendThread msgID is empty")
	}

	if len(client.uploads) != 1 {
		t.Fatalf("uploads = %+v, want one raw sendMessage upload", client.uploads)
	}
	upload := client.uploads[0]
	if upload.Endpoint != "sendMessage" {
		t.Fatalf("endpoint = %q, want sendMessage", upload.Endpoint)
	}
	if upload.Params["chat_id"] != "42" || upload.Params["message_thread_id"] != "777" || upload.Params["text"] != "hello *thread*" {
		t.Fatalf("params = %+v, want chat_id/message_thread_id/text", upload.Params)
	}
	if upload.Params["parse_mode"] != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("parse_mode = %q, want MarkdownV2", upload.Params["parse_mode"])
	}
}

func TestBot_SendThreadReplyOmitsGeneralTopicThreadID(t *testing.T) {
	client := newMockClient()
	b := New(Config{}, client, nil)

	if _, err := b.SendThreadReply(context.Background(), "-100123", "1", "99", "hello general"); err != nil {
		t.Fatalf("SendThreadReply: %v", err)
	}

	if len(client.uploads) != 1 {
		t.Fatalf("uploads = %+v, want one raw sendMessage upload", client.uploads)
	}
	upload := client.uploads[0]
	if upload.Endpoint != "sendMessage" {
		t.Fatalf("endpoint = %q, want sendMessage", upload.Endpoint)
	}
	if upload.Params["chat_id"] != "-100123" || upload.Params["reply_to_message_id"] != "99" || upload.Params["text"] != "hello general" {
		t.Fatalf("params = %+v, want chat/reply/text", upload.Params)
	}
	if _, ok := upload.Params["message_thread_id"]; ok {
		t.Fatalf("message_thread_id = %q, want omitted for General topic text send", upload.Params["message_thread_id"])
	}
}

func TestBot_SendThreadChatActionIncludesGeneralTopicThreadID(t *testing.T) {
	client := newMockClient()
	b := New(Config{}, client, nil)

	if err := b.SendThreadChatAction(context.Background(), "-100123", "1", "typing"); err != nil {
		t.Fatalf("SendThreadChatAction: %v", err)
	}

	if len(client.uploads) != 1 {
		t.Fatalf("uploads = %+v, want one raw sendChatAction upload", client.uploads)
	}
	upload := client.uploads[0]
	if upload.Endpoint != "sendChatAction" {
		t.Fatalf("endpoint = %q, want sendChatAction", upload.Endpoint)
	}
	if upload.Params["chat_id"] != "-100123" || upload.Params["message_thread_id"] != "1" || upload.Params["action"] != "typing" {
		t.Fatalf("params = %+v, want chat_id/message_thread_id/action", upload.Params)
	}
}

func TestBot_NotificationModeSilentPlaceholdersByDefault(t *testing.T) {
	client := newMockClient()
	b := New(Config{}, client, nil)

	if _, err := b.SendPlaceholder(context.Background(), "42"); err != nil {
		t.Fatalf("SendPlaceholder: %v", err)
	}
	sent := client.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent[0] = %T, want MessageConfig", sent[0])
	}
	if !msg.DisableNotification {
		t.Fatalf("DisableNotification = false, want silent placeholder by default")
	}

	if _, err := b.Send(context.Background(), "42", "final answer"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	sent = client.sentMessages()
	finalMsg, ok := sent[1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent[1] = %T, want MessageConfig", sent[1])
	}
	if finalMsg.DisableNotification {
		t.Fatalf("final Send DisableNotification = true, want ordinary final sends to notify")
	}
}

func TestBot_NotificationModeAllKeepsPlaceholdersNotifying(t *testing.T) {
	client := newMockClient()
	b := New(Config{Notifications: "all"}, client, nil)

	if _, err := b.SendPlaceholder(context.Background(), "42"); err != nil {
		t.Fatalf("SendPlaceholder: %v", err)
	}
	sent := client.sentMessages()
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent[0] = %T, want MessageConfig", sent[0])
	}
	if msg.DisableNotification {
		t.Fatalf("DisableNotification = true, want notifications=all to preserve legacy notifying placeholders")
	}
}

func TestBot_NotificationModeSilentThreadPlaceholdersByDefault(t *testing.T) {
	client := newMockClient()
	b := New(Config{}, client, nil)

	if _, err := b.SendThreadPlaceholder(context.Background(), "42", "777"); err != nil {
		t.Fatalf("SendThreadPlaceholder: %v", err)
	}
	uploads := client.uploadRequests()
	if len(uploads) != 1 {
		t.Fatalf("uploads = %+v, want one raw sendMessage upload", uploads)
	}
	if got := uploads[0].Params["disable_notification"]; got != "true" {
		t.Fatalf("thread placeholder disable_notification = %q, want true", got)
	}

	if _, err := b.SendThread(context.Background(), "42", "777", "final thread answer"); err != nil {
		t.Fatalf("SendThread: %v", err)
	}
	uploads = client.uploadRequests()
	if _, ok := uploads[1].Params["disable_notification"]; ok {
		t.Fatalf("final SendThread params = %+v, want no disable_notification", uploads[1].Params)
	}
}

func TestBot_NotificationModeSilentReplyPlaceholdersByDefault(t *testing.T) {
	client := newMockClient()
	b := New(Config{}, client, nil)

	if _, err := b.SendReplyPlaceholder(context.Background(), "42", "99"); err != nil {
		t.Fatalf("SendReplyPlaceholder: %v", err)
	}
	sent := client.sentMessages()
	replyMsg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent[0] = %T, want MessageConfig", sent[0])
	}
	if !replyMsg.DisableNotification {
		t.Fatalf("reply placeholder DisableNotification = false, want true")
	}

	if _, err := b.SendThreadReplyPlaceholder(context.Background(), "42", "777", "99"); err != nil {
		t.Fatalf("SendThreadReplyPlaceholder: %v", err)
	}
	uploads := client.uploadRequests()
	if got := uploads[0].Params["disable_notification"]; got != "true" {
		t.Fatalf("thread reply placeholder disable_notification = %q, want true", got)
	}
}
