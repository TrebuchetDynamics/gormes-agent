package config

import "time"

// ChannelMemorySettings is the channel-neutral projection of memory/extractor
// configuration consumed by gateway channel memory wiring.
//
// The public TOML/env compatibility source is still the legacy [telegram]
// fields, but callers outside config should depend on this narrow shape rather
// than re-reading individual Telegram memory keys.
type ChannelMemorySettings struct {
	QueueCap               int
	ExtractorBatchSize     int
	ExtractorPollInterval  time.Duration
	LegacyRecallEnabled    bool
	RecallWeightThreshold  float64
	RecallMaxFacts         int
	RecallDepth            int
	RecallDecayHorizonDays int
	MirrorEnabled          bool
	MirrorPath             string
	MirrorInterval         time.Duration
	SemanticEnabled        bool
	SemanticEndpoint       string
	SemanticModel          string
	SemanticTopK           int
	SemanticMinSimilarity  float64
	EmbedderPollInterval   time.Duration
	EmbedderBatchSize      int
	EmbedderCallTimeout    time.Duration
	QueryEmbedTimeout      time.Duration
}

// ChannelMemorySettings projects Config into the channel-neutral memory shape.
func (c Config) ChannelMemorySettings() ChannelMemorySettings {
	legacy := c.Telegram
	return ChannelMemorySettings{
		QueueCap:               legacy.MemoryQueueCap,
		ExtractorBatchSize:     legacy.ExtractorBatchSize,
		ExtractorPollInterval:  legacy.ExtractorPollInterval,
		LegacyRecallEnabled:    legacy.RecallEnabled,
		RecallWeightThreshold:  legacy.RecallWeightThreshold,
		RecallMaxFacts:         legacy.RecallMaxFacts,
		RecallDepth:            legacy.RecallDepth,
		RecallDecayHorizonDays: legacy.RecallDecayHorizonDays,
		MirrorEnabled:          legacy.MirrorEnabled,
		MirrorPath:             legacy.MirrorPath,
		MirrorInterval:         legacy.MirrorInterval,
		SemanticEnabled:        legacy.SemanticEnabled,
		SemanticEndpoint:       legacy.SemanticEndpoint,
		SemanticModel:          legacy.SemanticModel,
		SemanticTopK:           legacy.SemanticTopK,
		SemanticMinSimilarity:  legacy.SemanticMinSimilarity,
		EmbedderPollInterval:   legacy.EmbedderPollInterval,
		EmbedderBatchSize:      legacy.EmbedderBatchSize,
		EmbedderCallTimeout:    legacy.EmbedderCallTimeout,
		QueryEmbedTimeout:      legacy.QueryEmbedTimeout,
	}
}
