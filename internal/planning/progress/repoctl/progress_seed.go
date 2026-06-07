package repoctl

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl/progressseed"

type ProgressSeedOptions = progressseed.ProgressSeedOptions
type ProgressSeedResult = progressseed.ProgressSeedResult

func SeedProgressRows(opts ProgressSeedOptions) (ProgressSeedResult, error) {
	return progressseed.SeedProgressRows(opts)
}
