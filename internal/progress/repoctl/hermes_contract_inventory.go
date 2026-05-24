package repoctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/fidelity"
)

const (
	hermesContractInventoryJSONRel = "webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json"
	hermesContractInventoryMDRel   = "webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md"
)

type HermesContractInventoryOptions struct {
	Root             string
	ProgressPath     string
	HermesSrc        string
	SourcePairsPath  string
	JSONPath         string
	MarkdownPath     string
	CurrentHermesSHA string
	Strict           bool
	Now              func() time.Time
}

type HermesContractInventoryResult struct {
	Report       fidelity.Report
	JSONPath     string
	MarkdownPath string
}

func WriteHermesContractInventory(opts HermesContractInventoryOptions) (HermesContractInventoryResult, error) {
	report, err := BuildHermesContractInventory(opts)
	if err != nil {
		return HermesContractInventoryResult{}, err
	}
	root := hermesContractInventoryRoot(opts.Root)
	jsonPath := opts.JSONPath
	if jsonPath == "" {
		jsonPath = filepath.Join(root, hermesContractInventoryJSONRel)
	}
	mdPath := opts.MarkdownPath
	if mdPath == "" {
		mdPath = filepath.Join(root, hermesContractInventoryMDRel)
	}
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return HermesContractInventoryResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(mdPath), 0o755); err != nil {
		return HermesContractInventoryResult{}, err
	}

	report = preserveHermesContractInventoryTimestamp(report, jsonPath, mdPath)

	jsonReport, err := marshalHermesContractInventoryJSON(report)
	if err != nil {
		return HermesContractInventoryResult{}, err
	}
	if err := os.WriteFile(jsonPath, jsonReport, 0o644); err != nil {
		return HermesContractInventoryResult{}, err
	}
	if err := os.WriteFile(mdPath, []byte(RenderHermesContractInventoryMarkdown(report)), 0o644); err != nil {
		return HermesContractInventoryResult{}, err
	}
	return HermesContractInventoryResult{Report: report, JSONPath: jsonPath, MarkdownPath: mdPath}, nil
}

func BuildHermesContractInventory(opts HermesContractInventoryOptions) (fidelity.Report, error) {
	root := hermesContractInventoryRoot(opts.Root)
	return fidelity.GenerateHermesReport(context.Background(), fidelity.Options{
		RepoRoot:        root,
		ProgressPath:    opts.ProgressPath,
		HermesPath:      opts.HermesSrc,
		SourcePairsPath: opts.SourcePairsPath,
		HermesSHA:       opts.CurrentHermesSHA,
		Strict:          opts.Strict,
		Now:             opts.Now,
	})
}

func preserveHermesContractInventoryTimestamp(report fidelity.Report, jsonPath, mdPath string) fidelity.Report {
	existingJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		return report
	}
	existingMD, err := os.ReadFile(mdPath)
	if err != nil {
		return report
	}
	var existing fidelity.Report
	if err := json.Unmarshal(existingJSON, &existing); err != nil {
		return report
	}
	if strings.TrimSpace(existing.GeneratedAt) == "" {
		return report
	}
	candidate := report
	candidate.GeneratedAt = existing.GeneratedAt
	candidateJSON, err := marshalHermesContractInventoryJSON(candidate)
	if err != nil {
		return report
	}
	candidateMD := []byte(RenderHermesContractInventoryMarkdown(candidate))
	if bytes.Equal(existingJSON, candidateJSON) && bytes.Equal(existingMD, candidateMD) {
		return candidate
	}
	return report
}

func marshalHermesContractInventoryJSON(report fidelity.Report) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RenderHermesContractInventoryMarkdown(report fidelity.Report) string {
	var b strings.Builder
	b.WriteString("# Hermes Contract Inventory\n\n")
	fmt.Fprintf(&b, "- Hermes SHA: `%s`\n", report.HermesSHA)
	fmt.Fprintf(&b, "- Generated: `%s`\n", report.GeneratedAt)
	fmt.Fprintf(&b, "- Source pairs: `%s`", report.SourcePairsState)
	if report.SourcePairsSHA != "" {
		fmt.Fprintf(&b, " (`%s`)", report.SourcePairsSHA)
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Report mode: `%s`\n", reportMode(report.Strict))
	fmt.Fprintf(&b, "- Progress source: `%s`\n", report.ProgressPath)
	b.WriteString("- Backlog policy: `progress.json` remains the only backlog; this report classifies evidence and does not create work rows.\n")
	b.WriteString("- Claim boundary: Gormes may claim all Hermes features and architecture are paired only when every current-SHA inventory gap is classified as `covered`, `partial`, `planned`, `excluded`, or `owned_divergence`; strict mode additionally requires every critical surface to be `covered`, `excluded`, or `owned_divergence` and every upstream source/doc/test file to be mapped or explicitly excluded.\n\n")

	b.WriteString("## Headline Counts\n\n")
	fmt.Fprintf(&b, "- Source files: `%d`\n", report.Inventory.SourceFiles)
	fmt.Fprintf(&b, "- Docs files: `%d`\n", report.Inventory.DocsFiles)
	fmt.Fprintf(&b, "- Test files: `%d`\n", report.Inventory.TestFiles)
	fmt.Fprintf(&b, "- Unmapped upstream source files: `%d`\n", len(report.UnmappedUpstream.SourceFiles))
	fmt.Fprintf(&b, "- Unmapped upstream docs files: `%d`\n", len(report.UnmappedUpstream.DocsFiles))
	fmt.Fprintf(&b, "- Unmapped upstream test files: `%d`\n", len(report.UnmappedUpstream.TestFiles))
	fmt.Fprintf(&b, "- Release checkpoints: `%d`\n", len(report.ReleaseCheckpoints))
	fmt.Fprintf(&b, "- Critical surfaces: `%d`\n", report.Summary.Critical)
	fmt.Fprintf(&b, "- Surface strict failures: `%d`\n", report.Summary.SurfaceStrictFailures)
	fmt.Fprintf(&b, "- Strict failures: `%d`\n", report.Summary.StrictFailures)
	for _, status := range sortedStatusKeys(report.Summary.ByStatus) {
		fmt.Fprintf(&b, "- `%s`: `%d`\n", status, report.Summary.ByStatus[status])
	}
	b.WriteString("\n## Critical Surface Blockers\n\n")
	blockers := blockerSurfaces(report)
	if len(blockers) == 0 {
		if report.Summary.UnmappedUpstreamFiles > 0 {
			fmt.Fprintf(&b, "No critical surface blockers in the current classification. `%d` unmapped upstream source/doc/test files still block strict mode.\n\n", report.Summary.UnmappedUpstreamFiles)
		} else {
			b.WriteString("No critical surface blockers in the current classification.\n\n")
		}
	} else {
		for _, surface := range blockers {
			fmt.Fprintf(&b, "- `%s` (`%s`): %s\n", surface.ID, surface.Status, surface.Reason)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Release Checkpoints\n\n")
	b.WriteString("| Checkpoint | Present | Path |\n")
	b.WriteString("|---|---|---|\n")
	for _, checkpoint := range report.ReleaseCheckpoints {
		fmt.Fprintf(&b, "| %s | `%t` | `%s` |\n", escapeMarkdownCell(checkpoint.Label), checkpoint.Present, escapeMarkdownCell(checkpoint.Path))
	}
	b.WriteString("\n## Per-Module Gap Summary\n\n")
	b.WriteString("| Module | Surfaces | Covered | Partial | Planned | Missing | Blocker severity |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---|\n")
	for _, row := range moduleGapRows(report.Surfaces) {
		fmt.Fprintf(&b, "| `%s` | %s | `%d` | `%d` | `%d` | `%d` | `%s` |\n",
			row.Module,
			markdownCodeList(row.Surfaces),
			row.Covered,
			row.Partial,
			row.Planned,
			row.Missing,
			row.Severity,
		)
	}
	b.WriteString("\n## Continuity Categories\n\n")
	b.WriteString("| Category | Status | Severity | Surfaces | Evidence | Reason |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, category := range report.ContinuityCategories {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %s | %s | %s |\n",
			category.ID,
			category.Status,
			category.GapSeverity,
			markdownCodeList(category.SurfaceIDs),
			markdownCodeList(limitStrings(category.EvidenceRefs, 4)),
			escapeMarkdownCell(category.Reason),
		)
	}
	b.WriteString("\n## Unmapped Upstream Evidence\n\n")
	b.WriteString("Unmapped upstream files are strict-mode blockers until each file is joined to a progress row, a source-pair entry, a surface classification, or an explicit exclusion.\n\n")
	b.WriteString("| Family | Count | Examples |\n")
	b.WriteString("|---|---:|---|\n")
	fmt.Fprintf(&b, "| Source | `%d` | %s |\n", len(report.UnmappedUpstream.SourceFiles), markdownCodeList(limitStrings(report.UnmappedUpstream.SourceFiles, 10)))
	fmt.Fprintf(&b, "| Docs | `%d` | %s |\n", len(report.UnmappedUpstream.DocsFiles), markdownCodeList(limitStrings(report.UnmappedUpstream.DocsFiles, 10)))
	fmt.Fprintf(&b, "| Tests | `%d` | %s |\n", len(report.UnmappedUpstream.TestFiles), markdownCodeList(limitStrings(report.UnmappedUpstream.TestFiles, 10)))
	b.WriteString("\n")

	b.WriteString("## Unmapped Test Suite Classification\n\n")
	if len(report.UnmappedUpstream.TestSuites) == 0 {
		b.WriteString("No unmapped upstream test suites remain.\n\n")
	} else {
		b.WriteString("| Suite | Count | Source under test | Progress rows | Examples |\n")
		b.WriteString("|---|---:|---|---|---|\n")
		for _, suite := range report.UnmappedUpstream.TestSuites {
			fmt.Fprintf(&b, "| `%s` | `%d` | `%s` | %s | %s |\n",
				escapeMarkdownCell(suite.Suite),
				suite.Count,
				escapeMarkdownCell(suite.SourcePrefix),
				markdownCodeList(suite.ProgressRows),
				markdownCodeList(limitStrings(suite.Examples, 5)),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Plugin Catalog Classification\n\n")
	if len(report.PluginCatalog) == 0 {
		b.WriteString("No Hermes plugin catalog families were detected.\n\n")
	} else {
		b.WriteString("| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |\n")
		b.WriteString("|---|---|---:|---|---|---|---|\n")
		for _, family := range report.PluginCatalog {
			fmt.Fprintf(&b, "| `%s` | `%s` | `%d` | %s | %s | %s | %s |\n",
				escapeMarkdownCell(family.ID),
				family.Status,
				family.Count,
				markdownCodeList(progressRowNames(family.ProgressRows)),
				markdownCodeList(sourcePairFiles(family.SourcePairs)),
				markdownCodeList(limitStrings(family.Examples, 5)),
				escapeMarkdownCell(family.Reason),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Skill Catalog Classification\n\n")
	if len(report.SkillCatalog) == 0 {
		b.WriteString("No Hermes skill catalog families were detected.\n\n")
	} else {
		b.WriteString("| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |\n")
		b.WriteString("|---|---|---:|---|---|---|---|\n")
		for _, family := range report.SkillCatalog {
			fmt.Fprintf(&b, "| `%s` | `%s` | `%d` | %s | %s | %s | %s |\n",
				escapeMarkdownCell(family.ID),
				family.Status,
				family.Count,
				markdownCodeList(progressRowNames(family.ProgressRows)),
				markdownCodeList(sourcePairFiles(family.SourcePairs)),
				markdownCodeList(limitStrings(family.Examples, 5)),
				escapeMarkdownCell(family.Reason),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Gateway Platform Classification\n\n")
	if len(report.GatewayPlatformCatalog) == 0 {
		b.WriteString("No Hermes gateway platform families were detected.\n\n")
	} else {
		b.WriteString("| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |\n")
		b.WriteString("|---|---|---:|---|---|---|---|\n")
		for _, family := range report.GatewayPlatformCatalog {
			fmt.Fprintf(&b, "| `%s` | `%s` | `%d` | %s | %s | %s | %s |\n",
				escapeMarkdownCell(family.ID),
				family.Status,
				family.Count,
				markdownCodeList(progressRowNames(family.ProgressRows)),
				markdownCodeList(sourcePairFiles(family.SourcePairs)),
				markdownCodeList(limitStrings(family.Examples, 5)),
				escapeMarkdownCell(family.Reason),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Web Dashboard Classification\n\n")
	if len(report.WebDashboardCatalog) == 0 {
		b.WriteString("No Hermes web dashboard families were detected.\n\n")
	} else {
		b.WriteString("| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |\n")
		b.WriteString("|---|---|---:|---|---|---|---|\n")
		for _, family := range report.WebDashboardCatalog {
			fmt.Fprintf(&b, "| `%s` | `%s` | `%d` | %s | %s | %s | %s |\n",
				escapeMarkdownCell(family.ID),
				family.Status,
				family.Count,
				markdownCodeList(progressRowNames(family.ProgressRows)),
				markdownCodeList(sourcePairFiles(family.SourcePairs)),
				markdownCodeList(limitStrings(family.Examples, 5)),
				escapeMarkdownCell(family.Reason),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Candidate Inventory\n\n")
	b.WriteString("| Candidate family | Count | Examples |\n")
	b.WriteString("|---|---:|---|\n")
	candidateRows := []struct {
		name   string
		values []string
	}{
		{"CLI", report.Candidates.CLI},
		{"Tools", report.Candidates.Tools},
		{"Providers", report.Candidates.Providers},
		{"Channels", report.Candidates.Channels},
		{"Sessions", report.Candidates.Sessions},
		{"Memory", report.Candidates.Memory},
		{"Skills", report.Candidates.Skills},
		{"Learning loop", report.Candidates.LearningLoop},
	}
	for _, row := range candidateRows {
		fmt.Fprintf(&b, "| %s | `%d` | %s |\n", row.name, len(row.values), markdownCodeList(limitStrings(row.values, 5)))
	}
	b.WriteString("\n## Surface Classification\n\n")
	b.WriteString("| Surface | Status | Severity | Confidence | Progress rows | Source pairs | Reason |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, surface := range report.Surfaces {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | %s | %s | %s |\n",
			surface.ID,
			surface.Status,
			surface.GapSeverity,
			surface.Confidence,
			markdownCodeList(progressRowNames(surface.ProgressRows)),
			markdownCodeList(sourcePairFiles(surface.SourcePairs)),
			escapeMarkdownCell(surface.Reason),
		)
	}
	return b.String()
}

func hermesContractInventoryRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return "."
	}
	return root
}

func reportMode(strict bool) string {
	if strict {
		return "strict"
	}
	return "report-only"
}

func blockerSurfaces(report fidelity.Report) []fidelity.SurfaceReport {
	var blockers []fidelity.SurfaceReport
	for _, surface := range report.Surfaces {
		if surface.Critical && surface.GapSeverity == "blocker" {
			blockers = append(blockers, surface)
		}
	}
	return blockers
}

type moduleGapRow struct {
	Module   string
	Surfaces []string
	Covered  int
	Partial  int
	Planned  int
	Missing  int
	Severity string
}

func moduleGapRows(surfaces []fidelity.SurfaceReport) []moduleGapRow {
	byModule := map[string]*moduleGapRow{}
	for _, surface := range surfaces {
		module := strings.TrimSpace(surface.Category)
		if module == "" {
			module = "uncategorized"
		}
		row := byModule[module]
		if row == nil {
			row = &moduleGapRow{Module: module, Severity: "none"}
			byModule[module] = row
		}
		row.Surfaces = append(row.Surfaces, surface.ID)
		switch surface.Status {
		case fidelity.StatusCovered, fidelity.StatusExcluded, fidelity.StatusOwnedDivergence:
			row.Covered++
		case fidelity.StatusPartial:
			row.Partial++
		case fidelity.StatusPlanned:
			row.Planned++
		default:
			row.Missing++
		}
		row.Severity = worstSeverity(row.Severity, surface.GapSeverity)
	}
	rows := make([]moduleGapRow, 0, len(byModule))
	for _, row := range byModule {
		row.Surfaces = limitStrings(row.Surfaces, 6)
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Module < rows[j].Module
	})
	return rows
}

func worstSeverity(a, b string) string {
	if severityRank(b) > severityRank(a) {
		return b
	}
	return a
}

func severityRank(severity string) int {
	switch severity {
	case "blocker":
		return 3
	case "warning":
		return 2
	case "none":
		return 1
	default:
		return 0
	}
}

func sortedStatusKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key, count := range counts {
		if count > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func progressRowNames(rows []fidelity.ProgressRowEvidence) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return limitStrings(names, 4)
}

func sourcePairFiles(pairs []fidelity.SourcePairEvidence) []string {
	files := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		files = append(files, pair.HermesFile)
	}
	return limitStrings(files, 4)
}

func limitStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	out := append([]string(nil), values[:limit]...)
	out = append(out, fmt.Sprintf("+%d more", len(values)-limit))
	return out
}

func markdownCodeList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, "`"+escapeMarkdownCell(value)+"`")
	}
	return strings.Join(parts, ", ")
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
