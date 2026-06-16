package whisperapi

import (
	"context"

	embind "github.com/jerbob92/wazero-emscripten-embind"
)

func Init(engine embind.Engine, ctx context.Context, modelPath string) (uint32, error) {
	res, err := engine.CallPublicSymbol(ctx, "init", modelPath)
	if err != nil || res == nil {
		return 0, err
	}
	return res.(uint32), nil
}

func FullDefault(engine embind.Engine, ctx context.Context, modelIndex uint32, audio any, language string, threads int32, translate bool) (int32, error) {
	res, err := engine.CallPublicSymbol(ctx, "full_default", modelIndex, audio, language, threads, translate)
	if err != nil || res == nil {
		return 0, err
	}
	return res.(int32), nil
}

func Free(engine embind.Engine, ctx context.Context, modelIndex uint32) error {
	_, err := engine.CallPublicSymbol(ctx, "free", modelIndex)
	return err
}
