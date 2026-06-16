package assembly

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/skillprompt"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/turnmetadata"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

type PromptAssemblyOptions struct {
	IdentityOpts           contextfiles.IdentityLoaderOptions
	ContextFilesOpts       contextfiles.ContextFilesOptions
	HasMemoryTool          bool
	HasSessionSearch       bool
	SkillsOpts             skillprompt.SkillsPromptOptions
	ModelGuidanceOpts      guidance.ModelPromptGuidanceOptions
	TurnMetadataOpts       turnmetadata.TurnMetadataOptions
	DurableUserContextOpts contextfiles.DurableUserContextOptions
	Personality            string // optional personality system prompt override
}

type PromptAssemblyResult struct {
	Prompt string
	Blocks []PromptBlockEvidence
	Errors []string
}

type PromptBlockEvidence struct {
	Block    string
	Included bool
	Reason   string
}

func BuildSystemPrompt(opts PromptAssemblyOptions) PromptAssemblyResult {
	var blocks []string
	var evidence []PromptBlockEvidence
	var errors []string

	identity := contextfiles.LoadAgentIdentity(opts.IdentityOpts)
	blocks = append(blocks, identity.Identity)
	evidence = append(evidence, PromptBlockEvidence{
		Block:    "identity",
		Included: true,
		Reason:   identity.Source,
	})

	if p := strings.TrimSpace(opts.Personality); p != "" {
		block := "<personality>\n" + p + "\n</personality>"
		blocks = append(blocks, block)
		evidence = append(evidence, PromptBlockEvidence{
			Block: "personality", Included: true, Reason: "config",
		})
	}

	if ctx, report := contextfiles.BuildContextFilesPrompt(opts.ContextFilesOpts); ctx != "" {
		blocks = append(blocks, ctx)
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "context_files",
			Included: true,
			Reason:   report.Project.Source,
		})
	} else {
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "context_files",
			Included: false,
			Reason:   report.Project.Error,
		})
	}

	if durable, report := contextfiles.BuildDurableUserContextPrompt(opts.DurableUserContextOpts); durable != "" {
		blocks = append(blocks, durable)
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "durable_user_context",
			Included: true,
			Reason:   report.User.Source,
		})
	} else {
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "durable_user_context",
			Included: false,
			Reason:   report.User.Error,
		})
	}

	mem := guidance.BuildMemoryGuidance(opts.HasMemoryTool)
	if mem.Injected {
		blocks = append(blocks, mem.Guidance)
	}
	evidence = append(evidence, PromptBlockEvidence{
		Block:    "memory_guidance",
		Included: mem.Injected,
		Reason:   mem.Evidence,
	})

	search := guidance.BuildSessionSearchGuidance(opts.HasSessionSearch)
	if search.Injected {
		blocks = append(blocks, search.Guidance)
	}
	evidence = append(evidence, PromptBlockEvidence{
		Block:    "session_search_guidance",
		Included: search.Injected,
		Reason:   search.Evidence,
	})

	if skills, _, err := skillprompt.BuildSkillsSystemPrompt(opts.SkillsOpts); skills != "" {
		blocks = append(blocks, skills)
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "skills_snapshot",
			Included: true,
			Reason:   "skills_loaded",
		})
	} else if err != nil {
		errText := promptAssemblyErrorText(err)
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "skills_snapshot",
			Included: false,
			Reason:   errText,
		})
		errors = append(errors, errText)
	} else {
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "skills_snapshot",
			Included: false,
			Reason:   "no skills",
		})
	}

	model := guidance.BuildModelPromptGuidance(opts.ModelGuidanceOpts)
	if model.Guidance != "" {
		blocks = append(blocks, model.Guidance)
	}
	evidence = append(evidence, PromptBlockEvidence{
		Block:    "model_guidance",
		Included: model.Guidance != "",
		Reason:   strings.Join(model.Evidence, ", "),
	})

	meta := turnmetadata.BuildTurnMetadataBlock(opts.TurnMetadataOpts)
	if meta != "" {
		blocks = append(blocks, meta)
		evidence = append(evidence, PromptBlockEvidence{
			Block:    "turn_metadata",
			Included: true,
			Reason:   "metadata_injected",
		})
	}

	return PromptAssemblyResult{
		Prompt: strings.Join(blocks, "\n\n"),
		Blocks: evidence,
		Errors: errors,
	}
}

func promptAssemblyErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := redaction.RedactSecrets(err.Error())
	msg = strings.NewReplacer("`", "'", "*", "'", "#", "＃").Replace(msg)
	return strings.Join(strings.Fields(msg), " ")
}
