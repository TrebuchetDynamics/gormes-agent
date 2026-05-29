package discord

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDiscordReplyModeFirstAllOffAndInvalid(t *testing.T) {
	cases := []struct {
		name string
		mode string
		want []bool
	}{
		{name: "first default references only first send", mode: "", want: []bool{true, false}},
		{name: "first references only first send", mode: "first", want: []bool{true, false}},
		{name: "all references every send", mode: "all", want: []bool{true, true}},
		{name: "off references no sends", mode: "off", want: []bool{false, false}},
		{name: "invalid behaves like first", mode: "banana", want: []bool{true, false}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ms := newMockSession()
			b := New(Config{AllowedChannelID: "42", ReplyToMode: tc.mode}, ms, nil)
			for _, text := range []string{"chunk one", "chunk two"} {
				if _, err := b.SendReply(context.Background(), "42", "origin-99", text); err != nil {
					t.Fatalf("SendReply: %v", err)
				}
			}
			sent := ms.complexSnapshot()
			if len(sent) != len(tc.want) {
				t.Fatalf("complex sends = %d, want %d", len(sent), len(tc.want))
			}
			for i, wantRef := range tc.want {
				hasRef := sent[i].Data != nil && sent[i].Data.Reference != nil
				if hasRef != wantRef {
					t.Fatalf("send %d Reference present = %t, want %t; sent=%+v", i, hasRef, wantRef, sent)
				}
				if hasRef && sent[i].Data.Reference.MessageID != "origin-99" {
					t.Fatalf("send %d Reference.MessageID = %q, want origin-99", i, sent[i].Data.Reference.MessageID)
				}
			}
		})
	}
}

func TestDiscordReplyModeDeletedReferenceFallback(t *testing.T) {
	ms := newMockSession()
	ms.sendErrWhenReference = errors.New("HTTP 400 Bad Request, error code: 10008: Unknown Message")
	b := New(Config{AllowedChannelID: "42", ReplyToMode: "first"}, ms, nil)

	if _, err := b.SendReply(context.Background(), "42", "deleted-99", "hello"); err != nil {
		t.Fatalf("SendReply: %v", err)
	}
	sent := ms.complexSnapshot()
	if len(sent) != 1 {
		t.Fatalf("complex sends = %+v, want one successful fallback send", sent)
	}
	if sent[0].Data == nil || sent[0].Data.Reference != nil {
		t.Fatalf("fallback send Reference = %+v, want nil", sent[0].Data)
	}
}

func TestDiscordReplyModeNonReferenceErrorDoesNotFallback(t *testing.T) {
	ms := newMockSession()
	ms.sendErrWhenReference = errors.New("permission denied")
	b := New(Config{AllowedChannelID: "42", ReplyToMode: "first"}, ms, nil)

	_, err := b.SendReply(context.Background(), "42", "origin-99", "hello")
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("SendReply err = %v, want permission denied", err)
	}
}
