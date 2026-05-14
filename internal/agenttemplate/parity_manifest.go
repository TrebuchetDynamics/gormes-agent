package agenttemplate

type TemplatePairStatus string

const (
	TemplatePairCovered         TemplatePairStatus = "covered"
	TemplatePairOwnedDivergence TemplatePairStatus = "owned_divergence"
)

// TemplatePair records the source-backed parity classification for one
// first-run agent context template. Covered entries track a direct Hermes
// source; owned divergences document Gormes templates that Hermes consumes or
// inspires but does not seed as matching files.
type TemplatePair struct {
	Path          string
	Status        TemplatePairStatus
	HermesSources []string
	GormesSources []string
	Contract      string
}

func TemplatePairManifest() []TemplatePair {
	return []TemplatePair{
		{
			Path:   "SOUL.md",
			Status: TemplatePairCovered,
			HermesSources: []string{
				"hermes_cli/default_soul.py",
				"hermes_cli/config.py",
			},
			GormesSources: []string{
				"internal/hermes/default_soul.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Gormes seeds SOUL.md from the Hermes DEFAULT_SOUL_MD contract with only the declared Gormes product-identity substitution, then appends Gormes-owned operating and boundary guidance.",
		},
		{
			Path:   "AGENTS.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes consumes AGENTS.md as project context but does not seed a default AGENTS.md; Gormes owns a starter workspace contract so clean installs have editable project instructions.",
		},
		{
			Path:   "IDENTITY.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes has no matching seeded IDENTITY.md; Gormes owns this additive operational context file for stable agent and workspace identity facts.",
		},
		{
			Path:   "TOOLS.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes exposes tool guidance through prompt assembly rather than a seeded TOOLS.md file; Gormes owns this additive operational context file for workspace tool and verification rules.",
		},
		{
			Path:   "memory/USER.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/durable_user_context.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes supports durable user context in prompt assembly but does not seed this exact memory/USER.md template; Gormes owns the editable durable-user starter file.",
		},
		{
			Path:   "memory/MEMORY.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/durable_user_context.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes supports durable memory context in prompt assembly but does not seed this exact memory/MEMORY.md template; Gormes owns the editable durable-memory starter file.",
		},
	}
}
