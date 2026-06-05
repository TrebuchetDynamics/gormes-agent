package llm

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/backgroundreview"
)

const BackgroundReviewSkillWriteOrigin = backgroundreview.BackgroundReviewSkillWriteOrigin

type BackgroundReviewRuntime = backgroundreview.BackgroundReviewRuntime
type BackgroundReviewMessage = backgroundreview.BackgroundReviewMessage
type BackgroundReviewFork = backgroundreview.BackgroundReviewFork
type BackgroundReviewRequest = backgroundreview.BackgroundReviewRequest
type BackgroundReviewResult = backgroundreview.BackgroundReviewResult
type BackgroundReviewRunner = backgroundreview.BackgroundReviewRunner
type BackgroundReviewRunnerFunc = backgroundreview.BackgroundReviewRunnerFunc
type BackgroundReviewApprovalCallback = backgroundreview.BackgroundReviewApprovalCallback
type BackgroundReviewApprovalSlot = backgroundreview.BackgroundReviewApprovalSlot
type BackgroundReviewToolsetStatus = backgroundreview.BackgroundReviewToolsetStatus

const (
	BackgroundReviewToolsetAllowed     = backgroundreview.BackgroundReviewToolsetAllowed
	BackgroundReviewToolsetRestricted  = backgroundreview.BackgroundReviewToolsetRestricted
	BackgroundReviewToolsetUnavailable = backgroundreview.BackgroundReviewToolsetUnavailable
)

type BackgroundReviewToolsetEvidence = backgroundreview.BackgroundReviewToolsetEvidence
type BackgroundReviewToolsetPolicy = backgroundreview.BackgroundReviewToolsetPolicy

func DefaultBackgroundReviewToolsetPolicy() BackgroundReviewToolsetPolicy {
	return backgroundreview.DefaultBackgroundReviewToolsetPolicy()
}

func RunBackgroundReview(ctx context.Context, req BackgroundReviewRequest) (BackgroundReviewResult, error) {
	return backgroundreview.RunBackgroundReview(ctx, req)
}

func SummarizeBackgroundReviewActions(reviewMessages, priorSnapshot []BackgroundReviewMessage) []string {
	return backgroundreview.SummarizeBackgroundReviewActions(reviewMessages, priorSnapshot)
}
