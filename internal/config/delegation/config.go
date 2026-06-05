package delegation

import (
	"fmt"
	"time"

	"github.com/pelletier/go-toml/v2"
)

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
