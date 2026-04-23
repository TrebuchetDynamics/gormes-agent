package skills

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type promptSnapshot struct {
	GeneratedAt time.Time             `json:"generated_at"`
	UserMessage string                `json:"user_message"`
	Block       string                `json:"block"`
	Skills      []promptSnapshotSkill `json:"skills"`
}

type promptSnapshotSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type promptSnapshotWriter struct {
	path string
	mu   sync.Mutex
}

func newPromptSnapshotWriter(path string) *promptSnapshotWriter {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	return &promptSnapshotWriter{path: path}
}

func (w *promptSnapshotWriter) Write(userMessage, block string, skills []Skill) error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	doc := promptSnapshot{
		GeneratedAt: time.Now().UTC(),
		UserMessage: userMessage,
		Block:       block,
		Skills:      make([]promptSnapshotSkill, 0, len(skills)),
	}
	for _, skill := range skills {
		doc.Skills = append(doc.Skills, promptSnapshotSkill{
			Name:        skill.Name,
			Description: skill.Description,
			Path:        skill.Path,
		})
	}

	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}
	return writeJSON(w.path, doc)
}
