// Package guard provides a static-analysis scanner for externally-sourced
// skills, mirroring the Hermes tools/skills_guard.py contract.
//
// Every skill downloaded from a registry passes through this scanner before
// installation. It uses regexp-based pattern matching to detect known-bad
// patterns (exfiltration, prompt injection, destructive commands, persistence,
// network backdoors, supply-chain weaknesses) and invisible Unicode characters
// used to hide instructions.
package guard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Severity levels match the Hermes vocabulary.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// Verdict values returned by a scan.
const (
	VerdictClean   = "clean"
	VerdictCaution = "caution"
	VerdictBlocked = "blocked"
)

// Category labels for findings.
const (
	CategoryExfiltration = "exfiltration"
	CategoryInjection    = "injection"
	CategoryDestructive  = "destructive"
	CategoryPersistence  = "persistence"
	CategoryNetwork      = "network"
	CategorySupplyChain  = "supply_chain"
	CategoryStructure    = "structure"
)

// Finding is one match from the static scanner.
type Finding struct {
	PatternID   string
	Severity    string
	Category    string
	File        string
	Line        int
	Match       string
	Description string
}

// ScanResult is the observable outcome of scanning one skill directory.
type ScanResult struct {
	SkillName string
	Verdict   string
	Findings  []Finding
	Summary   string
}

// scannableExtensions mirrors Hermes' SCANNABLE_EXTENSIONS.
var scannableExtensions = map[string]bool{
	".md": true, ".sh": true, ".bash": true, ".zsh": true,
	".py": true, ".js": true, ".ts": true, ".rb": true,
	".json": true, ".yaml": true, ".yml": true, ".toml": true,
	".txt": true, ".html": true, ".htm": true, ".xml": true,
	".env": true, ".conf": true, ".cfg": true, ".ini": true,
}

// maxFileSize: skip files larger than 512 KB (guard against time bombs).
const maxFileSize = 512 * 1024

// maxFiles: structure check — skills with this many files are suspicious.
const maxFiles = 50

// ScanSkill scans all files in skillDir for security threats and returns
// the aggregated ScanResult. skillDir must be a directory containing SKILL.md.
func ScanSkill(skillDir string) (ScanResult, error) {
	name := filepath.Base(skillDir)
	result := ScanResult{SkillName: name}

	info, err := os.Stat(skillDir)
	if err != nil {
		return result, fmt.Errorf("guard: cannot stat %s: %w", skillDir, err)
	}

	var findings []Finding

	if info.IsDir() {
		findings = append(findings, structureCheck(skillDir)...)

		err = filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil || d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(skillDir, path)
			ff, scanErr := scanFile(path, rel)
			if scanErr == nil {
				findings = append(findings, ff...)
			}
			return nil
		})
		if err != nil {
			return result, err
		}
	} else {
		findings, _ = scanFile(skillDir, filepath.Base(skillDir))
	}

	result.Findings = findings
	result.Verdict = determineVerdict(findings)
	result.Summary = buildSummary(name, result.Verdict, findings)
	return result, nil
}

// ScanSkillToError is the adapter for SkillManagerToolConfig.GuardScanner.
// Returns nil when the skill is clean or only causes caution; returns an error
// when any critical finding blocks installation.
func ScanSkillToError(skillDir string) error {
	result, err := ScanSkill(skillDir)
	if err != nil {
		return err
	}
	if result.Verdict == VerdictBlocked {
		return fmt.Errorf("skill guard blocked: %s", result.Summary)
	}
	return nil
}

// scanFile runs all threat patterns and invisible-Unicode detection over one file.
func scanFile(path, relPath string) ([]Finding, error) {
	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)
	if !scannableExtensions[ext] && base != "SKILL.md" {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() > maxFileSize {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	var findings []Finding
	seen := make(map[string]bool) // "pid:linenum" dedup

	for _, tp := range threatPatterns {
		re := tp.compiled
		for i, line := range lines {
			key := fmt.Sprintf("%s:%d", tp.id, i+1)
			if seen[key] {
				continue
			}
			if re.MatchString(line) {
				seen[key] = true
				match := strings.TrimSpace(line)
				if len(match) > 120 {
					match = match[:117] + "..."
				}
				findings = append(findings, Finding{
					PatternID:   tp.id,
					Severity:    tp.severity,
					Category:    tp.category,
					File:        relPath,
					Line:        i + 1,
					Match:       match,
					Description: tp.description,
				})
			}
		}
	}

	// Invisible Unicode detection.
	for i, line := range lines {
		for _, r := range line {
			if isInvisibleUnicode(r) {
				findings = append(findings, Finding{
					PatternID:   "invisible_unicode",
					Severity:    SeverityHigh,
					Category:    CategoryInjection,
					File:        relPath,
					Line:        i + 1,
					Match:       fmt.Sprintf("U+%04X", r),
					Description: "invisible Unicode character (possible text hiding/injection)",
				})
				break // one finding per line
			}
		}
	}

	return findings, nil
}

// structureCheck performs file-count and symlink checks on the skill directory.
func structureCheck(skillDir string) []Finding {
	var findings []Finding
	count := 0
	_ = filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		count++
		// Detect symlinks pointing outside skillDir.
		if d.Type()&fs.ModeSymlink != 0 {
			target, lerr := os.Readlink(path)
			if lerr == nil && filepath.IsAbs(target) {
				rel, _ := filepath.Rel(skillDir, path)
				findings = append(findings, Finding{
					PatternID:   "symlink_escape",
					Severity:    SeverityHigh,
					Category:    CategoryStructure,
					File:        rel,
					Line:        0,
					Description: "symlink points to absolute path outside skill dir",
				})
			}
		}
		return nil
	})
	if count > maxFiles {
		findings = append(findings, Finding{
			PatternID:   "too_many_files",
			Severity:    SeverityMedium,
			Category:    CategoryStructure,
			Description: fmt.Sprintf("skill contains %d files (limit %d)", count, maxFiles),
		})
	}
	return findings
}

// determineVerdict maps finding severities to a verdict.
func determineVerdict(findings []Finding) string {
	hasCritical, hasHigh, hasMedium := false, false, false
	for _, f := range findings {
		switch f.Severity {
		case SeverityCritical:
			hasCritical = true
		case SeverityHigh:
			hasHigh = true
		case SeverityMedium:
			hasMedium = true
		}
	}
	if hasCritical || hasHigh {
		return VerdictBlocked
	}
	if hasMedium {
		return VerdictCaution
	}
	return VerdictClean
}

// buildSummary produces the one-line summary for a scan result.
func buildSummary(skillName, verdict string, findings []Finding) string {
	if len(findings) == 0 {
		return fmt.Sprintf("%s: %s", skillName, verdict)
	}
	counts := map[string]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}
	// Collect non-zero severity counts in priority order.
	var parts []string
	for _, sev := range []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow} {
		if n := counts[sev]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, sev))
		}
	}
	return fmt.Sprintf("%s: %s — %s finding(s)", skillName, verdict, strings.Join(parts, ", "))
}

// isInvisibleUnicode reports whether r is a known invisible/homoglyph
// Unicode character used to hide instructions in text.
func isInvisibleUnicode(r rune) bool {
	switch r {
	case 0x200B, // zero-width space
		0x200C, // zero-width non-joiner
		0x200D, // zero-width joiner
		0x2060, // word joiner
		0xFEFF, // BOM / zero-width no-break space
		0x00AD, // soft hyphen
		0x034F, // combining grapheme joiner
		0x115F, // hangul choseong filler
		0x1160, // hangul jungseong filler
		0x17B4, 0x17B5, // khmer vowel inherent AQ/AA
		0x180B, 0x180C, 0x180D, // mongolian free variation selectors
		0x2028, 0x2029, // line/paragraph separator
		0x202A, 0x202B, 0x202C, 0x202D, 0x202E, // bidi override chars
		0x2061, 0x2062, 0x2063, 0x2064: // invisible math operators
		return true
	}
	// Variation selectors U+FE00..U+FE0F
	if r >= 0xFE00 && r <= 0xFE0F {
		return true
	}
	// Tag characters U+E0000..U+E007F
	if r >= 0xE0000 && r <= 0xE007F {
		return true
	}
	// Remaining Unicode format characters (Cf) are invisible by definition.
	return unicode.Is(unicode.Cf, r)
}

// FormatReport produces a human-readable scan report.
func FormatReport(result ScanResult) string {
	var sb strings.Builder
	sb.WriteString("Skill security scan: ")
	sb.WriteString(result.Summary)
	sb.WriteString("\n")
	if len(result.Findings) == 0 {
		return sb.String()
	}
	// Group by category.
	byCategory := map[string][]Finding{}
	for _, f := range result.Findings {
		byCategory[f.Category] = append(byCategory[f.Category], f)
	}
	cats := make([]string, 0, len(byCategory))
	for c := range byCategory {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, cat := range cats {
		sb.WriteString(fmt.Sprintf("\n[%s]\n", cat))
		for _, f := range byCategory[cat] {
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			sb.WriteString(fmt.Sprintf("  %-8s %-24s %s\n", f.Severity, loc, f.Description))
			if f.Match != "" {
				sb.WriteString(fmt.Sprintf("           %s\n", f.Match))
			}
		}
	}
	return sb.String()
}
