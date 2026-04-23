package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateMirror_WriteIncludesHomeChannelsAndDirectory(t *testing.T) {
	homes := NewHomeChannels()
	if _, ok := homes.SetFromInbound(InboundEvent{
		Platform: " Discord ",
		ChatID:   " ops-room ",
		ChatName: " Ops Room ",
		UserID:   " u7 ",
		UserName: " Ada ",
	}); !ok {
		t.Fatal("SetFromInbound returned ok=false")
	}

	dir := NewChannelDirectory()
	if _, ok := dir.UpdateFromInbound(InboundEvent{
		Platform: " Telegram ",
		ChatID:   " 42 ",
		ChatName: " Ops DM ",
		UserID:   " u1 ",
		UserName: " Grace ",
		ThreadID: " topic-1 ",
	}); !ok {
		t.Fatal("UpdateFromInbound returned ok=false")
	}
	if _, ok := dir.UpdateFromInbound(InboundEvent{
		Platform: "discord",
		ChatID:   "ops-room",
		ChatName: "Ops Room",
		UserID:   "u7",
		UserName: "Ada",
	}); !ok {
		t.Fatal("UpdateFromInbound discord returned ok=false")
	}

	path := filepath.Join(t.TempDir(), "gateway", "channel_directory.json")
	mirror := NewStateMirror(homes, dir, path)
	fixed := time.Date(2026, 4, 23, 8, 39, 9, 0, time.UTC)
	mirror.now = func() time.Time { return fixed }

	if err := mirror.Write(); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}

	var doc stateMirrorDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", raw, err)
	}

	if len(doc.HomeChannels) != 1 {
		t.Fatalf("home channel count = %d, want 1", len(doc.HomeChannels))
	}
	home := doc.HomeChannels[0]
	if home.Platform != "discord" || home.ChatID != "ops-room" || home.ChatName != "Ops Room" || home.SetByUserName != "Ada" {
		t.Fatalf("home channel = %+v, want normalized discord home metadata", home)
	}

	if len(doc.Directory) != 2 {
		t.Fatalf("directory count = %d, want 2", len(doc.Directory))
	}
	if doc.Directory[0].Platform != "discord" || doc.Directory[1].Platform != "telegram" {
		t.Fatalf("directory platforms = %+v, want deterministic sort order", doc.Directory)
	}
	if doc.Directory[1].ThreadID != "topic-1" {
		t.Fatalf("telegram directory entry = %+v, want thread metadata", doc.Directory[1])
	}
	if !doc.UpdatedAt.Equal(fixed) {
		t.Fatalf("updated_at = %s, want %s", doc.UpdatedAt, fixed)
	}
}
