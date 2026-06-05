package redaction

import "testing"

func TestRedactPathTailBoundsOperatorPath(t *testing.T) {
	cases := map[string]string{
		"/home/alice/.gormes/profiles/work": ".../work",
		"  /tmp/profile-root/  ":            ".../profile-root",
		"":                                  "...",
		"/":                                 "...",
	}
	for input, want := range cases {
		if got := RedactPathTail(input); got != want {
			t.Fatalf("RedactPathTail(%q) = %q, want %q", input, got, want)
		}
	}
}
