package indicator

import "testing"

func TestParseSlash(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		current    Style
		wantStyle  Style
		wantStatus string
		wantApply  bool
	}{
		{name: "bare reports current", input: "/indicator", current: StyleUnicode, wantStyle: StyleUnicode, wantStatus: "indicator: unicode"},
		{name: "sets unicode", input: "/indicator unicode", current: StyleEmoji, wantStyle: StyleUnicode, wantStatus: "indicator → unicode", wantApply: true},
		{name: "invalid reports usage", input: "/indicator sparkle", current: StyleEmoji, wantStyle: StyleEmoji, wantStatus: SlashUsage},
		{name: "too many args reports usage", input: "/indicator unicode now", current: StyleEmoji, wantStyle: StyleEmoji, wantStatus: SlashUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSlash(tt.input, tt.current)
			if got.Style != tt.wantStyle || got.Status != tt.wantStatus || got.Apply != tt.wantApply {
				t.Fatalf("ParseSlash(%q, %q) = %#v, want style=%q status=%q apply=%v", tt.input, tt.current, got, tt.wantStyle, tt.wantStatus, tt.wantApply)
			}
		})
	}
}
