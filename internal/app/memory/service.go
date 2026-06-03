package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	corememory "github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type Options struct {
	BuildProvenance func() BuildProvenance
	OpenDB          func(path string) (*sql.DB, error)
}

type StatusReportJSON struct {
	Build       BuildProvenance      `json:"build"`
	Inventory   corememory.Inventory `json:"inventory"`
	Extractor   ExtractorJSON        `json:"extractor"`
	GonchoQueue goncho.QueueStatus   `json:"goncho_queue"`
}

type ExtractorJSON struct {
	WorkerHealth       string                              `json:"worker_health"`
	QueueDepth         int                                 `json:"queue_depth"`
	DeadLetterCount    int                                 `json:"dead_letter_count"`
	SkippedSyncCount   int                                 `json:"skipped_sync_count"`
	ErrorSummary       []corememory.DeadLetterErrorSummary `json:"error_summary"`
	RecentDeadLetters  []corememory.DeadLetterSummary      `json:"recent_dead_letters"`
	RecentSkippedSyncs []corememory.SkippedSyncSummary     `json:"recent_skipped_syncs"`
}

func NewStatusCommand(opts Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show extractor queue depth and dead letters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunStatus(cmd.Context(), cmd.OutOrStdout(), asJSON, opts)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, inventory, extractor, goncho_queue}` JSON document (suitable for SRE alerting on memory backlog)")
	return cmd
}

func RunStatus(ctx context.Context, out io.Writer, asJSON bool, opts Options) error {
	path := config.MemoryDBPath()
	inventory, err := ReadInventory(ctx, nil)
	if err != nil {
		return err
	}
	zero := func() error {
		if asJSON {
			return EmitStatusJSON(out, corememory.ExtractorStatus{}, goncho.QueueStatus{}, inventory, build(opts))
		}
		_, err := fmt.Fprint(out, FormatStatus(corememory.ExtractorStatus{}, goncho.QueueStatus{}, inventory))
		return err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero()
		}
		return err
	}

	db, err := openDB(path, opts)
	if err != nil {
		return fmt.Errorf("open memory db: %w", err)
	}
	defer db.Close()
	inventory, err = ReadInventory(ctx, db)
	if err != nil {
		return err
	}

	status, err := corememory.ReadExtractorStatus(context.Background(), db, 0)
	if err != nil {
		if strings.Contains(err.Error(), "no such table: turns") {
			return zero()
		}
		return err
	}
	cfg, err := config.Load(nil)
	if err != nil {
		return err
	}
	gonchoCfg := cfg.Goncho.RuntimeConfig()
	queueStatus, err := goncho.ReadQueueStatus(context.Background(), db, goncho.QueueStatusConfig{
		DreamEnabled:     gonchoCfg.DreamEnabled,
		WorkspaceID:      gonchoCfg.WorkspaceID,
		ObserverPeerID:   gonchoCfg.ObserverPeerID,
		DreamIdleTimeout: gonchoCfg.DreamIdleTimeout,
	})
	if err != nil {
		return err
	}
	if asJSON {
		return EmitStatusJSON(out, status, queueStatus, inventory, build(opts))
	}
	_, err = fmt.Fprint(out, FormatStatus(status, queueStatus, inventory))
	return err
}

func ReadInventory(ctx context.Context, db *sql.DB) (corememory.Inventory, error) {
	cwd, _ := os.Getwd()
	return corememory.ReadInventory(ctx, corememory.InventoryOptions{
		ProfileRoot:         config.GormesHome(),
		MemoryDBPath:        config.MemoryDBPath(),
		CWD:                 cwd,
		ContextMemoryDirEnv: os.Getenv("GORMES_CONTEXT_MEMORY_DIR"),
		DB:                  db,
	})
}

func EmitStatusJSON(out io.Writer, extractor corememory.ExtractorStatus, queue goncho.QueueStatus, inventory corememory.Inventory, build BuildProvenance) error {
	if extractor.ErrorSummary == nil {
		extractor.ErrorSummary = []corememory.DeadLetterErrorSummary{}
	}
	if extractor.RecentDeadLetters == nil {
		extractor.RecentDeadLetters = []corememory.DeadLetterSummary{}
	}
	if extractor.RecentSkippedSyncs == nil {
		extractor.RecentSkippedSyncs = []corememory.SkippedSyncSummary{}
	}
	if queue.WorkUnits == nil {
		queue.WorkUnits = map[string]goncho.QueueWorkUnitStatus{}
	}
	body, err := json.MarshalIndent(StatusReportJSON{
		Build:     build,
		Inventory: inventory,
		Extractor: ExtractorJSON{
			WorkerHealth:       extractor.WorkerHealth,
			QueueDepth:         extractor.QueueDepth,
			DeadLetterCount:    extractor.DeadLetterCount,
			SkippedSyncCount:   extractor.SkippedSyncCount,
			ErrorSummary:       extractor.ErrorSummary,
			RecentDeadLetters:  extractor.RecentDeadLetters,
			RecentSkippedSyncs: extractor.RecentSkippedSyncs,
		},
		GonchoQueue: queue,
	}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(body))
	return err
}

func FormatStatus(status corememory.ExtractorStatus, queueStatus goncho.QueueStatus, inventory corememory.Inventory) string {
	return FormatExtractorStatus(status) + FormatInventory(inventory) + FormatGonchoQueueStatus(queueStatus)
}

func FormatInventory(inventory corememory.Inventory) string {
	var b strings.Builder
	b.WriteString("Memory inventory\n")
	b.WriteString(fmt.Sprintf("goncho.database: %s\n", inventory.Goncho.Database.State))
	b.WriteString(fmt.Sprintf("goncho.active_items: %d\n", inventory.Goncho.ActiveItems))
	b.WriteString(fmt.Sprintf("goncho.turns_total: %d\n", inventory.Goncho.TurnsTotal))
	b.WriteString(fmt.Sprintf("goncho.eval_artifacts: %d\n", inventory.Goncho.EvalArtifacts))
	b.WriteString(fmt.Sprintf("durable_markdown_dir: %s files=%d\n", inventory.DurableMarkdown.Directory.State, inventory.DurableMarkdown.Directory.Files))
	b.WriteString(fmt.Sprintf("durable_markdown_user: %s size_bytes=%d\n", inventory.DurableMarkdown.User.State, inventory.DurableMarkdown.User.SizeBytes))
	b.WriteString(fmt.Sprintf("durable_markdown_memory: %s size_bytes=%d\n", inventory.DurableMarkdown.Memory.State, inventory.DurableMarkdown.Memory.SizeBytes))
	b.WriteString(fmt.Sprintf("legacy_hermes_dir: %s files=%d\n", inventory.LegacyHermes.Directory.State, inventory.LegacyHermes.Directory.Files))
	b.WriteString(fmt.Sprintf("legacy_hermes_user: %s size_bytes=%d\n", inventory.LegacyHermes.User.State, inventory.LegacyHermes.User.SizeBytes))
	b.WriteString(fmt.Sprintf("legacy_hermes_memory: %s size_bytes=%d\n", inventory.LegacyHermes.Memory.State, inventory.LegacyHermes.Memory.SizeBytes))
	b.WriteString(fmt.Sprintf("selected_prompt_memory_dir: %s\n", inventory.SelectedPromptMemoryDir))
	b.WriteString(fmt.Sprintf("legacy_import_needed: %t\n", inventory.LegacyImportNeeded))
	b.WriteString(fmt.Sprintf("session_transcripts: files=%d\n", inventory.SessionTranscripts.Files))
	if len(inventory.ContextFiles) == 0 {
		b.WriteString("context_files: none\n")
		return b.String()
	}
	b.WriteString("context_files:\n")
	for _, item := range inventory.ContextFiles {
		b.WriteString(fmt.Sprintf("- %s: %s size_bytes=%d\n", item.RelativePath, item.State, item.SizeBytes))
	}
	return b.String()
}

func FormatExtractorStatus(status corememory.ExtractorStatus) string {
	var b strings.Builder
	b.WriteString("Extractor status\n")
	b.WriteString(fmt.Sprintf("worker_health: %s\n", status.WorkerHealth))
	b.WriteString(fmt.Sprintf("queue_depth: %d\n", status.QueueDepth))
	b.WriteString(fmt.Sprintf("dead_letters: %d\n", status.DeadLetterCount))
	b.WriteString(FormatDeadLetterSummary(status.ErrorSummary))
	if len(status.RecentDeadLetters) == 0 {
		b.WriteString("recent_dead_letters: none\n")
		return b.String()
	}
	b.WriteString("recent_dead_letters:\n")
	for _, dl := range status.RecentDeadLetters {
		b.WriteString(fmt.Sprintf("- turn_id=%d session_id=%s chat_id=%s attempts=%d error=%q\n", dl.ID, dl.SessionID, dl.ChatID, dl.Attempts, dl.Error))
	}
	return b.String()
}

func FormatGonchoQueueStatus(status goncho.QueueStatus) string {
	var b strings.Builder
	b.WriteString("Goncho queue status (observability/debugging only; not synchronization; do not wait for empty queue)\n")
	for _, taskType := range goncho.QueueTaskTypes {
		counts := status.WorkUnits[taskType]
		b.WriteString(fmt.Sprintf("%s: total=%d pending=%d in_progress=%d completed=%d\n", taskType, counts.TotalWorkUnits, counts.PendingWorkUnits, counts.InProgressWorkUnits, counts.CompletedWorkUnits))
	}
	b.WriteString(FormatDreamQueueEvidence(status.Dream))
	if status.Degraded {
		if status.WorkUnits["dream"].TotalWorkUnits > 0 {
			b.WriteString("goncho_queue: degraded (dream work intent tracked locally; representation/summary workers unavailable)\n")
		} else {
			b.WriteString("goncho_queue: unavailable (zero tracked work units)\n")
		}
	}
	return b.String()
}

func FormatDreamQueueEvidence(status goncho.DreamQueueStatus) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("dream_status: %s\n", status.Status))
	b.WriteString(fmt.Sprintf("dream_scheduler_table: %s\n", availableWord(status.TablePresent)))
	if len(status.Evidence) == 0 {
		b.WriteString("dream_evidence: none\n")
		return b.String()
	}
	b.WriteString("dream_evidence:\n")
	for _, item := range status.Evidence {
		b.WriteString(fmt.Sprintf("- %s: %s", item.Code, item.Reason))
		if item.ObservedPeerID != "" {
			b.WriteString(fmt.Sprintf(" observed=%s", item.ObservedPeerID))
		}
		if item.CooldownUntil > 0 {
			b.WriteString(fmt.Sprintf(" cooldown_until=%d", item.CooldownUntil))
		}
		if item.IdleUntil > 0 {
			b.WriteString(fmt.Sprintf(" idle_until=%d", item.IdleUntil))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func FormatDeadLetterSummary(items []corememory.DeadLetterErrorSummary) string {
	if len(items) == 0 {
		return "dead_letter_summary: none\n"
	}
	var b strings.Builder
	b.WriteString("dead_letter_summary:\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf("- error=%q count=%d\n", item.Error, item.Count))
	}
	return b.String()
}

func openDB(path string, opts Options) (*sql.DB, error) {
	if opts.OpenDB != nil {
		return opts.OpenDB(path)
	}
	return sql.Open("sqlite3", path)
}

func build(opts Options) BuildProvenance {
	if opts.BuildProvenance != nil {
		return opts.BuildProvenance()
	}
	return BuildProvenance{Version: "unknown", GitCommit: "unknown"}
}

func availableWord(ok bool) string {
	if ok {
		return "present"
	}
	return "missing"
}
