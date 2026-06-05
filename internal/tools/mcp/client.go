package mcp

import (
	"context"
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/callresult"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/circuit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/lifecycle"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
)

var ErrBreakerOpen = circuit.ErrBreakerOpen

type CircuitEvidence = circuit.Evidence

const (
	CircuitEvidenceOK                = circuit.EvidenceOK
	CircuitEvidenceServerUnreachable = circuit.EvidenceServerUnreachable
	CircuitEvidenceBreakerOpen       = circuit.EvidenceBreakerOpen
	CircuitEvidenceHalfOpenFailed    = circuit.EvidenceHalfOpenFailed
	CircuitEvidenceReconnectRequired = circuit.EvidenceReconnectRequired
	CircuitEvidenceReconnectReset    = circuit.EvidenceReconnectReset
)

const (
	DefaultCircuitBreakerThreshold = circuit.DefaultBreakerThreshold
	DefaultCircuitBreakerCooldown  = circuit.DefaultBreakerCooldown
	DefaultServerName              = circuit.DefaultServerName
)

type CircuitBreakerOptions = circuit.BreakerOptions

type CircuitBreaker = circuit.Breaker

func NewCircuitBreaker(opts CircuitBreakerOptions) *CircuitBreaker {
	return circuit.NewBreaker(opts)
}

type ToolCallFunc = circuit.ToolCallFunc

func CallWithCircuitBreaker(ctx context.Context, breaker *CircuitBreaker, server string, call ToolCallFunc) (CallResult, CircuitEvidence, error) {
	return circuit.CallWithBreaker(ctx, breaker, server, call)
}

type LifecycleEvent = lifecycle.Event

const (
	LifecycleEventNone      = lifecycle.EventNone
	LifecycleEventReconnect = lifecycle.EventReconnect
	LifecycleEventShutdown  = lifecycle.EventShutdown
)

type ServerLifecycle = lifecycle.Server

func NewServerLifecycle() *ServerLifecycle {
	return lifecycle.NewServer()
}

type ProbeSession = probe.Session

type ProbeConnector = probe.Connector

func ProbeServerTools(ctx context.Context, servers []MCPServerDefinition, connect ProbeConnector) map[string][]RawTool {
	return probe.ServerTools(ctx, servers, connect)
}

type CallResult = callresult.Result

func ParseCallResult(raw json.RawMessage) (CallResult, error) {
	return callresult.Parse(raw)
}
