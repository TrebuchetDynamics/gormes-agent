package channels

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

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
	BotTokenRef            *credentials.SecretRef        `toml:"bot_token_ref" yaml:"bot_token_ref" json:"bot_token_ref,omitempty"`
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

func ApplyTelegramHomeChannel(cfg *TelegramCfg, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	cfg.HomeChannel.ChatID = value
	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		cfg.AllowedChatID = id
	}
}

func NormalizeTelegramConfig(cfg *TelegramCfg) {
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

// DiscordCfg drives the Discord channel adapter.
type DiscordAccountCfg struct {
	Token            string   `toml:"token" yaml:"token"`
	AllowedChannelID string   `toml:"allowed_channel_id" yaml:"allowed_channel_id"`
	AllowedChannels  []string `toml:"allowed_channels" yaml:"allowed_channels"`
}

type DiscordCfg struct {
	Token                string                       `toml:"token" yaml:"token"`
	TokenRef             *credentials.SecretRef       `toml:"token_ref" yaml:"token_ref" json:"token_ref,omitempty"`
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
	BotTokenRef          *credentials.SecretRef     `toml:"bot_token_ref" yaml:"bot_token_ref" json:"bot_token_ref,omitempty"`
	AppToken             string                     `toml:"app_token" yaml:"app_token"`
	AppTokenRef          *credentials.SecretRef     `toml:"app_token_ref" yaml:"app_token_ref" json:"app_token_ref,omitempty"`
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

func parseCSV(value string) []string {
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
		return parseCSV(v)
	default:
		return parseCSV(fmt.Sprint(v))
	}
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
