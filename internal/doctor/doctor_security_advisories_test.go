package doctor

import (
	"strings"
	"testing"
)

// Owned divergence (parity intent hermes_cli/doctor.py@55c9f3206:350): a
// pure-Go Gormes runtime has no Python venv to scan, so no hits is the
// faithful default. It must be a clean PASS "No active security advisories"
// (Hermes' check_ok / "silent otherwise"), NOT a SKIP/WARN, and must NOT
// inflate the computed Found-N count.
func TestCheckSecurityAdvisoriesNoHitsIsCleanNonActionablePass(t *testing.T) {
	r := CheckSecurityAdvisories(DoctorSecurityAdvisoryInventory{})

	if r.Name != "Security Advisories" {
		t.Fatalf("CheckResult.Name = %q, want %q", r.Name, "Security Advisories")
	}
	if r.Status != StatusPass {
		t.Fatalf("no hits must be a clean PASS, got %v summary=%q", r.Status, r.Summary)
	}
	if !strings.Contains(strings.ToLower(r.Summary+itemsJoin(r)), "no active security advisories") {
		t.Fatalf("no hits must report 'No active security advisories', got %q / %s", r.Summary, itemsJoin(r))
	}
	if got := len(CollectDoctorIssues([]CheckResult{r})); got != 0 {
		t.Fatalf("no-hits state must NOT inflate Found-N: want 0 issues, got %d", got)
	}
}

// A fresh (unacked) hit is an actionable FAIL: the item carries the advisory
// title + (pkg==installed_version) + the remediation lines, and it IS counted
// in CollectDoctorIssues (funneled, like Hermes' check_fail + manual_issues).
func TestCheckSecurityAdvisoriesFreshHitFailsAndIsActionable(t *testing.T) {
	r := CheckSecurityAdvisories(DoctorSecurityAdvisoryInventory{
		Hits: []DoctorAdvisoryView{{
			ID:      "shai-hulud-2026-05",
			Title:   "Mini Shai-Hulud worm — mistralai 2.4.6 compromised on PyPI",
			Package: "mistralai",
			Version: "2.4.6",
			Remediation: []string{
				"=== Mini Shai-Hulud worm — mistralai 2.4.6 compromised on PyPI ===",
				"Remediation:",
				"  1. Run: pip uninstall -y mistralai",
				"  2. Rotate API keys in your credential store.",
			},
			Acked: false,
		}},
	})

	if r.Status != StatusFail {
		t.Fatalf("a fresh unacked hit must FAIL the section, got %v", r.Status)
	}
	if got := len(CollectDoctorIssues([]CheckResult{r})); got != 1 {
		t.Fatalf("a fresh advisory is actionable: want 1 issue, got %d", got)
	}
	joined := itemsJoin(r)
	if !strings.Contains(joined, "shai-hulud-2026-05") {
		t.Fatalf("fresh hit must name the advisory id:\n%s", joined)
	}
	if !strings.Contains(joined, "mistralai==2.4.6") {
		t.Fatalf("fresh hit must show (pkg==installed_version):\n%s", joined)
	}
	if !strings.Contains(joined, "pip uninstall -y mistralai") {
		t.Fatalf("fresh hit must carry the remediation lines verbatim:\n%s", joined)
	}
}

// An acked-but-still-present advisory is informational: a WARN item, but the
// section stays a non-actionable PASS so it does NOT inflate Found-N (mirrors
// Hermes' acked-but-installed check_warn that is NOT funneled into the action
// list).
func TestCheckSecurityAdvisoriesAckedPresentIsInformationalNotActionable(t *testing.T) {
	r := CheckSecurityAdvisories(DoctorSecurityAdvisoryInventory{
		Hits: []DoctorAdvisoryView{{
			ID:      "shai-hulud-2026-05",
			Title:   "Mini Shai-Hulud worm",
			Package: "mistralai",
			Version: "2.4.6",
			Acked:   true,
		}},
	})

	if r.Status != StatusPass {
		t.Fatalf("acked-but-present must keep the section a non-actionable PASS, got %v", r.Status)
	}
	if got := len(CollectDoctorIssues([]CheckResult{r})); got != 0 {
		t.Fatalf("acked advisory must NOT inflate Found-N: want 0 issues, got %d", got)
	}
	var sawWarnItem bool
	for _, it := range r.Items {
		if it.Status == StatusWarn && strings.Contains(it.Note, "mistralai==2.4.6") {
			sawWarnItem = true
		}
	}
	if !sawWarnItem {
		t.Fatalf("acked-but-present must surface a WARN informational item:\n%s", itemsJoin(r))
	}
	if !strings.Contains(strings.ToLower(itemsJoin(r)), "acknowledged") {
		t.Fatalf("acked-but-present item must read as acknowledged:\n%s", itemsJoin(r))
	}
}

// A fresh + acked mix renders both: section FAILs on the fresh one and still
// shows the acked-but-present informational item.
func TestCheckSecurityAdvisoriesFreshAndAckedMixRendersBoth(t *testing.T) {
	r := CheckSecurityAdvisories(DoctorSecurityAdvisoryInventory{
		Hits: []DoctorAdvisoryView{
			{ID: "fresh-1", Title: "Fresh advisory", Package: "pkgfresh", Version: "1.0.0",
				Remediation: []string{"Remediation:", "  1. uninstall pkgfresh"}},
			{ID: "old-acked", Title: "Old advisory", Package: "pkgold", Version: "9.9.9", Acked: true},
		},
	})
	if r.Status != StatusFail {
		t.Fatalf("a fresh hit anywhere must FAIL the section, got %v", r.Status)
	}
	joined := itemsJoin(r)
	if !strings.Contains(joined, "pkgfresh==1.0.0") || !strings.Contains(joined, "pkgold==9.9.9") {
		t.Fatalf("both fresh and acked-present advisories must render:\n%s", joined)
	}
}

// The Security Advisories check must group under the ◆ Security Advisories
// header and render FIRST (parity: hermes doctor runs it first as the most
// urgent section).
func TestSecurityAdvisoriesSectionMappedAndRendersFirst(t *testing.T) {
	if got := sectionForCheck("Security Advisories"); got != SectionSecurityAdvisories {
		t.Fatalf("sectionForCheck(\"Security Advisories\") = %q, want %q", got, SectionSecurityAdvisories)
	}

	out := RenderSectionedReport([]CheckResult{
		{Name: "Goncho config", Status: StatusPass, Summary: "ok"},
		CheckSecurityAdvisories(DoctorSecurityAdvisoryInventory{}),
		{Name: "Profiles", Status: StatusPass, Summary: "default profile only"},
	})
	advIdx := strings.Index(out, "◆ Security Advisories")
	memIdx := strings.Index(out, "◆ Memory Provider")
	if advIdx < 0 {
		t.Fatalf("◆ Security Advisories header missing:\n%s", out)
	}
	if memIdx >= 0 && advIdx > memIdx {
		t.Fatalf("◆ Security Advisories must render before other sections:\n%s", out)
	}
	if !strings.Contains(out, "No active security advisories") {
		t.Fatalf("section body must render the clean-PASS line:\n%s", out)
	}
}

func itemsJoin(r CheckResult) string {
	parts := []string{r.Summary}
	for _, it := range r.Items {
		parts = append(parts, it.Name+" "+it.Note)
	}
	return strings.Join(parts, " | ")
}
