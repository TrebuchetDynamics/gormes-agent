package goncho

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultConfigMatchesRuntimeDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("Enabled default = false, want true")
	}
	if cfg.Workspace != "gormes" {
		t.Errorf("Workspace default = %q, want gormes", cfg.Workspace)
	}
	if cfg.ObserverPeer != "gormes" {
		t.Errorf("ObserverPeer default = %q, want gormes", cfg.ObserverPeer)
	}
	if cfg.RecentMessages != 4 {
		t.Errorf("RecentMessages default = %d, want 4", cfg.RecentMessages)
	}
	if cfg.MaxMessageSize != 25_000 {
		t.Errorf("MaxMessageSize default = %d, want 25000", cfg.MaxMessageSize)
	}
	if cfg.MaxFileSize != 5_242_880 {
		t.Errorf("MaxFileSize default = %d, want 5242880", cfg.MaxFileSize)
	}
	if cfg.GetContextMaxTokens != 100_000 {
		t.Errorf("GetContextMaxTokens default = %d, want 100000", cfg.GetContextMaxTokens)
	}
	if !cfg.ReasoningEnabled || !cfg.PeerCardEnabled || !cfg.SummaryEnabled {
		t.Errorf("reasoning/peer card/summary defaults = %t/%t/%t, want true", cfg.ReasoningEnabled, cfg.PeerCardEnabled, cfg.SummaryEnabled)
	}
	if cfg.DreamEnabled {
		t.Error("DreamEnabled default = true, want false until fixtures exist")
	}
	if cfg.DreamIdleTimeoutMinutes != 60 {
		t.Errorf("DreamIdleTimeoutMinutes default = %d, want 60", cfg.DreamIdleTimeoutMinutes)
	}
	if cfg.DeriverWorkers != 1 {
		t.Errorf("DeriverWorkers default = %d, want 1", cfg.DeriverWorkers)
	}
	if cfg.RepresentationBatchMaxTokens != 1024 {
		t.Errorf("RepresentationBatchMaxTokens default = %d, want 1024", cfg.RepresentationBatchMaxTokens)
	}
	if cfg.DialecticDefaultLevel != "low" {
		t.Errorf("DialecticDefaultLevel default = %q, want low", cfg.DialecticDefaultLevel)
	}
}

func TestNormalizeAndValidateRejectsInvalidDialecticAndLimits(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DialecticDefaultLevel = " EXTREME "
	if err := cfg.NormalizeAndValidate(); err == nil || !strings.Contains(err.Error(), "goncho.dialectic_default_level") {
		t.Fatalf("NormalizeAndValidate invalid dialectic error = %v, want dialectic error", err)
	}

	for _, tc := range []struct {
		name  string
		field string
		set   func(*GonchoCfg)
	}{
		{name: "recent messages", field: "recent_messages", set: func(c *GonchoCfg) { c.RecentMessages = -1 }},
		{name: "max message size", field: "max_message_size", set: func(c *GonchoCfg) { c.MaxMessageSize = -1 }},
		{name: "max file size", field: "max_file_size", set: func(c *GonchoCfg) { c.MaxFileSize = -1 }},
		{name: "context max tokens", field: "get_context_max_tokens", set: func(c *GonchoCfg) { c.GetContextMaxTokens = -1 }},
		{name: "dream idle timeout", field: "dream_idle_timeout_minutes", set: func(c *GonchoCfg) { c.DreamIdleTimeoutMinutes = -1 }},
		{name: "deriver workers", field: "deriver_workers", set: func(c *GonchoCfg) { c.DeriverWorkers = -1 }},
		{name: "representation batch max tokens", field: "representation_batch_max_tokens", set: func(c *GonchoCfg) { c.RepresentationBatchMaxTokens = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tc.set(&cfg)
			err := cfg.NormalizeAndValidate()
			if err == nil || !strings.Contains(err.Error(), "goncho."+tc.field) {
				t.Fatalf("NormalizeAndValidate error = %v, want goncho.%s", err, tc.field)
			}
		})
	}
}

func TestRuntimeConfigMapsAllFields(t *testing.T) {
	cfg := GonchoCfg{
		Enabled:                      true,
		Workspace:                    "runtime-workspace",
		ObserverPeer:                 "runtime-observer",
		RecentMessages:               8,
		MaxMessageSize:               12_345,
		MaxFileSize:                  67_890,
		GetContextMaxTokens:          555,
		ReasoningEnabled:             false,
		PeerCardEnabled:              false,
		SummaryEnabled:               false,
		DreamEnabled:                 true,
		DreamIdleTimeoutMinutes:      90,
		DeriverWorkers:               4,
		RepresentationBatchMaxTokens: 777,
		DialecticDefaultLevel:        "medium",
	}
	rt := cfg.RuntimeConfig()

	if rt.WorkspaceID != "runtime-workspace" || rt.ObserverPeerID != "runtime-observer" {
		t.Fatalf("runtime identity = %q/%q", rt.WorkspaceID, rt.ObserverPeerID)
	}
	if rt.RecentMessages != 8 || rt.MaxMessageSize != 12_345 || rt.MaxFileSize != 67_890 || rt.GetContextMaxTokens != 555 {
		t.Fatalf("runtime limits = %+v", rt)
	}
	if rt.ReasoningEnabled || rt.PeerCardEnabled || rt.SummaryEnabled || !rt.DreamEnabled {
		t.Fatalf("runtime booleans = reasoning:%t peer:%t summary:%t dream:%t", rt.ReasoningEnabled, rt.PeerCardEnabled, rt.SummaryEnabled, rt.DreamEnabled)
	}
	if rt.DreamIdleTimeout != 90*time.Minute || rt.DeriverWorkers != 4 || rt.RepresentationBatchMaxTokens != 777 || string(rt.DialecticDefaultLevel) != "medium" {
		t.Fatalf("runtime derived fields = %+v", rt)
	}
}
