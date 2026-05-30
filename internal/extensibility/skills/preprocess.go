package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/preprocessing"
)

type PreprocessOptions = preprocessing.PreprocessOptions

func PreprocessSkillContent(ctx context.Context, content string, opts PreprocessOptions) (string, error) {
	return preprocessing.PreprocessSkillContent(ctx, content, opts)
}
