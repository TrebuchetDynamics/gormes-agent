package snapshot

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const transactionalSnapshotMaxEntries = 50000

var transactionalSnapshotSkippedDirs = map[string]struct{}{
	".git":          {},
	"node_modules":  {},
	"dist":          {},
	"build":         {},
	".cache":        {},
	".next":         {},
	".nuxt":         {},
	"coverage":      {},
	".pytest_cache": {},
	".venv":         {},
	"venv":          {},
	"__pycache__":   {},
	"vendor":        {},
}

type snapshotEntryKind string

const (
	snapshotEntryDir     snapshotEntryKind = "dir"
	snapshotEntryFile    snapshotEntryKind = "file"
	snapshotEntrySymlink snapshotEntryKind = "symlink"
)

type workspaceSnapshotEntry struct {
	kind       snapshotEntryKind
	mode       fs.FileMode
	data       []byte
	linkTarget string
}

// WorkspaceSnapshot stores enough root-local filesystem state to restore a
// failed in-process tool call without following symlinks outside the workspace.
type WorkspaceSnapshot struct {
	root    string
	entries map[string]workspaceSnapshotEntry
}

// TakeWorkspaceSnapshot captures regular files, directories, and symlinks under
// root. It intentionally skips dependency/build/cache directories rather than
// trying to act as a full project backup system.
func TakeWorkspaceSnapshot(root string) (*WorkspaceSnapshot, error) {
	absRoot, err := normalizeSnapshotRoot(root)
	if err != nil {
		return nil, err
	}
	snapshot := &WorkspaceSnapshot{
		root:    absRoot,
		entries: make(map[string]workspaceSnapshotEntry),
	}
	count := 0
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() && shouldSkipTransactionalSnapshotDir(d.Name()) {
			return filepath.SkipDir
		}
		count++
		if count > transactionalSnapshotMaxEntries {
			return fmt.Errorf("workspace snapshot exceeds %d entries", transactionalSnapshotMaxEntries)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			snapshot.entries[rel] = workspaceSnapshotEntry{kind: snapshotEntrySymlink, mode: mode, linkTarget: target}
		case mode.IsDir():
			snapshot.entries[rel] = workspaceSnapshotEntry{kind: snapshotEntryDir, mode: mode}
		case mode.IsRegular():
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snapshot.entries[rel] = workspaceSnapshotEntry{kind: snapshotEntryFile, mode: mode, data: append([]byte(nil), raw...)}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// Restore returns the workspace to the state captured by TakeWorkspaceSnapshot.
func (s *WorkspaceSnapshot) Restore() error {
	if s == nil {
		return errors.New("workspace snapshot is nil")
	}
	if _, err := os.Stat(s.root); err != nil {
		return fmt.Errorf("snapshot root unavailable: %w", err)
	}
	if err := s.removeCreatedPaths(); err != nil {
		return err
	}
	if err := s.restoreDirectories(); err != nil {
		return err
	}
	if err := s.restoreLeafEntries(); err != nil {
		return err
	}
	return nil
}

// Commit discards the snapshot. It is explicit so TransactionalExecutor can
// report commit-vs-rollback behavior without exposing snapshot internals.
func (s *WorkspaceSnapshot) Commit() error {
	if s == nil {
		return nil
	}
	s.entries = nil
	return nil
}

func (s *WorkspaceSnapshot) removeCreatedPaths() error {
	var created []string
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() && shouldSkipTransactionalSnapshotDir(d.Name()) {
			return filepath.SkipDir
		}
		if _, ok := s.entries[rel]; !ok {
			created = append(created, rel)
			if d.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(created, func(i, j int) bool {
		return len(created[i]) > len(created[j])
	})
	for _, rel := range created {
		if err := os.RemoveAll(filepath.Join(s.root, rel)); err != nil {
			return fmt.Errorf("remove created path %s: %w", rel, err)
		}
	}
	return nil
}

func (s *WorkspaceSnapshot) restoreDirectories() error {
	dirs := make([]string, 0, len(s.entries))
	for rel, entry := range s.entries {
		if entry.kind == snapshotEntryDir {
			dirs = append(dirs, rel)
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) < len(dirs[j])
	})
	for _, rel := range dirs {
		entry := s.entries[rel]
		path := filepath.Join(s.root, rel)
		if info, err := os.Lstat(path); err == nil && !info.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("replace non-directory %s: %w", rel, err)
			}
		}
		if err := os.MkdirAll(path, entry.mode.Perm()); err != nil {
			return fmt.Errorf("restore directory %s: %w", rel, err)
		}
		_ = os.Chmod(path, entry.mode.Perm())
	}
	return nil
}

func (s *WorkspaceSnapshot) restoreLeafEntries() error {
	leaves := make([]string, 0, len(s.entries))
	for rel, entry := range s.entries {
		if entry.kind != snapshotEntryDir {
			leaves = append(leaves, rel)
		}
	}
	sort.Strings(leaves)
	for _, rel := range leaves {
		entry := s.entries[rel]
		path := filepath.Join(s.root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("restore parent %s: %w", rel, err)
		}
		if info, err := os.Lstat(path); err == nil {
			if info.IsDir() || info.Mode()&os.ModeSymlink != 0 || entry.kind == snapshotEntrySymlink {
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("replace existing path %s: %w", rel, err)
				}
			}
		}
		switch entry.kind {
		case snapshotEntryFile:
			if err := os.WriteFile(path, entry.data, entry.mode.Perm()); err != nil {
				return fmt.Errorf("restore file %s: %w", rel, err)
			}
			_ = os.Chmod(path, entry.mode.Perm())
		case snapshotEntrySymlink:
			if err := os.Symlink(entry.linkTarget, path); err != nil {
				return fmt.Errorf("restore symlink %s: %w", rel, err)
			}
		}
	}
	return nil
}

func normalizeSnapshotRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("workspace snapshot root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace snapshot root is not a directory: %s", abs)
	}
	if filepath.Dir(abs) == abs {
		return "", errors.New("refusing to snapshot filesystem root")
	}
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(home) == abs {
		return "", errors.New("refusing to snapshot home directory")
	}
	return abs, nil
}

func shouldSkipTransactionalSnapshotDir(name string) bool {
	_, ok := transactionalSnapshotSkippedDirs[name]
	return ok
}
