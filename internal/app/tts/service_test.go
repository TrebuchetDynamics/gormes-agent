package tts

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandExposesPiperVoiceShortcuts(t *testing.T) {
	cmd := NewCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"piper", "voices"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("tts piper voices: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	for _, want := range []string{"lessac-medium", "language=en_US", "quality=medium", "en_US-lessac-medium.onnx"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}
