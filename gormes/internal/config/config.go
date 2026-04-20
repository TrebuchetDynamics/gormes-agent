// Package config loads Gormes configuration from CLI flags > env > TOML > defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
)

type Config struct {
	Hermes   HermesCfg   `toml:"hermes"`
	TUI      TUICfg      `toml:"tui"`
	Input    InputCfg    `toml:"input"`
	Telegram TelegramCfg `toml:"telegram"`
	// Resume is set only via the --resume CLI flag; intentionally not
	// a TOML field. Empty means "use whatever internal/session had
	// persisted for this binary's default key."
	Resume string `toml:"-"`
}

type TelegramCfg struct {
	BotToken          string `toml:"bot_token"`
	AllowedChatID     int64  `toml:"allowed_chat_id"`
	CoalesceMs        int    `toml:"coalesce_ms"`
	FirstRunDiscovery bool   `toml:"first_run_discovery"`
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
}

type HermesCfg struct {
	Endpoint string `toml:"endpoint"`
	APIKey   string `toml:"api_key"`
	Model    string `toml:"model"`
}

type TUICfg struct {
	Theme string `toml:"theme"`
}

type InputCfg struct {
	MaxBytes int `toml:"max_bytes"`
	MaxLines int `toml:"max_lines"`
}

// Load resolves configuration from (in precedence order) CLI flags, env vars,
// a TOML file at $XDG_CONFIG_HOME/gormes/config.toml, and built-in defaults.
// Pass os.Args[1:] as args; pass nil to skip flag parsing entirely (useful in tests).
func Load(args []string) (Config, error) {
	cfg := defaults()
	if err := loadFile(&cfg); err != nil {
		return cfg, err
	}
	loadEnv(&cfg)
	if err := loadFlags(&cfg, args); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func defaults() Config {
	return Config{
		Hermes: HermesCfg{
			Endpoint: "http://127.0.0.1:8642",
			Model:    "hermes-agent",
		},
		TUI:   TUICfg{Theme: "dark"},
		Input: InputCfg{MaxBytes: 200_000, MaxLines: 10_000},
		Telegram: TelegramCfg{
			CoalesceMs:            1000,
			FirstRunDiscovery:     true,
			MemoryQueueCap:        1024,
			ExtractorBatchSize:    5,
			ExtractorPollInterval: 10 * time.Second,
			RecallEnabled:         true,
			RecallWeightThreshold: 1.0,
			RecallMaxFacts:        10,
			RecallDepth:           2,
		},
	}
}

func loadFile(cfg *Config) error {
	path := filepath.Join(xdgConfigHome(), "gormes", "config.toml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return toml.Unmarshal(data, cfg)
}

func loadEnv(cfg *Config) {
	if v := os.Getenv("GORMES_ENDPOINT"); v != "" {
		cfg.Hermes.Endpoint = v
	}
	if v := os.Getenv("GORMES_MODEL"); v != "" {
		cfg.Hermes.Model = v
	}
	if v := os.Getenv("GORMES_API_KEY"); v != "" {
		cfg.Hermes.APIKey = v
	}
	if v := os.Getenv("GORMES_TELEGRAM_TOKEN"); v != "" {
		cfg.Telegram.BotToken = v
	}
	if v := os.Getenv("GORMES_TELEGRAM_CHAT_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Telegram.AllowedChatID = id
		}
	}
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

// LogPath returns the default path for the Gormes log file.
func LogPath() string {
	return filepath.Join(xdgDataHome(), "gormes", "gormes.log")
}

// CrashLogDir returns the directory where TUI panic dumps are written.
func CrashLogDir() string {
	return filepath.Join(xdgDataHome(), "gormes")
}

// SessionDBPath returns the default location of the bbolt sessions map.
// Honors XDG_DATA_HOME; falls back to ~/.local/share/gormes/sessions.db.
func SessionDBPath() string {
	return filepath.Join(xdgDataHome(), "gormes", "sessions.db")
}

// MemoryDBPath returns the default location of the Phase-3.A SQLite
// memory database. Honors XDG_DATA_HOME; falls back to
// ~/.local/share/gormes/memory.db.
func MemoryDBPath() string {
	return filepath.Join(xdgDataHome(), "gormes", "memory.db")
}
