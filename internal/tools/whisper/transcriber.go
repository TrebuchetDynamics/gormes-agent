package whisper

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	embind "github.com/jerbob92/wazero-emscripten-embind"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	TranscriberModelUnavailable = "model_unavailable"
	TranscriberWAVUnsupported   = "wav_unsupported"
	TranscriberWASIInference    = "wasi_inference_failed"
	TranscriberClosed           = "transcriber_closed"
)

//go:embed testdata/whisper.wasm
var whisperWASM []byte

type TranscriberError struct {
	Code string
	Path string
	Err  error
}

func (e *TranscriberError) Error() string {
	var parts []string
	parts = append(parts, e.Code)
	if e.Path != "" {
		parts = append(parts, "path="+e.Path)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *TranscriberError) Unwrap() error {
	return e.Err
}

type Transcriber struct {
	mu sync.Mutex

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

func NewTranscriber(ctx context.Context, modelPath string) (*Transcriber, error) {
	info, err := os.Stat(modelPath)
	if err != nil {
		return nil, &TranscriberError{Code: TranscriberModelUnavailable, Path: filepath.Base(modelPath), Err: err}
	}
	if !info.Mode().IsRegular() {
		return nil, &TranscriberError{Code: TranscriberModelUnavailable, Path: filepath.Base(modelPath), Err: fmt.Errorf("model path is not a regular file")}
	}

	t := &Transcriber{modelPath: modelPath}
	if err := t.open(ctx); err != nil {
		_ = t.Close(context.Background())
		return nil, err
	}
	return t, nil
}

func (t *Transcriber) open(ctx context.Context) error {
	t.runtime = wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, t.runtime); err != nil {
		return &TranscriberError{Code: TranscriberWASIInference, Err: fmt.Errorf("instantiate wasi imports: %w", err)}
	}
	compiled, err := t.runtime.CompileModule(ctx, whisperWASM)
	if err != nil {
		return &TranscriberError{Code: TranscriberWASIInference, Err: fmt.Errorf("compile whisper wasm: %w", err)}
	}
	t.compiled = compiled

	builder := t.runtime.NewHostModuleBuilder("env")
	emscriptenExporter, err := emscripten.NewFunctionExporterForModule(compiled)
	if err != nil {
		return &TranscriberError{Code: TranscriberWASIInference, Err: fmt.Errorf("prepare emscripten imports: %w", err)}
	}
	emscriptenExporter.ExportFunctions(builder)

	t.engine = embind.CreateEngine(embind.NewConfig())
	embindExporter := t.engine.NewFunctionExporterForModule(compiled)
	if err := embindExporter.ExportFunctions(builder); err != nil {
		return &TranscriberError{Code: TranscriberWASIInference, Err: fmt.Errorf("prepare embind imports: %w", err)}
	}
	builder.NewFunctionBuilder().
		WithName("_emval_get_module_property").
		WithGoModuleFunction(api.GoModuleFunc(t.emvalGetModuleProperty), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("_emval_get_module_property")
	if _, err := builder.Instantiate(ctx); err != nil {
		return &TranscriberError{Code: TranscriberWASIInference, Err: fmt.Errorf("instantiate env imports: %w", err)}
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
		return &TranscriberError{Code: TranscriberWASIInference, Err: redactTranscriberError(err, t.modelPath)}
	}
	t.module = module

	modelIndex, err := callWhisperInit(t.engine, ctx, "/models/"+filepath.Base(t.modelPath))
	if err != nil {
		return &TranscriberError{Code: TranscriberModelUnavailable, Path: filepath.Base(t.modelPath), Err: redactTranscriberError(err, t.modelPath)}
	}
	if modelIndex == 0 {
		return &TranscriberError{Code: TranscriberModelUnavailable, Path: filepath.Base(t.modelPath), Err: fmt.Errorf("whisper init returned zero model handle")}
	}
	t.modelIndex = modelIndex
	t.stdout.Reset()
	t.stderr.Reset()
	return nil
}

func (t *Transcriber) TranscribeWAV(ctx context.Context, path string) (string, error) {
	samples, err := decodePCM16Mono16kWAV(path)
	if err != nil {
		return "", err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return "", &TranscriberError{Code: TranscriberClosed, Err: fmt.Errorf("transcriber is closed")}
	}
	t.stdout.Reset()
	t.stderr.Reset()
	ctx = t.engine.Attach(ctx)
	ret, err := callWhisperFullDefault(t.engine, ctx, t.modelIndex, newFloat32Array(samples), "en", 1, false)
	if err != nil {
		return "", &TranscriberError{Code: TranscriberWASIInference, Path: filepath.Base(path), Err: redactTranscriberError(err, path, t.modelPath)}
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
		return "", &TranscriberError{Code: TranscriberWASIInference, Path: filepath.Base(path), Err: fmt.Errorf("whisper full_default returned %d: %s", ret, redactedOutput(output, path, t.modelPath))}
	}
	if strings.TrimSpace(output) == "" {
		return "", &TranscriberError{Code: TranscriberWASIInference, Path: filepath.Base(path), Err: fmt.Errorf("empty whisper transcript")}
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
		return &TranscriberError{Code: TranscriberWASIInference, Err: errorsJoin(errs)}
	}
	return nil
}

func (t *Transcriber) emvalGetModuleProperty(ctx context.Context, mod api.Module, stack []uint64) {
	name := readCString(mod, uint32(api.DecodeI32(stack[0])))
	switch name {
	case "HEAPU8":
		memory := mod.Memory()
		buffer, ok := memory.Read(0, memory.Size())
		if !ok {
			panic("read whisper wasm memory")
		}
		stack[0] = api.EncodeI32(t.engine.EmvalToHandle(&emscriptenHeapU8{Buffer: buffer}))
	case "Float32Array":
		stack[0] = api.EncodeI32(t.engine.EmvalToHandle(float32ArrayConstructor{}))
	default:
		panic(fmt.Errorf("unsupported whisper wasm module property %q", name))
	}
}

func readCString(mod api.Module, offset uint32) string {
	if offset == 0 {
		return ""
	}
	var out []byte
	for i := offset; ; i++ {
		b, ok := mod.Memory().ReadByte(i)
		if !ok || b == 0 {
			break
		}
		out = append(out, b)
	}
	return string(out)
}

type emscriptenHeapU8 struct {
	Buffer []uint8 `embind_property:"buffer"`
}

type float32ArrayConstructor struct{}

func (float32ArrayConstructor) New(_ []string, args ...any) (any, error) {
	array := &float32Array{Constructor: &float32Array{}}
	switch len(args) {
	case 1:
		length, err := int32FromEmvalArg(args[0])
		if err != nil {
			return nil, err
		}
		array.BackingBuffer = make([]byte, int(length)*4)
		array.Length = length
		return array, nil
	case 3:
		backing, ok := args[0].([]uint8)
		if !ok {
			return nil, fmt.Errorf("Float32Array backing buffer has type %T", args[0])
		}
		offset, err := uint32FromEmvalArg(args[1])
		if err != nil {
			return nil, err
		}
		length, err := int32FromEmvalArg(args[2])
		if err != nil {
			return nil, err
		}
		array.BackingBuffer = backing
		array.ByteOffset = offset
		array.Length = length
		return array, nil
	default:
		return nil, fmt.Errorf("Float32Array constructor got %d args", len(args))
	}
}

func int32FromEmvalArg(arg any) (int32, error) {
	switch value := arg.(type) {
	case int32:
		return value, nil
	case uint32:
		return int32(value), nil
	case int:
		return int32(value), nil
	default:
		return 0, fmt.Errorf("expected int32-compatible arg, got %T", arg)
	}
}

func uint32FromEmvalArg(arg any) (uint32, error) {
	switch value := arg.(type) {
	case int32:
		return uint32(value), nil
	case uint32:
		return value, nil
	case int:
		return uint32(value), nil
	default:
		return 0, fmt.Errorf("expected uint32-compatible arg, got %T", arg)
	}
}

func callWhisperInit(engine embind.Engine, ctx context.Context, modelPath string) (uint32, error) {
	res, err := engine.CallPublicSymbol(ctx, "init", modelPath)
	if err != nil || res == nil {
		return 0, err
	}
	return res.(uint32), nil
}

func callWhisperFullDefault(engine embind.Engine, ctx context.Context, modelIndex uint32, audio any, language string, threads int32, translate bool) (int32, error) {
	res, err := engine.CallPublicSymbol(ctx, "full_default", modelIndex, audio, language, threads, translate)
	if err != nil || res == nil {
		return 0, err
	}
	return res.(int32), nil
}

func callWhisperFree(engine embind.Engine, ctx context.Context, modelIndex uint32) error {
	_, err := engine.CallPublicSymbol(ctx, "free", modelIndex)
	return err
}

type float32Array struct {
	BackingBuffer []uint8 `embind_arg:"0"`
	ByteOffset    uint32  `embind_arg:"1"`
	Length        int32   `embind_arg:"2"`
	Constructor   *float32Array
	Buffer        []float32
}

func newFloat32Array(samples []float32) *float32Array {
	return &float32Array{
		Buffer:      samples,
		Constructor: &float32Array{},
		Length:      int32(len(samples)),
	}
}

func (fa *float32Array) Set(in *float32Array) {
	if len(in.Buffer) == 0 {
		fa.Buffer = nil
		fa.Length = 0
		return
	}
	reslice := fa.BackingBuffer[fa.ByteOffset : fa.ByteOffset+(uint32(fa.Length)*4)]
	conv := unsafe.Slice((*uint8)(unsafe.Pointer(&in.Buffer[0])), len(in.Buffer)*4)
	copy(reslice, conv)
	fa.Buffer = in.Buffer
	fa.Length = in.Length
}

func decodePCM16Mono16kWAV(path string) ([]float32, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: err}
	}
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: fmt.Errorf("not a RIFF/WAVE file")}
	}

	var (
		haveFormat    bool
		audioFormat   uint16
		numChannels   uint16
		sampleRate    uint32
		bitsPerSample uint16
		data          []byte
	)
	for offset := 12; offset+8 <= len(raw); {
		chunkID := string(raw[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(raw[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkSize < 0 || chunkEnd > len(raw) {
			return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: fmt.Errorf("truncated %s chunk", chunkID)}
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: fmt.Errorf("short fmt chunk")}
			}
			haveFormat = true
			audioFormat = binary.LittleEndian.Uint16(raw[chunkStart : chunkStart+2])
			numChannels = binary.LittleEndian.Uint16(raw[chunkStart+2 : chunkStart+4])
			sampleRate = binary.LittleEndian.Uint32(raw[chunkStart+4 : chunkStart+8])
			bitsPerSample = binary.LittleEndian.Uint16(raw[chunkStart+14 : chunkStart+16])
		case "data":
			data = raw[chunkStart:chunkEnd]
		}
		offset = chunkEnd
		if offset%2 == 1 {
			offset++
		}
	}
	if !haveFormat || len(data) == 0 {
		return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: fmt.Errorf("missing fmt or data chunk")}
	}
	if audioFormat != 1 || numChannels != 1 || sampleRate != 16000 || bitsPerSample != 16 {
		return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: fmt.Errorf("want PCM16 mono 16000Hz, got format=%d channels=%d sample_rate=%d bits=%d", audioFormat, numChannels, sampleRate, bitsPerSample)}
	}
	if len(data)%2 != 0 {
		return nil, &TranscriberError{Code: TranscriberWAVUnsupported, Path: filepath.Base(path), Err: fmt.Errorf("odd PCM byte length")}
	}
	samples := make([]float32, len(data)/2)
	for i := range samples {
		sample := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		samples[i] = float32(sample) / 32768.0
	}
	return samples, nil
}

func redactTranscriberError(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", redactedOutput(err.Error(), paths...))
}

func redactedOutput(output string, paths ...string) string {
	redacted := output
	for _, path := range paths {
		if path == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, path, filepath.Base(path))
	}
	return redacted
}

func errorsJoin(errs []error) error {
	var parts []string
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	return errors.New(strings.Join(parts, "; "))
}
