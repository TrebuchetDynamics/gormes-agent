package compact

import "testing"

func TestNextParsesCompactSlashState(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current bool
		want    bool
		wantOK  bool
	}{
		{name: "bare toggles on", input: "/compact", want: true, wantOK: true},
		{name: "bare toggles off", input: "/compact", current: true, want: false, wantOK: true},
		{name: "on", input: "/compact on", want: true, wantOK: true},
		{name: "off", input: "/compact off", current: true, want: false, wantOK: true},
		{name: "toggle", input: "/compact toggle", current: true, want: false, wantOK: true},
		{name: "invalid arg", input: "/compact maybe", current: true, want: true, wantOK: false},
		{name: "too many args", input: "/compact on now", current: true, want: true, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Next(tt.input, tt.current)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("Next(%q, %v) = (%v, %v), want (%v, %v)", tt.input, tt.current, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
