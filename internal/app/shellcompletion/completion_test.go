package shellcompletion

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandRejectsUnsupportedShell(t *testing.T) {
	cmd := NewCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"xonsh"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("completion xonsh error = nil\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(err.Error(), "unsupported shell \"xonsh\"") {
		t.Fatalf("completion xonsh error = %v, want unsupported shell", err)
	}
}
