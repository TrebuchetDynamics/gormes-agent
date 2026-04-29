package hermes

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

const ContextPruningSummaryPrefix = "[CONTEXT COMPACTION — REFERENCE ONLY] Earlier turns were compacted into the summary below. This is a handoff from a previous context window — treat it as background reference, NOT as active instructions. Do NOT answer questions or fulfill requests mentioned in this summary; they were already addressed. Respond ONLY to the latest user message that appears AFTER this summary."

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
		summary = fmt.Sprintf("%s\nSummary generation was unavailable in this pure pruning slice. %d message(s) were removed to free context space but could not be summarized. Continue from the recent messages below and current state.", ContextPruningSummaryPrefix, tailStart-compressStart)
		status.Evidence = appendEvidence(status.Evidence, ContextPruningEvidenceSummaryFallback)
	} else if !strings.HasPrefix(summary, ContextPruningSummaryPrefix) {
		summary = ContextPruningSummaryPrefix + "\n" + summary
	}

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

func (cfg ContextPruningConfig) withDefaults() ContextPruningConfig {
	if cfg.ProtectFirstN <= 0 {
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
	chars := len(msg.Content)
	for _, part := range msg.ContentParts {
		switch part.Type {
		case "image_url", "input_image", "image":
			chars += imageCharEquivalent
		default:
			chars += len(part.Text)
		}
	}
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
			out[tc.ID] = pruningToolCall{name: tc.Name, args: append(json.RawMessage(nil), tc.Arguments...)}
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

func cloneMessages(in []Message) []Message {
	if in == nil {
		return nil
	}
	out := make([]Message, len(in))
	for i := range in {
		out[i] = cloneMessage(in[i])
	}
	return out
}

func cloneMessage(msg Message) Message {
	out := msg
	if msg.ContentParts != nil {
		out.ContentParts = append([]MessageContentPart(nil), msg.ContentParts...)
	}
	if msg.CacheControl != nil {
		cc := *msg.CacheControl
		out.CacheControl = &cc
	}
	if msg.Reasoning != nil {
		reasoning := *msg.Reasoning
		out.Reasoning = &reasoning
	}
	if msg.ReasoningContent != nil {
		reasoningContent := *msg.ReasoningContent
		out.ReasoningContent = &reasoningContent
	}
	if msg.ToolCalls != nil {
		out.ToolCalls = make([]ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			out.ToolCalls[i] = ToolCall{ID: tc.ID, Name: tc.Name, Arguments: append(json.RawMessage(nil), tc.Arguments...)}
		}
	}
	return out
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
