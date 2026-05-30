package markdown

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/roadmaporder"
)

// FormatPriorityCounts renders priority counters in the stable order used by
// progress rollups, followed by any non-standard keys in natural order.
func FormatPriorityCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "-"
	}
	var parts []string
	for _, priority := range []string{"P0", "P1", "P2", "P3", "P4", "unset"} {
		if n := counts[priority]; n > 0 {
			parts = append(parts, fmt.Sprintf("`%s`: %d", priority, n))
		}
	}
	for _, priority := range roadmaporder.SortedMapKeys(counts) {
		switch priority {
		case "P0", "P1", "P2", "P3", "P4", "unset":
			continue
		}
		parts = append(parts, fmt.Sprintf("`%s`: %d", priority, counts[priority]))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " · ")
}
