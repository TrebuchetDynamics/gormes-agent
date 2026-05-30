package pty

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/pty/bridge"
)

const (
	DefaultPtyReadTimeout   = bridge.DefaultPtyReadTimeout
	DefaultPtyReadChunkSize = bridge.DefaultPtyReadChunkSize
	MaxPtyWriteBytes        = bridge.MaxPtyWriteBytes
	MaxPtyCols              = bridge.MaxPtyCols
	MaxPtyRows              = bridge.MaxPtyRows
)

var (
	ErrPtyUnavailable    = bridge.ErrPtyUnavailable
	ErrInvalidPtyMessage = bridge.ErrInvalidPtyMessage
)

type PtyUnavailableError = bridge.PtyUnavailableError
type PtyInvalidMessageError = bridge.PtyInvalidMessageError
type PtySize = bridge.PtySize
type PtySpawnRequest = bridge.PtySpawnRequest
type PtySession = bridge.PtySession
type PtySpawnFunc = bridge.PtySpawnFunc
type PtyAdapterConfig = bridge.PtyAdapterConfig
type PtyAdapter = bridge.PtyAdapter

func NewPtyAdapter(ctx context.Context, req PtySpawnRequest, cfg PtyAdapterConfig) (*PtyAdapter, error) {
	return bridge.NewPtyAdapter(ctx, req, cfg)
}

func NewPtyAdapterForSession(session PtySession) *PtyAdapter {
	return bridge.NewPtyAdapterForSession(session)
}

func PtyAvailable() bool { return bridge.PtyAvailable() }
