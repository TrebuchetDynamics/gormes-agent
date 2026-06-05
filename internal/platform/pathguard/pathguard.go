package pathguard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CleanRelative normalizes an operator- or manifest-supplied relative path and
// rejects empty, absolute, or traversal paths before callers join it to a root.
func CleanRelative(path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal is not allowed")
	}
	return clean, nil
}

// Within reports whether target resolves to root or one of root's descendants.
func Within(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

// JoinRelative joins a relative path under root and verifies the result stays
// within root after cleaning.
func JoinRelative(root, rel string) (string, error) {
	clean, err := CleanRelative(rel)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(root, clean)
	if !Within(root, joined) {
		return "", fmt.Errorf("path escapes root")
	}
	return filepath.Clean(joined), nil
}
