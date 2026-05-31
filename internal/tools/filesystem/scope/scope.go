package scope

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FilesystemScope struct {
	AllowedReadPaths  []string
	AllowedWritePaths []string
	CWDOnly           bool
	// ReadOnly, when true, denies all write operations regardless of
	// AllowedWritePaths or CWDOnly.
	ReadOnly bool
}

type PathCheckResult struct {
	Allowed    bool
	Normalized string
	Evidence   string
	Root       string
	Relative   string
	Message    string
}

func NewFilesystemScope(cwd string, readPaths, writePaths []string) *FilesystemScope {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if readPaths == nil && writePaths == nil {
		return &FilesystemScope{CWDOnly: true}
	}
	return &FilesystemScope{
		AllowedReadPaths:  normalizePaths(readPaths, cwd),
		AllowedWritePaths: normalizePaths(writePaths, cwd),
	}
}

func normalizePaths(paths []string, cwd string) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		p, _ = normalizePath(p, cwd)
		result = append(result, p)
	}
	return result
}

func normalizePath(path, cwd string) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	cwd = filepath.Clean(cwd)
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return EvalPathOrExistingAncestor(filepath.Clean(abs)), nil
}

func (fs *FilesystemScope) CheckRead(path, cwd string) PathCheckResult {
	return fs.checkPath(path, cwd, fs.AllowedReadPaths, "filesystem_read_scope_violation")
}

func (fs *FilesystemScope) CheckWrite(path, cwd string) PathCheckResult {
	if fs.ReadOnly {
		normalized, err := normalizePath(path, cwd)
		if err != nil {
			return PathCheckResult{Allowed: false, Evidence: fmt.Sprintf("path_normalize_error: %v", err)}
		}
		return PathCheckResult{
			Allowed:    false,
			Normalized: normalized,
			Evidence:   "filesystem_readonly_mode",
			Message:    "workspace is in read-only mode: all writes are denied",
		}
	}
	return fs.checkPath(path, cwd, fs.AllowedWritePaths, "filesystem_write_scope_violation")
}

func (fs *FilesystemScope) checkPath(path, cwd string, allowed []string, violationEvidence string) PathCheckResult {
	normalized, err := normalizePath(path, cwd)
	if err != nil {
		return PathCheckResult{Allowed: false, Evidence: fmt.Sprintf("path_normalize_error: %v", err)}
	}

	if fs.CWDOnly {
		if !pathWithinRootForScope(cwd, normalized) {
			return PathCheckResult{Allowed: false, Normalized: normalized, Evidence: violationEvidence}
		}
		return PathCheckResult{Allowed: true, Normalized: normalized, Evidence: "path_normalized"}
	}

	if len(allowed) == 0 {
		return PathCheckResult{Allowed: true, Normalized: normalized, Evidence: "path_normalized"}
	}

	for _, a := range allowed {
		if pathWithinRootForScope(a, normalized) {
			return PathCheckResult{Allowed: true, Normalized: normalized, Evidence: "path_normalized"}
		}
	}

	return PathCheckResult{Allowed: false, Normalized: normalized, Evidence: violationEvidence}
}

func (fs *FilesystemScope) GetDoctorReport(cwd string) map[string]interface{} {
	return map[string]interface{}{
		"filesystem_scope_config": map[string]interface{}{
			"cwd_only":            fs.CWDOnly,
			"read_only":           fs.ReadOnly,
			"allowed_read_paths":  fs.AllowedReadPaths,
			"allowed_write_paths": fs.AllowedWritePaths,
			"cwd":                 cwd,
		},
	}
}

func EvalPathOrExistingAncestor(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(evaluated)
	}

	cleaned := filepath.Clean(path)
	ancestor := cleaned
	var suffix []string
	for {
		if evaluated, err := filepath.EvalSymlinks(ancestor); err == nil {
			resolved := filepath.Clean(evaluated)
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return filepath.Clean(resolved)
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return cleaned
		}
		suffix = append(suffix, filepath.Base(ancestor))
		ancestor = parent
	}
}

func pathWithinRootForScope(root, path string) bool {
	root = evalExistingPathForScope(root)
	path = evalPathOrParentForScope(path)
	if root == string(filepath.Separator) {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func ValidateWorkspaceRealPath(root, abs string) error {
	if realPath, err := filepath.EvalSymlinks(abs); err == nil {
		if !pathWithinRootForScope(root, filepath.Clean(realPath)) {
			return fmt.Errorf("path %q resolves outside workspace root %q", abs, root)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("resolve symlink path: %w", err)
	} else if info, lstatErr := os.Lstat(abs); lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a broken symlink", abs)
	}

	ancestor := filepath.Dir(abs)
	for {
		if ancestor == "" || ancestor == "." || ancestor == string(filepath.Separator) {
			break
		}
		if realAncestor, err := filepath.EvalSymlinks(ancestor); err == nil {
			if !pathWithinRootForScope(root, filepath.Clean(realAncestor)) {
				return fmt.Errorf("path parent %q resolves outside workspace root %q", ancestor, root)
			}
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("resolve parent symlink path: %w", err)
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			break
		}
		ancestor = next
	}
	return nil
}

func WorkspaceRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	if rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func evalPathOrParentForScope(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(evaluated)
	}
	parent := filepath.Dir(path)
	if evaluatedParent, err := filepath.EvalSymlinks(parent); err == nil {
		return filepath.Clean(filepath.Join(evaluatedParent, filepath.Base(path)))
	}
	return filepath.Clean(path)
}

func evalExistingPathForScope(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(evaluated)
	}
	return filepath.Clean(path)
}
