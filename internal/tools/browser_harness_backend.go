package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

const (
	browserHarnessSnapshotPollTimeout  = 5 * time.Second
	browserHarnessSnapshotPollInterval = 100 * time.Millisecond
	browserHarnessScreenshotMaxBytes   = 8 * 1024

	BrowserHarnessEvidenceActionTimeout    = "browser_harness_action_timeout"
	BrowserHarnessEvidenceScreenshotFailed = "browser_harness_screenshot_failed"
)

// BrowserHarnessActionResult is the JSON result contract produced by Gormes'
// in-process browser backend.
type BrowserHarnessActionResult struct {
	SchemaVersion string                  `json:"schema_version"`
	Evidence      string                  `json:"evidence"`
	Kind          string                  `json:"kind,omitempty"`
	TaskID        string                  `json:"task_id,omitempty"`
	URL           string                  `json:"url,omitempty"`
	Title         string                  `json:"title,omitempty"`
	Text          string                  `json:"text,omitempty"`
	Artifact      string                  `json:"artifact,omitempty"`
	Message       string                  `json:"message,omitempty"`
	Data          map[string]any          `json:"data,omitempty"`
	Interactive   []BrowserHarnessElement `json:"interactive,omitempty"`
}

// BrowserHarnessElement is one visible page target addressable as @eN.
type BrowserHarnessElement struct {
	Ref  string `json:"ref"`
	Tag  string `json:"tag,omitempty"`
	Role string `json:"role,omitempty"`
	Text string `json:"text,omitempty"`
	X    int    `json:"x,omitempty"`
	Y    int    `json:"y,omitempty"`
}

// BrowserHarnessActionBackend executes an accepted browser action inside
// Gormes.
type BrowserHarnessActionBackend interface {
	RunAction(context.Context, BrowserHarnessActionRequest, map[string]string) (BrowserHarnessActionResult, error)
}

type BrowserHarnessActionBackendFunc func(context.Context, BrowserHarnessActionRequest, map[string]string) (BrowserHarnessActionResult, error)

func (f BrowserHarnessActionBackendFunc) RunAction(ctx context.Context, req BrowserHarnessActionRequest, env map[string]string) (BrowserHarnessActionResult, error) {
	return f(ctx, req, env)
}

// BrowserHarnessEnvBackend resolves the operator-provided CDP endpoint at run
// time. It never launches Chrome.
type BrowserHarnessEnvBackend struct{}

func (BrowserHarnessEnvBackend) RunAction(ctx context.Context, req BrowserHarnessActionRequest, env map[string]string) (BrowserHarnessActionResult, error) {
	endpoint := firstNonEmpty(env["CHROME_REMOTE_DEBUGGING_URL"], env["BROWSER_CDP_URL"], os.Getenv("CHROME_REMOTE_DEBUGGING_URL"), os.Getenv("BROWSER_CDP_URL"))
	backend, err := NewBrowserHarnessChromedpBackend(ctx, endpoint)
	if err != nil {
		return BrowserHarnessUnavailableBackend{Reason: err.Error()}.RunAction(ctx, req, env)
	}
	return backend.RunAction(ctx, req, env)
}

// BrowserHarnessUnavailableBackend reports typed degraded evidence when no CDP
// backend is configured or reachable.
type BrowserHarnessUnavailableBackend struct {
	Reason string
}

func (b BrowserHarnessUnavailableBackend) RunAction(_ context.Context, req BrowserHarnessActionRequest, _ map[string]string) (BrowserHarnessActionResult, error) {
	reason := strings.TrimSpace(b.Reason)
	if reason == "" {
		reason = "CDP backend not configured"
	}
	result := BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceBackendUnavailable,
		Kind:          normalizeBrowserHarnessActionKind(req.Kind),
		TaskID:        strings.TrimSpace(req.TaskID),
		Message:       reason,
	}
	return result, errors.New(reason)
}

func ParseBrowserHarnessActionJSON(raw []byte) (BrowserHarnessActionRequest, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return BrowserHarnessActionRequest{}, errors.New("browser action json is required")
	}
	var req BrowserHarnessActionRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return BrowserHarnessActionRequest{}, fmt.Errorf("decode browser action json: %w", err)
	}
	return req, ValidateBrowserHarnessActionRequest(req)
}

func ValidateBrowserHarnessActionRequest(req BrowserHarnessActionRequest) error {
	if req.SchemaVersion != browserHarnessActionSchemaVersion {
		return fmt.Errorf("schema_version = %q, want %q", req.SchemaVersion, browserHarnessActionSchemaVersion)
	}
	switch normalizeBrowserHarnessActionKind(req.Kind) {
	case BrowserActionNavigate:
		if strings.TrimSpace(req.URL) == "" {
			return errors.New("navigate requires url")
		}
	case BrowserActionClick:
		if strings.TrimSpace(req.Ref) == "" {
			return errors.New("click requires ref")
		}
	case BrowserActionType:
		if strings.TrimSpace(req.Ref) == "" {
			return errors.New("type requires ref")
		}
	case BrowserActionSnapshot, BrowserActionScroll, BrowserActionBack, "press", "console", "get_images", "vision", "cdp", "dialog":
	default:
		return fmt.Errorf("unsupported browser action kind %q", req.Kind)
	}
	return nil
}

func RunBrowserHarnessAction(ctx context.Context, req BrowserHarnessActionRequest, backend BrowserHarnessActionBackend, env map[string]string) (BrowserHarnessActionResult, error) {
	if err := ValidateBrowserHarnessActionRequest(req); err != nil {
		return BrowserHarnessActionResult{
			SchemaVersion: browserHarnessActionSchemaVersion,
			Evidence:      BrowserHarnessEvidenceInvalidAction,
			Kind:          normalizeBrowserHarnessActionKind(req.Kind),
			TaskID:        strings.TrimSpace(req.TaskID),
			Message:       err.Error(),
		}, err
	}
	if backend == nil {
		backend = BrowserHarnessEnvBackend{}
	}
	result, err := backend.RunAction(ctx, req, env)
	if result.SchemaVersion == "" {
		result.SchemaVersion = browserHarnessActionSchemaVersion
	}
	if result.Kind == "" {
		result.Kind = normalizeBrowserHarnessActionKind(req.Kind)
	}
	if result.TaskID == "" {
		result.TaskID = strings.TrimSpace(req.TaskID)
	}
	if result.Evidence == "" {
		result.Evidence = BrowserHarnessEvidenceActionAccepted
	}
	return result, err
}

// BrowserHarnessCDPTransport is the narrow CDP command interface used by the
// internal browser backend. Tests provide fakes; production uses chromedp.
type BrowserHarnessCDPTransport interface {
	SendCommand(context.Context, string, any) (json.RawMessage, error)
}

// BrowserHarnessChromedpBackend dispatches browser actions through a CDP
// transport.
type BrowserHarnessChromedpBackend struct {
	transport BrowserHarnessCDPTransport
}

func NewBrowserHarnessChromedpBackend(ctx context.Context, endpoint string) (*BrowserHarnessChromedpBackend, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, errors.New("CDP backend not configured; set CHROME_REMOTE_DEBUGGING_URL or BROWSER_CDP_URL")
	}
	transport, err := newBrowserHarnessChromedpLiveTransport(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("CDP backend unavailable: %w", err)
	}
	return &BrowserHarnessChromedpBackend{transport: transport}, nil
}

func NewBrowserHarnessChromedpBackendFromTransport(transport BrowserHarnessCDPTransport) *BrowserHarnessChromedpBackend {
	if transport == nil {
		panic("NewBrowserHarnessChromedpBackendFromTransport: transport must not be nil")
	}
	return &BrowserHarnessChromedpBackend{transport: transport}
}

func (b *BrowserHarnessChromedpBackend) RunAction(ctx context.Context, req BrowserHarnessActionRequest, _ map[string]string) (BrowserHarnessActionResult, error) {
	if err := ctx.Err(); err != nil {
		return b.timeoutResult(req), fmt.Errorf("%s: context already done: %w", normalizeBrowserHarnessActionKind(req.Kind), err)
	}
	switch normalizeBrowserHarnessActionKind(req.Kind) {
	case BrowserActionNavigate:
		return b.runNavigate(ctx, req)
	case BrowserActionSnapshot:
		return b.runSnapshot(ctx, req)
	case BrowserActionClick:
		return b.runClick(ctx, req)
	case BrowserActionType:
		return b.runType(ctx, req)
	case BrowserActionScroll:
		return b.runScroll(ctx, req)
	case BrowserActionBack:
		return b.runBack(ctx, req)
	case "press":
		return b.runPress(ctx, req)
	case "console":
		return b.runConsole(ctx, req)
	case "get_images":
		return b.runGetImages(ctx, req)
	case "vision":
		return b.runVision(ctx, req)
	case "cdp":
		return b.runCDP(ctx, req)
	case "dialog":
		return b.runDialog(ctx, req)
	default:
		return BrowserHarnessActionResult{
			SchemaVersion: browserHarnessActionSchemaVersion,
			Evidence:      BrowserHarnessEvidenceInvalidAction,
			Kind:          normalizeBrowserHarnessActionKind(req.Kind),
			TaskID:        req.TaskID,
			Message:       fmt.Sprintf("unsupported browser action kind %q", req.Kind),
		}, fmt.Errorf("unsupported browser action kind %q", req.Kind)
	}
}

func (b *BrowserHarnessChromedpBackend) runNavigate(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	createResp, err := b.send(ctx, "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		return b.wrapErr(req, err)
	}
	var createResult struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(createResp, &createResult)
	if createResult.TargetID != "" {
		if _, err := b.send(ctx, "Target.activateTarget", map[string]any{"targetId": createResult.TargetID}); err != nil {
			return b.wrapErr(req, err)
		}
	}
	if _, err := b.send(ctx, "Page.navigate", map[string]any{"url": req.URL}); err != nil {
		return b.wrapErr(req, err)
	}
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runSnapshot(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runClick(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	coords, err := b.resolveRef(ctx, req.Ref)
	if err != nil {
		return b.invalidRefResult(req, req.Ref), err
	}
	mouseParams := map[string]any{"type": "mousePressed", "x": coords[0], "y": coords[1], "button": "left", "clickCount": 1}
	if _, err := b.send(ctx, "Input.dispatchMouseEvent", mouseParams); err != nil {
		return b.wrapErr(req, err)
	}
	mouseParams["type"] = "mouseReleased"
	if _, err := b.send(ctx, "Input.dispatchMouseEvent", mouseParams); err != nil {
		return b.wrapErr(req, err)
	}
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runType(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	if _, err := b.resolveRef(ctx, req.Ref); err != nil {
		return b.invalidRefResult(req, req.Ref), err
	}
	for _, ch := range req.Text {
		keyParams := map[string]any{"type": "keyDown", "key": string(ch), "text": string(ch)}
		if _, err := b.send(ctx, "Input.dispatchKeyEvent", keyParams); err != nil {
			return b.wrapErr(req, err)
		}
		keyParams["type"] = "keyUp"
		if _, err := b.send(ctx, "Input.dispatchKeyEvent", keyParams); err != nil {
			return b.wrapErr(req, err)
		}
	}
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runScroll(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	dy := 600.0
	if strings.ToLower(strings.TrimSpace(req.Direction)) == "up" {
		dy = -600.0
	}
	if _, err := b.send(ctx, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 500.0, "y": 400.0, "deltaX": 0.0, "deltaY": dy}); err != nil {
		return b.wrapErr(req, err)
	}
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runBack(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	if _, err := b.send(ctx, "Runtime.evaluate", map[string]any{"expression": "history.back(); void 0"}); err != nil {
		return b.wrapErr(req, err)
	}
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runPress(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	keyParams := map[string]any{"type": "keyDown", "key": req.Key}
	if _, err := b.send(ctx, "Input.dispatchKeyEvent", keyParams); err != nil {
		return b.wrapErr(req, err)
	}
	keyParams["type"] = "keyUp"
	if _, err := b.send(ctx, "Input.dispatchKeyEvent", keyParams); err != nil {
		return b.wrapErr(req, err)
	}
	return b.takeSnapshot(ctx, req)
}

func (b *BrowserHarnessChromedpBackend) runConsole(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	expr := strings.TrimSpace(req.Expression)
	if expr == "" {
		expr = "void 0"
	}
	raw, err := b.send(ctx, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true})
	if err != nil {
		return b.wrapErr(req, err)
	}
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionAccepted,
		Kind:          "console",
		TaskID:        req.TaskID,
		Data:          browserHarnessConsoleExpressionData(raw),
	}, nil
}

func (b *BrowserHarnessChromedpBackend) runGetImages(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	raw, err := b.send(ctx, "Runtime.evaluate", map[string]any{"expression": browserHarnessImagesJS(), "returnByValue": true})
	if err != nil {
		return b.wrapErr(req, err)
	}
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionAccepted,
		Kind:          "get_images",
		TaskID:        req.TaskID,
		Data:          map[string]any{"result": string(raw)},
	}, nil
}

func (b *BrowserHarnessChromedpBackend) runVision(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	raw, err := b.send(ctx, "Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 70})
	if err != nil {
		return BrowserHarnessActionResult{
			SchemaVersion: browserHarnessActionSchemaVersion,
			Evidence:      BrowserHarnessEvidenceScreenshotFailed,
			Kind:          "vision",
			TaskID:        req.TaskID,
			Message:       "Page.captureScreenshot failed",
		}, fmt.Errorf("Page.captureScreenshot: %w", err)
	}
	var screenshot struct {
		Data string `json:"data"`
	}
	_ = json.Unmarshal(raw, &screenshot)
	artifact := screenshot.Data
	const suffix = "...[truncated]"
	if len(artifact) > browserHarnessScreenshotMaxBytes {
		cutAt := browserHarnessScreenshotMaxBytes - len(suffix)
		if cutAt < 0 {
			cutAt = 0
		}
		artifact = artifact[:cutAt] + suffix
	}
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionAccepted,
		Kind:          "vision",
		TaskID:        req.TaskID,
		Artifact:      artifact,
		Message:       req.Question,
	}, nil
}

func (b *BrowserHarnessChromedpBackend) runCDP(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	method := strings.TrimSpace(req.Method)
	if method == "" {
		return BrowserHarnessActionResult{
			SchemaVersion: browserHarnessActionSchemaVersion,
			Evidence:      BrowserHarnessEvidenceInvalidAction,
			Kind:          "cdp",
			TaskID:        req.TaskID,
			Message:       "cdp requires method",
		}, errors.New("cdp requires method")
	}
	raw, err := b.send(ctx, method, req.Params)
	if err != nil {
		return b.wrapErr(req, err)
	}
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionAccepted,
		Kind:          "cdp",
		TaskID:        req.TaskID,
		Data:          map[string]any{"result": string(raw)},
	}, nil
}

func (b *BrowserHarnessChromedpBackend) runDialog(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	accept := true
	if strings.ToLower(strings.TrimSpace(req.DialogAction)) == "dismiss" {
		accept = false
	}
	params := map[string]any{"accept": accept}
	if req.PromptText != "" {
		params["promptText"] = req.PromptText
	}
	if _, err := b.send(ctx, "Page.handleJavaScriptDialog", params); err != nil {
		return b.wrapErr(req, err)
	}
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionAccepted,
		Kind:          "dialog",
		TaskID:        req.TaskID,
	}, nil
}

func (b *BrowserHarnessChromedpBackend) takeSnapshot(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, error) {
	deadline := time.Now().Add(browserHarnessSnapshotPollTimeout)
	for {
		result, snapshot, err := b.evaluateSnapshot(ctx, req)
		if err != nil {
			return result, err
		}
		if browserHarnessSnapshotReady(normalizeBrowserHarnessActionKind(req.Kind), snapshot) || time.Now().After(deadline) {
			return result, nil
		}
		timer := time.NewTimer(browserHarnessSnapshotPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return b.timeoutResult(req), ctx.Err()
		case <-timer.C:
		}
	}
}

func (b *BrowserHarnessChromedpBackend) evaluateSnapshot(ctx context.Context, req BrowserHarnessActionRequest) (BrowserHarnessActionResult, browserHarnessPageSnapshot, error) {
	raw, err := b.send(ctx, "Runtime.evaluate", map[string]any{"expression": browserHarnessSnapshotJS(4000), "returnByValue": true})
	if err != nil {
		result, wrappedErr := b.wrapErr(req, err)
		return result, browserHarnessPageSnapshot{}, wrappedErr
	}
	var evalResult struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if jsonErr := json.Unmarshal(raw, &evalResult); jsonErr != nil || len(evalResult.Result.Value) == 0 {
		evalResult.Result.Value = raw
	}
	var snapshot browserHarnessPageSnapshot
	_ = json.Unmarshal(evalResult.Result.Value, &snapshot)
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionAccepted,
		Kind:          normalizeBrowserHarnessActionKind(req.Kind),
		TaskID:        req.TaskID,
		URL:           snapshot.URL,
		Title:         snapshot.Title,
		Text:          snapshot.Text,
		Interactive:   snapshot.Interactive,
	}, snapshot, nil
}

func (b *BrowserHarnessChromedpBackend) resolveRef(ctx context.Context, ref string) ([2]float64, error) {
	expr := browserHarnessRefCenterJS() + "(" + jsonStringLiteral(ref) + ")"
	raw, err := b.send(ctx, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true})
	if err != nil {
		return [2]float64{}, err
	}
	var evalResult struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	valueRaw := raw
	if jsonErr := json.Unmarshal(raw, &evalResult); jsonErr == nil && len(evalResult.Result.Value) > 0 {
		valueRaw = evalResult.Result.Value
	}
	if string(valueRaw) == "null" || len(valueRaw) == 0 {
		return [2]float64{}, fmt.Errorf("ref %q not found in current snapshot", ref)
	}
	var coords struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := json.Unmarshal(valueRaw, &coords); err != nil {
		return [2]float64{}, fmt.Errorf("ref %q: unexpected coords shape", ref)
	}
	return [2]float64{coords.X, coords.Y}, nil
}

func (b *BrowserHarnessChromedpBackend) send(ctx context.Context, method string, params any) (json.RawMessage, error) {
	raw, err := b.transport.SendCommand(ctx, method, params)
	if err != nil {
		if browserHarnessIsTimeout(ctx, err) {
			return nil, fmt.Errorf("timeout: %w", err)
		}
		return nil, err
	}
	return raw, nil
}

func (b *BrowserHarnessChromedpBackend) wrapErr(req BrowserHarnessActionRequest, err error) (BrowserHarnessActionResult, error) {
	if browserHarnessIsTimeout(nil, err) {
		return b.timeoutResult(req), err
	}
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceInvalidAction,
		Kind:          normalizeBrowserHarnessActionKind(req.Kind),
		TaskID:        req.TaskID,
		Message:       sanitizeBrowserHarnessCDPError(err),
	}, err
}

func (b *BrowserHarnessChromedpBackend) timeoutResult(req BrowserHarnessActionRequest) BrowserHarnessActionResult {
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceActionTimeout,
		Kind:          normalizeBrowserHarnessActionKind(req.Kind),
		TaskID:        req.TaskID,
		Message:       "action timed out",
	}
}

func (b *BrowserHarnessChromedpBackend) invalidRefResult(req BrowserHarnessActionRequest, ref string) BrowserHarnessActionResult {
	return BrowserHarnessActionResult{
		SchemaVersion: browserHarnessActionSchemaVersion,
		Evidence:      BrowserHarnessEvidenceInvalidAction,
		Kind:          normalizeBrowserHarnessActionKind(req.Kind),
		TaskID:        req.TaskID,
		Message:       fmt.Sprintf("ref %q not found; take a fresh snapshot", sanitizeBrowserHarnessRef(ref)),
	}
}

type browserHarnessPageSnapshot struct {
	URL         string                  `json:"url"`
	Title       string                  `json:"title"`
	Text        string                  `json:"text"`
	Interactive []BrowserHarnessElement `json:"interactive"`
	ReadyState  string                  `json:"readyState"`
}

func browserHarnessSnapshotReady(kind string, snapshot browserHarnessPageSnapshot) bool {
	url := strings.TrimSpace(snapshot.URL)
	if strings.TrimSpace(snapshot.Title) != "" || strings.TrimSpace(snapshot.Text) != "" || len(snapshot.Interactive) > 0 {
		return true
	}
	if kind != BrowserActionNavigate && (url == "" || url == "about:blank") {
		return true
	}
	return false
}

func browserActionFromHarnessRequest(req BrowserHarnessActionRequest) BrowserAction {
	action := BrowserAction{
		Kind:   normalizeBrowserActionKindForEnvelope(req.Kind),
		TaskID: req.TaskID,
		URL:    req.URL,
		Text:   req.Text,
	}
	switch normalizeBrowserHarnessActionKind(req.Kind) {
	case BrowserActionClick, BrowserActionType:
		action.Selector = req.Ref
	}
	return action
}

func normalizeBrowserActionKindForEnvelope(kind string) string {
	switch normalizeBrowserHarnessActionKind(kind) {
	case BrowserActionNavigate, BrowserActionClick, BrowserActionType, BrowserActionSnapshot, BrowserActionBack, BrowserActionScroll:
		return normalizeBrowserHarnessActionKind(kind)
	case "press", "dialog":
		return BrowserActionWait
	default:
		return BrowserActionExtract
	}
}

func normalizeBrowserHarnessActionKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func browserHarnessIsTimeout(ctx context.Context, err error) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func sanitizeBrowserHarnessCDPError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, scheme := range []string{"ws://", "wss://", "http://", "https://"} {
		if idx := strings.Index(msg, scheme); idx >= 0 {
			msg = msg[:idx] + "[redacted]"
			break
		}
	}
	return truncateBrowserBridgeText(msg, defaultBrowserUseRedactedErrorPreviewLength)
}

func sanitizeBrowserHarnessRef(ref string) string {
	if len(ref) > 32 {
		return ref[:32]
	}
	return ref
}

type browserHarnessRuntimeEvaluateResponse struct {
	Result struct {
		Result struct {
			Type                string          `json:"type"`
			Value               json.RawMessage `json:"value"`
			UnserializableValue string          `json:"unserializableValue"`
			Description         string          `json:"description"`
		} `json:"result"`
		ExceptionDetails struct {
			Text      string `json:"text"`
			Exception struct {
				Description string          `json:"description"`
				Value       json.RawMessage `json:"value"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	} `json:"result"`
}

func browserHarnessConsoleExpressionData(raw json.RawMessage) map[string]any {
	var eval browserHarnessRuntimeEvaluateResponse
	if err := json.Unmarshal(raw, &eval); err != nil {
		return map[string]any{
			"success":     true,
			"result":      string(raw),
			"result_type": "str",
			"method":      "cdp_supervisor",
		}
	}
	if msg := browserHarnessRuntimeExceptionMessage(eval); msg != "" {
		return map[string]any{
			"success": false,
			"error":   msg,
		}
	}
	parsed := browserHarnessRuntimeValue(eval.Result.Result)
	if s, ok := parsed.(string); ok {
		var nested any
		if err := json.Unmarshal([]byte(s), &nested); err == nil {
			parsed = nested
		}
	}
	return map[string]any{
		"success":     true,
		"result":      parsed,
		"result_type": browserHarnessConsoleResultType(parsed),
		"method":      "cdp_supervisor",
	}
}

func browserHarnessRuntimeExceptionMessage(eval browserHarnessRuntimeEvaluateResponse) string {
	if msg := strings.TrimSpace(eval.Result.ExceptionDetails.Exception.Description); msg != "" {
		return msg
	}
	if len(eval.Result.ExceptionDetails.Exception.Value) > 0 {
		var value any
		if err := json.Unmarshal(eval.Result.ExceptionDetails.Exception.Value, &value); err == nil {
			return fmt.Sprint(value)
		}
		return string(eval.Result.ExceptionDetails.Exception.Value)
	}
	return strings.TrimSpace(eval.Result.ExceptionDetails.Text)
}

func browserHarnessRuntimeValue(remote struct {
	Type                string          `json:"type"`
	Value               json.RawMessage `json:"value"`
	UnserializableValue string          `json:"unserializableValue"`
	Description         string          `json:"description"`
}) any {
	if len(remote.Value) > 0 {
		var value any
		if err := json.Unmarshal(remote.Value, &value); err == nil {
			return value
		}
		return string(remote.Value)
	}
	if remote.UnserializableValue != "" {
		return remote.UnserializableValue
	}
	if remote.Description != "" {
		return remote.Description
	}
	if strings.TrimSpace(remote.Type) == "undefined" {
		return nil
	}
	return nil
}

func browserHarnessConsoleResultType(value any) string {
	switch v := value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case float64:
		if math.Trunc(v) == v {
			return "int"
		}
		return "float"
	case []any:
		return "list"
	case map[string]any:
		return "dict"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func jsonStringLiteral(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
