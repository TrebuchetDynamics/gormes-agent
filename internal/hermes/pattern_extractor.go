package hermes

import (
	"sort"
	"strings"
)

type ToolSequence struct {
	Tools   []string
	Success bool
	Count   int
}

type PatternExtractor struct {
	sequences []ToolSequence
}

func NewPatternExtractor() *PatternExtractor {
	return &PatternExtractor{}
}

func (pe *PatternExtractor) RecordSequence(tools []string, success bool) {
	for i := range pe.sequences {
		if toolsEqual(pe.sequences[i].Tools, tools) && pe.sequences[i].Success == success {
			pe.sequences[i].Count++
			return
		}
	}
	pe.sequences = append(pe.sequences, ToolSequence{
		Tools:   append([]string{}, tools...),
		Success: success,
		Count:   1,
	})
}

func (pe *PatternExtractor) SuccessfulPatterns(minCount int) []ToolSequence {
	return pe.filterPatterns(true, minCount)
}

func (pe *PatternExtractor) FailedPatterns(minCount int) []ToolSequence {
	return pe.filterPatterns(false, minCount)
}

func (pe *PatternExtractor) filterPatterns(success bool, minCount int) []ToolSequence {
	var result []ToolSequence
	for _, s := range pe.sequences {
		if s.Success == success && s.Count >= minCount {
			result = append(result, s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result
}

func (pe *PatternExtractor) PatternSummary() string {
	var b strings.Builder
	success := pe.SuccessfulPatterns(1)
	failed := pe.FailedPatterns(1)
	b.WriteString("Successful patterns:\n")
	for _, s := range success {
		b.WriteString("  " + strings.Join(s.Tools, " -> ") + "\n")
	}
	b.WriteString("Failed patterns:\n")
	for _, s := range failed {
		b.WriteString("  " + strings.Join(s.Tools, " -> ") + "\n")
	}
	return b.String()
}

func toolsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
