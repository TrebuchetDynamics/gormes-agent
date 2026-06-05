package cli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/pty"
)

const (
	DefaultPtyReadTimeout   = pty.DefaultPtyReadTimeout
	DefaultPtyReadChunkSize = pty.DefaultPtyReadChunkSize
	MaxPtyWriteBytes        = pty.MaxPtyWriteBytes
	MaxPtyCols              = pty.MaxPtyCols
	MaxPtyRows              = pty.MaxPtyRows
)

var ErrPtyUnavailable = pty.ErrPtyUnavailable
var ErrInvalidPtyMessage = pty.ErrInvalidPtyMessage

type PtyUnavailableError = pty.PtyUnavailableError
type PtyInvalidMessageError = pty.PtyInvalidMessageError
type PtySize = pty.PtySize
type PtySpawnRequest = pty.PtySpawnRequest
type PtySession = pty.PtySession
type PtySpawnFunc = pty.PtySpawnFunc
type PtyAdapterConfig = pty.PtyAdapterConfig
type PtyAdapter = pty.PtyAdapter

func NewPtyAdapter(ctx context.Context, req PtySpawnRequest, cfg PtyAdapterConfig) (*PtyAdapter, error) {
	return pty.NewPtyAdapter(ctx, req, cfg)
}

func NewPtyAdapterForSession(session PtySession) *PtyAdapter {
	return pty.NewPtyAdapterForSession(session)
}

func PtyAvailable() bool { return pty.PtyAvailable() }
