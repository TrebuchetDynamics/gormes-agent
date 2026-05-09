package wasi

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const RuntimeUnavailableMarker = "wasi_runtime_unavailable"

type RuntimeUnavailableError struct {
	Err error
}

func (e *RuntimeUnavailableError) Error() string {
	return fmt.Sprintf("%s: %v", RuntimeUnavailableMarker, e.Err)
}

func (e *RuntimeUnavailableError) Unwrap() error {
	return e.Err
}

func IsRuntimeUnavailable(err error) bool {
	var degraded *RuntimeUnavailableError
	return errors.As(err, &degraded)
}

func Run(ctx context.Context, wasm []byte, name string, args ...string) (string, error) {
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return "", &RuntimeUnavailableError{Err: err}
	}

	var stdout bytes.Buffer
	moduleArgs := append([]string{name}, args...)
	config := wazero.NewModuleConfig().
		WithName(name).
		WithArgs(moduleArgs...).
		WithStdout(&stdout)

	mod, err := runtime.InstantiateWithConfig(ctx, wasm, config)
	if err != nil {
		return "", &RuntimeUnavailableError{Err: err}
	}
	if err := mod.Close(ctx); err != nil {
		return "", &RuntimeUnavailableError{Err: err}
	}
	return stdout.String(), nil
}
