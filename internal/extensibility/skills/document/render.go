package document

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document/rendering"

func RenderBlock(skills []Skill) string {
	return rendering.RenderBlock(skills)
}

func RenderDocument(skill Skill) string {
	return rendering.RenderDocument(skill)
}
