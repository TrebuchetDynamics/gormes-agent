package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/hub"

// SkillMeta is a registry-agnostic metadata record returned by HubProvider
// searches. It is the canonical shape for unified hub results before any
// gateway-specific augmentation (scores, tags, etc.) is applied.
type SkillMeta = hub.SkillMeta

// HubProvider is the minimal interface a skill-hub source must implement to
// participate in UnifiedSearch. It mirrors the "SkillSource" ABC contract from
// Python's tools/skills_hub.py: sources are identified, queried, and graded by
// trust level.
type HubProvider = hub.HubProvider

// HubRegistry collects multiple HubProvider instances and exposes a unified
// search that merges, deduplicates, and returns combined results.
type HubRegistry = hub.HubRegistry

func NewHubRegistry(providers ...HubProvider) *HubRegistry {
	return hub.NewHubRegistry(providers...)
}

// SkillsShProvider is the singleton instance of the skills.sh stub provider.
var SkillsShProvider HubProvider = hub.SkillsShProvider
