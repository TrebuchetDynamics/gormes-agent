package wasi

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestWASIRuntime_RunsHelloWorld(t *testing.T) {
	wasm, err := os.ReadFile("testdata/hello.wasm")
	if err != nil {
		t.Fatalf("read hello fixture: %v", err)
	}

	stdout, err := Run(context.Background(), wasm, "hello")
	if err != nil {
		t.Fatalf("run hello fixture: %v", err)
	}

	if !strings.Contains(stdout, "hello from gormes wasi") {
		t.Fatalf("stdout = %q, want hello greeting", stdout)
	}
}

func TestWASIRuntime_InvalidModuleReturnsDegradedMarker(t *testing.T) {
	_, err := Run(context.Background(), []byte("not wasm"), "bad")
	if err == nil {
		t.Fatal("Run invalid module returned nil error")
	}
	if !IsRuntimeUnavailable(err) {
		t.Fatalf("IsRuntimeUnavailable(%v) = false, want true", err)
	}
}
