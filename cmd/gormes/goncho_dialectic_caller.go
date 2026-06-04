package main

import (
	channelgoncho "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/channelgoncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type HermesDialecticCaller = channelgoncho.HermesDialecticCaller

func NewHermesDialecticCaller(client llm.Client, model string) *HermesDialecticCaller {
	return channelgoncho.NewHermesDialecticCaller(client, model)
}
