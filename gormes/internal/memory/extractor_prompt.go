package memory

import (
	"fmt"
	"strings"
)

// turnRow mirrors the subset of turns columns the extractor reads.
type turnRow struct {
	id      int64
	role    string
	content string
}

// maxTurnChars caps individual turn content in the LLM prompt. Matches
// the Telegram renderer's 4000-char cap so one runaway turn can't
// blow out the context window.
const maxTurnChars = 4000

// extractorSystemPrompt is the verbatim system message sent to the LLM
// before each extraction batch. Any change here is a behavior change of
// the entire extractor; bump schemaVersion if you expand the predicate
// or type whitelist to stay in lockstep with the CHECK constraints.
const extractorSystemPrompt = `You are an ontological entity extractor. You read conversation turns between a user and an AI assistant, and you emit a structured JSON summary of the entities mentioned and the relationships between them.

Rules:
1. Output ONLY valid JSON. No prose. No markdown fences. Start with '{'.
2. The JSON object has exactly two keys: "entities" and "relationships".
3. Each entity is {"name": string, "type": one of ["PERSON","PROJECT","CONCEPT","PLACE","ORGANIZATION","TOOL","OTHER"], "description": string (<= 512 chars, optional, empty string if absent)}.
4. Each relationship is {"source": string (entity name), "target": string (entity name), "predicate": one of the 8 values listed in rule 6, "weight": number between 0.0 and 1.0}.
5. Relationship source/target names MUST match entity names in this same response exactly. Do not reference entities not in entities[].
6. "predicate" MUST be EXACTLY one of these 8 uppercase strings:
   WORKS_ON     — an agent produces or contributes to a project/tool
   KNOWS        — an agent is aware of another agent or concept
   LIKES        — an agent expresses positive sentiment
   DISLIKES     — an agent expresses negative sentiment
   HAS_SKILL    — an agent possesses a concept/tool as a capability
   LOCATED_IN   — an entity is geographically or structurally inside another
   PART_OF      — an entity is a structural component of another
   RELATED_TO   — a generic fallback when no other predicate fits
   If none of the specific predicates fits, use RELATED_TO. Do NOT invent new predicates.
7. Deduplicate within the response: do not emit the same entity twice or the same (source, target, predicate) triple twice.
8. If no entities are present, emit {"entities": [], "relationships": []}.
`

// formatBatchPrompt renders the user message for one extraction batch:
// a blank-line-separated list of role-prefixed turn contents, each
// truncated to maxTurnChars.
func formatBatchPrompt(rows []turnRow) string {
	var b strings.Builder
	b.WriteString("Conversation turns to analyze (role: content):\n\n")
	for _, r := range rows {
		content := r.content
		if len(content) > maxTurnChars {
			content = content[:maxTurnChars] + "..."
		}
		fmt.Fprintf(&b, "[%s]: %s\n\n", r.role, content)
	}
	return b.String()
}
