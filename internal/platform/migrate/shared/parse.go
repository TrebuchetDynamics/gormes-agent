// Package shared contains migration helpers used by source-specific
// migration packages. It intentionally stays small and behavior-preserving:
// helpers live here only after Hermes and OpenClaw need the same mechanics.
package shared

import (
	"bufio"
	"io"
	"os"
	"sort"
	"strings"
)

// DirExists reports whether path exists and is a directory.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// SortedStringAnyKeys returns sorted keys for deterministic manifest output.
func SortedStringAnyKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ReadDotenvKeys reads unique dotenv keys in the same limited syntax used by
// dry-run manifests: blank lines and comments are ignored, the first '=' splits
// key from value, and duplicate keys keep their first occurrence. Secret values
// are intentionally discarded by this helper.
func ReadDotenvKeys(r io.Reader) ([]string, error) {
	seen := make(map[string]bool)
	keys := []string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Strings(keys)
	return keys, nil
}
