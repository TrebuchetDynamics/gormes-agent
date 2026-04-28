package gonchotools

import (
	"context"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TurnIntegration wires local Goncho memory into a normal kernel turn without
// requiring Python, hosted Honcho, or a loopback HTTP service.
type TurnIntegration struct {
	service *goncho.Service
	peer    string

	mu     sync.Mutex
	status TurnIntegrationStatus
}

type TurnIntegrationStatus struct {
	RecallEvidence string `json:"recall_evidence,omitempty"`
	ToolsEvidence  string `json:"tools_evidence,omitempty"`
}

func NewTurnIntegration(service *goncho.Service, peer string) *TurnIntegration {
	return &TurnIntegration{
		service: service,
		peer:    strings.TrimSpace(peer),
		status: TurnIntegrationStatus{
			RecallEvidence: "goncho_recall_pending",
			ToolsEvidence:  "honcho_tools_pending",
		},
	}
}

func (i *TurnIntegration) RegisterTools(reg *tools.Registry) TurnIntegrationStatus {
	if i == nil || i.service == nil || reg == nil {
		i.setToolsEvidence("honcho_tools_unavailable")
		return i.Status()
	}
	RegisterHonchoTools(reg, i.service)
	i.setToolsEvidence("honcho_tools_ready")
	return i.Status()
}

func (i *TurnIntegration) RecallProvider() kernel.RecallProvider {
	return i
}

func (i *TurnIntegration) GetContext(ctx context.Context, params kernel.RecallParams) string {
	if i == nil || i.service == nil {
		i.setRecallEvidence("goncho_recall_unavailable")
		return ""
	}
	peer := firstTurnIntegrationValue(i.peer, params.ChatKey)
	if peer == "" {
		i.setRecallEvidence("goncho_recall_unavailable")
		return ""
	}
	got, err := i.service.Context(ctx, goncho.ContextParams{
		Peer:       peer,
		Query:      params.UserMessage,
		SessionKey: params.SessionID,
		MaxTokens:  400,
	})
	if err != nil {
		i.setRecallEvidence("goncho_recall_unavailable")
		return ""
	}
	i.setRecallEvidence("goncho_recall_ready")
	return "Goncho memory context:\n" + got.Representation
}

func (i *TurnIntegration) Status() TurnIntegrationStatus {
	if i == nil {
		return TurnIntegrationStatus{
			RecallEvidence: "goncho_recall_unavailable",
			ToolsEvidence:  "honcho_tools_unavailable",
		}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.status
}

func (i *TurnIntegration) setRecallEvidence(evidence string) {
	if i == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.status.RecallEvidence = evidence
}

func (i *TurnIntegration) setToolsEvidence(evidence string) {
	if i == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.status.ToolsEvidence = evidence
}

func firstTurnIntegrationValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
