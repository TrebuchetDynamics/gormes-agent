package tools

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestTermuxNotificationSendUsesFakeExecWithRedactedBoundedArgs(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	sender := TermuxNotificationSender{
		Env: map[string]string{
			"TERMUX_VERSION": "0.119.0",
			"PREFIX":         "/data/data/com.termux/files/usr",
			"HOME":           "/data/data/com.termux/files/home",
		},
		LookPath: func(name string) (string, error) {
			if name != "termux-notification" {
				return "", exec.ErrNotFound
			}
			return "/data/data/com.termux/files/usr/bin/termux-notification", nil
		},
		Run: func(_ context.Context, command string, args ...string) error {
			gotCommand = command
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	result := sender.Notify(context.Background(),
		"Gateway token=sk-testsecret1234567890",
		"Turn finished for TELEGRAM_BOT_TOKEN=123456:abcdefghijklmnopqrstuvwxyz with "+strings.Repeat("details ", 80),
	)

	if result.Status != TermuxNotificationStatusSent {
		t.Fatalf("Status = %q, want %q: %+v", result.Status, TermuxNotificationStatusSent, result)
	}
	if gotCommand != "/data/data/com.termux/files/usr/bin/termux-notification" {
		t.Fatalf("command = %q, want termux-notification path", gotCommand)
	}
	if len(gotArgs) != 4 || gotArgs[0] != "--title" || gotArgs[2] != "--content" {
		t.Fatalf("args = %#v, want --title/--content pairs", gotArgs)
	}
	joined := strings.Join(gotArgs, " ")
	for _, leak := range []string{"sk-testsecret1234567890", "123456:abcdefghijklmnopqrstuvwxyz"} {
		if strings.Contains(joined, leak) {
			t.Fatalf("notification args leaked secret %q: %#v", leak, gotArgs)
		}
	}
	if !strings.Contains(joined, "[redacted]") {
		t.Fatalf("notification args missing redaction marker: %#v", gotArgs)
	}
	if len([]rune(gotArgs[3])) > defaultTermuxNotificationBodyRunes {
		t.Fatalf("body length = %d runes, want <= %d", len([]rune(gotArgs[3])), defaultTermuxNotificationBodyRunes)
	}
	if !result.Redacted {
		t.Fatalf("Redacted = false, want true for secret/truncated notification")
	}
}

func TestTermuxNotificationMissingCommandReturnsUnavailableEvidence(t *testing.T) {
	called := false
	sender := TermuxNotificationSender{
		Env: map[string]string{
			"TERMUX_VERSION": "0.119.0",
			"PREFIX":         "/data/data/com.termux/files/usr",
			"HOME":           "/data/data/com.termux/files/home",
		},
		LookPath: func(string) (string, error) { return "", exec.ErrNotFound },
		Run: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}

	result := sender.Notify(context.Background(), "title", "body")
	if result.Status != TermuxNotificationStatusUnavailable {
		t.Fatalf("Status = %q, want %q: %+v", result.Status, TermuxNotificationStatusUnavailable, result)
	}
	if !strings.Contains(result.Message, "termux-notification missing") {
		t.Fatalf("Message = %q, want missing-command guidance", result.Message)
	}
	if called {
		t.Fatal("Run called even though termux-notification is missing")
	}
}

func TestNotificationTermuxNonTermuxSkipsWithoutRunner(t *testing.T) {
	called := false
	sender := TermuxNotificationSender{
		Env: map[string]string{
			"HOME": "/home/operator",
		},
		LookPath: func(string) (string, error) { return "/usr/bin/termux-notification", nil },
		Run: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}

	result := sender.Notify(context.Background(), "title", "body")
	if result.Status != TermuxNotificationStatusSkipped {
		t.Fatalf("Status = %q, want %q: %+v", result.Status, TermuxNotificationStatusSkipped, result)
	}
	if called {
		t.Fatal("Run called on non-Termux host")
	}
}

func TestTermuxNotificationCommandFailureReturnsUnavailableEvidence(t *testing.T) {
	sender := TermuxNotificationSender{
		Env: map[string]string{
			"TERMUX_VERSION": "0.119.0",
			"PREFIX":         "/data/data/com.termux/files/usr",
			"HOME":           "/data/data/com.termux/files/home",
		},
		LookPath: func(string) (string, error) {
			return "/data/data/com.termux/files/usr/bin/termux-notification", nil
		},
		Run: func(context.Context, string, ...string) error {
			return errors.New("raw shell error with token=sk-testsecret1234567890")
		},
	}

	result := sender.Notify(context.Background(), "title", "body")
	if result.Status != TermuxNotificationStatusUnavailable {
		t.Fatalf("Status = %q, want %q: %+v", result.Status, TermuxNotificationStatusUnavailable, result)
	}
	if strings.Contains(result.Message, "sk-testsecret1234567890") || strings.Contains(result.Message, "raw shell error") {
		t.Fatalf("Message leaked raw command error: %q", result.Message)
	}
}
