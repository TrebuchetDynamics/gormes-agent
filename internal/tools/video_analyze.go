package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/video"

type VideoAnalyzeStatus = video.AnalyzeStatus

const (
	VideoAnalyzeStatusOK                 VideoAnalyzeStatus = video.AnalyzeStatusOK
	VideoAnalyzeStatusUnsupportedVideo   VideoAnalyzeStatus = video.AnalyzeStatusUnsupportedVideo
	VideoAnalyzeStatusWorkspaceViolation VideoAnalyzeStatus = video.AnalyzeStatusWorkspaceViolation
	VideoAnalyzeStatusInvalidScheme      VideoAnalyzeStatus = video.AnalyzeStatusInvalidScheme
	VideoAnalyzeStatusInvalidArgs        VideoAnalyzeStatus = video.AnalyzeStatusInvalidArgs
	VideoAnalyzeStatusProviderError      VideoAnalyzeStatus = video.AnalyzeStatusProviderError
)

type VideoAnalyzeResult = video.AnalyzeResult
type VideoAnalyzeMeta = video.AnalyzeMeta
type VideoAnalyzeProvider = video.AnalyzeProvider
type VideoAnalyzeConfig = video.AnalyzeConfig
type VideoAnalyzeTool = video.AnalyzeTool

func NewVideoAnalyzeTool(cfg VideoAnalyzeConfig) *VideoAnalyzeTool {
	return video.NewAnalyzeTool(cfg)
}
