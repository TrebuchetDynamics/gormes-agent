package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrProfileCreateDefaultReserved = errors.New("profile_create_default_reserved")
	ErrProfileCreateTargetExists    = errors.New("profile_create_target_exists")
	ErrProfileCreateSourceMissing   = errors.New("profile_create_source_missing")
)

type ProfileCreateOptions struct {
	Name          string
	XDGConfigHome string
	TargetRoot    string
	SourceRoot    string
	CloneAll      bool
}

type ProfileCreateResult struct {
	Name     string
	Root     string
	CloneAll bool
}

var cloneAllDefaultExcludeRoot = map[string]struct{}{
	"gormes-agent": {},
	".worktrees":   {},
	"profiles":     {},
	"bin":          {},
	"node_modules": {},
}

var profileCloneAllStripRoot = []string{
	"gateway.pid",
	"gateway_state.json",
	"processes.json",
}

var profileCreateDefaultDirs = []string{
	"memories",
	"sessions",
	"skills",
	"skins",
	"logs",
	"plans",
	"workspace",
	"cron",
}

func CreateProfile(options ProfileCreateOptions) (ProfileCreateResult, error) {
	name := strings.TrimSpace(options.Name)
	if name == "default" {
		return ProfileCreateResult{}, fmt.Errorf("%w: default profile cannot be created", ErrProfileCreateDefaultReserved)
	}
	if err := ValidateProfileName(name); err != nil {
		return ProfileCreateResult{}, err
	}

	var targetRoot string
	var err error
	if strings.TrimSpace(options.TargetRoot) != "" {
		targetRoot = options.TargetRoot
	} else {
		xdgRoot := strings.TrimSpace(options.XDGConfigHome)
		if xdgRoot == "" {
			return ProfileCreateResult{}, ErrProfileXDGRootRequired
		}
		targetRoot, err = ResolveProfileRoot(name, xdgRoot)
		if err != nil {
			return ProfileCreateResult{}, err
		}
	}

	if _, err := os.Stat(targetRoot); err == nil {
		return ProfileCreateResult{}, fmt.Errorf("%w: %s", ErrProfileCreateTargetExists, targetRoot)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ProfileCreateResult{}, fmt.Errorf("profile create target: %w", err)
	}

	if options.CloneAll {
		var sourceRoot string
		if strings.TrimSpace(options.SourceRoot) != "" {
			sourceRoot = options.SourceRoot
		} else {
			xdgRoot := strings.TrimSpace(options.XDGConfigHome)
			if xdgRoot == "" {
				return ProfileCreateResult{}, ErrProfileXDGRootRequired
			}
			sourceRoot, err = ResolveProfileRoot("default", xdgRoot)
			if err != nil {
				return ProfileCreateResult{}, err
			}
		}
		if info, err := os.Stat(sourceRoot); err != nil || !info.IsDir() {
			if err == nil {
				err = fmt.Errorf("%s is not a directory", sourceRoot)
			}
			if errors.Is(err, os.ErrNotExist) {
				return ProfileCreateResult{}, fmt.Errorf("%w: %s", ErrProfileCreateSourceMissing, sourceRoot)
			}
			return ProfileCreateResult{}, fmt.Errorf("%w: %v", ErrProfileCreateSourceMissing, err)
		}
		if err := copyProfileTree(sourceRoot, targetRoot, true); err != nil {
			_ = os.RemoveAll(targetRoot)
			return ProfileCreateResult{}, err
		}
		if err := stripProfileRuntimeFiles(targetRoot); err != nil {
			_ = os.RemoveAll(targetRoot)
			return ProfileCreateResult{}, err
		}
		return ProfileCreateResult{Name: name, Root: targetRoot, CloneAll: true}, nil
	}

	if err := os.MkdirAll(targetRoot, 0o700); err != nil {
		return ProfileCreateResult{}, fmt.Errorf("profile create root: %w", err)
	}
	for _, dir := range profileCreateDefaultDirs {
		if err := os.MkdirAll(filepath.Join(targetRoot, dir), 0o700); err != nil {
			_ = os.RemoveAll(targetRoot)
			return ProfileCreateResult{}, fmt.Errorf("profile create dir %s: %w", dir, err)
		}
	}
	return ProfileCreateResult{Name: name, Root: targetRoot, CloneAll: false}, nil
}

func copyProfileTree(sourceRoot, targetRoot string, defaultSource bool) error {
	sourceRoot = filepath.Clean(sourceRoot)
	targetRoot = filepath.Clean(targetRoot)
	if err := os.MkdirAll(targetRoot, 0o700); err != nil {
		return fmt.Errorf("profile clone target: %w", err)
	}
	return filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		isRootEntry := filepath.Dir(rel) == "."
		if shouldIgnoreCloneAllEntry(entry.Name(), isRootEntry, defaultSource) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		targetPath := filepath.Join(targetRoot, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			return copyProfileSymlink(path, targetPath)
		case entry.IsDir():
			return os.MkdirAll(targetPath, mode.Perm())
		case mode.IsRegular():
			return copyProfileFile(path, targetPath, mode.Perm())
		default:
			return nil
		}
	})
}

func shouldIgnoreCloneAllEntry(name string, isRootEntry, defaultSource bool) bool {
	if name == "__pycache__" || strings.HasSuffix(name, ".pyc") || strings.HasSuffix(name, ".pyo") || strings.HasSuffix(name, ".sock") || strings.HasSuffix(name, ".tmp") {
		return true
	}
	if defaultSource && isRootEntry {
		_, ok := cloneAllDefaultExcludeRoot[name]
		return ok
	}
	return false
}

func copyProfileSymlink(sourcePath, targetPath string) error {
	link, err := os.Readlink(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return err
	}
	return os.Symlink(link, targetPath)
}

func copyProfileFile(sourcePath, targetPath string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return err
	}
	in, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
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

func stripProfileRuntimeFiles(targetRoot string) error {
	for _, rel := range profileCloneAllStripRoot {
		err := os.Remove(filepath.Join(targetRoot, rel))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("profile strip runtime file %s: %w", rel, err)
		}
	}
	return nil
}
