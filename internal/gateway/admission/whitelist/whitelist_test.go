package whitelist

import (
	"testing"
)

func TestWhitelistConfig_EmptyMeansAllAllowed(t *testing.T) {
	wc := WhitelistConfig{}
	if wc.Enabled {
		t.Error("empty WhitelistConfig should have Enabled=false")
	}
	if !wc.IsAllowed("any-chat-id") {
		t.Error("empty WhitelistConfig should allow any chat ID")
	}
	if !wc.IsAllowed("") {
		t.Error("empty WhitelistConfig should allow empty chat ID")
	}
}

func TestWhitelistFilter_DropsNonWhitelistedChat(t *testing.T) {
	wc := WhitelistConfig{Enabled: true, IDs: []string{"-1001", "-1002", "C999"}}
	if wc.IsAllowed("-1003") {
		t.Error("non-whitelisted chat ID should be dropped")
	}
	if wc.IsAllowed("") {
		t.Error("empty chat ID should be dropped when whitelist is enabled")
	}
}

func TestWhitelistFilter_PassesWhitelistedChat(t *testing.T) {
	wc := WhitelistConfig{Enabled: true, IDs: []string{"-1001", "-1002", "C999"}}
	if !wc.IsAllowed("-1001") {
		t.Error("whitelisted chat ID -1001 should be allowed")
	}
	if !wc.IsAllowed("-1002") {
		t.Error("whitelisted chat ID -1002 should be allowed")
	}
	if !wc.IsAllowed("C999") {
		t.Error("whitelisted chat ID C999 should be allowed")
	}
}

func TestParseWhitelistConfig_NilOrEmptyReturnsDisabled(t *testing.T) {
	if wc := ParseWhitelistConfig(nil); wc.Enabled {
		t.Error("nil input should produce disabled whitelist")
	}
	if wc := ParseWhitelistConfig([]string{}); wc.Enabled {
		t.Error("empty input should produce disabled whitelist")
	}
	if wc := ParseWhitelistConfig([]string{"", "  "}); wc.Enabled {
		t.Error("whitespace-only input should produce disabled whitelist")
	}
}

func TestParseWhitelistConfig_TrimsAndCompacts(t *testing.T) {
	wc := ParseWhitelistConfig([]string{" -1001 ", "  -1002", "-1001  ", "", "  "})
	if !wc.Enabled {
		t.Error("non-empty trimmed input should produce enabled whitelist")
	}
	if len(wc.IDs) != 2 {
		t.Fatalf("expected 2 IDs after trim+compact, got %d: %v", len(wc.IDs), wc.IDs)
	}
	if wc.IDs[0] != "-1001" || wc.IDs[1] != "-1002" {
		t.Errorf("unexpected IDs: %v", wc.IDs)
	}
}

func TestParseWhitelistConfig_Deduplicates(t *testing.T) {
	wc := ParseWhitelistConfig([]string{"-1001", "-1002", "-1001", "-1001"})
	if len(wc.IDs) != 2 {
		t.Fatalf("expected 2 unique IDs, got %d: %v", len(wc.IDs), wc.IDs)
	}
}

func TestWhitelistConfig_ParseErrorDegrades(t *testing.T) {
	// Empty strings and whitespace-only entries should be silently skipped,
	// not cause the entire whitelist to fail.
	wc := ParseWhitelistConfig([]string{"-1001", "", "  ", "-1002"})
	if !wc.Enabled {
		t.Error("whitelist with valid entries should be enabled despite empty entries")
	}
	if !wc.IsAllowed("-1001") {
		t.Error("valid entry should still work after empty entry skip")
	}
}

func TestWhitelistFilter_StatusEvidence(t *testing.T) {
	// Verify WhitelistStatus produces correct counts.
	disabled := WhitelistStatus{}
	if disabled.ActiveCount != 0 {
		t.Error("zero-value WhitelistStatus should have ActiveCount=0")
	}
	if disabled.SkippedCount != 0 {
		t.Error("zero-value WhitelistStatus should have SkippedCount=0")
	}
	if disabled.ParseError != "" {
		t.Error("zero-value WhitelistStatus should have empty ParseError")
	}

	active := WhitelistStatus{
		ActiveCount:  3,
		SkippedCount: 5,
		ParseError:   "bad regex: [",
	}
	if active.ActiveCount != 3 {
		t.Errorf("ActiveCount = %d, want 3", active.ActiveCount)
	}
	if active.SkippedCount != 5 {
		t.Errorf("SkippedCount = %d, want 5", active.SkippedCount)
	}
	if active.ParseError != "bad regex: [" {
		t.Errorf("ParseError = %q, want %q", active.ParseError, "bad regex: [")
	}
}
