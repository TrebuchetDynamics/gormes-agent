package tools

import (
	"fmt"
	"strings"
)

// Browser action and evidence vocabulary stays provider-neutral: live browser
// backends, local sidecars, cloud browsers, and future channel renderers can all
// consume the same validated action/result transcript without embedding a
// Telegram- or backend-specific bypass.
const (
	BrowserActionNavigate = "navigate"
	BrowserActionClick    = "click"
	BrowserActionType     = "type"
	BrowserActionSnapshot = "snapshot"
	BrowserActionExtract  = "extract"
	BrowserActionWait     = "wait"
	BrowserActionBack     = "back"
	BrowserActionScroll   = "scroll"

	BrowserEvidenceActionAccepted     = "browser_action_accepted"
	BrowserEvidenceInvalidAction      = "browser_action_invalid"
	BrowserEvidenceMissingURL         = "browser_action_missing_url"
	BrowserEvidenceMissingSelector    = "browser_action_missing_selector"
	BrowserEvidenceResultOK           = "browser_result_ok"
	BrowserEvidenceResultTruncated    = "browser_result_truncated"
	BrowserEvidenceBackendUnavailable = "browser_backend_unavailable"

	BrowserEventAction = "action"
	BrowserEventResult = "result"
	BrowserEventError  = "error"
)

// BrowserAction is the native, backend-independent description of a browser
// operation. It is intentionally small and serializable; concrete providers can
// translate it to chromedp/Rod/cloud APIs after ValidateBrowserAction accepts it.
type BrowserAction struct {
	Kind     string
	TaskID   string
	URL      string
	Selector string
	Text     string
	Options  BrowserSSRFGuardOptions
}

// BrowserActionDecision is the public validation result for a browser action.
type BrowserActionDecision struct {
	Allowed  bool
	Evidence string
	Route    BrowserRoute
}

// BrowserPageState is the provider-neutral state channels and transcripts need
// after an action completes. Screenshots and files are referenced by path or
// artifact evidence instead of embedding bytes here.
type BrowserPageState struct {
	URL            string
	Title          string
	Text           string
	Console        []string
	Errors         []string
	ScreenshotPath string
	Interactive    int
}

// BrowserResultInput carries raw provider output into the result envelope.
type BrowserResultInput struct {
	Action    BrowserAction
	State     BrowserPageState
	Output    []byte
	MediaType string
	Budget    ToolResultBudgetConfig
}

// BrowserResultEnvelope is the bounded, transcript-ready result of an action.
type BrowserResultEnvelope struct {
	Action   BrowserAction
	State    BrowserPageState
	Text     string
	Tool     ToolResultEvidence
	Evidence string
}

// BrowserTranscriptEvent is one append-only transcript entry.
type BrowserTranscriptEvent struct {
	Kind     string
	Evidence string
	Action   BrowserAction
	State    BrowserPageState
	Text     string
}

// BrowserTranscript stores browser events for one browser task/session.
type BrowserTranscript struct {
	TaskID string
	events []BrowserTranscriptEvent
}

// ValidateBrowserAction performs pure action validation and private-URL routing
// checks without opening a browser, resolving DNS, or contacting a provider.
func ValidateBrowserAction(action BrowserAction) BrowserActionDecision {
	kind := strings.ToLower(strings.TrimSpace(action.Kind))
	switch kind {
	case BrowserActionNavigate:
		if strings.TrimSpace(action.URL) == "" {
			return denyBrowserAction(BrowserEvidenceMissingURL)
		}
		guard := CheckBrowserSSRFGuard(action.TaskID, action.URL, action.Options)
		return BrowserActionDecision{Allowed: guard.Allowed, Evidence: guardEvidenceOrAccepted(guard), Route: guard.Route}
	case BrowserActionClick:
		if strings.TrimSpace(action.Selector) == "" {
			return denyBrowserAction(BrowserEvidenceMissingSelector)
		}
		return acceptBrowserAction(action.TaskID)
	case BrowserActionType:
		if strings.TrimSpace(action.Selector) == "" {
			return denyBrowserAction(BrowserEvidenceMissingSelector)
		}
		return acceptBrowserAction(action.TaskID)
	case BrowserActionSnapshot, BrowserActionExtract, BrowserActionWait, BrowserActionBack, BrowserActionScroll:
		return acceptBrowserAction(action.TaskID)
	default:
		return denyBrowserAction(BrowserEvidenceInvalidAction)
	}
}

// BuildBrowserResultEnvelope bounds provider output and returns structured
// evidence suitable for prompt context, channel delivery, and audit logs.
func BuildBrowserResultEnvelope(input BrowserResultInput) (BrowserResultEnvelope, error) {
	text, evidence, err := FormatToolResult(input.Budget, input.Output, input.MediaType)
	resultEvidence := BrowserEvidenceResultOK
	if evidence.Code == ToolResultEvidenceTruncated || evidence.Code == ToolResultEvidencePersisted || evidence.Code == ToolResultEvidencePersistenceFailed {
		resultEvidence = BrowserEvidenceResultTruncated
	}
	return BrowserResultEnvelope{
		Action:   input.Action,
		State:    input.State,
		Text:     text,
		Tool:     evidence,
		Evidence: resultEvidence,
	}, err
}

// BuildBrowserUnavailableResult creates a stable degraded-mode envelope when no
// browser backend is configured or available. It never starts a browser.
func BuildBrowserUnavailableResult(action BrowserAction, reason string) BrowserResultEnvelope {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unavailable"
	}
	return BrowserResultEnvelope{
		Action:   action,
		Evidence: BrowserEvidenceBackendUnavailable,
		Text:     fmt.Sprintf("[browser_unavailable reason=%s]", reason),
		Tool: ToolResultEvidence{
			Code: BrowserEvidenceBackendUnavailable,
		},
	}
}

// NewBrowserTranscript creates an in-memory transcript. Persistence and runtime
// lifecycle ownership stay with the caller/session store.
func NewBrowserTranscript(taskID string) *BrowserTranscript {
	return &BrowserTranscript{TaskID: normalizeBrowserTaskID(taskID)}
}

// Record appends one transcript event.
func (t *BrowserTranscript) Record(event BrowserTranscriptEvent) {
	if t == nil {
		return
	}
	t.events = append(t.events, event)
}

// Events returns a defensive copy so callers cannot mutate stored transcript
// evidence after it has been recorded.
func (t *BrowserTranscript) Events() []BrowserTranscriptEvent {
	if t == nil || len(t.events) == 0 {
		return nil
	}
	out := make([]BrowserTranscriptEvent, len(t.events))
	copy(out, t.events)
	return out
}

func acceptBrowserAction(taskID string) BrowserActionDecision {
	return BrowserActionDecision{Allowed: true, Evidence: BrowserEvidenceActionAccepted, Route: BrowserRoute{SessionKey: normalizeBrowserTaskID(taskID)}}
}

func denyBrowserAction(evidence string) BrowserActionDecision {
	return BrowserActionDecision{Allowed: false, Evidence: evidence, Route: BrowserRoute{SessionKey: defaultBrowserTaskID}}
}

func guardEvidenceOrAccepted(guard BrowserSSRFGuardDecision) string {
	if guard.Evidence != "" {
		return guard.Evidence
	}
	return BrowserEvidenceActionAccepted
}
