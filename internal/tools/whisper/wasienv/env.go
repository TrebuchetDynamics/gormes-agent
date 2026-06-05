package wasienv

import (
	"context"
	"fmt"

	embind "github.com/jerbob92/wazero-emscripten-embind"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func InstantiateWASI(ctx context.Context, runtime wazero.Runtime) error {
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return fmt.Errorf("instantiate wasi imports: %w", err)
	}
	return nil
}

func ExportEmscriptenEmbind(builder wazero.HostModuleBuilder, compiled wazero.CompiledModule, engine embind.Engine) error {
	emscriptenExporter, err := emscripten.NewFunctionExporterForModule(compiled)
	if err != nil {
		return fmt.Errorf("prepare emscripten imports: %w", err)
	}
	emscriptenExporter.ExportFunctions(builder)

	embindExporter := engine.NewFunctionExporterForModule(compiled)
	if err := embindExporter.ExportFunctions(builder); err != nil {
		return fmt.Errorf("prepare embind imports: %w", err)
	}
	return nil
}
