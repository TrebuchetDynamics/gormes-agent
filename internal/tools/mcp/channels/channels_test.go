package channels

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeDirectory struct {
	platforms map[string][]Entry
	called    bool
}

func (f *fakeDirectory) Platforms() (map[string][]Entry, error) {
	f.called = true
	return f.platforms, nil
}

func TestListReturnsPlatformsInStableOrder(t *testing.T) {
	result, err := List(context.Background(), &fakeDirectory{platforms: map[string][]Entry{
		"telegram": {{ID: "tg-1", Name: "Telegram"}},
		"discord":  {{ID: "dc-1", Name: "Discord"}},
		"gateway":  {{ID: "gw-1", Name: "Gateway"}},
	}}, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	payload := result.(map[string]interface{})
	got := platformOrder(payload["channels"].([]Output))
	want := []string{"discord", "gateway", "telegram"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("platform order = %v, want stable lexical order %v", got, want)
	}
}

func TestListPlatformFilterIgnoresSurroundingWhitespaceAndCase(t *testing.T) {
	result, err := List(context.Background(), &fakeDirectory{platforms: map[string][]Entry{
		" Discord ": {{ID: "dc-1", Name: "Discord"}},
		"telegram":  {{ID: "tg-1", Name: "Telegram"}},
	}}, map[string]interface{}{"platform": "discord"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	payload := result.(map[string]interface{})
	channels := payload["channels"].([]Output)
	if len(channels) != 1 {
		t.Fatalf("channels len = %d, want 1: %#v", len(channels), channels)
	}
	if channels[0].ID != "dc-1" {
		t.Fatalf("selected channel ID = %q, want dc-1", channels[0].ID)
	}
	if channels[0].Platform != "discord" {
		t.Fatalf("selected channel platform = %q, want normalized platform discord", channels[0].Platform)
	}
}

func TestListCanceledContextDoesNotReadDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := &fakeDirectory{platforms: map[string][]Entry{
		"telegram": {{ID: "tg-1", Name: "Telegram"}},
	}}

	result, err := List(ctx, dir, nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
	if result != nil {
		t.Fatalf("result = %#v, want nil when context is already canceled", result)
	}
	if dir.called {
		t.Fatalf("directory was read despite canceled context")
	}
}

func TestListWhitespaceOnlyPlatformFilterIsTreatedAsAbsent(t *testing.T) {
	result, err := List(context.Background(), &fakeDirectory{platforms: map[string][]Entry{
		"discord":  {{ID: "dc-1", Name: "Discord"}},
		"telegram": {{ID: "tg-1", Name: "Telegram"}},
	}}, map[string]interface{}{"platform": " \t "})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	payload := result.(map[string]interface{})
	got := platformOrder(payload["channels"].([]Output))
	want := []string{"discord", "telegram"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("platform order = %v, want whitespace-only filter to be absent and return %v", got, want)
	}
}

func TestPlatformSelectionMatchesOnlyNormalizedFilterWhenPresent(t *testing.T) {
	selection := newPlatformSelection(map[string]interface{}{"platform": " Discord "})
	if !selection.HasFilter {
		t.Fatalf("selection.HasFilter = false, want true")
	}
	if !selection.matches(newPlatformKey("discord")) {
		t.Fatalf("selection did not match normalized platform")
	}
	if selection.matches(newPlatformKey("telegram")) {
		t.Fatalf("selection matched unrelated platform")
	}
}

func TestPlatformCandidatesKeepRawPlatformAsTieBreaker(t *testing.T) {
	candidates := platformCandidates(map[string][]Entry{
		" discord ": {{ID: "spaced"}},
		"Discord":   {{ID: "title"}},
		"discord":   {{ID: "lower"}},
	}, platformSelection{})

	gotRaw := make([]string, 0, len(candidates))
	gotIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Normalized != "discord" {
			t.Fatalf("candidate %#v normalized platform = %q, want discord", candidate, candidate.Normalized)
		}
		gotRaw = append(gotRaw, candidate.Raw)
		gotIDs = append(gotIDs, candidate.Entries[0].ID)
	}
	wantRaw := []string{" discord ", "Discord", "discord"}
	wantIDs := []string{"spaced", "title", "lower"}
	if !reflect.DeepEqual(gotRaw, wantRaw) || !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("candidate provenance/raw order = %v ids %v, want %v ids %v", gotRaw, gotIDs, wantRaw, wantIDs)
	}
}

func platformOrder(channels []Output) []string {
	platforms := make([]string, 0, len(channels))
	for _, channel := range channels {
		platforms = append(platforms, channel.Platform)
	}
	return platforms
}
