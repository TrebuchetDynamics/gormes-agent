package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser"

// Browser action and evidence vocabulary stays provider-neutral: live browser
// backends, local sidecars, cloud browsers, and future channel renderers can all
// consume the same validated action/result transcript without embedding a
// Telegram- or backend-specific bypass.
const (
	BrowserActionNavigate = browser.BrowserActionNavigate
	BrowserActionClick    = browser.BrowserActionClick
	BrowserActionType     = browser.BrowserActionType
	BrowserActionSnapshot = browser.BrowserActionSnapshot
	BrowserActionExtract  = browser.BrowserActionExtract
	BrowserActionWait     = browser.BrowserActionWait
	BrowserActionBack     = browser.BrowserActionBack
	BrowserActionScroll   = browser.BrowserActionScroll

	BrowserEvidenceActionAccepted     = browser.BrowserEvidenceActionAccepted
	BrowserEvidenceInvalidAction      = browser.BrowserEvidenceInvalidAction
	BrowserEvidenceMissingURL         = browser.BrowserEvidenceMissingURL
	BrowserEvidenceMissingSelector    = browser.BrowserEvidenceMissingSelector
	BrowserEvidenceResultOK           = browser.BrowserEvidenceResultOK
	BrowserEvidenceResultTruncated    = browser.BrowserEvidenceResultTruncated
	BrowserEvidenceBackendUnavailable = browser.BrowserEvidenceBackendUnavailable

	BrowserEventAction = browser.BrowserEventAction
	BrowserEventResult = browser.BrowserEventResult
	BrowserEventError  = browser.BrowserEventError
)

// BrowserAction is the native, backend-independent description of a browser
// operation. It is intentionally small and serializable; concrete providers can
// translate it to chromedp/Rod/cloud APIs after ValidateBrowserAction accepts it.
type BrowserAction = browser.BrowserAction

// BrowserActionDecision is the public validation result for a browser action.
type BrowserActionDecision = browser.BrowserActionDecision

// BrowserPageState is the provider-neutral state channels and transcripts need
// after an action completes. Screenshots and files are referenced by path or
// artifact evidence instead of embedding bytes here.
type BrowserPageState = browser.BrowserPageState

// BrowserResultInput carries raw provider output into the result envelope.
type BrowserResultInput = browser.BrowserResultInput

// BrowserResultEnvelope is the bounded, transcript-ready result of an action.
type BrowserResultEnvelope = browser.BrowserResultEnvelope

// BrowserTranscriptEvent is one append-only transcript entry.
type BrowserTranscriptEvent = browser.BrowserTranscriptEvent

// BrowserTranscript stores browser events for one browser task/session.
type BrowserTranscript = browser.BrowserTranscript

// ValidateBrowserAction performs pure action validation and private-URL routing
// checks without opening a browser, resolving DNS, or contacting a provider.
func ValidateBrowserAction(action BrowserAction) BrowserActionDecision {
	return browser.ValidateBrowserAction(action)
}

// BuildBrowserResultEnvelope bounds provider output and returns structured
// evidence suitable for prompt context, channel delivery, and audit logs.
func BuildBrowserResultEnvelope(input BrowserResultInput) (BrowserResultEnvelope, error) {
	return browser.BuildBrowserResultEnvelope(input)
}

// BuildBrowserUnavailableResult creates a stable degraded-mode envelope when no
// browser backend is configured or available. It never starts a browser.
func BuildBrowserUnavailableResult(action BrowserAction, reason string) BrowserResultEnvelope {
	return browser.BuildBrowserUnavailableResult(action, reason)
}

// NewBrowserTranscript creates an in-memory transcript. Persistence and runtime
// lifecycle ownership stay with the caller/session store.
func NewBrowserTranscript(taskID string) *BrowserTranscript {
	return browser.NewBrowserTranscript(taskID)
}
