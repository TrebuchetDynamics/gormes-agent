package textvalue

import "testing"

func TestIsNonBlank(t *testing.T) {
	if !IsNonBlank(" value ") {
		t.Fatal("IsNonBlank(value) = false, want true")
	}
	if IsNonBlank(" \t\n ") {
		t.Fatal("IsNonBlank(blank) = true, want false")
	}
}

func TestLowerTrim(t *testing.T) {
	if got := LowerTrim("  OpenAI-Codex\t"); got != "openai-codex" {
		t.Fatalf("LowerTrim() = %q, want openai-codex", got)
	}
}

func TestFirstNonBlankPreservesSourceText(t *testing.T) {
	got := FirstNonBlank("", " \t ", "  provider  ", "fallback")
	if got != "  provider  " {
		t.Fatalf("FirstNonBlank() = %q, want source text", got)
	}
}

func TestFirstNonEmptyTrimmed(t *testing.T) {
	got := FirstNonEmptyTrimmed("", " \t ", "  provider  ", "fallback")
	if got != "provider" {
		t.Fatalf("FirstNonEmptyTrimmed() = %q, want provider", got)
	}
}

func TestFirstNonEmptyTrimmedAllBlank(t *testing.T) {
	if got := FirstNonEmptyTrimmed("", "  "); got != "" {
		t.Fatalf("FirstNonEmptyTrimmed blank = %q, want empty", got)
	}
}

func TestCompactWhitespace(t *testing.T) {
	if got := CompactWhitespace("  gormes\tprovider\nmodel   set  "); got != "gormes provider model set" {
		t.Fatalf("CompactWhitespace() = %q", got)
	}
}

func TestTrimmedLines(t *testing.T) {
	got := TrimmedLines(" one \n\t \n two\r ")
	want := []string{"one", "", "two"}
	if len(got) != len(want) {
		t.Fatalf("TrimmedLines len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TrimmedLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFirstNonBlankLine(t *testing.T) {
	if got := FirstNonBlankLine(" \n\t second \nthird"); got != "second" {
		t.Fatalf("FirstNonBlankLine() = %q, want second", got)
	}
	if got := FirstNonBlankLine(" \n\t "); got != "" {
		t.Fatalf("FirstNonBlankLine(blank) = %q, want empty", got)
	}
}

func TestCompactKeyToken(t *testing.T) {
	for _, in := range []string{" API_Key ", "api-key", "api.key", "api key"} {
		if got := CompactKeyToken(in); got != "apikey" {
			t.Fatalf("CompactKeyToken(%q) = %q, want apikey", in, got)
		}
	}
}
