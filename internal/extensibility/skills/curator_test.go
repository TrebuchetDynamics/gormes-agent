package skills

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCuratorState_FirstRunDefersAndGates(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	curator := NewCurator(CuratorConfig{Root: root, Now: func() time.Time { return now }, IntervalHours: 24})

	decision, err := curator.ShouldRunNow(context.Background())
	if err != nil {
		t.Fatalf("ShouldRunNow first run: %v", err)
	}
	if decision.Eligible || decision.Code != CuratorEvidenceFirstRunDeferred {
		t.Fatalf("first decision = %+v, want deferred ineligible", decision)
	}
	state, err := curator.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.LastRunAt == nil || !state.LastRunAt.Equal(now) {
		t.Fatalf("LastRunAt = %v, want seeded %s", state.LastRunAt, now)
	}
	if state.RunCount != 0 || !strings.Contains(state.LastRunSummary, "deferred first run") {
		t.Fatalf("state = %+v, want seeded no-run summary", state)
	}

	old := now.Add(-25 * time.Hour)
	if err := curator.SaveState(CuratorState{LastRunAt: &old}); err != nil {
		t.Fatalf("SaveState old: %v", err)
	}
	decision, err = curator.ShouldRunNow(context.Background())
	if err != nil {
		t.Fatalf("ShouldRunNow old: %v", err)
	}
	if !decision.Eligible {
		t.Fatalf("old decision = %+v, want eligible", decision)
	}

	if err := curator.SaveState(CuratorState{LastRunAt: &old, Paused: true}); err != nil {
		t.Fatalf("SaveState paused: %v", err)
	}
	decision, err = curator.ShouldRunNow(context.Background())
	if err != nil {
		t.Fatalf("ShouldRunNow paused: %v", err)
	}
	if decision.Eligible || decision.Code != CuratorEvidencePaused {
		t.Fatalf("paused decision = %+v, want paused", decision)
	}

	disabled := NewCurator(CuratorConfig{Root: root, Disabled: true, Now: func() time.Time { return now }})
	decision, err = disabled.ShouldRunNow(context.Background())
	if err != nil {
		t.Fatalf("ShouldRunNow disabled: %v", err)
	}
	if decision.Eligible || decision.Code != CuratorEvidenceDisabled {
		t.Fatalf("disabled decision = %+v, want disabled", decision)
	}
}

func TestCuratorTransitions_ActivityLifecycle(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	for _, name := range []string{"old", "ancient", "revived", "pinned", "manual"} {
		writeCuratorTestSkill(t, root, name)
	}
	mustUsageState(t, root, map[string]SkillUsageRecord{
		"old": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-50 * 24 * time.Hour),
			LastUsedAt:   now.Add(-45 * 24 * time.Hour),
			State:        SkillStateActive,
		},
		"ancient": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-130 * 24 * time.Hour),
			LastUsedAt:   now.Add(-120 * 24 * time.Hour),
			State:        SkillStateActive,
		},
		"revived": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-120 * 24 * time.Hour),
			LastUsedAt:   now.Add(-time.Hour),
			State:        SkillStateStale,
		},
		"pinned": {
			CreatedBy:    "agent",
			AgentCreated: true,
			Pinned:       true,
			CreatedAt:    now.Add(-130 * 24 * time.Hour),
			LastUsedAt:   now.Add(-120 * 24 * time.Hour),
			State:        SkillStateActive,
		},
		"manual": {
			CreatedAt:  now.Add(-130 * 24 * time.Hour),
			LastUsedAt: now.Add(-120 * 24 * time.Hour),
			State:      SkillStateActive,
		},
	})

	curator := NewCurator(CuratorConfig{
		Root:             root,
		Now:              func() time.Time { return now },
		StaleAfterDays:   30,
		ArchiveAfterDays: 90,
	})
	counts, err := curator.ApplyAutomaticTransitions(context.Background())
	if err != nil {
		t.Fatalf("ApplyAutomaticTransitions: %v", err)
	}
	if counts.Checked != 4 || counts.MarkedStale != 1 || counts.Archived != 1 || counts.Reactivated != 1 {
		t.Fatalf("counts = %+v, want checked/stale/archive/reactivate 4/1/1/1", counts)
	}
	if rec := mustUsageRecord(t, root, "old"); rec.State != SkillStateStale {
		t.Fatalf("old state = %q, want stale", rec.State)
	}
	if rec := mustUsageRecord(t, root, "ancient"); rec.State != SkillStateArchived || rec.ArchivedAt.IsZero() {
		t.Fatalf("ancient record = %+v, want archived with timestamp", rec)
	}
	if _, err := os.Stat(filepath.Join(root, "active", ".archive", "ancient", "SKILL.md")); err != nil {
		t.Fatalf("archived skill missing: %v", err)
	}
	if rec := mustUsageRecord(t, root, "revived"); rec.State != SkillStateActive {
		t.Fatalf("revived state = %q, want active", rec.State)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "pinned", "SKILL.md")); err != nil {
		t.Fatalf("pinned skill should remain active: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "manual", "SKILL.md")); err != nil {
		t.Fatalf("manual skill should remain active: %v", err)
	}
}

func TestCuratorRun_DryRunReportOnly(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	writeCuratorTestSkill(t, root, "old")
	mustUsageState(t, root, map[string]SkillUsageRecord{
		"old": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-120 * 24 * time.Hour),
			LastUsedAt:   now.Add(-120 * 24 * time.Hour),
			State:        SkillStateActive,
		},
	})
	curator := NewCurator(CuratorConfig{
		Root:             root,
		Now:              func() time.Time { return now },
		StaleAfterDays:   30,
		ArchiveAfterDays: 90,
		Reviewer: func(context.Context, CuratorReviewInput) (CuratorReviewResult, error) {
			return CuratorReviewResult{Summary: "dry preview", ToolCalls: []CuratorToolCall{{Name: "skills_list"}}}, nil
		},
	})

	report, err := curator.Run(context.Background(), CuratorRunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}
	if !report.DryRun || report.StateAdvanced || report.BackupID != "" {
		t.Fatalf("dry report = %+v, want report-only with no state advance/backup", report)
	}
	if report.AutoCounts.Archived != 0 || report.AutoCounts.MarkedStale != 0 {
		t.Fatalf("dry auto counts = %+v, want no transitions", report.AutoCounts)
	}
	state, err := curator.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.RunCount != 0 || state.LastRunAt != nil {
		t.Fatalf("dry state = %+v, want run_count=0 and no last_run_at", state)
	}
	if rec := mustUsageRecord(t, root, "old"); rec.State != SkillStateActive {
		t.Fatalf("dry run mutated old state to %q", rec.State)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "old", "SKILL.md")); err != nil {
		t.Fatalf("dry run moved skill: %v", err)
	}
	if report.LastReportPath == "" {
		t.Fatalf("dry run did not write preview report: %+v", report)
	}
}

func TestCuratorRun_ReportEvidence(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	writeCuratorTestSkill(t, root, "active-skill")
	if err := MarkAgentCreated(root, "active-skill"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}
	curator := NewCurator(CuratorConfig{
		Root: root,
		Now:  func() time.Time { return now },
		Reviewer: func(_ context.Context, input CuratorReviewInput) (CuratorReviewResult, error) {
			if !slices.Contains(input.CandidateNames, "active-skill") {
				t.Fatalf("CandidateNames = %v, want active-skill", input.CandidateNames)
			}
			return CuratorReviewResult{
				Summary: "reviewed active-skill",
				ToolCalls: []CuratorToolCall{{
					Name:      "skill_manage",
					Arguments: map[string]string{"action": "patch", "name": "active-skill"},
				}},
			}, nil
		},
	})

	report, err := curator.Run(context.Background(), CuratorRunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.StartedAt.IsZero() || report.Duration <= 0 || report.LastReportPath == "" || report.BackupID == "" {
		t.Fatalf("report missing timing/report/backup evidence: %+v", report)
	}
	if report.Summary != "reviewed active-skill" || len(report.ToolCalls) != 1 {
		t.Fatalf("report summary/tool calls = %q/%v", report.Summary, report.ToolCalls)
	}
	if !slices.Contains(report.BeforeNames, "active-skill") || !slices.Contains(report.AfterNames, "active-skill") {
		t.Fatalf("before/after = %v/%v, want active-skill", report.BeforeNames, report.AfterNames)
	}
	state, err := curator.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.RunCount != 1 || state.LastReportPath != report.LastReportPath || state.LastRunSummary != "reviewed active-skill" {
		t.Fatalf("state = %+v, report = %+v", state, report)
	}
	raw, err := os.ReadFile(report.LastReportPath)
	if err != nil {
		t.Fatalf("read report markdown: %v", err)
	}
	if !strings.Contains(string(raw), "reviewed active-skill") || !strings.Contains(string(raw), "tool_calls: 1") {
		t.Fatalf("report markdown missing summary/tool evidence:\n%s", raw)
	}
}

func TestCuratorPromptTransientEnvironmentGuard(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	writeCuratorTestSkill(t, root, "browser-troubleshooting")
	if err := MarkAgentCreated(root, "browser-troubleshooting"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}

	var gotPrompt string
	curator := NewCurator(CuratorConfig{
		Root: root,
		Now:  func() time.Time { return now },
		Reviewer: func(_ context.Context, input CuratorReviewInput) (CuratorReviewResult, error) {
			gotPrompt = input.Prompt
			return CuratorReviewResult{Summary: "reviewed prompt guard"}, nil
		},
	})
	if _, err := curator.Run(context.Background(), CuratorRunOptions{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	lower := strings.ToLower(gotPrompt)
	for _, want := range []string{
		"do not capture",
		"environment-dependent failures",
		"missing binaries",
		"post-migration path mismatches",
		"unconfigured credentials",
		"negative claims about tools",
		"session-specific transient errors",
		"one-off task narratives",
		"capture the fix",
		"never \"this tool does not work\"",
	} {
		if !strings.Contains(lower, want) {
			t.Fatalf("curator prompt missing %q:\n%s", want, gotPrompt)
		}
	}
}

func TestCuratorBackupAndRollbackSafety(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	skillPath := writeCuratorTestSkill(t, root, "recoverable")
	if err := os.WriteFile(filepath.Join(skillPath, "references.md"), []byte("original"), 0o600); err != nil {
		t.Fatalf("write reference: %v", err)
	}
	if err := MarkAgentCreated(root, "recoverable"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}

	backup, err := CreateCuratorBackup(root, now, "test", nil)
	if err != nil {
		t.Fatalf("CreateCuratorBackup: %v", err)
	}
	if backup.ID == "" || backup.ArchivePath == "" {
		t.Fatalf("backup = %+v, want id and archive path", backup)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "references.md"), []byte("mutated"), 0o600); err != nil {
		t.Fatalf("mutate reference: %v", err)
	}
	rollback, err := RollbackCuratorBackup(root, backup.ID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("RollbackCuratorBackup: %v", err)
	}
	if rollback.PreRollbackBackupID == "" {
		t.Fatalf("rollback = %+v, want undoable pre-rollback backup", rollback)
	}
	raw, err := os.ReadFile(filepath.Join(skillPath, "references.md"))
	if err != nil {
		t.Fatalf("read restored reference: %v", err)
	}
	if string(raw) != "original" {
		t.Fatalf("restored reference = %q, want original", raw)
	}

	maliciousID := "2026-05-06T12-01-00Z"
	maliciousDir := filepath.Join(root, ".curator_backups", maliciousID)
	if err := os.MkdirAll(maliciousDir, 0o700); err != nil {
		t.Fatalf("mkdir malicious backup: %v", err)
	}
	writeMaliciousCuratorTar(t, filepath.Join(maliciousDir, "skills.tar.gz"))
	if _, err := RollbackCuratorBackup(root, maliciousID, now.Add(2*time.Minute)); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("RollbackCuratorBackup malicious err = %v, want unsafe rejection", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pwned")); !os.IsNotExist(err) {
		t.Fatalf("unsafe tar wrote outside root, stat err = %v", err)
	}
}

func TestCuratorClassificationReconciliation(t *testing.T) {
	result := ClassifyRemovedSkills(
		[]string{"declared-cons", "declared-prune", "heuristic-cons", "missing-target"},
		[]string{"umbrella"},
		[]CuratorToolCall{
			{Name: "skill_manage", Arguments: map[string]string{"action": "delete", "name": "declared-cons", "absorbed_into": "umbrella"}},
			{Name: "skill_manage", Arguments: map[string]string{"action": "delete", "name": "declared-prune", "absorbed_into": ""}},
			{Name: "skill_manage", Arguments: map[string]string{"action": "write_file", "name": "umbrella", "file_path": "references/heuristic-cons.md", "file_content": "details"}},
			{Name: "skill_manage", Arguments: map[string]string{"action": "delete", "name": "missing-target", "absorbed_into": "ghost"}},
		},
	)
	if got := result.Consolidated["declared-cons"]; got.Into != "umbrella" || got.Source != "absorbed_into" {
		t.Fatalf("declared cons = %+v, want absorbed_into umbrella", got)
	}
	if got := result.Consolidated["heuristic-cons"]; got.Into != "umbrella" || got.Source != "heuristic" {
		t.Fatalf("heuristic cons = %+v, want heuristic umbrella", got)
	}
	if !slices.Contains(result.Pruned, "declared-prune") || !slices.Contains(result.Pruned, "missing-target") {
		t.Fatalf("pruned = %v, want declared-prune and missing-target", result.Pruned)
	}
}

func TestCuratorRenameSummary(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	for _, name := range []string{"pdf-extraction", "docx-extraction", "old-stale-thing", "document-tools"} {
		writeCuratorTestSkill(t, root, name)
	}
	mustUsageState(t, root, map[string]SkillUsageRecord{
		"pdf-extraction": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-24 * time.Hour),
			LastUsedAt:   now.Add(-24 * time.Hour),
			State:        SkillStateActive,
		},
		"docx-extraction": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-24 * time.Hour),
			LastUsedAt:   now.Add(-24 * time.Hour),
			State:        SkillStateActive,
		},
		"old-stale-thing": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-24 * time.Hour),
			LastUsedAt:   now.Add(-24 * time.Hour),
			State:        SkillStateActive,
		},
		"document-tools": {
			CreatedBy:    "agent",
			AgentCreated: true,
			CreatedAt:    now.Add(-24 * time.Hour),
			LastUsedAt:   now.Add(-24 * time.Hour),
			State:        SkillStateActive,
		},
	})
	curator := NewCurator(CuratorConfig{
		Root: root,
		Now:  func() time.Time { return now },
		Reviewer: func(context.Context, CuratorReviewInput) (CuratorReviewResult, error) {
			for _, name := range []string{"pdf-extraction", "docx-extraction", "old-stale-thing"} {
				if _, err := ArchiveAgentCreatedSkill(root, name, now); err != nil {
					t.Fatalf("ArchiveAgentCreatedSkill(%s): %v", name, err)
				}
			}
			return CuratorReviewResult{
				Summary: "auto: 1 marked stale; llm: consolidated 2 into 1, pruned 1",
				ToolCalls: []CuratorToolCall{
					{Name: "skill_manage", Arguments: map[string]string{"action": "delete", "name": "pdf-extraction", "absorbed_into": "document-tools"}},
					{Name: "skill_manage", Arguments: map[string]string{"action": "delete", "name": "docx-extraction", "absorbed_into": "document-tools"}},
					{Name: "skill_manage", Arguments: map[string]string{"action": "delete", "name": "old-stale-thing", "absorbed_into": ""}},
				},
			}, nil
		},
	})

	report, err := curator.Run(context.Background(), CuratorRunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, want := range []string{
		"auto: 1 marked stale; llm: consolidated 2 into 1, pruned 1",
		"archived 3 skill(s):",
		"pdf-extraction → document-tools",
		"docx-extraction → document-tools",
		"old-stale-thing — pruned (stale)",
		"full report: gormes curator status",
	} {
		if !strings.Contains(report.Summary, want) {
			t.Fatalf("summary missing %q:\n%s", want, report.Summary)
		}
	}
	state, err := curator.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.LastRunSummary != report.Summary {
		t.Fatalf("state summary = %q, want report summary %q", state.LastRunSummary, report.Summary)
	}
}

func TestCuratorRenameSummaryCapsLargeRunsAndSkipsNoop(t *testing.T) {
	if got := buildCuratorRenameSummary(CuratorClassification{}); got != "" {
		t.Fatalf("noop rename summary = %q, want empty", got)
	}

	classification := CuratorClassification{
		Consolidated: map[string]CuratorConsolidation{},
	}
	for i := 0; i < 15; i++ {
		name := "skill-" + string(rune('a'+i))
		classification.Consolidated[name] = CuratorConsolidation{Into: "umbrella", Source: "absorbed_into"}
	}
	got := buildCuratorRenameSummary(classification)
	if !strings.Contains(got, "archived 15 skill(s):") || !strings.Contains(got, "… and 5 more") {
		t.Fatalf("capped summary missing count/omitted line:\n%s", got)
	}
	if bullets := strings.Count(got, "  • "); bullets != 10 {
		t.Fatalf("bullet count = %d, want 10 in capped summary:\n%s", bullets, got)
	}
}

func writeCuratorTestSkill(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	body := "---\nname: " + name + "\ndescription: test skill\n---\n\nUse it.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	return dir
}

func mustUsageState(t *testing.T, root string, state map[string]SkillUsageRecord) {
	t.Helper()
	if err := saveUsageState(root, state); err != nil {
		t.Fatalf("saveUsageState: %v", err)
	}
}

func mustUsageRecord(t *testing.T, root, name string) SkillUsageRecord {
	t.Helper()
	rec, err := GetUsageRecord(root, name)
	if err != nil {
		t.Fatalf("GetUsageRecord(%q): %v", name, err)
	}
	return rec
}

func writeMaliciousCuratorTar(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create malicious tar: %v", err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	body := []byte("nope")
	if err := tw.WriteHeader(&tar.Header{Name: "../pwned", Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatalf("write malicious header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write malicious body: %v", err)
	}
}

func TestCuratorReportJSONRoundTrip(t *testing.T) {
	report := CuratorRunReport{Summary: "ok", AutoCounts: CuratorTransitionCounts{Checked: 1}}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal report: %v", err)
	}
	var got CuratorRunReport
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal report: %v", err)
	}
	if got.Summary != "ok" || got.AutoCounts.Checked != 1 {
		t.Fatalf("round trip = %+v", got)
	}
}
