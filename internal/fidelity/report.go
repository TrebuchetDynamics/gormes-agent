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
	SchemaVersion          string                    `json:"schema_version"`
	GeneratedAt            string                    `json:"generated_at"`
	HermesSHA              string                    `json:"hermes_sha,omitempty"`
	HermesPath             string                    `json:"hermes_path,omitempty"`
	ProgressPath           string                    `json:"progress_path,omitempty"`
	SourcePairsPath        string                    `json:"source_pairs_path,omitempty"`
	SourcePairsSHA         string                    `json:"source_pairs_sha,omitempty"`
	SourcePairsState       string                    `json:"source_pairs_state,omitempty"`
	Strict                 bool                      `json:"strict"`
	OK                     bool                      `json:"ok"`
	Inventory              InventoryCounts           `json:"inventory"`
	Candidates             CandidateInventory        `json:"candidates"`
	PluginCatalog          []CatalogFamilyReport     `json:"plugin_catalog,omitempty"`
	SkillCatalog           []CatalogFamilyReport     `json:"skill_catalog,omitempty"`
	GatewayPlatformCatalog []CatalogFamilyReport     `json:"gateway_platform_catalog,omitempty"`
	WebDashboardCatalog    []CatalogFamilyReport     `json:"web_dashboard_catalog,omitempty"`
	UnmappedUpstream       UpstreamUnmappedInventory `json:"unmapped_upstream"`
	ReleaseCheckpoints     []ReleaseCheckpoint       `json:"release_checkpoints,omitempty"`
	ContinuityCategories   []ContinuityCategory      `json:"continuity_categories,omitempty"`
	Summary                Summary                   `json:"summary"`
	UnmappedSurfaces       []string                  `json:"unmapped_surfaces,omitempty"`
	Surfaces               []SurfaceReport           `json:"surfaces"`
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

type UpstreamUnmappedInventory struct {
	SourceFiles []string                    `json:"source_files,omitempty"`
	DocsFiles   []string                    `json:"docs_files,omitempty"`
	TestFiles   []string                    `json:"test_files,omitempty"`
	TestSuites  []UpstreamUnmappedTestSuite `json:"test_suites,omitempty"`
}

type CatalogFamilyReport struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Count       int      `json:"count"`
	Examples    []string `json:"examples,omitempty"`
	Status      Status   `json:"status"`
	Reason      string   `json:"reason"`
	Confidence  string   `json:"confidence"`
	GapSeverity string   `json:"gap_severity"`

	SourcePairs  []SourcePairEvidence  `json:"source_pairs,omitempty"`
	ProgressRows []ProgressRowEvidence `json:"progress_rows,omitempty"`
}

type UpstreamUnmappedTestSuite struct {
	Suite        string   `json:"suite"`
	Count        int      `json:"count"`
	SourcePrefix string   `json:"source_prefix,omitempty"`
	Examples     []string `json:"examples,omitempty"`
	ProgressRows []string `json:"progress_rows,omitempty"`
}

type ReleaseCheckpoint struct {
	Label   string `json:"label"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type ContinuityCategory struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	SurfaceIDs   []string `json:"surface_ids"`
	Status       Status   `json:"status"`
	GapSeverity  string   `json:"gap_severity"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Reason       string   `json:"reason"`
}

type Summary struct {
	Total                 int            `json:"total"`
	Critical              int            `json:"critical"`
	SurfaceStrictFailures int            `json:"surface_strict_failures"`
	UnmappedUpstreamFiles int            `json:"unmapped_upstream_files"`
	StrictFailures        int            `json:"strict_failures"`
	ByStatus              map[string]int `json:"by_status"`
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
	Modules      []string
}

type continuityCategoryDefinition struct {
	ID         string
	Title      string
	SurfaceIDs []string
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
	report.PluginCatalog = buildPluginCatalogClassification(hermesPath, hermesSHA, sourcePairsState, sourcePairs.Pairs, rows)
	report.SkillCatalog = buildSkillCatalogClassification(hermesPath, hermesSHA, sourcePairsState, sourcePairs.Pairs, rows)
	report.GatewayPlatformCatalog = buildGatewayPlatformClassification(hermesPath, hermesSHA, sourcePairsState, sourcePairs.Pairs, rows)
	report.WebDashboardCatalog = buildWebDashboardClassification(hermesPath, hermesSHA, sourcePairsState, sourcePairs.Pairs, rows)
	definitions := defaultSurfaces()
	mappedUpstream := mappedUpstreamRefs(definitions, sourcePairs.Pairs, rows)
	report.UnmappedUpstream = buildUnmappedUpstreamInventory(hermesPath, mappedUpstream, rows)
	report.ReleaseCheckpoints = buildReleaseCheckpoints(repoRoot, hermesPath)
	for _, def := range definitions {
		surface := buildSurfaceReport(def, hermesPath, hermesSHA, sourcePairsState, sourcePairs.Pairs, rows)
		report.Surfaces = append(report.Surfaces, surface)
		report.Summary.Total++
		if surface.Critical {
			report.Summary.Critical++
		}
		report.Summary.ByStatus[string(surface.Status)]++
		if surface.Critical && !strictStatusPasses(surface.Status) {
			report.Summary.SurfaceStrictFailures++
			report.UnmappedSurfaces = append(report.UnmappedSurfaces, surface.ID)
		}
	}
	report.ContinuityCategories = buildContinuityCategories(report.Surfaces)
	report.Summary.UnmappedUpstreamFiles = report.UnmappedUpstream.total()
	report.Summary.StrictFailures = report.Summary.SurfaceStrictFailures + report.Summary.UnmappedUpstreamFiles
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
			Modules:      []string{"profiles", "config"},
		},
		{
			ID:           "sessions",
			Title:        "Sessions, Recaps, And Context Continuity",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"run_agent.py", "gateway/session.py", "tools/session_search_tool.py", "hermes_cli/session_recap.py"},
			Keywords:     []string{"session", "sessions", "transcript", "recap", "summary", "context"},
			Modules:      []string{"sessions"},
		},
		{
			ID:           "goncho_memory",
			Title:        "Memory, Goncho, And Honcho Compatibility",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"tools/memory_tool.py", "agent/memory_manager.py", "plugins/memory/__init__.py"},
			Keywords:     []string{"memory", "goncho", "honcho", "recall", "remember", "knowledge graph"},
			Modules:      []string{"goncho", "memory"},
		},
		{
			ID:           "learning_loop",
			Title:        "Learning Loop And Candidate Updates",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"agent/curator.py", "hermes_cli/curator.py", "tools/skill_usage.py"},
			Keywords:     []string{"learning", "curator", "candidate", "feedback", "outcome", "skill update", "memory update"},
			Modules:      []string{"skills", "memory", "goncho"},
		},
		{
			ID:           "prompt_assembly",
			Title:        "Prompt Assembly And Insertion Ordering",
			Category:     "continuity",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"agent/prompt_builder.py", "agent/skill_commands.py", "agent/skill_preprocessing.py", "agent/skill_utils.py"},
			Keywords:     []string{"prompt", "context", "insertion", "skill preprocessing", "budget", "assembly"},
			Modules:      []string{"runtime", "skills", "memory", "sessions"},
		},
		{
			ID:           "provider_auth_setup",
			Title:        "Provider, Auth, And Setup",
			Category:     "operator",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"hermes_cli/auth_commands.py", "hermes_cli/providers.py", "hermes_cli/setup.py"},
			Keywords:     []string{"provider", "auth", "oauth", "credential", "setup", "model", "openrouter", "codex"},
			Modules:      []string{"providers", "config", "doctor"},
		},
		{
			ID:           "gateway_channels",
			Title:        "Gateway And Channel Operation Sequence",
			Category:     "channels",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"gateway/run.py"},
			Keywords:     []string{"gateway", "channel", "telegram", "slack", "whatsapp", "discord", "outbound", "tool progress"},
			Modules:      []string{"gateway", "channels"},
		},
		{
			ID:           "tool_runtime",
			Title:        "Tool Runtime And Execution Safety",
			Category:     "runtime",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"tools/registry.py", "tools/file_tools.py", "tools/code_execution_tool.py"},
			Keywords:     []string{"tool", "tools", "registry", "execution", "permission", "sandbox", "file ops"},
			Modules:      []string{"tools"},
		},
		{
			ID:           "mcp_acp",
			Title:        "MCP And ACP",
			Category:     "tools",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"hermes_cli/mcp_config.py", "tools/mcp_tool.py", "acp_adapter/entry.py", "acp_adapter/server.py"},
			Keywords:     []string{"mcp", "acp", "adapter", "server", "protocol"},
			Modules:      []string{"tools"},
		},
		{
			ID:           "tui_cli",
			Title:        "TUI And CLI Surface",
			Category:     "operator",
			Priority:     "P0",
			Critical:     true,
			UpstreamRefs: []string{"ui-tui/package.json", "cli.py"},
			Keywords:     []string{"tui", "cli", "command", "terminal", "ui-tui", "curses", "cobra"},
			Modules:      []string{"tui"},
		},
	}
}

func defaultContinuityCategories() []continuityCategoryDefinition {
	return []continuityCategoryDefinition{
		{
			ID:         "sessions",
			Title:      "Sessions",
			SurfaceIDs: []string{"sessions"},
		},
		{
			ID:         "memory_goncho_honcho_compatibility",
			Title:      "Memory/Goncho/Honcho compatibility",
			SurfaceIDs: []string{"goncho_memory"},
		},
		{
			ID:         "workspace_peer_profile_identity_boundaries",
			Title:      "Workspace/peer/profile identity boundaries",
			SurfaceIDs: []string{"profiles"},
		},
		{
			ID:         "context_retrieval_and_prompt_budget",
			Title:      "Context retrieval and prompt budget",
			SurfaceIDs: []string{"sessions", "prompt_assembly"},
		},
		{
			ID:         "summaries_conclusions_search",
			Title:      "Summaries/conclusions/search",
			SurfaceIDs: []string{"sessions"},
		},
		{
			ID:         "skill_templates_and_skills_ux",
			Title:      "Skill templates and skills UX",
			SurfaceIDs: []string{"learning_loop", "prompt_assembly"},
		},
		{
			ID:         "skill_precedence_sync_update_reset",
			Title:      "Skill precedence/sync/update/reset",
			SurfaceIDs: []string{"learning_loop", "prompt_assembly"},
		},
		{
			ID:         "learning_loop_curator_behavior",
			Title:      "Learning-loop curator behavior",
			SurfaceIDs: []string{"learning_loop"},
		},
		{
			ID:         "candidate_memory_skill_updates",
			Title:      "Candidate memory/skill updates",
			SurfaceIDs: []string{"learning_loop", "goncho_memory"},
		},
		{
			ID:         "feedback_outcome_scoring",
			Title:      "Feedback/outcome scoring",
			SurfaceIDs: []string{"learning_loop"},
		},
		{
			ID:         "audit_trail",
			Title:      "Audit trail",
			SurfaceIDs: []string{"sessions", "tool_runtime", "learning_loop"},
		},
		{
			ID:         "mutation_safety",
			Title:      "Mutation safety",
			SurfaceIDs: []string{"tool_runtime", "learning_loop", "goncho_memory"},
		},
		{
			ID:         "prompt_context_memory_skill_insertion_ordering",
			Title:      "Prompt/context/memory/skill insertion ordering",
			SurfaceIDs: []string{"prompt_assembly", "goncho_memory", "sessions", "learning_loop"},
		},
		{
			ID:         "profile_scoped_isolation",
			Title:      "Profile-scoped isolation",
			SurfaceIDs: []string{"profiles", "sessions", "gateway_channels"},
		},
	}
}

func buildContinuityCategories(surfaces []SurfaceReport) []ContinuityCategory {
	byID := map[string]SurfaceReport{}
	for _, surface := range surfaces {
		byID[surface.ID] = surface
	}
	var categories []ContinuityCategory
	for _, def := range defaultContinuityCategories() {
		category := ContinuityCategory{
			ID:         def.ID,
			Title:      def.Title,
			SurfaceIDs: append([]string(nil), def.SurfaceIDs...),
			Status:     StatusCovered,
			Reason:     "Mapped surfaces are strictly covered.",
		}
		var matched []SurfaceReport
		var missing []string
		for _, surfaceID := range def.SurfaceIDs {
			surface, ok := byID[surfaceID]
			if !ok {
				missing = append(missing, surfaceID)
				continue
			}
			matched = append(matched, surface)
			category.EvidenceRefs = append(category.EvidenceRefs, surface.UpstreamRefs...)
			category.EvidenceRefs = append(category.EvidenceRefs, surface.GormesRefs...)
			for _, pair := range surface.SourcePairs {
				category.EvidenceRefs = append(category.EvidenceRefs, pair.HermesFile)
				category.EvidenceRefs = append(category.EvidenceRefs, pair.UpstreamTests...)
			}
		}
		if len(missing) > 0 {
			category.Status = StatusMissing
			category.Reason = "One or more expected fidelity surfaces are absent: " + strings.Join(missing, ", ") + "."
		} else if len(matched) == 0 {
			category.Status = StatusMissing
			category.Reason = "No fidelity surface maps this continuity category."
		} else {
			worst := matched[0]
			for _, surface := range matched[1:] {
				if statusRank(surface.Status) > statusRank(worst.Status) {
					worst = surface
				}
			}
			category.Status = worst.Status
			if strictStatusPasses(worst.Status) {
				category.Reason = "Mapped surfaces are strictly covered: " + strings.Join(def.SurfaceIDs, ", ") + "."
			} else {
				category.Reason = fmt.Sprintf("Mapped through surfaces: %s; worst status is %s=%s.", strings.Join(def.SurfaceIDs, ", "), worst.ID, worst.Status)
			}
		}
		category.GapSeverity = gapSeverity(category.Status, true)
		category.EvidenceRefs = uniqueSorted(category.EvidenceRefs)
		categories = append(categories, category)
	}
	return categories
}

func statusRank(status Status) int {
	switch status {
	case StatusBlocked, StatusStaleUpstream, StatusMissing:
		return 50
	case StatusVague:
		return 40
	case StatusPlanned:
		return 30
	case StatusPartial:
		return 20
	case StatusCovered, StatusOwnedDivergence, StatusExcluded:
		return 0
	default:
		return 10
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
	if surfaceAcceptsModule(def, row.Item.Module) && containsKeyword(tightRowText(row.Item), def.Keywords) {
		reasons = append(reasons, "taxonomy_keyword")
	}
	return uniqueSorted(reasons)
}

func surfaceAcceptsModule(def surfaceDefinition, module string) bool {
	module = strings.TrimSpace(module)
	for _, candidate := range def.Modules {
		if module == candidate {
			return true
		}
	}
	return false
}

func tightRowText(item progress.Item) string {
	parts := []string{
		item.Name,
		item.Contract,
		item.Module,
		item.Fixture,
		strings.Join(item.SourceRefs, " "),
	}
	return strings.ToLower(strings.Join(parts, " "))
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

func (u UpstreamUnmappedInventory) total() int {
	return len(u.SourceFiles) + len(u.DocsFiles) + len(u.TestFiles)
}

func mappedUpstreamRefs(definitions []surfaceDefinition, pairs []sourcePair, rows []progressRow) map[string]bool {
	mapped := map[string]bool{}
	for _, def := range definitions {
		for _, ref := range def.UpstreamRefs {
			addMappedUpstreamRef(mapped, ref)
		}
	}
	for _, pair := range pairs {
		addMappedUpstreamRef(mapped, pair.HermesFile)
		for _, ref := range pair.UpstreamTests {
			addMappedUpstreamRef(mapped, ref)
		}
	}
	for _, row := range rows {
		for _, ref := range row.Item.SourceRefs {
			addMappedUpstreamRef(mapped, ref)
		}
		if row.Item.Provenance != nil {
			addMappedUpstreamRef(mapped, row.Item.Provenance.UpstreamRef)
			for _, ref := range row.Item.Provenance.UpstreamRefs {
				addMappedUpstreamRef(mapped, ref)
			}
		}
	}
	return mapped
}

func addMappedUpstreamRef(mapped map[string]bool, ref string) {
	original := filepath.ToSlash(strings.TrimSpace(ref))
	ref = normalizeUpstreamRef(original)
	if ref == "" {
		ref = original
	}
	ref = strings.TrimPrefix(ref, "./")
	if ref != "" {
		mapped[ref] = true
	}
}

func buildUnmappedUpstreamInventory(root string, mapped map[string]bool, rows []progressRow) UpstreamUnmappedInventory {
	var unmapped UpstreamUnmappedInventory
	if root == "" {
		return unmapped
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if ignoredInventoryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if mapped[slash] {
			return nil
		}
		if isHermesReleaseCheckpointFile(slash) {
			return nil
		}
		switch {
		case isUpstreamTestFile(slash):
			unmapped.TestFiles = append(unmapped.TestFiles, slash)
		case isUpstreamDocsFile(slash):
			unmapped.DocsFiles = append(unmapped.DocsFiles, slash)
		case isUpstreamSourceFile(slash):
			unmapped.SourceFiles = append(unmapped.SourceFiles, slash)
		}
		return nil
	})
	unmapped.SourceFiles = uniqueSorted(unmapped.SourceFiles)
	unmapped.DocsFiles = uniqueSorted(unmapped.DocsFiles)
	unmapped.TestFiles = uniqueSorted(unmapped.TestFiles)
	unmapped.TestSuites = buildUnmappedTestSuites(unmapped.TestFiles, rows)
	return unmapped
}

func buildUnmappedTestSuites(testFiles []string, rows []progressRow) []UpstreamUnmappedTestSuite {
	bySuite := map[string]*UpstreamUnmappedTestSuite{}
	for _, file := range testFiles {
		suite, sourcePrefix := classifyUpstreamTestSuite(file)
		if suite == "" {
			continue
		}
		row := bySuite[suite]
		if row == nil {
			row = &UpstreamUnmappedTestSuite{Suite: suite, SourcePrefix: sourcePrefix}
			bySuite[suite] = row
		}
		row.Count++
		if len(row.Examples) < 5 {
			row.Examples = append(row.Examples, file)
		}
	}
	out := make([]UpstreamUnmappedTestSuite, 0, len(bySuite))
	for _, row := range bySuite {
		row.Examples = uniqueSorted(row.Examples)
		row.ProgressRows = progressRowsForSourcePrefix(row.SourcePrefix, rows)
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Suite < out[j].Suite })
	return out
}

func classifyUpstreamTestSuite(file string) (suite string, sourcePrefix string) {
	file = filepath.ToSlash(strings.TrimSpace(file))
	parts := strings.Split(file, "/")
	if len(parts) >= 2 && parts[0] == "tests" {
		suite = parts[1]
		if len(parts) >= 3 && parts[1] == "agent" {
			suite = strings.Join(parts[1:3], "/")
		}
		return suite, suite
	}
	if len(parts) >= 3 && parts[1] == "src" && parts[2] == "__tests__" {
		return parts[0], parts[0]
	}
	return "", ""
}

func progressRowsForSourcePrefix(sourcePrefix string, rows []progressRow) []string {
	if sourcePrefix == "" {
		return nil
	}
	var matches []string
	for _, row := range rows {
		for _, ref := range row.Item.SourceRefs {
			normalized := normalizeUpstreamRef(ref)
			if normalized == "" {
				continue
			}
			if normalized == sourcePrefix || strings.HasPrefix(normalized, sourcePrefix+"/") {
				matches = append(matches, row.Item.Name)
				break
			}
		}
	}
	matches = uniqueSorted(matches)
	if len(matches) > 5 {
		return matches[:5]
	}
	return matches
}

func buildReleaseCheckpoints(repoRoot, hermesPath string) []ReleaseCheckpoint {
	var checkpoints []ReleaseCheckpoint
	if hermesPath != "" {
		matches, _ := filepath.Glob(filepath.Join(hermesPath, "RELEASE_v*.md"))
		sort.Strings(matches)
		for _, match := range matches {
			base := strings.TrimSuffix(filepath.Base(match), ".md")
			checkpoints = append(checkpoints, ReleaseCheckpoint{
				Label:   "Hermes " + base,
				Path:    displayPath(repoRoot, match),
				Present: true,
			})
		}
	}
	for _, checkpoint := range []ReleaseCheckpoint{
		{
			Label: "Gormes Hermes v0.14 module pairings",
			Path:  "webpages/docs/content/building-gormes/architecture_plan/hermes-v0.14-module-pairings.md",
		},
		{
			Label: "Gormes Hermes source-pair manifest",
			Path:  "webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json",
		},
		{
			Label: "Gormes Hermes source-pair report",
			Path:  "webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.md",
		},
	} {
		abs := filepath.Join(repoRoot, filepath.FromSlash(checkpoint.Path))
		if _, err := os.Stat(abs); err == nil {
			checkpoint.Present = true
		}
		checkpoints = append(checkpoints, checkpoint)
	}
	return checkpoints
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
			if ignoredInventoryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		slash := filepath.ToSlash(rel)
		if isUpstreamSourceFile(slash) {
			counts.SourceFiles++
		}
		if isUpstreamDocsFile(slash) {
			counts.DocsFiles++
		}
		if isUpstreamTestFile(slash) {
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
			if ignoredInventoryDir(d.Name()) {
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

type catalogFamilyDefinition struct {
	ID       string
	Title    string
	Prefixes []string
	Contains []string
	Keywords []string
}

func defaultPluginCatalogFamilies() []catalogFamilyDefinition {
	return []catalogFamilyDefinition{
		{
			ID:       "browser_web_search",
			Title:    "Browser And Web Search Plugins",
			Prefixes: []string{"plugins/browser/", "plugins/web/"},
			Keywords: []string{"browser", "web", "firecrawl", "browser use", "web search"},
		},
		{
			ID:       "dashboard_observability",
			Title:    "Dashboard And Observability Plugins",
			Prefixes: []string{"plugins/hermes-achievements/", "plugins/kanban/"},
			Contains: []string{"/dashboard/", "/docs/assets/"},
			Keywords: []string{"dashboard", "observability", "achievement", "kanban", "plugin slots"},
		},
		{
			ID:       "google_meet",
			Title:    "Google Meet Plugin",
			Prefixes: []string{"plugins/google_meet/"},
			Keywords: []string{"google meet", "meet", "realtime", "audio bridge"},
		},
		{
			ID:       "image_video_generation",
			Title:    "Image And Video Generation Plugins",
			Prefixes: []string{"plugins/image_gen/", "plugins/video_gen/"},
			Keywords: []string{"image", "video", "generation", "media", "fal"},
		},
		{
			ID:       "memory_providers",
			Title:    "Memory Provider Plugins",
			Prefixes: []string{"plugins/memory/"},
			Keywords: []string{"memory", "honcho", "hindsight", "provider"},
		},
		{
			ID:       "model_providers",
			Title:    "Model Provider Plugins",
			Prefixes: []string{"plugins/model-providers/"},
			Keywords: []string{"model", "provider", "openrouter", "openai-codex", "auth"},
		},
		{
			ID:       "platform_adapters",
			Title:    "Platform Adapter Plugins",
			Prefixes: []string{"plugins/platforms/"},
			Keywords: []string{"platform", "adapter", "simplex", "teams", "channel"},
		},
		{
			ID:       "spotify",
			Title:    "Spotify Plugin",
			Prefixes: []string{"plugins/spotify/"},
			Keywords: []string{"spotify", "music"},
		},
		{
			ID:       "teams_pipeline",
			Title:    "Teams Pipeline Plugin",
			Prefixes: []string{"plugins/teams_pipeline/"},
			Keywords: []string{"teams pipeline", "pipeline", "teams"},
		},
	}
}

func buildPluginCatalogClassification(root, hermesSHA, sourcePairsState string, pairs []sourcePair, rows []progressRow) []CatalogFamilyReport {
	if root == "" {
		return nil
	}
	definitions := defaultPluginCatalogFamilies()
	byID := map[string]*CatalogFamilyReport{}
	byDef := map[string]catalogFamilyDefinition{}
	for _, def := range definitions {
		byDef[def.ID] = def
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if ignoredInventoryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if !pluginCatalogEvidenceFile(slash) {
			return nil
		}
		for _, def := range definitions {
			if !pluginFamilyMatchesPath(def, slash) {
				continue
			}
			family := byID[def.ID]
			if family == nil {
				family = &CatalogFamilyReport{ID: def.ID, Title: def.Title}
				byID[def.ID] = family
			}
			family.Count++
			family.Examples = append(family.Examples, slash)
			break
		}
		return nil
	})

	for id, family := range byID {
		def := byDef[id]
		matchedNames := map[string]bool{}
		for _, pair := range pairs {
			if !pluginFamilyMatchesSourcePair(def, pair) {
				continue
			}
			ev := SourcePairEvidence{
				HermesFile:           cleanHermesRel(pair.HermesFile),
				Status:               pair.Status,
				Contract:             pair.Contract,
				GormesTargets:        cleanStrings(pair.GormesTargets),
				Tests:                cleanStrings(pair.Tests),
				ProgressRows:         cleanStrings(pair.ProgressRows),
				UpstreamTests:        cleanStrings(pair.UpstreamTests),
				LastCheckedHermesSHA: pair.LastCheckedHermesSHA,
				Stale:                pairStale(hermesSHA, pair.LastCheckedHermesSHA) || sourcePairsState == "stale",
			}
			family.SourcePairs = append(family.SourcePairs, ev)
			for _, name := range pair.ProgressRows {
				matchedNames[strings.ToLower(strings.TrimSpace(name))] = true
			}
		}

		for _, row := range rows {
			reasons := pluginFamilyRowMatchReasons(def, row, matchedNames)
			if len(reasons) == 0 {
				continue
			}
			family.ProgressRows = append(family.ProgressRows, progressRowEvidence(row, reasons))
		}
		family.Examples = uniqueSorted(family.Examples)
		family.SourcePairs = sortSourcePairEvidence(family.SourcePairs)
		family.ProgressRows = sortProgressRowEvidence(family.ProgressRows)
		family.Status, family.Reason, family.Confidence = deriveCatalogFamilyStatus(*family)
		family.GapSeverity = gapSeverity(family.Status, true)
	}

	out := make([]CatalogFamilyReport, 0, len(byID))
	for _, family := range byID {
		out = append(out, *family)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func pluginCatalogEvidenceFile(path string) bool {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "plugins/") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py", ".ts", ".tsx", ".js", ".jsx", ".md", ".yaml", ".yml", ".json", ".css":
		return true
	default:
		return false
	}
}

func pluginFamilyMatchesPath(def catalogFamilyDefinition, path string) bool {
	path = cleanHermesRel(path)
	for _, prefix := range def.Prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, fragment := range def.Contains {
		if strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

func pluginFamilyMatchesSourcePair(def catalogFamilyDefinition, pair sourcePair) bool {
	if pluginFamilyMatchesPath(def, pair.HermesFile) {
		return true
	}
	text := strings.ToLower(cleanHermesRel(pair.HermesFile) + " " + pair.Contract + " " + strings.Join(pair.ProgressRows, " "))
	return containsKeyword(text, def.Keywords)
}

func pluginFamilyRowMatchReasons(def catalogFamilyDefinition, row progressRow, matchedNames map[string]bool) []string {
	var reasons []string
	name := strings.ToLower(strings.TrimSpace(row.Item.Name))
	if matchedNames[name] {
		reasons = append(reasons, "source_pair_progress_row")
	}
	for _, source := range row.Item.SourceRefs {
		normalized := cleanHermesRel(source)
		if normalized != "" && pluginFamilyMatchesPath(def, normalized) {
			reasons = append(reasons, "source_ref:"+normalized)
		}
	}
	if containsKeyword(row.Text, def.Keywords) {
		reasons = append(reasons, "taxonomy_keyword")
	}
	return uniqueSorted(reasons)
}

func deriveCatalogFamilyStatus(family CatalogFamilyReport) (Status, string, string) {
	for _, pair := range family.SourcePairs {
		if pair.Stale {
			return StatusStaleUpstream, "Source-pair evidence is stale for the selected Hermes SHA.", "high"
		}
	}
	if hasPairStatus(family.SourcePairs, "excluded") {
		return StatusExcluded, "Source-pair manifest explicitly excludes this plugin family.", "high"
	}
	if hasPairStatus(family.SourcePairs, "owned") && hasValidatedProgress(family.ProgressRows) {
		return StatusOwnedDivergence, "Validated Gormes-owned divergence is explicitly recorded for this plugin family.", "high"
	}
	if hasPairStatus(family.SourcePairs, "covered") && hasValidatedProgress(family.ProgressRows) {
		return StatusCovered, "Covered source-pair evidence joins to a validated complete progress row with test evidence.", "high"
	}
	if hasPairStatus(family.SourcePairs, "partial") || hasCompleteProgress(family.ProgressRows) {
		return StatusPartial, "Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered.", "medium"
	}
	if hasPairStatus(family.SourcePairs, "planned") || hasPlannedProgress(family.ProgressRows) {
		return StatusPlanned, "A planned progress row or source-pair entry exists for this plugin family.", "medium"
	}
	if hasVagueProgress(family.ProgressRows) {
		return StatusVague, "Only vague progress evidence matched this plugin family.", "low"
	}
	return StatusMissing, "Upstream plugin files are present without source-pair, progress-row, exclusion, or owned-divergence evidence.", "low"
}

type gatewayPlatformFamilyDefinition struct {
	ID       string
	Title    string
	Keywords []string
}

func defaultGatewayPlatformFamilies() []gatewayPlatformFamilyDefinition {
	return []gatewayPlatformFamilyDefinition{
		{
			ID:       "platform_enum_config",
			Title:    "Gateway Platform Enum And Config Surface",
			Keywords: []string{"platform enum", "platform registry", "gateway config", "configured platform"},
		},
		{
			ID:       "gateway_runtime_lifecycle",
			Title:    "Gateway Runtime Lifecycle",
			Keywords: []string{"gateway runtime", "start_gateway", "lifecycle", "restart", "status"},
		},
		{
			ID:       "builtin_platform_connectors",
			Title:    "Built-In Gateway Platform Connectors",
			Keywords: []string{"builtin", "connector", "adapter", "telegram", "discord", "slack", "whatsapp", "msgraph", "qqbot", "yuanbao"},
		},
		{
			ID:       "platform_helpers",
			Title:    "Gateway Platform Helper Modules",
			Keywords: []string{"helper", "base adapter", "http client", "connected checker", "platform helper"},
		},
		{
			ID:       "platform_docs",
			Title:    "Gateway Platform Documentation",
			Keywords: []string{"adding a platform", "platform docs", "documentation"},
		},
		{
			ID:       "bundled_platform_plugins",
			Title:    "Bundled Platform Plugins",
			Keywords: []string{"bundled platform", "platform plugin", "google chat", "simplex", "line", "teams", "irc"},
		},
		{
			ID:       "api_server_surface",
			Title:    "Gateway API Server Surface",
			Keywords: []string{"api server", "dashboard", "openai-compatible", "http"},
		},
		{
			ID:       "tui_gateway_bridge",
			Title:    "TUI Gateway Bridge",
			Keywords: []string{"tui gateway", "websocket", "render", "bridge", "tuigateway"},
		},
		{
			ID:       "generated_artifacts",
			Title:    "Generated Or Cache Artifacts",
			Keywords: []string{"generated", "pycache", "bytecode", "artifact", "cache"},
		},
	}
}

func buildGatewayPlatformClassification(root, hermesSHA, sourcePairsState string, pairs []sourcePair, rows []progressRow) []CatalogFamilyReport {
	if root == "" {
		return nil
	}
	definitions := defaultGatewayPlatformFamilies()
	byID := map[string]*CatalogFamilyReport{}
	byDef := map[string]gatewayPlatformFamilyDefinition{}
	for _, def := range definitions {
		byDef[def.ID] = def
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if ignoredInventoryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if !gatewayPlatformEvidencePath(slash) {
			return nil
		}
		for _, id := range gatewayPlatformFamilyIDsForEvidence(slash) {
			def := byDef[id]
			if def.ID == "" {
				continue
			}
			family := byID[id]
			if family == nil {
				family = &CatalogFamilyReport{ID: def.ID, Title: def.Title}
				byID[id] = family
			}
			family.Count++
			family.Examples = append(family.Examples, slash)
		}
		return nil
	})

	for id, family := range byID {
		def := byDef[id]
		matchedNames := map[string]bool{}
		for _, pair := range pairs {
			if !gatewayPlatformFamilyMatchesSourcePair(def, pair) {
				continue
			}
			ev := SourcePairEvidence{
				HermesFile:           cleanHermesRel(pair.HermesFile),
				Status:               pair.Status,
				Contract:             pair.Contract,
				GormesTargets:        cleanStrings(pair.GormesTargets),
				Tests:                cleanStrings(pair.Tests),
				ProgressRows:         cleanStrings(pair.ProgressRows),
				UpstreamTests:        cleanStrings(pair.UpstreamTests),
				LastCheckedHermesSHA: pair.LastCheckedHermesSHA,
				Stale:                pairStale(hermesSHA, pair.LastCheckedHermesSHA) || sourcePairsState == "stale",
			}
			family.SourcePairs = append(family.SourcePairs, ev)
			for _, name := range pair.ProgressRows {
				matchedNames[strings.ToLower(strings.TrimSpace(name))] = true
			}
		}

		for _, row := range rows {
			reasons := gatewayPlatformFamilyRowMatchReasons(def, row, matchedNames)
			if len(reasons) == 0 {
				continue
			}
			family.ProgressRows = append(family.ProgressRows, progressRowEvidence(row, reasons))
		}
		family.Examples = uniqueSorted(family.Examples)
		family.SourcePairs = sortSourcePairEvidence(family.SourcePairs)
		family.ProgressRows = sortProgressRowEvidence(family.ProgressRows)
		family.Status, family.Reason, family.Confidence = deriveCatalogFamilyStatus(*family)
		family.GapSeverity = gapSeverity(family.Status, true)
	}

	out := make([]CatalogFamilyReport, 0, len(byID))
	for _, family := range byID {
		out = append(out, *family)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func gatewayPlatformEvidencePath(path string) bool {
	path = cleanHermesRel(path)
	if path == "gateway/config.py" || path == "gateway/run.py" {
		return true
	}
	if strings.HasPrefix(path, "gateway/platforms/") || strings.HasPrefix(path, "plugins/platforms/") || strings.HasPrefix(path, "tui_gateway/") {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".py", ".pyc", ".md", ".yaml", ".yml", ".json", ".ts", ".tsx", ".js", ".jsx":
			return true
		default:
			return false
		}
	}
	return false
}

func gatewayPlatformFamilyIDsForEvidence(path string) []string {
	path = cleanHermesRel(path)
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	if strings.Contains(lower, "__pycache__") || strings.Contains(lower, "generated") || strings.HasSuffix(lower, ".pyc") {
		return []string{"generated_artifacts"}
	}
	var ids []string
	switch {
	case lower == "gateway/config.py":
		ids = append(ids, "platform_enum_config")
	case lower == "gateway/run.py":
		ids = append(ids, "gateway_runtime_lifecycle")
	case lower == "gateway/platforms/api_server.py":
		ids = append(ids, "api_server_surface")
	case lower == "gateway/platforms/adding_a_platform.md":
		ids = append(ids, "platform_docs")
	case lower == "gateway/platforms/base.py" || lower == "gateway/platforms/helpers.py" || lower == "gateway/platforms/_http_client_limits.py":
		ids = append(ids, "platform_helpers")
	case strings.HasPrefix(lower, "plugins/platforms/"):
		ids = append(ids, "bundled_platform_plugins")
	case strings.HasPrefix(lower, "tui_gateway/"):
		ids = append(ids, "tui_gateway_bridge")
	case strings.HasPrefix(lower, "gateway/platforms/") && base != "":
		ids = append(ids, "builtin_platform_connectors")
	}
	return uniqueSorted(ids)
}

func gatewayPlatformFamilyMatchesPath(def gatewayPlatformFamilyDefinition, path string) bool {
	for _, id := range gatewayPlatformFamilyIDsForEvidence(path) {
		if id == def.ID {
			return true
		}
	}
	return false
}

func gatewayPlatformFamilyMatchesSourcePair(def gatewayPlatformFamilyDefinition, pair sourcePair) bool {
	if gatewayPlatformFamilyMatchesPath(def, pair.HermesFile) {
		return true
	}
	text := strings.ToLower(cleanHermesRel(pair.HermesFile) + " " + pair.Contract + " " + strings.Join(pair.ProgressRows, " "))
	return containsKeyword(text, def.Keywords)
}

func gatewayPlatformFamilyRowMatchReasons(def gatewayPlatformFamilyDefinition, row progressRow, matchedNames map[string]bool) []string {
	var reasons []string
	name := strings.ToLower(strings.TrimSpace(row.Item.Name))
	if matchedNames[name] {
		reasons = append(reasons, "source_pair_progress_row")
	}
	for _, source := range row.Item.SourceRefs {
		normalized := cleanHermesRel(source)
		if normalized != "" && gatewayPlatformFamilyMatchesPath(def, normalized) {
			reasons = append(reasons, "source_ref:"+normalized)
		}
	}
	if containsKeyword(row.Text, def.Keywords) {
		reasons = append(reasons, "taxonomy_keyword")
	}
	return uniqueSorted(reasons)
}

type webDashboardFamilyDefinition struct {
	ID       string
	Title    string
	Keywords []string
}

func defaultWebDashboardFamilies() []webDashboardFamilyDefinition {
	return []webDashboardFamilyDefinition{
		{
			ID:       "gateway_client_events",
			Title:    "Dashboard Gateway Client Event Shapes",
			Keywords: []string{"gateway client", "websocket", "event shape", "session submit", "sse"},
		},
		{
			ID:       "terminal_chat_pty",
			Title:    "Terminal Chat And PTY Dashboard Surface",
			Keywords: []string{"terminal chat", "pty", "chat page", "tool call", "slash popover"},
		},
		{
			ID:       "sessions_page",
			Title:    "Dashboard Sessions Page",
			Keywords: []string{"sessions page", "session endpoint", "conversation history", "dashboard sessions"},
		},
		{
			ID:       "profiles_config",
			Title:    "Dashboard Profiles And Config Pages",
			Keywords: []string{"profiles", "profile config", "config page", "workspace profile"},
		},
		{
			ID:       "plugin_pages_slots",
			Title:    "Dashboard Plugin Pages And Page-Scoped Slots",
			Keywords: []string{"plugin slot", "plugins page", "page-scoped", "plugin registry", "plugin sdk"},
		},
		{
			ID:       "oauth_provider_panels",
			Title:    "Dashboard OAuth And Provider Panels",
			Keywords: []string{"oauth", "provider panel", "login modal", "provider card", "capabilities"},
		},
		{
			ID:       "model_picker",
			Title:    "Dashboard Model Picker",
			Keywords: []string{"model picker", "model dialog", "model info", "provider model"},
		},
		{
			ID:       "cron_admin_jobs",
			Title:    "Dashboard Cron Admin And Jobs",
			Keywords: []string{"cron", "admin jobs", "scheduled jobs", "cron admin"},
		},
		{
			ID:       "i18n_catalog",
			Title:    "Dashboard I18n Catalog",
			Keywords: []string{"i18n", "locale", "language switcher", "translation"},
		},
		{
			ID:       "theme_system",
			Title:    "Dashboard Theme System",
			Keywords: []string{"theme", "theme provider", "theme switcher", "preset"},
		},
	}
}

func buildWebDashboardClassification(root, hermesSHA, sourcePairsState string, pairs []sourcePair, rows []progressRow) []CatalogFamilyReport {
	if root == "" {
		return nil
	}
	definitions := defaultWebDashboardFamilies()
	byID := map[string]*CatalogFamilyReport{}
	byDef := map[string]webDashboardFamilyDefinition{}
	for _, def := range definitions {
		byDef[def.ID] = def
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if ignoredInventoryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if !webDashboardEvidencePath(slash) {
			return nil
		}
		for _, id := range webDashboardFamilyIDsForEvidence(slash) {
			def := byDef[id]
			if def.ID == "" {
				continue
			}
			family := byID[id]
			if family == nil {
				family = &CatalogFamilyReport{ID: def.ID, Title: def.Title}
				byID[id] = family
			}
			family.Count++
			family.Examples = append(family.Examples, slash)
		}
		return nil
	})

	for id, family := range byID {
		def := byDef[id]
		matchedNames := map[string]bool{}
		for _, pair := range pairs {
			if !webDashboardFamilyMatchesSourcePair(def, pair) {
				continue
			}
			ev := SourcePairEvidence{
				HermesFile:           cleanHermesRel(pair.HermesFile),
				Status:               pair.Status,
				Contract:             pair.Contract,
				GormesTargets:        cleanStrings(pair.GormesTargets),
				Tests:                cleanStrings(pair.Tests),
				ProgressRows:         cleanStrings(pair.ProgressRows),
				UpstreamTests:        cleanStrings(pair.UpstreamTests),
				LastCheckedHermesSHA: pair.LastCheckedHermesSHA,
				Stale:                pairStale(hermesSHA, pair.LastCheckedHermesSHA) || sourcePairsState == "stale",
			}
			family.SourcePairs = append(family.SourcePairs, ev)
			for _, name := range pair.ProgressRows {
				matchedNames[strings.ToLower(strings.TrimSpace(name))] = true
			}
		}

		for _, row := range rows {
			reasons := webDashboardFamilyRowMatchReasons(def, row, matchedNames)
			if len(reasons) == 0 {
				continue
			}
			family.ProgressRows = append(family.ProgressRows, progressRowEvidence(row, reasons))
		}
		family.Examples = uniqueSorted(family.Examples)
		family.SourcePairs = sortSourcePairEvidence(family.SourcePairs)
		family.ProgressRows = sortProgressRowEvidence(family.ProgressRows)
		family.Status, family.Reason, family.Confidence = deriveCatalogFamilyStatus(*family)
		family.GapSeverity = gapSeverity(family.Status, true)
	}

	out := make([]CatalogFamilyReport, 0, len(byID))
	for _, family := range byID {
		out = append(out, *family)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func webDashboardEvidencePath(path string) bool {
	path = cleanHermesRel(path)
	if !strings.HasPrefix(path, "web/src/") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts", ".tsx", ".js", ".jsx", ".css":
		return len(webDashboardFamilyIDsForEvidence(path)) > 0
	default:
		return false
	}
}

func webDashboardFamilyIDsForEvidence(path string) []string {
	path = cleanHermesRel(path)
	lower := strings.ToLower(path)
	base := strings.TrimSuffix(filepath.Base(lower), filepath.Ext(lower))
	var ids []string
	switch {
	case lower == "web/src/lib/gatewayclient.ts" || lower == "web/src/lib/api.ts" || lower == "web/src/lib/slashexec.ts":
		ids = append(ids, "gateway_client_events")
	case lower == "web/src/pages/chatpage.tsx" || lower == "web/src/components/chatsidebar.tsx" || lower == "web/src/components/toolcall.tsx" || lower == "web/src/components/slashpopover.tsx" || lower == "web/src/components/markdown.tsx":
		ids = append(ids, "terminal_chat_pty")
	case lower == "web/src/pages/sessionspage.tsx":
		ids = append(ids, "sessions_page")
	case lower == "web/src/pages/profilespage.tsx" || lower == "web/src/pages/configpage.tsx" || lower == "web/src/components/platformscard.tsx":
		ids = append(ids, "profiles_config")
	case lower == "web/src/pages/pluginspage.tsx" || strings.HasPrefix(lower, "web/src/plugins/"):
		ids = append(ids, "plugin_pages_slots")
	case lower == "web/src/components/oauthproviderscard.tsx" || lower == "web/src/components/oauthloginmodal.tsx":
		ids = append(ids, "oauth_provider_panels")
	case lower == "web/src/components/modelpickerdialog.tsx" || lower == "web/src/components/modelinfocard.tsx" || lower == "web/src/pages/modelspage.tsx":
		ids = append(ids, "model_picker")
	case lower == "web/src/pages/cronpage.tsx":
		ids = append(ids, "cron_admin_jobs")
	case strings.HasPrefix(lower, "web/src/i18n/") || base == "languageswitcher":
		ids = append(ids, "i18n_catalog")
	case strings.HasPrefix(lower, "web/src/themes/") || base == "themeswitcher":
		ids = append(ids, "theme_system")
	}
	return uniqueSorted(ids)
}

func webDashboardFamilyMatchesPath(def webDashboardFamilyDefinition, path string) bool {
	for _, id := range webDashboardFamilyIDsForEvidence(path) {
		if id == def.ID {
			return true
		}
	}
	return false
}

func webDashboardFamilyMatchesSourcePair(def webDashboardFamilyDefinition, pair sourcePair) bool {
	if webDashboardFamilyMatchesPath(def, pair.HermesFile) {
		return true
	}
	text := strings.ToLower(cleanHermesRel(pair.HermesFile) + " " + pair.Contract + " " + strings.Join(pair.ProgressRows, " "))
	return containsKeyword(text, def.Keywords)
}

func webDashboardFamilyRowMatchReasons(def webDashboardFamilyDefinition, row progressRow, matchedNames map[string]bool) []string {
	var reasons []string
	name := strings.ToLower(strings.TrimSpace(row.Item.Name))
	if matchedNames[name] {
		reasons = append(reasons, "source_pair_progress_row")
	}
	for _, source := range row.Item.SourceRefs {
		normalized := cleanHermesRel(source)
		if normalized != "" && webDashboardFamilyMatchesPath(def, normalized) {
			reasons = append(reasons, "source_ref:"+normalized)
		}
	}
	if containsKeyword(row.Text, def.Keywords) {
		reasons = append(reasons, "taxonomy_keyword")
	}
	return uniqueSorted(reasons)
}

type skillCatalogFamilyDefinition struct {
	ID       string
	Title    string
	Keywords []string
}

func defaultSkillCatalogFamilies() []skillCatalogFamilyDefinition {
	return []skillCatalogFamilyDefinition{
		{
			ID:       "bundled_catalog_metadata",
			Title:    "Bundled Skill Catalog Metadata",
			Keywords: []string{"bundled", "portable skill", "skill.md format", "skill metadata", "prompt exposure", "slash-command"},
		},
		{
			ID:       "optional_catalog_metadata",
			Title:    "Optional Skill Catalog Metadata",
			Keywords: []string{"optional", "optional skill", "catalog metadata", "metadata.hermes category"},
		},
		{
			ID:       "category_descriptions",
			Title:    "Skill Category Descriptions",
			Keywords: []string{"description", "category", "category description"},
		},
		{
			ID:       "prerequisites_readiness_metadata",
			Title:    "Prerequisites And Readiness Metadata",
			Keywords: []string{"prerequisite", "readiness", "env", "credential", "guard", "condition", "review state"},
		},
		{
			ID:       "triggers_tags_related_skills",
			Title:    "Triggers, Tags, And Related Skills",
			Keywords: []string{"trigger", "tag", "related", "related skill", "metadata.hermes tags"},
		},
		{
			ID:       "support_assets",
			Title:    "Skill Support Assets",
			Keywords: []string{"support", "asset", "reference", "template", "preprocess"},
		},
		{
			ID:       "sync_reset_boundaries",
			Title:    "Skill Sync And Reset Boundaries",
			Keywords: []string{"sync", "reset", "lockfile", "source lock", "index-cache", "manifest", "profile copy"},
		},
		{
			ID:       "python_script_examples",
			Title:    "Python And Script-Only Skill Examples",
			Keywords: []string{"python", "script", "script-only", "helper script"},
		},
	}
}

func buildSkillCatalogClassification(root, hermesSHA, sourcePairsState string, pairs []sourcePair, rows []progressRow) []CatalogFamilyReport {
	if root == "" {
		return nil
	}
	definitions := defaultSkillCatalogFamilies()
	byID := map[string]*CatalogFamilyReport{}
	byDef := map[string]skillCatalogFamilyDefinition{}
	for _, def := range definitions {
		byDef[def.ID] = def
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if ignoredInventoryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if !skillCatalogEvidencePath(slash) {
			return nil
		}
		body := ""
		if raw, readErr := os.ReadFile(path); readErr == nil {
			body = string(raw)
		}
		for _, id := range skillCatalogFamilyIDsForEvidence(slash, body) {
			def := byDef[id]
			if def.ID == "" {
				continue
			}
			family := byID[id]
			if family == nil {
				family = &CatalogFamilyReport{ID: def.ID, Title: def.Title}
				byID[id] = family
			}
			family.Count++
			family.Examples = append(family.Examples, slash)
		}
		return nil
	})

	for id, family := range byID {
		def := byDef[id]
		matchedNames := map[string]bool{}
		for _, pair := range pairs {
			if !skillCatalogFamilyMatchesSourcePair(def, pair) {
				continue
			}
			ev := SourcePairEvidence{
				HermesFile:           cleanHermesRel(pair.HermesFile),
				Status:               pair.Status,
				Contract:             pair.Contract,
				GormesTargets:        cleanStrings(pair.GormesTargets),
				Tests:                cleanStrings(pair.Tests),
				ProgressRows:         cleanStrings(pair.ProgressRows),
				UpstreamTests:        cleanStrings(pair.UpstreamTests),
				LastCheckedHermesSHA: pair.LastCheckedHermesSHA,
				Stale:                pairStale(hermesSHA, pair.LastCheckedHermesSHA) || sourcePairsState == "stale",
			}
			family.SourcePairs = append(family.SourcePairs, ev)
			for _, name := range pair.ProgressRows {
				matchedNames[strings.ToLower(strings.TrimSpace(name))] = true
			}
		}

		for _, row := range rows {
			reasons := skillCatalogFamilyRowMatchReasons(def, row, matchedNames)
			if len(reasons) == 0 {
				continue
			}
			family.ProgressRows = append(family.ProgressRows, progressRowEvidence(row, reasons))
		}
		family.Examples = uniqueSorted(family.Examples)
		family.SourcePairs = sortSourcePairEvidence(family.SourcePairs)
		family.ProgressRows = sortProgressRowEvidence(family.ProgressRows)
		family.Status, family.Reason, family.Confidence = deriveCatalogFamilyStatus(*family)
		family.GapSeverity = gapSeverity(family.Status, true)
	}

	out := make([]CatalogFamilyReport, 0, len(byID))
	for _, family := range byID {
		out = append(out, *family)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func skillCatalogEvidencePath(path string) bool {
	path = cleanHermesRel(path)
	if !strings.HasPrefix(path, "skills/") && !strings.HasPrefix(path, "optional-skills/") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".yaml", ".yml", ".json", ".html", ".css", ".js", ".ts", ".py", ".sh", ".txt", ".png", ".jpg", ".jpeg", ".svg", ".gif", ".webp":
		return true
	default:
		return filepath.Base(path) == "requirements.txt"
	}
}

func skillCatalogFamilyIDsForEvidence(path, body string) []string {
	path = cleanHermesRel(path)
	if !skillCatalogEvidencePath(path) {
		return nil
	}
	lowerPath := strings.ToLower(path)
	base := filepath.Base(lowerPath)
	lowerBody := strings.ToLower(body)
	var ids []string

	if base == "skill.md" {
		if strings.HasPrefix(lowerPath, "optional-skills/") {
			ids = append(ids, "optional_catalog_metadata")
		} else {
			ids = append(ids, "bundled_catalog_metadata")
		}
		if containsAny(lowerBody, []string{"prerequisites:", "required_environment_variables", "required_environment_variable_groups", "credential_groups", "conditions:", "review_state:"}) {
			ids = append(ids, "prerequisites_readiness_metadata")
		}
		if containsAny(lowerBody, []string{"triggers:", "tags:", "related_skills:", "metadata:", "hermes:"}) {
			ids = append(ids, "triggers_tags_related_skills")
		}
	}
	if base == "description.md" {
		ids = append(ids, "category_descriptions")
	}
	if strings.Contains(lowerPath, "/index-cache/") || containsAny(lowerPath, []string{"skill-lock", "source-lock", "manifest", "sync", "reset"}) {
		ids = append(ids, "sync_reset_boundaries")
	}
	if strings.Contains(lowerPath, "/scripts/") || strings.HasSuffix(lowerPath, ".py") || strings.HasSuffix(lowerPath, ".sh") {
		ids = append(ids, "python_script_examples")
	}
	if skillCatalogSupportAssetPath(lowerPath) {
		ids = append(ids, "support_assets")
	}
	return uniqueSorted(ids)
}

func skillCatalogSupportAssetPath(path string) bool {
	base := filepath.Base(path)
	if base == "skill.md" || base == "description.md" || strings.Contains(path, "/index-cache/") || strings.Contains(path, "/scripts/") {
		return false
	}
	if strings.Contains(path, "/references/") || strings.Contains(path, "/templates/") || strings.Contains(path, "/assets/") {
		return true
	}
	switch base {
	case "readme.md", "examples.md", "port_notes.md", "troubleshooting.md", "attribution.md", "requirements.txt":
		return true
	default:
		return false
	}
}

func skillCatalogFamilyMatchesPath(def skillCatalogFamilyDefinition, path string) bool {
	for _, id := range skillCatalogFamilyIDsForEvidence(path, "") {
		if id == def.ID {
			return true
		}
	}
	return false
}

func skillCatalogFamilyMatchesSourcePair(def skillCatalogFamilyDefinition, pair sourcePair) bool {
	if skillCatalogFamilyMatchesPath(def, pair.HermesFile) {
		return true
	}
	text := strings.ToLower(cleanHermesRel(pair.HermesFile) + " " + pair.Contract + " " + strings.Join(pair.ProgressRows, " "))
	return containsKeyword(text, def.Keywords)
}

func skillCatalogFamilyRowMatchReasons(def skillCatalogFamilyDefinition, row progressRow, matchedNames map[string]bool) []string {
	var reasons []string
	name := strings.ToLower(strings.TrimSpace(row.Item.Name))
	if matchedNames[name] {
		reasons = append(reasons, "source_pair_progress_row")
	}
	for _, source := range row.Item.SourceRefs {
		normalized := cleanHermesRel(source)
		if normalized != "" && skillCatalogFamilyMatchesPath(def, normalized) {
			reasons = append(reasons, "source_ref:"+normalized)
		}
	}
	if containsKeyword(row.Text, def.Keywords) {
		reasons = append(reasons, "taxonomy_keyword")
	}
	return uniqueSorted(reasons)
}

func containsAny(text string, fragments []string) bool {
	for _, fragment := range fragments {
		if fragment != "" && strings.Contains(text, fragment) {
			return true
		}
	}
	return false
}

func sortSourcePairEvidence(values []SourcePairEvidence) []SourcePairEvidence {
	sort.Slice(values, func(i, j int) bool { return values[i].HermesFile < values[j].HermesFile })
	return values
}

func sortProgressRowEvidence(values []ProgressRowEvidence) []ProgressRowEvidence {
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return values
}

func cleanHermesRel(ref string) string {
	ref = filepath.ToSlash(strings.TrimSpace(ref))
	ref = strings.TrimPrefix(ref, "./")
	for _, prefix := range []string{"hermes-agent/", "references/hermes-agent/"} {
		ref = strings.TrimPrefix(ref, prefix)
	}
	return ref
}

func candidateSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py", ".ts", ".tsx", ".js", ".jsx", ".md":
		return true
	default:
		return false
	}
}

func ignoredInventoryDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", "__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", "node_modules", "dist", "build", "coverage", ".venv", "venv":
		return true
	default:
		return false
	}
}

func isUpstreamSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py", ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}

func isUpstreamDocsFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".mdx":
		return true
	default:
		return false
	}
}

func isUpstreamTestFile(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	return strings.Contains(lower, "/tests/") ||
		strings.HasPrefix(lower, "tests/") ||
		strings.Contains(lower, "/test/") ||
		strings.HasPrefix(base, "test_") ||
		strings.HasSuffix(base, "_test.py") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.tsx")
}

func isHermesReleaseCheckpointFile(path string) bool {
	base := filepath.Base(filepath.ToSlash(path))
	return strings.HasPrefix(base, "RELEASE_v") && strings.HasSuffix(base, ".md")
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
