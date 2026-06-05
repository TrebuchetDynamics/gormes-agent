package review

import "context"

const (
	SkillWriteOriginForeground       = "foreground"
	SkillWriteOriginBackgroundReview = "background_review"
)

type skillWriteOriginContextKey struct{}

func WithSkillWriteOrigin(ctx context.Context, origin string) context.Context {
	if origin == "" {
		origin = SkillWriteOriginForeground
	}
	return context.WithValue(ctx, skillWriteOriginContextKey{}, origin)
}

func SkillWriteOrigin(ctx context.Context) string {
	value, _ := ctx.Value(skillWriteOriginContextKey{}).(string)
	if value == "" {
		return SkillWriteOriginForeground
	}
	return value
}

func IsBackgroundReviewSkillWrite(ctx context.Context) bool {
	return SkillWriteOrigin(ctx) == SkillWriteOriginBackgroundReview
}
