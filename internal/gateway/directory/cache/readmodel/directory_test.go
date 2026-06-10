package readmodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

func TestStoreSaveSanitizesUpdatedAtBeforePersisting(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	if err := store.Save(Directory{UpdatedAt: "now\nforged=true", Platforms: map[string][]model.Entry{"telegram": {{ID: "42", Name: "Ops"}}}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, channelDirectoryFileName))
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if strings.Contains(string(raw), `now\nforged=true`) || !strings.Contains(string(raw), `"updated_at": "now forged=true"`) {
		t.Fatalf("persisted directory = %s, want sanitized updated_at", raw)
	}
}

func TestDirectoryUpsertEntriesRejectsControlCharacterPlatform(t *testing.T) {
	dir := NewDirectory("")

	merged := dir.UpsertEntries("telegram\nforged", model.Entry{ID: "42", Name: "Ops"})

	if merged != 0 {
		t.Fatalf("merged = %d, want 0", merged)
	}
	if len(dir.Platforms) != 0 {
		t.Fatalf("platforms = %+v, want no control-character platform bucket", dir.Platforms)
	}
}

func TestStoreLoadRemovesHiddenFormattingPlatformKeys(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, channelDirectoryFileName), []byte(`{"platforms":{"tele\u202egram":[{"id":"42","name":"Ops"}]}}`), 0o600); err != nil {
		t.Fatalf("write directory: %v", err)
	}

	dir, evidence := NewStore(root).Load()
	if evidence.Code != "" {
		t.Fatalf("Load evidence = %+v, want none", evidence)
	}
	if _, ok := dir.Platforms["tele\u202egram"]; ok {
		t.Fatalf("platforms = %+v, want hidden-formatting key normalized away", dir.Platforms)
	}
	if got := dir.Platforms["telegram"]; len(got) != 1 || got[0].ID != "42" {
		t.Fatalf("telegram entries = %+v, want hidden-formatting platform normalized to telegram", got)
	}
}

func TestStoreLoadRejectsControlCharacterPlatformKeys(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, channelDirectoryFileName), []byte(`{"platforms":{"telegram\nforged":[{"id":"42","name":"Ops"}],"telegram":[{"id":"43","name":"Good"}]}}`), 0o600); err != nil {
		t.Fatalf("write directory: %v", err)
	}

	dir, evidence := NewStore(root).Load()
	if evidence.Code != "" {
		t.Fatalf("Load evidence = %+v, want none", evidence)
	}
	if _, ok := dir.Platforms["telegram\nforged"]; ok {
		t.Fatalf("platforms = %+v, want control-character key dropped", dir.Platforms)
	}
	if got := dir.Platforms["telegram"]; len(got) != 1 || got[0].ID != "43" {
		t.Fatalf("telegram entries = %+v, want only valid platform entries", got)
	}
}

func TestStoreLoadSanitizesDecodedUpdatedAt(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, channelDirectoryFileName), []byte(`{"updated_at":"now\nforged=true","platforms":{"telegram":[{"id":"42","name":"Ops"}]}}`), 0o600); err != nil {
		t.Fatalf("write directory: %v", err)
	}

	dir, evidence := NewStore(root).Load()
	if evidence.Code != "" {
		t.Fatalf("Load evidence = %+v, want none", evidence)
	}
	if strings.Contains(dir.UpdatedAt, "\n") || dir.UpdatedAt != "now forged=true" {
		t.Fatalf("UpdatedAt = %q, want sanitized single-line value", dir.UpdatedAt)
	}
}

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
