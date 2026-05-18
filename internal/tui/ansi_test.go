package tui

import "testing"

func TestTUIANSISanitizerStripsControlsAndDanglingCSI(t *testing.T) {
	input := "A\x1b[31mB\x1b[39m\x1b[2J\x1b]0;title\x07C\x1b[?25lD"
	if got := StripANSIForTUI(input); got != "ABCD" {
		t.Fatalf("StripANSIForTUI() = %q, want %q", got, "ABCD")
	}

	dangling := "A\x1b[31mB\x1b[12;\x1b[CD\rE"
	if got := StripANSIForTUI(dangling); got != "ABDE" {
		t.Fatalf("StripANSIForTUI(dangling) = %q, want %q", got, "ABDE")
	}

	for _, input := range []string{"A\x1b(0B", "A\x1b%GB", "A\x1b)0B"} {
		if got := StripANSIForTUI(input); got != "AB" {
			t.Fatalf("StripANSIForTUI(%q) = %q, want AB", input, got)
		}
	}
}

func TestTUIANSISanitizerKeepsOnlySGRForRender(t *testing.T) {
	input := "A\x1b[31mB\x1b[39m\x1b[2J\x1b]0;title\x07C\x1b[?25lD"
	want := "A\x1b[31mB\x1b[39mCD"
	if got := SanitizeANSIForRender(input); got != want {
		t.Fatalf("SanitizeANSIForRender() = %q, want %q", got, want)
	}

	dangling := "A\x1b[31mB\x1b[12;\x1b[CD\rE"
	want = "A\x1b[31mBDE"
	if got := SanitizeANSIForRender(dangling); got != want {
		t.Fatalf("SanitizeANSIForRender(dangling) = %q, want %q", got, want)
	}
}

func TestTUIANSIHasANSIDetectsNonCSIPrefixes(t *testing.T) {
	for _, input := range []string{"\x1b[31mred", "\x1b]0;title\x07", "\x1b(0"} {
		if !HasANSI(input) {
			t.Fatalf("HasANSI(%q) = false, want true", input)
		}
	}
	if HasANSI("plain text") {
		t.Fatal("HasANSI(plain text) = true, want false")
	}
}
