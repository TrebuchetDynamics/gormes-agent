package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newHermesWebContentProcessor(client llm.Client, model string) tools.WebContentProcessor {
	return tools.NewChunkedWebContentProcessor(client, model, tools.DefaultChunkedWebContentProcessorConfig())
}
