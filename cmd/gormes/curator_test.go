package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestCuratorCommand_Status(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "top-dog")
	writeCuratorCommandSkill(t, root, "middling")
	writeCuratorCommandSkill(t, root, "never-used")
	for _, name := range []string{"top-dog", "middling", "never-used"} {
		if err := skills.MarkAgentCreated(root, name); err != nil {
			t.Fatalf("MarkAgentCreated(%s): %v", name, err)
		}
	}
	for i := 0; i < 10; i++ {
		if err := skills.BumpUse(root, "top-dog"); err != nil {
			t.Fatalf("BumpUse top-dog: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := skills.BumpUse(root, "middling"); err != nil {
			t.Fatalf("BumpUse middling: %v", err)
		}
	}
	lastRun := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:      &lastRun,
		LastRunSummary: "curator run completed",
		LastReportPath: filepath.Join(root, "logs", "curator", "REPORT.md"),
		RunCount:       3,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "status")
	if err != nil {
		t.Fatalf("curator status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"curator: ENABLED",
		"runs:           3",
		"last summary:   curator run completed",
		"interval:       every 7d",
		"stale after:    30d unused",
		"archive after:  90d unused",
		"agent-created skills: 3 total",
		"least recently active (top 5):",
		"most active (top 5):",
		"least active (top 5):",
		"top-dog",
		"activity= 10",
		"never-used",
		"activity=  0",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("curator status stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCuratorCommand_Effectiveness(t *testing.T) {
	root := setupCuratorCommandHome(t)
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	ledger := skills.NewSkillEffectivenessLedger(skills.SkillEffectivenessLedgerPath(root), func() time.Time { return now })
	for _, event := range []skills.SkillEffectivenessEvent{
		{
			SkillName:        "planner",
			SessionID:        "sess-1",
			TurnID:           "turn-1",
			Prompt:           "please plan this without leaking token ghp_secret",
			SelectionSource:  "hybrid",
			LexicalScore:     25,
			SemanticScore:    0.5,
			TotalScore:       1.2,
			Outcome:          skills.SkillOutcomePositive,
			OperatorFeedback: "great result, api key sk-live-secret must stay hidden",
			FeedbackReason:   "operator_explicit",
			RecordedAt:       now.Add(-time.Hour),
		},
		{
			SkillName:  "stale-skill",
			SessionID:  "sess-1",
			TurnID:     "turn-old",
			Prompt:     "old selection",
			Outcome:    skills.SkillOutcomePositive,
			RecordedAt: now.Add(-15 * 24 * time.Hour),
		},
		{
			SkillName:  "bad-skill",
			SessionID:  "sess-1",
			TurnID:     "turn-2",
			Outcome:    skills.SkillOutcomeNegative,
			RecordedAt: now.Add(-time.Hour),
		},
	} {
		if err := ledger.Record(context.Background(), event); err != nil {
			t.Fatalf("Record(%s): %v", event.SkillName, err)
		}
	}
	f, err := os.OpenFile(skills.SkillEffectivenessLedgerPath(root), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open ledger for corrupt append: %v", err)
	}
	if _, err := f.WriteString("{not-json}\n"); err != nil {
		f.Close()
		t.Fatalf("append corrupt line: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close corrupt append: %v", err)
	}

	cmd := newCuratorCommandWithDeps(curatorCommandDeps{
		skillsRoot: func() string { return root },
		now:        func() time.Time { return now },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "effectiveness", "--json")
	if err != nil {
		t.Fatalf("curator effectiveness --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build      buildProvenanceJSON `json:"build"`
		LedgerPath string              `json:"ledger_path"`
		Invalid    []struct {
			Line  int    `json:"line"`
			Error string `json:"error"`
		} `json:"invalid"`
		Scores []skills.SkillEffectivenessScore `json:"scores"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator effectiveness --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Fatalf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.LedgerPath != skills.SkillEffectivenessLedgerPath(root) {
		t.Fatalf("ledger_path = %q, want %q", got.LedgerPath, skills.SkillEffectivenessLedgerPath(root))
	}
	if len(got.Invalid) != 1 || got.Invalid[0].Line != 4 || got.Invalid[0].Error == "" {
		t.Fatalf("invalid evidence = %+v, want one corrupt line", got.Invalid)
	}
	if got.Scores[0].SkillName != "planner" || got.Scores[0].Score <= got.Scores[1].Score {
		t.Fatalf("scores = %+v, want planner first with highest positive feedback score", got.Scores)
	}
	if !containsString(got.Scores[0].ReasonCodes, skills.SkillEffectivenessReasonOperatorFeedback) {
		t.Fatalf("planner reasons = %v, want operator feedback", got.Scores[0].ReasonCodes)
	}
	if !containsString(got.Scores[1].ReasonCodes, skills.SkillEffectivenessReasonStaleDecay) {
		t.Fatalf("stale-skill reasons = %v, want stale decay", got.Scores[1].ReasonCodes)
	}
	for _, forbidden := range []string{"ghp_secret", "sk-live-secret", "please plan this", "great result"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("effectiveness JSON leaked raw prompt or feedback %q:\n%s", forbidden, stdout)
		}
	}

	cmd = newCuratorCommandWithDeps(curatorCommandDeps{
		skillsRoot: func() string { return root },
		now:        func() time.Time { return now },
	})
	stdout, stderr, err = executeRootCommandForTest(cmd, "effectiveness")
	if err != nil {
		t.Fatalf("curator effectiveness: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"curator effectiveness: 3 skill(s), 1 invalid record(s)",
		"planner",
		"score=125.00",
		"reasons=operator_feedback,positive_outcome",
		"invalid records:",
		"line 4:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("curator effectiveness stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCuratorCommand_StatusMultilineSummary(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "document-tools")
	if err := skills.MarkAgentCreated(root, "document-tools"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}
	lastRun := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	summary := "auto: 1 marked stale; llm: consolidated 2 into 1\n" +
		"archived 2 skill(s):\n" +
		"  • pdf-extraction → document-tools\n" +
		"full report: gormes curator status"
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:      &lastRun,
		LastRunSummary: summary,
		RunCount:       1,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "status")
	if err != nil {
		t.Fatalf("curator status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"last summary:   auto: 1 marked stale; llm: consolidated 2 into 1",
		"                  archived 2 skill(s):",
		"                    • pdf-extraction → document-tools",
		"                  full report: gormes curator status",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("curator status stdout missing indented summary line %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "status", "--json")
	if err != nil {
		t.Fatalf("curator status --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		State struct {
			LastRunSummary string `json:"last_run_summary,omitempty"`
		} `json:"state"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator status --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.State.LastRunSummary != summary {
		t.Fatalf("json last_run_summary = %q, want raw summary %q", got.State.LastRunSummary, summary)
	}
}

// TestCuratorCommand_StatusJSONEmitsStructuredReport proves
// `gormes curator status --json` returns a parseable
// `{build, state: {paused, run_count, last_run_at, last_run_summary,
// last_report_path}, defaults: {interval_hours, stale_days, archive_days},
// skills: {total, rows: [...]}}` document so fleet dashboards can
// monitor curator across machines without scraping multi-section prose.
// The skill rows use the public usage shape so dashboards can sort by
// activity, identify stale skills, and feed alerts.
func TestCuratorCommand_StatusJSONEmitsStructuredReport(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "alpha")
	writeCuratorCommandSkill(t, root, "beta")
	for _, name := range []string{"alpha", "beta"} {
		if err := skills.MarkAgentCreated(root, name); err != nil {
			t.Fatalf("MarkAgentCreated(%s): %v", name, err)
		}
	}
	for i := 0; i < 5; i++ {
		if err := skills.BumpUse(root, "alpha"); err != nil {
			t.Fatalf("BumpUse alpha: %v", err)
		}
	}
	lastRun := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:      &lastRun,
		LastRunSummary: "curator run completed",
		LastReportPath: filepath.Join(root, "logs", "curator", "REPORT.md"),
		RunCount:       3,
		Paused:         false,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "status", "--json")
	if err != nil {
		t.Fatalf("curator status --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		State struct {
			Paused         bool       `json:"paused"`
			RunCount       int        `json:"run_count"`
			LastRunAt      *time.Time `json:"last_run_at,omitempty"`
			LastRunSummary string     `json:"last_run_summary,omitempty"`
			LastReportPath string     `json:"last_report_path,omitempty"`
		} `json:"state"`
		Defaults struct {
			IntervalHours int `json:"interval_hours"`
			StaleDays     int `json:"stale_days"`
			ArchiveDays   int `json:"archive_days"`
		} `json:"defaults"`
		Skills struct {
			Total int `json:"total"`
			Rows  []struct {
				Name     string `json:"name"`
				Activity int    `json:"activity"`
			} `json:"rows"`
		} `json:"skills"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator status --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.State.Paused {
		t.Errorf("paused must be false")
	}
	if got.State.RunCount != 3 {
		t.Errorf("run_count = %d, want 3", got.State.RunCount)
	}
	if got.State.LastRunSummary != "curator run completed" {
		t.Errorf("last_run_summary = %q, want %q", got.State.LastRunSummary, "curator run completed")
	}
	if got.Skills.Total != 2 {
		t.Errorf("skills.total = %d, want 2", got.Skills.Total)
	}
	var sawAlpha bool
	for _, row := range got.Skills.Rows {
		if row.Name == "alpha" {
			sawAlpha = true
			if row.Activity != 5 {
				t.Errorf("alpha activity = %d, want 5", row.Activity)
			}
		}
	}
	if !sawAlpha {
		t.Errorf("skills.rows must include alpha; got %+v", got.Skills.Rows)
	}
	if got.Defaults.IntervalHours <= 0 || got.Defaults.StaleDays <= 0 || got.Defaults.ArchiveDays <= 0 {
		t.Errorf("defaults must be populated; got %+v", got.Defaults)
	}
}

// TestCuratorCommand_BackupJSONEmitsStructuredOutcome proves
// `gormes curator backup --json` returns a parseable
// `{build, id, archive_path, manifest_path, reason}` document so
// fleet automation taking pre-deploy snapshots can record the
// snapshot id and archive path for later rollback. Without this,
// the only way to learn the id is to scrape the CLI output.
func TestCuratorCommand_BackupJSONEmitsStructuredOutcome(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "alpha")
	if err := skills.MarkAgentCreated(root, "alpha"); err != nil {
		t.Fatalf("MarkAgentCreated alpha: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "backup", "--reason", "pre-deploy", "--json")
	if err != nil {
		t.Fatalf("curator backup --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		ID           string `json:"id"`
		ArchivePath  string `json:"archive_path"`
		ManifestPath string `json:"manifest_path"`
		Reason       string `json:"reason"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator backup --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.ID == "" {
		t.Errorf("id must be populated")
	}
	if got.ArchivePath == "" {
		t.Errorf("archive_path must be populated")
	}
	if got.Reason != "pre-deploy" {
		t.Errorf("reason = %q, want %q", got.Reason, "pre-deploy")
	}
	// Verify on disk that the archive really landed.
	if _, statErr := os.Stat(got.ArchivePath); statErr != nil {
		t.Errorf("archive_path %s missing on disk: %v", got.ArchivePath, statErr)
	}
}

func TestCuratorCommand_RunDryRunPauseResumePinUnpin(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "agent-skill")
	writeCuratorCommandSkill(t, root, "bundled-skill")
	if err := skills.MarkAgentCreated(root, "agent-skill"); err != nil {
		t.Fatalf("MarkAgentCreated agent-skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".bundled_manifest"), []byte("bundled-skill\n"), 0o600); err != nil {
		t.Fatalf("write bundled manifest: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "run", "--dry-run", "--sync")
	if err != nil {
		t.Fatalf("curator run --dry-run --sync: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"curator: running DRY-RUN", "dry-run: no changes applied", "report:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("dry-run stdout missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".curator_backups")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created backups dir or stat failed: %v", err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pause")
	if err != nil || !strings.Contains(stdout, "curator: paused") {
		t.Fatalf("curator pause = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	state, err := skills.NewCurator(skills.CuratorConfig{Root: root}).LoadState()
	if err != nil {
		t.Fatalf("LoadState after pause: %v", err)
	}
	if !state.Paused {
		t.Fatalf("state.Paused = false after pause")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "resume")
	if err != nil || !strings.Contains(stdout, "curator: resumed") {
		t.Fatalf("curator resume = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	state, err = skills.NewCurator(skills.CuratorConfig{Root: root}).LoadState()
	if err != nil {
		t.Fatalf("LoadState after resume: %v", err)
	}
	if state.Paused {
		t.Fatalf("state.Paused = true after resume")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pin", "agent-skill")
	if err != nil || !strings.Contains(stdout, "curator: pinned 'agent-skill'") {
		t.Fatalf("curator pin = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	pinned, err := skills.IsPinned(root, "agent-skill")
	if err != nil {
		t.Fatalf("IsPinned: %v", err)
	}
	if !pinned {
		t.Fatalf("agent-skill pinned = false")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "unpin", "agent-skill")
	if err != nil || !strings.Contains(stdout, "curator: unpinned 'agent-skill'") {
		t.Fatalf("curator unpin = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	pinned, err = skills.IsPinned(root, "agent-skill")
	if err != nil {
		t.Fatalf("IsPinned after unpin: %v", err)
	}
	if pinned {
		t.Fatalf("agent-skill pinned = true after unpin")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pin", "bundled-skill")
	if err == nil {
		t.Fatalf("curator pin bundled-skill err = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "bundled or hub-installed") {
		t.Fatalf("curator pin bundled output missing refusal:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
}

func TestCuratorCommand_BackupRollbackRestore(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "keeper")
	if err := skills.MarkAgentCreated(root, "keeper"); err != nil {
		t.Fatalf("MarkAgentCreated keeper: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "backup", "--reason", "manual-test")
	if err != nil {
		t.Fatalf("curator backup: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: snapshot created") {
		t.Fatalf("curator backup stdout = %q, want snapshot created", stdout)
	}
	backupID := newestCuratorBackupID(t, root)
	if backupID == "" {
		t.Fatalf("missing curator backup id")
	}

	if err := os.RemoveAll(filepath.Join(root, "active", "keeper")); err != nil {
		t.Fatalf("remove keeper: %v", err)
	}
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "rollback", "--id", backupID, "--yes")
	if err != nil {
		t.Fatalf("curator rollback: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: rollback restored") {
		t.Fatalf("curator rollback stdout = %q, want restored evidence", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "keeper", "SKILL.md")); err != nil {
		t.Fatalf("keeper not restored: %v", err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "rollback", "--list")
	if err != nil {
		t.Fatalf("curator rollback --list: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, backupID) || !strings.Contains(stdout, "manual-test") {
		t.Fatalf("rollback --list stdout missing backup evidence:\n%s", stdout)
	}

	archivedDir := filepath.Join(root, "active", ".archive", "archived-skill")
	if err := os.MkdirAll(archivedDir, 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivedDir, "SKILL.md"), []byte("---\nname: archived-skill\ndescription: archived\n---\n# archived\n"), 0o600); err != nil {
		t.Fatalf("write archived skill: %v", err)
	}
	if err := skills.MarkAgentCreated(root, "archived-skill"); err != nil {
		t.Fatalf("MarkAgentCreated archived-skill: %v", err)
	}
	if err := skills.SetSkillState(root, "archived-skill", skills.SkillStateArchived); err != nil {
		t.Fatalf("SetSkillState archived-skill: %v", err)
	}
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "restore", "archived-skill")
	if err != nil {
		t.Fatalf("curator restore: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: restored 'archived-skill'") {
		t.Fatalf("restore stdout = %q, want restored archived-skill", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "archived-skill", "SKILL.md")); err != nil {
		t.Fatalf("archived-skill not restored: %v", err)
	}
}

func TestCuratorCommand_ArchiveListArchivedPrune(t *testing.T) {
	root := setupCuratorCommandHome(t)
	old := time.Now().UTC().AddDate(0, 0, -120)
	recent := time.Now().UTC().AddDate(0, 0, -2)

	for _, name := range []string{"manual", "pinned", "archive-now", "old-prune", "pinned-prune", "recent-prune"} {
		writeCuratorCommandSkill(t, root, name)
	}
	writeCuratorCommandUsage(t, root, map[string]skills.SkillUsageRecord{
		"pinned": {
			CreatedBy:    "agent",
			AgentCreated: true,
			Pinned:       true,
			State:        skills.SkillStateActive,
			CreatedAt:    old,
			LastUsedAt:   old,
		},
		"archive-now": {
			CreatedBy:    "agent",
			AgentCreated: true,
			State:        skills.SkillStateActive,
			CreatedAt:    old,
			LastUsedAt:   old,
		},
		"old-prune": {
			CreatedBy:    "agent",
			AgentCreated: true,
			State:        skills.SkillStateActive,
			CreatedAt:    old,
			LastUsedAt:   old,
		},
		"pinned-prune": {
			CreatedBy:    "agent",
			AgentCreated: true,
			Pinned:       true,
			State:        skills.SkillStateActive,
			CreatedAt:    old,
			LastUsedAt:   old,
		},
		"recent-prune": {
			CreatedBy:    "agent",
			AgentCreated: true,
			State:        skills.SkillStateActive,
			CreatedAt:    recent,
			LastUsedAt:   recent,
		},
	})

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "archive", "pinned")
	if err == nil {
		t.Fatalf("curator archive pinned err = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "pinned") || !strings.Contains(stdout+stderr+err.Error(), "unpin") {
		t.Fatalf("curator archive pinned output missing pinned refusal:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "archive", "manual")
	if err == nil {
		t.Fatalf("curator archive manual err = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "bundled or hub-installed") {
		t.Fatalf("curator archive manual output missing provenance refusal:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "archive", "archive-now")
	if err != nil {
		t.Fatalf("curator archive archive-now: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: archived") {
		t.Fatalf("curator archive stdout = %q, want archived evidence", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", ".archive", "archive-now", "SKILL.md")); err != nil {
		t.Fatalf("archive-now not archived: %v", err)
	}
	rec, err := skills.GetUsageRecord(root, "archive-now")
	if err != nil {
		t.Fatalf("GetUsageRecord archive-now: %v", err)
	}
	if rec.State != skills.SkillStateArchived || rec.ArchivedAt.IsZero() {
		t.Fatalf("archive-now record = %+v, want archived state with timestamp", rec)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "list-archived")
	if err != nil {
		t.Fatalf("curator list-archived: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "archive-now") {
		t.Fatalf("list-archived stdout missing archive-now:\n%s", stdout)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "prune", "--days", "0")
	if err == nil {
		t.Fatalf("curator prune --days 0 err = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "--days must be >= 1") {
		t.Fatalf("curator prune invalid days output missing validation:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "prune", "--days", "90", "--dry-run")
	if err != nil {
		t.Fatalf("curator prune dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "old-prune") || !strings.Contains(stdout, "dry run") {
		t.Fatalf("curator prune dry-run stdout missing preview:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "old-prune", "SKILL.md")); err != nil {
		t.Fatalf("old-prune mutated during dry-run: %v", err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "prune", "--days", "90", "--yes")
	if err != nil {
		t.Fatalf("curator prune --yes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: archived 1/1") {
		t.Fatalf("curator prune stdout = %q, want archived 1/1", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", ".archive", "old-prune", "SKILL.md")); err != nil {
		t.Fatalf("old-prune not archived by prune: %v", err)
	}
	for _, name := range []string{"pinned-prune", "recent-prune"} {
		if _, err := os.Stat(filepath.Join(root, "active", name, "SKILL.md")); err != nil {
			t.Fatalf("%s should remain active after prune: %v", name, err)
		}
	}
}

// TestCuratorCommand_RollbackJSONEmitsStructuredOutcome proves
// `gormes curator rollback --id X --yes --json` returns
// `{build, action, restored_backup_id, pre_rollback_backup_id}` so
// fleet automation can confirm rollback succeeded and capture the
// safety snapshot id (the snapshot taken just before rollback) for
// undo operations.
func TestCuratorCommand_RollbackJSONEmitsStructuredOutcome(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "keeper")
	if err := skills.MarkAgentCreated(root, "keeper"); err != nil {
		t.Fatalf("MarkAgentCreated keeper: %v", err)
	}
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "backup", "--reason", "rollback-test")
	if err != nil {
		t.Fatalf("curator backup: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	backupID := newestCuratorBackupID(t, root)
	if backupID == "" {
		t.Fatalf("missing curator backup id")
	}
	if err := os.RemoveAll(filepath.Join(root, "active", "keeper")); err != nil {
		t.Fatalf("remove keeper: %v", err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "rollback", "--id", backupID, "--yes", "--json")
	if err != nil {
		t.Fatalf("curator rollback --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action              string `json:"action"`
		RestoredBackupID    string `json:"restored_backup_id"`
		PreRollbackBackupID string `json:"pre_rollback_backup_id"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator rollback --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "rolled_back" {
		t.Errorf("action = %q, want %q", got.Action, "rolled_back")
	}
	if got.RestoredBackupID != backupID {
		t.Errorf("restored_backup_id = %q, want %q", got.RestoredBackupID, backupID)
	}
	if got.PreRollbackBackupID == "" {
		t.Errorf("pre_rollback_backup_id must be populated (safety snapshot)")
	}
	if _, err := os.Stat(filepath.Join(root, "active", "keeper", "SKILL.md")); err != nil {
		t.Errorf("keeper not restored: %v", err)
	}
}

// TestCuratorCommand_RollbackListJSONEmitsBackupArray proves that
// `--list --json` returns a parseable array of backups
// `{build, backups: [{id, reason, created_at}]}` so dashboards can
// list snapshots without scraping column-formatted output.
func TestCuratorCommand_RollbackListJSONEmitsBackupArray(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "keeper")
	if err := skills.MarkAgentCreated(root, "keeper"); err != nil {
		t.Fatalf("MarkAgentCreated keeper: %v", err)
	}
	stdout, _, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "backup", "--reason", "test-list")
	if err != nil {
		t.Fatalf("curator backup: %v\nstdout=%s", err, stdout)
	}
	backupID := newestCuratorBackupID(t, root)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "rollback", "--list", "--json")
	if err != nil {
		t.Fatalf("curator rollback --list --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Backups []struct {
			ID        string `json:"id"`
			Reason    string `json:"reason"`
			CreatedAt string `json:"created_at"`
		} `json:"backups"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator rollback --list --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	var sawSnapshot bool
	for _, b := range got.Backups {
		if b.ID == backupID {
			sawSnapshot = true
			if b.Reason != "test-list" {
				t.Errorf("backup reason = %q, want %q", b.Reason, "test-list")
			}
		}
	}
	if !sawSnapshot {
		t.Errorf("backups array missing %q; got %+v", backupID, got.Backups)
	}
}

// TestCuratorCommand_RunJSON proves `gormes curator run --json`
// returns a parseable
// `{build, dry_run, summary, auto_counts: {checked, marked_stale,
// archived, reactivated}, before_names, after_names, last_report_path,
// backup_id}` document so fleet automation triggering scheduled
// curator passes can audit per-machine activity without scraping
// the multi-line "auto: checked=N stale=M..." prose.
func TestCuratorCommand_RunJSON(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "alpha")
	if err := skills.MarkAgentCreated(root, "alpha"); err != nil {
		t.Fatalf("MarkAgentCreated alpha: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "run", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("curator run --dry-run --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		DryRun     bool `json:"dry_run"`
		AutoCounts struct {
			Checked     int `json:"checked"`
			MarkedStale int `json:"marked_stale"`
			Archived    int `json:"archived"`
			Reactivated int `json:"reactivated"`
		} `json:"auto_counts"`
		BeforeNames    []string `json:"before_names"`
		LastReportPath string   `json:"last_report_path,omitempty"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator run --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if !got.DryRun {
		t.Errorf("dry_run must be true in dry-run mode")
	}
}

// TestCuratorCommand_RestoreJSON proves
// `gormes curator restore <skill> --json` returns
// `{build, action: "restored", skill}` for fleet automation
// re-activating archived skills across machines.
func TestCuratorCommand_RestoreJSON(t *testing.T) {
	root := setupCuratorCommandHome(t)

	// Seed an archived skill.
	archivedDir := filepath.Join(root, "active", ".archive", "archived-alpha")
	if err := os.MkdirAll(archivedDir, 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivedDir, "SKILL.md"), []byte("---\nname: archived-alpha\ndescription: archived\n---\n# archived\n"), 0o600); err != nil {
		t.Fatalf("write archived skill: %v", err)
	}
	if err := skills.MarkAgentCreated(root, "archived-alpha"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}
	if err := skills.SetSkillState(root, "archived-alpha", skills.SkillStateArchived); err != nil {
		t.Fatalf("SetSkillState: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "restore", "archived-alpha", "--json")
	if err != nil {
		t.Fatalf("curator restore --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		Skill  string `json:"skill"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("curator restore --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "restored" {
		t.Errorf("action = %q, want %q", got.Action, "restored")
	}
	if got.Skill != "archived-alpha" {
		t.Errorf("skill = %q, want archived-alpha", got.Skill)
	}
	// Verify on disk.
	if _, err := os.Stat(filepath.Join(root, "active", "archived-alpha", "SKILL.md")); err != nil {
		t.Errorf("archived-alpha not actually restored: %v", err)
	}
}

// TestCuratorCommand_PinUnpinJSON proves `curator pin <skill> --json`
// and `curator unpin <skill> --json` return parseable
// `{build, action, skill, pinned}` documents so fleet automation
// reconciling pin lists across machines can confirm the state flip
// landed without scraping prose.
func TestCuratorCommand_PinUnpinJSON(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "alpha")
	if err := skills.MarkAgentCreated(root, "alpha"); err != nil {
		t.Fatalf("MarkAgentCreated alpha: %v", err)
	}

	// pin --json
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pin", "alpha", "--json")
	if err != nil {
		t.Fatalf("curator pin --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var pinned struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		Skill  string `json:"skill"`
		Pinned bool   `json:"pinned"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &pinned); jsonErr != nil {
		t.Fatalf("curator pin --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if pinned.Build.Version != Version {
		t.Errorf("pinned.build.version = %q, want %q", pinned.Build.Version, Version)
	}
	if pinned.Action != "pinned" {
		t.Errorf("pinned.action = %q, want %q", pinned.Action, "pinned")
	}
	if pinned.Skill != "alpha" {
		t.Errorf("pinned.skill = %q, want alpha", pinned.Skill)
	}
	if !pinned.Pinned {
		t.Errorf("pinned.pinned must be true after pin")
	}

	// unpin --json
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "unpin", "alpha", "--json")
	if err != nil {
		t.Fatalf("curator unpin --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var unpinned struct {
		Action string `json:"action"`
		Pinned bool   `json:"pinned"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &unpinned); jsonErr != nil {
		t.Fatalf("curator unpin --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if unpinned.Action != "unpinned" {
		t.Errorf("unpinned.action = %q, want %q", unpinned.Action, "unpinned")
	}
	if unpinned.Pinned {
		t.Errorf("unpinned.pinned must be false after unpin")
	}
}

// TestCuratorCommand_PauseResumeJSON proves `curator pause --json`
// and `curator resume --json` return parseable
// `{build, action, paused}` documents so fleet kill-switch
// automation pausing curator across machines can confirm the state
// flip landed without scraping prose.
func TestCuratorCommand_PauseResumeJSON(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "alpha")
	if err := skills.MarkAgentCreated(root, "alpha"); err != nil {
		t.Fatalf("MarkAgentCreated alpha: %v", err)
	}

	// pause --json
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pause", "--json")
	if err != nil {
		t.Fatalf("curator pause --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var paused struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		Paused bool   `json:"paused"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &paused); jsonErr != nil {
		t.Fatalf("curator pause --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if paused.Build.Version != Version {
		t.Errorf("paused.build.version = %q, want %q", paused.Build.Version, Version)
	}
	if paused.Action != "paused" {
		t.Errorf("paused.action = %q, want %q", paused.Action, "paused")
	}
	if !paused.Paused {
		t.Errorf("paused.paused must be true after pause")
	}

	// resume --json
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "resume", "--json")
	if err != nil {
		t.Fatalf("curator resume --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var resumed struct {
		Action string `json:"action"`
		Paused bool   `json:"paused"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &resumed); jsonErr != nil {
		t.Fatalf("curator resume --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if resumed.Action != "resumed" {
		t.Errorf("resumed.action = %q, want %q", resumed.Action, "resumed")
	}
	if resumed.Paused {
		t.Errorf("resumed.paused must be false after resume")
	}
}

func TestRootCommandIncludesCuratorCommand(t *testing.T) {
	root := newRootCommandWithRuntime(rootRuntime{})
	cmd, _, err := root.Find([]string{"curator", "status"})
	if err != nil {
		t.Fatalf("find curator status: %v", err)
	}
	if cmd == nil || cmd.Use != "status" {
		t.Fatalf("root command did not expose curator status: %#v", cmd)
	}
}

func setupCuratorCommandHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HERMES_HOME", filepath.Join(t.TempDir(), "hermes"))
	root := filepath.Join(home, "skills")
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatalf("mkdir skills active: %v", err)
	}
	return root
}

func writeCuratorCommandSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill %s: %v", name, err)
	}
	body := "---\nname: " + name + "\ndescription: test\n---\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write SKILL.md %s: %v", name, err)
	}
}

func writeCuratorCommandUsage(t *testing.T, root string, state map[string]skills.SkillUsageRecord) {
	t.Helper()
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal usage state: %v", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(root, ".usage.json"), raw, 0o600); err != nil {
		t.Fatalf("write usage state: %v", err)
	}
}

func newestCuratorBackupID(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".curator_backups"))
	if err != nil {
		t.Fatalf("read curator backups: %v", err)
	}
	var newest string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() > newest {
			newest = entry.Name()
		}
	}
	return newest
}
