package ansitext

import "testing"

func TestTrimToWidth(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  string
	}{
		{name: "fits", text: "hello", width: 8, want: "hello"},
		{name: "zero", text: "hello", width: 0, want: ""},
		{name: "ellipsis", text: "hello", width: 4, want: "hel…"},
		{name: "too narrow", text: "hello", width: 1, want: "."},
		{name: "wide rune", text: "猫猫猫", width: 5, want: "猫猫…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimToWidth(tt.text, tt.width); got != tt.want {
				t.Fatalf("TrimToWidth(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
		})
	}
}
