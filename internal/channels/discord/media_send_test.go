package discord

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_SendMediaSendsDiscordFileAttachment(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "forum-100"}, ms, nil)
	path := writeDiscordTestMedia(t, "photo.png", "image-bytes")

	msgID, err := b.SendMedia(context.Background(), "forum-100", "reply-99", gateway.OutboundMedia{
		Path:     path,
		Kind:     gateway.OutboundMediaKindImage,
		ThreadID: "thread-200",
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if msgID == "" {
		t.Fatal("msgID empty")
	}

	sent := ms.complexSnapshot()
	if len(sent) != 1 {
		t.Fatalf("complex sends = %+v, want one file send", sent)
	}
	if sent[0].ChannelID != "thread-200" {
		t.Fatalf("ChannelID = %q, want thread-200", sent[0].ChannelID)
	}
	if len(sent[0].FileNames) != 1 || sent[0].FileNames[0] != "photo.png" {
		t.Fatalf("FileNames = %+v, want photo.png", sent[0].FileNames)
	}
	if len(sent[0].FileBytes) != 1 || string(sent[0].FileBytes[0]) != "image-bytes" {
		t.Fatalf("FileBytes = %q, want image-bytes", sent[0].FileBytes)
	}
	if sent[0].Data == nil || sent[0].Data.Reference == nil || sent[0].Data.Reference.MessageID != "reply-99" {
		t.Fatalf("Reference = %+v, want reply-99", sent[0].Data)
	}
	if sent[0].Data.Reference.ChannelID != "thread-200" {
		t.Fatalf("Reference.ChannelID = %q, want thread-200", sent[0].Data.Reference.ChannelID)
	}
}

func TestBot_SendMediaMissingDiscordFileIsRedacted(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	missingPath := filepath.Join(t.TempDir(), "private", "missing.pdf")
	_, err := b.SendMedia(context.Background(), "42", "", gateway.OutboundMedia{Path: missingPath, Kind: gateway.OutboundMediaKindDocument})
	if err == nil {
		t.Fatal("SendMedia error = nil, want missing-file error")
	}
	if strings.Contains(err.Error(), missingPath) || strings.Contains(err.Error(), "missing.pdf") {
		t.Fatalf("error leaked local path: %v", err)
	}
}

func writeDiscordTestMedia(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
