package contextengine

import "testing"

func TestNewCompressor_ComputesThresholdTailAndSummaryCaps(t *testing.T) {
	c := NewCompressor(Config{
		ContextLength:    200_000,
		ThresholdPercent: 0.50,
		TargetRatio:      0.20,
	})

	if got := c.ContextLength(); got != 200_000 {
		t.Fatalf("ContextLength() = %d, want %d", got, 200_000)
	}
	if got := c.ThresholdTokens(); got != 100_000 {
		t.Fatalf("ThresholdTokens() = %d, want %d", got, 100_000)
	}
	if got := c.TailTokenBudget(); got != 20_000 {
		t.Fatalf("TailTokenBudget() = %d, want %d", got, 20_000)
	}
	if got := c.MaxSummaryTokens(); got != 10_000 {
		t.Fatalf("MaxSummaryTokens() = %d, want %d", got, 10_000)
	}
}

func TestNewCompressor_AppliesMinimumThresholdFloor(t *testing.T) {
	c := NewCompressor(Config{
		ContextLength:    70_000,
		ThresholdPercent: 0.50,
		TargetRatio:      0.20,
	})

	if got := c.ThresholdTokens(); got != MinimumContextLength {
		t.Fatalf("ThresholdTokens() = %d, want %d", got, MinimumContextLength)
	}
	if got := c.TailTokenBudget(); got != 12_800 {
		t.Fatalf("TailTokenBudget() = %d, want %d", got, 12_800)
	}
	if got := c.MaxSummaryTokens(); got != 3_500 {
		t.Fatalf("MaxSummaryTokens() = %d, want %d", got, 3_500)
	}
}

func TestSummaryBudget_ClampsBetweenMinimumAndContextCeiling(t *testing.T) {
	c := NewCompressor(Config{
		ContextLength:    200_000,
		ThresholdPercent: 0.50,
		TargetRatio:      0.20,
	})

	tests := []struct {
		name          string
		contentTokens int
		want          int
	}{
		{name: "minimum", contentTokens: 4_000, want: 2_000},
		{name: "scaled", contentTokens: 25_000, want: 5_000},
		{name: "ceiling", contentTokens: 100_000, want: 10_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.SummaryBudget(tt.contentTokens); got != tt.want {
				t.Fatalf("SummaryBudget(%d) = %d, want %d", tt.contentTokens, got, tt.want)
			}
		})
	}
}

func TestShouldCompress_UsesTrackedPromptTokensAndBacksOffAfterTwoIneffectiveCompressions(t *testing.T) {
	c := NewCompressor(Config{
		ContextLength:    128_000,
		ThresholdPercent: 0.50,
		TargetRatio:      0.20,
	})

	c.UpdateFromResponse(Usage{PromptTokens: 64_000, CompletionTokens: 1_000})
	if !c.ShouldCompress(0) {
		t.Fatal("ShouldCompress(0) = false, want true from tracked prompt tokens")
	}

	c.RecordCompression(100_000, 95_000)
	if !c.ShouldCompress(100_000) {
		t.Fatal("ShouldCompress() = false after one ineffective compression, want true")
	}

	c.RecordCompression(100_000, 92_000)
	if c.ShouldCompress(100_000) {
		t.Fatal("ShouldCompress() = true after two ineffective compressions, want false")
	}

	if got := c.IneffectiveCompressionCount(); got != 2 {
		t.Fatalf("IneffectiveCompressionCount() = %d, want %d", got, 2)
	}
	if got := c.CompressionCount(); got != 2 {
		t.Fatalf("CompressionCount() = %d, want %d", got, 2)
	}
}

func TestStepDownContextLength_UsesProbeTiersAndRecalculatesBudgets(t *testing.T) {
	c := NewCompressor(Config{
		ContextLength:    200_000,
		ThresholdPercent: 0.50,
		TargetRatio:      0.20,
	})

	if ok := c.StepDownContextLength(); !ok {
		t.Fatal("StepDownContextLength() = false, want true")
	}
	if got := c.ContextLength(); got != 128_000 {
		t.Fatalf("ContextLength() = %d after probe, want %d", got, 128_000)
	}
	if got := c.ThresholdTokens(); got != 64_000 {
		t.Fatalf("ThresholdTokens() = %d after probe, want %d", got, 64_000)
	}
	if got := c.TailTokenBudget(); got != 12_800 {
		t.Fatalf("TailTokenBudget() = %d after probe, want %d", got, 12_800)
	}
	if got := c.MaxSummaryTokens(); got != 6_400 {
		t.Fatalf("MaxSummaryTokens() = %d after probe, want %d", got, 6_400)
	}
	if !c.ContextProbed() {
		t.Fatal("ContextProbed() = false, want true")
	}
}
