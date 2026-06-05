package subagent

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/review"
)

const (
	SkillWriteOriginForeground       = review.SkillWriteOriginForeground
	SkillWriteOriginBackgroundReview = review.SkillWriteOriginBackgroundReview
)

func WithSkillWriteOrigin(ctx context.Context, origin string) context.Context {
	return review.WithSkillWriteOrigin(ctx, origin)
}

func SkillWriteOrigin(ctx context.Context) string {
	return review.SkillWriteOrigin(ctx)
}

func IsBackgroundReviewSkillWrite(ctx context.Context) bool {
	return review.IsBackgroundReviewSkillWrite(ctx)
}
