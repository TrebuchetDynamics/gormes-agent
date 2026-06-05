package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/ranking"

// RRFFuseIDs merges two ranked ID lists using Reciprocal Rank Fusion.
// Higher fused scores rank first. Duplicate IDs across sources get boosted.
func RRFFuseIDs(ftsIDs, semIDs []int64, k, ftsWeight, semWeight float64) []int64 {
	return ranking.FuseIDs(ftsIDs, semIDs, k, ftsWeight, semWeight)
}
