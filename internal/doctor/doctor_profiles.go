package doctor

import (
	"fmt"
	"sort"
	"strings"
)

// DoctorProfileInventory is the small caller-facing seam the ◆ Profiles
// check needs. cmd/gormes wires it from the real profileCommandSeams
// (ListKnownProfiles / ReadActiveProfileName / ResolveProfileRoot /
// ReadDistributionManifest); tests inject a hermetic fake. This keeps
// internal/doctor free of cli/cmd profile dependencies.
//
// Owned divergence (parity intent hermes_cli/doctor.py@55c9f3206:1768):
// Gormes profiles are ~/.gormes/profiles/<name> subdirs with an optional
// distribution manifest + an active marker. There is intentionally NO
// per-profile gateway_running, shell-wrapper alias, orphan-alias, or
// per-profile model[:30] — Gormes does not own those, so they are never
// fabricated here.
type DoctorProfileInventory struct {
	// Known are the named (non-default) profiles, e.g. from
	// profileCommandSeams.ListKnownProfiles().
	Known []string
	// Active is the active profile name ("" or "default" → the default).
	Active string
	// RootExists reports whether a profile's resolved root directory
	// exists. nil → treated as unknown-but-present (no false WARN).
	RootExists func(name string) bool
	// HasManifest reports whether a profile has a distribution manifest.
	// Absent is informational, never a WARN.
	HasManifest func(name string) bool
}

// CheckProfiles renders the Gormes-owned ◆ Profiles section content. With
// no named profiles it is a single clean PASS ("default profile only") —
// never WARN, since Gormes always has a usable default. A named profile
// whose root is missing is WARN for that item only; an absent distribution
// manifest is a NON-actionable PASS note (mirrors the Directory-Structure
// not-yet-created rule) so it cannot inflate the computed Found-N count.
func CheckProfiles(inv DoctorProfileInventory) CheckResult {
	active := strings.TrimSpace(inv.Active)
	if active == "" {
		active = "default"
	}

	named := make([]string, 0, len(inv.Known))
	for _, n := range inv.Known {
		n = strings.TrimSpace(n)
		if n == "" || n == "default" {
			continue
		}
		named = append(named, n)
	}
	sort.Strings(named)

	if len(named) == 0 {
		return CheckResult{
			Name:    "Profiles",
			Status:  StatusPass,
			Summary: "default profile only",
			Items: []ItemInfo{
				{Name: "default", Status: StatusPass, Note: "default profile only (create more with `gormes profile create`)"},
			},
		}
	}

	worst := StatusPass
	items := make([]ItemInfo, 0, len(named)+1)
	items = append(items, ItemInfo{
		Name:   "summary",
		Status: StatusPass,
		Note:   fmt.Sprintf("%d profile(s) found (active: %s)", len(named), active),
	})
	for _, name := range named {
		label := name
		if name == active {
			label = name + " (active)"
		}
		rootMissing := inv.RootExists != nil && !inv.RootExists(name)
		hasManifest := inv.HasManifest != nil && inv.HasManifest(name)

		switch {
		case rootMissing:
			worst = StatusWarn
			items = append(items, ItemInfo{
				Name:   label,
				Status: StatusWarn,
				Note:   "profile root missing — recreate with `gormes profile create`",
			})
		case hasManifest:
			items = append(items, ItemInfo{Name: label, Status: StatusPass, Note: "configured (distribution manifest present)"})
		default:
			// Absent manifest is informational only — non-actionable PASS,
			// never WARN, so it stays out of the Found-N issue summary.
			items = append(items, ItemInfo{Name: label, Status: StatusPass, Note: "configured (no distribution manifest)"})
		}
	}

	summary := fmt.Sprintf("%d profile(s) found", len(named))
	if worst == StatusWarn {
		summary += "; some profile roots are missing (recreate with `gormes profile create`)"
	}
	return CheckResult{Name: "Profiles", Status: worst, Summary: summary, Items: items}
}
