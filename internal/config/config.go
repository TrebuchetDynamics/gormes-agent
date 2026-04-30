// Package config loads Gormes configuration from CLI flags > env > TOML > defaults.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// CurrentConfigVersion is the schema version this binary writes + accepts.
// When a breaking change to the TOML schema lands, bump this constant and
// add a migration in runMigrations() so older files stay readable.
const CurrentConfigVersion = 1

type Config struct {
	// ConfigVersion is the schema version of the loaded TOML file. Read
	// before any struct fields so migrations can run against the raw
	// document. Absent in TOML = treated as 1.
	ConfigVersion int `toml:"_config_version"`

	Hermes     HermesCfg     `toml:"hermes"`
	Gateway    GatewayCfg    `toml:"gateway"`
	Display    DisplayCfg    `toml:"display"`
	TUI        TUICfg        `toml:"tui"`
	Input      InputCfg      `toml:"input"`
	Telegram   TelegramCfg   `toml:"telegram"`
	Discord    DiscordCfg    `toml:"discord"`
	Slack      SlackCfg      `toml:"slack"`
	Yuanbao    YuanbaoCfg    `toml:"yuanbao"`
	Web        WebCfg        `toml:"web"`
	Security   SecurityCfg   `toml:"security"`
	Cron       CronCfg       `toml:"cron"`
	Skills     SkillsCfg     `toml:"skills"`
	Delegation DelegationCfg `toml:"delegation"`
	Goncho     GonchoCfg     `toml:"goncho"`
	// Resume is set only via the --resume CLI flag; intentionally not
	// a TOML field. Empty means "use whatever internal/session had
	// persisted for this binary's default key."
	Resume string `toml:"-"`
}

type TelegramCfg struct {
	BotToken               string  `toml:"bot_token"`
	AllowedChatID          int64   `toml:"allowed_chat_id"`
	RequireMention         bool    `toml:"require_mention"`
	BotUsername            string  `toml:"bot_username"`
	CoalesceMs             int     `toml:"coalesce_ms"`
	FreshFinalAfterSeconds float64 `toml:"fresh_final_after_seconds"`
	FirstRunDiscovery      bool    `toml:"first_run_discovery"`
	// MemoryQueueCap (Phase 3.A): async worker queue capacity in
	// the telegram subcommand's SqliteStore. Defaults to 1024.
	MemoryQueueCap int `toml:"memory_queue_cap"`
	// ExtractorBatchSize / ExtractorPollInterval (Phase 3.B).
	ExtractorBatchSize    int           `toml:"extractor_batch_size"`
	ExtractorPollInterval time.Duration `toml:"extractor_poll_interval"`
	// RecallEnabled / RecallWeightThreshold / RecallMaxFacts / RecallDepth
	// (Phase 3.C).
	RecallEnabled         bool    `toml:"recall_enabled"`
	RecallWeightThreshold float64 `toml:"recall_weight_threshold"`
	RecallMaxFacts        int     `toml:"recall_max_facts"`
	RecallDepth           int     `toml:"recall_depth"`
	// RecallDecayHorizonDays (Phase 3.E.6) — maps to
	// RecallConfig.DecayHorizonDays. An edge's effective weight
	// decays linearly from raw at age=0 to 0 at this many days old.
	// 0 = unset (withDefaults promotes to 180). <0 = disabled.
	RecallDecayHorizonDays int `toml:"recall_decay_horizon_days"`
	// MirrorEnabled / MirrorPath / MirrorInterval (Phase 3.D.5).
	// The Memory Mirror exports SQLite entities/relationships to USER.md.
	MirrorEnabled  bool          `toml:"mirror_enabled"`
	MirrorPath     string        `toml:"mirror_path"`
	MirrorInterval time.Duration `toml:"mirror_interval"`
	// Phase 3.D semantic fusion — all opt-in via SemanticEnabled.
	SemanticEnabled       bool          `toml:"semantic_enabled"`
	SemanticEndpoint      string        `toml:"semantic_endpoint"`
	SemanticModel         string        `toml:"semantic_model"`
	SemanticTopK          int           `toml:"semantic_top_k"`
	SemanticMinSimilarity float64       `toml:"semantic_min_similarity"`
	EmbedderPollInterval  time.Duration `toml:"embedder_poll_interval"`
	EmbedderBatchSize     int           `toml:"embedder_batch_size"`
	EmbedderCallTimeout   time.Duration `toml:"embedder_call_timeout"`
	QueryEmbedTimeout     time.Duration `toml:"query_embed_timeout"`
}

// DiscordCfg drives the Discord channel adapter.
type DiscordCfg struct {
	Token             string   `toml:"token"`
	AllowedChannelID  string   `toml:"allowed_channel_id"`
	ServerActions     []string `toml:"server_actions"`
	CoalesceMs        int      `toml:"coalesce_ms"`
	FirstRunDiscovery bool     `toml:"first_run_discovery"`
}

func (c DiscordCfg) Enabled() bool {
	if c.Token == "" {
		return false
	}
	return c.AllowedChannelID != "" || c.FirstRunDiscovery
}

// SlackCfg drives the Slack Socket Mode channel adapter.
type SlackCfg struct {
	Enabled           bool   `toml:"enabled"`
	BotToken          string `toml:"bot_token"`
	AppToken          string `toml:"app_token"`
	AllowedChannelID  string `toml:"allowed_channel_id"`
	CoalesceMs        int    `toml:"coalesce_ms"`
	FirstRunDiscovery bool   `toml:"first_run_discovery"`
}

type CronCfg struct {
	Enabled        bool          `toml:"enabled"`
	CallTimeout    time.Duration `toml:"call_timeout"`
	MirrorInterval time.Duration `toml:"mirror_interval"`
	MirrorPath     string        `toml:"mirror_path"`
}

// WebCfg mirrors Hermes config.yaml's web.backend and web.use_gateway fields.
type WebCfg struct {
	Backend    string `toml:"backend"`
	UseGateway bool   `toml:"use_gateway"`
}

// SecurityCfg mirrors Hermes config.yaml security controls that affect native
// Go tools.
type SecurityCfg struct {
	WebsiteBlocklist WebsiteBlocklistCfg `toml:"website_blocklist"`
}

type WebsiteBlocklistCfg struct {
	Enabled     bool     `toml:"enabled"`
	Domains     []string `toml:"domains"`
	SharedFiles []string `toml:"shared_files"`
	BaseDir     string   `toml:"-"`
}

// SkillsCfg configures the Phase 2.G0 static skills runtime.
type SkillsCfg struct {
	Root             string `toml:"root"`
	SelectionCap     int    `toml:"selection_cap"`
	MaxDocumentBytes int    `toml:"max_document_bytes"`
	UsageLogPath     string `toml:"usage_log_path"`
}

// DelegationCfg configures Phase 2.E subagent execution.
type DelegationCfg struct {
	Enabled               bool          `toml:"enabled"`
	MaxDepth              int           `toml:"max_depth"`
	MaxConcurrentChildren int           `toml:"max_concurrent_children"`
	DefaultMaxIterations  int           `toml:"default_max_iterations"`
	DefaultTimeout        time.Duration `toml:"default_timeout"`
	RunLogPath            string        `toml:"run_log_path"`
	MaxWaiting            int           `toml:"max_waiting"`
}

// GonchoCfg configures the in-process Honcho-compatible memory facade.
type GonchoCfg struct {
	Enabled                      bool   `toml:"enabled"`
	Workspace                    string `toml:"workspace"`
	ObserverPeer                 string `toml:"observer_peer"`
	RecentMessages               int    `toml:"recent_messages"`
	MaxMessageSize               int    `toml:"max_message_size"`
	MaxFileSize                  int    `toml:"max_file_size"`
	GetContextMaxTokens          int    `toml:"get_context_max_tokens"`
	ReasoningEnabled             bool   `toml:"reasoning_enabled"`
	PeerCardEnabled              bool   `toml:"peer_card_enabled"`
	SummaryEnabled               bool   `toml:"summary_enabled"`
	DreamEnabled                 bool   `toml:"dream_enabled"`
	DreamIdleTimeoutMinutes      int    `toml:"dream_idle_timeout_minutes"`
	DeriverWorkers               int    `toml:"deriver_workers"`
	RepresentationBatchMaxTokens int    `toml:"representation_batch_max_tokens"`
	DialecticDefaultLevel        string `toml:"dialectic_default_level"`
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
		Enabled               bool   `toml:"enabled"`
		MaxDepth              int    `toml:"max_depth"`
		MaxConcurrentChildren int    `toml:"max_concurrent_children"`
		DefaultMaxIterations  int    `toml:"default_max_iterations"`
		DefaultTimeout        string `toml:"default_timeout"`
		RunLogPath            string `toml:"run_log_path"`
		MaxWaiting            int    `toml:"max_waiting"`
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
	Endpoint string `toml:"endpoint"`
	APIKey   string `toml:"api_key"`
	Model    string `toml:"model"`
	Provider string `toml:"provider"`
}

type GatewayCfg struct {
	ProxyURL string `toml:"proxy_url"`
	ProxyKey string `toml:"proxy_key"`
}

type DisplayCfg struct {
	ToolProgress string                        `toml:"tool_progress"`
	Platforms    map[string]DisplayPlatformCfg `toml:"platforms"`
}

type DisplayPlatformCfg struct {
	ToolProgress string `toml:"tool_progress"`
}

type TUICfg struct {
	Theme         string `toml:"theme"`
	MouseTracking bool   `toml:"mouse_tracking"`
}

type InputCfg struct {
	MaxBytes int `toml:"max_bytes"`
	MaxLines int `toml:"max_lines"`
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
}

type OneshotInferenceResolution = InferenceResolution
type TUIInferenceResolution = InferenceResolution

// ResolveOneshotInference applies the Hermes-compatible one-shot precedence:
// flag > GORMES_INFERENCE_* env > config defaults. A provider override without
// a flag/env model is rejected so a stale configured model is not silently
// paired with a different provider.
func ResolveOneshotInference(req OneshotInferenceRequest) (OneshotInferenceResolution, error) {
	req.CommandLabel = "gormes -z"
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
	return resolution, nil
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
	if err := loadHermesConfigYAML(&cfg); err != nil {
		return cfg, err
	}
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
	return cfg, nil
}

func defaults() Config {
	return Config{
		ConfigVersion: CurrentConfigVersion,
		Hermes: HermesCfg{
			Model: "hermes-agent",
		},
		TUI:   TUICfg{Theme: "dark", MouseTracking: true},
		Input: InputCfg{MaxBytes: 200_000, MaxLines: 10_000},
		Telegram: TelegramCfg{
			CoalesceMs:             1000,
			FreshFinalAfterSeconds: 60.0,
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
		},
		Security: SecurityCfg{
			WebsiteBlocklist: WebsiteBlocklistCfg{
				BaseDir: GormesHome(),
			},
		},
		Cron: CronCfg{
			Enabled:        false,
			CallTimeout:    60 * time.Second,
			MirrorInterval: 30 * time.Second,
			MirrorPath:     "",
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
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := toml.NewDecoder(bytes.NewReader(data)).EnableUnmarshalerInterface().Decode(cfg); err != nil {
		return err
	}
	// Absent _config_version in TOML = treat as v1 (pre-versioning files).
	// Defaults() set it to CurrentConfigVersion, but unmarshal overwrites
	// with 0 when the key isn't present.
	if cfg.ConfigVersion == 0 {
		cfg.ConfigVersion = 1
	}
	return migrateConfig(cfg)
}

type hermesConfigYAML struct {
	Model     hermesModelConfigYAML               `yaml:"model"`
	Web       hermesWebConfigYAML                 `yaml:"web"`
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

type hermesDisplayConfigYAML struct {
	ToolProgress          interface{}                          `yaml:"tool_progress"`
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

func loadHermesConfigYAML(cfg *Config) error {
	path := hermesConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var hc hermesConfigYAML
	if err := yaml.Unmarshal(data, &hc); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	applyHermesModelConfig(cfg, hc.Model)
	applyHermesWebConfig(cfg, hc.Web)
	applyHermesDisplayConfig(cfg, hc.Display)
	applyHermesSecurityConfig(cfg, hc.Security)
	applyHermesTelegramConfig(cfg, hc.Platforms["telegram"], hc.Streaming)
	applyHermesDiscordConfig(cfg, hc.Platforms["discord"])
	applyHermesSlackConfig(cfg, hc.Platforms["slack"])
	return nil
}

func applyHermesModelConfig(cfg *Config, model hermesModelConfigYAML) {
	if defaultModel := strings.TrimSpace(model.Default); defaultModel != "" {
		cfg.Hermes.Model = defaultModel
	}
	if provider := strings.TrimSpace(model.Provider); provider != "" {
		cfg.Hermes.Provider = provider
	}
}

func applyHermesWebConfig(cfg *Config, web hermesWebConfigYAML) {
	if backend := strings.ToLower(strings.TrimSpace(web.Backend)); backend != "" {
		cfg.Web.Backend = backend
	}
	if web.UseGateway {
		cfg.Web.UseGateway = true
	}
}

func applyHermesDisplayConfig(cfg *Config, display hermesDisplayConfigYAML) {
	if mode, ok := normalizeHermesToolProgressMode(display.ToolProgress); ok {
		cfg.Display.ToolProgress = mode
	}
	for platform, platformDisplay := range display.Platforms {
		if mode, ok := normalizeHermesToolProgressMode(platformDisplay.ToolProgress); ok {
			putDisplayPlatformToolProgress(&cfg.Display, platform, mode)
		}
	}
	for platform, rawMode := range display.ToolProgressOverrides {
		key := normalizeDisplayPlatformKey(platform)
		if key == "" {
			continue
		}
		if existing, ok := cfg.Display.Platforms[key]; ok && strings.TrimSpace(existing.ToolProgress) != "" {
			continue
		}
		if mode, ok := normalizeHermesToolProgressMode(rawMode); ok {
			putDisplayPlatformToolProgress(&cfg.Display, key, mode)
		}
	}
}

func putDisplayPlatformToolProgress(display *DisplayCfg, platform, mode string) {
	key := normalizeDisplayPlatformKey(platform)
	if key == "" {
		return
	}
	if display.Platforms == nil {
		display.Platforms = map[string]DisplayPlatformCfg{}
	}
	entry := display.Platforms[key]
	entry.ToolProgress = mode
	display.Platforms[key] = entry
}

func normalizeDisplayPlatformKey(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
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

func applyHermesSecurityConfig(cfg *Config, security hermesSecurityConfigYAML) {
	blocklist := security.WebsiteBlocklist
	if blocklist.Enabled || len(blocklist.Domains) > 0 || len(blocklist.SharedFiles) > 0 {
		cfg.Security.WebsiteBlocklist.BaseDir = filepath.Dir(hermesConfigPath())
	}
	if blocklist.Enabled {
		cfg.Security.WebsiteBlocklist.Enabled = true
	}
	if len(blocklist.Domains) > 0 {
		cfg.Security.WebsiteBlocklist.Domains = append([]string(nil), blocklist.Domains...)
	}
	if len(blocklist.SharedFiles) > 0 {
		cfg.Security.WebsiteBlocklist.SharedFiles = append([]string(nil), blocklist.SharedFiles...)
	}
}

func applyHermesTelegramConfig(cfg *Config, platform hermesPlatformConfigYAML, streaming hermesStreamingConfigYAML) {
	if token := strings.TrimSpace(firstNonEmpty(platform.Token, platform.APIKey)); token != "" {
		cfg.Telegram.BotToken = token
	}
	if chatID, ok := coerceHermesChatID(platform.HomeChannel.ChatID); ok {
		cfg.Telegram.AllowedChatID = chatID
	}
	if v, ok := boolFromInterface(platform.Extra["require_mention"]); ok {
		cfg.Telegram.RequireMention = v
	}
	if username := strings.TrimSpace(stringFromInterface(platform.Extra["bot_username"])); username != "" {
		cfg.Telegram.BotUsername = username
	}
	if streaming.FreshFinalAfterSeconds != nil {
		cfg.Telegram.FreshFinalAfterSeconds = *streaming.FreshFinalAfterSeconds
	}
}

func applyHermesDiscordConfig(cfg *Config, platform hermesPlatformConfigYAML) {
	if token := strings.TrimSpace(firstNonEmpty(platform.Token, platform.APIKey)); token != "" {
		cfg.Discord.Token = token
	}
	if chatID := strings.TrimSpace(hermesChatIDString(platform.HomeChannel.ChatID)); chatID != "" {
		cfg.Discord.AllowedChannelID = chatID
	}
	if v, ok := boolFromInterface(platform.Extra["first_run_discovery"]); ok {
		cfg.Discord.FirstRunDiscovery = v
	}
}

func applyHermesSlackConfig(cfg *Config, platform hermesPlatformConfigYAML) {
	if platform.Enabled {
		cfg.Slack.Enabled = true
	}
	if token := strings.TrimSpace(firstNonEmpty(platform.Token, platform.APIKey)); token != "" {
		cfg.Slack.BotToken = token
	}
	if appToken := strings.TrimSpace(stringFromInterface(platform.Extra["app_token"])); appToken != "" {
		cfg.Slack.AppToken = appToken
	}
	if chatID := strings.TrimSpace(hermesChatIDString(platform.HomeChannel.ChatID)); chatID != "" {
		cfg.Slack.AllowedChannelID = chatID
	}
	if v, ok := boolFromInterface(platform.Extra["first_run_discovery"]); ok {
		cfg.Slack.FirstRunDiscovery = v
	}
}

func hermesConfigPath() string {
	if hermesHome := strings.TrimSpace(os.Getenv("HERMES_HOME")); hermesHome != "" {
		return filepath.Join(hermesHome, "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".hermes", "config.yaml")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func coerceHermesChatID(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), v == float64(int64(v))
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func hermesChatIDString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return ""
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func boolFromInterface(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		return parsed, err == nil
	default:
		return false, false
	}
}

func stringFromInterface(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

// migrateConfig applies version-gated schema migrations in sequence,
// bumping cfg.ConfigVersion after each step. A config written by a
// newer binary (version > CurrentConfigVersion) is rejected with a
// clear error so operators know to upgrade — silently downgrading
// would quietly drop unknown fields.
func migrateConfig(cfg *Config) error {
	if cfg.ConfigVersion > CurrentConfigVersion {
		return fmt.Errorf(
			"config: _config_version=%d is from a newer binary (this binary knows up to v%d); upgrade gormes or hand-edit the file",
			cfg.ConfigVersion, CurrentConfigVersion)
	}
	// No migrations defined yet (v1 is the first version). When a v1->v2
	// schema change ships, add:
	//   if cfg.ConfigVersion == 1 { migrate1to2(cfg); cfg.ConfigVersion = 2 }
	// Each step is idempotent because it only runs when ConfigVersion
	// matches its source version.
	cfg.ConfigVersion = CurrentConfigVersion
	return nil
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
	if v := os.Getenv("GORMES_TUI_MOUSE_TRACKING"); v != "" {
		parsed, err := parseEnvBool("GORMES_TUI_MOUSE_TRACKING", v)
		if err != nil {
			return err
		}
		cfg.TUI.MouseTracking = parsed
	}
	if v := os.Getenv("GORMES_TELEGRAM_TOKEN"); v != "" {
		cfg.Telegram.BotToken = v
	}
	if v := os.Getenv("GORMES_TELEGRAM_CHAT_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Telegram.AllowedChatID = id
		}
	}
	if v := os.Getenv("GORMES_DISCORD_TOKEN"); v != "" {
		cfg.Discord.Token = v
	}
	if v := os.Getenv("GORMES_DISCORD_CHANNEL_ID"); v != "" {
		cfg.Discord.AllowedChannelID = v
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
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
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
	cfg.Slack.BotToken = strings.TrimSpace(cfg.Slack.BotToken)
	cfg.Slack.AppToken = strings.TrimSpace(cfg.Slack.AppToken)
	cfg.Slack.AllowedChannelID = strings.TrimSpace(cfg.Slack.AllowedChannelID)
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

func normalizeGatewayProxyURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func xdgConfigHome() string {
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

// ConfigPath returns the Gormes TOML config file path.
func ConfigPath() string {
	return filepath.Join(GormesHome(), "config.toml")
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

// LegacyHermesHome reports an upstream Hermes state directory if one is
// discoverable. Returns the path and true when either $HERMES_HOME is
// set OR ~/.hermes/ exists. Gormes does NOT read state from this path at
// runtime; it uses GormesHome instead. Surfacing the detection at startup lets
// operators know their Hermes state wasn't silently ignored.
//
// Planned: Phase 5.O will add a `gormes migrate --from-hermes` command
// that copies relevant state (sessions, memory snapshots) across.
// Until then, operators see a one-line info log and can migrate
// manually.
func LegacyHermesHome() (string, bool) {
	if v := strings.TrimSpace(os.Getenv("HERMES_HOME")); v != "" {
		return v, true
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	candidate := filepath.Join(home, ".hermes")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate, true
	}
	return "", false
}
