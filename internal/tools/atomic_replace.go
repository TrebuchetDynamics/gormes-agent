package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	AtomicWriteFailed        = "atomic_write_failed"
	AtomicWriteSymlinkEscape = "atomic_write_symlink_escape"
)

type AtomicReplaceOptions struct {
	// FirstWriteMode is applied when the resolved target does not already exist.
	FirstWriteMode os.FileMode
	// Root, when set, constrains resolved symlink targets to a caller-approved
	// root before replacing the temp file.
	Root string
}

type AtomicReplaceResult struct {
	Path             string
	PreservedSymlink bool
}

type AtomicReplaceError struct {
	Code string
	Op   string
	Err  error
}

func (e *AtomicReplaceError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Code + ": " + e.Op
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Op, e.Err)
}

func (e *AtomicReplaceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func AtomicReplace(tmpPath, targetPath string, opts AtomicReplaceOptions) (AtomicReplaceResult, error) {
	resolved, preserved, err := atomicReplaceTarget(targetPath)
	if err != nil {
		return AtomicReplaceResult{}, &AtomicReplaceError{Code: AtomicWriteFailed, Op: "resolve target", Err: err}
	}
	if opts.Root != "" {
		if err := atomicReplaceCheckRoot(resolved, opts.Root); err != nil {
			return AtomicReplaceResult{}, err
		}
	}

	mode, hasMode := atomicReplaceExistingMode(resolved)
	if err := os.Rename(tmpPath, resolved); err != nil {
		return AtomicReplaceResult{}, &AtomicReplaceError{Code: AtomicWriteFailed, Op: "replace target", Err: err}
	}
	switch {
	case hasMode:
		if err := os.Chmod(resolved, mode); err != nil {
			return AtomicReplaceResult{}, &AtomicReplaceError{Code: AtomicWriteFailed, Op: "restore mode", Err: err}
		}
	case opts.FirstWriteMode != 0:
		if err := os.Chmod(resolved, opts.FirstWriteMode); err != nil {
			return AtomicReplaceResult{}, &AtomicReplaceError{Code: AtomicWriteFailed, Op: "set first-write mode", Err: err}
		}
	}
	return AtomicReplaceResult{Path: resolved, PreservedSymlink: preserved}, nil
}

func atomicReplaceTarget(targetPath string) (string, bool, error) {
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return "", false, err
	}
	info, err := os.Lstat(targetAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return targetAbs, false, nil
		}
		return "", false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return targetAbs, false, nil
	}
	if resolved, err := filepath.EvalSymlinks(targetAbs); err == nil {
		return filepath.Clean(resolved), true, nil
	}
	linkTarget, err := os.Readlink(targetAbs)
	if err != nil {
		return "", true, err
	}
	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(targetAbs), linkTarget)
	}
	resolved := filepath.Clean(linkTarget)
	if resolved == targetAbs {
		return "", true, errors.New("symlink resolves to itself")
	}
	return resolved, true, nil
}

func atomicReplaceExistingMode(path string) (os.FileMode, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return info.Mode().Perm(), true
}

func atomicReplaceCheckRoot(path, root string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return &AtomicReplaceError{Code: AtomicWriteSymlinkEscape, Op: "resolve root", Err: err}
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return &AtomicReplaceError{Code: AtomicWriteSymlinkEscape, Op: "resolve path", Err: err}
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return &AtomicReplaceError{Code: AtomicWriteSymlinkEscape, Op: "compare root", Err: err}
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return &AtomicReplaceError{Code: AtomicWriteSymlinkEscape, Op: "target outside root"}
	}
	return nil
}

// AtomicWrite creates a temp file, writes data to it, and atomically renames
// it to targetPath. On failure the temp file is removed and an error is
// returned. Existing file permissions are preserved; first writes use 0644.
func AtomicWrite(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	f, err := os.CreateTemp(dir, ".gormes-write-*")
	if err != nil {
		return &AtomicReplaceError{Code: AtomicWriteFailed, Op: "create temp", Err: err}
	}
	tmpPath := f.Name()
	cleanup := func() { os.Remove(tmpPath) }

	if _, err := f.Write(data); err != nil {
		f.Close()
		cleanup()
		return &AtomicReplaceError{Code: AtomicWriteFailed, Op: "write temp", Err: err}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		cleanup()
		return &AtomicReplaceError{Code: AtomicWriteFailed, Op: "sync temp", Err: err}
	}
	if err := f.Close(); err != nil {
		cleanup()
		return &AtomicReplaceError{Code: AtomicWriteFailed, Op: "close temp", Err: err}
	}

	result, err := AtomicReplace(tmpPath, targetPath, AtomicReplaceOptions{FirstWriteMode: 0644})
	if err != nil {
		cleanup()
		return err
	}
	_ = result
	return nil
}
