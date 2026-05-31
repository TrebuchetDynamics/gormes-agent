package directory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

func TestChannelDirectoryAtomicWriteAndLoad(t *testing.T) {
	root := t.TempDir()
	store := NewChannelDirectoryStore(root)
	oldDir := ChannelDirectory{UpdatedAt: "2026-01-01T00:00:00Z", Platforms: map[string][]ChannelDirectoryEntry{
		"telegram": {{ID: "old", Name: "Alice", Type: "dm"}},
	}}
	if err := store.Save(oldDir); err != nil {
		t.Fatalf("Save old directory: %v", err)
	}
	if err := store.SaveWithWriter(ChannelDirectory{UpdatedAt: "2026-01-02T00:00:00Z", Platforms: map[string][]ChannelDirectoryEntry{
		"telegram": {{ID: "new", Name: "Bob", Type: "dm"}},
	}}, func(path string, data []byte, perm os.FileMode) error {
		if err := os.WriteFile(path, []byte(`{\"updated_at\":`), perm); err != nil {
			return err
		}
		return os.ErrInvalid
	}); err == nil {
		t.Fatal("SaveWithWriter error = nil, want injected failure")
	}
	got, evidence := store.Load()
	if evidence.Code != "" {
		t.Fatalf("Load evidence = %+v, want none", evidence)
	}
	if got.Platforms["telegram"][0].ID != "old" {
		t.Fatalf("loaded id = %q, want old cache preserved", got.Platforms["telegram"][0].ID)
	}
}

func TestChannelDirectoryResolveCompositeThreadTarget(t *testing.T) {
	dir := ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{
		"telegram": {{ID: "-1001:17585", Name: "Coaching Chat / topic 17585", Type: "group", ChatID: "-1001", ThreadID: "17585", ChatTopic: "topic 17585"}},
	}}
	got, evidence := dir.Resolve("telegram", "Coaching Chat / topic 17585")
	if evidence.Code != "" {
		t.Fatalf("Resolve evidence = %+v, want none", evidence)
	}
	want := gatewaydelivery.Target{Platform: "telegram", ChatID: "-1001", ThreadID: "17585", IsExplicit: true}
	if got != want {
		t.Fatalf("Resolve = %+v, want %+v", got, want)
	}
	byID, evidence := dir.Resolve("telegram", "-1001:17585")
	if evidence.Code != "" || byID != want {
		t.Fatalf("Resolve composite id = %+v evidence=%+v, want %+v", byID, evidence, want)
	}
}

func TestChannelDirectoryResolveDiscordGuildQualifiedName(t *testing.T) {
	dir := ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{
		"discord": {
			{ID: "111", Name: "general", Guild: "ServerA", Type: "channel"},
			{ID: "222", Name: "general", Guild: "ServerB", Type: "channel"},
		},
	}}
	for raw, wantID := range map[string]string{"ServerA/general": "111", "ServerB/general": "222"} {
		got, evidence := dir.Resolve("discord", raw)
		if evidence.Code != "" {
			t.Fatalf("Resolve(%q) evidence = %+v", raw, evidence)
		}
		if got.ChatID != wantID || got.Platform != "discord" {
			t.Fatalf("Resolve(%q) = %+v, want chat %s", raw, got, wantID)
		}
	}
}

func TestChannelDirectoryResolveDuplicateExactNamesAreAmbiguous(t *testing.T) {
	dir := ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{
		"discord": {
			{ID: "111", Name: "general", Guild: "ServerA", Type: "channel"},
			{ID: "222", Name: "general", Guild: "ServerB", Type: "channel"},
		},
	}}
	for _, raw := range []string{"general", "#general"} {
		got, evidence := dir.Resolve("discord", raw)
		if got.ChatID != "" {
			t.Fatalf("Resolve(%q) = %+v, want empty ambiguous target", raw, got)
		}
		if evidence.Code != model.EvidenceChannelTargetAmbiguous {
			t.Fatalf("Resolve(%q) evidence = %+v, want %s", raw, evidence, model.EvidenceChannelTargetAmbiguous)
		}
	}
}

func TestChannelDirectoryResolveAmbiguousPrefixReturnsMissing(t *testing.T) {
	dir := ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{
		"slack": {
			{ID: "C01", Name: "eng-backend", Type: "channel"},
			{ID: "C02", Name: "eng-frontend", Type: "channel"},
		},
	}}
	got, evidence := dir.Resolve("slack", "eng")
	if got.ChatID != "" {
		t.Fatalf("Resolve ambiguous target = %+v, want empty", got)
	}
	if evidence.Code != model.EvidenceChannelTargetAmbiguous {
		t.Fatalf("Resolve ambiguous evidence = %+v, want %s", evidence, model.EvidenceChannelTargetAmbiguous)
	}
}

func TestChannelDirectoryLookupTypeAndDisplay(t *testing.T) {
	dir := ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{
		"discord": {
			{ID: "100", Name: "ideas", Guild: "Server1", Type: "forum"},
			{ID: "200", Name: "general", Guild: "Server1", Type: "channel"},
		},
		"telegram": {
			{ID: "123", Name: "Alice", Type: "dm"},
			{ID: "-1001:17585", Name: "Coaching Chat / topic 17585", Type: "group", ChatID: "-1001", ThreadID: "17585"},
		},
	}}
	if got := dir.LookupType("discord", "100"); got != "forum" {
		t.Fatalf("LookupType forum = %q", got)
	}
	if got := dir.LookupType("telegram", "-1001:17585"); got != "group" {
		t.Fatalf("LookupType composite = %q", got)
	}
	display := dir.FormatForDisplay()
	for _, want := range []string{"Available messaging targets:", "Discord (Server1):", "discord:#ideas", "Telegram:", "telegram:Alice (dm)", "telegram:Coaching Chat / topic 17585 (group)", `Use these as the "target" parameter when sending.`} {
		if !strings.Contains(display, want) {
			t.Fatalf("display missing %q in:\n%s", want, display)
		}
	}
}

func TestChannelDirectoryLoadMissingAndInvalidEvidence(t *testing.T) {
	store := NewChannelDirectoryStore(t.TempDir())
	_, evidence := store.Load()
	if evidence.Code != model.EvidenceChannelDirectoryMissing {
		t.Fatalf("missing evidence = %+v", evidence)
	}
	if err := os.WriteFile(filepath.Join(store.Root(), "channel_directory.json"), []byte(`{bad json`), 0o600); err != nil {
		t.Fatalf("write invalid fixture: %v", err)
	}
	_, evidence = store.Load()
	if evidence.Code != model.EvidenceChannelDirectoryInvalid {
		t.Fatalf("invalid evidence = %+v", evidence)
	}
}
