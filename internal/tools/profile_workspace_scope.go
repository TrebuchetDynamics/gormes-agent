package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ProfileWorkspaceScopeViolation = "profile_workspace_scope_violation"

type ProfileWorkspaceAccess string

const (
	ProfileWorkspaceAccessRead     ProfileWorkspaceAccess = "read"
	ProfileWorkspaceAccessWrite    ProfileWorkspaceAccess = "write"
	ProfileWorkspaceAccessExecute  ProfileWorkspaceAccess = "execute"
	ProfileWorkspaceAccessDelegate ProfileWorkspaceAccess = "delegate"
)

type ProfileWorkspaceScopeOptions struct {
	ProjectRoots []string
	ProfileRoot  string
	OperatorHome string
}

type ProfileWorkspaceScope struct {
	projectRoots []string
	profileRoot  string
	profilesRoot string
	operatorHome string
	configured   bool
	configError  string
}

func NewProfileWorkspaceScope(opts ProfileWorkspaceScopeOptions) (*ProfileWorkspaceScope, error) {
	operatorHome := strings.TrimSpace(opts.OperatorHome)
	if operatorHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("profile workspace scope: resolve operator home: %w", err)
		}
		operatorHome = home
	}
	operatorHome, err := normalizeRequiredWorkspaceRoot(operatorHome)
	if err != nil {
		return nil, fmt.Errorf("profile workspace scope: operator home: %w", err)
	}

	projectRoots := cleanWorkspaceList(opts.ProjectRoots)
	configured := len(projectRoots) > 0
	if len(projectRoots) == 0 {
		projectRoots = []string{operatorHome}
	}
	normalizedRoots := make([]string, 0, len(projectRoots))
	for _, raw := range projectRoots {
		root, err := normalizeRequiredWorkspaceRoot(raw)
		if err != nil {
			return nil, fmt.Errorf("profile workspace scope: project root %q: %w", raw, err)
		}
		if !containsPath(normalizedRoots, root) {
			normalizedRoots = append(normalizedRoots, root)
		}
	}

	profileRoot := strings.TrimSpace(opts.ProfileRoot)
	if profileRoot != "" {
		profileRoot, err = normalizeWorkspacePath(profileRoot, operatorHome)
		if err != nil {
			return nil, fmt.Errorf("profile workspace scope: profile root: %w", err)
		}
	}
	return &ProfileWorkspaceScope{
		projectRoots: normalizedRoots,
		profileRoot:  profileRoot,
		profilesRoot: inferProfilesRoot(profileRoot),
		operatorHome: operatorHome,
		configured:   configured,
	}, nil
}

func NewFailClosedProfileWorkspaceScope(err error) *ProfileWorkspaceScope {
	msg := "profile workspace scope unavailable"
	if err != nil {
		msg = err.Error()
	}
	return &ProfileWorkspaceScope{configured: true, configError: msg}
}

func (s *ProfileWorkspaceScope) Configured() bool {
	return s != nil && s.configured
}

func (s *ProfileWorkspaceScope) DefaultRoot() string {
	if s == nil || len(s.projectRoots) == 0 {
		return ""
	}
	return s.projectRoots[0]
}

func (s *ProfileWorkspaceScope) ProjectRoots() []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s.projectRoots...)
}

func (s *ProfileWorkspaceScope) Resolve(rawPath, base string, access ProfileWorkspaceAccess) PathCheckResult {
	if s == nil {
		return PathCheckResult{Allowed: true, Evidence: "profile_workspace_scope_unset"}
	}
	if s.configError != "" {
		return s.deny("", fmt.Sprintf("profile workspace policy is invalid: %s", s.configError))
	}
	if strings.TrimSpace(rawPath) == "" {
		return s.deny("", "path is required")
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = s.DefaultRoot()
	}
	if base == "" {
		return s.deny("", "profile workspace policy has no project root")
	}
	normalized, err := normalizeWorkspacePath(rawPath, base)
	if err != nil {
		return s.deny("", fmt.Sprintf("resolve path: %v", err))
	}

	if s.configured && s.profileRoot != "" && pathWithinRoot(s.profileRoot, normalized) {
		if s.profileOwnedPathAllowed(normalized, access) {
			return s.allow(normalized, s.profileRoot)
		}
		return s.deny(normalized, "active profile runtime state is not a model-facing project workspace")
	}
	if s.configured && s.profilesRoot != "" && pathWithinRoot(s.profilesRoot, normalized) {
		return s.deny(normalized, "sibling profile roots are not model-facing project workspaces")
	}
	for _, root := range s.projectRoots {
		if pathWithinRoot(root, normalized) {
			return s.allow(normalized, root)
		}
	}
	return s.deny(normalized, fmt.Sprintf("%s: path is outside configured profile workspace roots", access))
}

func (s *ProfileWorkspaceScope) profileOwnedPathAllowed(path string, access ProfileWorkspaceAccess) bool {
	if s == nil || s.profileRoot == "" {
		return false
	}
	switch access {
	case ProfileWorkspaceAccessRead, ProfileWorkspaceAccessWrite, ProfileWorkspaceAccessDelegate:
	default:
		return false
	}
	for _, name := range []string{"SOUL.md", "IDENTITY.md"} {
		if filepath.Clean(path) == filepath.Join(s.profileRoot, name) {
			return true
		}
	}
	skillsRoot := filepath.Join(s.profileRoot, "skills")
	return pathWithinRoot(skillsRoot, path)
}

func (s *ProfileWorkspaceScope) allow(path, root string) PathCheckResult {
	if err := validateWorkspaceRealPath(root, path); err != nil {
		return s.deny(path, err.Error())
	}
	return PathCheckResult{
		Allowed:    true,
		Normalized: path,
		Root:       root,
		Relative:   workspaceRel(root, path),
		Evidence:   "path_normalized",
	}
}

func (s *ProfileWorkspaceScope) deny(path, message string) PathCheckResult {
	return PathCheckResult{
		Allowed:    false,
		Normalized: path,
		Evidence:   ProfileWorkspaceScopeViolation,
		Message:    message,
	}
}

func cleanWorkspaceList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeRequiredWorkspaceRoot(path string) (string, error) {
	normalized, err := normalizeWorkspacePath(path, "")
	if err != nil {
		return "", err
	}
	info, err := os.Stat(normalized)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", normalized)
	}
	return normalized, nil
}

func normalizeWorkspacePath(path, base string) (string, error) {
	expanded, err := expandUserPath(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		if strings.TrimSpace(base) == "" {
			base, err = os.Getwd()
			if err != nil {
				return "", err
			}
		}
		expanded = filepath.Join(base, expanded)
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	return evalPathOrExistingAncestor(filepath.Clean(abs)), nil
}

func inferProfilesRoot(profileRoot string) string {
	if profileRoot == "" {
		return ""
	}
	parent := filepath.Dir(profileRoot)
	if filepath.Base(parent) == "profiles" {
		return parent
	}
	return ""
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == filepath.Clean(target) {
			return true
		}
	}
	return false
}
