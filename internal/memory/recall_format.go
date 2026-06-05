package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/formatting"

// recalledEntity is the subset of an entity row that gets rendered into
// the fenced memory-context block. Copied out of the DB rows to avoid
// keeping the rows handle open during formatting.
type recalledEntity struct {
	Name        string
	Type        string
	Description string
}

// recalledRel is the subset of a relationship row that gets rendered.
type recalledRel struct {
	Source    string
	Predicate string
	Target    string
	Weight    float64
}

const maxCandidates = formatting.MaxCandidates
const memoryContextHeader = formatting.ContextHeader

func extractCandidates(msg string) []string {
	return formatting.ExtractCandidates(msg)
}

func sanitizeFenceContent(s string) string {
	return formatting.SanitizeFenceContent(s)
}

// formatContextBlock renders the entities + relationships into the
// verbatim fenced block layout specified in §7.1 of the spec. Returns
// an empty string if both slices are empty — callers must NOT inject
// an empty fence.
func formatContextBlock(entities []recalledEntity, relationships []recalledRel) string {
	formattedEntities := make([]formatting.Entity, 0, len(entities))
	for _, entity := range entities {
		formattedEntities = append(formattedEntities, formatting.Entity{
			Name:        entity.Name,
			Type:        entity.Type,
			Description: entity.Description,
		})
	}
	formattedRelationships := make([]formatting.Relationship, 0, len(relationships))
	for _, relationship := range relationships {
		formattedRelationships = append(formattedRelationships, formatting.Relationship{
			Source:    relationship.Source,
			Predicate: relationship.Predicate,
			Target:    relationship.Target,
			Weight:    relationship.Weight,
		})
	}
	return formatting.FormatContextBlock(formattedEntities, formattedRelationships)
}
