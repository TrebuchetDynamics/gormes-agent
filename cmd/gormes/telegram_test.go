package main

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestTelegramManagerConfig_LiveTurnMetadataProductionWiring(t *testing.T) {
	mgrCfg := telegramManagerConfig(
		config.Config{Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		}},
		nil,
	)
	if mgrCfg.LiveTurnNow == nil {
		t.Fatal("LiveTurnNow is nil; production telegram metadata block would omit timestamp")
	}
	if got := mgrCfg.LiveTurnActiveModel; got == nil || got() != "gpt-5.5" {
		if got == nil {
			t.Fatal("LiveTurnActiveModel is nil")
		}
		t.Fatalf("LiveTurnActiveModel() = %q, want configured model", got())
	}
	if got := mgrCfg.LiveTurnActiveProvider; got == nil || got() != "openai-codex" {
		if got == nil {
			t.Fatal("LiveTurnActiveProvider is nil")
		}
		t.Fatalf("LiveTurnActiveProvider() = %q, want configured provider", got())
	}
}
