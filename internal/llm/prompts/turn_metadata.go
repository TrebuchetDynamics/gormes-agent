package prompts

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/turnmetadata"

type TurnMetadataOptions = turnmetadata.TurnMetadataOptions

func BuildTurnMetadataBlock(opts TurnMetadataOptions) string {
	return turnmetadata.BuildTurnMetadataBlock(opts)
}
