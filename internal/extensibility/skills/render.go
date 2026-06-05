package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"

func RenderBlock(skills []Skill) string {
	return document.RenderBlock(skills)
}

func RenderDocument(skill Skill) string {
	return document.RenderDocument(skill)
}
