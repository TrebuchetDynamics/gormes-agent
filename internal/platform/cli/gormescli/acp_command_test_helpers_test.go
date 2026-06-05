package gormescli

import (
	"bytes"

	"github.com/spf13/cobra"
)

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
	if len(args) > 0 && args[0] == "acp" {
		args = args[1:]
	}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
