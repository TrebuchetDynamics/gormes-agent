package pty

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/pty/sidecar"

const DefaultPtyChatSidecarQueueSize = sidecar.DefaultPtyChatSidecarQueueSize

type PtyChatSidecarSink = sidecar.PtyChatSidecarSink
type PtyChatSidecarConfig = sidecar.PtyChatSidecarConfig
type PtyChatSidecar = sidecar.PtyChatSidecar

func NewPtyChatSidecar(cfg PtyChatSidecarConfig) *PtyChatSidecar {
	return sidecar.NewPtyChatSidecar(cfg)
}
