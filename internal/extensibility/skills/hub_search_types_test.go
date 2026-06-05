package skills

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"
)

func TestHubSearchResultZeroValue(t *testing.T) {
	var zero HubSearchResult

	if zero.Name != "" {
		t.Errorf("zero Name = %q, want empty", zero.Name)
	}
	if zero.Description != "" {
		t.Errorf("zero Description = %q, want empty", zero.Description)
	}
	if zero.Source != "" {
		t.Errorf("zero Source = %q, want empty", zero.Source)
	}
	if zero.InstallID != "" {
		t.Errorf("zero InstallID = %q, want empty", zero.InstallID)
	}
	if zero.Score != 0 {
		t.Errorf("zero Score = %v, want 0", zero.Score)
	}

	raw, err := json.Marshal(zero)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	got := map[string]any{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	wantFields := []string{"name", "description", "source", "install_id", "score"}
	gotFields := make([]string, 0, len(got))
	for k := range got {
		gotFields = append(gotFields, k)
	}
	sort.Strings(gotFields)
	sortedWant := append([]string(nil), wantFields...)
	sort.Strings(sortedWant)
	if !reflect.DeepEqual(gotFields, sortedWant) {
		t.Fatalf("zero JSON keys = %v, want %v (raw=%s)", gotFields, sortedWant, raw)
	}

	wantValues := map[string]any{
		"name":        "",
		"description": "",
		"source":      "",
		"install_id":  "",
		"score":       float64(0),
	}
	if !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("zero JSON values = %v, want %v", got, wantValues)
	}
}

func TestInMemoryRegistryProviderSnapshot(t *testing.T) {
	input := []HubSearchResult{
		{Name: "zeta", Description: "z skill", Source: "fixture", InstallID: "fixture/zeta", Score: 0.30},
		{Name: "alpha", Description: "a skill", Source: "fixture", InstallID: "fixture/alpha", Score: 0.90},
		{Name: "mu", Description: "m skill", Source: "fixture", InstallID: "fixture/mu", Score: 0.60},
	}

	provider := NewInMemoryRegistryProvider(input, nil)

	first, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}

	want := []HubSearchResult{
		{Name: "alpha", Description: "a skill", Source: "fixture", InstallID: "fixture/alpha", Score: 0.90},
		{Name: "mu", Description: "m skill", Source: "fixture", InstallID: "fixture/mu", Score: 0.60},
		{Name: "zeta", Description: "z skill", Source: "fixture", InstallID: "fixture/zeta", Score: 0.30},
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("Snapshot returned %v, want sorted %v", first, want)
	}

	second, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("second Snapshot returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Snapshot is not deterministic between calls: first=%v second=%v", first, second)
	}

	first[0].Name = "mutated"
	again, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("third Snapshot returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(again, want) {
		t.Fatalf("Snapshot leaked internal state to caller mutation: got=%v want=%v", again, want)
	}

	input[0].Name = "constructor-mutated"
	afterCtor, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("fourth Snapshot returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(afterCtor, want) {
		t.Fatalf("Snapshot reflects post-constructor input mutation: got=%v want=%v", afterCtor, want)
	}
}

func TestInMemoryRegistryProviderUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		injected error
	}{
		{name: "registry_unavailable", injected: ErrRegistryUnavailable},
		{name: "registry_rate_limited", injected: ErrRegistryRateLimited},
		{name: "wrapped", injected: errors.Join(ErrRegistryUnavailable, errors.New("github 503"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := NewInMemoryRegistryProvider([]HubSearchResult{
				{Name: "should-not-appear", Source: "fixture", InstallID: "fixture/x"},
			}, tc.injected)

			results, err := provider.Snapshot(context.Background())
			if err == nil {
				t.Fatalf("Snapshot returned nil error, want %v", tc.injected)
			}
			if !errors.Is(err, tc.injected) {
				t.Fatalf("Snapshot err = %v, want errors.Is(%v) to match", err, tc.injected)
			}
			if results != nil {
				t.Fatalf("Snapshot results = %v, want nil when error is set", results)
			}
		})
	}

	if ErrRegistryUnavailable.Error() != "registry_unavailable" {
		t.Errorf("ErrRegistryUnavailable text = %q, want %q", ErrRegistryUnavailable.Error(), "registry_unavailable")
	}
	if ErrRegistryRateLimited.Error() != "registry_rate_limited" {
		t.Errorf("ErrRegistryRateLimited text = %q, want %q", ErrRegistryRateLimited.Error(), "registry_rate_limited")
	}

	var _ HubRegistryProvider = (*InMemoryRegistryProvider)(nil)
}
