package tui

import "testing"

func TestTUIANSICompatibilityWrappers(t *testing.T) {
	input := "A\x1b[31mB\x1b[2JC"
	if got := StripANSIForTUI(input); got != "ABC" {
		t.Fatalf("StripANSIForTUI() = %q, want %q", got, "ABC")
	}
	if got := SanitizeANSIForRender(input); got != "A\x1b[31mBC" {
		t.Fatalf("SanitizeANSIForRender() = %q", got)
	}
	if !HasANSI(input) {
		t.Fatal("HasANSI() = false, want true")
	}
}
