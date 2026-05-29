package ansitext

import "testing"

func TestStripForTUIStripsControlsAndDanglingCSI(t *testing.T) {
	input := "A\x1b[31mB\x1b[39m\x1b[2J\x1b]0;title\x07C\x1b[?25lD"
	if got := StripForTUI(input); got != "ABCD" {
		t.Fatalf("StripForTUI() = %q, want %q", got, "ABCD")
	}

	dangling := "A\x1b[31mB\x1b[12;\x1b[CD\rE"
	if got := StripForTUI(dangling); got != "ABDE" {
		t.Fatalf("StripForTUI(dangling) = %q, want %q", got, "ABDE")
	}

	for _, input := range []string{"A\x1b(0B", "A\x1b%GB", "A\x1b)0B"} {
		if got := StripForTUI(input); got != "AB" {
			t.Fatalf("StripForTUI(%q) = %q, want AB", input, got)
		}
	}
}

func TestSanitizeForRenderKeepsOnlySGR(t *testing.T) {
	input := "A\x1b[31mB\x1b[39m\x1b[2J\x1b]0;title\x07C\x1b[?25lD"
	want := "A\x1b[31mB\x1b[39mCD"
	if got := SanitizeForRender(input); got != want {
		t.Fatalf("SanitizeForRender() = %q, want %q", got, want)
	}

	dangling := "A\x1b[31mB\x1b[12;\x1b[CD\rE"
	want = "A\x1b[31mBDE"
	if got := SanitizeForRender(dangling); got != want {
		t.Fatalf("SanitizeForRender(dangling) = %q, want %q", got, want)
	}
}

func TestHasANSIDetectsNonCSIPrefixes(t *testing.T) {
	for _, input := range []string{"\x1b[31mred", "\x1b]0;title\x07", "\x1b(0"} {
		if !HasANSI(input) {
			t.Fatalf("HasANSI(%q) = false, want true", input)
		}
	}
	if HasANSI("plain text") {
		t.Fatal("HasANSI(plain text) = true, want false")
	}
}

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
