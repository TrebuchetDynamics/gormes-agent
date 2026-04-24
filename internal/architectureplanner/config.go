package architectureplanner

import (
	"fmt"
	"path/filepath"
)

type Config struct {
	RepoRoot     string
	ProgressJSON string
	RunRoot      string
	Backend      string
	Mode         string
	HermesDir    string
	GBrainDir    string
	HonchoDir    string
	Validate     bool
}

func ConfigFromEnv(repoRoot string, env map[string]string) (Config, error) {
	if repoRoot == "" {
		return Config{}, fmt.Errorf("repo root is required")
	}

	parent := filepath.Dir(repoRoot)
	cfg := Config{
		RepoRoot:     repoRoot,
		ProgressJSON: filepath.Join(repoRoot, "docs", "content", "building-gormes", "architecture_plan", "progress.json"),
		RunRoot:      filepath.Join(repoRoot, ".codex", "architecture-planner"),
		Backend:      "codexu",
		Mode:         "safe",
		HermesDir:    filepath.Join(parent, "hermes-agent"),
		GBrainDir:    filepath.Join(parent, "gbrain"),
		HonchoDir:    filepath.Join(parent, "honcho"),
		Validate:     true,
	}

	if value := env["PROGRESS_JSON"]; value != "" {
		cfg.ProgressJSON = value
	}
	if value := env["RUN_ROOT"]; value != "" {
		cfg.RunRoot = value
	}
	if value := env["BACKEND"]; value != "" {
		cfg.Backend = value
	}
	if value := env["MODE"]; value != "" {
		cfg.Mode = value
	}
	if value := env["HERMES_DIR"]; value != "" {
		cfg.HermesDir = value
	}
	if value := env["GBRAIN_DIR"]; value != "" {
		cfg.GBrainDir = value
	}
	if value := env["HONCHO_DIR"]; value != "" {
		cfg.HonchoDir = value
	}
	if value := env["PLANNER_VALIDATE"]; value == "0" {
		cfg.Validate = false
	}

	return cfg, nil
}

func (cfg Config) SourceRoots() []SourceRoot {
	return []SourceRoot{
		{Name: "hermes-agent", Path: cfg.HermesDir},
		{Name: "gbrain", Path: cfg.GBrainDir},
		{Name: "honcho", Path: cfg.HonchoDir},
		{Name: "upstream-hermes", Path: filepath.Join(cfg.RepoRoot, "docs", "content", "upstream-hermes")},
		{Name: "upstream-gbrain", Path: filepath.Join(cfg.RepoRoot, "docs", "content", "upstream-gbrain")},
		{Name: "building-gormes", Path: filepath.Join(cfg.RepoRoot, "docs", "content", "building-gormes")},
	}
}
