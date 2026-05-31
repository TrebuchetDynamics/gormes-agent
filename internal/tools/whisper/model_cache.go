package whisper

import (
	"context"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/modelcache"
)

const (
	ModelCacheInvalidArtifact  = modelcache.ModelCacheInvalidArtifact
	ModelCacheInvalidCacheDir  = modelcache.ModelCacheInvalidCacheDir
	ModelCacheBadStatus        = modelcache.ModelCacheBadStatus
	ModelCacheDownloadFailed   = modelcache.ModelCacheDownloadFailed
	ModelCacheSizeMismatch     = modelcache.ModelCacheSizeMismatch
	ModelCacheChecksumMismatch = modelcache.ModelCacheChecksumMismatch
	ModelCacheWriteFailed      = modelcache.ModelCacheWriteFailed
)

type ModelArtifact = modelcache.ModelArtifact
type ModelCacheError = modelcache.ModelCacheError

var TinyEnModelArtifact = modelcache.TinyEnModelArtifact

func EnsureModel(ctx context.Context, model ModelArtifact, cacheDir string, client *http.Client) (string, error) {
	return modelcache.Ensure(ctx, model, cacheDir, client)
}

func verifyModelFile(path string, model ModelArtifact) error {
	return modelcache.Verify(path, model)
}
