package fileops

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// CopyFile copies source to target using the supplied target file mode.
func CopyFile(source, target string, perm os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// SHA256 returns the lowercase SHA-256 hex digest for path.
func SHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ReplaceFileAtomically copies source into a same-directory temp file, chmods it,
// and renames it over target. Existing directories are never replaced.
func ReplaceFileAtomically(source, target string, perm os.FileMode) error {
	if !textvalue.IsNonBlank(source) || !textvalue.IsNonBlank(target) {
		return fmt.Errorf("source and target are required")
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return fmt.Errorf("cannot replace directory %s", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", target, os.Getpid())
	_ = os.Remove(tmp)
	if err := CopyFile(source, tmp, perm); err != nil {
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// SamePath compares two paths by absolute path first, then by resolved symlink path.
func SamePath(a, b string) bool {
	if !textvalue.IsNonBlank(a) || !textvalue.IsNonBlank(b) {
		return false
	}
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA == nil && errB == nil && aa == bb {
		return true
	}
	ea, errA := filepath.EvalSymlinks(a)
	eb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ea == eb
}
