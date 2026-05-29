package email

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestEmailAllowlistEmptyPermitsAll(t *testing.T) {
	raw := testAllowlistEmail("Alice Example <alice@example.com>", "Deploy to production", "hello from plain")
	want, ok, err := NormalizeInbound(raw)
	if err != nil || !ok {
		t.Fatalf("NormalizeInbound() = ok %v err %v, want accepted baseline", ok, err)
	}

	var threadCalls, dispatchCalls int
	got, err := DispatchInboundWithAllowlist(raw, InboundDispatchOptions{
		BuildThreadContext: func(in NormalizedInbound) error {
			threadCalls++
			if in.Reply != want.Reply {
				t.Fatalf("thread reply = %+v, want %+v", in.Reply, want.Reply)
			}
			return nil
		},
		Dispatch: func(ev gateway.InboundEvent) error {
			dispatchCalls++
			if !reflect.DeepEqual(ev, want.Event) {
				t.Fatalf("dispatch event = %+v, want %+v", ev, want.Event)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DispatchInboundWithAllowlist() error = %v", err)
	}
	if !got.Accepted || got.Dropped || !got.Normalized {
		t.Fatalf("result = %+v, want accepted normalized event", got)
	}
	if threadCalls != 1 || dispatchCalls != 1 {
		t.Fatalf("thread/dispatch calls = %d/%d, want 1/1", threadCalls, dispatchCalls)
	}
}

func TestEmailAllowlistAllowedSenderProceeds(t *testing.T) {
	raw := testAllowlistEmail("ALICE Example <Alice@Example.COM>", "Re: Existing thread", "follow up")

	var threadCalls, dispatchCalls int
	got, err := DispatchInboundWithAllowlist(raw, InboundDispatchOptions{
		AllowedSenders: []string{"alice@example.com"},
		BuildThreadContext: func(in NormalizedInbound) error {
			threadCalls++
			if in.Reply.To != "alice@example.com" {
				t.Fatalf("Reply.To = %q, want alice@example.com", in.Reply.To)
			}
			return nil
		},
		Dispatch: func(ev gateway.InboundEvent) error {
			dispatchCalls++
			if ev.UserID != "alice@example.com" || ev.ChatID != "alice@example.com" {
				t.Fatalf("event IDs = %q/%q, want alice@example.com", ev.UserID, ev.ChatID)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DispatchInboundWithAllowlist() error = %v", err)
	}
	if !got.Accepted || got.Dropped || got.Evidence.Code != "" {
		t.Fatalf("result = %+v, want accepted without evidence", got)
	}
	if threadCalls != 1 || dispatchCalls != 1 {
		t.Fatalf("thread/dispatch calls = %d/%d, want 1/1", threadCalls, dispatchCalls)
	}
}

func TestEmailAllowlistDeniedSenderDropsBeforeDispatch(t *testing.T) {
	raw := testAllowlistEmail("Mallory <mallory@example.com>", "Attempt", "do the thing")

	var threadCalls, dispatchCalls int
	got, err := DispatchInboundWithAllowlist(raw, InboundDispatchOptions{
		AllowedSenders: []string{"alice@example.com"},
		BuildThreadContext: func(NormalizedInbound) error {
			threadCalls++
			return nil
		},
		Dispatch: func(gateway.InboundEvent) error {
			dispatchCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DispatchInboundWithAllowlist() error = %v", err)
	}
	if got.Accepted || !got.Dropped || got.Normalized {
		t.Fatalf("result = %+v, want dropped before normalization/dispatch", got)
	}
	if threadCalls != 0 || dispatchCalls != 0 {
		t.Fatalf("thread/dispatch calls = %d/%d, want 0/0", threadCalls, dispatchCalls)
	}
	if got.Evidence.Code != "email_sender_denied" {
		t.Fatalf("evidence code = %q, want email_sender_denied", got.Evidence.Code)
	}
	if got.Evidence.Domain != "example.com" {
		t.Fatalf("evidence domain = %q, want example.com", got.Evidence.Domain)
	}
}

func TestEmailAllowlistDeniedSenderRedactsEvidence(t *testing.T) {
	raw := strings.Join([]string{
		"From: Private Person <private.user+ticket@example.com>",
		"To: gormes@example.com",
		"Subject: Sensitive",
		"Message-ID: <sensitive@example.com>",
		"Authorization: Bearer should-not-leak",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"body-secret-should-not-leak",
		"",
	}, "\r\n")

	got, err := DispatchInboundWithAllowlist([]byte(raw), InboundDispatchOptions{
		AllowedSenders: []string{"trusted@example.com"},
		BuildThreadContext: func(NormalizedInbound) error {
			t.Fatal("BuildThreadContext called for denied sender")
			return nil
		},
		Dispatch: func(gateway.InboundEvent) error {
			t.Fatal("Dispatch called for denied sender")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DispatchInboundWithAllowlist() error = %v", err)
	}
	artifact := fmt.Sprintf("%+v", got.Evidence)
	for _, leaked := range []string{
		"private.user+ticket@example.com",
		"body-secret-should-not-leak",
		"should-not-leak",
	} {
		if strings.Contains(artifact, leaked) {
			t.Fatalf("evidence leaked %q: %+v", leaked, got.Evidence)
		}
	}
	if got.Evidence.Sender == "" || !strings.Contains(got.Evidence.Sender, "@example.com") {
		t.Fatalf("redacted sender = %q, want bounded domain evidence", got.Evidence.Sender)
	}
}

func testAllowlistEmail(from, subject, body string) []byte {
	return []byte(strings.Join([]string{
		"From: " + from,
		"To: gormes@example.com",
		"Subject: " + subject,
		"Message-ID: <allowlist@example.com>",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
		"",
	}, "\r\n"))
}
