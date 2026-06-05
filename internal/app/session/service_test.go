package session

import "testing"

func TestCoalesceSessionNameArgsMultiWordNames(t *testing.T) {
	got := CoalesceSessionNameArgs([]string{"-c", "my", "project", "sessions", "list"})
	want := []string{"-c", "my project", "sessions", "list"}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestTUISaveExportStemSanitizesEmptyAndPathSeparators(t *testing.T) {
	if got := tuiSaveExportStem("a/b\\c"); got != "a_b_c" {
		t.Fatalf("stem = %q, want a_b_c", got)
	}
	if got := tuiSaveExportStem("   "); got != "session" {
		t.Fatalf("empty stem = %q, want session", got)
	}
}
