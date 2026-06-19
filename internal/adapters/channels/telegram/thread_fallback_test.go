package telegram

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// errThreadNotFound simulates Telegram's BadRequest "Message thread not found".
var errThreadNotFound = errors.New("Bad Request: message thread not found")

// errBadRequestNonThread simulates a non-thread BadRequest (e.g. chat not found).
var errBadRequestNonThread = errors.New("Bad Request: chat not found")

// errReplyNotFound simulates Telegram's BadRequest for a deleted reply target.
var errReplyNotFound = errors.New("Bad Request: message to be replied not found")

func TestBot_SendReply_RetriesWithoutReplyWhenTargetNotFound(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.SendFn = func(c tgbotapi.Chattable) (tgbotapi.Message, error) {
		callCount++
		msg, ok := c.(tgbotapi.MessageConfig)
		if !ok {
			t.Fatalf("send config = %T, want MessageConfig", c)
		}
		if callCount == 1 {
			if msg.ReplyToMessageID != 99 {
				t.Fatalf("first reply_to_message_id = %d, want 99", msg.ReplyToMessageID)
			}
			return tgbotapi.Message{}, errReplyNotFound
		}
		if msg.ReplyToMessageID != 0 {
			t.Fatalf("retry reply_to_message_id = %d, want cleared", msg.ReplyToMessageID)
		}
		return tgbotapi.Message{MessageID: 1000}, nil
	}
	b := New(Config{}, client, nil)

	msgID, err := b.SendReply(context.Background(), "42", "99", "hello reply")
	if err != nil {
		t.Fatalf("SendReply retry failed: %v", err)
	}
	if msgID != "1000" {
		t.Fatalf("msgID = %q, want 1000", msgID)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}
}

func TestBot_SendThreadReply_RetriesWithoutReplyWhenTargetNotFound(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount == 1 {
			if params["reply_to_message_id"] != "99" || params["message_thread_id"] != "777" {
				t.Fatalf("first params = %+v, want reply and thread", params)
			}
			return nil, errReplyNotFound
		}
		if _, ok := params["reply_to_message_id"]; ok {
			t.Fatalf("retry reply_to_message_id = %q, want omitted", params["reply_to_message_id"])
		}
		if params["message_thread_id"] != "777" {
			t.Fatalf("retry message_thread_id = %q, want retained", params["message_thread_id"])
		}
		return client.uploadSuccess(1003), nil
	}
	b := New(Config{}, client, nil)

	msgID, err := b.SendThreadReply(context.Background(), "42", "777", "99", "hello reply")
	if err != nil {
		t.Fatalf("SendThreadReply retry failed: %v", err)
	}
	if msgID != "1003" {
		t.Fatalf("msgID = %q, want 1003", msgID)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}
}

func TestBot_SendThread_RetriesWithoutThreadWhenNotFound(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount == 1 {
			// First call: thread "777" was included, Telegram rejects it.
			if params["message_thread_id"] != "777" {
				t.Fatalf("first call message_thread_id = %q, want 777", params["message_thread_id"])
			}
			return nil, errThreadNotFound
		}
		// Second call: retried without thread ID.
		if _, ok := params["message_thread_id"]; ok {
			t.Fatalf("second call message_thread_id = %q, want omitted", params["message_thread_id"])
		}
		return client.uploadSuccess(1001), nil
	}
	b := New(Config{}, client, nil)

	msgID, err := b.SendThread(context.Background(), "42", "777", "hello thread")
	if err != nil {
		t.Fatalf("SendThread retry failed: %v", err)
	}
	if msgID != "1001" {
		t.Fatalf("msgID = %q, want 1001", msgID)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2 (one fail, one retry)", callCount)
	}
}

func TestBot_SendThread_NonThreadBadRequestFailsImmediately(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		return nil, errBadRequestNonThread
	}
	b := New(Config{}, client, nil)

	_, err := b.SendThread(context.Background(), "42", "777", "hello")
	if err == nil {
		t.Fatal("expected non-thread BadRequest to fail immediately")
	}
	if !errors.Is(err, errBadRequestNonThread) {
		t.Fatalf("err = %v, want errBadRequestNonThread", err)
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1 (no retry for non-thread BadRequest)", callCount)
	}
}

func TestBot_SendThread_NoThreadFallsBackToSendNoRetry(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.SendFn = func(c tgbotapi.Chattable) (tgbotapi.Message, error) {
		callCount++
		return tgbotapi.Message{}, errBadRequestNonThread
	}
	b := New(Config{}, client, nil)

	_, err := b.SendThread(context.Background(), "42", "", "hello")
	if err == nil {
		t.Fatal("expected error to propagate")
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1 (empty thread falls back to Send)", callCount)
	}
}

func TestBot_SendThreadReply_RetriesWithoutThreadWhenNotFound(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount == 1 {
			if params["message_thread_id"] != "777" {
				t.Fatalf("first call message_thread_id = %q, want 777", params["message_thread_id"])
			}
			if params["reply_to_message_id"] != "99" {
				t.Fatalf("reply_to_message_id = %q, want 99", params["reply_to_message_id"])
			}
			return nil, errThreadNotFound
		}
		if _, ok := params["message_thread_id"]; ok {
			t.Fatalf("second call message_thread_id = %q, want omitted", params["message_thread_id"])
		}
		if params["reply_to_message_id"] != "99" {
			t.Fatalf("retry dropped reply_to_message_id = %q, want 99", params["reply_to_message_id"])
		}
		return client.uploadSuccess(1002), nil
	}
	b := New(Config{}, client, nil)

	msgID, err := b.SendThreadReply(context.Background(), "42", "777", "99", "hello reply")
	if err != nil {
		t.Fatalf("SendThreadReply retry failed: %v", err)
	}
	if msgID != "1002" {
		t.Fatalf("msgID = %q, want 1002", msgID)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}
}

func TestBot_SendThreadChatAction_RetriesWithoutThreadWhenNotFound(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount == 1 {
			if params["message_thread_id"] != "777" {
				t.Fatalf("first call message_thread_id = %q, want 777", params["message_thread_id"])
			}
			return nil, errThreadNotFound
		}
		if _, ok := params["message_thread_id"]; ok {
			t.Fatalf("second call message_thread_id = %q, want omitted", params["message_thread_id"])
		}
		return &tgbotapi.APIResponse{Ok: true}, nil
	}
	b := New(Config{}, client, nil)

	err := b.SendThreadChatAction(context.Background(), "-100123", "777", "typing")
	if err != nil {
		t.Fatalf("SendThreadChatAction retry failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2 (one fail, one retry)", callCount)
	}
}

func TestBot_SendThreadChatAction_GeneralTopicThreadNotFoundRetriesWithoutThread(t *testing.T) {
	// Typing for forum General with thread "1": thread is included for typing,
	// but Telegram may reject it. Retry without thread.
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount == 1 {
			if params["message_thread_id"] != "1" {
				t.Fatalf("first call message_thread_id = %q, want 1", params["message_thread_id"])
			}
			return nil, errThreadNotFound
		}
		if _, ok := params["message_thread_id"]; ok {
			t.Fatalf("second call message_thread_id = %q, want omitted", params["message_thread_id"])
		}
		return &tgbotapi.APIResponse{Ok: true}, nil
	}
	b := New(Config{}, client, nil)

	err := b.SendThreadChatAction(context.Background(), "-100123", "1", "typing")
	if err != nil {
		t.Fatalf("SendThreadChatAction retry failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}
}

func TestBot_SendThreadChatAction_NonThreadBadRequestFailsImmediately(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		return nil, errBadRequestNonThread
	}
	b := New(Config{}, client, nil)

	err := b.SendThreadChatAction(context.Background(), "-100123", "777", "typing")
	if err == nil {
		t.Fatal("expected non-thread BadRequest to fail immediately")
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}
}
