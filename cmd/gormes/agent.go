package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	agentcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/agent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func newAgentCommand() *cobra.Command {
	return agentcmd.NewCommand(agentCommandOptions())
}

func agentCommandOptions() agentcmd.Options {
	return agentcmd.Options{
		DefaultResetTarget: config.GormesHome(),
		BuildProvenance: func() agentcmd.BuildProvenance {
			build := newBuildProvenance()
			return agentcmd.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
		OpenRegistry: openDynamicAgentRegistry,
	}
}

// openDynamicAgentRegistry opens the Goncho SQLite database used by the
// dynamic agent registry. The caller invokes cleanup() to close the
// underlying *sql.DB. The DB lives under $GORMES_HOME/memory.db, the same
// location the gateway uses for Goncho, and the open routes through
// sqlOpenGoncho so busy_timeout and WAL mode match the gateway's path.
func openDynamicAgentRegistry() (agentcmd.Registry, func(), error) {
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
