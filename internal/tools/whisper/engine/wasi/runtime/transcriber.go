package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine/wasi/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine/wasi/wav"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/pathredact"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wasienv"
	embind "github.com/jerbob92/wazero-emscripten-embind"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Transcriber struct {
	mu sync.Mutex

	wasm []byte

	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	module   api.Module
	engine   embind.Engine

	modelIndex uint32
	modelPath  string
	stdout     bytes.Buffer
	stderr     bytes.Buffer
	closed     bool
}

func NewTranscriber(ctx context.Context, modelPath string, wasm []byte) (*Transcriber, error) {
	info, err := os.Stat(modelPath)
	if err != nil {
		return nil, &contract.TranscriberError{Code: contract.TranscriberModelUnavailable, Path: filepath.Base(modelPath), Err: err}
	}
	if !info.Mode().IsRegular() {
		return nil, &contract.TranscriberError{Code: contract.TranscriberModelUnavailable, Path: filepath.Base(modelPath), Err: fmt.Errorf("model path is not a regular file")}
	}

	t := &Transcriber{modelPath: modelPath, wasm: wasm}
	if err := t.open(ctx); err != nil {
		_ = t.Close(context.Background())
		return nil, err
	}
	return t, nil
}

func (t *Transcriber) open(ctx context.Context) error {
	t.runtime = wazero.NewRuntime(ctx)
	if err := wasienv.InstantiateWASI(ctx, t.runtime); err != nil {
		return &contract.TranscriberError{Code: contract.TranscriberWASIInference, Err: err}
	}
	compiled, err := t.runtime.CompileModule(ctx, t.wasm)
	if err != nil {
		return &contract.TranscriberError{Code: contract.TranscriberWASIInference, Err: fmt.Errorf("compile whisper wasm: %w", err)}
	}
	t.compiled = compiled

	builder := t.runtime.NewHostModuleBuilder("env")
	t.engine = embind.CreateEngine(embind.NewConfig())
	if err := wasienv.ExportEmscriptenEmbind(builder, compiled, t.engine); err != nil {
		return &contract.TranscriberError{Code: contract.TranscriberWASIInference, Err: err}
	}
	builder.NewFunctionBuilder().
		WithName("_emval_get_module_property").
		WithGoModuleFunction(api.GoModuleFunc(t.emvalGetModuleProperty), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("_emval_get_module_property")
	if _, err := builder.Instantiate(ctx); err != nil {
		return &contract.TranscriberError{Code: contract.TranscriberWASIInference, Err: fmt.Errorf("instantiate env imports: %w", err)}
	}

	moduleConfig := wazero.NewModuleConfig().
		WithStartFunctions("_initialize").
		WithName("").
		WithStdout(&t.stdout).
		WithStderr(&t.stderr).
		WithFSConfig(wazero.NewFSConfig().WithDirMount(filepath.Dir(t.modelPath), "/models"))

	ctx = t.engine.Attach(ctx)
	module, err := t.runtime.InstantiateModule(ctx, compiled, moduleConfig)
	if err != nil {
		return &contract.TranscriberError{Code: contract.TranscriberWASIInference, Err: redactTranscriberError(err, t.modelPath)}
	}
	t.module = module

	modelIndex, err := callWhisperInit(t.engine, ctx, "/models/"+filepath.Base(t.modelPath))
	if err != nil {
		return &contract.TranscriberError{Code: contract.TranscriberModelUnavailable, Path: filepath.Base(t.modelPath), Err: redactTranscriberError(err, t.modelPath)}
	}
	if modelIndex == 0 {
		return &contract.TranscriberError{Code: contract.TranscriberModelUnavailable, Path: filepath.Base(t.modelPath), Err: fmt.Errorf("whisper init returned zero model handle")}
	}
	t.modelIndex = modelIndex
	t.stdout.Reset()
	t.stderr.Reset()
	return nil
}

func (t *Transcriber) TranscribeWAV(ctx context.Context, path string) (string, error) {
	samples, err := DecodePCM16Mono16kWAV(path)
	if err != nil {
		return "", err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return "", &contract.TranscriberError{Code: contract.TranscriberClosed, Err: fmt.Errorf("transcriber is closed")}
	}
	t.stdout.Reset()
	t.stderr.Reset()
	ctx = t.engine.Attach(ctx)
	ret, err := callWhisperFullDefault(t.engine, ctx, t.modelIndex, newFloat32Array(samples), "en", 1, false)
	if err != nil {
		return "", &contract.TranscriberError{Code: contract.TranscriberWASIInference, Path: filepath.Base(path), Err: redactTranscriberError(err, path, t.modelPath)}
	}
	output := strings.TrimSpace(t.stdout.String())
	if stderr := strings.TrimSpace(t.stderr.String()); stderr != "" {
		if output == "" {
			output = stderr
		} else {
			output += "\n" + stderr
		}
	}
	if ret != 0 {
		return "", &contract.TranscriberError{Code: contract.TranscriberWASIInference, Path: filepath.Base(path), Err: fmt.Errorf("whisper full_default returned %d: %s", ret, pathredact.Text(output, path, t.modelPath))}
	}
	if strings.TrimSpace(output) == "" {
		return "", &contract.TranscriberError{Code: contract.TranscriberWASIInference, Path: filepath.Base(path), Err: fmt.Errorf("empty whisper transcript")}
	}
	return output, nil
}

func (t *Transcriber) Close(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	var errs []error
	if t.engine != nil && t.modelIndex != 0 {
		if err := callWhisperFree(t.engine, t.engine.Attach(ctx), t.modelIndex); err != nil {
			errs = append(errs, err)
		}
		t.modelIndex = 0
	}
	if t.module != nil {
		if err := t.module.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if t.compiled != nil {
		if err := t.compiled.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if t.runtime != nil {
		if err := t.runtime.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return &contract.TranscriberError{Code: contract.TranscriberWASIInference, Err: errorsJoin(errs)}
	}
	return nil
}

func DecodePCM16Mono16kWAV(path string) ([]float32, error) {
	return wav.DecodePCM16Mono16k(path)
}
