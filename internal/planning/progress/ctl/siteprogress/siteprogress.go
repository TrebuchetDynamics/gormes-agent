// Package siteprogress owns the reduced progress JSON projection consumed by
// static site mirrors. It must stay pure except for writing its explicit output
// file.
package siteprogress

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// Slim returns a reduced copy of p containing exactly what the landing renderer
// reads: phase/subphase names, deliverable, dependency note, subphase
// priority/status/drift, and per-item Status.
func Slim(p *progress.Progress) *progress.Progress {
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

// Write marshals Slim(p) to dst.
func Write(p *progress.Progress, dst string) error {
	b, err := json.MarshalIndent(Slim(p), "", "  ")
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
