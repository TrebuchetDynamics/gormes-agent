package gormescli

import "github.com/spf13/cobra"

func newRootCommandWithFactoryForTest(commandName string, factory func() *cobra.Command) *cobra.Command {
	return newRootCommandWithFactoriesForTest(map[string]func() *cobra.Command{commandName: factory})
}

func newRootCommandWithFactoriesForTest(overrides map[string]func() *cobra.Command) *cobra.Command {
	factories := CommandFactories{}
	for _, name := range RootCommandOrder {
		name := name
		factories[name] = func() *cobra.Command {
			return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
		}
	}
	for name, factory := range overrides {
		factories[name] = factory
	}
	return NewRootCommand(RootOptions{Version: Version}, factories)
}

func testBuildProvenance() BuildProvenance {
	return BuildProvenance{Version: Version, GitCommit: "test-git"}
}
