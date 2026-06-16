package repoctl

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl/sourcepairs"

type SourcePairOptions = sourcepairs.SourcePairOptions
type SourcePairsManifest = sourcepairs.SourcePairsManifest
type SourcePair = sourcepairs.SourcePair
type SourcePairsValidation = sourcepairs.SourcePairsValidation
type SourcePairsSyncResult = sourcepairs.SourcePairsSyncResult

func ValidateSourcePairs(opts SourcePairOptions) (SourcePairsValidation, error) {
	return sourcepairs.ValidateSourcePairs(opts)
}

func RenderSourcePairsReport(opts SourcePairOptions) (string, error) {
	return sourcepairs.RenderSourcePairsReport(opts)
}

func WriteSourcePairsReport(opts SourcePairOptions) error {
	return sourcepairs.WriteSourcePairsReport(opts)
}

func SyncSourcePairsSHA(opts SourcePairOptions) (SourcePairsSyncResult, error) {
	return sourcepairs.SyncSourcePairsSHA(opts)
}
