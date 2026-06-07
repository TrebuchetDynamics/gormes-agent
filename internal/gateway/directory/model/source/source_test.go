package source

import "testing"

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
