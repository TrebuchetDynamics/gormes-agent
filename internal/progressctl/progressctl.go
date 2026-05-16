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

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
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

func toCountsJSON(c progress.Counts) countsJSON {
	return countsJSON{
		Total:      c.Total,
		Complete:   c.Complete,
		InProgress: c.InProgress,
		Planned:    c.Planned,
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
// progress.json and mirrors the JSON into the www.gormes.ai data directories.
// Errors from individual rewrites are collected, surfaced one per line, and
// returned via errors.Join so the caller fails the whole run while still
// telling the operator which markers updated and which did not.
func Write(stdout io.Writer, root string) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	paths := progressPaths(root)

	markers := []marker{
		{pathOf: func(s pathSet) string { return s.docsIndex }, kind: "docs-full-checklist", label: "_index.md regenerated", render: progress.RenderDocsChecklist},
		{pathOf: func(s pathSet) string { return s.readme }, kind: "readme-rollup", label: "README.md regenerated", render: progress.RenderReadmeRollup},
		{pathOf: func(s pathSet) string { return s.contractReadiness }, kind: "contract-readiness", label: "contract readiness regenerated", render: progress.RenderContractReadiness},
		{pathOf: func(s pathSet) string { return s.builderLoopHandoff }, kind: "builder-loop-handoff", label: "builder-loop handoff regenerated", render: progress.RenderBuilderLoopHandoff},
		{pathOf: func(s pathSet) string { return s.agentQueue }, kind: "agent-queue", label: "agent queue regenerated", render: func(p *progress.Progress) string { return progress.RenderAgentQueue(p, 10) }},
		{pathOf: func(s pathSet) string { return s.nextSlices }, kind: "next-slices", label: "next slices regenerated", render: func(p *progress.Progress) string { return progress.RenderNextSlices(p, 10) }},
		{pathOf: func(s pathSet) string { return s.blockedSlices }, kind: "blocked-slices", label: "blocked slices regenerated", render: progress.RenderBlockedSlices},
		{pathOf: func(s pathSet) string { return s.umbrellaCleanup }, kind: "umbrella-cleanup", label: "umbrella cleanup regenerated", render: progress.RenderUmbrellaCleanup},
		{pathOf: func(s pathSet) string { return s.progressSchema }, kind: "progress-schema", label: "progress schema regenerated", render: func(*progress.Progress) string { return progress.RenderProgressSchema() }},
	}

	var errs []error
	for _, m := range markers {
		if err := rewriteMarker(m.pathOf(paths), m.kind, m.render(p)); err != nil {
			errs = append(errs, err)
		} else {
			fmt.Fprintln(stdout, "progress:", m.label)
		}
	}
	for _, siteProgress := range paths.siteProgress {
		if err := syncFile(paths.progressJSON, siteProgress); err != nil {
			errs = append(errs, err)
		}
	}
	if paths.siteProgressSlim != "" {
		if err := writeSlimProgress(p, paths.siteProgressSlim); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		fmt.Fprintln(stdout, "progress: site progress data refreshed")
	}
	return joinErrors(stdout, errs)
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
	// Compact reads layout-agnostically (C2) but its write-back targets the
	// monolithic file. Collapsing an active split layout into the monolith is
	// the split-aware write path owned by C3; refuse rather than silently
	// diverging the two layouts.
	if canonicalSource(root) != progressPaths(root).progressJSON {
		return fmt.Errorf("progress: compact on a split layout is deferred to backlog-split C3 (split-aware write path); not collapsing the split into the monolith")
	}
	path := progressPaths(root).progressJSON
	if err := progress.SaveProgress(path, p); err != nil {
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
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "progress-emit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmp := filepath.Join(tmpDir, "progress.json")
	if err := progress.SaveProgress(tmp, p); err != nil {
		return err
	}
	raw, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	_, err = stdout.Write(raw)
	return err
}

type marker struct {
	pathOf func(pathSet) string
	kind   string
	label  string
	render func(*progress.Progress) string
}

type pathSet struct {
	progressJSON       string
	progressSplitDir   string
	readme             string
	docsIndex          string
	contractReadiness  string
	builderLoopHandoff string
	agentQueue         string
	nextSlices         string
	blockedSlices      string
	umbrellaCleanup    string
	progressSchema     string
	siteProgress       []string
	siteProgressSlim   string
}

func progressPaths(root string) pathSet {
	buildingGormes := filepath.Join(root, "webpages", "docs", "content", "building-gormes")
	builderLoopDir := filepath.Join(buildingGormes, "builder-loop")
	return pathSet{
		progressJSON:       filepath.Join(buildingGormes, "architecture_plan", "progress.json"),
		progressSplitDir:   filepath.Join(buildingGormes, "architecture_plan", "progress.split"),
		readme:             filepath.Join(root, "README.md"),
		docsIndex:          filepath.Join(buildingGormes, "architecture_plan", "_index.md"),
		contractReadiness:  filepath.Join(buildingGormes, "contract-readiness.md"),
		builderLoopHandoff: filepath.Join(builderLoopDir, "builder-loop-handoff.md"),
		agentQueue:         filepath.Join(builderLoopDir, "agent-queue.md"),
		nextSlices:         filepath.Join(builderLoopDir, "next-slices.md"),
		blockedSlices:      filepath.Join(builderLoopDir, "blocked-slices.md"),
		umbrellaCleanup:    filepath.Join(builderLoopDir, "umbrella-cleanup.md"),
		progressSchema:     filepath.Join(builderLoopDir, "progress-schema.md"),
		// Verbatim site mirrors: now empty. The dead
		// webpages/landing/src/data/progress.json mirror had no consumer
		// (nothing in the Astro site imports it) and is no longer
		// generated or tracked. Backlog-efficiency #1, 2026-05-16.
		siteProgress: nil,
		// The legacy go-renderer go:embed mirror MUST exist at build time
		// (//go:embed data/progress.json). It is regenerated SLIM
		// (phase/subphase names + statuses only — everything the renderer
		// reads, none of the per-item prose) so it stays a valid embed
		// without duplicating the 5.2 MB archive on every progress edit.
		siteProgressSlim: filepath.Join(root, "webpages", "landing", "legacy", "go-renderer", "internal", "site", "data", "progress.json"),
	}
}

// canonicalSource resolves which on-disk layout backs the canonical backlog:
// the split directory when it is present (opt-in, auto-detected), otherwise
// the monolithic progress.json. Because the split directory does not exist by
// default, the monolithic path — and therefore every generator's behavior —
// is byte-for-byte unchanged until an operator materializes the split layout.
// internal/progress.Load already accepts either a file or a directory (C1),
// so callers stay layout-agnostic and a malformed split surfaces
// progress.ErrMalformedSplit instead of a half-generated doc set.
func canonicalSource(root string) string {
	paths := progressPaths(root)
	if fi, err := os.Stat(paths.progressSplitDir); err == nil && fi.IsDir() {
		return paths.progressSplitDir
	}
	return paths.progressJSON
}

func loadValidProgress(root string) (*progress.Progress, error) {
	p, err := progress.Load(canonicalSource(root))
	if err != nil {
		return nil, err
	}
	if err := progress.Validate(p); err != nil {
		return nil, err
	}
	return p, nil
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

func syncFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
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
