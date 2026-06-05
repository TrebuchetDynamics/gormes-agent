package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/blockers"

type BlockerType = blockers.BlockerType

const (
	BlockerTypeAccess     BlockerType = blockers.BlockerTypeAccess
	BlockerTypeInfra      BlockerType = blockers.BlockerTypeInfra
	BlockerTypeDependency BlockerType = blockers.BlockerTypeDependency
	BlockerTypeDecision   BlockerType = blockers.BlockerTypeDecision
	BlockerTypeBug        BlockerType = blockers.BlockerTypeBug
	BlockerTypeUnknown    BlockerType = blockers.BlockerTypeUnknown
)

type BlockerStatus = blockers.BlockerStatus

const (
	BlockerStatusActive       BlockerStatus = blockers.BlockerStatusActive
	BlockerStatusUnclassified BlockerStatus = blockers.BlockerStatusUnclassified
)

type BlockerRecord = blockers.BlockerRecord

func NormalizeBlockerRecord(record BlockerRecord) BlockerRecord {
	return blockers.NormalizeBlockerRecord(record)
}

func FormatBlockerRecord(record BlockerRecord) string {
	return blockers.FormatBlockerRecord(record)
}

func SelectBlockerPivot(records []BlockerRecord) string {
	return blockers.SelectBlockerPivot(records)
}
