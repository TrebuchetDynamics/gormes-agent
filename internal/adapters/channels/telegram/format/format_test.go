package format

import "testing"

func TestTelegramMarkdownFallbackStripsEscapesAndMarkers(t *testing.T) {
	got := StripMarkdownV2(`*bold text* and _italic text_ plus ~struck\!~ and ||hidden\.value||`)
	want := "bold text and italic text plus struck! and hidden.value"
	if got != want {
		t.Fatalf("StripMarkdownV2 = %q, want %q", got, want)
	}
}

func TestTelegramMarkdownFallbackPreservesSnakeCase(t *testing.T) {
	got := StripMarkdownV2(`my_variable_name and some_func_call\(x\)`)
	want := "my_variable_name and some_func_call(x)"
	if got != want {
		t.Fatalf("StripMarkdownV2 = %q, want %q", got, want)
	}
}
