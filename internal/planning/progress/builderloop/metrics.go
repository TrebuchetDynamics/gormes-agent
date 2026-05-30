package builderloop

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/metrics"
)

type RowKind = metrics.RowKind

const (
	RowKindSelfImprovement = metrics.RowKindSelfImprovement
	RowKindUserFeature     = metrics.RowKindUserFeature
	RowKindUnclassified    = metrics.RowKindUnclassified
)

type ShippedRowEvent = metrics.ShippedRowEvent

type ShipRatio = metrics.ShipRatio

func ClassifySubphase(subphaseID string) RowKind {
	return metrics.ClassifySubphase(subphaseID)
}

func ComputeShipRatio(events []ShippedRowEvent, window time.Duration, now time.Time) ShipRatio {
	return metrics.ComputeShipRatio(events, window, now)
}
