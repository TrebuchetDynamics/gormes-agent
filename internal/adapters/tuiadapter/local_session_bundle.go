package tuiadapter

import (
	"context"

	appsession "github.com/TrebuchetDynamics/gormes-agent/internal/app/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	tuilocal "github.com/TrebuchetDynamics/gormes-agent/internal/tui/local"
)

// LocalSessionBundleOptions wires local, memory-backed session adapters into
// the native Bubble Tea TUI. Remote TUI construction intentionally skips this
// bundle and receives nil session callbacks.
type LocalSessionBundleOptions struct {
	RootContext context.Context
	Metadata    *sessionpkg.BoltMap
	Resume      func(string, []llm.Message) error
	Reset       func() error
}

func NewLocalSessionBundle(opts LocalSessionBundleOptions) SessionBundle {
	rootCtx := opts.RootContext
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	return SessionBundle{
		Export:      appsession.NewTUISaveExportFunc(),
		Branch:      tuilocal.NewSessionBranchFunc(rootCtx, opts.Metadata, opts.Resume),
		Title:       tuilocal.NewSessionTitleFunc(rootCtx, opts.Metadata),
		Directory:   tuilocal.NewSessionDirectoryFunc(rootCtx),
		Resume:      tuilocal.NewSessionResumeFunc(rootCtx, opts.Resume),
		Tree:        tuilocal.NewSessionTreeFunc(rootCtx, opts.Metadata),
		TreeLabel:   tuilocal.NewSessionTreeLabelFunc(rootCtx, opts.Metadata),
		TreeRestore: tuilocal.NewSessionTreeRestoreFunc(rootCtx),
		Reset:       opts.Reset,
	}
}
