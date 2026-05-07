package gateway

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

// ChannelSkillBinding mirrors Hermes channel_skill_bindings entries. Adapters
// accept both this typed form and map-shaped config decoded from TOML/YAML.
type ChannelSkillBinding struct {
	ID     string   `toml:"id" yaml:"id" json:"id"`
	Skill  string   `toml:"skill" yaml:"skill" json:"skill"`
	Skills []string `toml:"skills" yaml:"skills" json:"skills"`
}

// ResolveChannelSkills returns the ordered, deduplicated skill list for an
// exact channel match, falling back to the parent channel when present.
func ResolveChannelSkills(bindings any, channelID, parentID string) []string {
	entries := normalizeChannelSkillBindings(bindings)
	for _, id := range []string{strings.TrimSpace(channelID), strings.TrimSpace(parentID)} {
		if id == "" {
			continue
		}
		for _, entry := range entries {
			if strings.TrimSpace(entry.ID) != id {
				continue
			}
			if skills := compactUniqueSkills(entry.Skills, entry.Skill); len(skills) > 0 {
				return skills
			}
			return nil
		}
	}
	return nil
}

// ResolveChannelPrompt returns the exact channel prompt, falling back to the
// parent channel prompt. Blank prompts are treated as absent.
func ResolveChannelPrompt(prompts any, channelID, parentID string) string {
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

func normalizeChannelSkillBindings(raw any) []ChannelSkillBinding {
	switch v := raw.(type) {
	case nil:
		return nil
	case []ChannelSkillBinding:
		return append([]ChannelSkillBinding(nil), v...)
	case []map[string]any:
		out := make([]ChannelSkillBinding, 0, len(v))
		for _, item := range v {
			if entry, ok := channelSkillBindingFromMap(item); ok {
				out = append(out, entry)
			}
		}
		return out
	case []any:
		out := make([]ChannelSkillBinding, 0, len(v))
		for _, item := range v {
			switch typed := item.(type) {
			case ChannelSkillBinding:
				out = append(out, typed)
			case map[string]any:
				if entry, ok := channelSkillBindingFromMap(typed); ok {
					out = append(out, entry)
				}
			case map[string]string:
				if entry, ok := channelSkillBindingFromStringMap(typed); ok {
					out = append(out, entry)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func channelSkillBindingFromMap(item map[string]any) (ChannelSkillBinding, bool) {
	id := strings.TrimSpace(fmt.Sprint(item["id"]))
	if id == "" {
		return ChannelSkillBinding{}, false
	}
	entry := ChannelSkillBinding{ID: id}
	if skill := strings.TrimSpace(fmt.Sprint(item["skill"])); skill != "" && skill != "<nil>" {
		entry.Skill = skill
	}
	entry.Skills = skillsFromAny(item["skills"])
	return entry, true
}

func channelSkillBindingFromStringMap(item map[string]string) (ChannelSkillBinding, bool) {
	id := strings.TrimSpace(item["id"])
	if id == "" {
		return ChannelSkillBinding{}, false
	}
	return ChannelSkillBinding{ID: id, Skill: strings.TrimSpace(item["skill"])}, true
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

func compactUniqueSkills(values []string, single string) []string {
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
			return strings.TrimSpace(fmt.Sprint(raw))
		}
	case map[any]any:
		if raw, ok := v[id]; ok {
			return strings.TrimSpace(fmt.Sprint(raw))
		}
	}
	return ""
}

func channelPromptBlock(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	return "## Channel Prompt\n" + prompt
}

func prependChannelPromptBlock(sessionBlock, prompt string) string {
	block := channelPromptBlock(prompt)
	if block == "" {
		return sessionBlock
	}
	if strings.TrimSpace(sessionBlock) == "" {
		return block
	}
	return block + "\n\n" + sessionBlock
}

func (m *Manager) applyChannelAutoSkills(route agentRuntimeRoute, snapshot agentRuntimeSnapshot, autoSkills []string) agentRuntimeSnapshot {
	autoSkills = compactUniqueSkills(autoSkills, "")
	if len(autoSkills) == 0 || m.cfg.SkillRuntime == nil {
		return snapshot
	}
	names := append([]string(nil), route.Decision.Skills...)
	names = append(names, autoSkills...)
	names = compactUniqueSkills(names, "")
	reg := snapshot.Tools
	if reg == nil {
		reg = m.cfg.ToolRegistry
	}
	snapshot.Skills = agentSkillProvider{
		runtime: m.cfg.SkillRuntime,
		opts: skills.RuntimeOptions{
			AllowedSkillNames: policyNameMap(names),
			AvailableTools:    registryNames(reg),
		},
	}
	snapshot.Skills = channelSkillProvider{
		agentSkillProvider: snapshot.Skills.(agentSkillProvider),
		autoSkills:         autoSkills,
	}
	return snapshot
}

type channelSkillProvider struct {
	agentSkillProvider
	autoSkills []string
}

func (p channelSkillProvider) BuildSkillBlock(ctx context.Context, userMessage string) (string, []string, error) {
	query := strings.TrimSpace(strings.Join(p.autoSkills, " ") + " " + userMessage)
	return p.agentSkillProvider.BuildSkillBlock(ctx, query)
}

// SkillGroupRefreshResult is redacted evidence from refreshing adapter-owned
// skill command/autocomplete state after the gateway skill catalog changes.
type SkillGroupRefreshResult struct {
	Channel string
	Count   int
	Hidden  int
	Error   string
}

// SkillGroupRefresher is implemented by adapters that cache skill command
// state outside the shared Gormes skill runtime.
type SkillGroupRefresher interface {
	RefreshSkillGroup(context.Context) (SkillGroupRefreshResult, error)
}

// RefreshSkillGroups calls every registered adapter that can refresh cached
// skill command state. Failures are returned as evidence and do not stop other
// adapters from refreshing.
func (m *Manager) RefreshSkillGroups(ctx context.Context) []SkillGroupRefreshResult {
	m.mu.Lock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.Unlock()
	sort.Slice(channels, func(i, j int) bool { return channels[i].Name() < channels[j].Name() })

	results := []SkillGroupRefreshResult{}
	for _, ch := range channels {
		refresher, ok := ch.(SkillGroupRefresher)
		if !ok {
			continue
		}
		result, err := refresher.RefreshSkillGroup(ctx)
		if strings.TrimSpace(result.Channel) == "" {
			result.Channel = ch.Name()
		}
		if err != nil {
			result.Error = sanitizeConfigReloadError(err)
		}
		results = append(results, result)
	}
	return results
}
