package scope

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ProfileWorkspaceScopeViolation = "profile_workspace_scope_violation"
const ProfileWorkspaceDeniedMessage = "Path is outside this profile’s allowed workspace. Add it to allowed_paths to grant access."

var profileWorkspaceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type ProfileWorkspaceAccess string

const (
	ProfileWorkspaceAccessRead     ProfileWorkspaceAccess = "read"
	ProfileWorkspaceAccessWrite    ProfileWorkspaceAccess = "write"
	ProfileWorkspaceAccessExecute  ProfileWorkspaceAccess = "execute"
	ProfileWorkspaceAccessDelegate ProfileWorkspaceAccess = "delegate"
)

type ProfileWorkspaceScopeOptions struct {
	ProfileName   string
	ProjectRoots  []string
	ProfileRoot   string
	WorkspaceRoot string
	OperatorHome  string
}

type ProfileWorkspaceScope struct {
	projectRoots []string
	profileName  string
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
	operatorHome, err := normalizeRequiredWorkspaceRootWithHome(operatorHome, "")
	if err != nil {
		return nil, fmt.Errorf("profile workspace scope: operator home: %w", err)
	}

	profileName := strings.TrimSpace(opts.ProfileName)
	if profileName != "" && !validProfileWorkspaceName(profileName) {
		return nil, fmt.Errorf("profile workspace scope: profile name %q must match [a-zA-Z0-9][a-zA-Z0-9._-]*", profileName)
	}

	profileRoot := strings.TrimSpace(opts.ProfileRoot)
	if profileRoot != "" {
		profileRoot, err = normalizeRequiredWorkspaceRootWithHome(profileRoot, operatorHome)
		if err != nil {
			return nil, fmt.Errorf("profile workspace scope: profile root: %w", err)
		}
		baseName := filepath.Base(profileRoot)
		if profileName == "" {
			if !validProfileWorkspaceName(baseName) {
				return nil, fmt.Errorf("profile workspace scope: profile root name %q must match [a-zA-Z0-9][a-zA-Z0-9._-]*", baseName)
			}
			profileName = baseName
		}
	}
	if profileName == "" {
		profileName = "main"
	}

	workspaceRoot := strings.TrimSpace(opts.WorkspaceRoot)
	if workspaceRoot == "" {
		workspaceRoot = profileRoot
	}
	if workspaceRoot != "" {
		workspaceRoot, err = normalizeRequiredWorkspaceRootWithHome(workspaceRoot, operatorHome)
		if err != nil {
			return nil, fmt.Errorf("profile workspace scope: workspace root: %w", err)
		}
		if err := rejectBlanketAllowedRoot(workspaceRoot, operatorHome, profileRoot); err != nil {
			return nil, fmt.Errorf("profile workspace scope: workspace root: %w", err)
		}
	}

	normalizedRoots := make([]string, 0, len(opts.ProjectRoots)+2)
	for _, root := range []string{workspaceRoot, profileRoot} {
		if root != "" && !containsPath(normalizedRoots, root) {
			normalizedRoots = append(normalizedRoots, root)
		}
	}
	for _, raw := range cleanWorkspaceList(opts.ProjectRoots) {
		root, err := normalizeRequiredWorkspaceRootWithHome(raw, operatorHome)
		if err != nil {
			return nil, fmt.Errorf("profile workspace scope: allowed path %q: %w", raw, err)
		}
		if err := rejectBlanketAllowedRoot(root, operatorHome, profileRoot); err != nil {
			return nil, fmt.Errorf("profile workspace scope: allowed path %q: %w", raw, err)
		}
		if !containsPath(normalizedRoots, root) {
			normalizedRoots = append(normalizedRoots, root)
		}
	}
	if len(normalizedRoots) == 0 {
		return nil, fmt.Errorf("profile workspace scope: profile root is required")
	}

	return &ProfileWorkspaceScope{
		projectRoots: normalizedRoots,
		profileName:  profileName,
		profileRoot:  profileRoot,
		profilesRoot: inferProfilesRoot(profileRoot),
		operatorHome: operatorHome,
		configured:   true,
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
	normalized, err := normalizeWorkspacePathWithHome(rawPath, base, s.operatorHome)
	if err != nil {
		return s.deny("", fmt.Sprintf("resolve path: %v", err))
	}

	for _, root := range s.projectRoots {
		if pathWithinRootForScope(root, normalized) {
			return s.allow(normalized, root)
		}
	}
	return s.deny(normalized, ProfileWorkspaceDeniedMessage)
}

func (s *ProfileWorkspaceScope) allow(path, root string) PathCheckResult {
	if err := ValidateWorkspaceRealPath(root, path); err != nil {
		return s.deny(path, err.Error())
	}
	return PathCheckResult{
		Allowed:    true,
		Normalized: path,
		Root:       root,
		Relative:   WorkspaceRel(root, path),
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
	return normalizeRequiredWorkspaceRootWithHome(path, "")
}

func normalizeRequiredWorkspaceRootWithHome(path, operatorHome string) (string, error) {
	normalized, err := normalizeWorkspacePathWithHome(path, "", operatorHome)
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

func NormalizeWorkspacePath(path, base string) (string, error) {
	return normalizeWorkspacePath(path, base)
}

func normalizeWorkspacePath(path, base string) (string, error) {
	return normalizeWorkspacePathWithHome(path, base, "")
}

func normalizeWorkspacePathWithHome(path, base, operatorHome string) (string, error) {
	expanded, err := expandProfileUserPath(strings.TrimSpace(path), operatorHome)
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
	return EvalPathOrExistingAncestor(filepath.Clean(abs)), nil
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

func validProfileWorkspaceName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return false
	}
	return profileWorkspaceNamePattern.MatchString(name)
}

func rejectBlanketAllowedRoot(root, operatorHome, profileRoot string) error {
	root = filepath.Clean(strings.TrimSpace(root))
	operatorHome = filepath.Clean(strings.TrimSpace(operatorHome))
	profileRoot = filepath.Clean(strings.TrimSpace(profileRoot))
	if root == string(filepath.Separator) {
		return fmt.Errorf("%q grants access to the entire filesystem", root)
	}
	if operatorHome != "." && operatorHome != "" && root == operatorHome && root != profileRoot {
		return fmt.Errorf("%q grants access to the operator home", root)
	}
	return nil
}

func expandProfileUserPath(path string, operatorHome string) (string, error) {
	operatorHome = strings.TrimSpace(operatorHome)
	if path == "~" {
		if operatorHome != "" {
			return operatorHome, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		if operatorHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home directory: %w", err)
			}
			operatorHome = home
		}
		return filepath.Join(operatorHome, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}
