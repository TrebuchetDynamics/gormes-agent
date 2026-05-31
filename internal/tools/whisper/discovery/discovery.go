package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wasienv"
	embind "github.com/jerbob92/wazero-emscripten-embind"
	"github.com/tetratelabs/wazero"
)

const (
	ArtifactSource    = "github.com/agnivade/whisper-wasi"
	ArtifactCommit    = "48c8dc14efd1f74c4b3b8fcc1c045a977c2bf7d7"
	ArtifactSHA256    = "e575a73bff506574513709c26ced98b65a90b6960810078fc4d928882bc1bd2e"
	ArtifactSizeBytes = 3387207
)

type Import struct {
	Module string
	Name   string
}

type Discovery struct {
	SHA256            string
	SizeBytes         int
	ImportedFunctions []Import
	ExportedFunctions []string
	ExportedMemories  []string
	Entrypoints       []string
	Probe             string
}

func Inspect(ctx context.Context, wasm []byte) (Discovery, error) {
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		return Discovery{}, fmt.Errorf("compile whisper wasm: %w", err)
	}
	defer compiled.Close(ctx)

	sum := sha256.Sum256(wasm)
	discovery := Discovery{
		SHA256:    hex.EncodeToString(sum[:]),
		SizeBytes: len(wasm),
	}

	for _, fn := range compiled.ImportedFunctions() {
		module, name, ok := fn.Import()
		if !ok {
			continue
		}
		discovery.ImportedFunctions = append(discovery.ImportedFunctions, Import{Module: module, Name: name})
	}
	sort.Slice(discovery.ImportedFunctions, func(i, j int) bool {
		left := discovery.ImportedFunctions[i]
		right := discovery.ImportedFunctions[j]
		if left.Module == right.Module {
			return left.Name < right.Name
		}
		return left.Module < right.Module
	})

	for name := range compiled.ExportedFunctions() {
		discovery.ExportedFunctions = append(discovery.ExportedFunctions, name)
	}
	sort.Strings(discovery.ExportedFunctions)

	for name := range compiled.ExportedMemories() {
		discovery.ExportedMemories = append(discovery.ExportedMemories, name)
	}
	sort.Strings(discovery.ExportedMemories)

	for _, name := range []string{"_initialize", "_start", "main"} {
		if contains(discovery.ExportedFunctions, name) {
			discovery.Entrypoints = append(discovery.Entrypoints, name)
		}
	}

	public := exportedWithAnyPrefix(discovery.ExportedFunctions, "init", "full", "free", "_initialize")
	discovery.Probe = "embind exports: " + strings.Join(public, ", ")
	return discovery, nil
}

func InstantiateForDiscovery(ctx context.Context, wasm []byte) (Discovery, error) {
	discovery, err := Inspect(ctx, wasm)
	if err != nil {
		return Discovery{}, err
	}

	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	if err := wasienv.InstantiateWASI(ctx, runtime); err != nil {
		return Discovery{}, err
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		return Discovery{}, fmt.Errorf("compile whisper wasm: %w", err)
	}
	defer compiled.Close(ctx)

	builder := runtime.NewHostModuleBuilder("env")
	engine := embind.CreateEngine(embind.NewConfig())
	if err := wasienv.ExportEmscriptenEmbind(builder, compiled, engine); err != nil {
		return Discovery{}, err
	}
	if _, err := builder.Instantiate(ctx); err != nil {
		return Discovery{}, fmt.Errorf("instantiate emscripten embind imports: %w", err)
	}
	config := wazero.NewModuleConfig().
		WithName("whisper-wasi-discovery").
		WithStartFunctions("_initialize")
	ctx = engine.Attach(ctx)
	module, err := runtime.InstantiateModule(ctx, compiled, config)
	if err != nil {
		return Discovery{}, fmt.Errorf("instantiate whisper wasm: %w", err)
	}
	if err := module.Close(ctx); err != nil {
		return Discovery{}, fmt.Errorf("close whisper wasm: %w", err)
	}
	return discovery, nil
}

func contains(values []string, want string) bool {
	i := sort.SearchStrings(values, want)
	return i < len(values) && values[i] == want
}

func exportedWithAnyPrefix(values []string, prefixes ...string) []string {
	var matches []string
	for _, value := range values {
		for _, prefix := range prefixes {
			if strings.HasPrefix(value, prefix) {
				matches = append(matches, value)
				break
			}
		}
	}
	return matches
}
