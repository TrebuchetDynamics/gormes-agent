package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newHermesWebContentProcessor(client llm.Client, model string) tools.WebContentProcessor {
	return gormescli.NewWebContentProcessor(client, model)
}
