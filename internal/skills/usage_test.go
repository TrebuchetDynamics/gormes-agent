package skills

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
