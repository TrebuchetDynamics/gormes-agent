package doctor

import (
	"strings"
	"testing"
)

// hermes doctor (hermes_cli/doctor.py@55c9f3206:297 run_doctor) groups its
// checks under labeled `◆ <Section>` headers in a fixed order. gormes doctor
// must present the SAME organized UX over its existing checks: a blank line
// then `◆ <Section>`, in the upstream order, header only for sections that
// actually have a check, with each section's checks rendered beneath it.
func TestRenderSectionedReportGroupsExistingChecksInUpstreamOrder(t *testing.T) {
	results := []CheckResult{
		{Name: "build identity", Status: StatusPass, Summary: "version=0.2.12"},
		{Name: "SecretRef runtime", Status: StatusPass, Summary: "resolved=1"},
		{Name: "Goncho config", Status: StatusPass, Summary: "enabled=true"},
		{Name: "Browser runtime", Status: StatusWarn, Summary: "cdp_unconfigured"},
		{Name: "provider health", Status: StatusSkip, Summary: "skipped (--offline)"},
		{Name: "Toolbox", Status: StatusPass, Summary: "27 tools"},
	}

	out := RenderSectionedReport(results)

	// Headers present for sections that have a check.
	for _, want := range []string{
		"◆ Configuration Files",
		"◆ Auth Providers",
		"◆ External Tools",
		"◆ API Connectivity",
		"◆ Tool Availability",
		"◆ Memory Provider",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("sectioned report missing header %q:\n%s", want, out)
		}
	}

	// No header for sections with zero mapped checks (deferred content rows).
	for _, absent := range []string{"◆ Security Advisories", "◆ Directory Structure", "◆ Skills Hub", "◆ Profiles"} {
		if strings.Contains(out, absent) {
			t.Fatalf("empty section must not print a header, but found %q:\n%s", absent, out)
		}
	}

	// Upstream section order: Configuration Files < Auth Providers <
	// External Tools < API Connectivity < Tool Availability < Memory Provider.
	order := []string{
		"◆ Configuration Files",
		"◆ Auth Providers",
		"◆ External Tools",
		"◆ API Connectivity",
		"◆ Tool Availability",
		"◆ Memory Provider",
	}
	prev := -1
	for _, h := range order {
		idx := strings.Index(out, h)
		if idx <= prev {
			t.Fatalf("section %q is out of upstream order (idx=%d, prev=%d):\n%s", h, idx, prev, out)
		}
		prev = idx
	}

	// Each check renders under (after) its section header, not before it.
	cfgIdx := strings.Index(out, "◆ Configuration Files")
	biIdx := strings.Index(out, "Build identity")
	if biIdx < cfgIdx {
		t.Fatalf("check 'build identity' must render after its ◆ Configuration Files header:\n%s", out)
	}
	// A blank line precedes each header (parity with hermes print(); print(...)).
	if !strings.Contains(out, "\n\n◆ Configuration Files") {
		t.Fatalf("each ◆ header must be preceded by a blank line:\n%s", out)
	}
}

func TestRenderSectionedReportUsesHermesGlyphRowsAndCompactsPassingToolbox(t *testing.T) {
	out := RenderSectionedReport([]CheckResult{
		{Name: "Security Advisories", Status: StatusPass, Summary: "No active security advisories", Items: []ItemInfo{{Name: "advisories", Status: StatusPass, Note: "No active security advisories"}}},
		{Name: "Toolbox", Status: StatusPass, Summary: "3 tools registered (read_file, terminal, web_search)", Items: []ItemInfo{
			{Name: "read_file", Status: StatusPass, Note: "Read files"},
			{Name: "terminal", Status: StatusPass, Note: "Run commands"},
			{Name: "web_search", Status: StatusPass, Note: "Search web"},
		}},
	})

	for _, want := range []string{
		"◆ Security Advisories",
		"  ✓ No active security advisories\n",
		"◆ Tool Availability",
		"  ✓ Toolbox — 3 tools registered\n",
		"    → enabled: read_file, terminal, web_search\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("human doctor output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"[PASS]", "  ✓ read_file\n", "  ✓ terminal\n", "Read files", "Run commands"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("human doctor output should be glyph-based and compact; found %q:\n%s", forbidden, out)
		}
	}
}

func TestRenderSectionedReportStyledUsesANSIWhenEnabled(t *testing.T) {
	out := RenderSectionedReportWithStyle([]CheckResult{{Name: "provider setup", Status: StatusFail, Summary: "endpoint missing"}}, RenderStyle{Color: true})
	for _, want := range []string{"\x1b[36m\x1b[1m◆ API Connectivity\x1b[0m", "\x1b[31m✗\x1b[0m Provider setup"} {
		if !strings.Contains(out, want) {
			t.Fatalf("styled doctor output missing ANSI %q:\n%q", want, out)
		}
	}
}

func TestRenderDoctorHeaderMatchesHermesBoxShape(t *testing.T) {
	out := RenderDoctorHeader("Gormes Doctor")
	for _, want := range []string{
		"┌─────────────────────────────────────────────────────────┐",
		"Gormes Doctor",
		"└─────────────────────────────────────────────────────────┘",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor header missing %q:\n%s", want, out)
		}
	}
}

func TestRenderSectionedReportCompactsToolboxItemsForHumanParity(t *testing.T) {
	out := RenderSectionedReport([]CheckResult{{
		Name:    "Toolbox",
		Status:  StatusPass,
		Summary: "2 tools registered (read_file, terminal)",
		Items: []ItemInfo{
			{Name: "read_file", Status: StatusPass, Note: "Read a text file with line numbers and pagination."},
			{Name: "terminal", Status: StatusPass, Note: "Execute a local shell command with timeout handling."},
		},
	}})
	for _, want := range []string{"  ✓ Toolbox — 2 tools registered\n", "    → enabled: read_file, terminal\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("compact toolbox output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"[PASS]", "  ✓ read_file\n", "  ✓ terminal\n", "Read a text file", "Execute a local shell command"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("human toolbox output should be compact like Hermes and omit long passing notes %q:\n%s", forbidden, out)
		}
	}
}

func TestRenderSectionedReportKeepsToolboxFailureNotes(t *testing.T) {
	out := RenderSectionedReport([]CheckResult{{
		Name:    "Toolbox",
		Status:  StatusFail,
		Summary: "1 of 2 tools have invalid schemas (broken)",
		Items: []ItemInfo{
			{Name: "broken", Status: StatusFail, Note: `schema: missing "type"`},
			{Name: "ok", Status: StatusPass, Note: "long success note"},
		},
	}})
	if !strings.Contains(out, `  ✗ broken  schema: missing "type"`) {
		t.Fatalf("toolbox schema failures must keep diagnostic notes:\n%s", out)
	}
	if strings.Contains(out, "long success note") {
		t.Fatalf("passing toolbox notes should stay compact even in mixed output:\n%s", out)
	}
}

// Owned divergence: gormes-only checks with no 1:1 Hermes section are placed
// under a documented nearest section, never silently dropped and never
// dumped into the catch-all by accident.
func TestSectionForCheckOwnedDivergencePlacement(t *testing.T) {
	cases := map[string]string{
		"build identity":    SectionConfigurationFiles,
		"Custom endpoint":   SectionConfigurationFiles,
		"target readiness":  SectionConfigurationFiles,
		"SecretRef runtime": SectionAuthProviders,
		"Termux runtime":    SectionExternalTools,
		"Browser runtime":   SectionExternalTools,
		"ACP bridge":        SectionExternalTools,
		"GitHub auth":       SectionExternalTools,
		"provider health":   SectionAPIConnectivity,
		"Gateway Slack":     SectionAPIConnectivity,
		"gateway/telegram":  SectionAPIConnectivity,
		"gateway/discord":   SectionAPIConnectivity,
		"Native TUI":        SectionToolAvailability,
		"Toolbox":           SectionToolAvailability,
		"Web tools":         SectionToolAvailability,
		"Goncho config":     SectionMemoryProvider,
	}
	for name, want := range cases {
		if got := sectionForCheck(name); got != want {
			t.Fatalf("sectionForCheck(%q) = %q, want %q (owned-divergence placement must be stable)", name, got, want)
		}
		if want == SectionGormesRuntime {
			t.Fatalf("documented gormes-only check %q must NOT fall into the catch-all", name)
		}
	}

	// Every input check appears exactly once in the rendered report (none dropped).
	var results []CheckResult
	for name := range cases {
		results = append(results, CheckResult{Name: name, Status: StatusPass, Summary: "x"})
	}
	out := RenderSectionedReport(results)
	for name := range cases {
		want := "✓ " + displayCheckName(name) + " — x"
		if c := strings.Count(out, want); c != 1 {
			t.Fatalf("check %q rendered %d times, want exactly 1 as %q (no drop/dupe):\n%s", name, c, want, out)
		}
	}
}

// Degraded mode: a check that maps to no known upstream section lands under
// the explicit Gormes-owned catch-all `◆ Gormes Runtime` header — never
// dropped, never a panic, never an empty `◆` line.
func TestRenderSectionedReportUnknownCheckFallsToCatchAll(t *testing.T) {
	out := RenderSectionedReport([]CheckResult{
		{Name: "some-future-unmapped-check", Status: StatusWarn, Summary: "novel"},
	})
	if !strings.Contains(out, "◆ "+SectionGormesRuntime) {
		t.Fatalf("unmapped check must render under the ◆ %s catch-all:\n%s", SectionGormesRuntime, out)
	}
	if !strings.Contains(out, "⚠ Some-future-unmapped-check — novel") {
		t.Fatalf("unmapped check must still be rendered, not dropped:\n%s", out)
	}
}
