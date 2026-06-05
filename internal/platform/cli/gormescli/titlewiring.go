package gormescli

import (
	"context"

	apptitlewiring "github.com/TrebuchetDynamics/gormes-agent/internal/app/titlewiring"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func BuildTitleModelFunc(client llm.Client, model string) llm.TitleModelFunc {
	return apptitlewiring.BuildTitleModelFunc(client, model)
}

func BuildGatewayTitleSeam(ctx context.Context, smap session.Map, client llm.Client, model string) (session.SessionTitleStore, llm.TitleModelFunc) {
	return apptitlewiring.BuildGatewayTitleSeam(ctx, smap, client, model)
}
