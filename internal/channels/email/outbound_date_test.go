package email

import (
	"bufio"
	"bytes"
	"io"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestEmailOutboundDateHeader_AddsWhenMissing(t *testing.T) {
	fixed := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	delivery := Delivery{
		From:       "agent@gormes.test",
		To:         "user@example.test",
		Subject:    "Re: hi",
		Body:       "hello back",
		InReplyTo:  "<msg-1@example.test>",
		References: "<msg-1@example.test>",
	}
	raw := BuildDelivery(delivery, func() time.Time { return fixed })

	parsed, err := mail.ReadMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	date := parsed.Header.Get("Date")
	if date == "" {
		t.Fatal("Date header missing")
	}
	parsedDate, err := mail.ParseDate(date)
	if err != nil {
		t.Fatalf("Date %q is not RFC 5322: %v", date, err)
	}
	if !parsedDate.Equal(fixed) {
		t.Fatalf("Date = %v, want %v", parsedDate, fixed)
	}
	if dates := parsed.Header["Date"]; len(dates) != 1 {
		t.Fatalf("Date header count = %d, want exactly 1", len(dates))
	}
}

func TestEmailOutboundDateHeader_PreservesExisting(t *testing.T) {
	explicit := "Mon, 27 Apr 2026 12:00:00 +0000"
	delivery := Delivery{
		From:    "agent@gormes.test",
		To:      "user@example.test",
		Subject: "Re: hi",
		Body:    "hi",
		Date:    explicit,
	}
	clockShould := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	raw := BuildDelivery(delivery, func() time.Time { return clockShould })

	parsed, err := mail.ReadMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := parsed.Header.Get("Date"); got != explicit {
		t.Fatalf("Date = %q, want caller-supplied %q", got, explicit)
	}
	if dates := parsed.Header["Date"]; len(dates) != 1 {
		t.Fatalf("Date header count = %d, want exactly 1", len(dates))
	}
}

func TestEmailOutboundDateHeader_ReplyHeadersUnchanged(t *testing.T) {
	fixed := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	delivery := Delivery{
		From:       "agent@gormes.test",
		To:         "user@example.test",
		Subject:    "Re: thread",
		Body:       "yep",
		InReplyTo:  "<root@example.test>",
		References: "<root@example.test> <follow@example.test>",
	}
	raw := BuildDelivery(delivery, func() time.Time { return fixed })

	parsed, err := mail.ReadMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string]string{
		"From":        delivery.From,
		"To":          delivery.To,
		"Subject":     delivery.Subject,
		"In-Reply-To": delivery.InReplyTo,
		"References":  delivery.References,
	}
	for k, v := range want {
		if got := parsed.Header.Get(k); got != v {
			t.Fatalf("%s = %q, want %q", k, got, v)
		}
	}
	body, _ := io.ReadAll(parsed.Body)
	if got := strings.TrimSpace(string(body)); got != delivery.Body {
		t.Fatalf("body = %q, want %q", got, delivery.Body)
	}
}
