package send

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTextReplyID(t *testing.T) {
	t.Run("optional blank", func(t *testing.T) {
		got, err := TextReplyID("", false)
		if err != nil {
			t.Fatalf("TextReplyID returned error: %v", err)
		}
		if got != 0 {
			t.Fatalf("TextReplyID = %d, want 0", got)
		}
	})

	t.Run("required invalid", func(t *testing.T) {
		if _, err := TextReplyID("abc", true); err == nil {
			t.Fatal("expected invalid reply ID error")
		}
	})

	t.Run("valid", func(t *testing.T) {
		got, err := TextReplyID("42", true)
		if err != nil {
			t.Fatalf("TextReplyID returned error: %v", err)
		}
		if got != 42 {
			t.Fatalf("TextReplyID = %d, want 42", got)
		}
	})
}

func TestThreadIDForTextSend(t *testing.T) {
	thread, include, err := ThreadIDForTextSend("7", "1")
	if err != nil {
		t.Fatalf("ThreadIDForTextSend returned error: %v", err)
	}
	if thread != 7 || !include {
		t.Fatalf("ThreadIDForTextSend = (%d, %v), want (7, true)", thread, include)
	}

	thread, include, err = ThreadIDForTextSend("1", "1")
	if err != nil {
		t.Fatalf("ThreadIDForTextSend general topic returned error: %v", err)
	}
	if thread != 0 || include {
		t.Fatalf("general topic = (%d, %v), want (0, false)", thread, include)
	}

	if _, _, err := ThreadIDForTextSend("abc", "1"); err == nil {
		t.Fatal("expected invalid thread ID error")
	}
}

func TestThreadIDForAction(t *testing.T) {
	thread, include, err := ThreadIDForAction("9")
	if err != nil {
		t.Fatalf("ThreadIDForAction returned error: %v", err)
	}
	if thread != 9 || !include {
		t.Fatalf("ThreadIDForAction = (%d, %v), want (9, true)", thread, include)
	}

	thread, include, err = ThreadIDForAction(" ")
	if err != nil {
		t.Fatalf("ThreadIDForAction blank returned error: %v", err)
	}
	if thread != 0 || include {
		t.Fatalf("blank thread = (%d, %v), want (0, false)", thread, include)
	}
}

func TestMessageParams(t *testing.T) {
	params := MessageParams(123, 45, "hello", "MarkdownV2")
	if params["chat_id"] != "123" || params["reply_to_message_id"] != "45" || params["text"] != "hello" || params["parse_mode"] != "MarkdownV2" {
		t.Fatalf("MessageParams = %#v", params)
	}
}

func TestMessageFromAPIResponse(t *testing.T) {
	msg, err := MessageFromAPIResponse(&tgbotapi.APIResponse{Result: []byte(`{"message_id":77}`)})
	if err != nil {
		t.Fatalf("MessageFromAPIResponse returned error: %v", err)
	}
	if msg.MessageID != 77 {
		t.Fatalf("MessageID = %d, want 77", msg.MessageID)
	}

	if _, err := MessageFromAPIResponse(&tgbotapi.APIResponse{Result: []byte(`{`)}); err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestErrorClassifiers(t *testing.T) {
	cases := []struct {
		name string
		fn   func(error) bool
		err  error
	}{
		{"markdown", IsMarkdownParseError, errors.New("Bad Request: can't parse Markdown")},
		{"thread", IsThreadNotFoundError, errors.New("Bad Request: thread not found")},
		{"timeout", IsTimedOutError, errors.New("read: i/o timeout")},
		{"network", IsTransientNetworkError, errors.New("connection reset by peer")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.fn(tc.err) {
				t.Fatalf("classifier returned false for %v", tc.err)
			}
			if tc.fn(nil) {
				t.Fatal("classifier returned true for nil")
			}
		})
	}
}
