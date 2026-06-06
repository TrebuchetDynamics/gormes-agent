package discovery

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/testfixture"
)

func TestWhisperWASMArtifactProvenance(t *testing.T) {
	wasm := readWhisperWASM(t)
	discovery, err := Inspect(context.Background(), wasm)
	if err != nil {
		t.Fatalf("inspect whisper wasm: %v", err)
	}
	if discovery.SHA256 != ArtifactSHA256 {
		t.Fatalf("sha256 = %s, want %s", discovery.SHA256, ArtifactSHA256)
	}
	if discovery.SizeBytes != ArtifactSizeBytes {
		t.Fatalf("size = %d, want %d", discovery.SizeBytes, ArtifactSizeBytes)
	}
}

func TestWhisperWASMLoadsAndExposesExports(t *testing.T) {
	wasm := readWhisperWASM(t)
	discovery, err := Inspect(context.Background(), wasm)
	if err != nil {
		t.Fatalf("inspect whisper wasm: %v", err)
	}

	for _, want := range []string{"_initialize", "free"} {
		if !hasExport(discovery.ExportedFunctions, want) {
			t.Fatalf("missing export %q\nexports=%v", want, discovery.ExportedFunctions)
		}
	}
	if len(discovery.ImportedFunctions) == 0 {
		t.Fatal("expected Emscripten/WASI imports, got none")
	}
	if len(discovery.ExportedMemories) == 0 {
		t.Fatal("expected exported memory, got none")
	}
}

func TestWhisperWASMInstantiatesForDiscovery(t *testing.T) {
	wasm := readWhisperWASM(t)
	discovery, err := InstantiateForDiscovery(context.Background(), wasm)
	if err != nil {
		t.Fatalf("instantiate whisper wasm: %v", err)
	}
	if len(discovery.Entrypoints) == 0 {
		t.Fatalf("missing callable entrypoint\nexports=%v", discovery.ExportedFunctions)
	}
	if discovery.Probe == "" {
		t.Fatal("expected non-empty probe summary")
	}
}

func hasExport(exports []string, want string) bool {
	for _, export := range exports {
		if export == want {
			return true
		}
	}
	return false
}

func readWhisperWASM(t *testing.T) []byte {
	return testfixture.WhisperWASM(t)
}
