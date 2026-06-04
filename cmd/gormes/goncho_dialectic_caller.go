package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type HermesDialecticCaller = gormescli.HermesDialecticCaller

func NewHermesDialecticCaller(client llm.Client, model string) *HermesDialecticCaller {
	return gormescli.NewHermesDialecticCaller(client, model)
}
