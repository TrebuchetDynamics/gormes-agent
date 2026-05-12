package sandbox

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// PathFamily represents the access level for a virtual path prefix.
type PathFamily int

const (
	PathFamilyReadOnly  PathFamily = iota // readable paths (e.g. skills)
	PathFamilyReadWrite PathFamily = iota // writable paths (e.g. workspace, uploads)
)

// VirtualPathResolver maps virtual paths under a virtual root to
// real host paths and back, with path-traversal protection.
type VirtualPathResolver struct {
	virtualRoot string
	hostRoot    string
}

// NewVirtualPathResolver creates a resolver that maps paths under
// virtualRoot to corresponding paths under hostRoot.
func NewVirtualPathResolver(virtualRoot, hostRoot string) *VirtualPathResolver {
	return &VirtualPathResolver{
		virtualRoot: cleanPath(virtualRoot),
		hostRoot:    cleanPath(hostRoot),
	}
}

// Resolve converts a virtual path to a host filesystem path.
// Returns an error if the path is outside the virtual root or
// contains traversal components.
func (r *VirtualPathResolver) Resolve(virtualPath string) (string, error) {
	cleaned := cleanPath(virtualPath)

	if !strings.HasPrefix(cleaned, r.virtualRoot) {
		return "", fmt.Errorf("sandbox: path %q is outside virtual root %q", virtualPath, r.virtualRoot)
	}

	rel := strings.TrimPrefix(cleaned, r.virtualRoot)
	rel = strings.TrimPrefix(rel, "/")

	if containsTraversal(rel) {
		return "", fmt.Errorf("sandbox: path traversal detected in %q", virtualPath)
	}

	hostPath := filepath.Join(r.hostRoot, rel)
	return filepath.Clean(hostPath), nil
}

// HostToVirtual converts a host filesystem path back to a virtual path.
// Returns an error if the path is outside the host root.
func (r *VirtualPathResolver) HostToVirtual(hostPath string) (string, error) {
	cleaned := cleanPath(hostPath)

	if !strings.HasPrefix(cleaned, r.hostRoot) {
		return "", fmt.Errorf("sandbox: host path %q is outside host root %q", hostPath, r.hostRoot)
	}

	rel := strings.TrimPrefix(cleaned, r.hostRoot)
	rel = strings.TrimPrefix(rel, "/")

	if containsTraversal(rel) {
		return "", fmt.Errorf("sandbox: path traversal detected in %q", hostPath)
	}

	return r.virtualRoot + "/" + rel, nil
}

// MaskOutput replaces host filesystem paths in the given string with their
// virtual path equivalents, preventing host path leakage into agent return values.
func (r *VirtualPathResolver) MaskOutput(input string) string {
	if input == "" {
		return ""
	}
	// Match the hostRoot optionally followed by path components.
	pattern := regexp.QuoteMeta(r.hostRoot) + `(?:/\S*)?`
	re := regexp.MustCompile(pattern)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		virtual, err := r.HostToVirtual(match)
		if err != nil {
			// Leave unrecognized paths unchanged (e.g. prefix-matching false positives).
			return match
		}
		return virtual
	})
}

// PathFamily returns the access family for a given virtual path.
func (r *VirtualPathResolver) PathFamily(virtualPath string) (PathFamily, error) {
	cleaned := cleanPath(virtualPath)

	if !strings.HasPrefix(cleaned, r.virtualRoot) {
		return PathFamilyReadOnly, fmt.Errorf("sandbox: path %q is outside virtual root", virtualPath)
	}

	rel := strings.TrimPrefix(cleaned, r.virtualRoot)
	rel = strings.TrimPrefix(rel, "/")

	switch {
	case strings.HasPrefix(rel, "workspace/") || rel == "workspace":
		return PathFamilyReadWrite, nil
	case strings.HasPrefix(rel, "uploads/") || rel == "uploads":
		return PathFamilyReadWrite, nil
	case strings.HasPrefix(rel, "outputs/") || rel == "outputs":
		return PathFamilyReadWrite, nil
	default:
		return PathFamilyReadOnly, nil
	}
}

func cleanPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = filepath.ToSlash(p)
	return path.Clean(p)
}

func containsTraversal(rel string) bool {
	parts := strings.Split(rel, "/")
	depth := 0
	for _, part := range parts {
		switch part {
		case "..":
			depth--
		case ".", "":
			continue
		default:
			depth++
		}
	}
	return depth < 0
}
