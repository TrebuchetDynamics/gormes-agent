package prompts

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/titlegen"
)

type TitleStatus = titlegen.TitleStatus

const (
	TitleStatusGenerated        TitleStatus = titlegen.TitleStatusGenerated
	TitleStatusAutoTitleSkipped TitleStatus = titlegen.TitleStatusAutoTitleSkipped
	TitleStatusBlankResult      TitleStatus = titlegen.TitleStatusBlankResult
	TitleStatusProviderFailed   TitleStatus = titlegen.TitleStatusProviderFailed
	TitleStatusCallbackFailed   TitleStatus = titlegen.TitleStatusCallbackFailed
)

type TitleMessage = titlegen.TitleMessage
type TitleRequest = titlegen.TitleRequest
type TitleModelMessage = titlegen.TitleModelMessage
type TitleModelRequest = titlegen.TitleModelRequest
type TitleModelFunc = titlegen.TitleModelFunc
type TitleFailureCallback = titlegen.TitleFailureCallback
type TitleResult = titlegen.TitleResult
type TitleEvidence = titlegen.TitleEvidence
type TitleProviderError = titlegen.TitleProviderError

func GenerateTitle(ctx context.Context, req TitleRequest, model TitleModelFunc) TitleResult {
	return titlegen.GenerateTitle(ctx, req, model)
}
