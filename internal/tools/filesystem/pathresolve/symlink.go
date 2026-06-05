package pathresolve

import (
	"os"
	"path/filepath"
)

// ExistingOrSymlinkTarget returns a cleaned absolute path for regular, missing,
// and symlink targets. Broken symlinks resolve to their link target without
// requiring the target to exist.
func ExistingOrSymlinkTarget(path string) (resolved string, symlink bool, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, err
	}
	abs = filepath.Clean(abs)
	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, false, nil
		}
		return "", false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return abs, false, nil
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved), true, nil
	}
	linkTarget, err := os.Readlink(abs)
	if err != nil {
		return "", true, err
	}
	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(abs), linkTarget)
	}
	return filepath.Clean(linkTarget), true, nil
}
