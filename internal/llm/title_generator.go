package llm

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"
)

type TitleStatus = prompts.TitleStatus

const (
	TitleStatusGenerated        TitleStatus = prompts.TitleStatusGenerated
	TitleStatusAutoTitleSkipped TitleStatus = prompts.TitleStatusAutoTitleSkipped
	TitleStatusBlankResult      TitleStatus = prompts.TitleStatusBlankResult
	TitleStatusProviderFailed   TitleStatus = prompts.TitleStatusProviderFailed
	TitleStatusCallbackFailed   TitleStatus = prompts.TitleStatusCallbackFailed
)

type TitleMessage = prompts.TitleMessage
type TitleRequest = prompts.TitleRequest
type TitleModelMessage = prompts.TitleModelMessage
type TitleModelRequest = prompts.TitleModelRequest
type TitleModelFunc = prompts.TitleModelFunc
type TitleFailureCallback = prompts.TitleFailureCallback
type TitleResult = prompts.TitleResult
type TitleEvidence = prompts.TitleEvidence
type TitleProviderError = prompts.TitleProviderError

func GenerateTitle(ctx context.Context, req TitleRequest, model TitleModelFunc) TitleResult {
	return prompts.GenerateTitle(ctx, req, model)
}
