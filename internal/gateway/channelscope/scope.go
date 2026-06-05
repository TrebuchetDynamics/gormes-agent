package channelscope

import (
	"fmt"
	"strings"
)

// SkillBinding mirrors Hermes channel_skill_bindings entries. Adapters accept
// both this typed form and map-shaped config decoded from TOML/YAML.
type SkillBinding struct {
	ID     string   `toml:"id" yaml:"id" json:"id"`
	Skill  string   `toml:"skill" yaml:"skill" json:"skill"`
	Skills []string `toml:"skills" yaml:"skills" json:"skills"`
}

// ResolveSkills returns the ordered, deduplicated skill list for an exact
// channel match, falling back to the parent channel when present.
func ResolveSkills(bindings any, channelID, parentID string) []string {
	entries := NormalizeSkillBindings(bindings)
	for _, id := range []string{strings.TrimSpace(channelID), strings.TrimSpace(parentID)} {
		if id == "" {
			continue
		}
		for _, entry := range entries {
			if strings.TrimSpace(entry.ID) != id {
				continue
			}
			if skills := CompactUniqueSkills(entry.Skills, entry.Skill); len(skills) > 0 {
				return skills
			}
			return nil
		}
	}
	return nil
}

// ResolvePrompt returns the exact channel prompt, falling back to the parent
// channel prompt. Blank prompts are treated as absent.
func ResolvePrompt(prompts any, channelID, parentID string) string {
	for _, id := range []string{strings.TrimSpace(channelID), strings.TrimSpace(parentID)} {
		if id == "" {
			continue
		}
		if prompt := promptForChannelID(prompts, id); prompt != "" {
			return prompt
		}
	}
	return ""
}

func NormalizeSkillBindings(raw any) []SkillBinding {
	switch v := raw.(type) {
	case nil:
		return nil
	case []SkillBinding:
		return append([]SkillBinding(nil), v...)
	case []map[string]any:
		out := make([]SkillBinding, 0, len(v))
		for _, item := range v {
			if entry, ok := skillBindingFromMap(item); ok {
				out = append(out, entry)
			}
		}
		return out
	case []any:
		out := make([]SkillBinding, 0, len(v))
		for _, item := range v {
			switch typed := item.(type) {
			case SkillBinding:
				out = append(out, typed)
			case map[string]any:
				if entry, ok := skillBindingFromMap(typed); ok {
					out = append(out, entry)
				}
			case map[string]string:
				if entry, ok := skillBindingFromStringMap(typed); ok {
					out = append(out, entry)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func skillBindingFromMap(item map[string]any) (SkillBinding, bool) {
	id, ok := configScalarString(item["id"])
	if !ok {
		return SkillBinding{}, false
	}
	entry := SkillBinding{ID: id}
	if skill, ok := configScalarString(item["skill"]); ok {
		entry.Skill = skill
	}
	entry.Skills = skillsFromAny(item["skills"])
	return entry, true
}

func skillBindingFromStringMap(item map[string]string) (SkillBinding, bool) {
	id := strings.TrimSpace(item["id"])
	if id == "" {
		return SkillBinding{}, false
	}
	return SkillBinding{ID: id, Skill: strings.TrimSpace(item["skill"])}, true
}

func skillsFromAny(raw any) []string {
	switch v := raw.(type) {
	case nil:
		return nil
	case string:
		return []string{v}
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func CompactUniqueSkills(values []string, single string) []string {
	if len(values) == 0 && strings.TrimSpace(single) != "" {
		values = []string{single}
	}
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

func promptForChannelID(prompts any, id string) string {
	switch v := prompts.(type) {
	case nil:
		return ""
	case map[string]string:
		return strings.TrimSpace(v[id])
	case map[string]any:
		if raw, ok := v[id]; ok {
			prompt, _ := configScalarString(raw)
			return prompt
		}
	case map[any]any:
		if raw, ok := v[id]; ok {
			prompt, _ := configScalarString(raw)
			return prompt
		}
	}
	return ""
}

func configScalarString(raw any) (string, bool) {
	if raw == nil {
		return "", false
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return "", false
	}
	return value, true
}

func ChannelPromptBlock(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	return "## Channel Prompt\n" + prompt
}

func PrependChannelPromptBlock(sessionBlock, prompt string) string {
	block := ChannelPromptBlock(prompt)
	if block == "" {
		return sessionBlock
	}
	if strings.TrimSpace(sessionBlock) == "" {
		return block
	}
	return block + "\n\n" + sessionBlock
}
