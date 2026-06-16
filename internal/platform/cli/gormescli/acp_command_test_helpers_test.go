package gormescli

import "github.com/spf13/cobra"

func newACPCommandForTest() *cobra.Command {
	cmd := NewACPCommand(ACPCommandOptions{
		BuildProvenance: testBuildProvenance,
		ExitError:       NewExitCodeError,
	})
	silenceUsageForTest(cmd)
	return cmd
}

func silenceUsageForTest(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	for _, child := range cmd.Commands() {
		silenceUsageForTest(child)
	}
}

func executeACPCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	return executeCobraCommandForTest(cmd, cobraCommandExecutionOptions{StripLeadingArg: "acp"}, args...)
}
