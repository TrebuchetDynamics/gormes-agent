package transcript

import (
	"reflect"
	"strings"
	"testing"
)

func TestTrajectoryCompressorProtectedIndicesMirrorHermes(t *testing.T) {
	turns := []TrajectoryTurn{
		{From: "system", Value: "You are an agent."},
		{From: "human", Value: "Do something."},
		{From: "gpt", Value: "I will use a tool."},
		{From: "tool", Value: "Tool result."},
		{From: "gpt", Value: "More work."},
		{From: "tool", Value: "Another result."},
		{From: "gpt", Value: "Work continues."},
		{From: "tool", Value: "Result 3."},
		{From: "gpt", Value: "Done."},
		{From: "human", Value: "Thanks."},
	}

	protected, start, end := FindTrajectoryProtectedIndices(turns, DefaultTrajectoryCompressionConfig())

	for _, idx := range []int{0, 1, 2, 3, 6, 7, 8, 9} {
		if !protected[idx] {
			t.Fatalf("protected[%d] = false, want true; all=%v", idx, protected)
		}
	}
	if start != 4 || end != 6 {
		t.Fatalf("compressible range = [%d,%d), want [4,6)", start, end)
	}
}

func TestTrajectoryCompressorProtectedIndicesHonorConfig(t *testing.T) {
	cfg := DefaultTrajectoryCompressionConfig()
	cfg.ProtectLastNTurns = 0
	cfg.ProtectFirstSystem = false
	turns := []TrajectoryTurn{
		{From: "system", Value: "sys"},
		{From: "human", Value: "q"},
		{From: "gpt", Value: "a"},
		{From: "tool", Value: "r"},
		{From: "gpt", Value: "b"},
		{From: "tool", Value: "r2"},
		{From: "gpt", Value: "c"},
		{From: "tool", Value: "r3"},
	}

	protected, start, end := FindTrajectoryProtectedIndices(turns, cfg)

	if protected[0] {
		t.Fatalf("system turn protected despite ProtectFirstSystem=false: %v", protected)
	}
	for _, idx := range []int{1, 2, 3} {
		if !protected[idx] {
			t.Fatalf("protected[%d] = false, want true", idx)
		}
	}
	if protected[7] {
		t.Fatalf("last turn protected despite ProtectLastNTurns=0: %v", protected)
	}
	if start != 4 || end != len(turns) {
		t.Fatalf("compressible range = [%d,%d), want [4,%d)", start, end, len(turns))
	}
}

func TestTrajectoryCompressorExtractSummaryContentTruncatesLongTurns(t *testing.T) {
	turns := []TrajectoryTurn{
		{From: "gpt", Value: "I will search."},
		{From: "tool", Value: "x" + strings.Repeat("a", 4999)},
		{From: "gpt", Value: "Great, done."},
	}

	content := ExtractTrajectorySummaryContent(turns, 0, 2)

	for _, want := range []string{"[Turn 0 - GPT]", "I will search.", "[Turn 1 - TOOL]", "...[truncated]..."} {
		if !strings.Contains(content, want) {
			t.Fatalf("summary content missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "[Turn 2") {
		t.Fatalf("summary content included exclusive end turn:\n%s", content)
	}
	if len(content) >= 5000 {
		t.Fatalf("summary content len = %d, want truncated below 5000", len(content))
	}
}

func TestTrajectoryCompressorCompressesMiddleAndPreservesLineageMetrics(t *testing.T) {
	cfg := DefaultTrajectoryCompressionConfig()
	cfg.TargetMaxTokens = 35
	cfg.SummaryTargetTokens = 4
	turns := []TrajectoryTurn{
		{From: "system", Value: "system prompt"},
		{From: "human", Value: "initial task"},
		{From: "gpt", Value: "first answer"},
		{From: "tool", Value: "first tool output"},
		{From: "gpt", Value: strings.Repeat("middle a ", 10)},
		{From: "tool", Value: strings.Repeat("middle b ", 10)},
		{From: "gpt", Value: "tail one"},
		{From: "tool", Value: "tail two"},
		{From: "gpt", Value: "tail three"},
		{From: "human", Value: "tail four"},
	}
	original := append([]TrajectoryTurn(nil), turns...)

	compressed, metrics := CompressTrajectory(turns, cfg, "assistant searched and kept key findings", testTrajectoryTokenCounter)

	if !reflect.DeepEqual(turns, original) {
		t.Fatalf("CompressTrajectory mutated input:\n got %#v\nwant %#v", turns, original)
	}
	if !metrics.WasCompressed {
		t.Fatalf("WasCompressed = false, want true: %+v", metrics)
	}
	if metrics.TurnsCompressedStartIdx != 4 || metrics.TurnsCompressedEndIdx != 6 || metrics.TurnsInCompressedRegion != 2 {
		t.Fatalf("compression region = %+v, want start=4 end=6 count=2", metrics)
	}
	if metrics.OriginalTurns != len(turns) || metrics.CompressedTurns != len(compressed) {
		t.Fatalf("turn metrics = %+v, compressed len=%d original len=%d", metrics, len(compressed), len(turns))
	}
	if metrics.TurnsRemoved != 1 {
		t.Fatalf("TurnsRemoved = %d, want 1", metrics.TurnsRemoved)
	}
	if got := compressed[0].Value; !strings.Contains(got, "Some of your previous tool responses may be summarized") {
		t.Fatalf("system notice missing from compressed head: %q", got)
	}
	if compressed[4].From != "human" || !strings.HasPrefix(compressed[4].Value, "[CONTEXT SUMMARY]:") {
		t.Fatalf("summary turn = %+v, want human context summary", compressed[4])
	}
	if compressed[5].Value != "tail one" || compressed[len(compressed)-1].Value != "tail four" {
		t.Fatalf("tail not preserved after summary: %#v", compressed)
	}
}

func TestTrajectoryCompressorSkipsUnderTargetAndShortAllProtected(t *testing.T) {
	cfg := DefaultTrajectoryCompressionConfig()
	cfg.TargetMaxTokens = 1000
	turns := []TrajectoryTurn{
		{From: "system", Value: "sys"},
		{From: "human", Value: "hi"},
		{From: "gpt", Value: "hello"},
	}

	compressed, metrics := CompressTrajectory(turns, cfg, "unused", testTrajectoryTokenCounter)

	if !metrics.SkippedUnderTarget {
		t.Fatalf("SkippedUnderTarget = false, want true: %+v", metrics)
	}
	if metrics.WasCompressed || !reflect.DeepEqual(compressed, turns) {
		t.Fatalf("under-target compression changed output: compressed=%#v metrics=%+v", compressed, metrics)
	}
}

func TestTrajectoryCompressorSummaryPrefixNormalizesExactlyOnce(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "adds prefix", in: "summary", want: "[CONTEXT SUMMARY]: summary"},
		{name: "keeps prefix", in: "[CONTEXT SUMMARY]: summary", want: "[CONTEXT SUMMARY]: summary"},
		{name: "empty", in: " ", want: "[CONTEXT SUMMARY]:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnsureTrajectorySummaryPrefix(tt.in); got != tt.want {
				t.Fatalf("EnsureTrajectorySummaryPrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func testTrajectoryTokenCounter(text string) int {
	return len(text) / 4
}
