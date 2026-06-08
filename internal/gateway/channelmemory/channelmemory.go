package channelmemory

import (
	"time"

	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	gonchoadapter "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	gonchotools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho"
)

// Settings is the channel-neutral view of local memory settings.
// Today the public config source is still the legacy [telegram] section for
// compatibility, but channel runtimes should consume this neutral shape.
const ExtractorCallTimeout = 2 * time.Minute

type Settings struct {
	QueueCap              int
	ExtractorBatchSize    int
	ExtractorPollInterval time.Duration
	LegacyRecallEnabled   bool
	Recall                memory.RecallConfig
	MirrorEnabled         bool
	MirrorPath            string
	MirrorInterval        time.Duration
	SemanticEnabled       bool
	SemanticEndpoint      string
	SemanticModel         string
	EmbedderPollInterval  time.Duration
	EmbedderBatchSize     int
	EmbedderCallTimeout   time.Duration
}

func SettingsFromConfig(cfg config.Config) Settings {
	return SettingsFromProjection(cfg.ChannelMemorySettings())
}

func SettingsFromProjection(projection config.ChannelMemorySettings) Settings {
	return Settings{
		QueueCap:              projection.QueueCap,
		ExtractorBatchSize:    projection.ExtractorBatchSize,
		ExtractorPollInterval: projection.ExtractorPollInterval,
		LegacyRecallEnabled:   projection.LegacyRecallEnabled,
		Recall: memory.RecallConfig{
			WeightThreshold:       projection.RecallWeightThreshold,
			MaxFacts:              projection.RecallMaxFacts,
			Depth:                 projection.RecallDepth,
			DecayHorizonDays:      projection.RecallDecayHorizonDays,
			SemanticModel:         projection.SemanticModel,
			SemanticTopK:          projection.SemanticTopK,
			SemanticMinSimilarity: projection.SemanticMinSimilarity,
			QueryEmbedTimeout:     projection.QueryEmbedTimeout,
		},
		MirrorEnabled:        projection.MirrorEnabled,
		MirrorPath:           projection.MirrorPath,
		MirrorInterval:       projection.MirrorInterval,
		SemanticEnabled:      projection.SemanticEnabled,
		SemanticEndpoint:     projection.SemanticEndpoint,
		SemanticModel:        projection.SemanticModel,
		EmbedderPollInterval: projection.EmbedderPollInterval,
		EmbedderBatchSize:    projection.EmbedderBatchSize,
		EmbedderCallTimeout:  projection.EmbedderCallTimeout,
	}
}

func (s Settings) ExtractorConfig(model string) memory.ExtractorConfig {
	return memory.ExtractorConfig{
		Model:        model,
		BatchSize:    s.ExtractorBatchSize,
		PollInterval: s.ExtractorPollInterval,
		CallTimeout:  ExtractorCallTimeout,
	}
}

func (s Settings) LegacyRecallActive(peerConfigured bool) bool {
	return s.LegacyRecallEnabled && peerConfigured
}

func (s Settings) SemanticFusionActive(legacyRecallActive bool) bool {
	return legacyRecallActive && s.SemanticEnabled && s.SemanticModel != ""
}

// Options describes channel-agnostic memory wiring for any edge runtime.
// Channel-specific policy, such as whether a legacy fallback is allowed, stays
// with the caller.
type Options struct {
	GonchoEnabled       bool
	GonchoService       *gonchoservice.Service
	PeerID              string
	LegacyRecallEnabled bool
	LegacyRecall        func() kernel.RecallProvider
}

// Providers chooses the memory backend for channel runtimes.
// Goncho is the default memory path; legacy SQLite recall is retained only as
// an explicit caller-provided fallback when Goncho is disabled.
func Providers(opts Options) (kernel.GonchoStore, kernel.RecallProvider) {
	if opts.GonchoEnabled && opts.GonchoService != nil {
		integration := gonchotools.NewTurnIntegration(opts.GonchoService, opts.PeerID)
		return gonchoadapter.NewStore(opts.GonchoService), integration.RecallProvider()
	}
	if !opts.GonchoEnabled && opts.LegacyRecallEnabled && opts.LegacyRecall != nil {
		return nil, opts.LegacyRecall()
	}
	return nil, nil
}
