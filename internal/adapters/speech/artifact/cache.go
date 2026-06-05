package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	InvalidArtifact  = "artifact_invalid"
	InvalidCacheDir  = "artifact_cache_dir_invalid"
	BadStatus        = "artifact_bad_status"
	DownloadFailed   = "artifact_download_failed"
	SizeMismatch     = "artifact_size_mismatch"
	ChecksumMismatch = "artifact_checksum_mismatch"
	WriteFailed      = "artifact_write_failed"
)

type Artifact struct {
	Filename  string
	URL       string
	SizeBytes int64
	SHA256    string
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type CacheError struct {
	Code string
	Path string
	URL  string
	Err  error
}

func (e *CacheError) Error() string {
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

func (e *CacheError) Unwrap() error {
	return e.Err
}

func Ensure(ctx context.Context, artifact Artifact, cacheDir string, client Doer) (string, error) {
	if strings.TrimSpace(cacheDir) == "" {
		return "", &CacheError{Code: InvalidCacheDir, Err: fmt.Errorf("cache directory is required")}
	}
	if err := Validate(artifact); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", &CacheError{Code: InvalidCacheDir, Path: cacheDir, Err: err}
	}
	finalPath := filepath.Join(cacheDir, artifact.Filename)
	if err := Verify(finalPath, artifact); err == nil {
		return finalPath, nil
	} else if !os.IsNotExist(err) {
		if removeErr := os.Remove(finalPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return "", &CacheError{Code: WriteFailed, Path: finalPath, Err: removeErr}
		}
	}
	if nilDoer(client) {
		client = http.DefaultClient
	}
	if err := download(ctx, artifact, finalPath, client); err != nil {
		return "", err
	}
	return finalPath, nil
}

func nilDoer(client Doer) bool {
	if client == nil {
		return true
	}
	value := reflect.ValueOf(client)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func Validate(artifact Artifact) error {
	filename := strings.TrimSpace(artifact.Filename)
	if filename == "" || filename == "." || filename == ".." || strings.ContainsAny(filename, `/\\`) ||
		strings.TrimSpace(artifact.URL) == "" || artifact.SizeBytes <= 0 || strings.TrimSpace(artifact.SHA256) == "" {
		return &CacheError{Code: InvalidArtifact, Err: fmt.Errorf("artifact metadata is incomplete")}
	}
	return nil
}

func download(ctx context.Context, artifact Artifact, finalPath string, client Doer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return &CacheError{Code: InvalidArtifact, URL: artifact.URL, Err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return &CacheError{Code: DownloadFailed, URL: artifact.URL, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &CacheError{Code: BadStatus, URL: artifact.URL, Err: fmt.Errorf("http status %d", resp.StatusCode)}
	}

	partialPath := finalPath + ".partial"
	if err := os.Remove(partialPath); err != nil && !os.IsNotExist(err) {
		return &CacheError{Code: WriteFailed, Path: partialPath, Err: err}
	}
	out, err := os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return &CacheError{Code: WriteFailed, Path: partialPath, Err: err}
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(partialPath)
		return &CacheError{Code: DownloadFailed, URL: artifact.URL, Path: partialPath, Err: copyErr}
	}
	if closeErr != nil {
		_ = os.Remove(partialPath)
		return &CacheError{Code: WriteFailed, Path: partialPath, Err: closeErr}
	}
	if err := Verify(partialPath, artifact); err != nil {
		_ = os.Remove(partialPath)
		return err
	}
	if err := os.Rename(partialPath, finalPath); err != nil {
		_ = os.Remove(partialPath)
		return &CacheError{Code: WriteFailed, Path: finalPath, Err: err}
	}
	return nil
}

func Verify(path string, artifact Artifact) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return &CacheError{Code: InvalidArtifact, Path: path, Err: fmt.Errorf("artifact path is not a regular file")}
	}
	if info.Size() != artifact.SizeBytes {
		return &CacheError{Code: SizeMismatch, Path: path, Err: fmt.Errorf("size %d, want %d", info.Size(), artifact.SizeBytes)}
	}
	file, err := os.Open(path)
	if err != nil {
		return &CacheError{Code: WriteFailed, Path: path, Err: err}
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return &CacheError{Code: WriteFailed, Path: path, Err: err}
	}
	if got := hex.EncodeToString(hash.Sum(nil)); !strings.EqualFold(got, artifact.SHA256) {
		return &CacheError{Code: ChecksumMismatch, Path: path, Err: fmt.Errorf("sha256 %s, want %s", got, artifact.SHA256)}
	}
	return nil
}
