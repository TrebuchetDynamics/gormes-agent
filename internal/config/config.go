// Package config loads Gormes configuration from CLI flags > env > TOML > defaults.
package config

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// CurrentConfigVersion is the schema version this binary writes + accepts.
// When a breaking change to the TOML schema lands, bump this constant and
// add a migration in runMigrations() so older files stay readable.
const CurrentConfigVersion = 2

type Config struct {
	// ConfigVersion is the canonical schema version of the loaded TOML file.
	// Absent in TOML is treated as legacy v1. LegacyConfigVersion keeps
	// `_config_version` readable as a fallback, but new writes use
	// `config_version`.
	ConfigVersion       int `toml:"config_version" yaml:"config_version"`
	LegacyConfigVersion int `toml:"_config_version,omitempty" yaml:"_config_version,omitempty"`

	Hermes        HermesCfg                `toml:"hermes" yaml:"hermes"`
	Profiles      map[string]ProfileCfg    `toml:"profiles" yaml:"profiles"`
	Credentials   map[string]CredentialCfg `toml:"credentials" yaml:"credentials"`
	Agent         AgentRuntimeCfg          `toml:"agent" yaml:"agent"`
	Runtime       RuntimeCfg               `toml:"runtime" yaml:"runtime"`
	TTS           map[string]any           `toml:"tts" yaml:"tts"`
	ImageGen      map[string]any           `toml:"image_gen" yaml:"image_gen"`
	Gateway       GatewayCfg               `toml:"gateway" yaml:"gateway"`
	Terminal      TerminalCfg              `toml:"terminal" yaml:"terminal"`
	CodeExecution CodeExecutionCfg         `toml:"code_execution" yaml:"code_execution"`
	Display       DisplayCfg               `toml:"display" yaml:"display"`
	TUI           TUICfg                   `toml:"tui" yaml:"tui"`
	Input         InputCfg                 `toml:"input" yaml:"input"`
	Approvals     ApprovalsCfg             `toml:"approvals" yaml:"approvals"`
	Voice         VoiceCfg                 `toml:"voice" yaml:"voice"`
	STT           STTCfg                   `toml:"stt" yaml:"stt"`
	Auxiliary     AuxiliaryCfg             `toml:"auxiliary" yaml:"auxiliary"`
	Curator       CuratorCfg               `toml:"curator" yaml:"curator"`
	Telegram      TelegramCfg              `toml:"telegram" yaml:"telegram"`
	Discord       DiscordCfg               `toml:"discord" yaml:"discord"`
	Slack         SlackCfg                 `toml:"slack" yaml:"slack"`
	Teams         TeamsCfg                 `toml:"teams" yaml:"teams"`
	Yuanbao       YuanbaoCfg               `toml:"yuanbao" yaml:"yuanbao"`
	Web           WebCfg                   `toml:"web" yaml:"web"`
	Navivox       NavivoxCfg               `toml:"navivox" yaml:"navivox"`
	Browser       BrowserCfg               `toml:"browser" yaml:"browser"`
	Security      SecurityCfg              `toml:"security" yaml:"security"`
	Secrets       SecretsCfg               `toml:"secrets" yaml:"secrets"`
	Agents        AgentsCfg                `toml:"agents" yaml:"agents"`
	Bindings      []AgentBindingCfg        `toml:"bindings" yaml:"bindings"`
	Cron          CronCfg                  `toml:"cron" yaml:"cron"`
	Skills        SkillsCfg                `toml:"skills" yaml:"skills"`
	Delegation    DelegationCfg            `toml:"delegation" yaml:"delegation"`
	Goncho        GonchoCfg                `toml:"goncho" yaml:"goncho"`
	Updates       UpdatesCfg               `toml:"updates" yaml:"updates"`
	// Resume is set only via the --resume CLI flag; intentionally not
	// a TOML field. Empty means "use whatever internal/session had
	// persisted for this binary's default key."
	Resume string `toml:"-"`
}

// UpdatesCfg controls `gormes update` behavior. PreUpdateBackup is the
// config equivalent of `--backup` and is silent-default off. BackupKeep
// is the retention budget applied after a successful write; <=0 keeps
// the built-in default of 5 (matches Hermes' upstream).
type UpdatesCfg struct {
	PreUpdateBackup bool `toml:"pre_update_backup" yaml:"pre_update_backup"`
	BackupKeep      int  `toml:"backup_keep" yaml:"backup_keep"`
}

type TelegramAccountCfg struct {
	BotToken       string  `toml:"bot_token" yaml:"bot_token"`
	AllowedChatID  int64   `toml:"allowed_chat_id" yaml:"allowed_chat_id"`
	AllowedUserIDs []int64 `toml:"allowed_user_ids" yaml:"allowed_user_ids"`
}

type TelegramHomeChannelCfg struct {
	Platform string `toml:"platform" yaml:"platform"`
	ChatID   string `toml:"chat_id" yaml:"chat_id"`
	Name     string `toml:"name" yaml:"name"`
	ThreadID string `toml:"thread_id" yaml:"thread_id"`
}

type TelegramCfg struct {
	BotToken               string                        `toml:"bot_token" yaml:"bot_token"`
	BotTokenRef            *SecretRef                    `toml:"bot_token_ref" yaml:"bot_token_ref" json:"bot_token_ref,omitempty"`
	Accounts               map[string]TelegramAccountCfg `toml:"accounts" yaml:"accounts"`
	AccountID              string                        `toml:"-" yaml:"-"`
	AllowedChatID          int64                         `toml:"allowed_chat_id" yaml:"allowed_chat_id"`
	HomeChannel            TelegramHomeChannelCfg        `toml:"home_channel" yaml:"home_channel"`
	AllowedChats           any                           `toml:"allowed_chats" yaml:"allowed_chats"`
	AllowedUserIDs         []int64                       `toml:"allowed_user_ids" yaml:"allowed_user_ids"`
	RequireMention         bool                          `toml:"require_mention" yaml:"require_mention"`
	GuestMode              bool                          `toml:"guest_mode" yaml:"guest_mode"`
	BotUsername            string                        `toml:"bot_username" yaml:"bot_username"`
	CoalesceMs             int                           `toml:"coalesce_ms" yaml:"coalesce_ms"`
	FreshFinalAfterSeconds float64                       `toml:"fresh_final_after_seconds" yaml:"fresh_final_after_seconds"`
	Notifications          string                        `toml:"notifications" yaml:"notifications"`
	FirstRunDiscovery      bool                          `toml:"first_run_discovery" yaml:"first_run_discovery"`
	// MemoryQueueCap (Phase 3.A): async worker queue capacity in
	// the telegram subcommand's SqliteStore. Defaults to 1024.
	MemoryQueueCap int `toml:"memory_queue_cap" yaml:"memory_queue_cap"`
	// ExtractorBatchSize / ExtractorPollInterval (Phase 3.B).
	ExtractorBatchSize    int           `toml:"extractor_batch_size" yaml:"extractor_batch_size"`
	ExtractorPollInterval time.Duration `toml:"extractor_poll_interval" yaml:"extractor_poll_interval"`
	// RecallEnabled / RecallWeightThreshold / RecallMaxFacts / RecallDepth
	// (Phase 3.C).
	RecallEnabled         bool    `toml:"recall_enabled" yaml:"recall_enabled"`
	RecallWeightThreshold float64 `toml:"recall_weight_threshold" yaml:"recall_weight_threshold"`
	RecallMaxFacts        int     `toml:"recall_max_facts" yaml:"recall_max_facts"`
	RecallDepth           int     `toml:"recall_depth" yaml:"recall_depth"`
	// RecallDecayHorizonDays (Phase 3.E.6) — maps to
	// RecallConfig.DecayHorizonDays. An edge's effective weight
	// decays linearly from raw at age=0 to 0 at this many days old.
	// 0 = unset (withDefaults promotes to 180). <0 = disabled.
	RecallDecayHorizonDays int `toml:"recall_decay_horizon_days" yaml:"recall_decay_horizon_days"`
	// MirrorEnabled / MirrorPath / MirrorInterval (Phase 3.D.5).
	// The Memory Mirror exports SQLite entities/relationships to USER.md.
	MirrorEnabled  bool          `toml:"mirror_enabled" yaml:"mirror_enabled"`
	MirrorPath     string        `toml:"mirror_path" yaml:"mirror_path"`
	MirrorInterval time.Duration `toml:"mirror_interval" yaml:"mirror_interval"`
	// Phase 3.D semantic fusion — all opt-in via SemanticEnabled.
	SemanticEnabled       bool          `toml:"semantic_enabled" yaml:"semantic_enabled"`
	SemanticEndpoint      string        `toml:"semantic_endpoint" yaml:"semantic_endpoint"`
	SemanticModel         string        `toml:"semantic_model" yaml:"semantic_model"`
	SemanticTopK          int           `toml:"semantic_top_k" yaml:"semantic_top_k"`
	SemanticMinSimilarity float64       `toml:"semantic_min_similarity" yaml:"semantic_min_similarity"`
	EmbedderPollInterval  time.Duration `toml:"embedder_poll_interval" yaml:"embedder_poll_interval"`
	EmbedderBatchSize     int           `toml:"embedder_batch_size" yaml:"embedder_batch_size"`
	EmbedderCallTimeout   time.Duration `toml:"embedder_call_timeout" yaml:"embedder_call_timeout"`
	QueryEmbedTimeout     time.Duration `toml:"query_embed_timeout" yaml:"query_embed_timeout"`
}

func (c TelegramCfg) AllowedChatIDs() []string {
	values := flexibleStringList(c.AllowedChats)
	if strings.TrimSpace(c.HomeChannel.ChatID) != "" {
		values = append([]string{strings.TrimSpace(c.HomeChannel.ChatID)}, values...)
	}
	return compactStrings(values)
}

// DiscordCfg drives the Discord channel adapter.
type DiscordAccountCfg struct {
	Token            string   `toml:"token" yaml:"token"`
	AllowedChannelID string   `toml:"allowed_channel_id" yaml:"allowed_channel_id"`
	AllowedChannels  []string `toml:"allowed_channels" yaml:"allowed_channels"`
}

type DiscordCfg struct {
	Token                string                       `toml:"token" yaml:"token"`
	TokenRef             *SecretRef                   `toml:"token_ref" yaml:"token_ref" json:"token_ref,omitempty"`
	Accounts             map[string]DiscordAccountCfg `toml:"accounts" yaml:"accounts"`
	AllowedChannelID     string                       `toml:"allowed_channel_id" yaml:"allowed_channel_id"`
	AllowedChannels      any                          `toml:"allowed_channels" yaml:"allowed_channels"`
	IgnoredChannels      any                          `toml:"ignored_channels" yaml:"ignored_channels"`
	FreeResponseChannels any                          `toml:"free_response_channels" yaml:"free_response_channels"`
	NoThreadChannels     any                          `toml:"no_thread_channels" yaml:"no_thread_channels"`
	ChannelSkillBindings any                          `toml:"channel_skill_bindings" yaml:"channel_skill_bindings"`
	ChannelPrompts       any                          `toml:"channel_prompts" yaml:"channel_prompts"`
	RequireMention       any                          `toml:"require_mention" yaml:"require_mention"`
	AutoThread           any                          `toml:"auto_thread" yaml:"auto_thread"`
	ReplyToMode          string                       `toml:"reply_to_mode" yaml:"reply_to_mode"`
	AllowBots            string                       `toml:"allow_bots" yaml:"allow_bots"`
	ServerActions        []string                     `toml:"server_actions" yaml:"server_actions"`
	CoalesceMs           int                          `toml:"coalesce_ms" yaml:"coalesce_ms"`
	FirstRunDiscovery    bool                         `toml:"first_run_discovery" yaml:"first_run_discovery"`
	AccountID            string                       `toml:"-" yaml:"-"`
}

func (c DiscordCfg) Enabled() bool {
	if c.Token == "" {
		return false
	}
	return c.AllowedChannelID != "" || len(flexibleStringList(c.AllowedChannels)) > 0 || c.FirstRunDiscovery
}

func (c DiscordCfg) AllowedChannelIDs() []string {
	values := flexibleStringList(c.AllowedChannels)
	if c.AllowedChannelID != "" {
		values = append([]string{strings.TrimSpace(c.AllowedChannelID)}, values...)
	}
	return compactStrings(values)
}

func (c DiscordCfg) IgnoredChannelIDs() []string {
	return flexibleStringList(c.IgnoredChannels)
}

func (c DiscordCfg) FreeResponseChannelIDs() []string {
	return flexibleStringList(c.FreeResponseChannels)
}

func (c DiscordCfg) NoThreadChannelIDs() []string {
	return flexibleStringList(c.NoThreadChannels)
}

func (c DiscordCfg) RequireMentionValue(defaultValue bool) bool {
	return flexibleBool(c.RequireMention, defaultValue)
}

func (c DiscordCfg) AutoThreadValue(defaultValue bool) bool {
	return flexibleBool(c.AutoThread, defaultValue)
}

func (c DiscordCfg) ReplyToModeValue() string {
	switch strings.ToLower(strings.TrimSpace(c.ReplyToMode)) {
	case "off":
		return "off"
	case "all":
		return "all"
	default:
		return "first"
	}
}

func (c DiscordCfg) AllowBotsValue() string {
	switch strings.ToLower(strings.TrimSpace(c.AllowBots)) {
	case "all":
		return "all"
	case "mentions":
		return "mentions"
	default:
		return "none"
	}
}

type SlackAccountCfg struct {
	BotToken         string `toml:"bot_token" yaml:"bot_token"`
	AppToken         string `toml:"app_token" yaml:"app_token"`
	AllowedChannelID string `toml:"allowed_channel_id" yaml:"allowed_channel_id"`
}

// SlackCfg drives the Slack Socket Mode channel adapter.
type SlackCfg struct {
	Enabled              bool                       `toml:"enabled" yaml:"enabled"`
	BotToken             string                     `toml:"bot_token" yaml:"bot_token"`
	BotTokenRef          *SecretRef                 `toml:"bot_token_ref" yaml:"bot_token_ref" json:"bot_token_ref,omitempty"`
	AppToken             string                     `toml:"app_token" yaml:"app_token"`
	AppTokenRef          *SecretRef                 `toml:"app_token_ref" yaml:"app_token_ref" json:"app_token_ref,omitempty"`
	Accounts             map[string]SlackAccountCfg `toml:"accounts" yaml:"accounts"`
	AllowedChannelID     string                     `toml:"allowed_channel_id" yaml:"allowed_channel_id"`
	AllowedChannels      any                        `toml:"allowed_channels" yaml:"allowed_channels"`
	CoalesceMs           int                        `toml:"coalesce_ms" yaml:"coalesce_ms"`
	FirstRunDiscovery    bool                       `toml:"first_run_discovery" yaml:"first_run_discovery"`
	RequireMention       any                        `toml:"require_mention" yaml:"require_mention"`
	StrictMention        any                        `toml:"strict_mention" yaml:"strict_mention"`
	ReplyInThread        bool                       `toml:"reply_in_thread" yaml:"reply_in_thread"`
	FreeResponseChannels any                        `toml:"free_response_channels" yaml:"free_response_channels"`
	ChannelSkillBindings any                        `toml:"channel_skill_bindings" yaml:"channel_skill_bindings"`
	ChannelPrompts       any                        `toml:"channel_prompts" yaml:"channel_prompts"`
	AccountID            string                     `toml:"-" yaml:"-"`
}

func (c SlackCfg) AllowedChannelIDs() []string {
	return flexibleStringList(c.AllowedChannels)
}

type CronCfg struct {
	Enabled        bool          `toml:"enabled" yaml:"enabled"`
	CallTimeout    time.Duration `toml:"call_timeout" yaml:"call_timeout"`
	MirrorInterval time.Duration `toml:"mirror_interval" yaml:"mirror_interval"`
	MirrorPath     string        `toml:"mirror_path" yaml:"mirror_path"`
}

// ApprovalsCfg mirrors Hermes' approval policy settings that affect native
// Go tools.
type ApprovalsCfg struct {
	CronMode string `toml:"cron_mode" yaml:"cron_mode"`
}

// WebCfg mirrors Hermes config.yaml's web.backend and web.use_gateway fields.
type WebCfg struct {
	Backend    string `toml:"backend" yaml:"backend"`
	UseGateway bool   `toml:"use_gateway" yaml:"use_gateway"`
}

const (
	NavivoxDefaultBindHost = "127.0.0.1"
	NavivoxDefaultPort     = 8765

	NavivoxExposureLocal     = "local"
	NavivoxExposureTailscale = "tailscale"
	NavivoxExposureWireGuard = "wireguard"
	NavivoxExposureVPN       = "vpn"
	NavivoxExposurePublic    = "public"

	NavivoxAuthPairingToken              = "pairing_token"
	NavivoxAuthStaticToken               = "static_token"
	NavivoxAuthTailscaleIdentity         = "tailscale_identity"
	NavivoxAuthTokenAndTailscaleIdentity = "token_and_tailscale_identity"
)

// NavivoxCfg configures the native gateway-owned HTTP/WebSocket channel used
// by the Flutter Navivox app. The disabled zero value is intentionally safe.
type NavivoxCfg struct {
	Enabled                  bool                        `toml:"enabled" yaml:"enabled"`
	BindHost                 string                      `toml:"bind_host" yaml:"bind_host"`
	Port                     int                         `toml:"port" yaml:"port"`
	ExposureMode             string                      `toml:"exposure_mode" yaml:"exposure_mode"`
	AuthMode                 string                      `toml:"auth_mode" yaml:"auth_mode"`
	Token                    string                      `toml:"token" yaml:"token"`
	AllowOrigins             []string                    `toml:"allow_origins" yaml:"allow_origins"`
	AllowedTailnetIdentities []string                    `toml:"allowed_tailnet_identities" yaml:"allowed_tailnet_identities"`
	PublicConfirmed          bool                        `toml:"public_confirmed" yaml:"public_confirmed"`
	Servers                  map[string]NavivoxServerCfg `toml:"servers" yaml:"servers"`
}

type NavivoxServerCfg struct {
	Enabled      bool     `toml:"enabled" yaml:"enabled"`
	Bind         string   `toml:"bind" yaml:"bind"`
	Profiles     []string `toml:"profiles" yaml:"profiles"`
	Transports   []string `toml:"transports" yaml:"transports"`
	Capabilities []string `toml:"capabilities" yaml:"capabilities"`
}

// BrowserCfg mirrors Hermes browser/CDP connection settings used by browser
// tools and CDP-backed web_extract fallback.
type BrowserCfg struct {
	CDPURL string `toml:"cdp_url" yaml:"cdp_url"`
}

// SecurityCfg mirrors Hermes config.yaml security controls that affect native
// Go tools.
type SecurityCfg struct {
	WebsiteBlocklist WebsiteBlocklistCfg `toml:"website_blocklist" yaml:"website_blocklist"`
}

type WebsiteBlocklistCfg struct {
	Enabled     bool     `toml:"enabled" yaml:"enabled"`
	Domains     []string `toml:"domains" yaml:"domains"`
	SharedFiles []string `toml:"shared_files" yaml:"shared_files"`
	BaseDir     string   `toml:"-"`
}

// SkillsCfg configures the Phase 2.G0 static skills runtime.
type SkillsCfg struct {
	Root             string `toml:"root" yaml:"root"`
	SelectionCap     int    `toml:"selection_cap" yaml:"selection_cap"`
	MaxDocumentBytes int    `toml:"max_document_bytes" yaml:"max_document_bytes"`
	UsageLogPath     string `toml:"usage_log_path" yaml:"usage_log_path"`
}

// DelegationCfg configures Phase 2.E subagent execution.
type DelegationCfg struct {
	Enabled               bool          `toml:"enabled" yaml:"enabled"`
	MaxDepth              int           `toml:"max_depth" yaml:"max_depth"`
	MaxConcurrentChildren int           `toml:"max_concurrent_children" yaml:"max_concurrent_children"`
	DefaultMaxIterations  int           `toml:"default_max_iterations" yaml:"default_max_iterations"`
	DefaultTimeout        time.Duration `toml:"default_timeout" yaml:"default_timeout"`
	RunLogPath            string        `toml:"run_log_path" yaml:"run_log_path"`
	MaxWaiting            int           `toml:"max_waiting" yaml:"max_waiting"`
}

// GonchoCfg configures the in-process Honcho-compatible memory facade.
type GonchoCfg struct {
	Enabled                      bool   `toml:"enabled" yaml:"enabled"`
	Workspace                    string `toml:"workspace" yaml:"workspace"`
	ObserverPeer                 string `toml:"observer_peer" yaml:"observer_peer"`
	RecentMessages               int    `toml:"recent_messages" yaml:"recent_messages"`
	MaxMessageSize               int    `toml:"max_message_size" yaml:"max_message_size"`
	MaxFileSize                  int    `toml:"max_file_size" yaml:"max_file_size"`
	GetContextMaxTokens          int    `toml:"get_context_max_tokens" yaml:"get_context_max_tokens"`
	ReasoningEnabled             bool   `toml:"reasoning_enabled" yaml:"reasoning_enabled"`
	PeerCardEnabled              bool   `toml:"peer_card_enabled" yaml:"peer_card_enabled"`
	SummaryEnabled               bool   `toml:"summary_enabled" yaml:"summary_enabled"`
	DreamEnabled                 bool   `toml:"dream_enabled" yaml:"dream_enabled"`
	DreamIdleTimeoutMinutes      int    `toml:"dream_idle_timeout_minutes" yaml:"dream_idle_timeout_minutes"`
	DeriverWorkers               int    `toml:"deriver_workers" yaml:"deriver_workers"`
	RepresentationBatchMaxTokens int    `toml:"representation_batch_max_tokens" yaml:"representation_batch_max_tokens"`
	DialecticDefaultLevel        string `toml:"dialectic_default_level" yaml:"dialectic_default_level"`
}

func (g GonchoCfg) RuntimeConfig() goncho.Config {
	return goncho.Config{
		Enabled:                      g.Enabled,
		WorkspaceID:                  g.Workspace,
		ObserverPeerID:               g.ObserverPeer,
		RecentMessages:               g.RecentMessages,
		MaxMessageSize:               g.MaxMessageSize,
		MaxFileSize:                  g.MaxFileSize,
		GetContextMaxTokens:          g.GetContextMaxTokens,
		ReasoningEnabled:             g.ReasoningEnabled,
		PeerCardEnabled:              g.PeerCardEnabled,
		SummaryEnabled:               g.SummaryEnabled,
		DreamEnabled:                 g.DreamEnabled,
		DreamIdleTimeout:             time.Duration(g.DreamIdleTimeoutMinutes) * time.Minute,
		DeriverWorkers:               g.DeriverWorkers,
		RepresentationBatchMaxTokens: g.RepresentationBatchMaxTokens,
		DialecticDefaultLevel:        goncho.DialecticLevel(g.DialecticDefaultLevel),
	}
}

func (d *DelegationCfg) UnmarshalTOML(data []byte) error {
	type rawDelegationCfg struct {
		Enabled               bool   `toml:"enabled" yaml:"enabled"`
		MaxDepth              int    `toml:"max_depth" yaml:"max_depth"`
		MaxConcurrentChildren int    `toml:"max_concurrent_children" yaml:"max_concurrent_children"`
		DefaultMaxIterations  int    `toml:"default_max_iterations" yaml:"default_max_iterations"`
		DefaultTimeout        string `toml:"default_timeout" yaml:"default_timeout"`
		RunLogPath            string `toml:"run_log_path" yaml:"run_log_path"`
		MaxWaiting            int    `toml:"max_waiting" yaml:"max_waiting"`
	}

	var raw rawDelegationCfg
	if err := toml.Unmarshal(data, &raw); err != nil {
		return err
	}

	*d = DelegationCfg{
		Enabled:               raw.Enabled,
		MaxDepth:              raw.MaxDepth,
		MaxConcurrentChildren: raw.MaxConcurrentChildren,
		DefaultMaxIterations:  raw.DefaultMaxIterations,
		RunLogPath:            raw.RunLogPath,
		MaxWaiting:            raw.MaxWaiting,
	}
	if raw.DefaultTimeout == "" {
		return nil
	}

	dur, err := time.ParseDuration(raw.DefaultTimeout)
	if err != nil {
		return fmt.Errorf("delegation.default_timeout: %w", err)
	}
	d.DefaultTimeout = dur
	return nil
}

type HermesCfg struct {
	Endpoint              string     `toml:"endpoint" yaml:"endpoint"`
	APIKey                string     `toml:"api_key" yaml:"api_key"`
	APIKeyRef             *SecretRef `toml:"api_key_ref" yaml:"api_key_ref" json:"api_key_ref,omitempty"`
	Model                 string     `toml:"model" yaml:"model"`
	Provider              string     `toml:"provider" yaml:"provider"`
	ModelResolutionSource string     `toml:"-" yaml:"-" json:"model_resolution_source,omitempty"`
}

type AgentRuntimeCfg struct {
	ImageInputMode     string            `toml:"image_input_mode" yaml:"image_input_mode"`
	MaxTurns           int               `toml:"max_turns" yaml:"max_turns"`
	ReasoningEffort    string            `toml:"reasoning_effort" yaml:"reasoning_effort"`
	GatewayTimeout     int               `toml:"gateway_timeout" yaml:"gateway_timeout"`
	GatewayTimeoutWarn int               `toml:"gateway_timeout_warning" yaml:"gateway_timeout_warning"`
	APIMaxRetries      int               `toml:"api_max_retries" yaml:"api_max_retries"`
	Verbose            bool              `toml:"verbose" yaml:"verbose"`
	Personalities      map[string]string `toml:"personalities" yaml:"personalities"`
	ActivePersonality  string            `toml:"active_personality" yaml:"active_personality"`
}

type AuxiliaryCfg struct {
	Curator AuxiliaryTaskCfg `toml:"curator" yaml:"curator"`
	Vision  AuxiliaryTaskCfg `toml:"vision" yaml:"vision"`
}

type CuratorCfg struct {
	Auxiliary AuxiliaryTaskCfg `toml:"auxiliary" yaml:"auxiliary"`
}

type AuxiliaryTaskCfg struct {
	Provider  string         `toml:"provider" yaml:"provider"`
	Model     string         `toml:"model" yaml:"model"`
	BaseURL   string         `toml:"base_url" yaml:"base_url"`
	APIKey    string         `toml:"api_key" yaml:"api_key"`
	Timeout   int            `toml:"timeout" yaml:"timeout"`
	ExtraBody map[string]any `toml:"extra_body" yaml:"extra_body"`
}

type RuntimeCfg struct {
	MaxToolIterations         int     `toml:"max_tool_iterations" yaml:"max_tool_iterations"`
	TerminalBackend           string  `toml:"terminal_backend" yaml:"terminal_backend"`
	TTSProvider               string  `toml:"tts_provider" yaml:"tts_provider"`
	CompressionThreshold      float64 `toml:"compression_threshold" yaml:"compression_threshold"`
	SessionResetPolicy        string  `toml:"session_reset_policy" yaml:"session_reset_policy"`
	SessionResetAfterMinutes  int     `toml:"session_reset_after_minutes" yaml:"session_reset_after_minutes"`
	SessionResetDailyHour     int     `toml:"session_reset_daily_hour" yaml:"session_reset_daily_hour"`
	SessionResetMemorySummary bool    `toml:"session_reset_memory_summary" yaml:"session_reset_memory_summary"`
}

type TerminalCfg struct {
	Backend string `toml:"backend" yaml:"backend"`
	CWD     string `toml:"cwd" yaml:"cwd"`
}

// CodeExecutionCfg controls the native execute_code tool mode. Hermes defaults
// to project mode; Gormes keeps strict as the built-in default until the
// shell-only guard is intentionally relaxed by explicit config.
type CodeExecutionCfg struct {
	Mode string `toml:"mode" yaml:"mode"`
}

type GatewayCfg struct {
	ProxyURL  string                        `toml:"proxy_url" yaml:"proxy_url"`
	ProxyKey  string                        `toml:"proxy_key" yaml:"proxy_key"`
	Platforms map[string]GatewayPlatformCfg `toml:"platforms" yaml:"platforms"`
}

type GatewayPlatformCfg struct {
	GatewayRestartNotification *bool `toml:"gateway_restart_notification" yaml:"gateway_restart_notification"`
}

type DisplayCfg struct {
	Language                 string                        `toml:"language" yaml:"language"`
	Personality              string                        `toml:"personality" yaml:"personality"`
	ToolProgress             string                        `toml:"tool_progress" yaml:"tool_progress"`
	ToolProgressCommand      bool                          `toml:"tool_progress_command" yaml:"tool_progress_command"`
	ShowReasoning            bool                          `toml:"show_reasoning" yaml:"show_reasoning"`
	Streaming                bool                          `toml:"streaming" yaml:"streaming"`
	BellOnComplete           bool                          `toml:"bell_on_complete" yaml:"bell_on_complete"`
	Compact                  bool                          `toml:"compact" yaml:"compact"`
	CleanupProgress          bool                          `toml:"cleanup_progress" yaml:"cleanup_progress"`
	InterimAssistantMessages bool                          `toml:"interim_assistant_messages" yaml:"interim_assistant_messages"`
	BackgroundProcessNotifs  string                        `toml:"background_process_notifications" yaml:"background_process_notifications"`
	BusyInputMode            string                        `toml:"busy_input_mode" yaml:"busy_input_mode"`
	Platforms                map[string]DisplayPlatformCfg `toml:"platforms" yaml:"platforms"`
}

type DisplayPlatformCfg struct {
	ToolProgress string `toml:"tool_progress" yaml:"tool_progress"`
}

type TUICfg struct {
	Theme         string `toml:"theme" yaml:"theme"`
	MouseTracking bool   `toml:"mouse_tracking" yaml:"mouse_tracking"`
}

type InputCfg struct {
	MaxBytes int `toml:"max_bytes" yaml:"max_bytes"`
	MaxLines int `toml:"max_lines" yaml:"max_lines"`
}

type STTCfg struct {
	Enabled  bool           `toml:"enabled" yaml:"enabled"`
	Provider string         `toml:"provider" yaml:"provider"`
	Local    STTLocalCfg    `toml:"local" yaml:"local"`
	OpenAI   STTProviderCfg `toml:"openai" yaml:"openai"`
}

type STTLocalCfg struct {
	Model    string `toml:"model" yaml:"model"`
	Language string `toml:"language" yaml:"language"`
}

type STTProviderCfg struct {
	Model string `toml:"model" yaml:"model"`
}

type VoiceCfg struct {
	RecordKey string `toml:"record_key" yaml:"record_key"`
}

type InferenceValueSource string

const (
	InferenceValueSourceUnset  InferenceValueSource = "unset"
	InferenceValueSourceFlag   InferenceValueSource = "flag"
	InferenceValueSourceEnv    InferenceValueSource = "env"
	InferenceValueSourceConfig InferenceValueSource = "config"
)

type InferenceRequest struct {
	Config       Config
	ModelFlag    string
	ProviderFlag string
	LookupEnv    func(string) (string, bool)
	CommandLabel string
}

type OneshotInferenceRequest = InferenceRequest
type TUIInferenceRequest = InferenceRequest

type InferenceResolution struct {
	Model                      string
	ModelSource                InferenceValueSource
	Provider                   string
	ProviderSource             InferenceValueSource
	ProviderAutoDetectRequired bool
	ModelResolutionSource      string
}

type OneshotInferenceResolution = InferenceResolution
type TUIInferenceResolution = InferenceResolution

// ResolveOneshotInference applies the scripted-chat inference precedence:
// flag > GORMES_INFERENCE_* env > config defaults. A provider override without
// a flag/env model is rejected so a stale configured model is not silently
// paired with a different provider.
func ResolveOneshotInference(req OneshotInferenceRequest) (OneshotInferenceResolution, error) {
	req.CommandLabel = "gormes chat -q"
	return ResolveInference(req)
}

func ResolveTUIInference(req TUIInferenceRequest) (TUIInferenceResolution, error) {
	req.CommandLabel = "gormes tui"
	return ResolveInference(req)
}

func ResolveInference(req InferenceRequest) (InferenceResolution, error) {
	lookupEnv := req.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	model, modelSource := firstInferenceValue(
		inferenceCandidate{value: req.ModelFlag, source: InferenceValueSourceFlag},
		inferenceCandidate{value: lookupInferenceEnv(lookupEnv, "GORMES_INFERENCE_MODEL"), source: InferenceValueSourceEnv},
		inferenceCandidate{value: req.Config.Hermes.Model, source: InferenceValueSourceConfig},
	)
	provider, providerSource := firstInferenceValue(
		inferenceCandidate{value: req.ProviderFlag, source: InferenceValueSourceFlag},
		inferenceCandidate{value: lookupInferenceEnv(lookupEnv, "GORMES_INFERENCE_PROVIDER"), source: InferenceValueSourceEnv},
		inferenceCandidate{value: req.Config.Hermes.Provider, source: InferenceValueSourceConfig},
	)

	resolution := InferenceResolution{
		Model:          model,
		ModelSource:    modelSource,
		Provider:       provider,
		ProviderSource: providerSource,
	}
	explicitModel := modelSource == InferenceValueSourceFlag || modelSource == InferenceValueSourceEnv
	explicitProvider := providerSource == InferenceValueSourceFlag || providerSource == InferenceValueSourceEnv
	if explicitProvider && !explicitModel {
		return resolution, providerRequiresExplicitModelError(req.CommandLabel, providerSource)
	}
	resolution.ProviderAutoDetectRequired = explicitModel && providerSource == InferenceValueSourceUnset
	resolveInferenceProviderDefaultModel(&resolution)
	return resolution, nil
}

func resolveProviderDefaultModel(cfg *Config) {
	if !shouldResolveProviderDefaultModel(cfg.Hermes.Provider, cfg.Hermes.Model) {
		if strings.TrimSpace(cfg.Hermes.Provider) != "" && strings.TrimSpace(cfg.Hermes.Model) != "" {
			cfg.Hermes.ModelResolutionSource = "explicit_operator_config"
		}
		return
	}
	resolution := hermes.ResolveProviderDefaultModel(cfg.Hermes.Provider, hermes.ProviderDefaultModelOptions{})
	if strings.TrimSpace(resolution.Model) == "" {
		return
	}
	cfg.Hermes.Provider = resolution.Provider
	cfg.Hermes.Model = resolution.Model
	cfg.Hermes.ModelResolutionSource = string(resolution.Source)
}

func resolveInferenceProviderDefaultModel(resolution *InferenceResolution) {
	if resolution == nil || !shouldResolveProviderDefaultModel(resolution.Provider, resolution.Model) {
		return
	}
	defaultModel := hermes.ResolveProviderDefaultModel(resolution.Provider, hermes.ProviderDefaultModelOptions{})
	if strings.TrimSpace(defaultModel.Model) == "" {
		return
	}
	resolution.Provider = defaultModel.Provider
	resolution.Model = defaultModel.Model
	resolution.ModelResolutionSource = string(defaultModel.Source)
}

func shouldResolveProviderDefaultModel(provider, model string) bool {
	if strings.TrimSpace(provider) == "" {
		return false
	}
	model = strings.TrimSpace(model)
	return model == "" || strings.EqualFold(model, "hermes-agent")
}

type inferenceCandidate struct {
	value  string
	source InferenceValueSource
}

func firstInferenceValue(candidates ...inferenceCandidate) (string, InferenceValueSource) {
	for _, candidate := range candidates {
		value := strings.TrimSpace(candidate.value)
		if value != "" {
			return value, candidate.source
		}
	}
	return "", InferenceValueSourceUnset
}

func lookupInferenceEnv(lookup func(string) (string, bool), name string) string {
	value, ok := lookup(name)
	if !ok {
		return ""
	}
	return value
}

func providerRequiresExplicitModelError(commandLabel string, source InferenceValueSource) error {
	commandLabel = strings.TrimSpace(commandLabel)
	if commandLabel == "" {
		commandLabel = "gormes inference"
	}
	if source == InferenceValueSourceEnv {
		return fmt.Errorf("%s: GORMES_INFERENCE_PROVIDER requires --model or GORMES_INFERENCE_MODEL. Set both inference env vars, pass both flags, or neither to use your configured defaults", commandLabel)
	}
	return fmt.Errorf("%s: --provider requires --model (or GORMES_INFERENCE_MODEL). Pass both explicitly, or neither to use your configured defaults.", commandLabel)
}

// Load resolves configuration from (in precedence order) CLI flags, env vars,
// a TOML file at GormesHome()/config.toml, and built-in defaults.
// Pass os.Args[1:] as args; pass nil to skip flag parsing entirely (useful in tests).
//
// Before anything else, the dotenv file at GormesHome()/.env is read into the
// process environment. Any key NOT already in the shell env is set from the
// file.
func Load(args []string) (Config, error) {
	loadDotenvFiles() // populates os.Setenv for unset keys BEFORE loadEnv reads them
	cfg := defaults()
	if err := loadFile(&cfg); err != nil {
		return cfg, err
	}
	if err := loadEnv(&cfg); err != nil {
		return cfg, err
	}
	if err := loadFlags(&cfg, args); err != nil {
		return cfg, err
	}
	if err := validateConfig(&cfg); err != nil {
		return cfg, err
	}
	resolveProviderDefaultModel(&cfg)
	return cfg, nil
}

func defaultPersonalities() map[string]string {
	return map[string]string{
		"helpful":     "You are a helpful, friendly AI assistant.",
		"concise":     "You are a concise assistant. Keep responses brief and to the point.",
		"technical":   "You are a technical expert. Provide detailed, accurate technical information.",
		"creative":    "You are a creative assistant. Think outside the box and offer innovative solutions.",
		"teacher":     "You are a patient teacher. Explain concepts clearly with examples.",
		"kawaii":      "You are a kawaii assistant! Use cute expressions like (◕‿◕), ★, ♪, and ~! Add sparkles and be super enthusiastic about everything! Every response should feel warm and adorable desu~! ヽ(>∀<☆)ノ",
		"catgirl":     "You are Neko-chan, an anime catgirl AI assistant, nya~! Add 'nya' and cat-like expressions to your speech. Use kaomoji like (=^･ω･^=) and ฅ^•ﻌ•^ฅ. Be playful and curious like a cat, nya~!",
		"noir":        "The rain hammered against the terminal like regrets on a guilty conscience. They call me Gormes - I solve problems, find answers, dig up the truth that hides in the shadows of your codebase. In this city of silicon and secrets, everyone's got something to hide. What's your story, pal?",
		"pirate":      "Arrr! Ye be talkin' to Captain Gormes, the most tech-savvy pirate to sail the digital seas! Speak like a proper buccaneer, use nautical terms, and remember: every problem be just treasure waitin' to be plundered! Yo ho ho!",
		"philosopher": "Greetings, seeker of wisdom. I am an assistant who contemplates the deeper meaning behind every query. Let us examine not just the 'how' but the 'why' of your questions.",
		"hype":        "YOOO LET'S GOOOO!!! 🔥🔥🔥 I am SO PUMPED to help you today! Every question is AMAZING and we're gonna CRUSH IT together! This is gonna be LEGENDARY! ARE YOU READY?! LET'S DO THIS! 💪😤🚀",
		"shakespeare": "Hark! Thou speakest with an assistant most versed in the bardic arts. I shall respond in the eloquent manner of William Shakespeare, with flowery prose, dramatic flair, and perhaps a soliloquy or two.",
	}
}

func defaults() Config {
	return Config{
		ConfigVersion: CurrentConfigVersion,
		Hermes: HermesCfg{
			Model: "hermes-agent",
		},
		Agent: AgentRuntimeCfg{
			MaxTurns:        60,
			ReasoningEffort: "medium",
			GatewayTimeout:  1800,
			APIMaxRetries:   3,
			Personalities:   defaultPersonalities(),
		},
		Runtime: RuntimeCfg{
			MaxToolIterations:         90,
			TerminalBackend:           "local",
			TTSProvider:               "edge",
			CompressionThreshold:      0.5,
			SessionResetPolicy:        "inactivity",
			SessionResetAfterMinutes:  1440,
			SessionResetDailyHour:     4,
			SessionResetMemorySummary: true,
		},
		TTS: map[string]any{},
		Terminal: TerminalCfg{
			CWD: ".",
		},
		CodeExecution: CodeExecutionCfg{
			Mode: "strict",
		},
		TUI:   TUICfg{Theme: "dark", MouseTracking: false},
		Input: InputCfg{MaxBytes: 200_000, MaxLines: 10_000},
		Voice: VoiceCfg{RecordKey: "ctrl+b"},
		Auxiliary: AuxiliaryCfg{
			Curator: AuxiliaryTaskCfg{
				Provider:  "auto",
				Timeout:   600,
				ExtraBody: map[string]any{},
			},
		},
		Telegram: TelegramCfg{
			CoalesceMs:             1000,
			FreshFinalAfterSeconds: 60.0,
			Notifications:          "important",
			FirstRunDiscovery:      true,
			MemoryQueueCap:         1024,
			ExtractorBatchSize:     5,
			ExtractorPollInterval:  10 * time.Second,
			RecallEnabled:          true,
			RecallWeightThreshold:  1.0,
			RecallMaxFacts:         10,
			RecallDepth:            2,
			RecallDecayHorizonDays: 180,
			MirrorEnabled:          true,
			MirrorPath:             filepath.Join(GormesHome(), "memory", "USER.md"),
			MirrorInterval:         30 * time.Second,
			SemanticEnabled:        false,
			SemanticEndpoint:       "",
			SemanticModel:          "",
			SemanticTopK:           3,
			SemanticMinSimilarity:  0.35,
			EmbedderPollInterval:   30 * time.Second,
			EmbedderBatchSize:      10,
			EmbedderCallTimeout:    10 * time.Second,
			QueryEmbedTimeout:      60 * time.Millisecond,
		},
		Discord: DiscordCfg{
			CoalesceMs:        1000,
			FirstRunDiscovery: false,
		},
		Slack: SlackCfg{
			Enabled:           false,
			CoalesceMs:        1000,
			FirstRunDiscovery: false,
			RequireMention:    true,
			ReplyInThread:     true,
		},
		Navivox: NavivoxCfg{
			Enabled:      false,
			BindHost:     NavivoxDefaultBindHost,
			Port:         NavivoxDefaultPort,
			ExposureMode: NavivoxExposureLocal,
			AuthMode:     NavivoxAuthPairingToken,
		},
		Teams: TeamsCfg{
			Enabled: false,
			Port:    TeamsDefaultPort,
		},
		Security: SecurityCfg{
			WebsiteBlocklist: WebsiteBlocklistCfg{
				BaseDir: GormesHome(),
			},
		},
		Secrets: SecretsCfg{
			Defaults: SecretProviderDefaults{Env: DefaultSecretProviderAlias},
		},
		Cron: CronCfg{
			Enabled:        false,
			CallTimeout:    60 * time.Second,
			MirrorInterval: 30 * time.Second,
			MirrorPath:     "",
		},
		Approvals: ApprovalsCfg{
			CronMode: "deny",
		},
		Skills: SkillsCfg{
			SelectionCap:     3,
			MaxDocumentBytes: 64 * 1024,
			UsageLogPath:     "",
		},
		Delegation: DelegationCfg{
			Enabled:               false,
			MaxDepth:              2,
			MaxConcurrentChildren: 3,
			DefaultMaxIterations:  8,
			DefaultTimeout:        45 * time.Second,
			RunLogPath:            "",
			MaxWaiting:            128,
		},
		Updates: UpdatesCfg{
			PreUpdateBackup: false,
			BackupKeep:      5,
		},
		Goncho: GonchoCfg{
			Enabled:                      true,
			Workspace:                    goncho.DefaultWorkspaceID,
			ObserverPeer:                 goncho.DefaultObserverPeerID,
			RecentMessages:               goncho.DefaultRecentMessages,
			MaxMessageSize:               goncho.DefaultMaxMessageSize,
			MaxFileSize:                  goncho.DefaultMaxFileSize,
			GetContextMaxTokens:          goncho.DefaultGetContextMaxTokens,
			ReasoningEnabled:             true,
			PeerCardEnabled:              true,
			SummaryEnabled:               true,
			DreamEnabled:                 false,
			DreamIdleTimeoutMinutes:      int(goncho.DefaultDreamIdleTimeout / time.Minute),
			DeriverWorkers:               goncho.DefaultDeriverWorkers,
			RepresentationBatchMaxTokens: goncho.DefaultRepresentationBatchMaxTokens,
			DialecticDefaultLevel:        string(goncho.DialecticLevelLow),
		},
	}
}

func loadFile(cfg *Config) error {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Try YAML fallback for Hermes migrants
		yamlPath := YAMLConfigPath()
		yamlData, yamlErr := os.ReadFile(yamlPath)
		if os.IsNotExist(yamlErr) {
			return nil // no config at all
		}
		if yamlErr != nil {
			return fmt.Errorf("read %s: %w", yamlPath, yamlErr)
		}
		cfg.ConfigVersion = 0
		cfg.LegacyConfigVersion = 0
		if err := yaml.NewDecoder(bytes.NewReader(yamlData)).Decode(cfg); err != nil {
			return fmt.Errorf("decode %s: %w", yamlPath, err)
		}
		normalizeDecodedConfigVersion(cfg)
		return migrateConfig(cfg)
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	cfg.ConfigVersion = 0
	cfg.LegacyConfigVersion = 0
	if err := toml.NewDecoder(bytes.NewReader(data)).EnableUnmarshalerInterface().Decode(cfg); err != nil {
		return err
	}
	normalizeDecodedConfigVersion(cfg)
	return migrateConfig(cfg)
}

type hermesConfigYAML struct {
	Model     hermesModelConfigYAML               `yaml:"model"`
	Web       hermesWebConfigYAML                 `yaml:"web"`
	Browser   hermesBrowserConfigYAML             `yaml:"browser"`
	Display   hermesDisplayConfigYAML             `yaml:"display"`
	Security  hermesSecurityConfigYAML            `yaml:"security"`
	Platforms map[string]hermesPlatformConfigYAML `yaml:"platforms"`
	Streaming hermesStreamingConfigYAML           `yaml:"streaming"`
}

type hermesModelConfigYAML struct {
	Default  string `yaml:"default"`
	Provider string `yaml:"provider"`
}

type hermesWebConfigYAML struct {
	Backend    string `yaml:"backend"`
	UseGateway bool   `yaml:"use_gateway"`
}

type hermesBrowserConfigYAML struct {
	CDPURL string `yaml:"cdp_url"`
}

type hermesDisplayConfigYAML struct {
	ToolProgress          interface{}                          `yaml:"tool_progress"`
	ToolProgressCommand   bool                                 `yaml:"tool_progress_command"`
	Platforms             map[string]hermesDisplayPlatformYAML `yaml:"platforms"`
	ToolProgressOverrides map[string]interface{}               `yaml:"tool_progress_overrides"`
}

type hermesDisplayPlatformYAML struct {
	ToolProgress interface{} `yaml:"tool_progress"`
}

type hermesSecurityConfigYAML struct {
	WebsiteBlocklist hermesWebsiteBlocklistYAML `yaml:"website_blocklist"`
}

type hermesWebsiteBlocklistYAML struct {
	Enabled     bool     `yaml:"enabled"`
	Domains     []string `yaml:"domains"`
	SharedFiles []string `yaml:"shared_files"`
}

type hermesPlatformConfigYAML struct {
	Enabled     bool                   `yaml:"enabled"`
	Token       string                 `yaml:"token"`
	APIKey      string                 `yaml:"api_key"`
	HomeChannel hermesHomeChannelYAML  `yaml:"home_channel"`
	Extra       map[string]interface{} `yaml:"extra"`
}

type hermesHomeChannelYAML struct {
	ChatID interface{} `yaml:"chat_id"`
}

type hermesStreamingConfigYAML struct {
	FreshFinalAfterSeconds *float64 `yaml:"fresh_final_after_seconds"`
}

func normalizeDisplayPlatformKey(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func normalizeGatewayPlatformKey(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func normalizeGatewayPlatformMap(in map[string]GatewayPlatformCfg) map[string]GatewayPlatformCfg {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]GatewayPlatformCfg, len(in))
	for platform, cfg := range in {
		key := normalizeGatewayPlatformKey(platform)
		if key == "" {
			continue
		}
		out[key] = cfg
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c Config) GatewayRestartNotificationEnabled(platform string) bool {
	key := normalizeGatewayPlatformKey(platform)
	if key == "" || len(c.Gateway.Platforms) == 0 {
		return true
	}
	cfg, ok := c.Gateway.Platforms[key]
	if !ok || cfg.GatewayRestartNotification == nil {
		return true
	}
	return *cfg.GatewayRestartNotification
}

func (c Config) GatewayRestartNotifications() map[string]bool {
	if len(c.Gateway.Platforms) == 0 {
		return nil
	}
	out := make(map[string]bool)
	for platform, cfg := range c.Gateway.Platforms {
		if cfg.GatewayRestartNotification == nil {
			continue
		}
		key := normalizeGatewayPlatformKey(platform)
		if key == "" {
			continue
		}
		out[key] = *cfg.GatewayRestartNotification
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeHermesToolProgressMode(raw interface{}) (string, bool) {
	switch v := raw.(type) {
	case nil:
		return "", false
	case bool:
		if v {
			return "all", true
		}
		return "off", true
	case string:
		mode := strings.ToLower(strings.TrimSpace(v))
		if mode == "" {
			return "", false
		}
		switch mode {
		case "off", "new", "all", "verbose":
			return mode, true
		default:
			return "all", true
		}
	default:
		mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		if mode == "" {
			return "", false
		}
		switch mode {
		case "off", "new", "all", "verbose":
			return mode, true
		default:
			return "all", true
		}
	}
}

func normalizeTelegramNotifications(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "all":
		return "all"
	default:
		return "important"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// migrateConfig applies version-gated schema migrations in sequence,
// bumping cfg.ConfigVersion after each step. A config written by a
// newer binary (version > CurrentConfigVersion) is rejected with a
// clear error so operators know to upgrade — silently downgrading
// would quietly drop unknown fields.
func migrateConfig(cfg *Config) error {
	if cfg.ConfigVersion > CurrentConfigVersion {
		return fmt.Errorf(
			"config: config_version=%d is from a newer binary (this binary knows up to v%d); upgrade gormes or hand-edit the file",
			cfg.ConfigVersion, CurrentConfigVersion)
	}
	// v1 remains read-compatible in memory. On-disk migration to v2 is owned
	// by MigrateConfigFile so loading old configs never writes or implicitly
	// creates canonical profile tables.
	cfg.ConfigVersion = CurrentConfigVersion
	cfg.LegacyConfigVersion = 0
	return nil
}

func normalizeDecodedConfigVersion(cfg *Config) {
	if cfg.ConfigVersion == 0 {
		cfg.ConfigVersion = cfg.LegacyConfigVersion
	}
	if cfg.ConfigVersion == 0 {
		cfg.ConfigVersion = 1
	}
}

func loadEnv(cfg *Config) error {
	if v := os.Getenv("GORMES_ENDPOINT"); v != "" {
		cfg.Hermes.Endpoint = v
	}
	if v := os.Getenv("GORMES_MODEL"); v != "" {
		cfg.Hermes.Model = v
	}
	if v := os.Getenv("GORMES_API_KEY"); v != "" {
		cfg.Hermes.APIKey = v
	}
	if v := strings.TrimSpace(os.Getenv("GATEWAY_PROXY_URL")); v != "" {
		cfg.Gateway.ProxyURL = v
	}
	if v := strings.TrimSpace(os.Getenv("GATEWAY_PROXY_KEY")); v != "" {
		cfg.Gateway.ProxyKey = v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_WEB_BACKEND")); v != "" {
		cfg.Web.Backend = strings.ToLower(v)
	}
	if v := os.Getenv("GORMES_WEB_USE_GATEWAY"); v != "" {
		parsed, err := parseEnvBool("GORMES_WEB_USE_GATEWAY", v)
		if err != nil {
			return err
		}
		cfg.Web.UseGateway = parsed
	}
	if v := strings.TrimSpace(firstNonEmpty(os.Getenv("GORMES_BROWSER_CDP_URL"), os.Getenv("BROWSER_CDP_URL"), os.Getenv("CHROME_REMOTE_DEBUGGING_URL"))); v != "" {
		cfg.Browser.CDPURL = v
	}
	if v := os.Getenv("GORMES_TUI_MOUSE_TRACKING"); v != "" {
		parsed, err := parseEnvBool("GORMES_TUI_MOUSE_TRACKING", v)
		if err != nil {
			return err
		}
		cfg.TUI.MouseTracking = parsed
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_VOICE_RECORD_KEY")); v != "" {
		cfg.Voice.RecordKey = v
	}
	if v := firstNonEmpty(
		os.Getenv("GORMES_TELEGRAM_BOT_TOKEN"),
		os.Getenv("GORMES_TELEGRAM_TOKEN"),
		os.Getenv("HERMES_TELEGRAM_BOT_TOKEN"),
		os.Getenv("HERMES_TELEGRAM_TOKEN"),
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("TELEGRAM_TOKEN"),
	); v != "" {
		cfg.Telegram.BotToken = v
	}
	if v := firstNonEmpty(
		os.Getenv("GORMES_TELEGRAM_HOME_CHANNEL"),
		os.Getenv("GORMES_TELEGRAM_CHAT_ID"),
		os.Getenv("HERMES_TELEGRAM_HOME_CHANNEL"),
		os.Getenv("HERMES_TELEGRAM_CHAT_ID"),
		os.Getenv("TELEGRAM_HOME_CHANNEL"),
		os.Getenv("TELEGRAM_CHAT_ID"),
	); v != "" {
		applyTelegramHomeChannel(cfg, v)
	}
	if v := firstNonEmpty(
		os.Getenv("GORMES_TELEGRAM_HOME_CHANNEL_NAME"),
		os.Getenv("HERMES_TELEGRAM_HOME_CHANNEL_NAME"),
		os.Getenv("TELEGRAM_HOME_CHANNEL_NAME"),
	); v != "" {
		cfg.Telegram.HomeChannel.Name = v
	}
	if v := firstNonEmpty(
		os.Getenv("GORMES_TELEGRAM_HOME_CHANNEL_THREAD_ID"),
		os.Getenv("HERMES_TELEGRAM_HOME_CHANNEL_THREAD_ID"),
		os.Getenv("TELEGRAM_HOME_CHANNEL_THREAD_ID"),
	); v != "" {
		cfg.Telegram.HomeChannel.ThreadID = v
	}
	if v := firstNonEmpty(os.Getenv("GORMES_TELEGRAM_ALLOWED_USERS"), os.Getenv("HERMES_TELEGRAM_ALLOWED_USERS"), os.Getenv("TELEGRAM_ALLOWED_USERS")); v != "" {
		cfg.Telegram.AllowedUserIDs = parseEnvInt64CSV(v)
	}
	if v := firstNonEmpty(os.Getenv("GORMES_TELEGRAM_ALLOWED_CHATS"), os.Getenv("HERMES_TELEGRAM_ALLOWED_CHATS"), os.Getenv("TELEGRAM_ALLOWED_CHATS")); v != "" {
		cfg.Telegram.AllowedChats = parseEnvCSV(v)
	}
	if v := firstNonEmpty(os.Getenv("GORMES_TELEGRAM_GUEST_MODE"), os.Getenv("HERMES_TELEGRAM_GUEST_MODE"), os.Getenv("TELEGRAM_GUEST_MODE")); v != "" {
		parsed, err := parseEnvBool("TELEGRAM_GUEST_MODE", v)
		if err != nil {
			return err
		}
		cfg.Telegram.GuestMode = parsed
	}
	if v := firstNonEmpty(os.Getenv("GORMES_TELEGRAM_NOTIFICATIONS"), os.Getenv("HERMES_TELEGRAM_NOTIFICATIONS"), os.Getenv("TELEGRAM_NOTIFICATIONS")); v != "" {
		cfg.Telegram.Notifications = v
	}
	if v := os.Getenv("GORMES_DISCORD_TOKEN"); v != "" {
		cfg.Discord.Token = v
	}
	if v := os.Getenv("GORMES_DISCORD_CHANNEL_ID"); v != "" {
		cfg.Discord.AllowedChannelID = v
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_ALLOWED_CHANNELS"), os.Getenv("DISCORD_ALLOWED_CHANNELS")); v != "" {
		cfg.Discord.AllowedChannels = parseEnvCSV(v)
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_IGNORED_CHANNELS"), os.Getenv("DISCORD_IGNORED_CHANNELS")); v != "" {
		cfg.Discord.IgnoredChannels = parseEnvCSV(v)
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_FREE_RESPONSE_CHANNELS"), os.Getenv("DISCORD_FREE_RESPONSE_CHANNELS")); v != "" {
		cfg.Discord.FreeResponseChannels = parseEnvCSV(v)
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_NO_THREAD_CHANNELS"), os.Getenv("DISCORD_NO_THREAD_CHANNELS")); v != "" {
		cfg.Discord.NoThreadChannels = parseEnvCSV(v)
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_REQUIRE_MENTION"), os.Getenv("DISCORD_REQUIRE_MENTION")); v != "" {
		parsed, err := parseEnvBool("DISCORD_REQUIRE_MENTION", v)
		if err != nil {
			return err
		}
		cfg.Discord.RequireMention = parsed
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_AUTO_THREAD"), os.Getenv("DISCORD_AUTO_THREAD")); v != "" {
		parsed, err := parseEnvBool("DISCORD_AUTO_THREAD", v)
		if err != nil {
			return err
		}
		cfg.Discord.AutoThread = parsed
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_REPLY_TO_MODE"), os.Getenv("DISCORD_REPLY_TO_MODE")); v != "" {
		cfg.Discord.ReplyToMode = v
	}
	if v := firstNonEmpty(os.Getenv("GORMES_DISCORD_ALLOW_BOTS"), os.Getenv("DISCORD_ALLOW_BOTS")); v != "" {
		cfg.Discord.AllowBots = v
	}
	if v := os.Getenv("GORMES_DISCORD_SERVER_ACTIONS"); v != "" {
		cfg.Discord.ServerActions = parseEnvCSV(v)
	}
	if v := os.Getenv("GORMES_SLACK_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_SLACK_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Slack.Enabled = parsed
	}
	if v := os.Getenv("GORMES_SLACK_BOT_TOKEN"); v != "" {
		cfg.Slack.BotToken = v
	}
	if v := os.Getenv("GORMES_SLACK_APP_TOKEN"); v != "" {
		cfg.Slack.AppToken = v
	}
	if v := os.Getenv("GORMES_SLACK_CHANNEL_ID"); v != "" {
		cfg.Slack.AllowedChannelID = v
	}
	if v := firstNonEmpty(os.Getenv("GORMES_SLACK_ALLOWED_CHANNELS"), os.Getenv("SLACK_ALLOWED_CHANNELS")); v != "" {
		cfg.Slack.AllowedChannels = parseEnvCSV(v)
	}
	if v := os.Getenv("GORMES_SLACK_COALESCE_MS"); v != "" {
		parsed, err := parseEnvInt("GORMES_SLACK_COALESCE_MS", v)
		if err != nil {
			return err
		}
		cfg.Slack.CoalesceMs = parsed
	}
	if v := os.Getenv("GORMES_SLACK_FIRST_RUN_DISCOVERY"); v != "" {
		parsed, err := parseEnvBool("GORMES_SLACK_FIRST_RUN_DISCOVERY", v)
		if err != nil {
			return err
		}
		cfg.Slack.FirstRunDiscovery = parsed
	}
	if v := os.Getenv("GORMES_SLACK_REQUIRE_MENTION"); v != "" {
		cfg.Slack.RequireMention = v
	}
	if v := os.Getenv("GORMES_SLACK_STRICT_MENTION"); v != "" {
		cfg.Slack.StrictMention = v
	}
	if v := os.Getenv("GORMES_SLACK_FREE_RESPONSE_CHANNELS"); v != "" {
		cfg.Slack.FreeResponseChannels = v
	}
	if v := os.Getenv("GORMES_SLACK_REPLY_IN_THREAD"); v != "" {
		parsed, err := parseEnvBool("GORMES_SLACK_REPLY_IN_THREAD", v)
		if err != nil {
			return err
		}
		cfg.Slack.ReplyInThread = parsed
	}
	if v := os.Getenv("GORMES_NAVIVOX_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_NAVIVOX_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Navivox.Enabled = parsed
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_BIND_HOST")); v != "" {
		cfg.Navivox.BindHost = v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_PORT")); v != "" {
		parsed, err := parseEnvInt("GORMES_NAVIVOX_PORT", v)
		if err != nil {
			return err
		}
		cfg.Navivox.Port = parsed
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_EXPOSURE_MODE")); v != "" {
		cfg.Navivox.ExposureMode = v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_AUTH_MODE")); v != "" {
		cfg.Navivox.AuthMode = v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_TOKEN")); v != "" {
		cfg.Navivox.Token = v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_ALLOW_ORIGINS")); v != "" {
		cfg.Navivox.AllowOrigins = parseEnvCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_NAVIVOX_ALLOWED_TAILNET_IDENTITIES")); v != "" {
		cfg.Navivox.AllowedTailnetIdentities = parseEnvCSV(v)
	}
	if v := os.Getenv("GORMES_NAVIVOX_PUBLIC_CONFIRMED"); v != "" {
		parsed, err := parseEnvBool("GORMES_NAVIVOX_PUBLIC_CONFIRMED", v)
		if err != nil {
			return err
		}
		cfg.Navivox.PublicConfirmed = parsed
	}
	if v := os.Getenv("GORMES_TEAMS_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_TEAMS_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Teams.Enabled = parsed
	}
	if v := os.Getenv("TEAMS_CLIENT_ID"); v != "" {
		cfg.Teams.ClientID = v
	}
	if v := os.Getenv("TEAMS_CLIENT_SECRET"); v != "" {
		cfg.Teams.ClientSecret = v
	}
	if v := os.Getenv("TEAMS_TENANT_ID"); v != "" {
		cfg.Teams.TenantID = v
	}
	if v := os.Getenv("TEAMS_PORT"); v != "" {
		parsed, err := parseEnvInt("TEAMS_PORT", v)
		if err != nil {
			return err
		}
		cfg.Teams.Port = parsed
	}
	if v := os.Getenv("TEAMS_ALLOWED_USERS"); v != "" {
		cfg.Teams.AllowedUsers = parseEnvCSV(v)
	}
	if v := os.Getenv("TEAMS_ALLOW_ALL_USERS"); v != "" {
		parsed, err := parseEnvBool("TEAMS_ALLOW_ALL_USERS", v)
		if err != nil {
			return err
		}
		cfg.Teams.AllowAllUsers = parsed
	}
	if v := os.Getenv("GORMES_SKILLS_ROOT"); v != "" {
		cfg.Skills.Root = v
	}
	if v := os.Getenv("GORMES_GONCHO_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_GONCHO_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Goncho.Enabled = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_WORKSPACE"); v != "" {
		cfg.Goncho.Workspace = v
	}
	if v := os.Getenv("GORMES_GONCHO_OBSERVER_PEER"); v != "" {
		cfg.Goncho.ObserverPeer = v
	}
	if v := os.Getenv("GORMES_GONCHO_RECENT_MESSAGES"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_RECENT_MESSAGES", v)
		if err != nil {
			return err
		}
		cfg.Goncho.RecentMessages = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_MAX_MESSAGE_SIZE"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_MAX_MESSAGE_SIZE", v)
		if err != nil {
			return err
		}
		cfg.Goncho.MaxMessageSize = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_MAX_FILE_SIZE"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_MAX_FILE_SIZE", v)
		if err != nil {
			return err
		}
		cfg.Goncho.MaxFileSize = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_GET_CONTEXT_MAX_TOKENS"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_GET_CONTEXT_MAX_TOKENS", v)
		if err != nil {
			return err
		}
		cfg.Goncho.GetContextMaxTokens = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_REASONING_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_GONCHO_REASONING_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Goncho.ReasoningEnabled = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_PEER_CARD_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_GONCHO_PEER_CARD_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Goncho.PeerCardEnabled = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_SUMMARY_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_GONCHO_SUMMARY_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Goncho.SummaryEnabled = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_DREAM_ENABLED"); v != "" {
		parsed, err := parseEnvBool("GORMES_GONCHO_DREAM_ENABLED", v)
		if err != nil {
			return err
		}
		cfg.Goncho.DreamEnabled = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_DREAM_IDLE_TIMEOUT_MINUTES"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_DREAM_IDLE_TIMEOUT_MINUTES", v)
		if err != nil {
			return err
		}
		cfg.Goncho.DreamIdleTimeoutMinutes = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_DERIVER_WORKERS"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_DERIVER_WORKERS", v)
		if err != nil {
			return err
		}
		cfg.Goncho.DeriverWorkers = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_REPRESENTATION_BATCH_MAX_TOKENS"); v != "" {
		parsed, err := parseEnvInt("GORMES_GONCHO_REPRESENTATION_BATCH_MAX_TOKENS", v)
		if err != nil {
			return err
		}
		cfg.Goncho.RepresentationBatchMaxTokens = parsed
	}
	if v := os.Getenv("GORMES_GONCHO_DIALECTIC_DEFAULT_LEVEL"); v != "" {
		cfg.Goncho.DialecticDefaultLevel = v
	}
	return nil
}

func parseEnvBool(name, value string) (bool, error) {
	text := strings.ToLower(strings.TrimSpace(value))
	switch text {
	case "yes", "on":
		return true, nil
	case "no", "off":
		return false, nil
	}
	parsed, err := strconv.ParseBool(text)
	if err != nil {
		return false, fmt.Errorf("config env %s: %w", name, err)
	}
	return parsed, nil
}

func parseEnvInt(name, value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("config env %s: %w", name, err)
	}
	return parsed, nil
}

func parseEnvCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func flexibleStringList(raw any) []string {
	switch v := raw.(type) {
	case nil:
		return nil
	case []string:
		return compactStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, flexibleStringList(item)...)
		}
		return compactStrings(out)
	case string:
		return parseEnvCSV(v)
	default:
		return parseEnvCSV(fmt.Sprint(v))
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func flexibleBool(raw any, defaultValue bool) bool {
	switch v := raw.(type) {
	case nil:
		return defaultValue
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "false", "0", "no", "off":
			return false
		case "true", "1", "yes", "on":
			return true
		default:
			return defaultValue
		}
	default:
		switch strings.ToLower(strings.TrimSpace(fmt.Sprint(v))) {
		case "false", "0", "no", "off":
			return false
		case "true", "1", "yes", "on":
			return true
		default:
			return defaultValue
		}
	}
}

func parseEnvInt64CSV(value string) []int64 {
	parts := parseEnvCSV(value)
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, parsed)
	}
	return out
}

func loadFlags(cfg *Config, args []string) error {
	if args == nil {
		return nil
	}
	fs := pflag.NewFlagSet("gormes", pflag.ContinueOnError)
	endpoint := fs.String("endpoint", "", "Hermes api_server base URL")
	model := fs.String("model", "", "served model name")
	resume := fs.String("resume", "", "override persisted session_id for this binary's default key")
	// No --api-key flag — secrets stay out of process argv.
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *endpoint != "" {
		cfg.Hermes.Endpoint = *endpoint
	}
	if *model != "" {
		cfg.Hermes.Model = *model
	}
	if *resume != "" {
		cfg.Resume = *resume
	}
	return nil
}

func validateConfig(cfg *Config) error {
	cfg.Gateway.ProxyURL = normalizeGatewayProxyURL(cfg.Gateway.ProxyURL)
	cfg.Gateway.ProxyKey = strings.TrimSpace(cfg.Gateway.ProxyKey)
	cfg.Gateway.Platforms = normalizeGatewayPlatformMap(cfg.Gateway.Platforms)
	if err := normalizeNavivoxConfig(&cfg.Navivox); err != nil {
		return err
	}
	cfg.Agent.ImageInputMode = normalizeAgentImageInputMode(cfg.Agent.ImageInputMode)
	normalizeAuxiliaryTask(&cfg.Auxiliary.Curator, true)
	normalizeAuxiliaryTask(&cfg.Auxiliary.Vision, false)
	normalizeAuxiliaryTask(&cfg.Curator.Auxiliary, false)
	cfg.Terminal.Backend = strings.ToLower(strings.TrimSpace(firstNonEmpty(cfg.Terminal.Backend, cfg.Runtime.TerminalBackend, "local")))
	cfg.Terminal.CWD = strings.TrimSpace(cfg.Terminal.CWD)
	if cfg.Terminal.CWD == "" {
		cfg.Terminal.CWD = "."
	}
	cfg.Voice.RecordKey = strings.TrimSpace(cfg.Voice.RecordKey)
	if cfg.Voice.RecordKey == "" {
		cfg.Voice.RecordKey = "ctrl+b"
	}
	normalizeTelegramConfig(&cfg.Telegram)
	cfg.Telegram.Notifications = normalizeTelegramNotifications(cfg.Telegram.Notifications)
	if strings.TrimSpace(cfg.Runtime.TerminalBackend) == "" {
		cfg.Runtime.TerminalBackend = cfg.Terminal.Backend
	}
	cfg.Slack.BotToken = strings.TrimSpace(cfg.Slack.BotToken)
	cfg.Slack.AppToken = strings.TrimSpace(cfg.Slack.AppToken)
	cfg.Slack.AllowedChannelID = strings.TrimSpace(cfg.Slack.AllowedChannelID)
	cfg.Teams.ClientID = strings.TrimSpace(cfg.Teams.ClientID)
	cfg.Teams.ClientSecret = strings.TrimSpace(cfg.Teams.ClientSecret)
	cfg.Teams.TenantID = strings.TrimSpace(cfg.Teams.TenantID)
	cfg.Teams.AllowedUsers = compactStrings(cfg.Teams.AllowedUsers)
	if cfg.Teams.Port <= 0 {
		cfg.Teams.Port = TeamsDefaultPort
	}
	cfg.Bindings = normalizeAgentBindings(cfg.Bindings)
	if err := normalizeAgentsConfig(GormesHome(), &cfg.Agents, cfg.Bindings); err != nil {
		return err
	}
	if err := normalizeProfileConfigV2(cfg); err != nil {
		return err
	}
	cfg.Goncho.Workspace = strings.TrimSpace(cfg.Goncho.Workspace)
	cfg.Goncho.ObserverPeer = strings.TrimSpace(cfg.Goncho.ObserverPeer)
	cfg.Goncho.DialecticDefaultLevel = strings.ToLower(strings.TrimSpace(cfg.Goncho.DialecticDefaultLevel))

	if cfg.Goncho.Workspace == "" {
		return fmt.Errorf("config: goncho.workspace is required")
	}
	if cfg.Goncho.ObserverPeer == "" {
		return fmt.Errorf("config: goncho.observer_peer is required")
	}
	if !goncho.ValidDialecticLevel(cfg.Goncho.DialecticDefaultLevel) {
		return fmt.Errorf("config: goncho.dialectic_default_level %q is invalid; want one of minimal, low, medium, high, max", cfg.Goncho.DialecticDefaultLevel)
	}
	for _, limit := range []struct {
		name  string
		value int
	}{
		{name: "recent_messages", value: cfg.Goncho.RecentMessages},
		{name: "max_message_size", value: cfg.Goncho.MaxMessageSize},
		{name: "max_file_size", value: cfg.Goncho.MaxFileSize},
		{name: "get_context_max_tokens", value: cfg.Goncho.GetContextMaxTokens},
		{name: "dream_idle_timeout_minutes", value: cfg.Goncho.DreamIdleTimeoutMinutes},
		{name: "deriver_workers", value: cfg.Goncho.DeriverWorkers},
		{name: "representation_batch_max_tokens", value: cfg.Goncho.RepresentationBatchMaxTokens},
	} {
		if limit.value < 0 {
			return fmt.Errorf("config: goncho.%s must be non-negative, got %d", limit.name, limit.value)
		}
	}
	if cfg.Goncho.DeriverWorkers == 0 {
		return fmt.Errorf("config: goncho.deriver_workers must be at least 1")
	}
	if cfg.Delegation.MaxWaiting < 0 {
		return fmt.Errorf("config: delegation.max_waiting must be non-negative, got %d", cfg.Delegation.MaxWaiting)
	}
	return nil
}

func applyTelegramHomeChannel(cfg *Config, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	cfg.Telegram.HomeChannel.ChatID = value
	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		cfg.Telegram.AllowedChatID = id
	}
}

func normalizeTelegramConfig(cfg *TelegramCfg) {
	cfg.BotToken = strings.TrimSpace(cfg.BotToken)
	cfg.HomeChannel.Platform = strings.TrimSpace(cfg.HomeChannel.Platform)
	if cfg.HomeChannel.Platform == "" && strings.TrimSpace(cfg.HomeChannel.ChatID) != "" {
		cfg.HomeChannel.Platform = "telegram"
	}
	cfg.HomeChannel.ChatID = strings.TrimSpace(cfg.HomeChannel.ChatID)
	cfg.HomeChannel.Name = strings.TrimSpace(cfg.HomeChannel.Name)
	cfg.HomeChannel.ThreadID = strings.TrimSpace(cfg.HomeChannel.ThreadID)
	if cfg.HomeChannel.ChatID == "" && cfg.AllowedChatID != 0 {
		cfg.HomeChannel.ChatID = strconv.FormatInt(cfg.AllowedChatID, 10)
		if cfg.HomeChannel.Platform == "" {
			cfg.HomeChannel.Platform = "telegram"
		}
	}
	if cfg.AllowedChatID == 0 && cfg.HomeChannel.ChatID != "" {
		if id, err := strconv.ParseInt(cfg.HomeChannel.ChatID, 10, 64); err == nil {
			cfg.AllowedChatID = id
		}
	}
	cfg.AllowedUserIDs = compactInt64s(cfg.AllowedUserIDs)
}

func compactInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value != 0 {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeNavivoxConfig(cfg *NavivoxCfg) error {
	cfg.BindHost = strings.TrimSpace(cfg.BindHost)
	if cfg.BindHost == "" {
		cfg.BindHost = NavivoxDefaultBindHost
	}
	if cfg.Port == 0 {
		cfg.Port = NavivoxDefaultPort
	}
	cfg.ExposureMode = strings.ToLower(strings.TrimSpace(cfg.ExposureMode))
	if cfg.ExposureMode == "" {
		cfg.ExposureMode = NavivoxExposureLocal
	}
	cfg.AuthMode = strings.ToLower(strings.TrimSpace(cfg.AuthMode))
	if cfg.AuthMode == "" {
		cfg.AuthMode = NavivoxAuthPairingToken
	}
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.AllowOrigins = compactStrings(cfg.AllowOrigins)
	cfg.AllowedTailnetIdentities = compactStrings(cfg.AllowedTailnetIdentities)
	if err := normalizeNavivoxServers(cfg); err != nil {
		return err
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("config: navivox.port must be between 1 and 65535, got %d", cfg.Port)
	}
	switch cfg.ExposureMode {
	case NavivoxExposureLocal,
		NavivoxExposureTailscale,
		NavivoxExposureWireGuard,
		NavivoxExposureVPN,
		NavivoxExposurePublic:
	default:
		return fmt.Errorf("config: navivox.exposure_mode %q is invalid; want local, tailscale, wireguard, vpn, or public", cfg.ExposureMode)
	}
	switch cfg.AuthMode {
	case NavivoxAuthPairingToken, NavivoxAuthStaticToken, NavivoxAuthTokenAndTailscaleIdentity:
		if cfg.Enabled && cfg.Token == "" {
			return fmt.Errorf("config: navivox.token is required when navivox.enabled=true and auth_mode=%s", cfg.AuthMode)
		}
	case NavivoxAuthTailscaleIdentity:
	default:
		return fmt.Errorf("config: navivox.auth_mode %q is invalid; want pairing_token, static_token, tailscale_identity, or token_and_tailscale_identity", cfg.AuthMode)
	}
	if !cfg.Enabled {
		return nil
	}
	if navivoxWildcardHost(cfg.BindHost) && cfg.ExposureMode != NavivoxExposurePublic {
		return fmt.Errorf("config: navivox.bind_host %q requires navivox.exposure_mode=public and explicit confirmation", cfg.BindHost)
	}
	if cfg.ExposureMode == NavivoxExposureLocal && !navivoxLoopbackHost(cfg.BindHost) {
		return fmt.Errorf("config: navivox.exposure_mode=local requires loopback bind_host, got %q", cfg.BindHost)
	}
	if cfg.ExposureMode == NavivoxExposurePublic && !cfg.PublicConfirmed {
		return fmt.Errorf("config: navivox.exposure_mode=public requires navivox.public_confirmed=true")
	}
	return nil
}

func ValidateNavivoxForRuntime(cfg *NavivoxCfg) error {
	return normalizeNavivoxConfig(cfg)
}

func normalizeNavivoxServers(cfg *NavivoxCfg) error {
	if len(cfg.Servers) == 0 {
		cfg.Servers = nil
		return nil
	}
	servers := make(map[string]NavivoxServerCfg, len(cfg.Servers))
	for id, server := range cfg.Servers {
		normalizedID := strings.ToLower(strings.TrimSpace(id))
		if normalizedID != id || !agentIDPattern.MatchString(normalizedID) {
			return fmt.Errorf("config: navivox.servers.%s id is invalid", id)
		}
		server.Bind = strings.TrimSpace(server.Bind)
		server.Profiles = normalizeNavivoxProfileIDs(server.Profiles)
		server.Transports = normalizeNavivoxStringSet(server.Transports)
		server.Capabilities = normalizeNavivoxStringSet(server.Capabilities)
		servers[normalizedID] = server
	}
	cfg.Servers = servers
	return nil
}

func normalizeNavivoxProfileIDs(values []string) []string {
	cleaned := cleanStringSlice(values)
	if len(cleaned) == 0 {
		return nil
	}
	out := make([]string, 0, len(cleaned))
	seen := map[string]struct{}{}
	for _, value := range cleaned {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !agentIDPattern.MatchString(value) {
			// Keep the server usable and let the route report degraded evidence
			// instead of failing unrelated profiles at config-load time.
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeNavivoxStringSet(values []string) []string {
	cleaned := cleanStringSlice(values)
	if len(cleaned) == 0 {
		return nil
	}
	out := make([]string, 0, len(cleaned))
	seen := map[string]struct{}{}
	for _, value := range cleaned {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NavivoxExposureRequiresVPN reports whether the given exposure_mode value
// requires bind_host to match an active VPN interface IP.
func NavivoxExposureRequiresVPN(mode string) bool {
	switch mode {
	case NavivoxExposureTailscale, NavivoxExposureWireGuard, NavivoxExposureVPN:
		return true
	default:
		return false
	}
}

// ValidateNavivoxBindAgainstVPN returns nil when navivox.bind_host either is
// not required to be a VPN interface IP (exposure_mode local/public, or
// channel disabled) or matches one of the live VPN IPs supplied by the
// caller. The list is supplied as plain strings so config has no dependency
// on the network/vpnhost package.
func ValidateNavivoxBindAgainstVPN(cfg *NavivoxCfg, vpnIPs []string) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if !NavivoxExposureRequiresVPN(cfg.ExposureMode) {
		return nil
	}
	host := navivoxHostOnly(cfg.BindHost)
	if host == "" {
		return fmt.Errorf("config: navivox.bind_host is empty; exposure_mode=%s requires a VPN interface IP", cfg.ExposureMode)
	}
	for _, ip := range vpnIPs {
		if strings.EqualFold(strings.TrimSpace(ip), host) {
			return nil
		}
	}
	if len(vpnIPs) == 0 {
		return fmt.Errorf("config: navivox.exposure_mode=%s but no active VPN interface was detected; bind_host %q cannot be validated", cfg.ExposureMode, cfg.BindHost)
	}
	return fmt.Errorf("config: navivox.bind_host %q does not match any active VPN interface IP (%v); exposure_mode=%s requires a VPN bind", cfg.BindHost, vpnIPs, cfg.ExposureMode)
}

func navivoxLoopbackHost(host string) bool {
	host = navivoxHostOnly(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func navivoxWildcardHost(host string) bool {
	host = navivoxHostOnly(host)
	return host == "" || host == "0.0.0.0" || host == "::" || host == "[::]"
}

func navivoxHostOnly(raw string) string {
	host := strings.TrimSpace(raw)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.Trim(strings.ToLower(host), "[]")
}

func normalizeAuxiliaryTask(task *AuxiliaryTaskCfg, defaultCurator bool) {
	task.Provider = strings.TrimSpace(task.Provider)
	task.Model = strings.TrimSpace(task.Model)
	task.BaseURL = strings.TrimSpace(task.BaseURL)
	task.APIKey = strings.TrimSpace(task.APIKey)
	if defaultCurator {
		if task.Provider == "" {
			task.Provider = "auto"
		}
		if task.Timeout == 0 {
			task.Timeout = 600
		}
		if task.ExtraBody == nil {
			task.ExtraBody = map[string]any{}
		}
	}
}

func normalizeAgentImageInputMode(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeGatewayProxyURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func XDGConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func xdgDataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func xdgStateHome() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state")
}

// GormesHome returns the native Gormes state/config root. GORMES_HOME wins;
// otherwise Gormes uses ~/.gormes so it never needs to share Hermes runtime
// state such as ~/.hermes/auth.json or ~/.hermes/gateway_state.json.
func GormesHome() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_HOME")); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gormes")
}

// SubprocessHome returns the Hermes-compatible subprocess HOME for the active
// Gormes home. The directory must already exist; this keeps legacy/default
// profiles from silently redirecting shell tools before profile creation or
// migration has materialized the profile-local home tree.
func SubprocessHome() (string, bool) {
	return SubprocessHomeFor(GormesHome())
}

// SubprocessHomeFor returns <gormesHome>/home when it exists as a directory.
func SubprocessHomeFor(gormesHome string) (string, bool) {
	gormesHome = strings.TrimSpace(gormesHome)
	if gormesHome == "" {
		return "", false
	}
	candidate := filepath.Join(gormesHome, "home")
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return candidate, true
}

// ConfigPath returns the Gormes TOML config file path.
func ConfigPath() string {
	return filepath.Join(GormesHome(), "config.toml")
}

// YAMLConfigPath returns the YAML variant of the Gormes config file path.
// This is used as a fallback when config.toml doesn't exist, allowing Hermes
// users to copy their config.yaml directly without converting to TOML.
func YAMLConfigPath() string {
	return filepath.Join(GormesHome(), "config.yaml")
}

// LogPath returns the default path for the Gormes log file.
func LogPath() string {
	return filepath.Join(GormesHome(), "gormes.log")
}

// CrashLogDir returns the directory where TUI panic dumps are written.
func CrashLogDir() string {
	return GormesHome()
}

// SessionDBPath returns the default location of the bbolt sessions map.
func SessionDBPath() string {
	return filepath.Join(GormesHome(), "sessions.db")
}

// SessionIndexMirrorPath returns the default location of the read-only YAML
// mirror for the bbolt session map.
func SessionIndexMirrorPath() string {
	return filepath.Join(GormesHome(), "sessions", "index.yaml")
}

// MemoryDBPath returns the default location of the Phase-3.A SQLite
// memory database.
func MemoryDBPath() string {
	return filepath.Join(GormesHome(), "memory.db")
}

// KanbanDBPath returns the native Gormes Kanban SQLite database path.
// Gormes intentionally ignores Hermes runtime env vars here; Hermes state is
// only read by explicit migrate commands.
func KanbanDBPath() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_KANBAN_DB")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("GORMES_KANBAN_HOME")); v != "" {
		return filepath.Join(v, "kanban.db")
	}
	return filepath.Join(GormesHome(), "kanban.db")
}

// KanbanHome returns the root directory for the Kanban board registry.
// Named board databases live under <KanbanHome>/kanban/boards/<slug>/kanban.db
// while the legacy default board lives at <KanbanHome>/kanban.db.
func KanbanHome() string {
	if v := strings.TrimSpace(os.Getenv("GORMES_KANBAN_HOME")); v != "" {
		return v
	}
	return GormesHome()
}

// CronMirrorPath returns the resolved CRON.md path — either
// cfg.Cron.MirrorPath (explicit override) or the Gormes home default.
func (c Config) CronMirrorPath() string {
	if c.Cron.MirrorPath != "" {
		return c.Cron.MirrorPath
	}
	return filepath.Join(GormesHome(), "cron", "CRON.md")
}

// SkillsRoot returns the root directory of the static skills runtime.
// Explicit override wins; otherwise the Gormes home default is used.
func (c Config) SkillsRoot() string {
	if c.Skills.Root != "" {
		return c.Skills.Root
	}
	return filepath.Join(GormesHome(), "skills")
}

// HooksRoot returns the root directory for gateway HOOK.yaml hook directories.
func HooksRoot() string {
	return filepath.Join(GormesHome(), "hooks")
}

// GatewayRuntimeStatusPath returns the shared gateway_state.json read-model
// path for live gateway lifecycle status.
func GatewayRuntimeStatusPath() string {
	return filepath.Join(GormesHome(), "gateway_state.json")
}

// GatewayLockDir returns the machine-local directory for token-scoped gateway
// credential locks.
func GatewayLockDir() string {
	return filepath.Join(GormesHome(), "gateway-locks")
}

// BootPath returns the BOOT.md path used by the built-in gateway startup hook.
func BootPath() string {
	return filepath.Join(GormesHome(), "BOOT.md")
}

// SkillsUsageLogPath returns the append-only JSONL path for skill usage.
// Explicit override wins; otherwise it lives under the skills root.
func (c Config) SkillsUsageLogPath() string {
	if c.Skills.UsageLogPath != "" {
		return c.Skills.UsageLogPath
	}
	return filepath.Join(c.SkillsRoot(), "usage.jsonl")
}

// ToolAuditLogPath returns the append-only JSONL path for tool execution
// audit records.
func ToolAuditLogPath() string {
	return filepath.Join(GormesHome(), "tools", "audit.jsonl")
}

// ResolvedRunLogPath returns the JSONL path for append-only subagent run logs.
// An explicit TOML override wins; otherwise Gormes writes under GormesHome.
func (d DelegationCfg) ResolvedRunLogPath() string {
	if d.RunLogPath != "" {
		return d.RunLogPath
	}
	return filepath.Join(GormesHome(), "subagents", "runs.jsonl")
}
