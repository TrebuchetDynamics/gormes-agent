package directory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

func TestChannelDirectoryRefreshMergesAdapterInventory(t *testing.T) {
	root := t.TempDir()
	sources := NewChannelDirectorySourceStore(root)
	if err := sources.RememberSource(context.Background(), RememberedSourceEntry{Platform: "telegram", ID: "-100:7", Name: "Ops / topic 7", Type: "group", ChatID: "-100", ThreadID: "7"}); err != nil {
		t.Fatalf("RememberSource: %v", err)
	}
	refresher := ChannelDirectoryRefresher{
		Directory: NewChannelDirectoryStore(root),
		Sources:   sources,
		Now:       func() time.Time { return time.Date(2026, 4, 30, 20, 0, 0, 0, time.UTC) },
		Inventory: func(context.Context) ([]ChannelDirectoryAdapterSnapshot, error) {
			return []ChannelDirectoryAdapterSnapshot{
				{Platform: "slack", Entries: []ChannelDirectoryEntry{{ID: "C02", Name: "eng", Type: "channel"}}},
				{Platform: "discord", Entries: []ChannelDirectoryEntry{{ID: "D01", Name: "general", Guild: "Sages", Type: "channel"}}},
			}, nil
		},
	}
	got, evidence := refresher.Refresh(context.Background())
	if evidence.Code != "" {
		t.Fatalf("Refresh evidence = %+v, want none", evidence)
	}
	if got.UpdatedAt != "2026-04-30T20:00:00Z" {
		t.Fatalf("UpdatedAt = %q", got.UpdatedAt)
	}
	if len(got.Platforms["discord"]) != 1 || got.Platforms["discord"][0].ID != "D01" {
		t.Fatalf("discord entries = %+v", got.Platforms["discord"])
	}
	if len(got.Platforms["slack"]) != 1 || got.Platforms["slack"][0].ID != "C02" {
		t.Fatalf("slack entries = %+v", got.Platforms["slack"])
	}
	if len(got.Platforms["telegram"]) != 1 || got.Platforms["telegram"][0].ThreadID != "7" {
		t.Fatalf("telegram remembered entries = %+v", got.Platforms["telegram"])
	}
}

func TestChannelDirectoryRefreshReportsInvalidRememberedSources(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "channel_directory_sources.json"), []byte(`{bad json`), 0o600); err != nil {
		t.Fatalf("write invalid source ledger: %v", err)
	}
	refresher := ChannelDirectoryRefresher{
		Directory: NewChannelDirectoryStore(root),
		Sources:   NewChannelDirectorySourceStore(root),
		Inventory: func(context.Context) ([]ChannelDirectoryAdapterSnapshot, error) {
			return []ChannelDirectoryAdapterSnapshot{{Platform: "slack", Entries: []ChannelDirectoryEntry{{ID: "C01", Name: "ops"}}}}, nil
		},
	}

	got, evidence := refresher.Refresh(context.Background())
	if evidence.Code != model.EvidenceChannelDirectorySourcesInvalid {
		t.Fatalf("Refresh evidence = %+v, want %s", evidence, model.EvidenceChannelDirectorySourcesInvalid)
	}
	if len(got.Platforms["slack"]) != 1 || got.Platforms["slack"][0].ID != "C01" {
		t.Fatalf("adapter entries = %+v, want refresh to keep usable inventory", got.Platforms["slack"])
	}
	if len(got.Platforms["telegram"]) != 0 {
		t.Fatalf("telegram remembered entries = %+v, want invalid source ledger omitted", got.Platforms["telegram"])
	}
}

func TestChannelDirectoryRefreshFailurePreservesLastGood(t *testing.T) {
	root := t.TempDir()
	store := NewChannelDirectoryStore(root)
	lastGood := ChannelDirectory{UpdatedAt: "old", Platforms: map[string][]ChannelDirectoryEntry{"slack": {{ID: "C01", Name: "ops"}}}}
	if err := store.Save(lastGood); err != nil {
		t.Fatalf("Save last good: %v", err)
	}
	refresher := ChannelDirectoryRefresher{
		Directory: store,
		Inventory: func(context.Context) ([]ChannelDirectoryAdapterSnapshot, error) {
			return nil, os.ErrPermission
		},
	}
	got, evidence := refresher.Refresh(context.Background())
	if evidence.Code != model.EvidenceChannelDirectoryRefreshFailed {
		t.Fatalf("Refresh evidence = %+v, want %s", evidence, model.EvidenceChannelDirectoryRefreshFailed)
	}
	if got.Platforms["slack"][0].ID != "C01" {
		t.Fatalf("got = %+v, want last good preserved", got)
	}
	loaded, loadEvidence := store.Load()
	if loadEvidence.Code != "" || loaded.Platforms["slack"][0].ID != "C01" {
		t.Fatalf("loaded = %+v evidence=%+v, want persisted last good", loaded, loadEvidence)
	}
}

func TestChannelDirectoryStaleTargetInvalidation(t *testing.T) {
	dir := ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{"discord": {{ID: "fresh", Name: "general"}}}}
	_, evidence := dir.ValidateDeliveryTarget(gatewaydelivery.Target{Platform: "discord", ChatID: "old", IsExplicit: true})
	if evidence.Code != model.EvidenceChannelTargetStale {
		t.Fatalf("Validate stale evidence = %+v, want %s", evidence, model.EvidenceChannelTargetStale)
	}
	configuredHome, ok := gatewaydelivery.ResolveHomeTarget(gatewaydelivery.Target{Platform: "discord"}, gatewaydelivery.HomeTargets{"discord": {ChatID: "old"}})
	if !ok {
		t.Fatal("ResolveHomeTarget did not resolve configured home")
	}
	_, evidence = dir.ValidateDeliveryTarget(configuredHome)
	if evidence.Code != model.EvidenceChannelTargetStale {
		t.Fatalf("Validate stale configured home evidence = %+v, want %s", evidence, model.EvidenceChannelTargetStale)
	}
	fresh, evidence := dir.ValidateDeliveryTarget(gatewaydelivery.Target{Platform: "discord", ChatID: "fresh", IsExplicit: true})
	if evidence.Code != "" || fresh.ChatID != "fresh" {
		t.Fatalf("Validate fresh = %+v evidence=%+v", fresh, evidence)
	}
}

func TestChannelDirectoryRefreshSerializesWrites(t *testing.T) {
	root := t.TempDir()
	refresher := ChannelDirectoryRefresher{
		Directory: NewChannelDirectoryStore(root),
		Now:       func() time.Time { return time.Date(2026, 4, 30, 20, 1, 0, 0, time.UTC) },
		Inventory: func(context.Context) ([]ChannelDirectoryAdapterSnapshot, error) {
			return []ChannelDirectoryAdapterSnapshot{{Platform: "slack", Entries: []ChannelDirectoryEntry{{ID: "C01", Name: "ops"}}}}, nil
		},
	}
	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			refresher.Refresh(context.Background())
		}()
	}
	<-done
	<-done
	matches, err := filepath.Glob(filepath.Join(root, ".channel_directory-*.tmp"))
	if err != nil {
		t.Fatalf("glob tmp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("leftover temp files = %v", matches)
	}
	loaded, evidence := refresher.Directory.Load()
	if evidence.Code != "" || len(loaded.Platforms["slack"]) != 1 {
		t.Fatalf("loaded = %+v evidence=%+v", loaded, evidence)
	}
}
