package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"

// Parse converts a SKILL.md document into a typed Skill.
func Parse(raw []byte, maxBytes int) (Skill, error) {
	return document.Parse(raw, maxBytes)
}
