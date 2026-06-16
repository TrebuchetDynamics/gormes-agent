package commandtest

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// Execute runs a fresh test command with captured stdout/stderr.
func Execute(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
