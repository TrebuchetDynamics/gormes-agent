package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SandboxProvider manages the lifecycle of sandbox environments.
type SandboxProvider interface {
	Acquire(ctx context.Context, sessionID string) (Sandbox, error)
	Get(ctx context.Context, sessionID string) (Sandbox, error)
	Release(ctx context.Context, sessionID string) error
	Shutdown(ctx context.Context) error
}

// LocalSandboxConfig configures a LocalSandboxProvider.
type LocalSandboxConfig struct {
	BaseDir string
}

// LocalSandboxProvider is a filesystem-backed SandboxProvider.
type LocalSandboxProvider struct {
	mu       sync.Mutex
	baseDir  string
	sandboxes map[string]*sandbox
}

// NewLocalSandboxProvider creates a new LocalSandboxProvider.
func NewLocalSandboxProvider(cfg LocalSandboxConfig) *LocalSandboxProvider {
	return &LocalSandboxProvider{
		baseDir:   cfg.BaseDir,
		sandboxes: make(map[string]*sandbox),
	}
}

func (p *LocalSandboxProvider) Acquire(ctx context.Context, sessionID string) (Sandbox, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if sb, ok := p.sandboxes[sessionID]; ok {
		return sb, nil
	}

	sbDir := filepath.Join(p.baseDir, "sessions", sessionID)
	if err := os.MkdirAll(sbDir, 0755); err != nil {
		return nil, fmt.Errorf("sandbox: create session dir: %w", err)
	}

	sb := &sandbox{
		sessionID:     sessionID,
		workspaceDir:  filepath.Join(sbDir, "workspace"),
		uploadsDir:    filepath.Join(sbDir, "uploads"),
		outputsDir:    filepath.Join(sbDir, "outputs"),
	}

	for _, dir := range []string{sb.workspaceDir, sb.uploadsDir, sb.outputsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("sandbox: create subdir: %w", err)
		}
	}

	p.sandboxes[sessionID] = sb
	return sb, nil
}

func (p *LocalSandboxProvider) Get(ctx context.Context, sessionID string) (Sandbox, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	sb, ok := p.sandboxes[sessionID]
	if !ok {
		return nil, fmt.Errorf("sandbox: session %q not found", sessionID)
	}
	return sb, nil
}

func (p *LocalSandboxProvider) Release(ctx context.Context, sessionID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.sandboxes, sessionID)
	return nil
}

func (p *LocalSandboxProvider) Shutdown(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sandboxes = make(map[string]*sandbox)
	return nil
}
