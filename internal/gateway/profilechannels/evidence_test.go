package profilechannels

import "testing"

func TestHasEvidenceCode(t *testing.T) {
	items := []ProfileChannelReadinessEvidence{{Code: "missing"}, {Code: "ready"}}
	if !HasEvidenceCode(items, "ready") {
		t.Fatal("HasEvidenceCode missing ready evidence")
	}
	if HasEvidenceCode(items, "other") {
		t.Fatal("HasEvidenceCode returned true for absent evidence")
	}
}
