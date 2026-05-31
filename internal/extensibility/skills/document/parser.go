package document

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document/parsing"

// Parse converts a SKILL.md document into a typed Skill.
func Parse(raw []byte, maxBytes int) (Skill, error) {
	return parsing.Parse(raw, maxBytes)
}
