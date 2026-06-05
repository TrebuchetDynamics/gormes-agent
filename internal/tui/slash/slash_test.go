package slash

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := map[string]string{
		"save":  "save",
		"/save": "save",
		"/SAVE": "save",
		"":      "",
	}
	for input, want := range tests {
		if got := NormalizeName(input); got != want {
			t.Fatalf("NormalizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestInvocationArgs(t *testing.T) {
	tests := map[string]string{
		"/queue follow up":       "follow up",
		"  /queue   follow up  ": "follow up",
		"/queue\tfollow up":      "follow up",
		"/queue":                 "",
		"":                       "",
	}
	for input, want := range tests {
		if got := InvocationArgs(input); got != want {
			t.Fatalf("InvocationArgs(%q) = %q, want %q", input, got, want)
		}
	}
}
