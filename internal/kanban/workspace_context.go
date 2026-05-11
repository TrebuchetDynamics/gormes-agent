package kanban

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceContext holds workspace metadata loaded for a kanban worker.
type WorkspaceContext struct {
	// WorkspaceDir is the resolved working directory for the worker.
	WorkspaceDir string `json:"workspace_dir"`
	// AGENTSMD is the content of AGENTS.md loaded from the workspace (if present).
	AGENTSMD string `json:"agents_md,omitempty"`
}

// LoadWorkspaceContext reads workspace context files (AGENTS.md) from the
// given workspace directory.  It returns an empty context when the directory
// does not exist or AGENTS.md is absent — the worker can start without it.
func LoadWorkspaceContext(workspaceDir string) (WorkspaceContext, error) {
	wsDir := strings.TrimSpace(workspaceDir)
	if wsDir == "" {
		return WorkspaceContext{}, fmt.Errorf("load workspace context: workspace directory is empty")
	}
	ctx := WorkspaceContext{WorkspaceDir: wsDir}

	info, err := os.Stat(wsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ctx, nil
		}
		return ctx, fmt.Errorf("load workspace context: stat %q: %w", wsDir, err)
	}
	if !info.IsDir() {
		return WorkspaceContext{}, fmt.Errorf("load workspace context: %q is not a directory", wsDir)
	}

	data, err := os.ReadFile(filepath.Join(wsDir, "AGENTS.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return ctx, nil
		}
		return ctx, fmt.Errorf("load workspace context: read AGENTS.md: %w", err)
	}
	ctx.AGENTSMD = strings.TrimSpace(string(data))
	return ctx, nil
}
