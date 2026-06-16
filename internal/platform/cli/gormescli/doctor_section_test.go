package gormescli

import (
	"strings"
	"testing"
)

// `gormes doctor` (human mode) must present its checks under upstream-parity
// `◆ <Section>` headers, with the shipped `Found N issue(s)` summary still
// rendered AFTER the sections, and each check rendered exactly once (the old
// flat per-check stream must be replaced by the grouped render, not doubled).
func TestDoctorCommandHumanOutputIsSectionGrouped(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, stderr, _ := executeCobraCommandForTest(newRootCommand(), cobraCommandExecutionOptions{}, "doctor", "--offline")

	out := stdout + stderr
	if !strings.Contains(out, "Gormes Doctor") {
		t.Fatalf("human doctor output must start with a Hermes-style boxed Gormes Doctor header:\n%s", out)
	}
	if !strings.Contains(out, "◆ Tool Availability") {
		t.Fatalf("human doctor output must group checks under ◆ section headers:\n%s", out)
	}
	if !strings.Contains(out, "◆ Configuration Files") {
		t.Fatalf("human doctor output missing ◆ Configuration Files section:\n%s", out)
	}
	// Toolbox check renders exactly once (grouped, compacted, not also flat-streamed).
	if c := strings.Count(out, "✓ Toolbox —"); c != 1 {
		t.Fatalf("Toolbox check rendered %d times, want exactly 1 (grouped render must replace the flat stream):\n%s", c, out)
	}
	if strings.Contains(out, "[PASS]") || strings.Contains(out, "[WARN]") || strings.Contains(out, "[FAIL]") {
		t.Fatalf("human doctor output should use Hermes-style glyph rows, not bracketed status tags:\n%s", out)
	}
	// The shipped Found-N summary still renders, and AFTER the sections.
	foundIdx := strings.Index(out, "issue(s) to address:")
	if foundIdx < 0 {
		foundIdx = strings.Index(out, "No issues")
	}
	secIdx := strings.Index(out, "◆ Tool Availability")
	if foundIdx < 0 {
		t.Fatalf("shipped Found-N / no-issues summary must still render:\n%s", out)
	}
	if secIdx < 0 || foundIdx < secIdx {
		t.Fatalf("the issues summary must render AFTER the ◆ sections (sec=%d found=%d):\n%s", secIdx, foundIdx, out)
	}
	// Gormes-owned wording only.
	for _, forbidden := range []string{"hermes setup", "~/.hermes", "hermes doctor"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("doctor leaked Hermes-owned wording %q:\n%s", forbidden, out)
		}
	}
}

func TestDoctorCommandFreshHumanModeContinuesAfterProviderFailure(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("HOME", t.TempDir())

	stdout, stderr, err := executeCobraCommandForTest(newRootCommand(), cobraCommandExecutionOptions{}, "doctor")
	if err == nil {
		t.Fatalf("doctor fresh human mode err = nil, want provider setup failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1 for provider setup failure\nstdout=%s\nstderr=%s\nerr=%v", code, stdout, stderr, err)
	}

	out := stdout + stderr
	for _, want := range []string{
		"Gormes Doctor",
		"◆ Configuration Files",
		"◆ API Connectivity",
		"◆ Tool Availability",
		"✗ Provider setup —",
		"✓ Toolbox —",
		"Found ",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("fresh human doctor output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "No provider endpoint or credential-backed provider is configured.") {
		t.Fatalf("provider setup failure should use operator-facing provider wording:\n%s", out)
	}
	if strings.Contains(out, "hermes endpoint unconfigured") {
		t.Fatalf("provider setup failure leaked raw hermes endpoint wording:\n%s", out)
	}
	if strings.Contains(out, "Configure Gormes provider credentials/endpoint") {
		t.Fatalf("provider failure must be rendered inside the sectioned report, not as a pre-report stderr banner:\n%s", out)
	}
}

// JSON mode is a fleet-stable wire contract: sections are a human-mode
// presentation concern only and must NOT appear in `gormes doctor --json`.
func TestDoctorCommandJSONHasNoSectionHeaders(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, _, _ := executeCobraCommandForTest(newRootCommand(), cobraCommandExecutionOptions{}, "doctor", "--offline", "--json")

	out := stdout
	if strings.Contains(out, "◆") {
		t.Fatalf("--json output must not contain ◆ section headers (human-mode only):\n%s", out)
	}
	if !strings.Contains(out, `"checks"`) {
		t.Fatalf("--json must still emit the {checks:[...]} document:\n%s", out)
	}
}
