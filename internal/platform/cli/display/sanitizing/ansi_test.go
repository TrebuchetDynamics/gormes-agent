package sanitizing

import "testing"

func TestStripANSIUsesSharedECMA48Redaction(t *testing.T) {
	input := "plain\x1b[31mred\x1b[0m \x1b]0;title\x07done \u009b32mgreen\x1b[0m"
	got := StripANSI(input)
	if got != "plainred done green" {
		t.Fatalf("StripANSI() = %q", got)
	}
}
