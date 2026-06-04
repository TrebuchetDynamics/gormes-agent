package webcontent

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func NewProcessor(client llm.Client, model string) tools.WebContentProcessor {
	return tools.NewChunkedWebContentProcessor(client, model, tools.DefaultChunkedWebContentProcessorConfig())
}
