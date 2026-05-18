package fidelity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

const SchemaVersion = "hermes-fidelity/v1"

type Status string

const (
	StatusCovered         Status = "covered"
	StatusPartial         Status = "partial"
	StatusPlanned         Status = "planned"
	StatusMissing         Status = "missing"
	StatusVague           Status = "vague"
	StatusOwnedDivergence Status = "owned_divergence"
	StatusExcluded        Status = "excluded"
	StatusBlocked         Status = "blocked"
	StatusStaleUpstream   Status = "stale_upstream"
)

type Options struct {
	RepoRoot        string
	ProgressPath    string
	HermesPath      string
	SourcePairsPath string
	HermesSHA       string
	Strict          bool
	Now             func() time.Time
	GitRevision     func(context.Context, string) (string, error)
}

type Report struct {
	SchemaVersion    string          `json:"schema_version"`
	GeneratedAt      string          `json:"generated_at"`
	HermesSHA        string          `json:"hermes_sha,omitempty"`
	HermesPath       string          `json:"hermes_path,omitempty"`
	ProgressPath     string          `json:"progress_path,omitempty"`
	SourcePairsPath  string          `json:"source_pairs_path,omitempty"`
	SourcePairsSHA   string          `json:"source_pairs_sha,omitempty"`
	SourcePairsState string          `json:"source_pairs_state,omitempty"`
	Strict           bool            `json:"strict"`
	OK               bool            `json:"ok"`
	Inventory        InventoryCounts `json:"inventory"`
	Candidates       CandidateInventory `json:"candidates"`
	Summary          Summary         `json:"summary"`
	UnmappedSurfaces []string        `json:"unmapped_surfaces,omitempty"`
	Surfaces         []SurfaceReport `json:"surfaces"`
}

type InventoryCounts struct {
	SourceFiles int `json:"source_files"`
	DocsFiles   int `json:"docs_files"`
	TestFiles   int `json:"test_files"`
}

type CandidateInventory struct {
	CLI          []string `json:"cli"`
	Tools        []string `json:"tools"`
	Providers    []string `json:"providers"`
	Channels     []string `json:"channels"`
	Sessions     []string `json:"sessions"`
	Memory       []string `json:"memory"`
	Skills       []string `json:"skills"`
	LearningLoop []string `json:"learning_loop"`
}

type Summary struct {
	Total          int            `json:"total"`
	Critical       int            `json:"critical"`
	StrictFailures int            `json:"strict_failures"`
	ByStatus       map[string]int `json:"by_status"`
}

type SurfaceReport struct {
	ID                  string                `json:"id"`
	Title               string                `json:"title"`
	Category            string                `json:"category"`
	Priority            string                `json:"priority"`
	Critical            bool                  `json:"critical"`
	Status              Status                `json:"status"`
	Reason              string                `json:"reason"`
	Confidence          string                `json:"confidence"`
	GapSeverity         string                `json:"gap_severity"`
	UpstreamRefs        []string              `json:"upstream_refs"`
	MissingUpstreamRefs []string              `json:"missing_upstream_refs,omitempty"`
	GormesRefs          []string              `json:"gormes_refs,omitempty"`
	SourcePairs         []SourcePairEvidence  `json:"source_pairs,omitempty"`
	ProgressRows        []ProgressRowEvidence `json:"progress_rows,omitempty"`
}

type SourcePairEvidence struct {
	HermesFile           string   `json:"hermes_file"`
	Status               string   `json:"status"`
	Contract             string   `json:"contract,omitempty"`
	GormesTargets        []string `json:"gormes_targets,omitempty"`
	Tests                []string `json:"tests,omitempty"`
	ProgressRows         []string `json:"progress_rows,omitempty"`
	UpstreamTests        []string `json:"upstream_tests,omitempty"`
	LastCheckedHermesSHA string   `json:"last_checked_hermes_sha,omitempty"`
	Stale                bool     `json:"stale,omitempty"`
}

type ProgressRowEvidence struct {
	PhaseID              string   `json:"phase_id"`
	SubphaseID           string   `json:"subphase_id"`
	Name                 string   `json:"name"`
	Priority             string   `json:"priority,omitempty"`
	Status               string   `json:"status"`
	ContractStatus       string   `json:"contract_status,omitempty"`
	Module               string   `json:"module,omitempty"`
	ProvenanceOriginType string   `json:"provenance_origin_type,omitempty"`
	SourceRefs           []string `json:"source_refs,omitempty"`
	TestCommands         []string `json:"test_commands,omitempty"`
	NoTestRequired       string   `json:"no_test_required,omitempty"`
	MatchReasons         []string `json:"match_reasons,omitempty"`
}

type surfaceDefinition struct {
	ID           string
	Title        string
	Category     string
	Priority     string
	Critical     bool
	UpstreamRefs []string
	Keywords     []string
}

type sourcePairsManifest struct {
	SchemaVersion string       `json:"schema_version"`
	HermesSHA     string       `json:"hermes_sha"`
	Pairs         []sourcePair `json:"pairs"`
}

type sourcePair struct {
	HermesFile           string   `json:"hermes_file"`
	GormesTargets        []string `json:"gormes_targets,omitempty"`
	Status               string   `json:"status"`
	Contract             string   `json:"contract"`
	Tests                []string `json:"tests,omitempty"`
	ProgressRows         []string `json:"progress_rows,omitempty"`
	UpstreamTests        []string `json:"upstream_tests,omitempty"`
	LastCheckedHermesSHA string   `json:"last_checked_hermes_sha"`
}

type progressRow struct {
	PhaseID    string
	SubphaseID string
	Item       progress.Item
	Text       string
}

func GenerateHermesReport(ctx context.Context, opts Options) (Report, error) {
	repoRoot, err := resolveRoot(opts.RepoRoot)
	if err != nil {
		return Report{}, err
	}
	progressPath := resolveInputPath(repoRoot, opts.ProgressPath, []string{
		"webpages/docs/content/building-gormes/architecture_plan/progress.json",
		"docs/content/building-gormes/architecture_plan/progress.json",
	})
	prog, err := progress.Load(progressPath)
	if err != nil {
		return Report{}, fmt.Errorf("fidelity: load progress: %w", err)
	}

	hermesPath := resolveHermesPath(repoRoot, opts.HermesPath)
	hermesSHA := strings.TrimSpace(opts.HermesSHA)
	if hermesSHA == "" && hermesPath != "" {
		revision := opts.GitRevision
		if revision == nil {
			revision = gitRevision
		}
		if got, err := revision(ctx, hermesPath); err == nil {
			hermesSHA = strings.TrimSpace(got)
		}
	}
	if hermesSHA == "" {
		hermesSHA = "unknown"
	}

	sourcePairsPath := resolveInputPath(repoRoot, opts.SourcePairsPath, []string{
		"webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json",
		"docs/content/building-gormes/architecture_plan/hermes-source-pairs.json",
	})
	sourcePairs, sourcePairsState, err := loadSourcePairs(sourcePairsPath, hermesSHA)
	if err != nil {
		return Report{}, err
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	report := Report{
		SchemaVersion:    SchemaVersion,
		GeneratedAt:      now().UTC().Format(time.RFC3339),
		HermesSHA:        hermesSHA,
		HermesPath:       displayPath(repoRoot, hermesPath),
		ProgressPath:     displayPath(repoRoot, progressPath),
		SourcePairsPath:  displayPath(repoRoot, sourcePairsPath),
		SourcePairsSHA:   sourcePairs.HermesSHA,
		SourcePairsState: sourcePairsState,
		Strict:           opts.Strict,
		Inventory:        countHermesInventory(hermesPath),
		Candidates:       buildCandidateInventory(hermesPath),
		Summary: Summary{
			ByStatus: map[string]int{},
		},
	}
	rows := flattenProgress(prog)
	for _, def := range defaultSurfaces() {
		surface := buildSurfaceReport(def, hermesPath, hermesSHA, sourcePairsState, sourcePairs.Pairs, rows)
		report.Surfaces = append(report.Surfaces, surface)
		report.Summary.Total++
		if surface.Critical {
			report.Summary.Critical++
		}
		report.Summary.ByStatus[string(surface.Status)]++
		if surface.Critical && !strictStatusPasses(surface.Status) {
			report.Summary.StrictFailures++
			report.UnmappedSurfaces = append(report.UnmappedSurfaces, surface.ID)
		}
	}
	report.OK = !opts.Strict || report.Summary.StrictFailures == 0
	return report, nil
}

func defaultSurfaces() []surfaceDefinition {
	return []surfaceDefinition{
		{
			ID:           "profiles",
			Title:        "Profiles And Identity Boundaries",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"hermes_cli/profiles.py"},
			Keywords:     []string{"profile", "profiles", "profile-scoped", "identity", "workspace", "peer"},
		},
		{
			ID:           "sessions",
			Title:        "Sessions, Recaps, And Context Continuity",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"run_agent.py", "gateway/session.py", "tools/session_search_tool.py", "hermes_cli/session_recap.py"},
			Keywords:     []string{"session", "sessions", "transcript", "recap", "summary", "context"},
		},
		{
			ID:           "goncho_memory",
			Title:        "Memory, Goncho, And Honcho Compatibility",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"tools/memory_tool.py", "agent/memory_manager.py", "plugins/memory/__init__.py"},
			Keywords:     []string{"memory", "goncho", "honcho", "recall", "remember", "knowledge graph"},
		},
		{
			ID:           "learning_loop",
			Title:        "Learning Loop And Candidate Updates",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"agent/curator.py", "hermes_cli/curator.py", "tools/skill_usage.py"},
			Keywords:     []string{"learning", "curator", "candidate", "feedback", "outcome", "skill update", "memory update"},
		},
		{
			ID:           "prompt_assembly",
			Title:        "Prompt Assembly And Insertion Ordering",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"agent/prompt_builder.py", "agent/skill_commands.py", "agent/skill_preprocessing.py", "agent/skill_utils.py"},
			Keywords:     []string{"prompt", "context", "insertion", "skill preprocessing", "budget", "assembly"},
		},
		{
			ID:           "provider_auth_setup",
			Title:        "Provider, Auth, And Setup",
			Category:     "operator",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"hermes_cli/auth_commands.py", "hermes_cli/providers.py", "hermes_cli/setup.py"},
			Keywords:     []string{"provider", "auth", "oauth", "credential", "setup", "model", "openrouter", "codex"},
		},
		{
			ID:           "gateway_channels",
			Title:        "Gateway And Channel Operation Sequence",
			Category:     "channels",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"gateway/run.py"},
			Keywords:     []string{"gateway", "channel", "telegram", "slack", "whatsapp", "discord", "outbound", "tool progress"},
		},
		{
			ID:           "tool_runtime",
			Title:        "Tool Runtime And Execution Safety",
			Category:     "runtime",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"tools/registry.py", "tools/file_tools.py", "tools/code_execution_tool.py"},
			Keywords:     []string{"tool", "tools", "registry", "execution", "permission", "sandbox", "file ops"},
		},
		{
			ID:           "mcp_acp",
			Title:        "MCP And ACP",
			Category:     "tools",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"hermes_cli/mcp_config.py", "tools/mcp_tool.py", "acp_adapter/entry.py", "acp_adapter/server.py"},
			Keywords:     []string{"mcp", "acp", "adapter", "server", "protocol"},
		},
		{
			ID:           "tui_cli",
			Title:        "TUI And CLI Surface",
			Category:     "operator",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"ui-tui/package.json", "cli.py"},
			Keywords:     []string{"tui", "cli", "command", "terminal", "ui-tui", "curses", "cobra"},
		},
	}
}

func buildSurfaceReport(def surfaceDefinition, hermesPath, hermesSHA, sourcePairsState string, pairs []sourcePair, rows []progressRow) SurfaceReport {
	surface := SurfaceReport{
		ID:           def.ID,
		Title:        def.Title,
		Category:     def.Category,
		Priority:     def.Priority,
		Critical:     def.Critical,
		UpstreamRefs: append([]string(nil), def.UpstreamRefs...),
		Confidence:   "low",
	}
	if hermesPath == "" {
		surface.Status = StatusBlocked
		surface.Reason = "Hermes checkout is unavailable; no live home directories were inspected."
		return surface
	}
	for _, ref := range def.UpstreamRefs {
		if _, err := os.Stat(filepath.Join(hermesPath, filepath.FromSlash(ref))); err != nil {
			surface.MissingUpstreamRefs = append(surface.MissingUpstreamRefs, ref)
		}
	}

	matchedNames := map[string]bool{}
	for _, pair := range pairs {
		if !surfaceMatchesPair(def, pair) {
			continue
		}
		ev := SourcePairEvidence{
			HermesFile:           pair.HermesFile,
			Status:               pair.Status,
			Contract:             pair.Contract,
			GormesTargets:        cleanStrings(pair.GormesTargets),
			Tests:                cleanStrings(pair.Tests),
			ProgressRows:         cleanStrings(pair.ProgressRows),
			UpstreamTests:        cleanStrings(pair.UpstreamTests),
			LastCheckedHermesSHA: pair.LastCheckedHermesSHA,
			Stale:                pairStale(hermesSHA, pair.LastCheckedHermesSHA) || sourcePairsState == "stale",
		}
		surface.SourcePairs = append(surface.SourcePairs, ev)
		surface.GormesRefs = append(surface.GormesRefs, ev.GormesTargets...)
		for _, name := range pair.ProgressRows {
			matchedNames[strings.ToLower(strings.TrimSpace(name))] = true
		}
	}

	for _, row := range rows {
		reasons := rowMatchReasons(def, row, matchedNames)
		if len(reasons) == 0 {
			continue
		}
		ev := progressRowEvidence(row, reasons)
		surface.ProgressRows = append(surface.ProgressRows, ev)
		for _, ref := range ev.SourceRefs {
			if !isUpstreamRef(ref) {
				surface.GormesRefs = append(surface.GormesRefs, ref)
			}
		}
	}
	surface.GormesRefs = uniqueSorted(surface.GormesRefs)
	sort.Slice(surface.SourcePairs, func(i, j int) bool {
		return surface.SourcePairs[i].HermesFile < surface.SourcePairs[j].HermesFile
	})
	sort.Slice(surface.ProgressRows, func(i, j int) bool {
		return surface.ProgressRows[i].Name < surface.ProgressRows[j].Name
	})

	surface.Status, surface.Reason, surface.Confidence = deriveStatus(surface)
	surface.GapSeverity = gapSeverity(surface.Status, surface.Critical)
	return surface
}

func deriveStatus(surface SurfaceReport) (Status, string, string) {
	if len(surface.MissingUpstreamRefs) > 0 {
		return StatusStaleUpstream, "One or more recorded upstream refs are missing at the selected Hermes checkout.", "high"
	}
	for _, pair := range surface.SourcePairs {
		if pair.Stale {
			return StatusStaleUpstream, "Source-pair evidence is stale for the selected Hermes SHA.", "high"
		}
	}
	if hasPairStatus(surface.SourcePairs, "excluded") {
		return StatusExcluded, "Source-pair manifest explicitly excludes this surface.", "high"
	}
	if hasPairStatus(surface.SourcePairs, "owned") && hasValidatedProgress(surface.ProgressRows) {
		return StatusOwnedDivergence, "Validated Gormes-owned divergence is explicitly recorded.", "high"
	}
	if hasPairStatus(surface.SourcePairs, "covered") && hasValidatedProgress(surface.ProgressRows) {
		return StatusCovered, "Covered source-pair evidence joins to a validated complete progress row with test evidence.", "high"
	}
	if hasPairStatus(surface.SourcePairs, "partial") || hasCompleteProgress(surface.ProgressRows) {
		return StatusPartial, "Some source-pair or complete-row evidence exists, but the surface is not strictly covered.", "medium"
	}
	if hasPairStatus(surface.SourcePairs, "planned") || hasPlannedProgress(surface.ProgressRows) {
		return StatusPlanned, "A planned progress row or source-pair entry exists, but coverage is not validated.", "medium"
	}
	if hasVagueProgress(surface.ProgressRows) {
		return StatusVague, "Only vague progress evidence matched this surface.", "low"
	}
	return StatusMissing, "No matching source-pair or progress-row evidence was found.", "low"
}

func hasPairStatus(pairs []SourcePairEvidence, status string) bool {
	for _, pair := range pairs {
		if strings.EqualFold(pair.Status, status) {
			return true
		}
	}
	return false
}

func hasValidatedProgress(rows []ProgressRowEvidence) bool {
	for _, row := range rows {
		if row.Status == string(progress.StatusComplete) &&
			row.ContractStatus == string(progress.ContractStatusValidated) &&
			len(row.SourceRefs) > 0 &&
			(len(row.TestCommands) > 0 || row.NoTestRequired != "") {
			return true
		}
	}
	return false
}

func hasCompleteProgress(rows []ProgressRowEvidence) bool {
	for _, row := range rows {
		if row.Status == string(progress.StatusComplete) || row.Status == string(progress.StatusInProgress) {
			return true
		}
	}
	return false
}

func hasPlannedProgress(rows []ProgressRowEvidence) bool {
	for _, row := range rows {
		if row.Status == string(progress.StatusPlanned) {
			return true
		}
	}
	return false
}

func hasVagueProgress(rows []ProgressRowEvidence) bool {
	for _, row := range rows {
		if row.ContractStatus == "" || row.ContractStatus == string(progress.ContractStatusMissing) || len(row.SourceRefs) == 0 {
			return true
		}
	}
	return false
}

func strictStatusPasses(status Status) bool {
	switch status {
	case StatusCovered, StatusExcluded, StatusOwnedDivergence:
		return true
	default:
		return false
	}
}

func gapSeverity(status Status, critical bool) string {
	if strictStatusPasses(status) {
		return "none"
	}
	if !critical {
		return "warning"
	}
	switch status {
	case StatusBlocked, StatusStaleUpstream, StatusMissing, StatusVague:
		return "blocker"
	default:
		return "warning"
	}
}

func surfaceMatchesPair(def surfaceDefinition, pair sourcePair) bool {
	if containsExact(def.UpstreamRefs, pair.HermesFile) {
		return true
	}
	text := strings.ToLower(pair.HermesFile + " " + pair.Contract + " " + strings.Join(pair.ProgressRows, " "))
	return containsKeyword(text, def.Keywords)
}

func rowMatchReasons(def surfaceDefinition, row progressRow, matchedNames map[string]bool) []string {
	var reasons []string
	name := strings.ToLower(strings.TrimSpace(row.Item.Name))
	if matchedNames[name] {
		reasons = append(reasons, "source_pair_progress_row")
	}
	for _, source := range row.Item.SourceRefs {
		normalized := normalizeUpstreamRef(source)
		if normalized != "" && containsExact(def.UpstreamRefs, normalized) {
			reasons = append(reasons, "source_ref:"+normalized)
		}
	}
	if containsKeyword(row.Text, def.Keywords) {
		reasons = append(reasons, "taxonomy_keyword")
	}
	return uniqueSorted(reasons)
}

func progressRowEvidence(row progressRow, reasons []string) ProgressRowEvidence {
	origin := ""
	if row.Item.Provenance != nil {
		origin = row.Item.Provenance.OriginType
	}
	return ProgressRowEvidence{
		PhaseID:              row.PhaseID,
		SubphaseID:           row.SubphaseID,
		Name:                 row.Item.Name,
		Priority:             row.Item.Priority,
		Status:               string(row.Item.Status),
		ContractStatus:       string(row.Item.ContractStatus),
		Module:               row.Item.Module,
		ProvenanceOriginType: origin,
		SourceRefs:           cleanStrings(row.Item.SourceRefs),
		TestCommands:         cleanStrings(row.Item.TestCommands),
		NoTestRequired:       strings.TrimSpace(row.Item.NoTestRequiredReason),
		MatchReasons:         reasons,
	}
}

func flattenProgress(prog *progress.Progress) []progressRow {
	if prog == nil {
		return nil
	}
	var rows []progressRow
	for _, phaseID := range sortedMapKeys(prog.Phases) {
		phase := prog.Phases[phaseID]
		for _, subphaseID := range sortedMapKeys(phase.Subphases) {
			subphase := phase.Subphases[subphaseID]
			for _, item := range subphase.Items {
				rows = append(rows, progressRow{
					PhaseID:    phaseID,
					SubphaseID: subphaseID,
					Item:       item,
					Text:       rowSearchText(phaseID, phase, subphaseID, subphase, item),
				})
			}
		}
	}
	return rows
}

func rowSearchText(phaseID string, phase progress.Phase, subphaseID string, subphase progress.Subphase, item progress.Item) string {
	parts := []string{
		phaseID,
		phase.Name,
		subphaseID,
		subphase.Name,
		item.Name,
		item.Priority,
		string(item.Status),
		item.Contract,
		string(item.ContractStatus),
		item.Module,
		string(item.ExecutionOwner),
		item.DegradedMode,
		item.Fixture,
		item.Note,
		item.NoTestRequiredReason,
		strings.Join(item.TrustClass, " "),
		strings.Join(item.SourceRefs, " "),
		strings.Join(item.ReadyWhen, " "),
		strings.Join(item.NotReadyWhen, " "),
		strings.Join(item.BlockedBy, " "),
		strings.Join(item.Acceptance, " "),
		strings.Join(item.WriteScope, " "),
		strings.Join(item.TestCommands, " "),
		strings.Join(item.DoneSignal, " "),
	}
	if item.Provenance != nil {
		parts = append(parts, item.Provenance.OriginType, item.Provenance.UpstreamRef, strings.Join(item.Provenance.UpstreamRefs, " "), item.Provenance.Note)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func loadSourcePairs(path, hermesSHA string) (sourcePairsManifest, string, error) {
	if strings.TrimSpace(path) == "" {
		return sourcePairsManifest{}, "missing", nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sourcePairsManifest{}, "missing", nil
		}
		return sourcePairsManifest{}, "", fmt.Errorf("fidelity: read source pairs %s: %w", path, err)
	}
	var manifest sourcePairsManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return sourcePairsManifest{}, "", fmt.Errorf("fidelity: parse source pairs %s: %w", path, err)
	}
	state := "current"
	if pairStale(hermesSHA, manifest.HermesSHA) {
		state = "stale"
	}
	return manifest, state, nil
}

func pairStale(current, recorded string) bool {
	current = strings.TrimSpace(current)
	recorded = strings.TrimSpace(recorded)
	if current == "" || current == "unknown" || recorded == "" {
		return false
	}
	return !strings.HasPrefix(current, recorded) && !strings.HasPrefix(recorded, current)
}

func resolveRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("fidelity: resolve repo root: %w", err)
	}
	return abs, nil
}

func resolveInputPath(root, explicit string, defaults []string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return filepath.Clean(explicit)
		}
		return filepath.Join(root, filepath.FromSlash(explicit))
	}
	for _, rel := range defaults {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if len(defaults) == 0 {
		return ""
	}
	return filepath.Join(root, filepath.FromSlash(defaults[0]))
}

func resolveHermesPath(root, explicit string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return filepath.Clean(explicit)
		}
		return filepath.Join(root, filepath.FromSlash(explicit))
	}
	for _, candidate := range []string{
		filepath.Join(root, "hermes-agent"),
		filepath.Join(root, "..", "hermes-agent"),
		filepath.Join(root, "references", "hermes-agent"),
	} {
		if _, err := os.Stat(filepath.Join(candidate, "hermes_cli", "main.py")); err == nil {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func gitRevision(ctx context.Context, dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", nil
	}
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func countHermesInventory(root string) InventoryCounts {
	var counts InventoryCounts
	if root == "" {
		return counts
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		slash := filepath.ToSlash(rel)
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".py", ".ts", ".tsx", ".js", ".jsx":
			counts.SourceFiles++
		case ".md", ".mdx":
			counts.DocsFiles++
		}
		base := filepath.Base(path)
		if strings.Contains(slash, "/tests/") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") || strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") {
			counts.TestFiles++
		}
		return nil
	})
	return counts
}

func buildCandidateInventory(root string) CandidateInventory {
	var candidates CandidateInventory
	if root == "" {
		return candidates
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if !candidateSourceFile(slash) {
			return nil
		}
		lower := strings.ToLower(slash)
		if strings.HasPrefix(lower, "hermes_cli/") || lower == "cli.py" {
			candidates.CLI = append(candidates.CLI, slash)
		}
		if strings.HasPrefix(lower, "tools/") {
			candidates.Tools = append(candidates.Tools, slash)
		}
		if strings.HasPrefix(lower, "providers/") || strings.Contains(lower, "provider") {
			candidates.Providers = append(candidates.Providers, slash)
		}
		if strings.HasPrefix(lower, "gateway/") || strings.Contains(lower, "slack") || strings.Contains(lower, "telegram") || strings.Contains(lower, "whatsapp") || strings.Contains(lower, "discord") {
			candidates.Channels = append(candidates.Channels, slash)
		}
		if strings.Contains(lower, "session") || strings.Contains(lower, "recap") {
			candidates.Sessions = append(candidates.Sessions, slash)
		}
		if strings.Contains(lower, "memory") || strings.Contains(lower, "honcho") {
			candidates.Memory = append(candidates.Memory, slash)
		}
		if strings.Contains(lower, "skill") {
			candidates.Skills = append(candidates.Skills, slash)
		}
		if strings.Contains(lower, "curator") || strings.Contains(lower, "skill_usage") || strings.Contains(lower, "learning") {
			candidates.LearningLoop = append(candidates.LearningLoop, slash)
		}
		return nil
	})
	candidates.CLI = uniqueSorted(candidates.CLI)
	candidates.Tools = uniqueSorted(candidates.Tools)
	candidates.Providers = uniqueSorted(candidates.Providers)
	candidates.Channels = uniqueSorted(candidates.Channels)
	candidates.Sessions = uniqueSorted(candidates.Sessions)
	candidates.Memory = uniqueSorted(candidates.Memory)
	candidates.Skills = uniqueSorted(candidates.Skills)
	candidates.LearningLoop = uniqueSorted(candidates.LearningLoop)
	return candidates
}

func candidateSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py", ".ts", ".tsx", ".js", ".jsx", ".md":
		return true
	default:
		return false
	}
}

func displayPath(root, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func normalizeUpstreamRef(ref string) string {
	ref = filepath.ToSlash(strings.TrimSpace(ref))
	ref = strings.TrimPrefix(ref, "./")
	for _, prefix := range []string{"hermes-agent/", "references/hermes-agent/"} {
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix)
		}
	}
	return ""
}

func isUpstreamRef(ref string) bool {
	return normalizeUpstreamRef(ref) != ""
}

func containsExact(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func containsKeyword(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func cleanStrings(values []string) []string {
	var cleaned []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, filepath.ToSlash(value))
		}
	}
	return uniqueSorted(cleaned)
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
