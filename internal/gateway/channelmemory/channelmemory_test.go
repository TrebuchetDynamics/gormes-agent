package channelmemory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	gonchoconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestSettingsFromProjectionAreChannelNeutral(t *testing.T) {
	settings := SettingsFromProjection(config.ChannelMemorySettings{
		QueueCap:               99,
		ExtractorBatchSize:     7,
		ExtractorPollInterval:  2 * time.Second,
		LegacyRecallEnabled:    true,
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
	})

	if settings.QueueCap != 99 || settings.ExtractorBatchSize != 7 || settings.ExtractorPollInterval != 2*time.Second {
		t.Fatalf("settings queue/extractor = %#v", settings)
	}
	extractor := settings.ExtractorConfig("gpt-test")
	if extractor.Model != "gpt-test" || extractor.BatchSize != 7 || extractor.PollInterval != 2*time.Second || extractor.CallTimeout != ExtractorCallTimeout {
		t.Fatalf("ExtractorConfig = %#v, want channel settings with extended call timeout", extractor)
	}
	if !settings.LegacyRecallActive(true) || settings.LegacyRecallActive(false) {
		t.Fatalf("LegacyRecallActive did not stay caller-scoped")
	}
	if !settings.SemanticFusionActive(true) || settings.SemanticFusionActive(false) {
		t.Fatalf("SemanticFusionActive did not require active legacy recall")
	}
	if settings.Recall.WeightThreshold != 0.75 || settings.Recall.MaxFacts != 8 || settings.Recall.Depth != 3 {
		t.Fatalf("settings recall = %#v", settings.Recall)
	}
	if settings.MirrorPath != "/tmp/user.md" || settings.SemanticModel != "nomic-embed-text" || settings.EmbedderBatchSize != 11 {
		t.Fatalf("settings mirror/semantic = %#v", settings)
	}
}

func TestSemanticFusionRequiresNonBlankSemanticModel(t *testing.T) {
	settings := Settings{SemanticEnabled: true, SemanticModel: " \t\n "}
	if settings.SemanticFusionActive(true) {
		t.Fatal("SemanticFusionActive = true with blank semantic model, want false")
	}
}

func TestDefaultMemoryUsesGonchoInsteadOfLegacyRecall(t *testing.T) {
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = store.Close(ctx)
	}()

	gcfg := gonchoconfig.DefaultConfig()
	svc := goncho.NewService(store.DB(), gcfg.RuntimeConfig(), nil)
	legacyCalled := false

	gonchoStore, recall := Providers(Options{
		GonchoEnabled:       true,
		GonchoService:       svc,
		PeerID:              "channel:42",
		LegacyRecallEnabled: true,
		LegacyRecall: func() kernel.RecallProvider {
			legacyCalled = true
			return testRecall{}
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

func TestMemoryDoesNotFallbackToLegacyWhileGonchoEnabled(t *testing.T) {
	legacyCalled := false

	gonchoStore, recall := Providers(Options{
		GonchoEnabled:       true,
		LegacyRecallEnabled: true,
		LegacyRecall: func() kernel.RecallProvider {
			legacyCalled = true
			return testRecall{}
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

func TestLegacyRecallOnlyWhenGonchoDisabled(t *testing.T) {
	legacy := testRecall{}
	legacyCalled := false

	gonchoStore, recall := Providers(Options{
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

type testRecall struct{}

func (testRecall) GetContext(context.Context, kernel.RecallParams) string {
	return "legacy memory"
}
