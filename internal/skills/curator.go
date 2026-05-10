package skills

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultCuratorIntervalHours = 24 * 7
	DefaultCuratorMinIdleHours  = 2
	DefaultCuratorStaleDays     = 30
	DefaultCuratorArchiveDays   = 90

	CuratorEvidenceDisabled         = "curator_disabled"
	CuratorEvidencePaused           = "curator_paused"
	CuratorEvidenceFirstRunDeferred = "curator_first_run_deferred"
	CuratorEvidenceIntervalPending  = "curator_interval_pending"
	CuratorEvidenceReady            = "curator_ready"
)

type CuratorConfig struct {
	Root             string
	Disabled         bool
	IntervalHours    int
	MinIdleHours     float64
	StaleAfterDays   int
	ArchiveAfterDays int
	Now              func() time.Time
	Reviewer         CuratorReviewer
}

type CuratorReviewer func(context.Context, CuratorReviewInput) (CuratorReviewResult, error)

type Curator struct {
	cfg CuratorConfig
}

type CuratorState struct {
	LastRunAt              *time.Time `json:"last_run_at,omitempty"`
	LastRunDurationSeconds float64    `json:"last_run_duration_seconds,omitempty"`
	LastRunSummary         string     `json:"last_run_summary,omitempty"`
	LastRunSummaryShownAt  *time.Time `json:"last_run_summary_shown_at,omitempty"`
	LastReportPath         string     `json:"last_report_path,omitempty"`
	Paused                 bool       `json:"paused,omitempty"`
	RunCount               int        `json:"run_count,omitempty"`
}

type CuratorDecision struct {
	Eligible bool   `json:"eligible"`
	Code     string `json:"code"`
	Message  string `json:"message,omitempty"`
}

type CuratorTransitionCounts struct {
	Checked     int `json:"checked"`
	MarkedStale int `json:"marked_stale"`
	Archived    int `json:"archived"`
	Reactivated int `json:"reactivated"`
}

type CuratorRunOptions struct {
	DryRun bool
}

type CuratorReviewInput struct {
	DryRun         bool     `json:"dry_run"`
	CandidateNames []string `json:"candidate_names"`
	Prompt         string   `json:"prompt"`
}

type CuratorReviewResult struct {
	Summary   string            `json:"summary,omitempty"`
	ToolCalls []CuratorToolCall `json:"tool_calls,omitempty"`
}

type CuratorToolCall struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type CuratorRunReport struct {
	StartedAt      time.Time               `json:"started_at"`
	Duration       time.Duration           `json:"duration"`
	DryRun         bool                    `json:"dry_run,omitempty"`
	StateAdvanced  bool                    `json:"state_advanced,omitempty"`
	Summary        string                  `json:"summary,omitempty"`
	AutoCounts     CuratorTransitionCounts `json:"auto_counts"`
	BeforeNames    []string                `json:"before_names,omitempty"`
	AfterNames     []string                `json:"after_names,omitempty"`
	ToolCalls      []CuratorToolCall       `json:"tool_calls,omitempty"`
	BackupID       string                  `json:"backup_id,omitempty"`
	LastReportPath string                  `json:"last_report_path,omitempty"`
	Classification CuratorClassification   `json:"classification"`
}

type CuratorBackup struct {
	ID           string `json:"id"`
	ArchivePath  string `json:"archive_path"`
	ManifestPath string `json:"manifest_path"`
}

type CuratorRollback struct {
	RestoredBackupID    string `json:"restored_backup_id"`
	PreRollbackBackupID string `json:"pre_rollback_backup_id"`
}

type CuratorClassification struct {
	Consolidated map[string]CuratorConsolidation `json:"consolidated,omitempty"`
	Pruned       []string                        `json:"pruned,omitempty"`
}

type CuratorConsolidation struct {
	Into   string `json:"into"`
	Source string `json:"source"`
}

func NewCurator(cfg CuratorConfig) *Curator {
	if cfg.Root == "" {
		cfg.Root = DefaultRoot()
	}
	if cfg.IntervalHours <= 0 {
		cfg.IntervalHours = DefaultCuratorIntervalHours
	}
	if cfg.MinIdleHours <= 0 {
		cfg.MinIdleHours = DefaultCuratorMinIdleHours
	}
	if cfg.StaleAfterDays <= 0 {
		cfg.StaleAfterDays = DefaultCuratorStaleDays
	}
	if cfg.ArchiveAfterDays <= 0 {
		cfg.ArchiveAfterDays = DefaultCuratorArchiveDays
	}
	return &Curator{cfg: cfg}
}

func (c *Curator) now() time.Time {
	if c != nil && c.cfg.Now != nil {
		return c.cfg.Now().UTC()
	}
	return time.Now().UTC()
}

func (c *Curator) root() string {
	if c == nil || c.cfg.Root == "" {
		return DefaultRoot()
	}
	return c.cfg.Root
}

func (c *Curator) statePath() string {
	return filepath.Join(c.root(), ".curator_state")
}

func (c *Curator) LoadState() (CuratorState, error) {
	raw, err := os.ReadFile(c.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return CuratorState{}, nil
	}
	if err != nil {
		return CuratorState{}, err
	}
	if len(raw) == 0 {
		return CuratorState{}, nil
	}
	var state CuratorState
	if err := json.Unmarshal(raw, &state); err != nil {
		return CuratorState{}, nil
	}
	return state, nil
}

func (c *Curator) SaveState(state CuratorState) error {
	path := c.statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".curator_state.tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (c *Curator) ShouldRunNow(ctx context.Context) (CuratorDecision, error) {
	select {
	case <-ctx.Done():
		return CuratorDecision{}, ctx.Err()
	default:
	}
	if c.cfg.Disabled {
		return CuratorDecision{Code: CuratorEvidenceDisabled, Message: "curator disabled"}, nil
	}
	state, err := c.LoadState()
	if err != nil {
		return CuratorDecision{}, err
	}
	if state.Paused {
		return CuratorDecision{Code: CuratorEvidencePaused, Message: "curator paused"}, nil
	}
	now := c.now()
	if state.LastRunAt == nil {
		state.LastRunAt = &now
		state.LastRunSummary = "deferred first run - curator seeded, will run after one interval"
		if err := c.SaveState(state); err != nil {
			return CuratorDecision{}, err
		}
		return CuratorDecision{Code: CuratorEvidenceFirstRunDeferred, Message: state.LastRunSummary}, nil
	}
	if now.Sub((*state.LastRunAt).UTC()) < time.Duration(c.cfg.IntervalHours)*time.Hour {
		return CuratorDecision{Code: CuratorEvidenceIntervalPending, Message: "curator interval has not elapsed"}, nil
	}
	return CuratorDecision{Eligible: true, Code: CuratorEvidenceReady, Message: "curator interval elapsed"}, nil
}

func (c *Curator) ApplyAutomaticTransitions(ctx context.Context) (CuratorTransitionCounts, error) {
	rows, err := ListAgentCreatedSkillUsage(c.root())
	if err != nil {
		return CuratorTransitionCounts{}, err
	}
	now := c.now()
	staleCutoff := now.AddDate(0, 0, -c.cfg.StaleAfterDays)
	archiveCutoff := now.AddDate(0, 0, -c.cfg.ArchiveAfterDays)
	var counts CuratorTransitionCounts
	for _, row := range rows {
		select {
		case <-ctx.Done():
			return counts, ctx.Err()
		default:
		}
		counts.Checked++
		if row.Record.Pinned {
			continue
		}
		anchor := row.LastActivityAt
		if anchor.IsZero() {
			anchor = now
		}
		state := row.Record.State
		if state == "" {
			state = SkillStateActive
		}
		switch {
		case !anchor.After(archiveCutoff) && state != SkillStateArchived:
			if err := c.archiveSkill(row.Name, row.SkillDir, now); err != nil {
				return counts, err
			}
			counts.Archived++
		case !anchor.After(staleCutoff) && state == SkillStateActive:
			if err := SetSkillState(c.root(), row.Name, SkillStateStale); err != nil {
				return counts, err
			}
			counts.MarkedStale++
		case anchor.After(staleCutoff) && state == SkillStateStale:
			if err := SetSkillState(c.root(), row.Name, SkillStateActive); err != nil {
				return counts, err
			}
			counts.Reactivated++
		}
	}
	return counts, nil
}

func (c *Curator) archiveSkill(name, skillDir string, now time.Time) error {
	archiveRoot := filepath.Join(c.root(), "active", ".archive")
	if err := os.MkdirAll(archiveRoot, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(archiveRoot, name)
	if _, err := os.Stat(dest); err == nil {
		dest = filepath.Join(archiveRoot, fmt.Sprintf("%s-%s", name, timestampID(now)))
	}
	if err := os.Rename(skillDir, dest); err != nil {
		return err
	}
	return updateUsageRecord(c.root(), name, func(rec *SkillUsageRecord) {
		rec.State = SkillStateArchived
		rec.ArchivedAt = now.UTC()
	})
}

func (c *Curator) Run(ctx context.Context, opts CuratorRunOptions) (CuratorRunReport, error) {
	started := c.now()
	beforeRows, err := ListAgentCreatedSkillUsage(c.root())
	if err != nil {
		return CuratorRunReport{}, err
	}
	beforeNames := usageNames(beforeRows)
	report := CuratorRunReport{
		StartedAt:   started,
		DryRun:      opts.DryRun,
		BeforeNames: beforeNames,
	}
	if !opts.DryRun {
		backup, err := CreateCuratorBackup(c.root(), started, "curator run", nil)
		if err != nil {
			return CuratorRunReport{}, err
		}
		report.BackupID = backup.ID
		counts, err := c.ApplyAutomaticTransitions(ctx)
		if err != nil {
			return CuratorRunReport{}, err
		}
		report.AutoCounts = counts
	}

	candidates, err := ListAgentCreatedSkillUsage(c.root())
	if err != nil {
		return CuratorRunReport{}, err
	}
	candidateNames := usageNames(candidates)
	review, err := c.review(ctx, CuratorReviewInput{
		DryRun:         opts.DryRun,
		CandidateNames: candidateNames,
		Prompt:         curatorPrompt(opts.DryRun),
	})
	if err != nil {
		return CuratorRunReport{}, err
	}
	report.Summary = review.Summary
	report.ToolCalls = review.ToolCalls
	if report.Summary == "" {
		if len(candidateNames) == 0 {
			report.Summary = "curator skipped: no agent-created skill candidates"
		} else if opts.DryRun {
			report.Summary = "curator dry-run completed"
		} else {
			report.Summary = "curator run completed"
		}
	}

	afterRows, err := ListAgentCreatedSkillUsage(c.root())
	if err != nil {
		return CuratorRunReport{}, err
	}
	report.AfterNames = usageNames(afterRows)
	report.Classification = ClassifyRemovedSkills(diffNames(beforeNames, report.AfterNames), report.AfterNames, report.ToolCalls)
	report.Summary = appendCuratorRenameSummary(report.Summary, report.Classification)
	finished := c.now()
	report.Duration = finished.Sub(started)
	if report.Duration <= 0 {
		report.Duration = time.Millisecond
	}
	reportPath, err := writeCuratorReport(c.root(), report, started)
	if err != nil {
		return CuratorRunReport{}, err
	}
	report.LastReportPath = reportPath

	if opts.DryRun {
		state, err := c.LoadState()
		if err != nil {
			return CuratorRunReport{}, err
		}
		state.LastRunSummary = "dry-run: " + report.Summary
		state.LastReportPath = report.LastReportPath
		if err := c.SaveState(state); err != nil {
			return CuratorRunReport{}, err
		}
		return report, nil
	}

	state, err := c.LoadState()
	if err != nil {
		return CuratorRunReport{}, err
	}
	state.LastRunAt = &started
	state.LastRunDurationSeconds = report.Duration.Seconds()
	state.LastRunSummary = report.Summary
	state.LastReportPath = report.LastReportPath
	state.RunCount++
	if err := c.SaveState(state); err != nil {
		return CuratorRunReport{}, err
	}
	report.StateAdvanced = true
	return report, nil
}

func (c *Curator) review(ctx context.Context, input CuratorReviewInput) (CuratorReviewResult, error) {
	if c.cfg.Reviewer != nil {
		return c.cfg.Reviewer(ctx, input)
	}
	if len(input.CandidateNames) == 0 {
		return CuratorReviewResult{Summary: "curator skipped: no agent-created skill candidates"}, nil
	}
	return CuratorReviewResult{Summary: "curator review unavailable in this runtime"}, nil
}

func curatorPrompt(dryRun bool) string {
	prompt := "You are running as Gormes' Hermes-compatible background skill CURATOR."
	if dryRun {
		return "DRY-RUN - REPORT ONLY. DO NOT MUTATE THE SKILL LIBRARY.\n\n" + prompt
	}
	return prompt
}

func usageNames(rows []AgentCreatedSkillUsage) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	sort.Strings(out)
	return out
}

func diffNames(before, after []string) []string {
	afterSet := map[string]bool{}
	for _, name := range after {
		afterSet[name] = true
	}
	var out []string
	for _, name := range before {
		if !afterSet[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func ClassifyRemovedSkills(removed, afterNames []string, calls []CuratorToolCall) CuratorClassification {
	after := map[string]bool{}
	for _, name := range afterNames {
		after[name] = true
	}
	result := CuratorClassification{Consolidated: map[string]CuratorConsolidation{}}
	for _, name := range removed {
		if name == "" {
			continue
		}
		if cons, ok := declaredConsolidation(name, after, calls); ok {
			result.Consolidated[name] = cons
			continue
		}
		if cons, ok := heuristicConsolidation(name, after, calls); ok {
			result.Consolidated[name] = cons
			continue
		}
		result.Pruned = append(result.Pruned, name)
	}
	sort.Strings(result.Pruned)
	if len(result.Consolidated) == 0 {
		result.Consolidated = nil
	}
	return result
}

func appendCuratorRenameSummary(summary string, classification CuratorClassification) string {
	renameSummary := buildCuratorRenameSummary(classification)
	if renameSummary == "" {
		return summary
	}
	summary = strings.TrimRight(summary, "\n")
	if summary == "" {
		return renameSummary
	}
	return summary + "\n" + renameSummary
}

func buildCuratorRenameSummary(classification CuratorClassification) string {
	total := len(classification.Consolidated) + len(classification.Pruned)
	if total == 0 {
		return ""
	}
	const showLimit = 10
	lines := []string{fmt.Sprintf("archived %d skill(s):", total)}
	shown := 0

	consolidated := make([]string, 0, len(classification.Consolidated))
	for name := range classification.Consolidated {
		consolidated = append(consolidated, name)
	}
	sort.Strings(consolidated)
	for _, name := range consolidated {
		if shown >= showLimit {
			break
		}
		lines = append(lines, fmt.Sprintf("  • %s → %s", name, classification.Consolidated[name].Into))
		shown++
	}
	for _, name := range classification.Pruned {
		if shown >= showLimit {
			break
		}
		lines = append(lines, fmt.Sprintf("  • %s — pruned (stale)", name))
		shown++
	}
	if total > showLimit {
		lines = append(lines, fmt.Sprintf("  … and %d more", total-showLimit))
	}
	lines = append(lines, "full report: gormes curator status")
	return strings.Join(lines, "\n")
}

func declaredConsolidation(name string, after map[string]bool, calls []CuratorToolCall) (CuratorConsolidation, bool) {
	for _, call := range calls {
		if call.Name != "skill_manage" {
			continue
		}
		if call.Arguments["action"] != "delete" || call.Arguments["name"] != name {
			continue
		}
		into := strings.TrimSpace(call.Arguments["absorbed_into"])
		if into != "" && after[into] {
			return CuratorConsolidation{Into: into, Source: "absorbed_into"}, true
		}
		return CuratorConsolidation{}, false
	}
	return CuratorConsolidation{}, false
}

func heuristicConsolidation(name string, after map[string]bool, calls []CuratorToolCall) (CuratorConsolidation, bool) {
	needles := []string{name, strings.ReplaceAll(name, "-", "_"), strings.ReplaceAll(name, "_", "-")}
	for _, call := range calls {
		if call.Name != "skill_manage" {
			continue
		}
		target := strings.TrimSpace(call.Arguments["name"])
		if target == "" || target == name || !after[target] {
			continue
		}
		for _, key := range []string{"file_path", "file_content", "content", "new_string"} {
			hay := call.Arguments[key]
			for _, needle := range needles {
				if needle != "" && strings.Contains(hay, needle) {
					return CuratorConsolidation{Into: target, Source: "heuristic"}, true
				}
			}
		}
	}
	return CuratorConsolidation{}, false
}

func writeCuratorReport(root string, report CuratorRunReport, now time.Time) (string, error) {
	dir := filepath.Join(root, "logs", "curator", timestampID(now))
	for i := 1; ; i++ {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			break
		}
		dir = filepath.Join(root, "logs", "curator", fmt.Sprintf("%s-%02d", timestampID(now), i))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "run.json"), append(raw, '\n'), 0o600); err != nil {
		return "", err
	}
	md := fmt.Sprintf("# Curator Run\n\nsummary: %s\n\ndry_run: %t\ntool_calls: %d\nchecked: %d\nmarked_stale: %d\narchived: %d\nreactivated: %d\n",
		report.Summary,
		report.DryRun,
		len(report.ToolCalls),
		report.AutoCounts.Checked,
		report.AutoCounts.MarkedStale,
		report.AutoCounts.Archived,
		report.AutoCounts.Reactivated,
	)
	reportPath := filepath.Join(dir, "REPORT.md")
	return reportPath, os.WriteFile(reportPath, []byte(md), 0o600)
}

func CreateCuratorBackup(root string, now time.Time, reason string, cronSkillRefs map[string][]string) (CuratorBackup, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	id := timestampID(now)
	dir := filepath.Join(root, ".curator_backups", id)
	for i := 1; ; i++ {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			break
		}
		id = fmt.Sprintf("%s-%02d", timestampID(now), i)
		dir = filepath.Join(root, ".curator_backups", id)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return CuratorBackup{}, err
	}
	archivePath := filepath.Join(dir, "skills.tar.gz")
	if err := writeCuratorTar(root, archivePath); err != nil {
		return CuratorBackup{}, err
	}
	if cronSkillRefs != nil {
		raw, err := json.MarshalIndent(cronSkillRefs, "", "  ")
		if err != nil {
			return CuratorBackup{}, err
		}
		if err := os.WriteFile(filepath.Join(dir, "cron-skill-refs.json"), append(raw, '\n'), 0o600); err != nil {
			return CuratorBackup{}, err
		}
	}
	manifest := map[string]any{
		"id":         id,
		"reason":     reason,
		"created_at": now.UTC().Format(time.RFC3339),
		"archive":    filepath.Base(archivePath),
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return CuratorBackup{}, err
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, append(raw, '\n'), 0o600); err != nil {
		return CuratorBackup{}, err
	}
	return CuratorBackup{ID: id, ArchivePath: archivePath, ManifestPath: manifestPath}, nil
}

func RollbackCuratorBackup(root, id string, now time.Time) (CuratorRollback, error) {
	if id == "" || filepath.Base(id) != id || strings.Contains(id, "..") {
		return CuratorRollback{}, fmt.Errorf("unsafe curator backup id %q", id)
	}
	archivePath := filepath.Join(root, ".curator_backups", id, "skills.tar.gz")
	pre, err := CreateCuratorBackup(root, now, "pre-rollback "+id, nil)
	if err != nil {
		return CuratorRollback{}, err
	}
	tmp, err := os.MkdirTemp(filepath.Join(root, ".curator_backups"), ".restore-"+id+".")
	if err != nil {
		return CuratorRollback{}, err
	}
	defer os.RemoveAll(tmp)
	if err := extractCuratorTar(archivePath, tmp); err != nil {
		return CuratorRollback{}, err
	}
	if err := replacePath(filepath.Join(root, "active"), filepath.Join(tmp, "active")); err != nil {
		return CuratorRollback{}, err
	}
	if _, err := os.Stat(filepath.Join(tmp, ".usage.json")); err == nil {
		if err := replaceFile(filepath.Join(root, ".usage.json"), filepath.Join(tmp, ".usage.json")); err != nil {
			return CuratorRollback{}, err
		}
	}
	return CuratorRollback{RestoredBackupID: id, PreRollbackBackupID: pre.ID}, nil
}

func writeCuratorTar(root, archivePath string) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for _, path := range []string{filepath.Join(root, "active"), filepath.Join(root, ".usage.json")} {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := addTarPath(tw, root, path); err != nil {
			return err
		}
	}
	return nil
}

func addTarPath(tw *tar.Writer, root, path string) error {
	return filepath.WalkDir(path, func(current string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(current)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func extractCuratorTar(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(header.Name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe curator backup path %q", header.Name)
		}
		target := filepath.Join(dest, clean)
		if !pathWithinDir(dest, target) {
			return fmt.Errorf("unsafe curator backup path %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsafe curator backup entry type %d for %q", header.Typeflag, header.Name)
		}
	}
	return nil
}

func replacePath(target, replacement string) error {
	if _, err := os.Stat(replacement); errors.Is(err, os.ErrNotExist) {
		if err := os.RemoveAll(target); err != nil {
			return err
		}
		return os.MkdirAll(target, 0o755)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(replacement, target)
}

func replaceFile(target, replacement string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(replacement, target)
}

func pathWithinDir(root, target string) bool {
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
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func timestampID(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format("2006-01-02T15-04-05Z")
}
