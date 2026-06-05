package llm

const contextCompressionCharsPerToken = charsPerToken

// ContextCompressionBoundaryOptions controls the deterministic head/tail
// selection used before summarizing middle conversation turns.
type ContextCompressionBoundaryOptions struct {
	// ProtectFirstN is the count of non-system head messages to keep verbatim.
	// A leading system message is always protected in addition to this count.
	ProtectFirstN int
	// TailTokenBudget is the approximate token budget for recent tail context.
	// The selector may exceed it slightly to avoid splitting active turns.
	TailTokenBudget int
}

// ContextCompressionBoundaryPlan describes the protected regions selected for
// context compression. Messages before CompressStart and from TailStart onward
// stay verbatim; messages in between are eligible for summarization.
type ContextCompressionBoundaryPlan struct {
	HeadEnd              int
	CompressStart        int
	TailStart            int
	HasContentToCompress bool
}

// PlanContextCompressionBoundary mirrors Hermes' protected-head and protected
// token-budget-tail selection without performing summarization or provider IO.
// It preserves a leading system prompt, protects the first N non-system head
// messages, keeps the latest user message in the tail, and never cuts a tail
// boundary through an assistant tool-call/result group.
func PlanContextCompressionBoundary(messages []Message, opts ContextCompressionBoundaryOptions) ContextCompressionBoundaryPlan {
	n := len(messages)
	if n == 0 {
		return ContextCompressionBoundaryPlan{}
	}
	headEnd := contextCompressionProtectHeadSize(messages, opts.ProtectFirstN)
	compressStart := contextCompressionAlignBoundaryForward(messages, headEnd)
	if compressStart >= n {
		return ContextCompressionBoundaryPlan{
			HeadEnd:       headEnd,
			CompressStart: compressStart,
			TailStart:     n,
		}
	}
	tailStart := contextCompressionFindTailCutByTokens(messages, compressStart, opts.TailTokenBudget)
	return ContextCompressionBoundaryPlan{
		HeadEnd:              headEnd,
		CompressStart:        compressStart,
		TailStart:            tailStart,
		HasContentToCompress: compressStart < tailStart,
	}
}

func contextCompressionProtectHeadSize(messages []Message, protectFirstN int) int {
	if protectFirstN < 0 {
		protectFirstN = 0
	}
	head := 0
	if len(messages) > 0 && messages[0].Role == "system" {
		head = 1
	}
	head += protectFirstN
	if head > len(messages) {
		return len(messages)
	}
	return head
}

func contextCompressionAlignBoundaryForward(messages []Message, idx int) int {
	for idx < len(messages) && messages[idx].Role == "tool" {
		idx++
	}
	return idx
}

func contextCompressionFindTailCutByTokens(messages []Message, headEnd, tokenBudget int) int {
	n := len(messages)
	if n == 0 {
		return 0
	}
	if headEnd >= n {
		return n
	}
	if tokenBudget < 0 {
		tokenBudget = 0
	}

	minTail := 0
	if n-headEnd > 1 {
		minTail = minInt(3, n-headEnd-1)
	}
	softCeiling := int(float64(tokenBudget) * 1.5)
	accumulated := 0
	cutIdx := n

	for i := n - 1; i >= headEnd; i-- {
		msgTokens := contextCompressionMessageTokenEstimate(messages[i])
		if accumulated+msgTokens > softCeiling && n-i >= minTail {
			break
		}
		accumulated += msgTokens
		cutIdx = i
	}

	fallbackCut := n - minTail
	if cutIdx > fallbackCut {
		cutIdx = fallbackCut
	}
	if cutIdx <= headEnd {
		cutIdx = fallbackCut
		if cutIdx <= headEnd {
			cutIdx = headEnd + 1
		}
	}
	if cutIdx > n {
		cutIdx = n
	}

	cutIdx = contextCompressionAlignBoundaryBackward(messages, cutIdx)
	cutIdx = contextCompressionEnsureLatestUserInTail(messages, cutIdx, headEnd)
	if cutIdx < headEnd {
		cutIdx = headEnd
	}
	if cutIdx > n {
		cutIdx = n
	}
	return cutIdx
}

func contextCompressionAlignBoundaryBackward(messages []Message, idx int) int {
	if idx <= 0 || idx >= len(messages) {
		return idx
	}
	check := idx - 1
	for check >= 0 && messages[check].Role == "tool" {
		check--
	}
	if check >= 0 && messages[check].Role == "assistant" && len(messages[check].ToolCalls) > 0 {
		return check
	}
	return idx
}

func contextCompressionEnsureLatestUserInTail(messages []Message, cutIdx, headEnd int) int {
	latestUser := -1
	for i := len(messages) - 1; i >= headEnd; i-- {
		if messages[i].Role == "user" {
			latestUser = i
			break
		}
	}
	if latestUser < 0 || latestUser >= cutIdx {
		return cutIdx
	}
	if latestUser < headEnd+1 {
		return headEnd + 1
	}
	return latestUser
}

func contextCompressionMessageTokenEstimate(msg Message) int {
	tokens := contextCompressionContentLength(msg) / contextCompressionCharsPerToken
	tokens += 10
	for _, call := range msg.ToolCalls {
		tokens += len(call.Arguments) / contextCompressionCharsPerToken
	}
	return tokens
}

func contextCompressionContentLength(msg Message) int {
	return messageCompressionContentBudgetLength(msg)
}
