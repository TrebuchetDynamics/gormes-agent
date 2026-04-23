package gateway

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestChannelDirectory_UpdateLookupAndInvalidateRenamedChat(t *testing.T) {
	dir := NewChannelDirectory()

	entry, ok := dir.UpdateFromInbound(InboundEvent{
		Platform: " Telegram ",
		ChatID:   " -100123 ",
		ChatName: " Ops Room ",
		UserID:   " 7 ",
		UserName: " Ada ",
		ThreadID: " 99 ",
	})
	if !ok {
		t.Fatal("UpdateFromInbound returned ok=false for a complete event")
	}
	if entry.Platform != "telegram" || entry.ChatID != "-100123" || entry.ChatName != "Ops Room" ||
		entry.UserID != "7" || entry.UserName != "Ada" || entry.ThreadID != "99" {
		t.Fatalf("entry = %+v, want normalized inbound metadata", entry)
	}

	got, ok := dir.Lookup("telegram", "Ops Room")
	if !ok {
		t.Fatal("Lookup by chat name failed")
	}
	if got.ChatID != "-100123" {
		t.Fatalf("Lookup by chat name ChatID = %q, want -100123", got.ChatID)
	}
	got, ok = dir.Lookup("TELEGRAM", " -100123 ")
	if !ok {
		t.Fatal("Lookup by chat ID failed")
	}
	if got.ChatName != "Ops Room" {
		t.Fatalf("Lookup by chat ID ChatName = %q, want Ops Room", got.ChatName)
	}

	dir.UpdateFromInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "-100123",
		ChatName: "Ops Renamed",
		UserID:   "7",
		UserName: "Ada Lovelace",
	})

	if _, ok := dir.Lookup("telegram", "Ops Room"); ok {
		t.Fatal("stale chat-name alias still resolves after rename")
	}
	got, ok = dir.Lookup("telegram", "Ops Renamed")
	if !ok {
		t.Fatal("renamed chat alias did not resolve")
	}
	if got.UserName != "Ada Lovelace" {
		t.Fatalf("renamed chat UserName = %q, want Ada Lovelace", got.UserName)
	}
}

func TestChannelDirectory_RenameDoesNotDeleteAliasClaimedByAnotherChat(t *testing.T) {
	dir := NewChannelDirectory()
	dir.UpdateFromInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "1",
		ChatName: "Ops",
	})
	dir.UpdateFromInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "2",
		ChatName: "Ops",
	})

	dir.UpdateFromInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "1",
		ChatName: "Archive",
	})

	got, ok := dir.Lookup("telegram", "Ops")
	if !ok {
		t.Fatal("shared alias disappeared after a different chat was renamed")
	}
	if got.ChatID != "2" {
		t.Fatalf("shared alias ChatID = %q, want 2", got.ChatID)
	}
}

func TestManager_InboundUpdatesChannelDirectory(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	dir := NewChannelDirectory()

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:     map[string]string{"telegram": "42"},
		ChannelDirectory: dir,
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		ChatName: "Ops DM",
		UserID:   "u1",
		UserName: "Operator",
		Kind:     EventSubmit,
		Text:     "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		_, ok := dir.Lookup("telegram", "Ops DM")
		return ok
	})
	got, ok := dir.Lookup("telegram", "42")
	if !ok {
		t.Fatal("manager did not record inbound chat ID")
	}
	if got.UserID != "u1" || got.UserName != "Operator" {
		t.Fatalf("directory entry = %+v, want inbound contact metadata", got)
	}
}
