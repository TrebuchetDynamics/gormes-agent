package doctor

import "strings"

// Doctor output section taxonomy. Parity with upstream hermes doctor
// (hermes_cli/doctor.py@55c9f3206:297 run_doctor), which renders its checks
// under labeled `◆ <Section>` headers in a fixed order:
//
//	Security Advisories  (doctor.py:350)
//	Configuration Files  (:460)
//	Auth Providers       (:749)
//	Directory Structure  (:812)
//	External Tools       (:1009)
//	API Connectivity     (:1263)
//	Tool Availability    (:1604)
//	Skills Hub           (:1637)
//	Memory Provider      (:1683)
//	Profiles             (:1768)
//
// Gormes groups its EXISTING checks under this same taxonomy (no new
// diagnostic data). Sections with no mapped check print no header. Any
// gormes check that maps to no upstream section falls under a single
// explicit Gormes-owned catch-all section rather than being dropped.
const (
	SectionSecurityAdvisories = "Security Advisories"
	SectionConfigurationFiles = "Configuration Files"
	SectionAuthProviders      = "Auth Providers"
	SectionDirectoryStructure = "Directory Structure"
	SectionExternalTools      = "External Tools"
	SectionAPIConnectivity    = "API Connectivity"
	SectionToolAvailability   = "Tool Availability"
	SectionSkillsHub          = "Skills Hub"
	SectionMemoryProvider     = "Memory Provider"
	SectionProfiles           = "Profiles"
	// SectionGormesRuntime is the Gormes-owned catch-all. It is an
	// intentional owned divergence: gormes-only checks with no 1:1 Hermes
	// section land here (named, never silently dropped). Rendered last,
	// after Profiles.
	SectionGormesRuntime = "Gormes Runtime"
)

// RenderDoctorHeader renders the same boxed report banner shape as upstream
// Hermes doctor, with the caller-provided product title.
func RenderDoctorHeader(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Doctor"
	}
	width := 55
	if len(title) > width {
		width = len(title)
	}
	left := (width - len(title)) / 2
	right := width - len(title) - left
	var b strings.Builder
	b.WriteString("┌")
	b.WriteString(strings.Repeat("─", width+2))
	b.WriteString("┐\n")
	b.WriteString("│ ")
	b.WriteString(strings.Repeat(" ", left))
	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", right))
	b.WriteString(" │\n")
	b.WriteString("└")
	b.WriteString(strings.Repeat("─", width+2))
	b.WriteString("┘\n")
	return b.String()
}

// OrderedDoctorSections is the fixed render order: the ten upstream
// hermes-doctor sections followed by the Gormes-owned catch-all.
func OrderedDoctorSections() []string {
	return []string{
		SectionSecurityAdvisories,
		SectionConfigurationFiles,
		SectionAuthProviders,
		SectionDirectoryStructure,
		SectionExternalTools,
		SectionAPIConnectivity,
		SectionToolAvailability,
		SectionSkillsHub,
		SectionMemoryProvider,
		SectionProfiles,
		SectionGormesRuntime,
	}
}

// sectionForCheck maps a top-level CheckResult.Name to its doctor section.
// Gormes-only checks (build identity, Native TUI, Termux runtime, Goncho
// config, Custom endpoint, gateway channels) are placed under their nearest
// upstream section as an explicit, documented owned divergence. An unmapped
// name falls back to the Gormes-owned catch-all (never dropped).
func sectionForCheck(name string) string {
	switch name {
	case "build identity", "Custom endpoint", "target readiness", "config schema":
		return SectionConfigurationFiles
	case "SecretRef runtime":
		return SectionAuthProviders
	case "Termux runtime", "Browser runtime", "ACP bridge", "GitHub auth":
		return SectionExternalTools
	case "provider health", "provider setup",
		"Gateway Slack", "gateway", "gateway/telegram", "gateway/discord":
		return SectionAPIConnectivity
	case "Native TUI", "Toolbox", "Web tools":
		return SectionToolAvailability
	case "Goncho config":
		return SectionMemoryProvider
	default:
		return SectionGormesRuntime
	}
}

// RenderSectionedReport groups results under `◆ <Section>` headers in the
// fixed OrderedDoctorSections order. Parity with hermes doctor's
// `print(); print(color("◆ <Name>", ...))`: each header is preceded by a
// blank line. A section with zero mapped checks prints no header. Within a
// section, checks render via the existing CheckResult.Format() in arrival
// order, so no check's data or PASS/WARN/FAIL semantics change.
func RenderSectionedReport(results []CheckResult) string {
	grouped := make(map[string][]CheckResult, len(OrderedDoctorSections()))
	for _, r := range results {
		s := sectionForCheck(r.Name)
		grouped[s] = append(grouped[s], r)
	}
	var b strings.Builder
	for _, section := range OrderedDoctorSections() {
		checks := grouped[section]
		if len(checks) == 0 {
			continue
		}
		// Blank line before every header, including the first, for parity
		// with hermes doctor's `print(); print(color("◆ <Name>", ...))`.
		b.WriteString("\n◆ ")
		b.WriteString(section)
		b.WriteString("\n")
		for _, c := range checks {
			b.WriteString(formatSectionedCheck(c))
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "\n" + b.String()
}

func formatSectionedCheck(c CheckResult) string {
	if c.Name != "Toolbox" || len(c.Items) == 0 {
		return c.Format()
	}
	var b strings.Builder
	b.WriteString("[")
	b.WriteString(c.Status.String())
	b.WriteString("] ")
	b.WriteString(c.Name)
	b.WriteString(": ")
	b.WriteString(c.Summary)
	b.WriteString("\n")
	for _, it := range c.Items {
		b.WriteString("  ")
		b.WriteString(it.Status.Symbol())
		b.WriteString(" ")
		b.WriteString(it.Name)
		if it.Status != StatusPass && strings.TrimSpace(it.Note) != "" {
			b.WriteString("  ")
			b.WriteString(it.Note)
		}
		b.WriteString("\n")
	}
	return b.String()
}
