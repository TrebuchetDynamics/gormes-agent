package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newAgentCommand() *cobra.Command {
	return gormescli.NewAgentCommand(agentCommandOptions())
}

func agentCommandOptions() gormescli.AgentOptions {
	return gormescli.AgentOptions{
		DefaultResetTarget: config.GormesHome(),
		BuildProvenance: func() gormescli.AgentBuildProvenance {
			build := newBuildProvenance()
			return gormescli.AgentBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
		OpenRegistry: openDynamicAgentRegistry,
	}
}

// openDynamicAgentRegistry opens the Goncho SQLite database used by the
// dynamic agent registry. The caller invokes cleanup() to close the
// underlying *sql.DB. The DB lives under $GORMES_HOME/memory.db, the same
// location the gateway uses for Goncho, and the open routes through
// sqlOpenGoncho so busy_timeout and WAL mode match the gateway's path.
func openDynamicAgentRegistry() (gormescli.AgentRegistry, func(), error) {
	path := config.MemoryDBPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: create memory dir: %w", err)
	}
	db, err := sqlOpenGoncho(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: open registry db: %w", err)
	}
	reg, err := goncho.NewDynamicAgentRegistry(db)
	if err != nil {
		_ = db.Close()
		return nil, func() {}, err
	}
	return reg, func() { _ = db.Close() }, nil
}
