package whisper

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/speech/artifact"
)

const (
	ModelCacheInvalidArtifact  = "model_cache_invalid_artifact"
	ModelCacheInvalidCacheDir  = "model_cache_invalid_cache_dir"
	ModelCacheBadStatus        = "model_cache_bad_status"
	ModelCacheDownloadFailed   = "model_cache_download_failed"
	ModelCacheSizeMismatch     = "model_cache_size_mismatch"
	ModelCacheChecksumMismatch = "model_cache_checksum_mismatch"
	ModelCacheWriteFailed      = "model_cache_write_failed"
)

type ModelArtifact = artifact.Artifact

type ModelCacheError struct {
	Code string
	Path string
	URL  string
	Err  error
}

func (e *ModelCacheError) Error() string {
	var parts []string
	parts = append(parts, e.Code)
	if e.Path != "" {
		parts = append(parts, "path="+e.Path)
	}
	if e.URL != "" {
		parts = append(parts, "url="+e.URL)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *ModelCacheError) Unwrap() error {
	return e.Err
}

var TinyEnModelArtifact = ModelArtifact{
	Filename:  "ggml-tiny.en.bin",
	URL:       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin",
	SizeBytes: 77704715,
	SHA256:    "921e4cf8686fdd993dcd081a5da5b6c365bfde1162e72b08d75ac75289920b1f",
}

func EnsureModel(ctx context.Context, model ModelArtifact, cacheDir string, client *http.Client) (string, error) {
	path, err := artifact.Ensure(ctx, model, cacheDir, client)
	if err != nil {
		return "", modelCacheError(err)
	}
	return path, nil
}

func verifyModelFile(path string, model ModelArtifact) error {
	return modelCacheError(artifact.Verify(path, model))
}

func modelCacheError(err error) error {
	if err == nil {
		return nil
	}
	var cacheErr *artifact.CacheError
	if !errors.As(err, &cacheErr) {
		return err
	}
	return &ModelCacheError{
		Code: mapModelCacheCode(cacheErr.Code),
		Path: cacheErr.Path,
		URL:  cacheErr.URL,
		Err:  cacheErr.Err,
	}
}

func mapModelCacheCode(code string) string {
	switch code {
	case artifact.InvalidArtifact:
		return ModelCacheInvalidArtifact
	case artifact.InvalidCacheDir:
		return ModelCacheInvalidCacheDir
	case artifact.BadStatus:
		return ModelCacheBadStatus
	case artifact.DownloadFailed:
		return ModelCacheDownloadFailed
	case artifact.SizeMismatch:
		return ModelCacheSizeMismatch
	case artifact.ChecksumMismatch:
		return ModelCacheChecksumMismatch
	case artifact.WriteFailed:
		return ModelCacheWriteFailed
	default:
		return code
	}
}
