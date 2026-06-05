package goncho

import (
	"fmt"
	"strings"
	"time"

	service "github.com/TrebuchetDynamics/goncho/service"
)

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

func DefaultConfig() GonchoCfg {
	return GonchoCfg{
		Enabled:                      true,
		Workspace:                    service.DefaultWorkspaceID,
		ObserverPeer:                 service.DefaultObserverPeerID,
		RecentMessages:               service.DefaultRecentMessages,
		MaxMessageSize:               service.DefaultMaxMessageSize,
		MaxFileSize:                  service.DefaultMaxFileSize,
		GetContextMaxTokens:          service.DefaultGetContextMaxTokens,
		ReasoningEnabled:             true,
		PeerCardEnabled:              true,
		SummaryEnabled:               true,
		DreamEnabled:                 false,
		DreamIdleTimeoutMinutes:      int(service.DefaultDreamIdleTimeout / time.Minute),
		DeriverWorkers:               service.DefaultDeriverWorkers,
		RepresentationBatchMaxTokens: service.DefaultRepresentationBatchMaxTokens,
		DialecticDefaultLevel:        string(service.DialecticLevelLow),
	}
}

func (g *GonchoCfg) NormalizeAndValidate() error {
	g.Workspace = strings.TrimSpace(g.Workspace)
	g.ObserverPeer = strings.TrimSpace(g.ObserverPeer)
	g.DialecticDefaultLevel = strings.ToLower(strings.TrimSpace(g.DialecticDefaultLevel))

	if g.Workspace == "" {
		return fmt.Errorf("config: goncho.workspace is required")
	}
	if g.ObserverPeer == "" {
		return fmt.Errorf("config: goncho.observer_peer is required")
	}
	if !service.ValidDialecticLevel(g.DialecticDefaultLevel) {
		return fmt.Errorf("config: goncho.dialectic_default_level %q is invalid; want one of minimal, low, medium, high, max", g.DialecticDefaultLevel)
	}
	for _, limit := range []struct {
		name  string
		value int
	}{
		{name: "recent_messages", value: g.RecentMessages},
		{name: "max_message_size", value: g.MaxMessageSize},
		{name: "max_file_size", value: g.MaxFileSize},
		{name: "get_context_max_tokens", value: g.GetContextMaxTokens},
		{name: "dream_idle_timeout_minutes", value: g.DreamIdleTimeoutMinutes},
		{name: "deriver_workers", value: g.DeriverWorkers},
		{name: "representation_batch_max_tokens", value: g.RepresentationBatchMaxTokens},
	} {
		if limit.value < 0 {
			return fmt.Errorf("config: goncho.%s must be non-negative, got %d", limit.name, limit.value)
		}
	}
	if g.DeriverWorkers == 0 {
		return fmt.Errorf("config: goncho.deriver_workers must be at least 1")
	}
	return nil
}

func (g GonchoCfg) RuntimeConfig() service.Config {
	return service.Config{
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
		DialecticDefaultLevel:        service.DialecticLevel(g.DialecticDefaultLevel),
	}
}
