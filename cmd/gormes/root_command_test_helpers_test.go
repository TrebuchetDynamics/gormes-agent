package main

import (
	"bytes"

	"github.com/spf13/cobra"
)

func executeRootCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := executeRootCommand(cmd, args...)
	return stdout.String(), stderr.String(), err
}
