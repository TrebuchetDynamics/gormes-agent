package modelcache

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/speech/artifact"
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

var whisperModelArtifacts = map[string]ModelArtifact{
	"tiny.en": TinyEnModelArtifact,
	"tiny": {
		Filename:  "ggml-tiny.bin",
		URL:       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
		SizeBytes: 77691713,
		SHA256:    "be07e048e1e599ad46341c8d2a135645097a538221678b7acdd1b1919c6e1b21",
	},
	"base.en": {
		Filename:  "ggml-base.en.bin",
		URL:       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
		SizeBytes: 147964211,
		SHA256:    "a03779c86df3323075f5e796cb2ce5029f00ec8869eee3fdfb897afe36c6d002",
	},
	"base": {
		Filename:  "ggml-base.bin",
		URL:       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
		SizeBytes: 147951465,
		SHA256:    "60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe",
	},
	"small.en": {
		Filename:  "ggml-small.en.bin",
		URL:       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin",
		SizeBytes: 487614201,
		SHA256:    "c6138d6d58ecc8322097e0f987c32f1be8bb0a18532a3f88f734d1bbf9c41e5d",
	},
	"small": {
		Filename:  "ggml-small.bin",
		URL:       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin",
		SizeBytes: 487601967,
		SHA256:    "1be3a9b2063867b937e64e2ec7483364a79917e157fa98c5d94b5c1fffea987b",
	},
}

func ResolveModelArtifact(model, language string) (string, ModelArtifact) {
	key := normalizeModelName(model)
	if key == "" || key == "auto" {
		key = "base"
	}
	if artifact, ok := whisperModelArtifacts[key]; ok {
		return key, artifact
	}
	return "base", whisperModelArtifacts["base"]
}

func normalizeModelName(model string) string {
	key := strings.ToLower(strings.TrimSpace(model))
	key = strings.TrimPrefix(key, "ggml-")
	key = strings.TrimSuffix(key, ".bin")
	return key
}

func Ensure(ctx context.Context, model ModelArtifact, cacheDir string, client *http.Client) (string, error) {
	path, err := artifact.Ensure(ctx, model, cacheDir, client)
	if err != nil {
		return "", cacheError(err)
	}
	return path, nil
}

func Verify(path string, model ModelArtifact) error {
	return cacheError(artifact.Verify(path, model))
}

func cacheError(err error) error {
	if err == nil {
		return nil
	}
	var cacheErr *artifact.CacheError
	if !errors.As(err, &cacheErr) {
		return err
	}
	return &ModelCacheError{
		Code: mapCacheCode(cacheErr.Code),
		Path: cacheErr.Path,
		URL:  cacheErr.URL,
		Err:  cacheErr.Err,
	}
}

func mapCacheCode(code string) string {
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
