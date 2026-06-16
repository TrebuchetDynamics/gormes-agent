package gormescli

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"
)

type cobraCommandExecutionOptions struct {
	StripLeadingArg string
	SilenceUsage    bool
	SilenceErrors   bool
	Input           io.Reader
}

func executeCobraCommandForTest(cmd *cobra.Command, opts cobraCommandExecutionOptions, args ...string) (string, string, error) {
	if opts.StripLeadingArg != "" && len(args) > 0 && args[0] == opts.StripLeadingArg {
		args = args[1:]
	}
	stdout, stderr := attachCobraCommandBuffersForTest(cmd)
	if opts.Input != nil {
		cmd.SetIn(opts.Input)
	}
	if opts.SilenceUsage {
		cmd.SilenceUsage = true
	}
	if opts.SilenceErrors {
		cmd.SilenceErrors = true
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func attachCobraCommandBuffersForTest(cmd *cobra.Command) (*bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	return &stdout, &stderr
}
