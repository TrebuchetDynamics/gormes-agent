package gormescli

import (
	"context"
	"io"

	approuter "github.com/TrebuchetDynamics/gormes-agent/internal/app/router"
)

type RouterOptions = approuter.Options
type RouterRequest = approuter.Request
type RouterReadModel = approuter.ReadModel

func RunRouter(ctx context.Context, out io.Writer, request RouterRequest, opts RouterOptions) error {
	return approuter.Run(ctx, out, request, opts)
}

func PrintRouterDryRun(out io.Writer, model RouterReadModel) {
	approuter.PrintDryRun(out, model)
}

func RouterOpenAIBaseURL(listen string) string {
	return approuter.OpenAIBaseURL(listen)
}
