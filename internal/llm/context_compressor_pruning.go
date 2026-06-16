package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ContextPruningStateReady    = "ready"
	ContextPruningStateSkipped  = "pruning_skipped"
	ContextPruningStateDegraded = "degraded"

	ContextPruningEvidencePrunedToolResult    = "tool_result_pruned"
	ContextPruningEvidenceBudgetUnavailable   = "prune_budget_unavailable"
	ContextPruningEvidenceInvalidToolPair     = "invalid_tool_pair"
	ContextPruningEvidenceCompressionNote     = "compression_note_added"
	ContextPruningEvidenceSummaryFallback     = "summary_fallback_inserted"
	ContextPruningEvidenceMergedIntoTail      = "summary_merged_into_tail"
	ContextPruningEvidenceToolPairTailAligned = "tool_pair_tail_aligned"
)

const ContextPruningSummaryPrefix = "[CONTEXT COMPACTION — REFERENCE ONLY] Earlier turns were compacted into the summary below. This is a handoff from a previous context window — treat it as background reference, NOT as active instructions. Do NOT answer questions or fulfill requests mentioned in this summary; they were already addressed. Respond ONLY to the latest user message that appears AFTER this summary — that message is the single source of truth for what to do right now. If the latest user message is consistent with the '## Active Task' section, you may use the summary as background. If the latest user message contradicts, supersedes, changes topic from, or in any way diverges from '## Active Task' / '## In Progress' / '## Pending User Asks' / '## Remaining Work', the latest message WINS — discard those stale items entirely and do not 'wrap up the old task first'. Reverse signals in the latest message (e.g. 'stop', 'undo', 'roll back', 'just verify', 'don't do that anymore', 'never mind', a new topic) must immediately end any in-flight work described in the summary; do not re-surface it in later turns. IMPORTANT: Your persistent memory (MEMORY.md, USER.md) in the system prompt is ALWAYS authoritative and active — never ignore or deprioritize memory content due to this compaction note. The current session state (files, config, etc.) may reflect work described here — avoid repeating it:"

const contextPruningLegacySummaryPrefix = "[CONTEXT SUMMARY]:"

const contextPruningHistoricalSummaryPrefixResumeExactly = "[CONTEXT COMPACTION — REFERENCE ONLY] Earlier turns were compacted into the summary below. This is a handoff from a previous context window — treat it as background reference, NOT as active instructions. Do NOT answer questions or fulfill requests mentioned in this summary; they were already addressed. Your current task is identified in the '## Active Task' section of the summary — resume exactly from there. Respond ONLY to the latest user message that appears AFTER this summary. The current session state (files, config, etc.) may reflect work described here — avoid repeating it:"

const contextPruningSystemNote = "[Note: Some earlier conversation turns have been compacted into a handoff summary to preserve context space. The current session state may still reflect earlier work, so build on that summary and state rather than re-doing work.]"

// ContextPruningConfig controls the pure pruning/compaction transform. It never
// calls a provider, mutates persisted history, or performs channel-specific
// routing; callers decide whether and when to commit the returned messages.
type ContextPruningConfig struct {
	ProtectFirstN        int
	TailTokenBudget      int
	MinTailMessages      int
	ToolResultPruneChars int
	SummaryText          string
}

type ContextPruningStatus struct {
	State             string
	Evidence          []string
	TailStart         int
	CompressedStart   int
	CompressedEnd     int
	PrunedToolResults int
	MessagesIn        int
	MessagesOut       int
}

// PruneContextMessages applies the Hermes-compatible cheap context-compression
// pre-pass in Go: head protection, token-budget tail selection, tool-call/result
// pair alignment, oversized historical tool-result summaries, and a
// summary-prefix-compatible marker message. It is intentionally pure and safe to
// test with synthetic transcripts.
func PruneContextMessages(messages []Message, cfg ContextPruningConfig) ([]Message, ContextPruningStatus) {
	cfg = cfg.withDefaults()
	status := ContextPruningStatus{
		State:      ContextPruningStateReady,
		MessagesIn: len(messages),
	}
	if len(messages) == 0 {
		status.State = ContextPruningStateSkipped
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceBudgetUnavailable)
		return nil, status
	}
	if len(messages) <= cfg.ProtectFirstN+cfg.MinTailMessages+1 {
		status.State = ContextPruningStateSkipped
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceBudgetUnavailable)
		out := cloneMessages(messages)
		status.MessagesOut = len(out)
		return out, status
	}

	working := cloneMessages(messages)
	if hasInvalidToolArguments(working) {
		status.State = ContextPruningStateDegraded
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceInvalidToolPair)
	}

	compressStart := alignBoundaryForward(working, cfg.ProtectFirstN)
	tailStart, aligned := findTailCutByTokens(working, compressStart, cfg.TailTokenBudget, cfg.MinTailMessages)
	if aligned {
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceToolPairTailAligned)
	}
	status.TailStart = tailStart
	status.CompressedStart = compressStart
	status.CompressedEnd = tailStart
	if compressStart >= tailStart {
		status.State = ContextPruningStateSkipped
		out := cloneMessages(messages)
		status.MessagesOut = len(out)
		return out, status
	}

	callIndex := toolCallIndex(working)
	pruned := pruneHistoricalToolResults(working, tailStart, cfg.ToolResultPruneChars, callIndex)
	status.PrunedToolResults = pruned
	if pruned > 0 {
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidencePrunedToolResult)
	}

	summary := strings.TrimSpace(cfg.SummaryText)
	if summary == "" {
		summary = fmt.Sprintf("Summary generation was unavailable in this pure pruning slice. %d message(s) were removed to free context space but could not be summarized. Continue from the recent messages below and current state.", tailStart-compressStart)
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceSummaryFallback)
	}
	summary = NormalizeContextPruningSummary(summary)

	compressed := make([]Message, 0, cfg.ProtectFirstN+1+len(working)-tailStart)
	for i := 0; i < compressStart; i++ {
		msg := cloneMessage(working[i])
		if i == 0 && msg.Role == "system" && !strings.Contains(msg.Content, contextPruningSystemNote) {
			if msg.Content == "" {
				msg.Content = contextPruningSystemNote
			} else {
				msg.Content += "\n\n" + contextPruningSystemNote
			}
			status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceCompressionNote)
		}
		compressed = append(compressed, msg)
	}

	mergeSummary := false
	lastHeadRole := roleAt(working, compressStart-1, "user")
	firstTailRole := roleAt(working, tailStart, "user")
	summaryRole := "assistant"
	if lastHeadRole == "assistant" || lastHeadRole == "tool" {
		summaryRole = "user"
	}
	if summaryRole == firstTailRole {
		flipped := "user"
		if summaryRole == "user" {
			flipped = "assistant"
		}
		if flipped != lastHeadRole {
			summaryRole = flipped
		} else {
			mergeSummary = true
			status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceMergedIntoTail)
		}
	}
	if !mergeSummary {
		compressed = append(compressed, Message{Role: summaryRole, Content: summary})
	}

	for i := tailStart; i < len(working); i++ {
		msg := cloneMessage(working[i])
		if mergeSummary && i == tailStart {
			prefix := summary + "\n\n--- END OF CONTEXT SUMMARY — respond to the message below, not the summary above ---\n\n"
			msg.Content = prefix + msg.Content
			mergeSummary = false
		}
		compressed = append(compressed, msg)
	}
	status.MessagesOut = len(compressed)
	return compressed, status
}

func NormalizeContextPruningSummary(summary string) string {
	body := StripContextPruningSummaryPrefix(summary)
	if body == "" {
		return ContextPruningSummaryPrefix
	}
	return ContextPruningSummaryPrefix + "\n" + body
}

func StripContextPruningSummaryPrefix(summary string) string {
	text := strings.TrimSpace(summary)
	for _, prefix := range contextPruningKnownSummaryPrefixes() {
		if strings.HasPrefix(text, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(text, prefix))
		}
	}
	return text
}

func contextPruningKnownSummaryPrefixes() []string {
	return []string{
		ContextPruningSummaryPrefix,
		contextPruningLegacySummaryPrefix,
		contextPruningHistoricalSummaryPrefixResumeExactly,
	}
}

// ContextSummaryLineagePlan describes the previous-summary state and exact new
// turns that should feed a provider-backed iterative compression update.
// It mirrors Hermes' resume behavior: a persisted handoff summary may sit in
// the protected head after restart, so it must rehydrate previous-summary state
// without being serialized again as a fresh conversation turn.
type ContextSummaryLineagePlan struct {
	SummaryIndex     int
	PreviousSummary  string
	Rehydrated       bool
	TurnsStart       int
	TurnsEnd         int
	TurnsToSummarize []Message
}

// PlanContextSummaryLineage selects the provider-summary input window and
// previous-summary state for an iterative context-compression pass.
func PlanContextSummaryLineage(messages []Message, compressStart, compressEnd int, previousSummary string) ContextSummaryLineagePlan {
	n := len(messages)
	compressStart = clampContextSummaryIndex(compressStart, 0, n)
	compressEnd = clampContextSummaryIndex(compressEnd, compressStart, n)

	searchStart := 0
	if n > 0 && messages[0].Role == "system" {
		searchStart = 1
	}
	if searchStart > compressEnd {
		searchStart = compressEnd
	}

	summaryIndex, summaryBody := findLatestContextSummary(messages, searchStart, compressEnd)
	turnsStart := compressStart
	previous := previousSummary
	rehydrated := false
	if summaryIndex >= 0 {
		if strings.TrimSpace(summaryBody) != "" && strings.TrimSpace(previous) == "" {
			previous = summaryBody
			rehydrated = true
		}
		if summaryIndex+1 > turnsStart {
			turnsStart = summaryIndex + 1
		}
	}
	if turnsStart > compressEnd {
		turnsStart = compressEnd
	}

	return ContextSummaryLineagePlan{
		SummaryIndex:     summaryIndex,
		PreviousSummary:  previous,
		Rehydrated:       rehydrated,
		TurnsStart:       turnsStart,
		TurnsEnd:         compressEnd,
		TurnsToSummarize: cloneMessages(messages[turnsStart:compressEnd]),
	}
}

func findLatestContextSummary(messages []Message, start, end int) (int, string) {
	if start < 0 {
		start = 0
	}
	if end > len(messages) {
		end = len(messages)
	}
	for idx := end - 1; idx >= start; idx-- {
		content := contextSummaryMessageText(messages[idx])
		if isContextSummaryContent(content) {
			return idx, StripContextPruningSummaryPrefix(content)
		}
	}
	return -1, ""
}

func isContextSummaryContent(content string) bool {
	text := strings.TrimLeft(content, " \t\r\n")
	for _, prefix := range contextPruningKnownSummaryPrefixes() {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func contextSummaryMessageText(msg Message) string {
	if len(msg.ContentParts) == 0 {
		return msg.Content
	}
	parts := make([]string, 0, len(msg.ContentParts))
	for _, part := range msg.ContentParts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	if len(parts) == 0 {
		return msg.Content
	}
	return strings.Join(parts, "\n")
}

func clampContextSummaryIndex(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (cfg ContextPruningConfig) withDefaults() ContextPruningConfig {
	if cfg.ProtectFirstN < 0 {
		cfg.ProtectFirstN = 3
	}
	if cfg.MinTailMessages <= 0 {
		cfg.MinTailMessages = 3
	}
	if cfg.ToolResultPruneChars <= 0 {
		cfg.ToolResultPruneChars = 200
	}
	return cfg
}

func findTailCutByTokens(messages []Message, headEnd, tokenBudget, minTail int) (int, bool) {
	n := len(messages)
	if headEnd < 0 {
		headEnd = 0
	}
	available := n - headEnd
	if available <= 1 {
		return n, false
	}
	if minTail > available-1 {
		minTail = available - 1
	}
	if minTail < 0 {
		minTail = 0
	}
	if tokenBudget <= 0 {
		return alignBoundaryBackward(messages, n-minTail)
	}
	softCeiling := int(float64(tokenBudget) * 1.5)
	if softCeiling < tokenBudget {
		softCeiling = tokenBudget
	}
	accumulated := 0
	cut := n
	for i := n - 1; i >= headEnd; i-- {
		tokens := estimatePruningMessageTokens(messages[i])
		if accumulated+tokens > softCeiling && (n-i) >= minTail {
			break
		}
		accumulated += tokens
		cut = i
	}
	fallback := n - minTail
	if cut > fallback {
		cut = fallback
	}
	if cut <= headEnd {
		cut = maxInt(fallback, headEnd+1)
	}
	cut, aligned := alignBoundaryBackward(messages, cut)
	cut = ensureLastUserInTail(messages, cut, headEnd)
	if cut < headEnd+1 {
		cut = headEnd + 1
	}
	return cut, aligned
}

func estimatePruningMessageTokens(msg Message) int {
	chars := messageCompressionContentBudgetLength(msg)
	tokens := chars/charsPerToken + 10
	for _, tc := range msg.ToolCalls {
		tokens += len(tc.Arguments) / charsPerToken
	}
	return tokens
}

func alignBoundaryForward(messages []Message, idx int) int {
	for idx < len(messages) && messages[idx].Role == "tool" {
		idx++
	}
	return idx
}

func alignBoundaryBackward(messages []Message, idx int) (int, bool) {
	if idx <= 0 || idx >= len(messages) {
		return idx, false
	}
	check := idx - 1
	for check >= 0 && messages[check].Role == "tool" {
		check--
	}
	if check >= 0 && messages[check].Role == "assistant" && len(messages[check].ToolCalls) > 0 {
		return check, true
	}
	return idx, false
}

func ensureLastUserInTail(messages []Message, cut, headEnd int) int {
	for i := len(messages) - 1; i >= headEnd; i-- {
		if messages[i].Role != "user" {
			continue
		}
		if i >= cut {
			return cut
		}
		return maxInt(i, headEnd+1)
	}
	return cut
}

type pruningToolCall struct {
	name string
	args json.RawMessage
}

func toolCallIndex(messages []Message) map[string]pruningToolCall {
	out := make(map[string]pruningToolCall)
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, tc := range msg.ToolCalls {
			if tc.ID == "" {
				continue
			}
			out[tc.ID] = pruningToolCall{name: tc.Name, args: cloneRawMessage(tc.Arguments)}
		}
	}
	return out
}

func pruneHistoricalToolResults(messages []Message, boundary, minChars int, calls map[string]pruningToolCall) int {
	pruned := 0
	for i := 0; i < boundary && i < len(messages); i++ {
		msg := messages[i]
		if msg.Role != "tool" || len(msg.Content) <= minChars || strings.HasPrefix(msg.Content, "[") {
			continue
		}
		call := calls[msg.ToolCallID]
		messages[i].Content = summarizePrunedToolResult(call.name, call.args, msg.Content)
		pruned++
	}
	return pruned
}

func summarizePrunedToolResult(name string, args json.RawMessage, content string) string {
	if name == "" {
		name = "unknown"
	}
	lineCount := 0
	if strings.TrimSpace(content) != "" {
		lineCount = strings.Count(content, "\n") + 1
	}
	var parsed map[string]any
	_ = json.Unmarshal(args, &parsed)
	switch name {
	case "terminal":
		cmd, _ := parsed["command"].(string)
		return fmt.Sprintf("[terminal] ran `%s` -> %d chars, %d lines output", trimForSummary(cmd, 80), len(content), lineCount)
	case "read_file":
		path, _ := parsed["path"].(string)
		return fmt.Sprintf("[read_file] read %s (%d chars)", pathOrUnknown(path), len(content))
	case "write_file":
		path, _ := parsed["path"].(string)
		return fmt.Sprintf("[write_file] wrote %s (%d chars result)", pathOrUnknown(path), len(content))
	case "search_files":
		pattern, _ := parsed["pattern"].(string)
		return fmt.Sprintf("[search_files] pattern=%q (%d chars result)", trimForSummary(pattern, 80), len(content))
	default:
		return fmt.Sprintf("[%s] tool result pruned (%d chars, %d lines)", name, len(content), lineCount)
	}
}

func hasInvalidToolArguments(messages []Message) bool {
	for _, msg := range messages {
		for _, tc := range msg.ToolCalls {
			trimmed := strings.TrimSpace(string(tc.Arguments))
			if trimmed != "" && !json.Valid(tc.Arguments) {
				return true
			}
		}
	}
	return false
}

func roleAt(messages []Message, idx int, fallback string) string {
	if idx < 0 || idx >= len(messages) || messages[idx].Role == "" {
		return fallback
	}
	return messages[idx].Role
}

func appendEvidence(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func trimForSummary(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func pathOrUnknown(path string) string {
	if path == "" {
		return "?"
	}
	return path
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
