package channels

import (
	"context"
	"reflect"
	"testing"
)

type fakeDirectory struct {
	platforms map[string][]Entry
}

func (f fakeDirectory) Platforms() (map[string][]Entry, error) {
	return f.platforms, nil
}

func TestListReturnsPlatformsInStableOrder(t *testing.T) {
	result, err := List(context.Background(), fakeDirectory{platforms: map[string][]Entry{
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
	result, err := List(context.Background(), fakeDirectory{platforms: map[string][]Entry{
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

func platformOrder(channels []Output) []string {
	platforms := make([]string, 0, len(channels))
	for _, channel := range channels {
		platforms = append(platforms, channel.Platform)
	}
	return platforms
}
