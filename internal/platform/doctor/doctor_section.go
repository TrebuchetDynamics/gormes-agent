package doctor

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

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

const (
	doctorANSIReset  = "\x1b[0m"
	doctorANSIBold   = "\x1b[1m"
	doctorANSIDim    = "\x1b[2m"
	doctorANSICyan   = "\x1b[36m"
	doctorANSIGreen  = "\x1b[32m"
	doctorANSIYellow = "\x1b[33m"
	doctorANSIRed    = "\x1b[31m"
)

// RenderStyle controls human doctor ANSI color. The default zero value is
// deliberately plain text so tests, pipes, and JSON-adjacent captures remain
// stable. cmd/gormes passes RenderStyleForWriter(stdout) for terminal color.
type RenderStyle struct {
	Color bool
}

// RenderStyleForWriter follows the NO_COLOR convention and only enables color
// for real TTY writers. It duplicates the tiny terminal gate locally so the
// doctor package stays usable without depending on the setup-wizard UI helpers.
func RenderStyleForWriter(w io.Writer) RenderStyle {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return RenderStyle{}
	}
	file, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return RenderStyle{}
	}
	return RenderStyle{Color: true}
}

func (s RenderStyle) wrap(code, text string) string {
	if !s.Color || text == "" {
		return text
	}
	return code + text + doctorANSIReset
}

func (s RenderStyle) cyanBold(text string) string { return s.wrap(doctorANSICyan+doctorANSIBold, text) }
func (s RenderStyle) dim(text string) string      { return s.wrap(doctorANSIDim, text) }
func (s RenderStyle) green(text string) string    { return s.wrap(doctorANSIGreen, text) }
func (s RenderStyle) yellow(text string) string   { return s.wrap(doctorANSIYellow, text) }
func (s RenderStyle) red(text string) string      { return s.wrap(doctorANSIRed, text) }

func (s RenderStyle) glyph(status Status) string {
	glyph := statusGlyph(status)
	switch status {
	case StatusPass:
		return s.green(glyph)
	case StatusFail:
		return s.red(glyph)
	case StatusWarn:
		return s.yellow(glyph)
	case StatusSkip:
		return s.dim(glyph)
	default:
		return glyph
	}
}

func statusGlyph(status Status) string {
	switch status {
	case StatusPass:
		return "✓"
	case StatusFail:
		return "✗"
	case StatusWarn:
		return "⚠"
	case StatusSkip:
		return "-"
	default:
		return "?"
	}
}

// RenderDoctorHeader renders the same boxed report banner shape as upstream
// Hermes doctor, with the caller-provided product title.
func RenderDoctorHeader(title string) string {
	return RenderDoctorHeaderStyled(title, RenderStyle{})
}

// RenderDoctorHeaderStyled is RenderDoctorHeader plus optional terminal color.
func RenderDoctorHeaderStyled(title string, style RenderStyle) string {
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
	lines := []string{
		"┌" + strings.Repeat("─", width+2) + "┐",
		"│ " + strings.Repeat(" ", left) + title + strings.Repeat(" ", right) + " │",
		"└" + strings.Repeat("─", width+2) + "┘",
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(style.wrap(doctorANSICyan, line))
		b.WriteString("\n")
	}
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
	case "Directory Structure":
		return SectionDirectoryStructure
	case "Auth Providers", "SecretRef runtime":
		return SectionAuthProviders
	case "Termux runtime", "Browser runtime", "ACP bridge", "GitHub auth":
		return SectionExternalTools
	case "provider health", "provider setup",
		"Gateway Slack", "gateway", "gateway/telegram", "gateway/discord":
		return SectionAPIConnectivity
	case "Native TUI", "Toolbox", "Web tools":
		return SectionToolAvailability
	case "Skills Hub":
		return SectionSkillsHub
	case "Goncho config":
		return SectionMemoryProvider
	case "Profiles":
		return SectionProfiles
	case "Security Advisories":
		return SectionSecurityAdvisories
	default:
		return SectionGormesRuntime
	}
}

// RenderDoctorStatusSummary renders a compact count strip for human doctor
// runs. It is intentionally separate from RenderSectionedReport so JSON mode
// and section-order tests can stay focused on check grouping.
func RenderDoctorStatusSummary(results []CheckResult, style RenderStyle) string {
	if len(results) == 0 {
		return ""
	}
	var pass, warn, fail, skip int
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			pass++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		case StatusSkip:
			skip++
		}
	}
	parts := []string{fmt.Sprintf("%d pass", pass)}
	if warn > 0 {
		parts = append(parts, fmt.Sprintf("%d warn", warn))
	}
	if fail > 0 {
		parts = append(parts, fmt.Sprintf("%d fail", fail))
	}
	if skip > 0 {
		parts = append(parts, fmt.Sprintf("%d skip", skip))
	}
	return "\n  " + style.dim("Summary:") + " " + strings.Join(parts, " · ") + "\n"
}

// RenderSectionedReport groups results under `◆ <Section>` headers in the
// fixed OrderedDoctorSections order. Parity with hermes doctor's
// `print(); print(color("◆ <Name>", ...))`: each header is preceded by a
// blank line. A section with zero mapped checks prints no header. Within a
// section, checks render in arrival order using the Hermes-like glyph rows
// (`✓`, `⚠`, `✗`) instead of the old flat `[PASS]` stream.
func RenderSectionedReport(results []CheckResult) string {
	return RenderSectionedReportWithStyle(results, RenderStyle{})
}

// RenderSectionedReportWithStyle is RenderSectionedReport plus optional ANSI
// color for TTY output.
func RenderSectionedReportWithStyle(results []CheckResult, style RenderStyle) string {
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
		b.WriteString("\n")
		b.WriteString(style.cyanBold("◆ " + section))
		b.WriteString("\n")
		for _, c := range checks {
			b.WriteString(formatSectionedCheckStyled(c, section, style))
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "\n" + b.String()
}

func formatSectionedCheckStyled(c CheckResult, section string, style RenderStyle) string {
	if c.Name == "Toolbox" {
		return formatToolboxCheck(c, style)
	}
	summary := strings.TrimSpace(c.Summary)
	name := displayCheckName(c.Name)
	lineText := summary
	if lineText == "" {
		lineText = name
	}
	if strings.TrimSpace(c.Name) != section {
		if summary == "" {
			lineText = name
		} else {
			lineText = name + " — " + summary
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  %s %s\n", style.glyph(c.Status), lineText)
	if len(c.Items) == 0 || redundantSingleItem(c) {
		return b.String()
	}
	b.WriteString(formatItems(c.Items, style))
	return b.String()
}

func formatToolboxCheck(c CheckResult, style RenderStyle) string {
	var b strings.Builder
	if allItemsStatus(c.Items, StatusPass) {
		headline := toolboxHeadline(c)
		fmt.Fprintf(&b, "  %s Toolbox", style.glyph(c.Status))
		if headline != "" {
			fmt.Fprintf(&b, " — %s", headline)
		}
		b.WriteString("\n")
		if names := itemNames(c.Items); len(names) > 0 {
			appendWrappedLine(&b, "    "+style.cyanBold("→")+" enabled: ", strings.Join(names, ", "), "      ", 110)
		}
		return b.String()
	}

	summary := strings.TrimSpace(c.Summary)
	if summary == "" {
		summary = "tool availability needs attention"
	}
	fmt.Fprintf(&b, "  %s Toolbox — %s\n", style.glyph(c.Status), summary)
	for _, it := range c.Items {
		if it.Status == StatusPass {
			continue
		}
		b.WriteString(formatItem(it, maxItemNameWidth(c.Items), style))
	}
	return b.String()
}

func formatItems(items []ItemInfo, style RenderStyle) string {
	var b strings.Builder
	nameW := maxItemNameWidth(items)
	for _, it := range items {
		b.WriteString(formatItem(it, nameW, style))
	}
	return b.String()
}

func formatItem(it ItemInfo, nameW int, style RenderStyle) string {
	name := strings.TrimSpace(it.Name)
	if name == "" {
		name = "item"
	}
	note := strings.TrimSpace(it.Note)
	prefix := fmt.Sprintf("    %s %-*s", style.glyph(it.Status), nameW, name)
	if note == "" {
		return strings.TrimRight(prefix, " ") + "\n"
	}
	var b strings.Builder
	appendWrappedLine(&b, prefix+"  ", note, strings.Repeat(" ", len(prefix)+2), 110)
	return b.String()
}

func appendWrappedLine(b *strings.Builder, prefix, text, continuationPrefix string, width int) {
	text = strings.TrimSpace(text)
	if text == "" {
		b.WriteString(strings.TrimRight(prefix, " "))
		b.WriteString("\n")
		return
	}
	lineBudget := width - len(prefix)
	if lineBudget < 24 {
		lineBudget = 24
	}
	for first := true; text != ""; first = false {
		currentPrefix := continuationPrefix
		budget := width - len(continuationPrefix)
		if first {
			currentPrefix = prefix
			budget = lineBudget
		}
		if budget < 24 {
			budget = 24
		}
		part := text
		if len(part) > budget {
			cut := strings.LastIndexAny(part[:budget], " ,;")
			if cut < budget/2 {
				cut = budget
			}
			part = strings.TrimSpace(text[:cut])
			text = strings.TrimLeft(strings.TrimSpace(text[cut:]), ",; ")
		} else {
			text = ""
		}
		b.WriteString(currentPrefix)
		b.WriteString(part)
		b.WriteString("\n")
	}
}

func maxItemNameWidth(items []ItemInfo) int {
	nameW := 0
	for _, it := range items {
		if n := len(strings.TrimSpace(it.Name)); n > nameW {
			nameW = n
		}
	}
	return nameW
}

func allItemsStatus(items []ItemInfo, status Status) bool {
	if len(items) == 0 {
		return false
	}
	for _, it := range items {
		if it.Status != status {
			return false
		}
	}
	return true
}

func itemNames(items []ItemInfo) []string {
	names := make([]string, 0, len(items))
	for _, it := range items {
		if name := strings.TrimSpace(it.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func redundantSingleItem(c CheckResult) bool {
	if len(c.Items) != 1 {
		return false
	}
	it := c.Items[0]
	return it.Status == c.Status && strings.EqualFold(strings.TrimSpace(it.Note), strings.TrimSpace(c.Summary))
}

func toolboxHeadline(c CheckResult) string {
	summary := strings.TrimSpace(c.Summary)
	if idx := strings.Index(summary, " ("); idx > 0 {
		summary = strings.TrimSpace(summary[:idx])
	}
	if summary == "" && len(c.Items) > 0 {
		summary = fmt.Sprintf("%d tools registered", len(c.Items))
	}
	return summary
}

func displayCheckName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Check"
	}
	if strings.Contains(name, "/") {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
