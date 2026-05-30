package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/isolation"

type IsolationLevel = isolation.IsolationLevel

const (
	IsolationProcess   = isolation.IsolationProcess
	IsolationContainer = isolation.IsolationContainer
	IsolationVM        = isolation.IsolationVM
)

type IsolationConfig = isolation.IsolationConfig
type IsolationModeError = isolation.IsolationModeError

func DefaultIsolationConfig() IsolationConfig {
	return isolation.DefaultIsolationConfig()
}

func ParseIsolationLevel(s string) (IsolationLevel, bool) {
	return isolation.ParseIsolationLevel(s)
}

func NewIsolationConfigFromMode(mode string, containerImage string, vmSocket string) (IsolationConfig, error) {
	return isolation.NewIsolationConfigFromMode(mode, containerImage, vmSocket)
}
