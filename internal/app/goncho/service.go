package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"

	"github.com/TrebuchetDynamics/goncho/service"
	internalgoncho "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/goncho"
	memoryapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho"
)

// GonchoDoctorReport is the structured diagnostic report for goncho doctor.
type GonchoDoctorReport struct {
	Build                  BuildProvenance                   `json:"build"`
	Service                string                            `json:"service"`
	Status                 string                            `json:"status"`
	ExitCode               int                               `json:"exit_code"`
	Config                 GonchoDoctorConfig                `json:"config"`
	Schema                 memory.SchemaStatus               `json:"schema"`
	MemoryContract         memory.GonchoMemoryV1Status       `json:"memory_contract"`
	LocalMarkdownMemory    goncho.LocalMarkdownMemoryStatus  `json:"local_markdown_memory"`
	SessionCatalog         SessionCatalogStatus              `json:"session_catalog"`
	ToolRegistration       ToolRegistrationStatus            `json:"tool_registration"`
	ContextDryRun          ContextDryRunStatus               `json:"context_dry_run"`
	QueueStatus            DoctorQueueStatus                 `json:"queue_status"`
	ConclusionAvailability ConclusionAvailability            `json:"conclusion_availability"`
	SummaryAvailability    SummaryAvailability               `json:"summary_availability"`
	ProviderReadiness      ProviderReadiness                 `json:"provider_readiness"`
	DegradedModes          []DegradedMode                    `json:"degraded_modes"`
}

// BuildProvenance carries version and git commit metadata.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// GonchoDoctorConfig carries the effective config snapshot for the report.
type GonchoDoctorConfig struct {
	ConfigPath     string `json:"config_path"`
	ConfigExists   bool   `json:"config_exists"`
	MemoryDBPath   string `json:"memory_db_path"`
	SessionDBPath  string `json:"session_db_path"`
	Workspace      string `json:"workspace"`
	ObserverPeer   string `json:"observer_peer"`
	RecentMessages int    `json:"recent_messages"`
	DreamEnabled   bool   `json:"dream_enabled"`
	DreamIdleMins  int    `json:"dream_idle_timeout_minutes"`
	HermesModel    string `json:"hermes_model"`
}

// SessionCatalogStatus describes the session catalog state.
type SessionCatalogStatus struct {
	Status        string `json:"status"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	SessionCount  int    `json:"session_count"`
	MetadataCount int    `json:"metadata_count"`
	Message       string `json:"message"`
}

// ToolRegistrationStatus describes tool registration health.
type ToolRegistrationStatus struct {
	Status     string   `json:"status"`
	Summary    string   `json:"summary"`
	Registered []string `json:"registered"`
	Invalid    []string `json:"invalid,omitempty"`
}

// ContextDryRunStatus describes the result of a context dry-run.
type ContextDryRunStatus struct {
	Status         string                              `json:"status"`
	Peer           string                              `json:"peer"`
	SessionKey     string                              `json:"session_key"`
	Representation string                              `json:"representation"`
	ScopeEvidence  *goncho.CrossChatRecallEvidence     `json:"scope_evidence,omitempty"`
	SearchResults  []goncho.SearchHit                  `json:"search_results,omitempty"`
	Unavailable    []goncho.ContextUnavailableEvidence `json:"unavailable"`
}

// ExtractorQueueSnapshot describes the extractor queue state.
type ExtractorQueueSnapshot struct {
	WorkerHealth     string `json:"worker_health"`
	QueueDepth       int    `json:"queue_depth"`
	DeadLetterCount  int    `json:"dead_letter_count"`
	SkippedSyncCount int    `json:"skipped_sync_count"`
}

// DoctorQueueStatus describes the queue status section.
type DoctorQueueStatus struct {
	Status            string                                `json:"status"`
	ObservabilityOnly bool                                  `json:"observability_only"`
	Extractor         ExtractorQueueSnapshot                 `json:"extractor"`
	WorkUnits         map[string]goncho.QueueWorkUnitStatus `json:"work_units"`
	Dream             goncho.DreamQueueStatus               `json:"dream"`
	Message           string                                `json:"message"`
}

// ConclusionAvailability describes conclusion availability.
type ConclusionAvailability struct {
	Status string           `json:"status"`
	Total  int              `json:"total"`
	Pairs  []ConclusionPair `json:"pairs"`
}

// ConclusionPair is a pair of peer IDs with conclusion count.
type ConclusionPair struct {
	ObserverPeerID string `json:"observer_peer_id"`
	PeerID         string `json:"peer_id"`
	Count          int    `json:"count"`
}

// SummaryAvailability describes summary availability.
type SummaryAvailability struct {
	Status       string `json:"status"`
	TablePresent bool   `json:"table_present"`
	Total        int    `json:"total"`
	Message      string `json:"message"`
}

// ProviderReadiness describes provider readiness.
type ProviderReadiness struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Checked  bool   `json:"checked"`
	Message  string `json:"message"`
}

// DegradedMode describes a degraded capability.
type DegradedMode struct {
	Capability string `json:"capability"`
	Severity   string `json:"severity"`
	Reason     string `json:"reason"`
}

// ExitCodeError wraps an error with an exit code.
type ExitCodeError struct {
	Code int
	Err  error
}

func (e *ExitCodeError) Error() string {
	return e.Err.Error()
}

func (e *ExitCodeError) ExitCode() int {
	return e.Code
}

// NewExitCodeError creates a new ExitCodeError.
func NewExitCodeError(code int, err error) error {
	return &ExitCodeError{Code: code, Err: err}
}

// exitCodeError is an alias for creating exit code errors.
var exitCodeError = NewExitCodeError

// RunGonchoDoctor executes the goncho doctor diagnostic.
func RunGonchoDoctor(cmd *cobra.Command, _ []string, build BuildProvenance) error {
	emitJSON, _ := cmd.Flags().GetBool("json")
	peer, _ := cmd.Flags().GetString("peer")
	sessionKey, _ := cmd.Flags().GetString("session")
	scope, _ := cmd.Flags().GetString("scope")
	sources, _ := cmd.Flags().GetStringSlice("sources")
	requireProvider, _ := cmd.Flags().GetBool("require-provider")

	peer = strings.TrimSpace(peer)
	sessionKey = strings.TrimSpace(sessionKey)
	scope = strings.TrimSpace(scope)
	if peer == "" {
		return exitCodeError(1, errors.New("goncho doctor: --peer is required"))
	}

	cfg, err := config.Load(nil)
	if err != nil {
		return exitCodeError(1, err)
	}

	memoryPath := config.MemoryDBPath()
	memoryInfo, err := os.Stat(memoryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return exitCodeError(1, fmt.Errorf("memory database not found at %s", memoryPath))
		}
		return exitCodeError(1, err)
	}

	var db *sql.DB
	if memoryInfo.Size() == 0 {
		db, err = sqlOpenGonchoUnmigrated(memoryPath)
	} else {
		db, err = sqlOpenGoncho(memoryPath)
	}
	if err != nil {
		return exitCodeError(2, fmt.Errorf("open memory db: %w", err))
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	report, code, err := BuildGonchoDoctorReport(ctx, cfg, db, peer, sessionKey, scope, sources, requireProvider, build)
	if err != nil {
		return exitCodeError(code, err)
	}

	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprint(cmd.OutOrStdout(), FormatGonchoDoctorReport(report)); err != nil {
			return err
		}
	}

	if report.ExitCode != 0 {
		return exitCodeError(report.ExitCode, fmt.Errorf("goncho doctor: %s", report.Status))
	}
	return nil
}

// BuildGonchoDoctorReport builds the full diagnostic report.
func BuildGonchoDoctorReport(ctx context.Context, cfg config.Config, db *sql.DB, peer, sessionKey, scope string, sources []string, requireProvider bool, build BuildProvenance) (GonchoDoctorReport, int, error) {
	schema, err := memory.ReadSchemaStatus(ctx, db)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			schema = memory.SchemaStatus{}
		} else {
			return GonchoDoctorReport{}, 2, err
		}
	}
	if !schema.Current || !RequiredSchemaTablesPresent(schema.Tables) {
		report := GonchoDoctorReport{
			Build:    build,
			Service:  "goncho",
			Status:   "runtime_storage_error",
			ExitCode: 2,
			Config:   CurrentGonchoDoctorConfig(cfg),
			Schema:   schema,
		}
		return report, 2, nil
	}
	memoryContract, err := memory.ReadGonchoMemoryV1Status(ctx, db)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}

	gonchoCfg := cfg.Goncho.RuntimeConfig()
	extractor, err := memory.ReadExtractorStatus(ctx, db, 5)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	queue, err := goncho.ReadQueueStatus(ctx, db, goncho.QueueStatusConfig{
		DreamEnabled:     gonchoCfg.DreamEnabled,
		WorkspaceID:      gonchoCfg.WorkspaceID,
		ObserverPeerID:   gonchoCfg.ObserverPeerID,
		DreamIdleTimeout: gonchoCfg.DreamIdleTimeout,
	})
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	sessionCatalog, err := ReadSessionCatalogStatus(config.SessionDBPath())
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}

	sessionDir, closeSessionDir, err := OpenSessionDirectoryForGonchoDoctor(scope)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	defer closeSessionDir()

	gonchoCfg.SessionDirectory = sessionDir
	svc := goncho.NewService(db, gonchoCfg, nil)
	localMemory := goncho.NewLocalMarkdownMemoryStore(db, goncho.LocalMarkdownMemoryConfig{
		Path:           DefaultGonchoMarkdownMemoryPath(),
		AgentID:        gonchoCfg.ObserverPeerID,
		WorkspaceID:    gonchoCfg.WorkspaceID,
		ObserverPeerID: gonchoCfg.ObserverPeerID,
		PeerID:         peer,
		SessionID:      sessionKey,
	})
	localMemoryStatus, err := localMemory.Status(ctx)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	toolStatus := ReadToolRegistration(svc, localMemory)
	contextStatus, err := ReadContextDryRun(ctx, svc, peer, sessionKey, scope, sources)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	conclusions, err := ReadConclusionAvailability(ctx, db)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	summaries, err := ReadSummaryAvailability(ctx, db)
	if err != nil {
		return GonchoDoctorReport{}, 2, err
	}
	provider := ReadProviderReadiness(cfg, requireProvider)

	degraded := CollectGonchoDegradedModes(queue, summaries, provider)
	exitCode := 0
	status := "healthy"
	if len(degraded) > 0 {
		status = "degraded"
	}
	if toolStatus.Status == "fail" {
		exitCode = 2
		status = "runtime_storage_error"
	}
	if provider.Status == "fail" {
		exitCode = 3
		status = "auth_provider_error"
	}

	report := GonchoDoctorReport{
		Build:               build,
		Service:             "goncho",
		Status:              status,
		ExitCode:            exitCode,
		Config:              CurrentGonchoDoctorConfig(cfg),
		Schema:              schema,
		MemoryContract:      memoryContract,
		LocalMarkdownMemory: localMemoryStatus,
		SessionCatalog:      sessionCatalog,
		ToolRegistration:    toolStatus,
		ContextDryRun:       contextStatus,
		QueueStatus: DoctorQueueStatus{
			Status:            queue.Status,
			ObservabilityOnly: queue.ObservabilityOnly,
			Extractor: ExtractorQueueSnapshot{
				WorkerHealth:     extractor.WorkerHealth,
				QueueDepth:       extractor.QueueDepth,
				DeadLetterCount:  extractor.DeadLetterCount,
				SkippedSyncCount: extractor.SkippedSyncCount,
			},
			WorkUnits: queue.WorkUnits,
			Dream:     queue.Dream,
			Message:   queue.Message,
		},
		ConclusionAvailability: conclusions,
		SummaryAvailability:    summaries,
		ProviderReadiness:      provider,
		DegradedModes:          degraded,
	}
	return report, report.ExitCode, nil
}

// OpenSessionDirectoryForGonchoDoctor opens a session directory for the doctor.
func OpenSessionDirectoryForGonchoDoctor(scope string) (goncho.SessionDirectory, func(), error) {
	if !strings.EqualFold(strings.TrimSpace(scope), "user") {
		return nil, func() {}, nil
	}
	path := config.SessionDBPath()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, func() {}, nil
		}
		return nil, func() {}, err
	}
	smap, err := session.OpenBolt(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("session catalog: open %s: %w", path, err)
	}
	return &internalgoncho.SessionDirectoryAdapter{Map: smap}, func() { _ = smap.Close() }, nil
}

// CurrentGonchoDoctorConfig returns the current goncho doctor config snapshot.
func CurrentGonchoDoctorConfig(cfg config.Config) GonchoDoctorConfig {
	configPath := config.ConfigPath()
	_, err := os.Stat(configPath)
	return GonchoDoctorConfig{
		ConfigPath:     configPath,
		ConfigExists:   err == nil,
		MemoryDBPath:   config.MemoryDBPath(),
		SessionDBPath:  config.SessionDBPath(),
		Workspace:      cfg.Goncho.Workspace,
		ObserverPeer:   cfg.Goncho.ObserverPeer,
		RecentMessages: cfg.Goncho.RecentMessages,
		DreamEnabled:   cfg.Goncho.DreamEnabled,
		DreamIdleMins:  cfg.Goncho.DreamIdleTimeoutMinutes,
		HermesModel:    cfg.Hermes.Model,
	}
}

// RequiredSchemaTablesPresent checks that all required schema tables exist.
func RequiredSchemaTablesPresent(tables map[string]bool) bool {
	for _, table := range []string{"turns", "turns_fts", "goncho_peer_cards", "goncho_conclusions", "goncho_conclusions_fts"} {
		if !tables[table] {
			return false
		}
	}
	return true
}

// ReadToolRegistration reads tool registration status.
func ReadToolRegistration(svc *goncho.Service, localMemory goncho.MemoryToolStore) ToolRegistrationStatus {
	reg := tools.NewRegistry()
	gonchotools.RegisterHonchoTools(reg, svc)
	if localMemory != nil {
		gonchotools.RegisterMemoryV1Tools(reg, localMemory)
	}
	result := doctor.CheckTools(reg)

	out := ToolRegistrationStatus{
		Status:  "ok",
		Summary: result.Summary,
	}
	if result.Status == doctor.StatusFail {
		out.Status = "fail"
	}
	for _, item := range result.Items {
		out.Registered = append(out.Registered, item.Name)
		if item.Status == doctor.StatusFail {
			out.Invalid = append(out.Invalid, item.Name)
		}
	}
	return out
}

// DefaultGonchoMarkdownMemoryPath returns the default path for the markdown memory file.
func DefaultGonchoMarkdownMemoryPath() string {
	return filepath.Join(config.GormesHome(), "memory", "GONCHO_MEMORY.md")
}

// ReadContextDryRun performs a context dry-run for the doctor.
func ReadContextDryRun(ctx context.Context, svc *goncho.Service, peer, sessionKey, scope string, sources []string) (ContextDryRunStatus, error) {
	includeDreamStatus := true
	result, err := svc.Context(ctx, goncho.ContextParams{
		Peer:               peer,
		Query:              "doctor dry-run",
		MaxTokens:          400,
		SessionKey:         sessionKey,
		Scope:              scope,
		Sources:            sources,
		IncludeDreamStatus: &includeDreamStatus,
	})
	if err != nil {
		return ContextDryRunStatus{}, err
	}
	return ContextDryRunStatus{
		Status:         "ok",
		Peer:           result.Peer,
		SessionKey:     result.SessionKey,
		Representation: result.Representation,
		ScopeEvidence:  result.ScopeEvidence,
		SearchResults:  result.SearchResults,
		Unavailable:    result.Unavailable,
	}, nil
}

// ReadConclusionAvailability reads conclusion availability from the database.
func ReadConclusionAvailability(ctx context.Context, db *sql.DB) (ConclusionAvailability, error) {
	var total int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goncho_conclusions`).Scan(&total); err != nil {
		return ConclusionAvailability{}, fmt.Errorf("goncho doctor: conclusion count: %w", err)
	}
	status := "ok"
	if total == 0 {
		status = "zero_state"
	}
	out := ConclusionAvailability{Status: status, Total: total}

	rows, err := db.QueryContext(ctx, `
		SELECT observer_peer_id, peer_id, COUNT(*)
		FROM goncho_conclusions
		GROUP BY observer_peer_id, peer_id
		ORDER BY COUNT(*) DESC, observer_peer_id, peer_id
		LIMIT 10
	`)
	if err != nil {
		return ConclusionAvailability{}, fmt.Errorf("goncho doctor: conclusion pairs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pair ConclusionPair
		if err := rows.Scan(&pair.ObserverPeerID, &pair.PeerID, &pair.Count); err != nil {
			return ConclusionAvailability{}, fmt.Errorf("goncho doctor: scan conclusion pair: %w", err)
		}
		out.Pairs = append(out.Pairs, pair)
	}
	if err := rows.Err(); err != nil {
		return ConclusionAvailability{}, fmt.Errorf("goncho doctor: conclusion pair rows: %w", err)
	}
	return out, nil
}

// ReadSummaryAvailability reads summary availability from the database.
func ReadSummaryAvailability(ctx context.Context, db *sql.DB) (SummaryAvailability, error) {
	present, err := sqliteTablePresent(ctx, db, "goncho_session_summaries")
	if err != nil {
		return SummaryAvailability{}, err
	}
	if !present {
		return SummaryAvailability{
			Status:       "degraded",
			TablePresent: false,
			Message:      "goncho_session_summaries table unavailable; summary capability is degraded",
		}, nil
	}

	var total int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goncho_session_summaries`).Scan(&total); err != nil {
		return SummaryAvailability{}, fmt.Errorf("goncho doctor: summary count: %w", err)
	}
	status := "ok"
	if total == 0 {
		status = "zero_state"
	}
	return SummaryAvailability{Status: status, TablePresent: true, Total: total}, nil
}

// ReadSessionCatalogStatus reads session catalog status from the bolt DB.
func ReadSessionCatalogStatus(path string) (SessionCatalogStatus, error) {
	out := SessionCatalogStatus{
		Status:  "zero_state",
		Path:    path,
		Message: "no session catalog data",
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return SessionCatalogStatus{}, err
	}
	out.Exists = true

	db, err := bolt.Open(path, 0o600, &bolt.Options{ReadOnly: true, Timeout: 100 * time.Millisecond})
	if err != nil {
		return SessionCatalogStatus{}, fmt.Errorf("session catalog: open %s: %w", path, err)
	}
	defer db.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		out.SessionCount = countBoltBucket(tx.Bucket([]byte("sessions_v1")))
		out.MetadataCount = countBoltBucket(tx.Bucket([]byte("session_meta_v1")))
		return nil
	}); err != nil {
		return SessionCatalogStatus{}, err
	}
	if out.SessionCount > 0 || out.MetadataCount > 0 {
		out.Status = "ok"
		out.Message = "session catalog readable"
	}
	return out, nil
}

func countBoltBucket(bucket *bolt.Bucket) int {
	if bucket == nil {
		return 0
	}
	count := 0
	_ = bucket.ForEach(func(_, _ []byte) error {
		count++
		return nil
	})
	return count
}

// ReadProviderReadiness checks provider readiness.
func ReadProviderReadiness(cfg config.Config, required bool) ProviderReadiness {
	if required {
		if strings.TrimSpace(cfg.Hermes.APIKey) == "" {
			return ProviderReadiness{
				Status:   "fail",
				Required: true,
				Checked:  true,
				Message:  "provider auth required but GORMES_API_KEY is not configured",
			}
		}
		return ProviderReadiness{
			Status:   "ok",
			Required: true,
			Checked:  true,
			Message:  "provider auth is configured; network reachability is not checked by Goncho doctor",
		}
	}
	return ProviderReadiness{
		Status:   "degraded",
		Required: false,
		Checked:  false,
		Message:  "missing optional model/provider features are degraded, not startup failures; no provider network check was run",
	}
}

// CollectGonchoDegradedModes collects all degraded modes from various subsystems.
func CollectGonchoDegradedModes(queue goncho.QueueStatus, summaries SummaryAvailability, provider ProviderReadiness) []DegradedMode {
	var out []DegradedMode
	if queue.Degraded {
		out = append(out, DegradedMode{
			Capability: "goncho_task_queue",
			Severity:   "degraded",
			Reason:     queue.Message,
		})
	}
	if queue.Dream.Status == "dream_disabled" || queue.Dream.Status == "dream_unavailable" {
		reason := queue.Dream.Status
		if len(queue.Dream.Evidence) > 0 {
			reason = queue.Dream.Evidence[0].Reason
		}
		out = append(out, DegradedMode{
			Capability: "goncho_dream_scheduler",
			Severity:   "degraded",
			Reason:     reason,
		})
	}
	if summaries.Status == "degraded" {
		out = append(out, DegradedMode{
			Capability: "session_summaries",
			Severity:   "degraded",
			Reason:     summaries.Message,
		})
	}
	if provider.Status == "degraded" {
		out = append(out, DegradedMode{
			Capability: "model_provider",
			Severity:   "degraded",
			Reason:     provider.Message,
		})
	}
	return out
}

func sqliteTablePresent(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var found string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("sqlite table %s: %w", name, err)
	}
	return found == name, nil
}

// FormatGonchoDoctorReport renders the doctor report as human-readable text.
func FormatGonchoDoctorReport(report GonchoDoctorReport) string {
	var b strings.Builder
	b.WriteString("Goncho doctor\n")
	fmt.Fprintf(&b, "status: %s\n", report.Status)
	fmt.Fprintf(&b, "exit_code: %d\n\n", report.ExitCode)

	b.WriteString("Config\n")
	fmt.Fprintf(&b, "config_path: %s\n", report.Config.ConfigPath)
	fmt.Fprintf(&b, "config_exists: %t\n", report.Config.ConfigExists)
	fmt.Fprintf(&b, "memory_db_path: %s\n", report.Config.MemoryDBPath)
	fmt.Fprintf(&b, "session_db_path: %s\n", report.Config.SessionDBPath)
	fmt.Fprintf(&b, "workspace: %s\n", report.Config.Workspace)
	fmt.Fprintf(&b, "observer_peer: %s\n", report.Config.ObserverPeer)
	fmt.Fprintf(&b, "recent_messages: %d\n", report.Config.RecentMessages)
	fmt.Fprintf(&b, "dream_enabled: %t\n", report.Config.DreamEnabled)
	fmt.Fprintf(&b, "dream_idle_timeout_minutes: %d\n\n", report.Config.DreamIdleMins)

	b.WriteString("Schema\n")
	fmt.Fprintf(&b, "schema_version: %s\n", report.Schema.Version)
	fmt.Fprintf(&b, "current_schema_version: %s\n", report.Schema.CurrentVersion)
	for _, table := range []string{"turns", "turns_fts", "goncho_peer_cards", "goncho_conclusions", "goncho_conclusions_fts"} {
		fmt.Fprintf(&b, "%s: %s\n", table, presentWord(report.Schema.Tables[table]))
	}
	b.WriteString("\n")

	b.WriteString("Memory contract\n")
	fmt.Fprintf(&b, "contract_version: %s\n", report.MemoryContract.ContractVersion)
	fmt.Fprintf(&b, "markdown_format_version: %s\n", report.MemoryContract.MarkdownFormatVersion)
	fmt.Fprintf(&b, "mcp_tool_contract_version: %s\n", report.MemoryContract.MCPToolContractVersion)
	fmt.Fprintf(&b, "private_agent_memory_default: %t\n", report.MemoryContract.PrivateAgentMemoryDefault)
	fmt.Fprintf(&b, "self_improvement_per_agent_default: %t\n", report.MemoryContract.SelfImprovementPerAgentDefault)
	fmt.Fprintf(&b, "foreign_config_runtime_reads: %s\n", report.MemoryContract.ForeignConfigRuntimeReads)
	fmt.Fprintf(&b, "fast_recall_path: %s\n", strings.Join(report.MemoryContract.FastRecallPath, ", "))
	fmt.Fprintf(&b, "optional_quality_layers: %s\n", strings.Join(report.MemoryContract.OptionalQualityLayers, ", "))
	for _, table := range []string{"goncho_memory_items", "goncho_memory_items_fts", "goncho_memory_eval_artifacts"} {
		fmt.Fprintf(&b, "%s: %s\n", table, presentWord(report.MemoryContract.Tables[table]))
	}
	b.WriteString("\n")

	b.WriteString("Local markdown memory\n")
	fmt.Fprintf(&b, "path: %s\n", report.LocalMarkdownMemory.Path)
	fmt.Fprintf(&b, "enabled: %t\n", report.LocalMarkdownMemory.Enabled)
	fmt.Fprintf(&b, "local_first: %t\n", report.LocalMarkdownMemory.LocalFirst)
	fmt.Fprintf(&b, "sqlite_backed: %t\n", report.LocalMarkdownMemory.SQLiteBacked)
	fmt.Fprintf(&b, "markdown_backed: %t\n", report.LocalMarkdownMemory.MarkdownBacked)
	fmt.Fprintf(&b, "network_required: %t\n", report.LocalMarkdownMemory.NetworkRequired)
	fmt.Fprintf(&b, "ollama_required: %t\n", report.LocalMarkdownMemory.OllamaRequired)
	fmt.Fprintf(&b, "mcp_tools: %s\n", strings.Join(report.LocalMarkdownMemory.MCPTools, ", "))
	if len(report.LocalMarkdownMemory.Evidence) > 0 {
		fmt.Fprintf(&b, "evidence: %s\n", strings.Join(report.LocalMarkdownMemory.Evidence, ", "))
	}
	b.WriteString("\n")

	b.WriteString("Session catalog\n")
	fmt.Fprintf(&b, "path: %s\n", report.SessionCatalog.Path)
	fmt.Fprintf(&b, "status: %s\n", report.SessionCatalog.Status)
	fmt.Fprintf(&b, "sessions: %d\n", report.SessionCatalog.SessionCount)
	fmt.Fprintf(&b, "metadata_rows: %d\n", report.SessionCatalog.MetadataCount)
	fmt.Fprintf(&b, "%s\n\n", report.SessionCatalog.Message)

	b.WriteString("Tool registration\n")
	fmt.Fprintf(&b, "status: %s\n", report.ToolRegistration.Status)
	fmt.Fprintf(&b, "summary: %s\n", report.ToolRegistration.Summary)
	fmt.Fprintf(&b, "tools: %s\n\n", strings.Join(report.ToolRegistration.Registered, ", "))

	b.WriteString("Context dry-run\n")
	fmt.Fprintf(&b, "peer: %s\n", report.ContextDryRun.Peer)
	fmt.Fprintf(&b, "session_key: %s\n", valueOrNone(report.ContextDryRun.SessionKey))
	fmt.Fprintf(&b, "representation: %s\n", report.ContextDryRun.Representation)
	b.WriteString(FormatCrossChatScopeEvidence(report.ContextDryRun.ScopeEvidence))
	b.WriteString(FormatContextSearchResults(report.ContextDryRun.SearchResults))
	if len(report.ContextDryRun.Unavailable) == 0 {
		b.WriteString("unavailable: none\n\n")
	} else {
		b.WriteString("unavailable:\n")
		for _, item := range report.ContextDryRun.Unavailable {
			fmt.Fprintf(&b, "- %s: %s\n", item.Capability, item.Reason)
		}
		b.WriteString("\n")
	}

	b.WriteString("Queue status (observability/debugging only; not synchronization; do not wait for empty queue)\n")
	fmt.Fprintf(&b, "extractor_worker_health: %s\n", report.QueueStatus.Extractor.WorkerHealth)
	fmt.Fprintf(&b, "extractor_queue_depth: %d\n", report.QueueStatus.Extractor.QueueDepth)
	fmt.Fprintf(&b, "extractor_dead_letters: %d\n", report.QueueStatus.Extractor.DeadLetterCount)
	for _, taskType := range goncho.QueueTaskTypes {
		counts := report.QueueStatus.WorkUnits[taskType]
		fmt.Fprintf(&b, "%s: total=%d pending=%d in_progress=%d completed=%d\n",
			taskType,
			counts.TotalWorkUnits,
			counts.PendingWorkUnits,
			counts.InProgressWorkUnits,
			counts.CompletedWorkUnits,
		)
	}
	b.WriteString(memoryapp.FormatDreamQueueEvidence(report.QueueStatus.Dream))
	fmt.Fprintf(&b, "goncho_queue: %s\n\n", report.QueueStatus.Message)

	b.WriteString("Conclusion availability\n")
	fmt.Fprintf(&b, "status: %s\n", report.ConclusionAvailability.Status)
	fmt.Fprintf(&b, "conclusion_count: %d\n\n", report.ConclusionAvailability.Total)

	b.WriteString("Summary availability\n")
	fmt.Fprintf(&b, "status: %s\n", report.SummaryAvailability.Status)
	fmt.Fprintf(&b, "summary_table: %s\n", availableWord(report.SummaryAvailability.TablePresent))
	fmt.Fprintf(&b, "summary_count: %d\n\n", report.SummaryAvailability.Total)

	b.WriteString("Provider readiness\n")
	fmt.Fprintf(&b, "status: %s\n", report.ProviderReadiness.Status)
	fmt.Fprintf(&b, "required: %t\n", report.ProviderReadiness.Required)
	fmt.Fprintf(&b, "checked: %t\n", report.ProviderReadiness.Checked)
	fmt.Fprintf(&b, "optional_provider_checks: %s\n", report.ProviderReadiness.Status)
	fmt.Fprintf(&b, "message: %s\n\n", report.ProviderReadiness.Message)

	b.WriteString("Degraded modes\n")
	if len(report.DegradedModes) == 0 {
		b.WriteString("none\n")
		return b.String()
	}
	for _, item := range report.DegradedModes {
		fmt.Fprintf(&b, "- %s: %s (%s)\n", item.Capability, item.Severity, item.Reason)
	}
	return b.String()
}

// FormatCrossChatScopeEvidence formats cross-chat scope evidence for display.
func FormatCrossChatScopeEvidence(evidence *goncho.CrossChatRecallEvidence) string {
	if evidence == nil {
		return "scope_evidence: none\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "scope_evidence: decision=%s user_id=%s scope=%s",
		evidence.Decision,
		valueOrNone(evidence.UserID),
		valueOrNone(evidence.Scope),
	)
	if evidence.FallbackScope != "" {
		fmt.Fprintf(&b, " fallback_scope=%s", evidence.FallbackScope)
	}
	if len(evidence.SourceAllowlist) > 0 {
		fmt.Fprintf(&b, " source_allowlist=%s", strings.Join(evidence.SourceAllowlist, ","))
	}
	fmt.Fprintf(&b, " sessions_considered=%d widened_sessions_considered=%d\n",
		evidence.SessionsConsidered,
		evidence.WidenedSessionsConsidered,
	)
	if evidence.Reason != "" {
		fmt.Fprintf(&b, "scope_reason: %s\n", evidence.Reason)
	}
	if evidence.CurrentBinding != nil {
		fmt.Fprintf(&b, "current_binding: session_id=%s source=%s chat_id=%s chat_key=%s\n",
			valueOrNone(evidence.CurrentBinding.SessionID),
			valueOrNone(evidence.CurrentBinding.Source),
			valueOrNone(evidence.CurrentBinding.ChatID),
			valueOrNone(evidence.CurrentBinding.ChatKey),
		)
	}
	if len(evidence.Sessions) > 0 {
		b.WriteString("scope_sessions:\n")
		for _, item := range evidence.Sessions {
			fmt.Fprintf(&b, "- session_id=%s source=%s chat_id=%s current=%t\n",
				valueOrNone(item.SessionID),
				valueOrNone(item.Source),
				valueOrNone(item.ChatID),
				item.Current,
			)
		}
	}
	return b.String()
}

// FormatContextSearchResults formats search results for display.
func FormatContextSearchResults(results []goncho.SearchHit) string {
	if len(results) == 0 {
		return "search_results: none\n"
	}
	var b strings.Builder
	b.WriteString("search_results:\n")
	for _, hit := range results {
		fmt.Fprintf(&b, "- source=%s origin_source=%s session_key=%s",
			valueOrNone(hit.Source),
			valueOrNone(hit.OriginSource),
			valueOrNone(hit.SessionKey),
		)
		appendSearchLineageEvidence(&b, hit.Lineage)
		fmt.Fprintf(&b, " content=%q\n", hit.Content)
	}
	return b.String()
}

func appendSearchLineageEvidence(b *strings.Builder, lineage *goncho.SearchLineage) {
	if lineage == nil {
		return
	}
	fmt.Fprintf(b, " lineage_status=%s", valueOrNone(lineage.Status))
	if lineage.ParentSessionID != "" {
		fmt.Fprintf(b, " parent_session_id=%s", lineage.ParentSessionID)
	}
	if lineage.LineageKind != "" {
		fmt.Fprintf(b, " lineage_kind=%s", lineage.LineageKind)
	}
	if len(lineage.ChildSessionIDs) > 0 {
		fmt.Fprintf(b, " child_session_ids=%s", strings.Join(lineage.ChildSessionIDs, ","))
	}
}

func presentWord(ok bool) string {
	if ok {
		return "present"
	}
	return "missing"
}

func availableWord(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
}

func valueOrNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}