package channels

import (
	"bytes"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const Version = "test-version"

func newRootCommand() *cobra.Command {
	factories := gormescli.CommandFactories{}
	for _, name := range gormescli.RootCommandOrder {
		name := name
		factories[name] = func() *cobra.Command {
			return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
		}
	}
	factories["channels"] = func() *cobra.Command { return newChannelsCommandForTest() }
	return gormescli.NewRootCommand(gormescli.RootOptions{Version: Version}, factories)
}

func newChannelsCommandForTest() *cobra.Command {
	return NewCommandWithSeams(Seams{
		LoadConfig:        func() (config.Config, error) { return config.Load(nil) },
		ConfiguredDetails: gormescli.ConfiguredChannelCapabilityDetails,
	}, Options{BuildProvenance: func() gormescli.BuildProvenance {
		return gormescli.BuildProvenance{Version: Version, GitCommit: "test-git"}
	}})
}

func executeOneshotFlagCommand(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
