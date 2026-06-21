// Package config loads Gormes configuration from CLI flags > env > TOML > defaults.
package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/externalsecrets"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// ConfigVersion is the canonical schema version of the loaded TOML file.
	// Absent in TOML is treated as legacy v1. LegacyConfigVersion keeps
	// `_config_version` readable as a fallback, but new writes use
	// `config_version`.
	ConfigVersion       int `toml:"config_version" yaml:"config_version"`
	LegacyConfigVersion int `toml:"_config_version,omitempty" yaml:"_config_version,omitempty"`

	Hermes        HermesCfg                `toml:"hermes" yaml:"hermes"`
	Router        RouterCfg                `toml:"router" yaml:"router"`
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
	Workspace     WorkspaceCfg             `toml:"workspace" yaml:"workspace"`
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
	// a TOML field. Empty means "use whatever internal/persistence/session had
	// persisted for this binary's default key."
	Resume string `toml:"-"`
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
	applyExternalSecretSources(cfg)
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

func applyExternalSecretSources(cfg Config) {
	bw := cfg.Secrets.Bitwarden
	_ = externalsecrets.ApplyBitwarden(context.Background(), externalsecrets.BitwardenConfig{
		Enabled:          bw.Enabled,
		AccessTokenEnv:   bw.AccessTokenEnv,
		ProjectID:        bw.ProjectID,
		CacheTTLSeconds:  bw.CacheTTLSeconds,
		OverrideExisting: bw.OverrideExisting,
		AutoInstall:      bw.AutoInstall,
		ServerURL:        bw.ServerURL,
	}, externalsecrets.BitwardenOptions{HomeDir: GormesHome()})
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
		Router: RouterCfg{
			Listen:     "127.0.0.1:8787",
			RedactLogs: true,
			SetupMode:  "local_gateway",
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
			GatewayLabel: NavivoxDefaultGatewayLabel,
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
			Bitwarden: BitwardenSecretSourceCfg{
				AccessTokenEnv:   externalsecrets.DefaultBitwardenAccessTokenEnv,
				CacheTTLSeconds:  externalsecrets.DefaultBitwardenCacheTTLSeconds,
				OverrideExisting: true,
				AutoInstall:      true,
			},
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
			DefaultTimeout:        0, // 0 = no wall-clock cap; mirrors Hermes fix(delegation) bba9b519a
			RunLogPath:            "",
			MaxWaiting:            128,
		},
		Updates: UpdatesCfg{
			PreUpdateBackup: false,
			BackupKeep:      5,
		},
		Goncho: defaultGonchoCfg(),
	}
}

func loadFile(cfg *Config) error {
	currentHome := GormesHome()
	baseHome := GormesBaseHome()
	baseLoaded := false
	if baseHome != currentHome {
		if loaded, err := loadConfigFileAt(cfg, filepath.Join(baseHome, "config.toml"), filepath.Join(baseHome, "config.yaml"), true); err != nil {
			return err
		} else if loaded {
			baseLoaded = true
			normalizeDecodedConfigVersion(cfg)
			if err := migrateConfig(cfg); err != nil {
				return err
			}
		}
	}
	loaded, err := loadConfigFileAt(cfg, ConfigPath(), YAMLConfigPath(), !baseLoaded)
	if err != nil {
		return err
	}
	if !loaded {
		return nil
	}
	normalizeDecodedConfigVersion(cfg)
	return migrateConfig(cfg)
}

func loadConfigFileAt(cfg *Config, tomlPath, yamlPath string, resetVersion bool) (bool, error) {
	data, err := os.ReadFile(tomlPath)
	if os.IsNotExist(err) {
		// Try YAML fallback for Hermes migrants.
		yamlData, yamlErr := os.ReadFile(yamlPath)
		if os.IsNotExist(yamlErr) {
			return false, nil
		}
		if yamlErr != nil {
			return false, fmt.Errorf("read %s: %w", yamlPath, yamlErr)
		}
		if resetVersion {
			cfg.ConfigVersion = 0
			cfg.LegacyConfigVersion = 0
		}
		if err := yaml.NewDecoder(bytes.NewReader(yamlData)).Decode(cfg); err != nil {
			return false, fmt.Errorf("decode %s: %w", yamlPath, err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", tomlPath, err)
	}
	if resetVersion {
		cfg.ConfigVersion = 0
		cfg.LegacyConfigVersion = 0
	}
	if err := toml.NewDecoder(bytes.NewReader(data)).EnableUnmarshalerInterface().Decode(cfg); err != nil {
		return false, err
	}
	return true, nil
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
	if strings.TrimSpace(cfg.Display.ToolProgress) == "" {
		if v := strings.TrimSpace(os.Getenv("HERMES_TOOL_PROGRESS")); v != "" {
			parsed, err := parseEnvBool("HERMES_TOOL_PROGRESS", v)
			if err != nil {
				return err
			}
			if parsed {
				cfg.Display.ToolProgress = "all"
			} else {
				cfg.Display.ToolProgress = "off"
			}
		} else if mode, ok := normalizeHermesToolProgressMode(os.Getenv("HERMES_TOOL_PROGRESS_MODE")); ok {
			cfg.Display.ToolProgress = mode
		}
	}
	if v := strings.TrimSpace(firstNonEmpty(os.Getenv("GORMES_PREFILL_MESSAGES_FILE"), os.Getenv("HERMES_PREFILL_MESSAGES_FILE"))); v != "" {
		cfg.Agent.PrefillMessagesFile = v
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
	if err := normalizeGonchoConfig(&cfg.Goncho); err != nil {
		return err
	}
	if cfg.Delegation.MaxWaiting < 0 {
		return fmt.Errorf("config: delegation.max_waiting must be non-negative, got %d", cfg.Delegation.MaxWaiting)
	}
	return nil
}

func normalizeAgentImageInputMode(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeGatewayProxyURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}
