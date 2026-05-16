package doctor

import (
	"strings"
	"testing"
)

// Owned divergence (parity intent hermes_cli/doctor.py@55c9f3206:1768):
// Gormes always has a usable default. With no named profiles the ◆ Profiles
// section is a single clean PASS "default profile only" — never WARN.
func TestCheckProfilesDefaultOnlyIsCleanPass(t *testing.T) {
	r := CheckProfiles(DoctorProfileInventory{Known: nil, Active: ""})

	if r.Name != "Profiles" {
		t.Fatalf("CheckResult.Name = %q, want %q", r.Name, "Profiles")
	}
	if r.Status != StatusPass {
		t.Fatalf("default-only profiles must be a clean PASS, got %v summary=%q", r.Status, r.Summary)
	}
	joined := r.Summary
	for _, it := range r.Items {
		joined += " | " + it.Name + " " + it.Note
	}
	if !strings.Contains(strings.ToLower(joined), "default profile only") {
		t.Fatalf("default-only must report 'default profile only', got: %s", joined)
	}
	for _, forbidden := range []string{"~/.hermes", "hermes ", "wrapper", "alias", "gateway running"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("Profiles leaked Hermes-owned/fabricated wording %q: %s", forbidden, joined)
		}
	}
}

// Behavior 2: ≥1 named profile whose root exists and has a distribution
// manifest renders a summary item "N profile(s) found (active: <name>)" plus a
// PASS item per profile, with the active profile marked. Owned divergence:
// no per-profile gateway_running or model[:30] is fabricated.
func TestCheckProfilesNamedWithManifestPassesAndMarksActive(t *testing.T) {
	r := CheckProfiles(DoctorProfileInventory{
		Known:       []string{"work", "default", "play"},
		Active:      "work",
		RootExists:  func(string) bool { return true },
		HasManifest: func(name string) bool { return name == "work" },
	})

	if r.Status != StatusPass {
		t.Fatalf("all named roots present must be PASS, got %v summary=%q", r.Status, r.Summary)
	}
	if !strings.Contains(r.Summary, "2 profile(s) found") {
		t.Fatalf("summary must count only named profiles (default filtered): %q", r.Summary)
	}

	byName := map[string]ItemInfo{}
	for _, it := range r.Items {
		byName[it.Name] = it
	}
	summary, ok := byName["summary"]
	if !ok || !strings.Contains(summary.Note, "2 profile(s) found (active: work)") {
		t.Fatalf("missing/incorrect summary item: %+v", byName["summary"])
	}
	active, ok := byName["work (active)"]
	if !ok {
		t.Fatalf("active profile must be labeled %q, items=%+v", "work (active)", r.Items)
	}
	if active.Status != StatusPass || !strings.Contains(active.Note, "distribution manifest present") {
		t.Fatalf("active profile w/ manifest must PASS w/ manifest note, got %+v", active)
	}
	if play, ok := byName["play"]; !ok || play.Status != StatusPass {
		t.Fatalf("non-active named profile must be a plain PASS item, got %+v", byName["play"])
	}
	if _, leaked := byName["default"]; leaked {
		t.Fatalf("the default profile must not be enumerated as a named profile: %+v", r.Items)
	}
	for _, it := range r.Items {
		if strings.Contains(strings.ToLower(it.Note), "gateway running") ||
			strings.Contains(it.Note, "model[:") {
			t.Fatalf("Profiles must not fabricate Hermes-only per-profile data: %+v", it)
		}
	}
}

// Behavior 3: a named profile whose root is missing is WARN for that item and
// promotes the section to WARN (a real actionable issue). A named profile that
// merely lacks a distribution manifest is a NON-actionable PASS — it must NOT
// appear in CollectDoctorIssues, exactly like doctor_directory.go's
// not-yet-created rule, so it cannot inflate the computed Found-N count.
func TestCheckProfilesRootMissingWarnsButManifestAbsentIsNonActionable(t *testing.T) {
	missing := CheckProfiles(DoctorProfileInventory{
		Known:      []string{"ghost"},
		Active:     "default",
		RootExists: func(string) bool { return false },
	})
	if missing.Status != StatusWarn {
		t.Fatalf("missing profile root must promote section to WARN, got %v", missing.Status)
	}
	if got := len(CollectDoctorIssues([]CheckResult{missing})); got != 1 {
		t.Fatalf("a missing profile root is actionable: want 1 issue, got %d", got)
	}

	noManifest := CheckProfiles(DoctorProfileInventory{
		Known:       []string{"solo"},
		Active:      "solo",
		RootExists:  func(string) bool { return true },
		HasManifest: func(string) bool { return false },
	})
	if noManifest.Status != StatusPass {
		t.Fatalf("present root + absent manifest must stay PASS (non-actionable), got %v summary=%q",
			noManifest.Status, noManifest.Summary)
	}
	if got := len(CollectDoctorIssues([]CheckResult{noManifest})); got != 0 {
		t.Fatalf("absent distribution manifest must NOT inflate Found-N: want 0 issues, got %d", got)
	}
	var soloNote string
	for _, it := range noManifest.Items {
		if it.Name == "solo (active)" {
			soloNote = it.Note
		}
	}
	if !strings.Contains(soloNote, "no distribution manifest") {
		t.Fatalf("absent manifest must be an informational PASS note, got %q", soloNote)
	}
}
