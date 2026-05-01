package kernel

import (
	"context"
	"errors"
	"fmt"
)

var ErrGonchoUnavailable = errors.New("goncho not configured")

type GonchoStore interface {
	AppendTurn(ctx context.Context, peer, sessionKey, role, content string) error
	GetContext(ctx context.Context, sessionKey string, maxTokens int) (string, error)
}

func (k *Kernel) writeGonchoUserTurn(ctx context.Context, text string) {
	if k.cfg.Goncho == nil {
		return
	}
	if err := k.cfg.Goncho.AppendTurn(ctx, "user", k.sessionID, "user", text); err != nil {
		k.log.Warn("goncho user turn write failed", "err", err)
	}
}

func (k *Kernel) writeGonchoAssistantTurn(ctx context.Context, content string) {
	if k.cfg.Goncho == nil {
		return
	}
	if err := k.cfg.Goncho.AppendTurn(ctx, "gormes", k.sessionID, "assistant", content); err != nil {
		k.log.Warn("goncho assistant turn write failed", "err", err)
	}
}

func (k *Kernel) gonchoContext(ctx context.Context) string {
	if k.cfg.Goncho == nil {
		return ""
	}
	ctxStr, err := k.cfg.Goncho.GetContext(ctx, k.sessionID, 2000)
	if err != nil {
		if !errors.Is(err, ErrGonchoUnavailable) {
			k.log.Warn("goncho context read failed", "err", err)
		}
		return ""
	}
	if ctxStr == "" {
		return ""
	}
	return fmt.Sprintf("## Recent Conversation History\n%s", ctxStr)
}
