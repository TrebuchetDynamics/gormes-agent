package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"

type TurnMetadataOptions = prompts.TurnMetadataOptions

func BuildTurnMetadataBlock(opts TurnMetadataOptions) string {
	return prompts.BuildTurnMetadataBlock(opts)
}
