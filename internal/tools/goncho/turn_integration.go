package gonchotools

import (
	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho/integration"
)

// TurnIntegration wires local Goncho memory into a normal kernel turn without
// requiring Python, hosted Honcho, or a loopback HTTP service.
type TurnIntegration = integration.TurnIntegration

type TurnIntegrationStatus = integration.TurnIntegrationStatus

func NewTurnIntegration(service *goncho.Service, peer string) *TurnIntegration {
	return integration.NewTurnIntegration(service, peer)
}
