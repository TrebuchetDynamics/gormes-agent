package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/evidence"

type VectorDivergenceStatus = evidence.VectorDivergenceStatus

const (
	VectorDivergenceOwned    = evidence.VectorDivergenceOwned
	VectorDivergenceExcluded = evidence.VectorDivergenceExcluded
)

type VectorDivergenceRow = evidence.VectorDivergenceRow

func VectorStoreDivergenceRows() []VectorDivergenceRow {
	return evidence.VectorStoreDivergenceRows()
}

func ValidateVectorStoreDivergence(rows []VectorDivergenceRow) error {
	return evidence.ValidateVectorStoreDivergence(rows)
}
