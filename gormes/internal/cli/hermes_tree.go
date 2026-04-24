package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// HermesCLIPortStatus is a single enumerated port state for one
// upstream hermes_cli/*.py file. See testdata/hermes_cli_tree.json for
// the authoritative manifest.
type HermesCLIPortStatus string

const (
	// HermesCLIPortStatusPorted means the upstream file has a Go
	// equivalent that covers the tracked scope; no follow-on work is
	// required for that file inside Phase 5.O's tracked scope.
	HermesCLIPortStatusPorted HermesCLIPortStatus = "ported"
	// HermesCLIPortStatusInProgress means the upstream file has a
	// partial Go equivalent — a per-file follow-on port is expected.
	HermesCLIPortStatusInProgress HermesCLIPortStatus = "in_progress"
	// HermesCLIPortStatusPlanned means no Go equivalent exists yet.
	HermesCLIPortStatusPlanned HermesCLIPortStatus = "planned"
	// HermesCLIPortStatusNotApplicable means the upstream file is
	// Python-specific plumbing with no Go counterpart (e.g. package
	// markers, Typer callback shims replaced by cobra idioms).
	HermesCLIPortStatusNotApplicable HermesCLIPortStatus = "not_applicable"
)

// HermesCLIFile is one entry in the Phase 5.O manifest.
type HermesCLIFile struct {
	Source string              `json:"source"`
	Status HermesCLIPortStatus `json:"status"`
	Go     []string            `json:"go,omitempty"`
	Reason string              `json:"reason,omitempty"`
}

// HermesCLITree is the full manifest — one entry per upstream hermes_cli/*.py.
type HermesCLITree struct {
	Comment string          `json:"comment,omitempty"`
	Files   []HermesCLIFile `json:"files"`
}

// HermesCLISummary reports aggregate counts for operator dashboards.
type HermesCLISummary struct {
	Total         int
	Ported        int
	InProgress    int
	Planned       int
	NotApplicable int
}

// LoadHermesCLITree decodes the manifest at the given path.
func LoadHermesCLITree(path string) (HermesCLITree, error) {
	var out HermesCLITree
	b, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("read hermes cli manifest %q: %w", path, err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("parse hermes cli manifest %q: %w", path, err)
	}
	return out, nil
}

// Summary returns bucketed port-status counts.
func (t HermesCLITree) Summary() HermesCLISummary {
	s := HermesCLISummary{Total: len(t.Files)}
	for _, entry := range t.Files {
		switch entry.Status {
		case HermesCLIPortStatusPorted:
			s.Ported++
		case HermesCLIPortStatusInProgress:
			s.InProgress++
		case HermesCLIPortStatusPlanned:
			s.Planned++
		case HermesCLIPortStatusNotApplicable:
			s.NotApplicable++
		}
	}
	return s
}
