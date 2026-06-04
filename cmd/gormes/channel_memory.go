package main

import (
	"time"

	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	gonchotools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho"
)

// channelMemorySettings is the channel-neutral view of local memory settings.
// Today the public config source is still the legacy [telegram] section for
// compatibility, but channel runtimes should consume this neutral shape.
type channelMemorySettings struct {
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

func channelMemorySettingsFromConfig(cfg config.Config) channelMemorySettings {
	legacy := cfg.Telegram
	return channelMemorySettings{
		QueueCap:              legacy.MemoryQueueCap,
		ExtractorBatchSize:    legacy.ExtractorBatchSize,
		ExtractorPollInterval: legacy.ExtractorPollInterval,
		LegacyRecallEnabled:   legacy.RecallEnabled,
		Recall: memory.RecallConfig{
			WeightThreshold:       legacy.RecallWeightThreshold,
			MaxFacts:              legacy.RecallMaxFacts,
			Depth:                 legacy.RecallDepth,
			DecayHorizonDays:      legacy.RecallDecayHorizonDays,
			SemanticModel:         legacy.SemanticModel,
			SemanticTopK:          legacy.SemanticTopK,
			SemanticMinSimilarity: legacy.SemanticMinSimilarity,
			QueryEmbedTimeout:     legacy.QueryEmbedTimeout,
		},
		MirrorEnabled:        legacy.MirrorEnabled,
		MirrorPath:           legacy.MirrorPath,
		MirrorInterval:       legacy.MirrorInterval,
		SemanticEnabled:      legacy.SemanticEnabled,
		SemanticEndpoint:     legacy.SemanticEndpoint,
		SemanticModel:        legacy.SemanticModel,
		EmbedderPollInterval: legacy.EmbedderPollInterval,
		EmbedderBatchSize:    legacy.EmbedderBatchSize,
		EmbedderCallTimeout:  legacy.EmbedderCallTimeout,
	}
}

func (s channelMemorySettings) legacyRecallActive(peerConfigured bool) bool {
	return s.LegacyRecallEnabled && peerConfigured
}

func (s channelMemorySettings) semanticFusionActive(legacyRecallActive bool) bool {
	return legacyRecallActive && s.SemanticEnabled && s.SemanticModel != ""
}

// channelMemoryOptions describes channel-agnostic memory wiring for any edge
// runtime. Channel-specific policy, such as whether a legacy fallback is
// allowed, stays with the caller.
type channelMemoryOptions struct {
	GonchoEnabled       bool
	GonchoService       *gonchoservice.Service
	PeerID              string
	LegacyRecallEnabled bool
	LegacyRecall        func() kernel.RecallProvider
}

// channelMemoryProviders chooses the memory backend for channel runtimes.
// Goncho is the default memory path; legacy SQLite recall is retained only as
// an explicit caller-provided fallback when Goncho is disabled.
func channelMemoryProviders(opts channelMemoryOptions) (kernel.GonchoStore, kernel.RecallProvider) {
	if opts.GonchoEnabled && opts.GonchoService != nil {
		integration := gonchotools.NewTurnIntegration(opts.GonchoService, opts.PeerID)
		return newGonchoAdapter(opts.GonchoService), integration.RecallProvider()
	}
	if !opts.GonchoEnabled && opts.LegacyRecallEnabled && opts.LegacyRecall != nil {
		return nil, opts.LegacyRecall()
	}
	return nil, nil
}
