package plannerloop

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/plannerloop/implinventory"
)

type ImplInventory = implinventory.ImplInventory

var DefaultGormesOriginalPaths = implinventory.DefaultGormesOriginalPaths

func ScanImplementation(repoRoot string, gormesOriginalPaths []string, lookback time.Duration, now time.Time) (ImplInventory, error) {
	return implinventory.ScanImplementation(repoRoot, gormesOriginalPaths, lookback, now)
}

func computeOwnedSubphases(prog *progress.Progress, gormesOriginalPaths []string) []string {
	return implinventory.ComputeOwnedSubphases(prog, gormesOriginalPaths)
}
