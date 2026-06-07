// Package progressctl owns the regeneration logic for the building-gormes
// progress control plane: validating progress.json, rewriting the markered docs
// (README rollup, agent queue, blocked slices, etc.), and mirroring
// progress.json into the www.gormes.ai data directory. It is exposed through
// cmd/progress for skill-driven planner and builder passes.
package progressctl

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/ctl/inventory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/ctl/nextwork"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/ctl/siteprogress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/workspace"
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
type ListOptions = inventory.ListOptions

// NextWorkOptions controls the read-only next-work selector.
type NextWorkOptions = nextwork.Options

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
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	return inventory.List(stdout, p, opts)
}

// NextWork emits the single next action over the canonical backlog without
// mutating it. It uses the progress active-handoff projection so command users
// and generated docs share one row classification and ordering seam.
func NextWork(stdout io.Writer, root string) error {
	return NextWorkWithOptions(stdout, root, NextWorkOptions{})
}

// NextWorkWithOptions emits the single next action over the canonical backlog
// without mutating it. Command-level filters apply after canonical ordering.
func NextWorkWithOptions(stdout io.Writer, root string, opts NextWorkOptions) error {
	p, err := progress.Load(canonicalSource(root))
	if err != nil {
		return err
	}
	return nextwork.NextWorkWithOptions(stdout, root, p, opts)
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

// slimProgress returns a reduced copy of p containing exactly what the
// landing renderer reads — phase/subphase names, deliverable, dependency
// note, subphase priority/status/drift, and per-item Status — and nothing
// else. All per-item prose (name, contract, notes, acceptance, source refs,
// write scope, ...) is dropped. DerivedStatus and Stats are byte-for-byte
// equivalent because they only depend on the preserved fields.
func slimProgress(p *progress.Progress) *progress.Progress {
	return siteprogress.Slim(p)
}

// writeSlimProgress marshals slimProgress(p) to dst. The legacy go-renderer
// go:embed's this path, so it must always exist and be valid JSON, but it is
// now KB (status/name only) instead of a 5.2 MB verbatim archive copy.
func writeSlimProgress(p *progress.Progress, dst string) error {
	return siteprogress.Write(p, dst)
}
