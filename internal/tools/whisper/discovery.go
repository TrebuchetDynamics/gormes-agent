package whisper

import (
	"context"

	discoverypkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/discovery"
)

const (
	ArtifactSource    = discoverypkg.ArtifactSource
	ArtifactCommit    = discoverypkg.ArtifactCommit
	ArtifactSHA256    = discoverypkg.ArtifactSHA256
	ArtifactSizeBytes = discoverypkg.ArtifactSizeBytes
)

type Import = discoverypkg.Import
type Discovery = discoverypkg.Discovery

func Inspect(ctx context.Context, wasm []byte) (Discovery, error) {
	return discoverypkg.Inspect(ctx, wasm)
}

func InstantiateForDiscovery(ctx context.Context, wasm []byte) (Discovery, error) {
	return discoverypkg.InstantiateForDiscovery(ctx, wasm)
}
