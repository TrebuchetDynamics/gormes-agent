package transcript

import (
	"strings"
	"unicode/utf8"
)

const defaultTrajectorySummaryNotice = "\n\nSome of your previous tool responses may be summarized to preserve context."

// TrajectoryTurn is the Hermes trajectory row shape used by the offline
// compressor: {"from": role, "value": content}.
type TrajectoryTurn struct {
	From  string `json:"from"`
	Value string `json:"value"`
}

// TrajectoryCompressionConfig controls the pure compression algorithm. It
// intentionally excludes provider, filesystem, and batch-processing settings.
type TrajectoryCompressionConfig struct {
	TargetMaxTokens     int
	SummaryTargetTokens int

	ProtectFirstSystem bool
	ProtectFirstHuman  bool
	ProtectFirstGPT    bool
	ProtectFirstTool   bool
	ProtectLastNTurns  int

	AddSummaryNotice  bool
	SummaryNoticeText string
}

// TrajectoryCompressionMetrics mirrors the upstream compressor's per-trajectory
// accounting fields.
type TrajectoryCompressionMetrics struct {
	OriginalTokens   int
	CompressedTokens int
	TokensSaved      int
	CompressionRatio float64

	OriginalTurns   int
	CompressedTurns int
	TurnsRemoved    int

	TurnsCompressedStartIdx int
	TurnsCompressedEndIdx   int
	TurnsInCompressedRegion int

	WasCompressed      bool
	StillOverLimit     bool
	SkippedUnderTarget bool
}

// TrajectoryTokenCounter counts a string for compression budgeting.
type TrajectoryTokenCounter func(string) int

// DefaultTrajectoryCompressionConfig returns the Hermes default compressor
// knobs used for protected-turn and summary-size decisions.
func DefaultTrajectoryCompressionConfig() TrajectoryCompressionConfig {
	return TrajectoryCompressionConfig{
		TargetMaxTokens:     15250,
		SummaryTargetTokens: 750,
		ProtectFirstSystem:  true,
		ProtectFirstHuman:   true,
		ProtectFirstGPT:     true,
		ProtectFirstTool:    true,
		ProtectLastNTurns:   4,
		AddSummaryNotice:    true,
		SummaryNoticeText:   defaultTrajectorySummaryNotice,
	}
}

// FindTrajectoryProtectedIndices returns protected row indexes and the
// compressible middle range [start,end), matching trajectory_compressor.py's
// _find_protected_indices behavior.
func FindTrajectoryProtectedIndices(turns []TrajectoryTurn, cfg TrajectoryCompressionConfig) (map[int]bool, int, int) {
	cfg = normalizeTrajectoryCompressionConfig(cfg)
	n := len(turns)
	protected := make(map[int]bool)
	firstSystem, firstHuman, firstGPT, firstTool := -1, -1, -1, -1
	for i, turn := range turns {
		switch turn.From {
		case "system":
			if firstSystem == -1 {
				firstSystem = i
			}
		case "human":
			if firstHuman == -1 {
				firstHuman = i
			}
		case "gpt":
			if firstGPT == -1 {
				firstGPT = i
			}
		case "tool":
			if firstTool == -1 {
				firstTool = i
			}
		}
	}
	if cfg.ProtectFirstSystem && firstSystem >= 0 {
		protected[firstSystem] = true
	}
	if cfg.ProtectFirstHuman && firstHuman >= 0 {
		protected[firstHuman] = true
	}
	if cfg.ProtectFirstGPT && firstGPT >= 0 {
		protected[firstGPT] = true
	}
	if cfg.ProtectFirstTool && firstTool >= 0 {
		protected[firstTool] = true
	}
	if cfg.ProtectLastNTurns > 0 {
		for i := max(0, n-cfg.ProtectLastNTurns); i < n; i++ {
			protected[i] = true
		}
	}

	midpoint := n / 2
	headMax := -1
	tailMin := n + 1
	for idx := range protected {
		if idx < midpoint && idx > headMax {
			headMax = idx
		}
		if idx >= midpoint && idx < tailMin {
			tailMin = idx
		}
	}
	start := 0
	if headMax >= 0 {
		start = headMax + 1
	}
	end := n
	if tailMin <= n {
		end = tailMin
	}
	return protected, start, end
}

// ExtractTrajectorySummaryContent formats the middle turns that will be sent
// to a summarizer. Long values follow Hermes' 1500-head/500-tail truncation.
func ExtractTrajectorySummaryContent(turns []TrajectoryTurn, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(turns) {
		end = len(turns)
	}
	if start >= end {
		return ""
	}
	parts := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		role := turns[i].From
		if role == "" {
			role = "unknown"
		}
		value := truncateTrajectorySummaryValue(turns[i].Value)
		parts = append(parts, "[Turn "+itoa(i)+" - "+strings.ToUpper(role)+"]:\n"+value)
	}
	return strings.Join(parts, "\n\n")
}

// EnsureTrajectorySummaryPrefix ensures the replacement turn starts with the
// exact prefix expected by downstream context consumers.
func EnsureTrajectorySummaryPrefix(summary string) string {
	text := strings.TrimSpace(summary)
	if strings.HasPrefix(text, "[CONTEXT SUMMARY]:") {
		return text
	}
	if text == "" {
		return "[CONTEXT SUMMARY]:"
	}
	return "[CONTEXT SUMMARY]: " + text
}

// CompressTrajectory applies the protected-middle compression plan with a
// caller-supplied summary. It never mutates the input turns.
func CompressTrajectory(turns []TrajectoryTurn, cfg TrajectoryCompressionConfig, summary string, counter TrajectoryTokenCounter) ([]TrajectoryTurn, TrajectoryCompressionMetrics) {
	cfg = normalizeTrajectoryCompressionConfig(cfg)
	if counter == nil {
		counter = RoughTrajectoryTokenCount
	}
	metrics := TrajectoryCompressionMetrics{OriginalTurns: len(turns)}
	turnTokens := CountTrajectoryTurnTokens(turns, counter)
	totalTokens := sumInts(turnTokens)
	metrics.OriginalTokens = totalTokens

	if totalTokens <= cfg.TargetMaxTokens {
		metrics.SkippedUnderTarget = true
		metrics.CompressedTokens = totalTokens
		metrics.CompressedTurns = len(turns)
		metrics.CompressionRatio = 1.0
		return cloneTrajectoryTurns(turns), metrics
	}

	_, start, end := FindTrajectoryProtectedIndices(turns, cfg)
	if start >= end {
		metrics.CompressedTokens = totalTokens
		metrics.CompressedTurns = len(turns)
		metrics.CompressionRatio = 1.0
		metrics.StillOverLimit = totalTokens > cfg.TargetMaxTokens
		return cloneTrajectoryTurns(turns), metrics
	}

	tokensToSave := totalTokens - cfg.TargetMaxTokens
	targetTokensToCompress := tokensToSave + cfg.SummaryTargetTokens
	accumulated := 0
	compressUntil := start
	for i := start; i < end; i++ {
		accumulated += turnTokens[i]
		compressUntil = i + 1
		if accumulated >= targetTokensToCompress {
			break
		}
	}
	if accumulated < targetTokensToCompress && compressUntil < end {
		compressUntil = end
		accumulated = sumInts(turnTokens[start:end])
	}

	metrics.TurnsCompressedStartIdx = start
	metrics.TurnsCompressedEndIdx = compressUntil
	metrics.TurnsInCompressedRegion = compressUntil - start

	out := make([]TrajectoryTurn, 0, len(turns)-(compressUntil-start)+1)
	for i := 0; i < start; i++ {
		turn := turns[i]
		if turn.From == "system" && cfg.AddSummaryNotice {
			turn.Value += cfg.SummaryNoticeText
		}
		out = append(out, turn)
	}
	out = append(out, TrajectoryTurn{From: "human", Value: EnsureTrajectorySummaryPrefix(summary)})
	for i := compressUntil; i < len(turns); i++ {
		out = append(out, turns[i])
	}

	metrics.CompressedTurns = len(out)
	metrics.CompressedTokens = CountTrajectoryTokens(out, counter)
	metrics.TurnsRemoved = metrics.OriginalTurns - metrics.CompressedTurns
	metrics.TokensSaved = metrics.OriginalTokens - metrics.CompressedTokens
	metrics.CompressionRatio = float64(metrics.CompressedTokens) / float64(max(metrics.OriginalTokens, 1))
	metrics.WasCompressed = true
	metrics.StillOverLimit = metrics.CompressedTokens > cfg.TargetMaxTokens
	return out, metrics
}

// CountTrajectoryTurnTokens counts each turn's Value field independently.
func CountTrajectoryTurnTokens(turns []TrajectoryTurn, counter TrajectoryTokenCounter) []int {
	if counter == nil {
		counter = RoughTrajectoryTokenCount
	}
	out := make([]int, len(turns))
	for i, turn := range turns {
		out[i] = counter(turn.Value)
	}
	return out
}

// CountTrajectoryTokens sums turn token estimates.
func CountTrajectoryTokens(turns []TrajectoryTurn, counter TrajectoryTokenCounter) int {
	return sumInts(CountTrajectoryTurnTokens(turns, counter))
}

// RoughTrajectoryTokenCount is a conservative no-tokenizer fallback for tests
// and offline diagnostics.
func RoughTrajectoryTokenCount(text string) int {
	if text == "" {
		return 0
	}
	return utf8.RuneCountInString(text) / 4
}

func normalizeTrajectoryCompressionConfig(cfg TrajectoryCompressionConfig) TrajectoryCompressionConfig {
	defaults := DefaultTrajectoryCompressionConfig()
	if cfg.TargetMaxTokens <= 0 {
		cfg.TargetMaxTokens = defaults.TargetMaxTokens
	}
	if cfg.SummaryTargetTokens <= 0 {
		cfg.SummaryTargetTokens = defaults.SummaryTargetTokens
	}
	if cfg.SummaryNoticeText == "" {
		cfg.SummaryNoticeText = defaults.SummaryNoticeText
	}
	return cfg
}

func truncateTrajectorySummaryValue(value string) string {
	if len(value) <= 3000 {
		return value
	}
	return value[:1500] + "\n...[truncated]...\n" + value[len(value)-500:]
}

func cloneTrajectoryTurns(in []TrajectoryTurn) []TrajectoryTurn {
	out := make([]TrajectoryTurn, len(in))
	copy(out, in)
	return out
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
