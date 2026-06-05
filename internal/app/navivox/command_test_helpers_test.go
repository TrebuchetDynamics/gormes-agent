package navivox

import (
	"bytes"

	"github.com/spf13/cobra"
)

func executeNavivoxCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := executeNavivoxCommand(cmd, args...)
	return stdout.String(), stderr.String(), err
}

func executeNavivoxCommand(cmd *cobra.Command, args ...string) error {
	if len(args) > 0 {
		cmd.SetArgs(args)
	}
	return cmd.Execute()
}
