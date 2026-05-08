package cli

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteBackupZip walks sourceDir and writes a zip archive containing
// every file that does NOT match the IsExcludedFromBackup rules. The
// zip is written atomically: it streams to destPath.tmp first and
// renames on success.
//
// Returns BackupResult with the resolved destPath, byte size of the
// final archive, and total wall-clock duration. On any error, the
// partial .tmp file is removed and a non-nil error is returned.
func WriteBackupZip(ctx context.Context, sourceDir, destPath string) (BackupResult, error) {
	start := time.Now()
	if strings.TrimSpace(sourceDir) == "" {
		return BackupResult{}, fmt.Errorf("backup: source dir is empty")
	}
	if strings.TrimSpace(destPath) == "" {
		return BackupResult{}, fmt.Errorf("backup: dest path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return BackupResult{}, fmt.Errorf("backup: mkdir dest parent: %w", err)
	}
	tmpPath := destPath + ".tmp"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return BackupResult{}, fmt.Errorf("backup: open tmp dest: %w", err)
	}
	zw := zip.NewWriter(out)
	walkErr := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if errCtx := ctx.Err(); errCtx != nil {
			return errCtx
		}
		rel, relErr := filepath.Rel(sourceDir, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		// Skip the destination zip itself if it lives under sourceDir
		// (avoids the backup including a partial copy of itself).
		if path == tmpPath || path == destPath {
			return nil
		}
		if IsExcludedFromBackup(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		f, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer f.Close()
		header := &zip.FileHeader{Name: filepath.ToSlash(rel), Method: zip.Deflate}
		w, hdrErr := zw.CreateHeader(header)
		if hdrErr != nil {
			return hdrErr
		}
		if _, copyErr := io.Copy(w, f); copyErr != nil {
			return copyErr
		}
		return nil
	})
	if closeErr := zw.Close(); walkErr == nil && closeErr != nil {
		walkErr = closeErr
	}
	if syncErr := out.Sync(); walkErr == nil && syncErr != nil {
		walkErr = syncErr
	}
	if cErr := out.Close(); walkErr == nil && cErr != nil {
		walkErr = cErr
	}
	if walkErr != nil {
		_ = os.Remove(tmpPath)
		return BackupResult{}, fmt.Errorf("backup: walk/write: %w", walkErr)
	}
	if renameErr := os.Rename(tmpPath, destPath); renameErr != nil {
		_ = os.Remove(tmpPath)
		return BackupResult{}, fmt.Errorf("backup: rename to dest: %w", renameErr)
	}
	info, statErr := os.Stat(destPath)
	if statErr != nil {
		return BackupResult{}, fmt.Errorf("backup: stat dest: %w", statErr)
	}
	return BackupResult{
		Path:       destPath,
		SizeBytes:  info.Size(),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}
