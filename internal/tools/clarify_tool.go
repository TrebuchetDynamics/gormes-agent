package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/clarify"

var (
	ErrClarifyInvalidArgs  = clarify.ErrClarifyInvalidArgs
	ErrClarifyUnavailable  = clarify.ErrClarifyUnavailable
	ErrClarifyRouteMissing = clarify.ErrClarifyRouteMissing
	ErrClarifyTimeout      = clarify.ErrClarifyTimeout
)

type ClarifyCallback = clarify.ClarifyCallback
type ClarifyCallbackFunc = clarify.ClarifyCallbackFunc
type ClarifyRequest = clarify.ClarifyRequest
type ClarifyResponse = clarify.ClarifyResponse
type ClarifyTool = clarify.ClarifyTool

func NewClarifyTool(callback ClarifyCallback) *ClarifyTool {
	return clarify.NewClarifyTool(callback)
}
