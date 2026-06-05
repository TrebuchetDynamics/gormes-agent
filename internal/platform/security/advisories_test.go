package security

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Owned divergence (parity intent hermes_cli/security_advisories.py@55c9f3206):
// the Gormes catalog mirrors the upstream Advisory shape and carries the same
// shai-hulud advisory as catalog DATA, but Gormes is a single Go binary with no
// Python venv, so the detector is an injectable seam — the default finds
// nothing and the catalog produces no hits (Hermes' "silent otherwise").
func TestDefaultCatalogCarriesShaiHuludAsData(t *testing.T) {
	cat := DefaultCatalog()
	if len(cat) == 0 {
		t.Fatalf("default catalog must carry the upstream advisory data")
	}
	var found *Advisory
	for i := range cat {
		if cat[i].ID == "shai-hulud-2026-05" {
			found = &cat[i]
		}
	}
	if found == nil {
		t.Fatalf("catalog must include shai-hulud-2026-05, got %+v", cat)
	}
	if found.Severity != "critical" || len(found.Remediation) == 0 || len(found.Compromised) == 0 {
		t.Fatalf("advisory data incomplete: %+v", *found)
	}
	if found.Compromised[0].Package != "mistralai" {
		t.Fatalf("compromised package = %q, want mistralai", found.Compromised[0].Package)
	}
}

// DetectCompromised matches only when the injected seam reports a bad version.
// The default (no-arg / empty) seam yields zero hits in a pure-Go runtime.
func TestDetectCompromisedHonorsInjectedSeam(t *testing.T) {
	cat := DefaultCatalog()

	none := DetectCompromised(cat, func(string) string { return "" })
	if len(none) != 0 {
		t.Fatalf("empty detector seam must yield no hits, got %+v", none)
	}

	clean := DetectCompromised(cat, func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.7" // not the compromised version
		}
		return ""
	})
	if len(clean) != 0 {
		t.Fatalf("non-compromised installed version must not hit, got %+v", clean)
	}

	bad := DetectCompromised(cat, func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	})
	if len(bad) != 1 {
		t.Fatalf("compromised version must produce exactly one hit, got %+v", bad)
	}
	if bad[0].Package != "mistralai" || bad[0].InstalledVersion != "2.4.6" {
		t.Fatalf("hit = %+v, want mistralai==2.4.6", bad[0])
	}
	if bad[0].Advisory.ID != "shai-hulud-2026-05" {
		t.Fatalf("hit advisory = %q", bad[0].Advisory.ID)
	}
}

// The ack store is Gormes-owned under ~/.gormes (dir-injected like
// CheckDirectoryStructure(home)). Round-trips; a missing store is an empty set
// with no error and never panics; FilterUnacked drops acked advisories.
func TestAckStoreRoundTripAndFilterUnacked(t *testing.T) {
	home := t.TempDir()
	store := NewAckStore(home)

	acked, err := store.AckedIDs()
	if err != nil {
		t.Fatalf("missing ack store must not error: %v", err)
	}
	if len(acked) != 0 {
		t.Fatalf("missing ack store must be empty, got %v", acked)
	}

	if err := store.Ack("shai-hulud-2026-05"); err != nil {
		t.Fatalf("Ack failed: %v", err)
	}
	if err := store.Ack("shai-hulud-2026-05"); err != nil {
		t.Fatalf("Ack must be idempotent: %v", err)
	}

	reloaded := NewAckStore(home)
	acked, err = reloaded.AckedIDs()
	if err != nil {
		t.Fatalf("AckedIDs after persist: %v", err)
	}
	if _, ok := acked["shai-hulud-2026-05"]; !ok {
		t.Fatalf("ack did not persist across store instances: %v", acked)
	}

	hits := DetectCompromised(DefaultCatalog(), func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	})
	if got := len(FilterUnacked(hits, map[string]struct{}{})); got != 1 {
		t.Fatalf("no acks → all hits unacked, want 1 got %d", got)
	}
	if got := len(FilterUnacked(hits, acked)); got != 0 {
		t.Fatalf("acked advisory must be filtered out, want 0 got %d", got)
	}

	if !strings.Contains(filepath.Dir(store.path()), home) {
		t.Fatalf("ack store must live under the injected ~/.gormes home, path=%q", store.path())
	}
}

// FullRemediationText renders the upstream advisory content verbatim (the
// `pip uninstall` line is upstream advisory DATA, not Gormes scaffolding).
func TestFullRemediationTextRendersAdvisoryContent(t *testing.T) {
	hits := DetectCompromised(DefaultCatalog(), func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	})
	if len(hits) != 1 {
		t.Fatalf("setup: want 1 hit, got %d", len(hits))
	}
	lines := FullRemediationText(hits[0])
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "shai-hulud-2026-05") {
		t.Fatalf("remediation must name the advisory id:\n%s", joined)
	}
	if !strings.Contains(joined, "mistralai==2.4.6") {
		t.Fatalf("remediation must name the detected package==version:\n%s", joined)
	}
	if !strings.Contains(strings.ToLower(joined), "remediation") {
		t.Fatalf("remediation block must include the steps header:\n%s", joined)
	}
}

func TestRenderDoctorSectionReportsUnackedSecurityAdvisories(t *testing.T) {
	hits := DetectCompromised(DefaultCatalog(), func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	})
	if len(hits) != 1 {
		t.Fatalf("setup: want 1 hit, got %d", len(hits))
	}

	hasProblems, lines := RenderDoctorSection(hits, map[string]struct{}{})
	if !hasProblems {
		t.Fatal("active advisory should mark doctor section as problematic")
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"=== Mini Shai-Hulud worm",
		"ID:        shai-hulud-2026-05",
		"Detected:  mistralai==2.4.6",
		"Remediation:",
		"Run: pip uninstall -y mistralai",
		"After cleanup: gormes doctor --ack shai-hulud-2026-05",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("doctor section missing %q:\n%s", want, joined)
		}
	}

	ackedProblems, ackedLines := RenderDoctorSection(hits, map[string]struct{}{"shai-hulud-2026-05": {}})
	if ackedProblems {
		t.Fatalf("acked advisory should not mark doctor section as problematic: %v", ackedLines)
	}
	if got := strings.Join(ackedLines, "\n"); !strings.Contains(got, "No active security advisories.  ✓") {
		t.Fatalf("acked/clean doctor section should render no-active message, got:\n%s", got)
	}

	second := hits[0]
	second.Advisory.ID = "secondary-2026-05"
	second.Advisory.Title = "Secondary compromised package"
	second.Package = "secondarypkg"
	second.InstalledVersion = "0.1.0"
	_, multiLines := RenderDoctorSection([]AdvisoryHit{hits[0], second}, map[string]struct{}{})
	multi := strings.Join(multiLines, "\n")
	if !strings.Contains(multi, "\n\n=== Secondary compromised package ===") {
		t.Fatalf("multiple doctor advisories should be separated by a blank line:\n%s", multi)
	}
}

func TestStartupBannerShowsUnackedHitsOncePerRepeatWindow(t *testing.T) {
	hits := DetectCompromised(DefaultCatalog(), func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	})
	if len(hits) != 1 {
		t.Fatalf("setup: want 1 hit, got %d", len(hits))
	}

	cache := NewBannerCache(t.TempDir())
	now := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)
	banner := StartupBanner(hits, map[string]struct{}{}, cache, now)
	if banner == "" {
		t.Fatal("first active advisory should render a startup banner")
	}
	for _, want := range []string{
		"SECURITY ADVISORY [shai-hulud-2026-05]",
		"mistralai==2.4.6",
		"Run 'gormes doctor' for remediation steps.",
	} {
		if !strings.Contains(banner, want) {
			t.Fatalf("startup banner missing %q:\n%s", want, banner)
		}
	}

	if got := StartupBanner(hits, map[string]struct{}{}, cache, now.Add(time.Hour)); got != "" {
		t.Fatalf("recently shown advisory should be suppressed, got:\n%s", got)
	}
	if got := StartupBanner(hits, map[string]struct{}{"shai-hulud-2026-05": {}}, cache, now.Add(48*time.Hour)); got != "" {
		t.Fatalf("acked advisory should not banner, got:\n%s", got)
	}
	if got := StartupBanner(hits, map[string]struct{}{}, cache, now.Add(25*time.Hour)); got == "" {
		t.Fatal("unacked advisory should banner again after repeat window")
	}
}

func TestGatewayLogMessageSummarizesUnackedSecurityAdvisories(t *testing.T) {
	hits := DetectCompromised(DefaultCatalog(), func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	})
	if len(hits) != 1 {
		t.Fatalf("setup: want 1 hit, got %d", len(hits))
	}

	msg := GatewayLogMessage(hits, map[string]struct{}{})
	for _, want := range []string{
		"Security advisory [shai-hulud-2026-05] active",
		"mistralai==2.4.6",
		"Mini Shai-Hulud worm",
		"https://socket.dev/blog/mini-shai-hulud-worm-pypi",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("gateway log message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "hermes doctor") {
		t.Fatalf("Gormes gateway log message must not point operators at hermes doctor:\n%s", msg)
	}

	if got := GatewayLogMessage(hits, map[string]struct{}{"shai-hulud-2026-05": {}}); got != "" {
		t.Fatalf("acked advisory should not produce a gateway log message, got:\n%s", got)
	}

	second := hits[0]
	second.Advisory.ID = "secondary-2026-05"
	second.Advisory.Title = "Secondary compromised package"
	second.Package = "secondarypkg"
	second.InstalledVersion = "0.1.0"
	multi := GatewayLogMessage([]AdvisoryHit{hits[0], second}, map[string]struct{}{})
	for _, want := range []string{
		"2 security advisories active",
		"shai-hulud-2026-05, secondary-2026-05",
		"Run `gormes doctor` on the gateway host for details.",
	} {
		if !strings.Contains(multi, want) {
			t.Fatalf("multi-advisory gateway log message missing %q:\n%s", want, multi)
		}
	}
}
