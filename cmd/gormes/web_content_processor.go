package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newHermesWebContentProcessor(client hermes.Client, model string) tools.WebContentProcessor {
	return tools.NewChunkedWebContentProcessor(client, model, tools.DefaultChunkedWebContentProcessorConfig())
}
