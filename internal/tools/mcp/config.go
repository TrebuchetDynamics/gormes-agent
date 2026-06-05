package mcp

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
)

const RedactedMCPConfigValue = config.RedactedMCPConfigValue

type MCPTransport = config.MCPTransport

const (
	MCPTransportStdio = config.MCPTransportStdio
	MCPTransportHTTP  = config.MCPTransportHTTP
)

type MCPConfigStatus = config.MCPConfigStatus

const (
	MCPConfigStatusReady            = config.MCPConfigStatusReady
	MCPConfigStatusDisabled         = config.MCPConfigStatusDisabled
	MCPConfigStatusMissingSDK       = config.MCPConfigStatusMissingSDK
	MCPConfigStatusInvalidTransport = config.MCPConfigStatusInvalidTransport
	MCPConfigStatusInvalidEnv       = config.MCPConfigStatusInvalidEnv
	MCPConfigStatusInvalidConfig    = config.MCPConfigStatusInvalidConfig
)

type MCPConfigOptions = config.MCPConfigOptions

type MCPConfigResolution = config.MCPConfigResolution

type MCPServerDefinition = config.MCPServerDefinition

type MCPSamplingConfig = config.MCPSamplingConfig

type MCPServerStatus = config.MCPServerStatus

type MCPConfigError = config.MCPConfigError

func ParseMCPConfigYAML(data []byte, opts MCPConfigOptions) (MCPConfigResolution, error) {
	return config.ParseMCPConfigYAML(data, opts)
}

func ParseMCPConfigJSON(data []byte, opts MCPConfigOptions) (MCPConfigResolution, error) {
	return config.ParseMCPConfigJSON(data, opts)
}

func ResolveMCPConfig(raw any, opts MCPConfigOptions) (MCPConfigResolution, error) {
	return config.ResolveMCPConfig(raw, opts)
}

func RedactString(value string) string { return config.RedactString(value) }
