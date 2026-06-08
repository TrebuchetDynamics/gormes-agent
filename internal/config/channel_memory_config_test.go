package config

import (
	"testing"
	"time"
)

func TestChannelMemorySettingsProjectsLegacyTelegramMemoryFields(t *testing.T) {
	cfg := Config{
		Telegram: TelegramCfg{
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
	}

	settings := cfg.ChannelMemorySettings()
	if settings.QueueCap != 99 || settings.ExtractorBatchSize != 7 || settings.ExtractorPollInterval != 2*time.Second {
		t.Fatalf("projection queue/extractor = %#v", settings)
	}
	if !settings.LegacyRecallEnabled || settings.RecallWeightThreshold != 0.75 || settings.RecallMaxFacts != 8 || settings.RecallDepth != 3 || settings.RecallDecayHorizonDays != 45 {
		t.Fatalf("projection recall = %#v", settings)
	}
	if !settings.MirrorEnabled || settings.MirrorPath != "/tmp/user.md" || settings.MirrorInterval != 3*time.Second {
		t.Fatalf("projection mirror = %#v", settings)
	}
	if !settings.SemanticEnabled || settings.SemanticEndpoint != "http://embed" || settings.SemanticModel != "nomic-embed-text" || settings.SemanticTopK != 4 || settings.SemanticMinSimilarity != 0.42 {
		t.Fatalf("projection semantic = %#v", settings)
	}
	if settings.EmbedderPollInterval != 4*time.Second || settings.EmbedderBatchSize != 11 || settings.EmbedderCallTimeout != 5*time.Second || settings.QueryEmbedTimeout != 6*time.Millisecond {
		t.Fatalf("projection embedder = %#v", settings)
	}
}
