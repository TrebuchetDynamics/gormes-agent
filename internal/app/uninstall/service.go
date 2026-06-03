package uninstall

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArtifactGroup describes one uninstall category and the paths in it.
type ArtifactGroup struct {
	Name  string
	Paths []string
}

// CollectPublishedBinaryPaths enumerates install-published PATH symlinks
// that point back into the managed Gormes home. Real binaries are left alone.
func CollectPublishedBinaryPaths(home string) []string {
	if home == "" {
		return nil
	}
	candidates := PublishedBinaryCandidates()
	homeAbs, _ := filepath.Abs(home)
	out := make([]string, 0, len(candidates))
	seen := make(map[string]bool)
	for _, path := range candidates {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(path)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			continue
		}
		if homeAbs != "" && (targetAbs == homeAbs || strings.HasPrefix(targetAbs, homeAbs+string(os.PathSeparator))) {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func PublishedBinaryCandidates() []string {
	const exe = "gormes"
	candidates := make([]string, 0, 4)
	if dir := strings.TrimSpace(os.Getenv("GORMES_BIN_DIR")); dir != "" {
		candidates = append(candidates, filepath.Join(dir, exe))
	}
	if prefix := strings.TrimSpace(os.Getenv("GORMES_PREFIX")); prefix != "" {
		candidates = append(candidates, filepath.Join(prefix, "bin", exe))
	}
	if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
		candidates = append(candidates, filepath.Join(userHome, ".local", "bin", exe))
	}
	candidates = append(candidates, filepath.Join("/usr", "local", "bin", exe))
	return candidates
}

func SortedExisting(paths ...string) []string {
	existing := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		entry := p
		if info.IsDir() {
			entry += "/"
		}
		existing = append(existing, entry)
	}
	sort.Strings(existing)
	return existing
}

func RemoveGroup(groups []ArtifactGroup, name string) []ArtifactGroup {
	out := make([]ArtifactGroup, 0, len(groups))
	for _, g := range groups {
		if g.Name != name {
			out = append(out, g)
		}
	}
	return out
}
