// Package progressctl owns the regeneration logic for the building-gormes
// progress control plane: validating progress.json, rewriting the markered docs
// (README rollup, agent queue, blocked slices, etc.), and mirroring
// progress.json into the www.gormes.ai data directory. It is exposed through
// cmd/progress for skill-driven planner and builder passes.
package progressctl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/builderloop"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/workspace"
)

type validateReport struct {
	OK        bool       `json:"ok"`
	Phases    int        `json:"phases"`
	Subphases countsJSON `json:"subphases"`
	Items     countsJSON `json:"items"`
}

type countsJSON struct {
	Total      int `json:"total"`
	Complete   int `json:"complete"`
	InProgress int `json:"in_progress"`
	Planned    int `json:"planned"`
}

// ListOptions controls the read-only progress inventory view.
type ListOptions struct {
	Module string
}

// NextWorkOptions controls the read-only next-work selector.
type NextWorkOptions struct {
	// RepoOnly filters candidates whose write_scope resolves outside root.
	RepoOnly bool
}

type moduleListRow struct {
	PhaseID    string
	SubphaseID string
	Item       progress.Item
}

func toCountsJSON(c progress.Counts) countsJSON {
	return countsJSON{
		Total:      c.Total,
		Complete:   c.Complete,
		InProgress: c.InProgress,
		Planned:    c.Planned,
	}
}

// List emits a read-only inventory view over the single logical backlog. The
// first supported scope is exactly one module, so planner/builder agents can
// choose a feature boundary without creating independent queues.
func List(stdout io.Writer, root string, opts ListOptions) error {
	module := strings.TrimSpace(opts.Module)
	if module == "" {
		return fmt.Errorf("progress: list requires --module <module>")
	}
	if strings.Contains(module, ",") {
		return fmt.Errorf("progress: --module accepts exactly one module; comma-separated module filters are not supported")
	}
	if !progress.ValidModule(module) {
		return fmt.Errorf("progress: unknown module %q (allowed: %s)", module, strings.Join(progress.AllowedModules(), ", "))
	}

	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	rows := rowsForModule(p, module)
	rowNoun := "rows"
	if len(rows) == 1 {
		rowNoun = "row"
	}
	if _, err := fmt.Fprintf(stdout, "progress: module %s (%d %s)\n", module, len(rows), rowNoun); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "phase\tsubphase\tstatus\tpriority\tname"); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			row.PhaseID, row.SubphaseID, row.Item.Status, row.Item.Priority, row.Item.Name); err != nil {
			return err
		}
	}
	return nil
}

// NextWork emits the single next action over the canonical backlog without
// mutating it. It reuses builderloop.NormalizeCandidates so command users and
// autonomous agents see the same ranking as the builder loop instead of
// reimplementing selection policy from generated markdown.
func NextWork(stdout io.Writer, root string) error {
	return NextWorkWithOptions(stdout, root, NextWorkOptions{})
}

// NextWorkWithOptions emits the single next action over the canonical backlog
// without mutating it. It reuses builderloop.NormalizeCandidates for ranking
// and applies command-level filters only after the canonical ordering exists.
func NextWorkWithOptions(stdout io.Writer, root string, opts NextWorkOptions) error {
	candidates, err := builderloop.NormalizeCandidates(canonicalSource(root), builderloop.CandidateOptions{ActiveFirst: true})
	if err != nil {
		return err
	}
	if opts.RepoOnly {
		candidates, err = filterRepoScopedCandidates(root, candidates)
		if err != nil {
			return err
		}
	}
	if len(candidates) == 0 {
		return printNoNextWork(stdout, opts)
	}

	noun := "candidates"
	if len(candidates) == 1 {
		noun = "candidate"
	}
	top := candidates[0]
	if _, err := fmt.Fprintf(stdout, "progress: next-work builder-ready (%d %s)\n", len(candidates), noun); err != nil {
		return err
	}
	fields := []struct {
		key   string
		value string
	}{
		{key: "decision", value: "build"},
	}
	if opts.RepoOnly {
		fields = append(fields, struct {
			key   string
			value string
		}{key: "scope", value: "repo"})
	}
	fields = append(fields, []struct {
		key   string
		value string
	}{
		{key: "phase", value: top.PhaseID},
		{key: "subphase", value: top.SubphaseID},
		{key: "name", value: top.ItemName},
		{key: "reason", value: top.SelectionReason()},
		{key: "priority", value: top.Priority},
		{key: "status", value: top.Status},
		{key: "contract_status", value: top.ContractStatus},
		{key: "slice_size", value: top.SliceSize},
		{key: "owner", value: top.ExecutionOwner},
	}...)
	for _, field := range fields {
		if _, err := fmt.Fprintf(stdout, "%s=%s\n", field.key, field.value); err != nil {
			return err
		}
	}
	return nil
}

func printNoNextWork(stdout io.Writer, opts NextWorkOptions) error {
	if opts.RepoOnly {
		if _, err := fmt.Fprintln(stdout, "progress: next-work no in-repo builder-ready rows"); err != nil {
			return err
		}
		for _, line := range []string{
			"decision=plan",
			"scope=repo",
			"reason=no unblocked builder-ready rows within repo write scope",
			"planner_action=split or repair one row whose write_scope stays under the repo root",
		} {
			if _, err := fmt.Fprintln(stdout, line); err != nil {
				return err
			}
		}
		return nil
	}

	if _, err := fmt.Fprintln(stdout, "progress: next-work no builder-ready rows"); err != nil {
		return err
	}
	for _, line := range []string{
		"decision=plan",
		"reason=no unblocked builder-ready rows",
		"planner_action=repair one planned/draft row until it satisfies the handoff contract",
	} {
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}

func filterRepoScopedCandidates(root string, candidates []builderloop.Candidate) ([]builderloop.Candidate, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootAbs = filepath.Clean(rootAbs)
	out := make([]builderloop.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidateWriteScopeWithinRoot(rootAbs, candidate.WriteScope) {
			out = append(out, candidate)
		}
	}
	return out, nil
}

func candidateWriteScopeWithinRoot(rootAbs string, scopes []string) bool {
	if len(scopes) == 0 {
		return false
	}
	for _, scope := range scopes {
		if !writeScopePathWithinRoot(rootAbs, scope) {
			return false
		}
	}
	return true
}

func writeScopePathWithinRoot(rootAbs, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	lower := strings.ToLower(scope)
	if strings.Contains(lower, "separate repo") || strings.Contains(lower, "external repo") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	candidate := scope
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootAbs, candidate)
	}
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(rootAbs, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func rowsForModule(p *progress.Progress, module string) []moduleListRow {
	if p == nil {
		return nil
	}
	var rows []moduleListRow
	for _, phaseID := range roadmapKeys(p.Phases) {
		phase := p.Phases[phaseID]
		for _, subphaseID := range roadmapKeys(phase.Subphases) {
			subphase := phase.Subphases[subphaseID]
			for _, item := range subphase.Items {
				if progress.Module(item, phaseID, subphaseID) != module {
					continue
				}
				rows = append(rows, moduleListRow{
					PhaseID: phaseID, SubphaseID: subphaseID, Item: item,
				})
			}
		}
	}
	return rows
}

func roadmapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareRoadmapKeys(keys[i], keys[j]) < 0
	})
	return keys
}

func compareRoadmapKeys(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if diff := compareRoadmapPart(aParts[i], bParts[i]); diff != 0 {
			return diff
		}
	}
	switch {
	case len(aParts) < len(bParts):
		return -1
	case len(aParts) > len(bParts):
		return 1
	default:
		return 0
	}
}

func compareRoadmapPart(a, b string) int {
	aNum, aErr := strconv.Atoi(a)
	bNum, bErr := strconv.Atoi(b)
	switch {
	case aErr == nil && bErr == nil:
		switch {
		case aNum < bNum:
			return -1
		case aNum > bNum:
			return 1
		default:
			return 0
		}
	case aErr == nil:
		return -1
	case bErr == nil:
		return 1
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Validate parses progress.json under root, runs progress.Validate, and emits
// a single line summarizing the phase count. format may be "text" (default)
// or "json".
func Validate(stdout io.Writer, root, format string) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	if format == "json" {
		stats := p.Stats()
		return json.NewEncoder(stdout).Encode(validateReport{
			OK:        true,
			Phases:    len(p.Phases),
			Subphases: toCountsJSON(stats.Subphases),
			Items:     toCountsJSON(stats.Items),
		})
	}
	_, err = fmt.Fprintf(stdout, "progress: validated %d phases\n", len(p.Phases))
	return err
}

// Write regenerates every markered section in the docs tree from
// progress.json, writes generated module roadmap pages, and mirrors the JSON
// into the www.gormes.ai data directories.
// Errors from individual rewrites are collected, surfaced one per line, and
// returned via errors.Join so the caller fails the whole run while still
// telling the operator which markers updated and which did not.
func Write(stdout io.Writer, root string) error {
	return WriteWithOptions(stdout, root, WriteOptions{})
}

// Compact rewrites every completed row's verbose shipped-evidence note to a
// one-line "SHIPPED <date> see git log — <summary>" pointer and writes the
// canonical progress.json back through the same stable marshaller Write uses,
// so only `note` strings change in the diff. It is deliberately a standalone
// maintenance action: Write and Validate never invoke it, keeping doc
// regeneration and schema validation pure. It is idempotent and fully
// reversible via git.
func Compact(stdout io.Writer, root string) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	n := progress.CompactCompletedNotes(p)
	if n == 0 {
		_, err = fmt.Fprintln(stdout, "progress: notes already compact (no changes)")
		return err
	}
	ws := workspace.New(root)
	if err := ws.Save(p); err != nil {
		return err
	}
	if err := progress.Validate(p); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "progress: compacted %d completed-row note(s)\n", n)
	return err
}

// Split writes the canonical backlog to destDir as a split layout
// (index.json + phases/<id>.json) without touching the canonical
// progress.json. It is a standalone, explicit, read-only-against-canonical
// maintenance action: Validate, Write, and Compact never invoke it, and it
// moves no rows. internal/progress.Load already reads such a directory back
// into the identical model (dual-layout transparency), so this is the
// operator entry point for the module-split umbrella's later children. The
// inverse write-back ("flip the on-disk layout") is intentionally deferred
// to a later child so this shim stays non-behavior-changing.
func Split(stdout io.Writer, root, destDir string) error {
	if destDir == "" {
		return fmt.Errorf("progress: split requires a destination directory")
	}
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	if err := progress.WriteSplit(destDir, p); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "progress: split %d phases into %s\n", len(p.Phases), destDir)
	return err
}

// Emit writes the merged canonical backlog as stable JSON to stdout. It is a
// pure, read-only, non-default action: it loads through loadValidProgress
// (canonicalSource + internal/progress.Load, dual-layout transparent) and
// never mutates the canonical. It is the split-safe seam gormes-* skill
// discovery pipes through instead of jq-ing the canonical path directly —
// `go run ./cmd/progress emit | jq ...` keeps working byte-identically
// whether the canonical is a monolithic file or a (phase- or module-keyed)
// split directory. Bytes are produced by the shipped stable encoder
// (progress.SaveProgress) so emit stays faithful to the on-disk monolith.
func Emit(stdout io.Writer, root string) error {
	raw, err := workspace.New(root).EmitBytes()
	if err != nil {
		return err
	}
	_, err = stdout.Write(raw)
	return err
}

type pathSet struct {
	progressJSON       string
	readme             string
	docsIndex          string
	contractReadiness  string
	builderLoopHandoff string
	agentQueue         string
	nextSlices         string
	blockedSlices      string
	umbrellaCleanup    string
	progressSchema     string
	moduleRoadmapsDir  string
	siteProgress       []string
	siteProgressSlim   string
}

func progressPaths(root string) pathSet {
	paths := workspace.New(root).Paths
	return pathSet{
		progressJSON:       paths.ProgressJSON,
		readme:             paths.Readme,
		docsIndex:          paths.DocsIndex,
		contractReadiness:  paths.ContractReadiness,
		builderLoopHandoff: paths.BuilderLoopHandoff,
		agentQueue:         paths.AgentQueue,
		nextSlices:         paths.NextSlices,
		blockedSlices:      paths.BlockedSlices,
		umbrellaCleanup:    paths.UmbrellaCleanup,
		progressSchema:     paths.ProgressSchema,
		moduleRoadmapsDir:  paths.ModuleRoadmapsDir,
		siteProgress:       paths.SiteProgress,
		siteProgressSlim:   paths.SiteProgressSlim,
	}
}

// canonicalSource returns the single logical backlog path. The path may be a
// monolithic file or a split directory; the historical progress.split staging
// directory is no longer part of canonical resolution.
func canonicalSource(root string) string {
	return workspace.New(root).CanonicalSource()
}

func loadValidProgress(root string) (*progress.Progress, error) {
	return workspace.New(root).LoadValid()
}

func rewriteMarker(path, kind, body string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	out, err := progress.ReplaceMarker(string(b), kind, body)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// slimProgress returns a reduced copy of p containing exactly what the
// landing renderer reads — phase/subphase names, deliverable, dependency
// note, subphase priority/status/drift, and per-item Status — and nothing
// else. All per-item prose (name, contract, notes, acceptance, source refs,
// write scope, ...) is dropped. DerivedStatus and Stats are byte-for-byte
// equivalent because they only depend on the preserved fields.
func slimProgress(p *progress.Progress) *progress.Progress {
	if p == nil {
		return nil
	}
	out := &progress.Progress{Meta: p.Meta, Phases: make(map[string]progress.Phase, len(p.Phases))}
	for pk, ph := range p.Phases {
		sps := make(map[string]progress.Subphase, len(ph.Subphases))
		for sk, sp := range ph.Subphases {
			var items []progress.Item
			if len(sp.Items) > 0 {
				items = make([]progress.Item, 0, len(sp.Items))
				for _, it := range sp.Items {
					items = append(items, progress.Item{Status: it.Status})
				}
			}
			sps[sk] = progress.Subphase{
				Name:       sp.Name,
				Priority:   sp.Priority,
				Items:      items,
				Status:     sp.Status,
				DriftState: sp.DriftState,
			}
		}
		out.Phases[pk] = progress.Phase{
			Name:           ph.Name,
			Deliverable:    ph.Deliverable,
			DependencyNote: ph.DependencyNote,
			Subphases:      sps,
		}
	}
	return out
}

// writeSlimProgress marshals slimProgress(p) to dst. The legacy go-renderer
// go:embed's this path, so it must always exist and be valid JSON, but it is
// now KB (status/name only) instead of a 5.2 MB verbatim archive copy.
func writeSlimProgress(p *progress.Progress, dst string) error {
	b, err := json.MarshalIndent(slimProgress(p), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal slim progress: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func joinErrors(stdout io.Writer, errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	for _, err := range errs {
		fmt.Fprintln(stdout, "progress:", err)
	}
	return errors.Join(errs...)
}
