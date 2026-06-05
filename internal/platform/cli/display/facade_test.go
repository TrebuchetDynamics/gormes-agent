package display

import "testing"

func TestFacadePreservesPublicDisplayContracts(t *testing.T) {
	if got, want := FormatContextLength(128000), "128K"; got != want {
		t.Fatalf("FormatContextLength facade = %q, want %q", got, want)
	}
	if got, want := FormatPrompt("Provider", "openrouter"), "  Provider [openrouter]: "; got != want {
		t.Fatalf("FormatPrompt facade = %q, want %q", got, want)
	}
	if got, want := RenderDumpSummary(DumpInput{Version: "0.1.0"}), "version: 0.1.0\nos: unknown\narch: unknown\nprofile: unknown\ntoolsets: (none)\n"; got != want {
		t.Fatalf("RenderDumpSummary facade = %q, want %q", got, want)
	}
	if got := StripANSI("plain\x1b[31mred\x1b[0m"); got != "plainred" {
		t.Fatalf("StripANSI facade = %q", got)
	}
	if got := TipFor(0); got == "" || got != Tips[0] {
		t.Fatalf("TipFor facade = %q, want first non-empty tip", got)
	}
}
