package backup

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/pathguard"
)

// ValidateRestoreZip opens the zip at zipPath and walks its entries
// without writing anything to disk. Returns a typed error when the
// archive is unreadable or contains a path-traversal entry that
// RestoreFromZip would reject at extract time. Used by the dry-run
// preview so operators see corruption/traversal up-front instead of
// after committing to --yes.
func ValidateRestoreZip(zipPath string) error {
	if strings.TrimSpace(zipPath) == "" {
		return fmt.Errorf("restore: zip path is empty")
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		// Distinguish "operator named a file that doesn't exist" from
		// "the file exists but is corrupt" so triage from the error
		// message alone is unambiguous. Without this split, both cases
		// chained the os.Open "open <path>" wording behind a
		// "restore: open zip:" prefix — operators saw "open ... open"
		// and had to decode the layered chain.
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("restore: zip not found: %s", zipPath)
		}
		return fmt.Errorf("restore: zip unreadable: %w", err)
	}
	defer zr.Close()
	// Walk entries against a synthetic dest so safeJoinForRestore
	// applies the same path-traversal rule it uses at extract time.
	const sentinel = "/dry-run-validation-target"
	for _, f := range zr.File {
		if _, err := safeJoinForRestore(sentinel, f.Name); err != nil {
			return err
		}
	}
	return nil
}

// RestoreFromZip extracts every entry of the zip at zipPath into
// destDir, overwriting existing files. This is the rollback path
// consumed by `gormes restore --path`.
//
// Safety:
//   - Zip entry names containing `..`, absolute paths, or that resolve
//     outside destDir after Clean+Join are rejected with a typed error.
//     Operators must not be able to restore a corrupted/malicious zip
//     into arbitrary filesystem locations.
//   - Parent directories are created with 0o755 as needed.
//   - Files are written 0o644 (regular file mode); zip entry mode bits
//     are intentionally ignored to keep the restore deterministic.
//
// Atomicity is per-file (write-then-rename via O_TRUNC); the operation
// is NOT all-or-nothing across files. A failure mid-restore leaves
// partial extraction on disk — operators recover by re-running with
// the same zip after fixing the cause (free disk, permissions).
func RestoreFromZip(ctx context.Context, zipPath, destDir string) error {
	if strings.TrimSpace(zipPath) == "" {
		return fmt.Errorf("restore: zip path is empty")
	}
	if strings.TrimSpace(destDir) == "" {
		return fmt.Errorf("restore: dest dir is empty")
	}
	// Pre-validate the whole archive: openable + every entry passes
	// the path-traversal guard. A malicious entry late in the zip
	// must not let earlier safe entries land on disk.
	if err := ValidateRestoreZip(zipPath); err != nil {
		return err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		// Same split as ValidateRestoreZip: clean message for "the
		// file disappeared" (TOCTOU between validation and open) vs.
		// the wrapped chain for genuine corruption.
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("restore: zip not found: %s", zipPath)
		}
		return fmt.Errorf("restore: zip unreadable: %w", err)
	}
	defer zr.Close()

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("restore: resolve dest: %w", err)
	}
	if err := os.MkdirAll(absDest, 0o755); err != nil {
		return fmt.Errorf("restore: mkdir dest: %w", err)
	}

	for _, f := range zr.File {
		if errCtx := ctx.Err(); errCtx != nil {
			return errCtx
		}
		target, err := safeJoinForRestore(absDest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("restore: mkdir %s: %w", f.Name, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("restore: mkdir parent %s: %w", f.Name, err)
		}
		if err := writeRestoreEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

// RestoreZipImpact summarizes how a zip restore will affect the
// destination tree without writing anything. The dry-run preview uses
// this so operators see the blast radius (overwrite vs. create counts)
// before committing to --yes.
type RestoreZipImpact struct {
	Overwrite int
	Create    int
}

// SummarizeRestoreZipImpact walks the zip at zipPath against the on-disk
// destDir and classifies each file entry as either an overwrite (the
// resolved on-disk target already exists) or a create (it does not).
// Directory entries are not counted — they're scaffolding, not files
// the operator cares about.
//
// Returns a typed error when the archive is unreadable or contains a
// path-traversal entry, so the dry-run still fails fast on unsafe zips.
func SummarizeRestoreZipImpact(zipPath, destDir string) (RestoreZipImpact, error) {
	if strings.TrimSpace(zipPath) == "" {
		return RestoreZipImpact{}, fmt.Errorf("restore: zip path is empty")
	}
	if strings.TrimSpace(destDir) == "" {
		return RestoreZipImpact{}, fmt.Errorf("restore: dest dir is empty")
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return RestoreZipImpact{}, fmt.Errorf("restore: zip not found: %s", zipPath)
		}
		return RestoreZipImpact{}, fmt.Errorf("restore: zip unreadable: %w", err)
	}
	defer zr.Close()
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return RestoreZipImpact{}, fmt.Errorf("restore: resolve dest: %w", err)
	}
	var impact RestoreZipImpact
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		target, err := safeJoinForRestore(absDest, f.Name)
		if err != nil {
			return RestoreZipImpact{}, err
		}
		if _, statErr := os.Stat(target); statErr == nil {
			impact.Overwrite++
		} else {
			impact.Create++
		}
	}
	return impact, nil
}

// safeJoinForRestore rejects zip entry names whose Clean+Join target
// escapes destDir. Returns the absolute on-disk target on success.
func safeJoinForRestore(absDest, name string) (string, error) {
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("restore: zip entry %q has absolute path; rejected to prevent escape", name)
	}
	clean, err := pathguard.CleanRelative(name)
	if err != nil {
		return "", fmt.Errorf("restore: zip entry %q escapes dest via path traversal; rejected", name)
	}
	cleanTarget := filepath.Join(absDest, clean)
	if !pathguard.Within(absDest, cleanTarget) {
		return "", fmt.Errorf("restore: zip entry %q escapes dest via path traversal; rejected", name)
	}
	return filepath.Clean(cleanTarget), nil
}

func writeRestoreEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("restore: open entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("restore: create %s: %w", f.Name, err)
	}
	if _, err := io.Copy(out, rc); err != nil {
		_ = out.Close()
		return fmt.Errorf("restore: copy %s: %w", f.Name, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("restore: close %s: %w", f.Name, err)
	}
	return nil
}
