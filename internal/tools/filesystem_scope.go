package tools

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
}

type PathCheckResult struct {
	Allowed   bool
	Normalized string
	Evidence  string
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
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		p, _ = filepath.EvalSymlinks(p)
		p = filepath.Clean(p)
		result = append(result, p)
	}
	return result
}

func normalizePath(path, cwd string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	path = filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil
	}
	return resolved, nil
}

func (fs *FilesystemScope) CheckRead(path, cwd string) PathCheckResult {
	return fs.checkPath(path, cwd, fs.AllowedReadPaths, "filesystem_read_scope_violation")
}

func (fs *FilesystemScope) CheckWrite(path, cwd string) PathCheckResult {
	return fs.checkPath(path, cwd, fs.AllowedWritePaths, "filesystem_write_scope_violation")
}

func (fs *FilesystemScope) checkPath(path, cwd string, allowed []string, violationEvidence string) PathCheckResult {
	normalized, err := normalizePath(path, cwd)
	if err != nil {
		return PathCheckResult{Allowed: false, Evidence: fmt.Sprintf("path_normalize_error: %v", err)}
	}

	if fs.CWDOnly {
		if !strings.HasPrefix(normalized, cwd) {
			return PathCheckResult{Allowed: false, Normalized: normalized, Evidence: violationEvidence}
		}
		return PathCheckResult{Allowed: true, Normalized: normalized, Evidence: "path_normalized"}
	}

	if len(allowed) == 0 {
		return PathCheckResult{Allowed: true, Normalized: normalized, Evidence: "path_normalized"}
	}

	for _, a := range allowed {
		if strings.HasPrefix(normalized, a) || normalized == a {
			return PathCheckResult{Allowed: true, Normalized: normalized, Evidence: "path_normalized"}
		}
	}

	return PathCheckResult{Allowed: false, Normalized: normalized, Evidence: violationEvidence}
}

func (fs *FilesystemScope) GetDoctorReport(cwd string) map[string]interface{} {
	return map[string]interface{}{
		"filesystem_scope_config": map[string]interface{}{
			"cwd_only":            fs.CWDOnly,
			"allowed_read_paths":  fs.AllowedReadPaths,
			"allowed_write_paths": fs.AllowedWritePaths,
			"cwd":                 cwd,
		},
	}
}
