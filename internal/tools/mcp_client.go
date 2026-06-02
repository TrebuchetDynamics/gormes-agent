package tools

import (
	"context"
	"encoding/json"
	"errors"

	mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
)

var ErrMCPBreakerOpen = mcptools.ErrBreakerOpen

type MCPCircuitEvidence = mcptools.CircuitEvidence

const (
	MCPCircuitEvidenceOK                = mcptools.CircuitEvidenceOK
	MCPCircuitEvidenceServerUnreachable = mcptools.CircuitEvidenceServerUnreachable
	MCPCircuitEvidenceBreakerOpen       = mcptools.CircuitEvidenceBreakerOpen
	MCPCircuitEvidenceHalfOpenFailed    = mcptools.CircuitEvidenceHalfOpenFailed
	MCPCircuitEvidenceReconnectRequired = mcptools.CircuitEvidenceReconnectRequired
	MCPCircuitEvidenceReconnectReset    = mcptools.CircuitEvidenceReconnectReset
)

const (
	defaultMCPCircuitBreakerThreshold = 3
	defaultMCPCircuitBreakerCooldown  = mcptools.DefaultCircuitBreakerCooldown
	defaultMCPServerName              = mcptools.DefaultServerName
)

type MCPCircuitBreakerOptions = mcptools.CircuitBreakerOptions
type MCPCircuitBreaker = mcptools.CircuitBreaker

func NewMCPCircuitBreaker(opts MCPCircuitBreakerOptions) *MCPCircuitBreaker {
	return mcptools.NewCircuitBreaker(opts)
}

type MCPToolCallFunc = mcptools.ToolCallFunc

func CallMCPWithCircuitBreaker(ctx context.Context, breaker *MCPCircuitBreaker, server string, call MCPToolCallFunc) (MCPCallResult, MCPCircuitEvidence, error) {
	return mcptools.CallWithCircuitBreaker(ctx, breaker, server, call)
}

type MCPLifecycleEvent = mcptools.LifecycleEvent

const (
	MCPLifecycleEventNone      = mcptools.LifecycleEventNone
	MCPLifecycleEventReconnect = mcptools.LifecycleEventReconnect
	MCPLifecycleEventShutdown  = mcptools.LifecycleEventShutdown
)

type MCPServerLifecycle = mcptools.ServerLifecycle

func NewMCPServerLifecycle() *MCPServerLifecycle { return mcptools.NewServerLifecycle() }

type MCPProbeSession = mcptools.ProbeSession
type MCPProbeConnector = mcptools.ProbeConnector

func ProbeMCPServerTools(ctx context.Context, servers []MCPServerDefinition, connect MCPProbeConnector) map[string][]MCPRawTool {
	return mcptools.ProbeServerTools(ctx, servers, connect)
}

type MCPCallResult = mcptools.CallResult

type OSVEvidence = mcptools.OSVEvidence

type OSVPackageQuery = mcptools.OSVPackageQuery

type OSVVulnerability = mcptools.OSVVulnerability

type OSVClient = mcptools.OSVClient

type OSVCheckResult = mcptools.OSVCheckResult

const (
	OSVEvidenceAllowed      = mcptools.OSVEvidenceAllowed
	OSVEvidenceSkipped      = mcptools.OSVEvidenceSkipped
	OSVEvidenceFailOpen     = mcptools.OSVEvidenceFailOpen
	OSVEvidenceMalwareFound = mcptools.OSVEvidenceMalwareFound
)

func CheckMCPServerPackageLaunch(ctx context.Context, server MCPServerDefinition, client OSVClient) OSVCheckResult {
	return mcptools.CheckMCPServerPackageLaunch(ctx, server, client)
}

func CheckPackageLaunchForMalware(ctx context.Context, command string, args []string, client OSVClient) OSVCheckResult {
	return mcptools.CheckPackageLaunchForMalware(ctx, command, args, client)
}

func parseMCPCallResult(raw json.RawMessage) (MCPCallResult, error) {
	return mcptools.ParseCallResult(raw)
}

func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments map[string]any) (MCPCallResult, error) {
	if arguments == nil {
		arguments = map[string]any{}
	}
	params := map[string]any{
		"name":      name,
		"arguments": arguments,
	}
	var raw json.RawMessage
	if err := c.call(ctx, "tools/call", params, &raw); err != nil {
		return MCPCallResult{}, err
	}
	return parseMCPCallResult(raw)
}

var _ = errors.Is
