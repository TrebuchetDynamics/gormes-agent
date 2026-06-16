package skills

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUsageLoggerAppendsOneJSONLRecordPerSkill(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.jsonl")
	logger := NewUsageLogger(path)

	if err := logger.Record(context.Background(), []string{"careful-review"}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("line count = %d, want 1", len(lines))
	}
	if !strings.Contains(lines[0], `"skill_name":"careful-review"`) {
		t.Fatalf("usage line = %q, want skill name", lines[0])
	}
}

func TestUsageLoggerSkipsEmptySelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.jsonl")
	logger := NewUsageLogger(path)

	if err := logger.Record(context.Background(), nil); err != nil {
		t.Fatalf("Record(nil) error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("usage file exists after empty record, stat err = %v", err)
	}
}

func TestSkillUsageAgentCreatedPinnedPatchAndForget(t *testing.T) {
	root := t.TempDir()

	if err := MarkAgentCreated(root, "agent-skill"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}
	if !IsAgentCreated(root, "agent-skill") {
		t.Fatal("agent-skill not marked agent-created")
	}

	if err := SetPinned(root, "agent-skill", true); err != nil {
		t.Fatalf("SetPinned(true): %v", err)
	}
	pinned, err := IsPinned(root, "agent-skill")
	if err != nil {
		t.Fatalf("IsPinned: %v", err)
	}
	if !pinned {
		t.Fatal("agent-skill not pinned")
	}

	if err := BumpPatch(root, "agent-skill"); err != nil {
		t.Fatalf("BumpPatch: %v", err)
	}
	record, err := GetUsageRecord(root, "agent-skill")
	if err != nil {
		t.Fatalf("GetUsageRecord: %v", err)
	}
	if record.CreatedBy != "agent" || !record.AgentCreated || record.PatchCount != 1 || !record.Pinned {
		t.Fatalf("usage record = %+v, want agent-created pinned patch_count=1", record)
	}

	if err := ForgetUsageRecord(root, "agent-skill"); err != nil {
		t.Fatalf("ForgetUsageRecord: %v", err)
	}
	if _, err := GetUsageRecord(root, "agent-skill"); !errors.Is(err, ErrUsageRecordNotFound) {
		t.Fatalf("GetUsageRecord after forget err = %v, want ErrUsageRecordNotFound", err)
	}
}

func TestArchiveAgentCreatedSkillAndListArchivedNames(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, "active")
	for _, name := range []string{"agent-skill", "pinned-skill", "manual-skill"} {
		writeSkillDoc(t, filepath.Join(active, name, "SKILL.md"), name, name+" description", "# "+name)
	}
	if err := MarkAgentCreated(root, "agent-skill"); err != nil {
		t.Fatalf("MarkAgentCreated agent-skill: %v", err)
	}
	if err := MarkAgentCreated(root, "pinned-skill"); err != nil {
		t.Fatalf("MarkAgentCreated pinned-skill: %v", err)
	}
	if err := SetPinned(root, "pinned-skill", true); err != nil {
		t.Fatalf("SetPinned pinned-skill: %v", err)
	}

	if _, err := ArchiveAgentCreatedSkill(root, "pinned-skill", time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("ArchiveAgentCreatedSkill pinned err = %v, want pinned refusal", err)
	}
	if _, err := ArchiveAgentCreatedSkill(root, "manual-skill", time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "bundled or hub-installed") {
		t.Fatalf("ArchiveAgentCreatedSkill manual err = %v, want provenance refusal", err)
	}

	archivedAt := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	dest, err := ArchiveAgentCreatedSkill(root, "agent-skill", archivedAt)
	if err != nil {
		t.Fatalf("ArchiveAgentCreatedSkill agent-skill: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(dest), "active/.archive/agent-skill") {
		t.Fatalf("archive dest = %q, want active/.archive/agent-skill", dest)
	}
	if _, err := os.Stat(filepath.Join(active, ".archive", "agent-skill", "SKILL.md")); err != nil {
		t.Fatalf("archived skill missing on disk: %v", err)
	}
	rec, err := GetUsageRecord(root, "agent-skill")
	if err != nil {
		t.Fatalf("GetUsageRecord agent-skill: %v", err)
	}
	if rec.State != SkillStateArchived || !rec.ArchivedAt.Equal(archivedAt) {
		t.Fatalf("usage record = %+v, want archived at %s", rec, archivedAt)
	}
	names, err := ListArchivedSkillNames(root)
	if err != nil {
		t.Fatalf("ListArchivedSkillNames: %v", err)
	}
	if len(names) != 1 || names[0] != "agent-skill" {
		t.Fatalf("archived names = %v, want [agent-skill]", names)
	}
}
