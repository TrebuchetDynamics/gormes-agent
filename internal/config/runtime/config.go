package runtimeconfig

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

// Updates controls `gormes update` behavior. PreUpdateBackup is the
// config equivalent of `--backup` and is silent-default off. BackupKeep
// is the retention budget applied after a successful write; <=0 keeps
// the built-in default of 5 (matches Hermes' upstream).
type Updates struct {
	PreUpdateBackup bool `toml:"pre_update_backup" yaml:"pre_update_backup"`
	BackupKeep      int  `toml:"backup_keep" yaml:"backup_keep"`
}

type Cron struct {
	Enabled        bool          `toml:"enabled" yaml:"enabled"`
	CallTimeout    time.Duration `toml:"call_timeout" yaml:"call_timeout"`
	MirrorInterval time.Duration `toml:"mirror_interval" yaml:"mirror_interval"`
	MirrorPath     string        `toml:"mirror_path" yaml:"mirror_path"`
}

// Approvals mirrors Hermes' approval policy settings that affect native Go tools.
type Approvals struct {
	CronMode string `toml:"cron_mode" yaml:"cron_mode"`
}

// Web mirrors Hermes config.yaml's web.backend and web.use_gateway fields.
type Web struct {
	Backend    string `toml:"backend" yaml:"backend"`
	UseGateway bool   `toml:"use_gateway" yaml:"use_gateway"`
}

// Browser mirrors Hermes browser/CDP connection settings used by browser tools
// and CDP-backed web_extract fallback.
type Browser struct {
	CDPURL string `toml:"cdp_url" yaml:"cdp_url"`
}

// Workspace configures workspace-level file access policy.
type Workspace struct {
	// Mode controls file write access for tools operating inside the
	// workspace. Empty string or "readwrite" allows writes;
	// "readonly" denies all tool writes regardless of scope.
	Mode string `toml:"mode" yaml:"mode"`
}

// WorkspaceModeReadonly is the canonical value for readonly mode.
const WorkspaceModeReadonly = "readonly"

// WorkspaceModeReadWrite is the canonical value for read-write mode.
const WorkspaceModeReadWrite = "readwrite"

// IsWorkspaceReadonly returns true when the workspace mode enforces read-only access.
func (w Workspace) IsWorkspaceReadonly() bool {
	return w.Mode == WorkspaceModeReadonly
}

// Security mirrors Hermes config.yaml security controls that affect native Go tools.
type Security struct {
	WebsiteBlocklist WebsiteBlocklist `toml:"website_blocklist" yaml:"website_blocklist"`
}

type WebsiteBlocklist struct {
	Enabled     bool     `toml:"enabled" yaml:"enabled"`
	Domains     []string `toml:"domains" yaml:"domains"`
	SharedFiles []string `toml:"shared_files" yaml:"shared_files"`
	BaseDir     string   `toml:"-"`
}

type Hermes struct {
	Endpoint              string                 `toml:"endpoint" yaml:"endpoint"`
	APIKey                string                 `toml:"api_key" yaml:"api_key"`
	APIKeyRef             *credentials.SecretRef `toml:"api_key_ref" yaml:"api_key_ref" json:"api_key_ref,omitempty"`
	Model                 string                 `toml:"model" yaml:"model"`
	Provider              string                 `toml:"provider" yaml:"provider"`
	ModelResolutionSource string                 `toml:"-" yaml:"-" json:"model_resolution_source,omitempty"`
}

type Agent struct {
	ImageInputMode      string            `toml:"image_input_mode" yaml:"image_input_mode"`
	MaxTurns            int               `toml:"max_turns" yaml:"max_turns"`
	ReasoningEffort     string            `toml:"reasoning_effort" yaml:"reasoning_effort"`
	GatewayTimeout      int               `toml:"gateway_timeout" yaml:"gateway_timeout"`
	GatewayTimeoutWarn  int               `toml:"gateway_timeout_warning" yaml:"gateway_timeout_warning"`
	APIMaxRetries       int               `toml:"api_max_retries" yaml:"api_max_retries"`
	Verbose             bool              `toml:"verbose" yaml:"verbose"`
	Personalities       map[string]string `toml:"personalities" yaml:"personalities"`
	ActivePersonality   string            `toml:"active_personality" yaml:"active_personality"`
	PrefillMessagesFile string            `toml:"prefill_messages_file" yaml:"prefill_messages_file"`
}

type Runtime struct {
	MaxToolIterations         int     `toml:"max_tool_iterations" yaml:"max_tool_iterations"`
	TerminalBackend           string  `toml:"terminal_backend" yaml:"terminal_backend"`
	TTSProvider               string  `toml:"tts_provider" yaml:"tts_provider"`
	CompressionThreshold      float64 `toml:"compression_threshold" yaml:"compression_threshold"`
	SessionResetPolicy        string  `toml:"session_reset_policy" yaml:"session_reset_policy"`
	SessionResetAfterMinutes  int     `toml:"session_reset_after_minutes" yaml:"session_reset_after_minutes"`
	SessionResetDailyHour     int     `toml:"session_reset_daily_hour" yaml:"session_reset_daily_hour"`
	SessionResetMemorySummary bool    `toml:"session_reset_memory_summary" yaml:"session_reset_memory_summary"`
}

type Terminal struct {
	Backend string `toml:"backend" yaml:"backend"`
	CWD     string `toml:"cwd" yaml:"cwd"`
}

// CodeExecution controls the native execute_code tool mode. Hermes defaults to
// project mode; Gormes keeps strict as the built-in default until the shell-only
// guard is intentionally relaxed by explicit config.
type CodeExecution struct {
	Mode string `toml:"mode" yaml:"mode"`
}

type TUI struct {
	Theme         string `toml:"theme" yaml:"theme"`
	MouseTracking bool   `toml:"mouse_tracking" yaml:"mouse_tracking"`
}

type Input struct {
	MaxBytes int `toml:"max_bytes" yaml:"max_bytes"`
	MaxLines int `toml:"max_lines" yaml:"max_lines"`
}
