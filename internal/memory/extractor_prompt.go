package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/extraction"

// turnRow mirrors the subset of turns columns the extractor reads.
type turnRow struct {
	id      int64
	role    string
	content string
}

const maxTurnChars = extraction.MaxTurnChars
const extractorSystemPrompt = extraction.SystemPrompt

// formatBatchPrompt renders the user message for one extraction batch:
// a blank-line-separated list of role-prefixed turn contents, each
// truncated to maxTurnChars.
func formatBatchPrompt(rows []turnRow) string {
	turns := make([]extraction.Turn, 0, len(rows))
	for _, row := range rows {
		turns = append(turns, extraction.Turn{
			ID:      row.id,
			Role:    row.role,
			Content: row.content,
		})
	}
	return extraction.FormatBatchPrompt(turns)
}
