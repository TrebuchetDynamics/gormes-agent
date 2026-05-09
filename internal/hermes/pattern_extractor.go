package hermes

import (
	"fmt"
	"sort"
	"strings"
)

const (
	successPatternThreshold = 0.80
	antiPatternThreshold    = 0.30
)

type ToolSequenceObservation struct {
	SessionID        string
	Tools            []string
	Success          bool
	ReasoningPattern string
}

type ReasoningPattern struct {
	Text  string
	Count int
}

type ToolSequence struct {
	Tools             []string
	Success           bool
	Count             int
	Observations      int
	Successes         int
	Failures          int
	SuccessRate       float64
	ReasoningPatterns []ReasoningPattern
	Evidence          []string
}

type BehavioralKnowledge struct {
	Kind         string   `json:"kind"`
	Source       string   `json:"source"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags"`
	Tools        []string `json:"tools"`
	SuccessRate  float64  `json:"success_rate"`
	Observations int      `json:"observations"`
	Successes    int      `json:"successes"`
	Failures     int      `json:"failures"`
	Evidence     []string `json:"evidence"`
}

type sequenceStats struct {
	tools     []string
	successes int
	failures  int
	reasoning map[string]int
	sessions  map[string]struct{}
}

type PatternExtractor struct {
	sequences map[string]*sequenceStats
	order     []string
}

func NewPatternExtractor() *PatternExtractor {
	return &PatternExtractor{
		sequences: make(map[string]*sequenceStats),
	}
}

func (pe *PatternExtractor) RecordSequence(tools []string, success bool) {
	pe.RecordObservation(ToolSequenceObservation{Tools: tools, Success: success})
}

func (pe *PatternExtractor) RecordObservation(obs ToolSequenceObservation) {
	tools := normalizeToolSequence(obs.Tools)
	if len(tools) == 0 {
		return
	}

	pe.ensure()
	key := toolSequenceKey(tools)
	stats := pe.sequences[key]
	if stats == nil {
		stats = &sequenceStats{
			tools:     tools,
			reasoning: make(map[string]int),
			sessions:  make(map[string]struct{}),
		}
		pe.sequences[key] = stats
		pe.order = append(pe.order, key)
	}

	if obs.Success {
		stats.successes++
		reasoning := strings.TrimSpace(obs.ReasoningPattern)
		if reasoning != "" {
			stats.reasoning[reasoning]++
		}
	} else {
		stats.failures++
	}
	sessionID := strings.TrimSpace(obs.SessionID)
	if sessionID != "" {
		stats.sessions[sessionID] = struct{}{}
	}
}

func (pe *PatternExtractor) SuccessfulPatterns(minCount int) []ToolSequence {
	return pe.filterPatterns(true, minCount, successPatternThreshold)
}

func (pe *PatternExtractor) FailedPatterns(minCount int) []ToolSequence {
	return pe.filterPatterns(false, minCount, antiPatternThreshold)
}

func (pe *PatternExtractor) filterPatterns(success bool, minCount int, threshold float64) []ToolSequence {
	pe.ensure()
	if minCount < 1 {
		minCount = 1
	}

	var result []ToolSequence
	for _, key := range pe.order {
		stats := pe.sequences[key]
		if stats == nil {
			continue
		}
		sequence := stats.toolSequence(success)
		if success {
			if sequence.Successes >= minCount && sequence.SuccessRate > threshold {
				result = append(result, sequence)
			}
			continue
		}
		if sequence.Failures >= minCount && sequence.SuccessRate < threshold {
			result = append(result, sequence)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SuccessRate != result[j].SuccessRate {
			if success {
				return result[i].SuccessRate > result[j].SuccessRate
			}
			return result[i].SuccessRate < result[j].SuccessRate
		}
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return toolSequenceKey(result[i].Tools) < toolSequenceKey(result[j].Tools)
	})
	return result
}

func (pe *PatternExtractor) GonchoKnowledge(minCount int) []BehavioralKnowledge {
	var records []BehavioralKnowledge
	for _, sequence := range pe.SuccessfulPatterns(minCount) {
		records = append(records, sequence.behavioralKnowledge(
			"tool_sequence_success_pattern",
			[]string{"behavioral_pattern", "tool_sequence", "success_pattern"},
		))
	}
	for _, sequence := range pe.FailedPatterns(minCount) {
		records = append(records, sequence.behavioralKnowledge(
			"tool_sequence_anti_pattern",
			[]string{"behavioral_pattern", "tool_sequence", "anti_pattern"},
		))
	}
	return records
}

func (pe *PatternExtractor) PatternSummary() string {
	var b strings.Builder
	success := pe.SuccessfulPatterns(1)
	failed := pe.FailedPatterns(1)
	b.WriteString("Successful patterns:\n")
	for _, s := range success {
		b.WriteString("  " + s.summaryLine() + "\n")
		for _, reasoning := range s.ReasoningPatterns {
			b.WriteString(fmt.Sprintf("    reasoning: %s (%d)\n", reasoning.Text, reasoning.Count))
		}
	}
	b.WriteString("Failed patterns:\n")
	for _, s := range failed {
		b.WriteString("  " + s.summaryLine() + "\n")
	}
	return b.String()
}

func (pe *PatternExtractor) ensure() {
	if pe.sequences == nil {
		pe.sequences = make(map[string]*sequenceStats)
	}
}

func (s *sequenceStats) toolSequence(success bool) ToolSequence {
	observations := s.successes + s.failures
	rate := 0.0
	if observations > 0 {
		rate = float64(s.successes) / float64(observations)
	}
	count := s.failures
	if success {
		count = s.successes
	}
	return ToolSequence{
		Tools:             append([]string{}, s.tools...),
		Success:           success,
		Count:             count,
		Observations:      observations,
		Successes:         s.successes,
		Failures:          s.failures,
		SuccessRate:       rate,
		ReasoningPatterns: sortedReasoningPatterns(s.reasoning),
		Evidence:          s.evidence(),
	}
}

func (s *sequenceStats) evidence() []string {
	out := []string{
		fmt.Sprintf("sequence=%s", strings.Join(s.tools, " -> ")),
		fmt.Sprintf("observations=%d", s.successes+s.failures),
		fmt.Sprintf("successes=%d", s.successes),
		fmt.Sprintf("failures=%d", s.failures),
	}
	if len(s.sessions) > 0 {
		out = append(out, fmt.Sprintf("sessions=%d", len(s.sessions)))
	}
	return out
}

func (s ToolSequence) summaryLine() string {
	return fmt.Sprintf("%s (success_rate=%.1f%%, observations=%d, successes=%d, failures=%d)",
		strings.Join(s.Tools, " -> "),
		s.SuccessRate*100,
		s.Observations,
		s.Successes,
		s.Failures,
	)
}

func (s ToolSequence) behavioralKnowledge(kind string, tags []string) BehavioralKnowledge {
	content := s.summaryLine()
	if len(s.ReasoningPatterns) > 0 {
		var parts []string
		for _, reasoning := range s.ReasoningPatterns {
			parts = append(parts, fmt.Sprintf("%s (%d)", reasoning.Text, reasoning.Count))
		}
		content += "; reasoning: " + strings.Join(parts, "; ")
	}
	return BehavioralKnowledge{
		Kind:         kind,
		Source:       "pattern_extractor",
		Content:      content,
		Tags:         append([]string{}, tags...),
		Tools:        append([]string{}, s.Tools...),
		SuccessRate:  s.SuccessRate,
		Observations: s.Observations,
		Successes:    s.Successes,
		Failures:     s.Failures,
		Evidence:     append([]string{}, s.Evidence...),
	}
}

func sortedReasoningPatterns(counts map[string]int) []ReasoningPattern {
	if len(counts) == 0 {
		return nil
	}
	out := make([]ReasoningPattern, 0, len(counts))
	for text, count := range counts {
		out = append(out, ReasoningPattern{Text: text, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Text < out[j].Text
	})
	return out
}

func normalizeToolSequence(tools []string) []string {
	normalized := make([]string, 0, len(tools))
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		normalized = append(normalized, tool)
	}
	return normalized
}

func toolSequenceKey(tools []string) string {
	return strings.Join(tools, "\x00")
}

func toolsEqual(a, b []string) bool {
	a = normalizeToolSequence(a)
	b = normalizeToolSequence(b)
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
