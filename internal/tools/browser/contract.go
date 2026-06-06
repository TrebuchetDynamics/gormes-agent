package browser

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser/contractcore"
)

// Browser action and evidence vocabulary stays provider-neutral: live browser
// backends, local sidecars, cloud browsers, and future channel renderers can all
// consume the same validated action/result transcript without embedding a
// Telegram- or backend-specific bypass.
const (
	BrowserActionNavigate = contractcore.BrowserActionNavigate
	BrowserActionClick    = contractcore.BrowserActionClick
	BrowserActionType     = contractcore.BrowserActionType
	BrowserActionSnapshot = contractcore.BrowserActionSnapshot
	BrowserActionExtract  = contractcore.BrowserActionExtract
	BrowserActionWait     = contractcore.BrowserActionWait
	BrowserActionBack     = contractcore.BrowserActionBack
	BrowserActionScroll   = contractcore.BrowserActionScroll

	BrowserEvidenceActionAccepted     = contractcore.BrowserEvidenceActionAccepted
	BrowserEvidenceInvalidAction      = contractcore.BrowserEvidenceInvalidAction
	BrowserEvidenceMissingURL         = contractcore.BrowserEvidenceMissingURL
	BrowserEvidenceMissingSelector    = contractcore.BrowserEvidenceMissingSelector
	BrowserEvidenceResultOK           = contractcore.BrowserEvidenceResultOK
	BrowserEvidenceResultTruncated    = contractcore.BrowserEvidenceResultTruncated
	BrowserEvidenceBackendUnavailable = contractcore.BrowserEvidenceBackendUnavailable

	BrowserEventAction = contractcore.BrowserEventAction
	BrowserEventResult = contractcore.BrowserEventResult
	BrowserEventError  = contractcore.BrowserEventError
)

type BrowserAction = contractcore.BrowserAction
type BrowserActionDecision = contractcore.BrowserActionDecision
type BrowserPageState = contractcore.BrowserPageState
type BrowserResultInput = contractcore.BrowserResultInput
type BrowserResultEnvelope = contractcore.BrowserResultEnvelope
type BrowserTranscriptEvent = contractcore.BrowserTranscriptEvent
type BrowserTranscript = contractcore.BrowserTranscript

// ValidateBrowserAction performs pure action validation and private-URL routing
// checks without opening a browser, resolving DNS, or contacting a provider.
func ValidateBrowserAction(action BrowserAction) BrowserActionDecision {
	return contractcore.ValidateBrowserAction(action)
}

// BuildBrowserResultEnvelope bounds provider output and returns structured
// evidence suitable for prompt context, channel delivery, and audit logs.
func BuildBrowserResultEnvelope(input BrowserResultInput) (BrowserResultEnvelope, error) {
	return contractcore.BuildBrowserResultEnvelope(input)
}

// BuildBrowserUnavailableResult creates a stable degraded-mode envelope when no
// browser backend is configured or available. It never starts a browser.
func BuildBrowserUnavailableResult(action BrowserAction, reason string) BrowserResultEnvelope {
	return contractcore.BuildBrowserUnavailableResult(action, reason)
}

// NewBrowserTranscript creates an in-memory transcript. Persistence and runtime
// lifecycle ownership stay with the caller/session store.
func NewBrowserTranscript(taskID string) *BrowserTranscript {
	return contractcore.NewBrowserTranscript(taskID)
}
