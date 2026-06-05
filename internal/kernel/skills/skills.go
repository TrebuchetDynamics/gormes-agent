package skills

import "context"

type Provider interface {
	BuildSkillBlock(ctx context.Context, userMessage string) (string, []string, error)
}

type UsageRecorder interface {
	RecordSkillUsage(ctx context.Context, skillNames []string) error
}
