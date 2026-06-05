package prefill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// Load loads a JSON array of Hermes messages. Empty, missing, invalid, or
// non-array files degrade to no prefill messages, matching Hermes' nonfatal
// behavior. Relative paths resolve from baseHome, the Go-native equivalent of
// Hermes resolving from ~/.hermes.
func Load(filePath, baseHome string) ([]llm.Message, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, nil
	}
	path := os.ExpandEnv(filePath)
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseHome, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var messages []llm.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, nil
	}
	if messages == nil {
		return nil, nil
	}
	return messages, nil
}
