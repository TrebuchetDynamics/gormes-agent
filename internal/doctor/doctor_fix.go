package doctor

import (
	"fmt"
	"strings"
)

// DoctorIssue is one actionable WARN/FAIL distilled from a doctor run. Hint is
// Gormes-owned remediation guidance (never Hermes wording/paths). Fixable
// marks the source-backed auto-remediation classes `gormes doctor --fix`
// understands.
type DoctorIssue struct {
	Name    string
	Hint    string
	Fixable bool
	Class   string
}

// Auto-fixable class identifiers (parity with Hermes doctor.py fixable
// issues: config migrate 693/722, published-command symlink repair 979/1003).
const (
	doctorFixClassConfigVersion    = "config-version"
	doctorFixClassPublishedSymlink = "published-symlink"
)

// FixClassConfigVersion is the exported identifier the cmd layer matches
// when wiring the real config.MigrateConfigFile applier to the
// config-version auto-fix class.
const FixClassConfigVersion = doctorFixClassConfigVersion

// CollectDoctorIssues reduces a completed doctor run to its actionable issues.
// Only StatusWarn and StatusFail count; the N in the summary is therefore
// computed from real results, never narrated (consistent with the
// "doctor counts must be computed" contract).
func CollectDoctorIssues(results []CheckResult) []DoctorIssue {
	issues := make([]DoctorIssue, 0, len(results))
	for _, r := range results {
		if r.Status != StatusWarn && r.Status != StatusFail {
			continue
		}
		class, fixable := classifyDoctorFix(r.Name, r.Summary)
		issues = append(issues, DoctorIssue{
			Name:    r.Name,
			Hint:    doctorRemediationHint(r.Name, r.Summary, class),
			Fixable: fixable,
			Class:   class,
		})
	}
	return issues
}

func classifyDoctorFix(name, summary string) (string, bool) {
	hay := strings.ToLower(name + " " + summary)
	if strings.Contains(hay, "config") && (strings.Contains(hay, "schema") ||
		strings.Contains(hay, "version") || strings.Contains(hay, "migrate")) {
		return doctorFixClassConfigVersion, true
	}
	if (strings.Contains(hay, "symlink") || strings.Contains(hay, "published command")) &&
		(strings.Contains(hay, "missing") || strings.Contains(hay, "broken")) {
		return doctorFixClassPublishedSymlink, true
	}
	return "", false
}

// doctorRemediationHint is always Gormes-owned: `gormes` commands and
// `~/.gormes` paths only — never `hermes`/`~/.hermes`.
func doctorRemediationHint(name, summary, class string) string {
	switch class {
	case doctorFixClassConfigVersion:
		return "run `gormes config migrate` (or `gormes doctor --fix`)"
	case doctorFixClassPublishedSymlink:
		return "run `gormes setup` to repair the published `gormes` command (or `gormes doctor --fix`)"
	}
	return "run `gormes setup` to configure this"
}

// DoctorFixOutcome is the computed result of attempting one auto-fix
// class. The rendered report's fixed/still-manual counts are derived from
// these values, never narrated (same "doctor counts must be computed"
// contract as the issues summary). Detail is Gormes-owned evidence or
// guidance — never Hermes wording/paths.
type DoctorFixOutcome struct {
	Class     string
	Name      string
	Fixed     bool
	AlreadyOK bool
	Detail    string
}

// AutoFixableClasses lists, in deterministic order, the source-backed
// classes `gormes doctor --fix` attempts. Parity with Hermes doctor.py
// fixable issue classes (config migrate 693/722). The published-symlink
// class is guided-manual in this slice (`gormes setup` repairs it); it is
// surfaced as still-manual rather than silently dropped.
func AutoFixableClasses() []string {
	return []string{doctorFixClassConfigVersion}
}

// RunDoctorFix applies each auto-fixable class through apply and returns
// the computed outcomes in AutoFixableClasses order. apply is the cmd-side
// seam that performs the real source-backed remediation (config migrate);
// injecting it keeps this package dependency-light and the orchestration
// hermetically testable.
func RunDoctorFix(apply func(class string) DoctorFixOutcome) []DoctorFixOutcome {
	classes := AutoFixableClasses()
	out := make([]DoctorFixOutcome, 0, len(classes))
	for _, class := range classes {
		out = append(out, apply(class))
	}
	return out
}

// RenderDoctorFixReport renders the `gormes doctor --fix` outcome block in
// Gormes wording. Fixed lines and the still-manual count are computed from
// outcomes and the residual non-auto-fixable issues, never narrated.
func RenderDoctorFixReport(outcomes []DoctorFixOutcome, stillManual []DoctorIssue) string {
	var b strings.Builder
	b.WriteString("  Auto-fix results:\n")
	fixed := 0
	for _, o := range outcomes {
		switch {
		case o.Fixed:
			fixed++
			fmt.Fprintf(&b, "  ✓ Fixed: %s — %s\n", o.Name, o.Detail)
		case o.AlreadyOK:
			fmt.Fprintf(&b, "  • %s: already current\n", o.Name)
		default:
			fmt.Fprintf(&b, "  ✗ %s: %s\n", o.Name, o.Detail)
		}
	}
	if len(stillManual) > 0 {
		fmt.Fprintf(&b, "  Still manual (%d):\n", len(stillManual))
		for i, is := range stillManual {
			fmt.Fprintf(&b, "  %d. %s — %s\n", i+1, is.Name, is.Hint)
		}
	} else if fixed == 0 {
		b.WriteString("  ✓ Nothing left to fix.\n")
	}
	return b.String()
}

// RenderDoctorIssuesSummary renders the end-of-report block parity with
// Hermes doctor.py:1824/1830, in Gormes wording. Zero issues yields a clean
// no-issues line and no `--fix` tip; the tip appears only when at least one
// issue is auto-fixable.
func RenderDoctorIssuesSummary(issues []DoctorIssue) string {
	if len(issues) == 0 {
		return "  ✓ No issues found.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  Found %d issue(s) to address:\n", len(issues))
	anyFixable := false
	for i, is := range issues {
		fmt.Fprintf(&b, "  %d. %s — %s\n", i+1, is.Name, is.Hint)
		if is.Fixable {
			anyFixable = true
		}
	}
	if anyFixable {
		b.WriteString("  Tip: run `gormes doctor --fix` to auto-fix what's possible.\n")
	}
	return b.String()
}
