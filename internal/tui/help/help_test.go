package help

import (
	"strings"
	"testing"
)

func TestNativeStatusListsLocalCommandsAndSafety(t *testing.T) {
	got := NativeStatus()
	for _, want := range []string{"/help", "/model", "/browser", "never sent as prompts"} {
		if !strings.Contains(got, want) {
			t.Fatalf("NativeStatus() missing %q: %q", want, got)
		}
	}
}
