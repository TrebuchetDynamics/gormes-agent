package cli

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// backupFilenamePrefix matches the writer's destination naming
// convention: pre-update-<UTC-timestamp>.zip. PruneBackups uses this
// prefix to filter operator-owned files away from the prune target set.
const backupFilenamePrefix = "pre-update-"

// BackupListing describes one pre-update backup zip discovered under
// the operator's backups directory. Returned by ListBackups for the
// `gormes restore --list` operator surface.
type BackupListing struct {
	Path      string
	SizeBytes int64
	ModTime   time.Time
}

// ListBackups enumerates `pre-update-*.zip` files under backupDir and
// returns them sorted newest-first by mtime. Files that don't match the
// pattern are ignored (operators may store unrelated files in the same
// directory). A missing backupDir is a quiet empty-list — fresh
// installs with no prior backups should see "no backups found", not an
// error.
func ListBackups(backupDir string) ([]BackupListing, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("backup list: read dir: %w", err)
	}
	out := make([]BackupListing, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, backupFilenamePrefix) || filepath.Ext(name) != ".zip" {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		out = append(out, BackupListing{
			Path:      filepath.Join(backupDir, name),
			SizeBytes: info.Size(),
			ModTime:   info.ModTime(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ModTime.After(out[j].ModTime)
	})
	return out, nil
}

// PruneBackups removes older pre-update backup zips from backupDir,
// keeping the newest `keep` files by mtime. Returns the number of files
// removed and total bytes freed. Files that don't match the
// `pre-update-*.zip` pattern are ignored — operators can store their
// own files in the same directory without losing them.
//
// Safety:
//   - keep <= 0    → no-op (operators who pass --backup-keep 0 by mistake
//                   should not lose data).
//   - missing dir  → no-op (fresh installs with no prior backups must not
//                   error during a post-write prune).
//
// On any per-file removal error, the helper reports the partial count
// and freed total without surfacing the underlying error — pruning is
// best-effort and must not block update completion.
func PruneBackups(backupDir string, keep int) (removedCount int, freedBytes int64, err error) {
	if keep <= 0 {
		return 0, 0, nil
	}
	entries, readErr := os.ReadDir(backupDir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("backup prune: read dir: %w", readErr)
	}
	type backupEntry struct {
		path  string
		mtime time.Time
		size  int64
	}
	candidates := make([]backupEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, backupFilenamePrefix) || filepath.Ext(name) != ".zip" {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		candidates = append(candidates, backupEntry{
			path:  filepath.Join(backupDir, name),
			mtime: info.ModTime(),
			size:  info.Size(),
		})
	}
	if len(candidates) <= keep {
		return 0, 0, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].mtime.After(candidates[j].mtime)
	})
	for _, c := range candidates[keep:] {
		if rmErr := os.Remove(c.path); rmErr != nil {
			continue
		}
		removedCount++
		freedBytes += c.size
	}
	return removedCount, freedBytes, nil
}

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
	var fileCount int
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
		fileCount++
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
		FileCount:  fileCount,
	}, nil
}
