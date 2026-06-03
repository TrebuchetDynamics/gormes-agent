package gormescli

import (
	"github.com/spf13/cobra"

	appadmin "github.com/TrebuchetDynamics/gormes-agent/internal/app/admin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tuiadmin "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

func AdminCommandEntries(root *cobra.Command) []tuiadmin.CommandEntry {
	return appadmin.CommandEntries(root)
}

func AdminCommandRunnable(path string) bool {
	return appadmin.CommandRunnable(path)
}

func AdminCommandRunLabel(path string) string {
	return appadmin.CommandRunLabel(path)
}

func AdminCommandRunArgs(path string) ([]string, string, error) {
	return appadmin.CommandRunArgs(path, func() (config.Config, error) { return config.Load(nil) })
}
