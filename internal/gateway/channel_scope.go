package gateway

import (
	"context"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	gatewaychannelscope "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/channelscope"
)

// ChannelSkillBinding mirrors Hermes channel_skill_bindings entries. Adapters
// accept both this typed form and map-shaped config decoded from TOML/YAML.
type ChannelSkillBinding = gatewaychannelscope.SkillBinding

// ResolveChannelSkills returns the ordered, deduplicated skill list for an
// exact channel match, falling back to the parent channel when present.
func ResolveChannelSkills(bindings any, channelID, parentID string) []string {
	return gatewaychannelscope.ResolveSkills(bindings, channelID, parentID)
}

// ResolveChannelPrompt returns the exact channel prompt, falling back to the
// parent channel prompt. Blank prompts are treated as absent.
func ResolveChannelPrompt(prompts any, channelID, parentID string) string {
	return gatewaychannelscope.ResolvePrompt(prompts, channelID, parentID)
}

func normalizeChannelSkillBindings(raw any) []ChannelSkillBinding {
	return gatewaychannelscope.NormalizeSkillBindings(raw)
}

func compactUniqueSkills(values []string, single string) []string {
	return gatewaychannelscope.CompactUniqueSkills(values, single)
}

func channelPromptBlock(prompt string) string {
	return gatewaychannelscope.ChannelPromptBlock(prompt)
}

func prependChannelPromptBlock(sessionBlock, prompt string) string {
	return gatewaychannelscope.PrependChannelPromptBlock(sessionBlock, prompt)
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
