// internal/security/tirith_test.go
package security

import (
	"os"
	"path/filepath"
	"testing"
)

// Acceptance: TestTirithLoadsFindings proves findings are parsed and
// classified by severity.
func TestTirithLoadsFindings(t *testing.T) {
	src := writeTempFindings(t, `{
		"findings": [
			{"rule_id": "R001", "severity": "critical", "message": "Hardcoded API key", "file": "config.yaml"},
			{"rule_id": "R002", "severity": "high", "message": "SQL injection risk", "file": "query.go"},
			{"rule_id": "R003", "severity": "medium", "message": "Weak TLS version", "file": "server.go"},
			{"rule_id": "R004", "severity": "low", "message": "Unused variable", "file": "main.go"},
			{"rule_id": "R005", "severity": "info", "message": "Style suggestion", "file": "helper.go"}
		]
	}`)

	client, err := NewTirithClient(src)
	if err != nil {
		t.Fatalf("NewTirithClient: %v", err)
	}

	findings := client.Findings()
	if len(findings) != 5 {
		t.Fatalf("got %d findings, want 5", len(findings))
	}

	// Verify severity ordering: critical → high → medium → low → info.
	expected := []TirithSeverity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
	for i, f := range findings {
		if f.Severity != expected[i] {
			t.Errorf("finding[%d].Severity = %q, want %q", i, f.Severity, expected[i])
		}
	}

	// Verify individual fields.
	if findings[0].RuleID != "R001" {
		t.Errorf("findings[0].RuleID = %q, want %q", findings[0].RuleID, "R001")
	}
	if findings[0].Message != "Hardcoded API key" {
		t.Errorf("findings[0].Message = %q, want %q", findings[0].Message, "Hardcoded API key")
	}
	if findings[0].File != "config.yaml" {
		t.Errorf("findings[0].File = %q, want %q", findings[0].File, "config.yaml")
	}

	// Decision: any finding with severity >= high → deny.
	ev := client.Decision()
	if ev.Allow {
		t.Errorf("Decision.Allow = true, want false (critical finding present)")
	}
	if ev.Reason == "" {
		t.Errorf("Decision.Reason is empty")
	}
}

// Acceptance: TestTirithEmptySourceReturnsSafeEvidence proves empty/missing
// source returns safe (allow) with typed evidence.
func TestTirithEmptySourceReturnsSafeEvidence(t *testing.T) {
	src := writeTempFindings(t, `{"findings": []}`)

	client, err := NewTirithClient(src)
	if err != nil {
		t.Fatalf("NewTirithClient: %v", err)
	}

	findings := client.Findings()
	if len(findings) != 0 {
		t.Fatalf("got %d findings, want 0", len(findings))
	}

	ev := client.Decision()
	if !ev.Allow {
		t.Errorf("Decision.Allow = false, want true (no findings)")
	}
	if ev.EvidenceType != "tirith_no_findings" && ev.EvidenceType != "" {
		t.Errorf("Decision.EvidenceType = %q, want tirith_no_findings or empty", ev.EvidenceType)
	}

	// Missing source file should also return safe.
	client2, err := NewTirithClient("/nonexistent/tirith_findings.json")
	if err != nil {
		t.Fatalf("NewTirithClient(missing): %v", err)
	}
	ev2 := client2.Decision()
	if !ev2.Allow {
		t.Errorf("Decision (missing source).Allow = false, want true")
	}
}

// Acceptance: TestTirithCorruptSourceDegrades proves corrupt findings return
// deny with tirith_corrupt_evidence.
func TestTirithCorruptSourceDegrades(t *testing.T) {
	src := writeTempFindings(t, `{this is not valid json`)

	client, err := NewTirithClient(src)
	if err != nil {
		t.Fatalf("NewTirithClient: %v", err)
	}

	ev := client.Decision()
	if ev.Allow {
		t.Errorf("Decision.Allow = true, want false (corrupt source)")
	}
	if ev.EvidenceType != "tirith_corrupt_evidence" {
		t.Errorf("Decision.EvidenceType = %q, want tirith_corrupt_evidence", ev.EvidenceType)
	}
}

// TestTirithDegradedMode proves a file-not-found source returns
// tirith_unavailable evidence and allows (fallback to allowlist).
func TestTirithDegradedMode(t *testing.T) {
	src := filepath.Join(t.TempDir(), "no_such_file.json")

	client, err := NewTirithClient(src)
	if err != nil {
		t.Fatalf("NewTirithClient: %v", err)
	}

	ev := client.Decision()
	if !ev.Allow {
		t.Errorf("Decision.Allow = false, want true (degraded mode: missing source → allow)")
	}
	if ev.EvidenceType != "tirith_unavailable" {
		t.Errorf("Decision.EvidenceType = %q, want tirith_unavailable", ev.EvidenceType)
	}
}

// TestTirithSeverityDeny proves each severity level maps to the correct
// allow/deny threshold.
func TestTirithSeverityDeny(t *testing.T) {
	tests := []struct {
		severity TirithSeverity
		allow    bool
	}{
		{SeverityCritical, false},
		{SeverityHigh, false},
		{SeverityMedium, true},
		{SeverityLow, true},
		{SeverityInfo, true},
	}

	for _, tt := range tests {
		src := writeTempFindings(t, `{
			"findings": [{"rule_id":"T001","severity":"`+string(tt.severity)+`","message":"test","file":"x.go"}]
		}`)
		client, err := NewTirithClient(src)
		if err != nil {
			t.Fatalf("severity=%q NewTirithClient: %v", tt.severity, err)
		}
		ev := client.Decision()
		if ev.Allow != tt.allow {
			t.Errorf("severity=%q Decision.Allow = %v, want %v", tt.severity, ev.Allow, tt.allow)
		}
	}
}

// writeTempFindings writes findings JSON to a temp file and returns its path.
func writeTempFindings(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tirith_findings.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTempFindings: %v", err)
	}
	return path
}
