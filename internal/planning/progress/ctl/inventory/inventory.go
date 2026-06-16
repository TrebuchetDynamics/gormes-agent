// Package inventory owns read-only module inventory projection for the progress
// control CLI. It must stay pure: no filesystem writes, no doc generation, and
// no command-line parsing.
package inventory

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// ListOptions controls the read-only progress inventory view.
type ListOptions struct {
	Module string
}

type moduleListRow struct {
	PhaseID    string
	SubphaseID string
	Item       progress.Item
}

// List emits a read-only inventory view over the single logical backlog. The
// first supported scope is exactly one module, so planner/builder agents can
// choose a feature boundary without creating independent queues.
func List(stdout io.Writer, p *progress.Progress, opts ListOptions) error {
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
