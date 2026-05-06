package skills

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type UsageLogger struct {
	path string
	mu   sync.Mutex
}

type usageRecord struct {
	SkillName string    `json:"skill_name"`
	UsedAt    time.Time `json:"used_at"`
}

var ErrUsageRecordNotFound = errors.New("skill usage record not found")

// SkillUsageRecord is the curator-facing usage/provenance sidecar state for a
// skill. It intentionally lives outside SKILL.md so telemetry never rewrites
// user-authored instructions.
type SkillUsageRecord struct {
	CreatedBy     string    `json:"created_by,omitempty"`
	AgentCreated  bool      `json:"agent_created,omitempty"`
	Pinned        bool      `json:"pinned,omitempty"`
	State         string    `json:"state,omitempty"`
	UseCount      int       `json:"use_count,omitempty"`
	ViewCount     int       `json:"view_count,omitempty"`
	PatchCount    int       `json:"patch_count,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	LastUsedAt    time.Time `json:"last_used_at,omitempty"`
	LastViewedAt  time.Time `json:"last_viewed_at,omitempty"`
	LastPatchedAt time.Time `json:"last_patched_at,omitempty"`
	ArchivedAt    time.Time `json:"archived_at,omitempty"`
}

const (
	SkillStateActive   = "active"
	SkillStateStale    = "stale"
	SkillStateArchived = "archived"
)

func NewUsageLogger(path string) *UsageLogger {
	if path == "" {
		return nil
	}
	return &UsageLogger{path: path}
}

func (l *UsageLogger) Record(ctx context.Context, skillNames []string) error {
	if l == nil || len(skillNames) == 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, name := range skillNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		raw, err := json.Marshal(usageRecord{
			SkillName: name,
			UsedAt:    time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		if _, err := f.Write(append(raw, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func usageStatePath(root string) string {
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".usage.json")
}

func loadUsageState(root string) (map[string]SkillUsageRecord, error) {
	path := usageStatePath(root)
	if path == "" {
		return map[string]SkillUsageRecord{}, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]SkillUsageRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]SkillUsageRecord{}, nil
	}
	state := map[string]SkillUsageRecord{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func saveUsageState(root string, state map[string]SkillUsageRecord) error {
	path := usageStatePath(root)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".usage.json.tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func updateUsageRecord(root, name string, fn func(*SkillUsageRecord)) error {
	state, err := loadUsageState(root)
	if err != nil {
		return err
	}
	rec := state[name]
	fn(&rec)
	state[name] = rec
	return saveUsageState(root, state)
}

func GetUsageRecord(root, name string) (SkillUsageRecord, error) {
	state, err := loadUsageState(root)
	if err != nil {
		return SkillUsageRecord{}, err
	}
	rec, ok := state[name]
	if !ok {
		return SkillUsageRecord{}, ErrUsageRecordNotFound
	}
	return rec, nil
}

func MarkAgentCreated(root, name string) error {
	now := time.Now().UTC()
	return updateUsageRecord(root, name, func(rec *SkillUsageRecord) {
		rec.CreatedBy = "agent"
		rec.AgentCreated = true
		if rec.CreatedAt.IsZero() {
			rec.CreatedAt = now
		}
		if rec.State == "" {
			rec.State = SkillStateActive
		}
	})
}

func IsAgentCreated(root, name string) bool {
	rec, err := GetUsageRecord(root, name)
	if err != nil {
		return false
	}
	return rec.CreatedBy == "agent" || rec.AgentCreated
}

func SetPinned(root, name string, pinned bool) error {
	return updateUsageRecord(root, name, func(rec *SkillUsageRecord) {
		rec.Pinned = pinned
		if rec.State == "" {
			rec.State = SkillStateActive
		}
	})
}

func IsPinned(root, name string) (bool, error) {
	rec, err := GetUsageRecord(root, name)
	if errors.Is(err, ErrUsageRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return rec.Pinned, nil
}

func BumpPatch(root, name string) error {
	now := time.Now().UTC()
	return updateUsageRecord(root, name, func(rec *SkillUsageRecord) {
		rec.PatchCount++
		rec.LastPatchedAt = now
		if rec.State == "" {
			rec.State = SkillStateActive
		}
	})
}

func BumpUse(root, name string) error {
	now := time.Now().UTC()
	return updateUsageRecord(root, name, func(rec *SkillUsageRecord) {
		rec.UseCount++
		rec.LastUsedAt = now
		if rec.State == "" || rec.State == SkillStateStale {
			rec.State = SkillStateActive
		}
	})
}

func BumpView(root, name string) error {
	now := time.Now().UTC()
	return updateUsageRecord(root, name, func(rec *SkillUsageRecord) {
		rec.ViewCount++
		rec.LastViewedAt = now
		if rec.State == "" {
			rec.State = SkillStateActive
		}
	})
}

func SetSkillState(root, name, state string) error {
	now := time.Now().UTC()
	return updateUsageRecord(root, name, func(rec *SkillUsageRecord) {
		rec.State = state
		if state == SkillStateArchived && rec.ArchivedAt.IsZero() {
			rec.ArchivedAt = now
		}
		if state == SkillStateActive {
			rec.ArchivedAt = time.Time{}
		}
	})
}

func ForgetUsageRecord(root, name string) error {
	state, err := loadUsageState(root)
	if err != nil {
		return err
	}
	delete(state, name)
	return saveUsageState(root, state)
}

type AgentCreatedSkillUsage struct {
	Name           string
	Record         SkillUsageRecord
	SkillDir       string
	LastActivityAt time.Time
}

func ListAgentCreatedSkillUsage(root string) ([]AgentCreatedSkillUsage, error) {
	state, err := loadUsageState(root)
	if err != nil {
		return nil, err
	}
	out := make([]AgentCreatedSkillUsage, 0, len(state))
	for name, rec := range state {
		if rec.CreatedBy != "agent" && !rec.AgentCreated {
			continue
		}
		dir, ok := findActiveSkillDir(root, name)
		if !ok {
			continue
		}
		if isArchivedSkillPath(root, dir) || isHubSkillPath(root, dir) || isBundledSkill(root, name) {
			continue
		}
		if rec.State == "" {
			rec.State = SkillStateActive
		}
		out = append(out, AgentCreatedSkillUsage{
			Name:           name,
			Record:         rec,
			SkillDir:       dir,
			LastActivityAt: lastSkillActivity(rec),
		})
	}
	return out, nil
}

func lastSkillActivity(rec SkillUsageRecord) time.Time {
	last := rec.LastUsedAt
	for _, candidate := range []time.Time{rec.LastViewedAt, rec.LastPatchedAt, rec.CreatedAt} {
		if candidate.After(last) {
			last = candidate
		}
	}
	return last
}

func findActiveSkillDir(root, name string) (string, bool) {
	active := filepath.Join(root, "active")
	var found string
	_ = filepath.WalkDir(active, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".archive" || d.Name() == ".hub" {
			return filepath.SkipDir
		}
		if d.Name() != name {
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(path, "SKILL.md")); statErr == nil {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found, found != ""
}

func isArchivedSkillPath(root, path string) bool {
	rel, err := filepath.Rel(filepath.Join(root, "active"), path)
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	return len(parts) > 0 && parts[0] == ".archive"
}

func isHubSkillPath(root, path string) bool {
	rel, err := filepath.Rel(filepath.Join(root, "active"), path)
	if err != nil {
		return false
	}
	return rel == ".hub" || strings.HasPrefix(rel, ".hub"+string(filepath.Separator))
}

func isBundledSkill(root, name string) bool {
	for _, manifest := range []string{
		filepath.Join(root, ".bundled_manifest"),
		filepath.Join(root, "active", ".bundled_manifest"),
	} {
		raw, err := os.ReadFile(manifest)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if line == name || strings.HasPrefix(line, name+":") || strings.HasPrefix(line, name+" ") || strings.HasPrefix(line, name+"\t") {
				return true
			}
		}
	}
	return false
}
