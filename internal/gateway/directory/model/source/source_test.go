package source

import "testing"

func TestRememberedSourceEntryFromSourcePreservesChatTopic(t *testing.T) {
	entry := RememberedSourceEntryFromSource(Source{
		Platform:  " Telegram ",
		ChatID:    " -100 ",
		ChatName:  " Ops ",
		ThreadID:  " 42 ",
		ChatTopic: " Release War Room ",
	})

	if entry.ChatTopic != "Release War Room" {
		t.Fatalf("ChatTopic = %q, want source chat topic preserved", entry.ChatTopic)
	}
	if got := entry.ChannelDirectoryEntry().ChatTopic; got != "Release War Room" {
		t.Fatalf("ChannelDirectoryEntry().ChatTopic = %q, want source chat topic preserved", got)
	}
}

func TestRememberedSourceEntryIDAvoidsColonDelimiterCollisions(t *testing.T) {
	first := RememberedSourceEntryFromSource(Source{Platform: "telegram", ChatID: "room:thread", ThreadID: "x"})
	second := RememberedSourceEntryFromSource(Source{Platform: "telegram", ChatID: "room", ThreadID: "thread:x"})
	if first.ID == second.ID {
		t.Fatalf("remembered source ID collision for colon-bearing chat/thread IDs: %q", first.ID)
	}
	if got := RememberedSourceEntryFromSource(Source{Platform: "telegram", ChatID: "room", ThreadID: "thread"}).ID; got != "room:thread" {
		t.Fatalf("normal remembered source ID = %q, want legacy readable room:thread", got)
	}
}

func TestUpsertRememberedSourceEntryNormalizesAndReplacesByID(t *testing.T) {
	entries, ok := UpsertRememberedSourceEntry(nil, RememberedSourceEntry{Platform: " Telegram ", ChatID: " -100 ", ChatName: " Ops "})
	if !ok {
		t.Fatalf("UpsertRememberedSourceEntry ok = false, want true")
	}
	if len(entries) != 1 || entries[0].Platform != "telegram" || entries[0].ID != "-100" || entries[0].Name != "Ops" {
		t.Fatalf("entries = %+v, want normalized inserted source", entries)
	}

	entries, ok = UpsertRememberedSourceEntry(entries, RememberedSourceEntry{Platform: "telegram", ID: " -100 ", Name: " Ops Renamed "})
	if !ok {
		t.Fatalf("UpsertRememberedSourceEntry replacement ok = false, want true")
	}
	if len(entries) != 1 || entries[0].Name != "Ops Renamed" {
		t.Fatalf("entries = %+v, want replacement by normalized ID", entries)
	}

	entries, ok = UpsertRememberedSourceEntry(entries, RememberedSourceEntry{Platform: "telegram"})
	if ok {
		t.Fatalf("UpsertRememberedSourceEntry incomplete ok = true, want false")
	}
	if len(entries) != 1 || entries[0].ID != "-100" {
		t.Fatalf("entries = %+v, want unchanged entries after incomplete source", entries)
	}
}
