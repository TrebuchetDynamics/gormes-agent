package cache

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

func TestDirectoryUpsertEntriesNormalizesPlatformAndSkipsIncompleteEntries(t *testing.T) {
	dir := NewDirectory("")

	merged := dir.UpsertEntries(" Telegram ",
		model.Entry{ID: " -100 ", Name: " Ops "},
		model.Entry{ID: " missing-name "},
	)

	if merged != 1 {
		t.Fatalf("merged = %d, want 1", merged)
	}
	entries := dir.Platforms["telegram"]
	if len(entries) != 1 || entries[0].ID != "-100" || entries[0].Name != "Ops" {
		t.Fatalf("telegram entries = %+v, want one normalized entry", entries)
	}
}

func TestDirectoryUpsertEntriesReplacesExistingEntryByID(t *testing.T) {
	dir := NewDirectory("")
	dir.UpsertEntries("telegram", model.Entry{ID: "-100", Name: "Ops"})

	merged := dir.UpsertEntries("telegram", model.Entry{ID: " -100 ", Name: " Ops Renamed "})

	if merged != 1 {
		t.Fatalf("merged = %d, want 1", merged)
	}
	entries := dir.Platforms["telegram"]
	if len(entries) != 1 || entries[0].Name != "Ops Renamed" {
		t.Fatalf("telegram entries = %+v, want replaced entry", entries)
	}
}
