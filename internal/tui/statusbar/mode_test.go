package statusbar

import "testing"

func TestSlashNextParsesStatusBarMode(t *testing.T) {
	tests := []struct {
		name    string
		initial Mode
		input   string
		want    Mode
		wantOK  bool
	}{
		{name: "bare toggles off from top", initial: ModeTop, input: "/statusbar", want: ModeOff, wantOK: true},
		{name: "toggle restores top from off", initial: ModeOff, input: "/statusbar toggle", want: ModeTop, wantOK: true},
		{name: "on maps to top", initial: ModeOff, input: "/statusbar on", want: ModeTop, wantOK: true},
		{name: "bottom accepted", initial: ModeTop, input: "/statusbar bottom", want: ModeBottom, wantOK: true},
		{name: "invalid keeps normalized current", initial: ModeBottom, input: "/statusbar sideways", want: ModeBottom, wantOK: false},
		{name: "too many fields rejected", initial: ModeTop, input: "/statusbar top extra", want: ModeTop, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SlashNext(tt.input, tt.initial)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("SlashNext(%q, %q) = (%q, %v), want (%q, %v)", tt.input, tt.initial, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNormalizeModeDefaultsUnknownToTop(t *testing.T) {
	if got := NormalizeMode(Mode("sideways")); got != ModeTop {
		t.Fatalf("NormalizeMode(sideways) = %q, want %q", got, ModeTop)
	}
}
