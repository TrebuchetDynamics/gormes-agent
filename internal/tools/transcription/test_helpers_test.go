//go:build !slim

package transcription

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestAudioFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
