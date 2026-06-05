package gormescli

import "github.com/spf13/cobra"

func newSessionCommandForTest() *cobra.Command {
	return NewSessionCommand(SessionCommandOptions{
		Build: func() SessionBuildProvenance {
			return SessionBuildProvenance{Version: Version, GitCommit: "test-git"}
		},
	})
}

func newSessionRootCommandForTest() *cobra.Command {
	return newRootCommandWithFactoryForTest("session", newSessionCommandForTest)
}
