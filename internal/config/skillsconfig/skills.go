package skillsconfig

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
)

const (
	ExternalDirResolved = "skills_external_dir_resolved"
	ExternalDirSkipped  = "skills_external_dir_skipped"
)

type Config struct {
	Root             string   `toml:"root" yaml:"root"`
	SelectionCap     int      `toml:"selection_cap" yaml:"selection_cap"`
	MaxDocumentBytes int      `toml:"max_document_bytes" yaml:"max_document_bytes"`
	UsageLogPath     string   `toml:"usage_log_path" yaml:"usage_log_path"`
	ExternalDirs     []string `toml:"external_dirs" yaml:"external_dirs"`
}

type ExternalDirEvidence struct {
	Code   string
	Input  string
	Path   string
	Reason string
}

func Root(configuredRoot string) string {
	if configuredRoot != "" {
		return configuredRoot
	}
	return filepath.Join(paths.GormesHome(), "skills")
}

// ExternalDirs resolves Hermes-compatible skills.external_dirs entries. Paths
// expand ~ and environment variables, relative entries resolve against
// GormesHome rather than process cwd, and invalid/duplicate/local roots are
// skipped with typed evidence instead of failing provider startup.
func ExternalDirs(root string, entries []string) ([]string, []ExternalDirEvidence) {
	localRoot := canonicalDirPath(root)
	localActive := canonicalDirPath(filepath.Join(root, "active"))
	localRoots := map[string]bool{localRoot: true, localActive: true}
	seen := map[string]bool{}
	var out []string
	var evidence []ExternalDirEvidence
	for _, entry := range entries {
		raw := strings.TrimSpace(entry)
		if raw == "" {
			evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirSkipped, Input: entry, Reason: "empty"})
			continue
		}
		resolved := resolveDir(raw, paths.GormesHome())
		canonical := canonicalDirPath(resolved)
		if localRoots[canonical] {
			evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirSkipped, Input: raw, Path: resolved, Reason: "local_root"})
			continue
		}
		if seen[canonical] {
			evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirSkipped, Input: raw, Path: resolved, Reason: "duplicate"})
			continue
		}
		info, err := os.Stat(resolved)
		switch {
		case os.IsNotExist(err):
			evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirSkipped, Input: raw, Path: resolved, Reason: "missing"})
			continue
		case err != nil:
			evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirSkipped, Input: raw, Path: resolved, Reason: "stat_failed"})
			continue
		case !info.IsDir():
			evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirSkipped, Input: raw, Path: resolved, Reason: "not_directory"})
			continue
		}
		seen[canonical] = true
		out = append(out, resolved)
		evidence = append(evidence, ExternalDirEvidence{Code: ExternalDirResolved, Input: raw, Path: resolved})
	}
	return out, evidence
}

func UsageLogPath(root, configuredPath string) string {
	if configuredPath != "" {
		return configuredPath
	}
	return filepath.Join(root, "usage.jsonl")
}

func resolveDir(raw, gormesHome string) string {
	expanded := expandLeadingTilde(os.ExpandEnv(strings.TrimSpace(raw)))
	expanded = filepath.FromSlash(expanded)
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(gormesHome, expanded)
	}
	if abs, err := filepath.Abs(expanded); err == nil {
		expanded = abs
	}
	if real, err := filepath.EvalSymlinks(expanded); err == nil && real != "" {
		expanded = real
	}
	return filepath.Clean(expanded)
}

func expandLeadingTilde(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}

func canonicalDirPath(path string) string {
	path = filepath.FromSlash(strings.TrimSpace(path))
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if real, err := filepath.EvalSymlinks(path); err == nil && real != "" {
		path = real
	}
	return filepath.Clean(path)
}
