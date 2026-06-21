// Package credentialfiles provides a session-scoped registry of host credential
// files that must be mounted into remote sandbox environments (Docker, SSH,
// Modal). It mirrors Hermes' tools/credential_files.py register/mount/skills
// contract, mapping relative GORMES_HOME paths to container-absolute paths.
package credentialfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Mount describes a resolved mount from a host path to a container path.
type Mount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// Registry is a session-scoped allowlist of credential files to mount into
// remote sandboxes. It is safe for concurrent use.
type Registry struct {
	mu         sync.Mutex
	gormesHome string
	// container_path → host_path
	entries map[string]string
}

// NewRegistry creates a Registry rooted at gormesHome and pre-registers any
// paths provided by configuredPaths (from config.terminal.credential_files).
// configuredPaths entries that are not found on disk are silently skipped.
func NewRegistry(gormesHome string, configuredPaths []string) *Registry {
	r := &Registry{
		gormesHome: filepath.Clean(gormesHome),
		entries:    make(map[string]string),
	}
	for _, p := range configuredPaths {
		_, _ = r.register(p, "/root/.gormes")
	}
	return r
}

// Register adds relativePath (relative to GORMES_HOME) to the registry with
// the given containerBase prefix. Returns false when the path is missing,
// an absolute path, or contains a traversal sequence. Returns an error only
// for programmer-level invariant violations (empty gormesHome).
func (r *Registry) Register(relativePath, containerBase string) (bool, error) {
	return r.register(relativePath, containerBase)
}

func (r *Registry) register(relativePath, containerBase string) (bool, error) {
	if r.gormesHome == "" {
		return false, fmt.Errorf("credentialfiles: Registry has no gormesHome")
	}
	relPath := strings.TrimSpace(relativePath)
	if relPath == "" {
		return false, nil
	}
	// Reject absolute paths — they bypass the GORMES_HOME sandbox.
	if filepath.IsAbs(relPath) {
		return false, nil
	}
	hostPath := filepath.Join(r.gormesHome, relPath)
	// Resolve symlinks and normalise before containment check to prevent
	// traversal sequences like ../../.ssh/id_rsa from escaping GORMES_HOME.
	resolved, err := filepath.EvalSymlinks(hostPath)
	if err != nil {
		// File not found is expected; treat as not-registered.
		return false, nil
	}
	if err := validateWithinDir(r.gormesHome, resolved); err != nil {
		return false, nil
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return false, nil
	}
	base := strings.TrimRight(containerBase, "/")
	if base == "" {
		base = "/root/.gormes"
	}
	containerPath := base + "/" + filepath.ToSlash(relPath)
	r.mu.Lock()
	r.entries[containerPath] = resolved
	r.mu.Unlock()
	return true, nil
}

// RegisterMany registers a batch of relativePaths and returns the slice of
// paths that were NOT found on disk. Entries that contain security violations
// (absolute, traversal) are also included in missing.
func (r *Registry) RegisterMany(relativePaths []string, containerBase string) []string {
	var missing []string
	for _, p := range relativePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		ok, _ := r.Register(p, containerBase)
		if !ok {
			missing = append(missing, p)
		}
	}
	return missing
}

// Mounts returns all registered credential file mounts in a stable order.
func (r *Registry) Mounts() []Mount {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.entries))
	for k := range r.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Mount, 0, len(keys))
	for _, containerPath := range keys {
		out = append(out, Mount{
			HostPath:      r.entries[containerPath],
			ContainerPath: containerPath,
			ReadOnly:      true,
		})
	}
	return out
}

// Clear removes all registered entries.
func (r *Registry) Clear() {
	r.mu.Lock()
	r.entries = make(map[string]string)
	r.mu.Unlock()
}

// Len returns the number of registered entries.
func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

// SkillsDirectoryMount returns the mount that maps the GORMES_HOME skills
// directory to the container. Returns nil when the skills directory does not
// exist. containerBase defaults to "/root/.gormes" when empty.
func SkillsDirectoryMount(gormesHome, containerBase string) *Mount {
	base := strings.TrimRight(containerBase, "/")
	if base == "" {
		base = "/root/.gormes"
	}
	skillsDir := filepath.Join(gormesHome, "skills")
	info, err := os.Stat(skillsDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	return &Mount{
		HostPath:      skillsDir,
		ContainerPath: base + "/skills",
		ReadOnly:      true,
	}
}

// IterSkillsFiles calls fn for each file under the GORMES_HOME skills
// directory, passing the host path and the container-relative path.
// Skips directories and files that cannot be stat'd. Iteration order is
// lexicographic. containerBase defaults to "/root/.gormes" when empty.
func IterSkillsFiles(gormesHome, containerBase string, fn func(hostPath, containerPath string)) error {
	base := strings.TrimRight(containerBase, "/")
	if base == "" {
		base = "/root/.gormes"
	}
	skillsDir := filepath.Join(gormesHome, "skills")
	return filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(skillsDir, path)
		if relErr != nil {
			return nil
		}
		containerPath := base + "/skills/" + filepath.ToSlash(rel)
		fn(path, containerPath)
		return nil
	})
}

// validateWithinDir returns an error when candidate is not within root.
func validateWithinDir(root, candidate string) error {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	if rel == "." || rel == "" ||
		strings.HasPrefix(rel, ".."+string(os.PathSeparator)) ||
		rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes GORMES_HOME")
	}
	return nil
}
