package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/pty"

const DefaultPtyChatSidecarQueueSize = pty.DefaultPtyChatSidecarQueueSize

type PtyChatSidecarSink = pty.PtyChatSidecarSink
type PtyChatSidecarConfig = pty.PtyChatSidecarConfig
type PtyChatSidecar = pty.PtyChatSidecar

func NewPtyChatSidecar(cfg PtyChatSidecarConfig) *PtyChatSidecar {
	return pty.NewPtyChatSidecar(cfg)
}
