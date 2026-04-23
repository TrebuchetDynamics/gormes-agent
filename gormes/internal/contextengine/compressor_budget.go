package contextengine

const (
	DefaultThresholdPercent              = 0.50
	DefaultTargetRatio                   = 0.20
	DefaultFallbackContext               = 128_000
	MinimumContextLength                 = 64_000
	maxSummaryContextFraction            = 0.05
	minSummaryTokens                     = 2_000
	summaryRatio                         = 0.20
	summaryTokensCeiling                 = 12_000
	ineffectiveCompressionSavingsPercent = 10.0
	ineffectiveCompressionLimit          = 2
)

var contextProbeTiers = []int{128_000, 64_000, 32_000, 16_000, 8_000}

type Config struct {
	ContextLength    int
	ThresholdPercent float64
	TargetRatio      float64
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Compressor carries the provider-free budget math needed before the later
// context-engine and summarizer slices get wired into the kernel loop.
type Compressor struct {
	contextLength               int
	thresholdPercent            float64
	targetRatio                 float64
	thresholdTokens             int
	tailTokenBudget             int
	maxSummaryTokens            int
	lastPromptTokens            int
	lastCompletionTokens        int
	compressionCount            int
	contextProbed               bool
	lastCompressionSavingsPct   float64
	ineffectiveCompressionCount int
}

func NewCompressor(cfg Config) *Compressor {
	thresholdPercent := cfg.ThresholdPercent
	if thresholdPercent <= 0 {
		thresholdPercent = DefaultThresholdPercent
	}
	targetRatio := cfg.TargetRatio
	if targetRatio <= 0 {
		targetRatio = DefaultTargetRatio
	}
	targetRatio = clamp(targetRatio, 0.10, 0.80)

	contextLength := cfg.ContextLength
	if contextLength <= 0 {
		contextLength = DefaultFallbackContext
	}

	c := &Compressor{
		contextLength:             contextLength,
		thresholdPercent:          thresholdPercent,
		targetRatio:               targetRatio,
		lastCompressionSavingsPct: 100.0,
	}
	c.recomputeBudgets()
	return c
}

func (c *Compressor) ContextLength() int { return c.contextLength }

func (c *Compressor) ThresholdTokens() int { return c.thresholdTokens }

func (c *Compressor) TailTokenBudget() int { return c.tailTokenBudget }

func (c *Compressor) MaxSummaryTokens() int { return c.maxSummaryTokens }

func (c *Compressor) ContextProbed() bool { return c.contextProbed }

func (c *Compressor) CompressionCount() int { return c.compressionCount }

func (c *Compressor) IneffectiveCompressionCount() int { return c.ineffectiveCompressionCount }

func (c *Compressor) UpdateFromResponse(usage Usage) {
	c.lastPromptTokens = usage.PromptTokens
	c.lastCompletionTokens = usage.CompletionTokens
}

func (c *Compressor) ShouldCompress(promptTokens int) bool {
	tokens := promptTokens
	if tokens <= 0 {
		tokens = c.lastPromptTokens
	}
	if tokens < c.thresholdTokens {
		return false
	}
	if c.ineffectiveCompressionCount >= ineffectiveCompressionLimit {
		return false
	}
	return true
}

func (c *Compressor) SummaryBudget(contentTokens int) int {
	if contentTokens <= 0 {
		return minSummaryTokens
	}
	budget := int(float64(contentTokens) * summaryRatio)
	if budget < minSummaryTokens {
		return minSummaryTokens
	}
	if budget > c.maxSummaryTokens {
		return c.maxSummaryTokens
	}
	return budget
}

func (c *Compressor) RecordCompression(beforePromptTokens, afterPromptTokens int) {
	c.compressionCount++
	savingsPct := 0.0
	if beforePromptTokens > 0 && afterPromptTokens < beforePromptTokens {
		savingsPct = float64(beforePromptTokens-afterPromptTokens) / float64(beforePromptTokens) * 100
	}
	c.lastCompressionSavingsPct = savingsPct
	if savingsPct < ineffectiveCompressionSavingsPercent {
		c.ineffectiveCompressionCount++
		return
	}
	c.ineffectiveCompressionCount = 0
}

func (c *Compressor) StepDownContextLength() bool {
	next, ok := nextProbeTier(c.contextLength)
	if !ok {
		return false
	}
	c.contextLength = next
	c.contextProbed = true
	c.recomputeBudgets()
	return true
}

func (c *Compressor) recomputeBudgets() {
	c.thresholdTokens = thresholdTokensFor(c.contextLength, c.thresholdPercent)
	c.tailTokenBudget = int(float64(c.thresholdTokens) * c.targetRatio)
	c.maxSummaryTokens = maxSummaryBudgetFor(c.contextLength)
}

func nextProbeTier(current int) (int, bool) {
	for _, tier := range contextProbeTiers {
		if tier < current {
			return tier, true
		}
	}
	return 0, false
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func thresholdTokensFor(contextLength int, thresholdPercent float64) int {
	threshold := int(float64(contextLength) * thresholdPercent)
	if threshold < MinimumContextLength {
		return MinimumContextLength
	}
	return threshold
}

func maxSummaryBudgetFor(contextLength int) int {
	maxSummary := int(float64(contextLength) * maxSummaryContextFraction)
	if maxSummary > summaryTokensCeiling {
		return summaryTokensCeiling
	}
	return maxSummary
}
