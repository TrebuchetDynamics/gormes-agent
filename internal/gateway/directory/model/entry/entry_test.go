package entry

import "testing"

func TestNormalizeQueryTrimsBeforeChannelMarker(t *testing.T) {
	for raw, want := range map[string]string{
		"#General":   "general",
		" #General ": "general",
		"General":    "general",
	} {
		if got := NormalizeQuery(raw); got != want {
			t.Fatalf("NormalizeQuery(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestUpsertValidEntryNormalizesAndSkipsIncompleteEntries(t *testing.T) {
	entries, ok := UpsertValidEntry(nil, Entry{ID: " 123 ", Name: " Ops ", Type: " group "})
	if !ok {
		t.Fatalf("UpsertValidEntry ok = false, want true")
	}
	if len(entries) != 1 || entries[0].ID != "123" || entries[0].Name != "Ops" || entries[0].Type != "group" {
		t.Fatalf("entries = %+v, want normalized inserted entry", entries)
	}

	entries, ok = UpsertValidEntry(entries, Entry{ID: "123", Name: "Ops Renamed"})
	if !ok {
		t.Fatalf("UpsertValidEntry replacement ok = false, want true")
	}
	if len(entries) != 1 || entries[0].Name != "Ops Renamed" {
		t.Fatalf("entries = %+v, want replacement by ID", entries)
	}

	entries, ok = UpsertValidEntry(entries, Entry{ID: "456"})
	if ok {
		t.Fatalf("UpsertValidEntry incomplete ok = true, want false")
	}
	if len(entries) != 1 || entries[0].ID != "123" {
		t.Fatalf("entries = %+v, want unchanged entries after incomplete entry", entries)
	}
}
