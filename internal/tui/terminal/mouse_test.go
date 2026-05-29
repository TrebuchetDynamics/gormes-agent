package terminal

import "testing"

func TestParseMouseTrackingSlash(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current bool
		want    MouseSlashResult
	}{
		{name: "bare toggles", input: "/mouse", current: true, want: MouseSlashResult{Handled: true, Valid: true, Next: false}},
		{name: "scroll alias", input: "/scroll on", want: MouseSlashResult{Handled: true, Valid: true, Next: true}},
		{name: "invalid", input: "/mouse sideways", want: MouseSlashResult{Handled: true, Message: MouseSlashUsage}},
		{name: "other slash", input: "/help", current: true, want: MouseSlashResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseMouseTrackingSlash(tt.input, tt.current); got != tt.want {
				t.Fatalf("ParseMouseTrackingSlash() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
