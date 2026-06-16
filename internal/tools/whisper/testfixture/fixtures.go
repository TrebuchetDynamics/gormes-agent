package testfixture

import (
	"os"
	"path/filepath"
	"testing"
)

func WhisperWASM(t testing.TB) []byte {
	t.Helper()
	wasm, err := os.ReadFile(filepath.Join("..", "testdata", "whisper.wasm"))
	if err != nil {
		t.Fatalf("read whisper.wasm: %v", err)
	}
	return wasm
}

func JFKWAVPath() string {
	return filepath.Join("..", "testdata", "jfk.wav")
}
