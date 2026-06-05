package kernel

import kernelextensions "github.com/TrebuchetDynamics/gormes-agent/internal/kernel/extensions"

type ExtensionHook = kernelextensions.ExtensionHook

const (
	ExtensionHookAgentInit            = kernelextensions.ExtensionHookAgentInit
	ExtensionHookMonologueStart       = kernelextensions.ExtensionHookMonologueStart
	ExtensionHookMonologueEnd         = kernelextensions.ExtensionHookMonologueEnd
	ExtensionHookMessageLoopStart     = kernelextensions.ExtensionHookMessageLoopStart
	ExtensionHookMessageLoopEnd       = kernelextensions.ExtensionHookMessageLoopEnd
	ExtensionHookBeforeMainLLMCall    = kernelextensions.ExtensionHookBeforeMainLLMCall
	ExtensionHookPromptBefore         = kernelextensions.ExtensionHookPromptBefore
	ExtensionHookPromptAfter          = kernelextensions.ExtensionHookPromptAfter
	ExtensionHookResponseStreamChunk  = kernelextensions.ExtensionHookResponseStreamChunk
	ExtensionHookReasoningStreamChunk = kernelextensions.ExtensionHookReasoningStreamChunk
	ExtensionHookToolBefore           = kernelextensions.ExtensionHookToolBefore
	ExtensionHookToolAfter            = kernelextensions.ExtensionHookToolAfter
	ExtensionHookContextDeleted       = kernelextensions.ExtensionHookContextDeleted
)

func AllExtensionHooks() []ExtensionHook { return kernelextensions.AllExtensionHooks() }

type ExtensionStatus = kernelextensions.ExtensionStatus

const (
	ExtensionStatusCompleted = kernelextensions.ExtensionStatusCompleted
	ExtensionStatusError     = kernelextensions.ExtensionStatusError
	ExtensionStatusTimeout   = kernelextensions.ExtensionStatusTimeout
	ExtensionStatusPanic     = kernelextensions.ExtensionStatusPanic
	ExtensionStatusSkipped   = kernelextensions.ExtensionStatusSkipped
)

type ExtensionData = kernelextensions.ExtensionData
type ExtensionUIStatus = kernelextensions.ExtensionUIStatus

const (
	ExtensionUIApplied     = kernelextensions.ExtensionUIApplied
	ExtensionUICleared     = kernelextensions.ExtensionUICleared
	ExtensionUIUnavailable = kernelextensions.ExtensionUIUnavailable
	ExtensionUINoop        = kernelextensions.ExtensionUINoop
)

type ExtensionUIResult = kernelextensions.ExtensionUIResult
type ExtensionUIWidgetPlacement = kernelextensions.ExtensionUIWidgetPlacement

const (
	ExtensionUIWidgetAboveEditor = kernelextensions.ExtensionUIWidgetAboveEditor
	ExtensionUIWidgetBelowEditor = kernelextensions.ExtensionUIWidgetBelowEditor
)

type ExtensionUIWidgetOptions = kernelextensions.ExtensionUIWidgetOptions
type ExtensionUIWorkingIndicator = kernelextensions.ExtensionUIWorkingIndicator
type ExtensionUI = kernelextensions.ExtensionUI

type ExtensionHandler = kernelextensions.ExtensionHandler
type ExtensionRegistration = kernelextensions.ExtensionRegistration
type ExtensionChainOptions = kernelextensions.ExtensionChainOptions
type ExtensionChain = kernelextensions.ExtensionChain
type ExtensionRunReport = kernelextensions.ExtensionRunReport
type ExtensionResult = kernelextensions.ExtensionResult

func NewNoopExtensionUI(reason string) ExtensionUI {
	return kernelextensions.NewNoopExtensionUI(reason)
}

func NormalizeExtensionUIWidgetPlacement(placement ExtensionUIWidgetPlacement) ExtensionUIWidgetPlacement {
	return kernelextensions.NormalizeExtensionUIWidgetPlacement(placement)
}

func NewExtensionChain(opts ExtensionChainOptions) *ExtensionChain {
	return kernelextensions.NewExtensionChain(opts)
}
