package slash

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator/style"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		current    style.Style
		wantStyle  style.Style
		wantStatus string
		wantApply  bool
	}{
		{name: "bare reports current", input: "/indicator", current: style.Unicode, wantStyle: style.Unicode, wantStatus: "indicator: unicode"},
		{name: "sets unicode", input: "/indicator unicode", current: style.Emoji, wantStyle: style.Unicode, wantStatus: "indicator → unicode", wantApply: true},
		{name: "invalid reports usage", input: "/indicator sparkle", current: style.Emoji, wantStyle: style.Emoji, wantStatus: Usage},
		{name: "too many args reports usage", input: "/indicator unicode now", current: style.Emoji, wantStyle: style.Emoji, wantStatus: Usage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input, tt.current)
			if got.Style != tt.wantStyle || got.Status != tt.wantStatus || got.Apply != tt.wantApply {
				t.Fatalf("Parse(%q, %q) = %#v, want style=%q status=%q apply=%v", tt.input, tt.current, got, tt.wantStyle, tt.wantStatus, tt.wantApply)
			}
		})
	}
}
