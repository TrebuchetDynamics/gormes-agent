package skills

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/usage"
)

type UsageLogger = usage.UsageLogger
type SkillUsageRecord = usage.SkillUsageRecord
type AgentCreatedSkillUsage = usage.AgentCreatedSkillUsage

var ErrUsageRecordNotFound = usage.ErrUsageRecordNotFound

const (
	SkillStateActive   = usage.SkillStateActive
	SkillStateStale    = usage.SkillStateStale
	SkillStateArchived = usage.SkillStateArchived
)

func NewUsageLogger(path string) *UsageLogger { return usage.NewUsageLogger(path) }
func GetUsageRecord(root, name string) (SkillUsageRecord, error) {
	return usage.GetUsageRecord(root, name)
}
func MarkAgentCreated(root, name string) error       { return usage.MarkAgentCreated(root, name) }
func IsAgentCreated(root, name string) bool          { return usage.IsAgentCreated(root, name) }
func SetPinned(root, name string, pinned bool) error { return usage.SetPinned(root, name, pinned) }
func IsPinned(root, name string) (bool, error)       { return usage.IsPinned(root, name) }
func BumpPatch(root, name string) error              { return usage.BumpPatch(root, name) }
func BumpUse(root, name string) error                { return usage.BumpUse(root, name) }
func BumpView(root, name string) error               { return usage.BumpView(root, name) }
func SetSkillState(root, name, state string) error   { return usage.SetSkillState(root, name, state) }
func ForgetUsageRecord(root, name string) error      { return usage.ForgetUsageRecord(root, name) }
func ListAgentCreatedSkillUsage(root string) ([]AgentCreatedSkillUsage, error) {
	return usage.ListAgentCreatedSkillUsage(root)
}
func ListArchivedSkillNames(root string) ([]string, error) { return usage.ListArchivedSkillNames(root) }
func ArchiveAgentCreatedSkill(root, name string, now time.Time) (string, error) {
	return usage.ArchiveAgentCreatedSkill(root, name, now)
}

func usageStatePath(root string) string { return usage.UsageStatePath(root) }
func loadUsageState(root string) (map[string]SkillUsageRecord, error) {
	return usage.LoadUsageState(root)
}
func saveUsageState(root string, state map[string]SkillUsageRecord) error {
	return usage.SaveUsageState(root, state)
}
func updateUsageRecord(root, name string, fn func(*SkillUsageRecord)) error {
	return usage.UpdateUsageRecord(root, name, fn)
}
func moveSkillDirToArchive(root, name, skillDir string, now time.Time) (string, time.Time, error) {
	return usage.MoveSkillDirToArchive(root, name, skillDir, now)
}
