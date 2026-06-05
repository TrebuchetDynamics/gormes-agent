package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/extraction"

// ValidatedOutput is the cleaned, whitelist-conformant result of
// validating raw LLM extractor output. Every field is safe to pass to
// the graph upsert layer without further sanitation.
type ValidatedOutput = extraction.ValidatedOutput

type ValidatedEntity = extraction.ValidatedEntity
type ValidatedRelationship = extraction.ValidatedRelationship

// ValidateExtractorOutput parses + sanitizes raw LLM JSON into a
// ValidatedOutput. Malformed JSON returns an error; everything else
// coerces silently (invalid types -> OTHER, unknown predicates ->
// RELATED_TO, orphan relationships dropped, etc.).
func ValidateExtractorOutput(raw []byte) (ValidatedOutput, error) {
	return extraction.ValidateExtractorOutput(raw)
}
