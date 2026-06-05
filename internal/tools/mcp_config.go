package tools

import (
	"time"

	mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
)

const (
	defaultMCPToolTimeout    = 120 * time.Second
	defaultMCPConnectTimeout = 60 * time.Second
	defaultMCPSamplingTime   = 30 * time.Second

	RedactedMCPConfigValue = mcptools.RedactedMCPConfigValue
)

type MCPTransport = mcptools.MCPTransport

const (
	MCPTransportStdio MCPTransport = mcptools.MCPTransportStdio
	MCPTransportHTTP  MCPTransport = mcptools.MCPTransportHTTP
)

type MCPConfigStatus = mcptools.MCPConfigStatus

const (
	MCPConfigStatusReady            MCPConfigStatus = mcptools.MCPConfigStatusReady
	MCPConfigStatusDisabled         MCPConfigStatus = mcptools.MCPConfigStatusDisabled
	MCPConfigStatusMissingSDK       MCPConfigStatus = mcptools.MCPConfigStatusMissingSDK
	MCPConfigStatusInvalidTransport MCPConfigStatus = mcptools.MCPConfigStatusInvalidTransport
	MCPConfigStatusInvalidEnv       MCPConfigStatus = mcptools.MCPConfigStatusInvalidEnv
	MCPConfigStatusInvalidConfig    MCPConfigStatus = mcptools.MCPConfigStatusInvalidConfig
)

type MCPConfigOptions = mcptools.MCPConfigOptions
type MCPConfigResolution = mcptools.MCPConfigResolution
type MCPServerDefinition = mcptools.MCPServerDefinition
type MCPSamplingConfig = mcptools.MCPSamplingConfig
type MCPServerStatus = mcptools.MCPServerStatus
type MCPConfigError = mcptools.MCPConfigError

func ParseMCPConfigYAML(data []byte, opts MCPConfigOptions) (MCPConfigResolution, error) {
	return mcptools.ParseMCPConfigYAML(data, opts)
}

func ParseMCPConfigJSON(data []byte, opts MCPConfigOptions) (MCPConfigResolution, error) {
	return mcptools.ParseMCPConfigJSON(data, opts)
}

func ResolveMCPConfig(raw any, opts MCPConfigOptions) (MCPConfigResolution, error) {
	return mcptools.ResolveMCPConfig(raw, opts)
}

func redactMCPString(value string) string { return mcptools.RedactString(value) }
