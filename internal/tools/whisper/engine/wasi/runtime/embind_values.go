package runtime

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
)

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
