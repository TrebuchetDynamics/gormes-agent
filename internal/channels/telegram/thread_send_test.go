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
