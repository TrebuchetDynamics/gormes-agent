package plannerloop

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/plannerloop/driftstate"
)

func diffSubphaseStates(before, after *progress.Progress) []DriftPromotion {
	return driftstate.DiffSubphaseStates(before, after)
}

func driftStatusOrPorting(sub progress.Subphase) string {
	return driftstate.DriftStatusOrPorting(sub)
}
