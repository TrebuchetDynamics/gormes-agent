package whisper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

type ModelArtifact struct {
	Filename  string
	URL       string
	SizeBytes int64
	SHA256    string
}

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

func EnsureModel(ctx context.Context, artifact ModelArtifact, cacheDir string, client *http.Client) (string, error) {
	if strings.TrimSpace(cacheDir) == "" {
		return "", &ModelCacheError{Code: ModelCacheInvalidCacheDir, Err: fmt.Errorf("cache directory is required")}
	}
	if err := validateModelArtifact(artifact); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", &ModelCacheError{Code: ModelCacheInvalidCacheDir, Path: cacheDir, Err: err}
	}
	finalPath := filepath.Join(cacheDir, artifact.Filename)
	if err := verifyModelFile(finalPath, artifact); err == nil {
		return finalPath, nil
	} else if !os.IsNotExist(err) {
		if removeErr := os.Remove(finalPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return "", &ModelCacheError{Code: ModelCacheWriteFailed, Path: finalPath, Err: removeErr}
		}
	}
	if client == nil {
		client = http.DefaultClient
	}
	if err := downloadModel(ctx, artifact, finalPath, client); err != nil {
		return "", err
	}
	return finalPath, nil
}

func validateModelArtifact(artifact ModelArtifact) error {
	filename := strings.TrimSpace(artifact.Filename)
	if filename == "" || filename == "." || filename == ".." || strings.ContainsAny(filename, `/\`) ||
		strings.TrimSpace(artifact.URL) == "" || artifact.SizeBytes <= 0 || strings.TrimSpace(artifact.SHA256) == "" {
		return &ModelCacheError{Code: ModelCacheInvalidArtifact, Err: fmt.Errorf("model artifact metadata is incomplete")}
	}
	return nil
}

func downloadModel(ctx context.Context, artifact ModelArtifact, finalPath string, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return &ModelCacheError{Code: ModelCacheInvalidArtifact, URL: artifact.URL, Err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return &ModelCacheError{Code: ModelCacheDownloadFailed, URL: artifact.URL, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &ModelCacheError{Code: ModelCacheBadStatus, URL: artifact.URL, Err: fmt.Errorf("http status %d", resp.StatusCode)}
	}

	partialPath := finalPath + ".partial"
	if err := os.Remove(partialPath); err != nil && !os.IsNotExist(err) {
		return &ModelCacheError{Code: ModelCacheWriteFailed, Path: partialPath, Err: err}
	}
	out, err := os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return &ModelCacheError{Code: ModelCacheWriteFailed, Path: partialPath, Err: err}
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(partialPath)
		return &ModelCacheError{Code: ModelCacheDownloadFailed, URL: artifact.URL, Path: partialPath, Err: copyErr}
	}
	if closeErr != nil {
		_ = os.Remove(partialPath)
		return &ModelCacheError{Code: ModelCacheWriteFailed, Path: partialPath, Err: closeErr}
	}
	if err := verifyModelFile(partialPath, artifact); err != nil {
		_ = os.Remove(partialPath)
		return err
	}
	if err := os.Rename(partialPath, finalPath); err != nil {
		_ = os.Remove(partialPath)
		return &ModelCacheError{Code: ModelCacheWriteFailed, Path: finalPath, Err: err}
	}
	return nil
}

func verifyModelFile(path string, artifact ModelArtifact) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return &ModelCacheError{Code: ModelCacheInvalidArtifact, Path: path, Err: fmt.Errorf("model path is not a regular file")}
	}
	if info.Size() != artifact.SizeBytes {
		return &ModelCacheError{Code: ModelCacheSizeMismatch, Path: path, Err: fmt.Errorf("size %d, want %d", info.Size(), artifact.SizeBytes)}
	}
	file, err := os.Open(path)
	if err != nil {
		return &ModelCacheError{Code: ModelCacheWriteFailed, Path: path, Err: err}
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return &ModelCacheError{Code: ModelCacheWriteFailed, Path: path, Err: err}
	}
	if got := hex.EncodeToString(hash.Sum(nil)); !strings.EqualFold(got, artifact.SHA256) {
		return &ModelCacheError{Code: ModelCacheChecksumMismatch, Path: path, Err: fmt.Errorf("sha256 %s, want %s", got, artifact.SHA256)}
	}
	return nil
}
