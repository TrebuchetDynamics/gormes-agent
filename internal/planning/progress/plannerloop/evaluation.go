package plannerloop

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/plannerloop/reshapeforensics"
)

const DefaultEvaluationWindow = reshapeforensics.DefaultEvaluationWindow

type ReshapeOutcome = reshapeforensics.ReshapeOutcome

func Evaluate(plannerLedgerPath, autoloopLedgerPath string, window time.Duration, now time.Time) ([]ReshapeOutcome, error) {
	return reshapeforensics.Evaluate(plannerLedgerPath, autoloopLedgerPath, window, now)
}
