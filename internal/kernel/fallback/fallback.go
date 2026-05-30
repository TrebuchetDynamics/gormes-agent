package fallback

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type ClientFactory func(context.Context, llm.ModelRoute) (llm.Client, error)
