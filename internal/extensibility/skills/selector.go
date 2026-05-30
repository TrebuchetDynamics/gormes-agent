package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/selection"

func Select(skills []Skill, query string, max int) []Skill {
	return selection.Select(skills, query, max)
}

func skillNames(skills []Skill) []string {
	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		out = append(out, skill.Name)
	}
	return out
}
