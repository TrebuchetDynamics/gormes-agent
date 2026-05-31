package refresh

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

func TestBuildDirectoryMergesSnapshotsAndRememberedSources(t *testing.T) {
	ledger := model.RememberedSourceLedger{Platforms: map[string][]model.RememberedSourceEntry{
		"telegram": {{Platform: "telegram", ID: "-100:7", Name: "Ops / topic 7", Type: "thread", ChatID: "-100", ThreadID: "7"}},
	}}
	dir := buildDirectory("now", []AdapterSnapshot{
		{Platform: " slack ", Entries: []model.Entry{{ID: "C02", Name: "zeta"}, {ID: "C01", Name: "alpha"}}},
	}, ledger, true)

	if dir.UpdatedAt != "now" {
		t.Fatalf("UpdatedAt = %q, want now", dir.UpdatedAt)
	}
	if got := dir.Platforms["slack"]; len(got) != 2 || got[0].ID != "C01" || got[1].ID != "C02" {
		t.Fatalf("slack entries = %+v, want sorted merged adapter entries", got)
	}
	if got := dir.Platforms["telegram"]; len(got) != 1 || got[0].ThreadID != "7" {
		t.Fatalf("telegram entries = %+v, want remembered source merged", got)
	}
}

func TestBuildDirectorySkipsRememberedSourcesWhenLedgerInvalid(t *testing.T) {
	ledger := model.RememberedSourceLedger{Platforms: map[string][]model.RememberedSourceEntry{
		"telegram": {{Platform: "telegram", ID: "-100", Name: "Ops"}},
	}}
	dir := buildDirectory("now", []AdapterSnapshot{
		{Platform: "slack", Entries: []model.Entry{{ID: "C01", Name: "alpha"}}},
	}, ledger, false)

	if len(dir.Platforms["slack"]) != 1 {
		t.Fatalf("slack entries = %+v, want adapter inventory retained", dir.Platforms["slack"])
	}
	if got := dir.Platforms["telegram"]; len(got) != 0 {
		t.Fatalf("telegram entries = %+v, want invalid remembered ledger skipped", got)
	}
}
