package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	gonchoconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestChannelMemorySettingsFromConfigAreChannelNeutral(t *testing.T) {
	settings := channelMemorySettingsFromConfig(config.Config{
		Telegram: config.TelegramCfg{
			MemoryQueueCap:         99,
			ExtractorBatchSize:     7,
			ExtractorPollInterval:  2 * time.Second,
			RecallEnabled:          true,
			RecallWeightThreshold:  0.75,
			RecallMaxFacts:         8,
			RecallDepth:            3,
			RecallDecayHorizonDays: 45,
			MirrorEnabled:          true,
			MirrorPath:             "/tmp/user.md",
			MirrorInterval:         3 * time.Second,
			SemanticEnabled:        true,
			SemanticEndpoint:       "http://embed",
			SemanticModel:          "nomic-embed-text",
			SemanticTopK:           4,
			SemanticMinSimilarity:  0.42,
			EmbedderPollInterval:   4 * time.Second,
			EmbedderBatchSize:      11,
			EmbedderCallTimeout:    5 * time.Second,
			QueryEmbedTimeout:      6 * time.Millisecond,
		},
	})

	if settings.QueueCap != 99 || settings.ExtractorBatchSize != 7 || settings.ExtractorPollInterval != 2*time.Second {
		t.Fatalf("settings queue/extractor = %#v", settings)
	}
	if !settings.legacyRecallActive(true) || settings.legacyRecallActive(false) {
		t.Fatalf("legacyRecallActive did not stay caller-scoped")
	}
	if !settings.semanticFusionActive(true) || settings.semanticFusionActive(false) {
		t.Fatalf("semanticFusionActive did not require active legacy recall")
	}
	if settings.Recall.WeightThreshold != 0.75 || settings.Recall.MaxFacts != 8 || settings.Recall.Depth != 3 {
		t.Fatalf("settings recall = %#v", settings.Recall)
	}
	if settings.MirrorPath != "/tmp/user.md" || settings.SemanticModel != "nomic-embed-text" || settings.EmbedderBatchSize != 11 {
		t.Fatalf("settings mirror/semantic = %#v", settings)
	}
}

func TestChannelDefaultMemoryUsesGonchoInsteadOfLegacyRecall(t *testing.T) {
	db, err := sqlOpenGoncho(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer db.Close()

	gcfg := gonchoconfig.DefaultConfig()
	svc := goncho.NewService(db, gcfg.RuntimeConfig(), nil)
	legacyCalled := false

	gonchoStore, recall := channelMemoryProviders(channelMemoryOptions{
		GonchoEnabled:       true,
		GonchoService:       svc,
		PeerID:              "channel:42",
		LegacyRecallEnabled: true,
		LegacyRecall: func() kernel.RecallProvider {
			legacyCalled = true
			return channelMemoryTestRecall{}
		},
	})

	if gonchoStore == nil {
		t.Fatal("channel default memory did not install Goncho store")
	}
	if recall == nil {
		t.Fatal("channel default memory did not install Goncho recall provider")
	}
	if legacyCalled {
		t.Fatal("channel default memory constructed legacy SQLite recall; want Goncho-only default")
	}
}

func TestChannelMemoryDoesNotFallbackToLegacyWhileGonchoEnabled(t *testing.T) {
	legacyCalled := false

	gonchoStore, recall := channelMemoryProviders(channelMemoryOptions{
		GonchoEnabled:       true,
		LegacyRecallEnabled: true,
		LegacyRecall: func() kernel.RecallProvider {
			legacyCalled = true
			return channelMemoryTestRecall{}
		},
	})

	if gonchoStore != nil {
		t.Fatalf("nil Goncho service produced store %#v", gonchoStore)
	}
	if recall != nil {
		t.Fatalf("enabled Goncho with unavailable service produced legacy recall %#v", recall)
	}
	if legacyCalled {
		t.Fatal("enabled Goncho constructed legacy recall fallback; want explicit Goncho disabled opt-out")
	}
}

func TestChannelLegacyRecallOnlyWhenGonchoDisabled(t *testing.T) {
	legacy := channelMemoryTestRecall{}
	legacyCalled := false

	gonchoStore, recall := channelMemoryProviders(channelMemoryOptions{
		GonchoEnabled:       false,
		LegacyRecallEnabled: true,
		LegacyRecall: func() kernel.RecallProvider {
			legacyCalled = true
			return legacy
		},
	})

	if gonchoStore != nil {
		t.Fatalf("disabled Goncho produced store %#v", gonchoStore)
	}
	if !legacyCalled {
		t.Fatal("disabled Goncho did not fall back to explicit legacy recall")
	}
	if recall != legacy {
		t.Fatalf("recall = %#v, want injected legacy recall", recall)
	}
}

type channelMemoryTestRecall struct{}

func (channelMemoryTestRecall) GetContext(context.Context, kernel.RecallParams) string {
	return "legacy memory"
}
