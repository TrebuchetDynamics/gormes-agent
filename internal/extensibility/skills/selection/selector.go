package selection

import (
	"sort"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"
)

func Select(skills []document.Skill, query string, max int) []document.Skill {
	if len(skills) == 0 {
		return nil
	}
	if max <= 0 {
		max = document.DefaultSelectionCap
	}
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil
	}

	type scoredSkill struct {
		skill document.Skill
		score int
	}

	scored := make([]scoredSkill, 0, len(skills))
	for _, skill := range skills {
		scored = append(scored, scoredSkill{
			skill: skill,
			score: scoreSkill(skill, tokens),
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].skill.Name != scored[j].skill.Name {
			return scored[i].skill.Name < scored[j].skill.Name
		}
		return scored[i].skill.Path < scored[j].skill.Path
	})

	out := make([]document.Skill, 0, max)
	for _, scoredSkill := range scored {
		if scoredSkill.score <= 0 {
			break
		}
		out = append(out, scoredSkill.skill)
		if len(out) == max {
			break
		}
	}
	return out
}

func tokenize(query string) []string {
	return strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func scoreSkill(skill document.Skill, tokens []string) int {
	if len(tokens) == 0 {
		return 0
	}
	name := strings.ToLower(skill.Name)
	description := strings.ToLower(skill.Description)
	body := strings.ToLower(skill.Body)
	tagBag := strings.ToLower(strings.Join(skill.HermesTags, " "))
	triggerBag := lowerJoin(skill.Triggers)
	query := strings.ToLower(strings.Join(tokens, " "))

	score := 0
	for _, trigger := range skill.Triggers {
		trigger = strings.ToLower(strings.TrimSpace(trigger))
		if trigger != "" && strings.Contains(query, trigger) {
			score += 25
		}
	}
	for _, token := range tokens {
		switch {
		case strings.Contains(name, token):
			score += 10
		case strings.Contains(description, token):
			score += 4
		case strings.Contains(tagBag, token):
			score += 3
		case strings.Contains(triggerBag, token):
			score += 2
		case strings.Contains(body, token):
			score++
		}
	}
	return score
}

func lowerJoin(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = strings.ToLower(value)
	}
	return strings.Join(out, " ")
}
