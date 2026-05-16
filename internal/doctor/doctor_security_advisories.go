package doctor

import (
	"fmt"
	"strings"
)

// DoctorAdvisoryView is the small caller-facing seam the ◆ Security Advisories
// check needs: one detected advisory hit plus whether the user has dismissed
// it. cmd/gormes builds these from the real internal/security subsystem
// (DetectCompromised + the ~/.gormes ack store); tests inject hermetic fakes.
// This keeps internal/doctor decoupled from internal/security (same pattern as
// DoctorProfileInventory).
type DoctorAdvisoryView struct {
	ID          string
	Title       string
	Package     string
	Version     string
	Remediation []string // pre-rendered remediation lines (carried verbatim)
	Acked       bool
}

// DoctorSecurityAdvisoryInventory is the injected input for the section.
// Hits empty → clean PASS. ScanError degrades to one typed WARN item rather
// than fabricating a (Python-only) package scan Gormes cannot faithfully run.
type DoctorSecurityAdvisoryInventory struct {
	Hits      []DoctorAdvisoryView
	ScanError string
}

// CheckSecurityAdvisories renders the Gormes-owned ◆ Security Advisories
// section content (parity intent hermes_cli/doctor.py@55c9f3206:350, which
// runs FIRST). Owned divergence: Gormes is a single Go binary with no Python
// venv, so the default (no-hit) state is the faithful behavior — a clean
// non-actionable PASS "No active security advisories" (Hermes' check_ok /
// "silent otherwise"), never a SKIP/WARN.
//
//   - fresh (unacked) hit  → StatusFail item: title + (pkg==version) +
//     remediation lines; the section FAILs so it is funneled into the
//     computed Found-N action list (like Hermes' check_fail + manual_issues).
//   - acked-but-still-present hit → StatusWarn informational item, but the
//     section stays a non-actionable PASS so it does NOT inflate Found-N
//     (mirrors Hermes' acked-but-installed check_warn that is NOT funneled).
//   - no hits → clean PASS "No active security advisories".
func CheckSecurityAdvisories(inv DoctorSecurityAdvisoryInventory) CheckResult {
	if strings.TrimSpace(inv.ScanError) != "" {
		return CheckResult{
			Name:    "Security Advisories",
			Status:  StatusWarn,
			Summary: "advisory scan degraded",
			Items: []ItemInfo{
				{Name: "scan", Status: StatusWarn, Note: strings.TrimSpace(inv.ScanError)},
			},
		}
	}

	var fresh, ackedPresent []DoctorAdvisoryView
	for _, h := range inv.Hits {
		if h.Acked {
			ackedPresent = append(ackedPresent, h)
		} else {
			fresh = append(fresh, h)
		}
	}

	if len(fresh) == 0 && len(ackedPresent) == 0 {
		return CheckResult{
			Name:    "Security Advisories",
			Status:  StatusPass,
			Summary: "No active security advisories",
			Items: []ItemInfo{
				{Name: "advisories", Status: StatusPass, Note: "No active security advisories"},
			},
		}
	}

	items := make([]ItemInfo, 0, len(inv.Hits))
	for _, h := range fresh {
		note := fmt.Sprintf("%s (%s==%s)", strings.TrimSpace(h.Title), h.Package, h.Version)
		for _, line := range h.Remediation {
			note += "\n        " + line
		}
		items = append(items, ItemInfo{Name: advisoryItemName(h.ID), Status: StatusFail, Note: note})
	}
	for _, h := range ackedPresent {
		items = append(items, ItemInfo{
			Name:   advisoryItemName(h.ID),
			Status: StatusWarn,
			Note: fmt.Sprintf("%s==%s still present (advisory %s acknowledged)",
				h.Package, h.Version, h.ID),
		})
	}

	if len(fresh) > 0 {
		// Fresh advisories are actionable: the section FAILs so the computed
		// Found-N action list surfaces them. Acked-but-present items still
		// render as informational WARN rows under the failed section.
		return CheckResult{
			Name:    "Security Advisories",
			Status:  StatusFail,
			Summary: fmt.Sprintf("%d active security %s", len(fresh), advisoryWord(len(fresh))),
			Items:   items,
		}
	}

	// Only acked-but-present: non-actionable. Keep the section a clean PASS so
	// it does NOT inflate Found-N, but surface the WARN informational rows.
	return CheckResult{
		Name:   "Security Advisories",
		Status: StatusPass,
		Summary: fmt.Sprintf("No active security advisories (%d acknowledged %s still present)",
			len(ackedPresent), advisoryWord(len(ackedPresent))),
		Items: items,
	}
}

func advisoryItemName(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "advisory"
	}
	return id
}

func advisoryWord(n int) string {
	if n == 1 {
		return "advisory"
	}
	return "advisories"
}
