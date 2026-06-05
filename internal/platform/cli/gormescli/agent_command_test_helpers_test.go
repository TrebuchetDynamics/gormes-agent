package gormescli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func newAgentCommandForTest() *cobra.Command {
	return NewAgentCommand(AgentOptions{
		DefaultResetTarget: config.GormesHome(),
		BuildProvenance: func() AgentBuildProvenance {
			return AgentBuildProvenance{Version: Version, GitCommit: "test-git"}
		},
		OpenRegistry: openDynamicAgentRegistryForTest,
	})
}

func newAgentRootCommandForTest() *cobra.Command {
	return newRootCommandWithFactoryForTest("agent", newAgentCommandForTest)
}

func openDynamicAgentRegistryForTest() (AgentRegistry, func(), error) {
	path := config.MemoryDBPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: create memory dir: %w", err)
	}
	db, err := sql.Open("sqlite3", path)
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
