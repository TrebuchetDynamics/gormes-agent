package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	BrowserToolNavigate  = "browser_navigate"
	BrowserToolSnapshot  = "browser_snapshot"
	BrowserToolClick     = "browser_click"
	BrowserToolType      = "browser_type"
	BrowserToolScroll    = "browser_scroll"
	BrowserToolBack      = "browser_back"
	BrowserToolPress     = "browser_press"
	BrowserToolConsole   = "browser_console"
	BrowserToolGetImages = "browser_get_images"
	BrowserToolVision    = "browser_vision"
	BrowserToolCDP       = "browser_cdp"
	BrowserToolDialog    = "browser_dialog"

	browserHarnessActionSchemaVersion = "gormes.browser.action.v1"
)

// BrowserHarnessToolsConfig wires Hermes-visible browser_* tools onto the
// in-process Gormes browser backend. Runner stays for explicit legacy
// browser-harness compatibility tests.
type BrowserHarnessToolsConfig struct {
	Runner    BrowserHarnessProcessRunner
	Backend   BrowserHarnessActionBackend
	Command   string
	Protocol  string
	Env       map[string]string
	Budget    ToolResultBudgetConfig
	Timeout   time.Duration
	MediaType string
}

// BrowserHarnessToolResponse is the JSON payload returned by browser_* harness
// wrappers. Text is already bounded by BrowserResultEnvelope.
type BrowserHarnessToolResponse struct {
	Tool           string             `json:"tool"`
	Evidence       string             `json:"evidence"`
	ResultEvidence string             `json:"result_evidence"`
	Text           string             `json:"text,omitempty"`
	Artifact       string             `json:"artifact,omitempty"`
	Bytes          int                `json:"bytes,omitempty"`
	ToolEvidence   ToolResultEvidence `json:"tool_evidence"`
}

// BrowserHarnessActionRequest is the Go-native action JSON contract consumed
// by Gormes' in-process browser backend.
type BrowserHarnessActionRequest struct {
	SchemaVersion string         `json:"schema_version"`
	Kind          string         `json:"kind"`
	TaskID        string         `json:"task_id,omitempty"`
	URL           string         `json:"url,omitempty"`
	NewTab        bool           `json:"new_tab,omitempty"`
	Ref           string         `json:"ref,omitempty"`
	Text          string         `json:"text,omitempty"`
	Key           string         `json:"key,omitempty"`
	Direction     string         `json:"direction,omitempty"`
	Expression    string         `json:"expression,omitempty"`
	Method        string         `json:"method,omitempty"`
	Params        map[string]any `json:"params,omitempty"`
	DialogAction  string         `json:"dialog_action,omitempty"`
	PromptText    string         `json:"prompt_text,omitempty"`
	Full          bool           `json:"full,omitempty"`
	Question      string         `json:"question,omitempty"`
}

// BrowserHarnessTool is one Hermes-visible browser_* tool backed by Gormes'
// internal browser runtime. It preserves public tool names while hiding
// protocol generation and result budgeting behind the shared Tool interface.
type BrowserHarnessTool struct {
	name   string
	cfg    BrowserHarnessToolsConfig
	schema json.RawMessage
	desc   string
}

// NewBrowserHarnessTools returns the Gormes-backed Hermes browser
// tool surface in stable sorted-by-name order for deterministic descriptors.
func NewBrowserHarnessTools(cfg BrowserHarnessToolsConfig) []Tool {
	names := []string{
		BrowserToolBack,
		BrowserToolCDP,
		BrowserToolClick,
		BrowserToolConsole,
		BrowserToolDialog,
		BrowserToolGetImages,
		BrowserToolNavigate,
		BrowserToolPress,
		BrowserToolScroll,
		BrowserToolSnapshot,
		BrowserToolType,
		BrowserToolVision,
	}
	out := make([]Tool, 0, len(names))
	for _, name := range names {
		out = append(out, NewBrowserHarnessTool(name, cfg))
	}
	return out
}

// NewBrowserHarnessTool creates one named browser harness tool. Unknown names
// still produce a Tool whose Execute fails, which keeps tests explicit.
func NewBrowserHarnessTool(name string, cfg BrowserHarnessToolsConfig) *BrowserHarnessTool {
	desc, schema := browserHarnessToolDescriptor(name)
	return &BrowserHarnessTool{name: name, cfg: cloneBrowserHarnessToolsConfig(cfg), desc: desc, schema: schema}
}

func (t *BrowserHarnessTool) Name() string { return t.name }

func (t *BrowserHarnessTool) Description() string { return t.desc }

func (t *BrowserHarnessTool) Schema() json.RawMessage {
	return append(json.RawMessage(nil), t.schema...)
}

func (t *BrowserHarnessTool) Timeout() time.Duration {
	if t.cfg.Timeout > 0 {
		return t.cfg.Timeout + 5*time.Second
	}
	return defaultBrowserHarnessTimeout + 5*time.Second
}

func (t *BrowserHarnessTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in map[string]any
	if len(args) == 0 {
		in = map[string]any{}
	} else if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("%s: invalid args: %w", t.name, err)
	}
	request, action, err := t.buildCommandRequest(in)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", t.name, err)
	}
	taskID := stringArg(in, "task_id")
	bridge := BrowserHarnessBridge{
		Command:  t.cfg.Command,
		Protocol: t.cfg.Protocol,
		Runner:   t.cfg.Runner,
		Backend:  t.cfg.Backend,
	}
	request.Command = t.cfg.Command
	request.Protocol = t.cfg.Protocol
	request.TaskID = taskID
	request.Action = action
	request.Env = cloneStringMap(t.cfg.Env)
	request.Timeout = t.cfg.Timeout
	request.MediaType = firstNonEmpty(t.cfg.MediaType, "application/json")
	request.Budget = t.cfg.Budget
	result, err := bridge.Run(ctx, BrowserHarnessCommandRequest{
		Command:    request.Command,
		Protocol:   request.Protocol,
		Code:       request.Code,
		ActionJSON: request.ActionJSON,
		TaskID:     request.TaskID,
		Action:     request.Action,
		Backend:    t.cfg.Backend,
		Env:        request.Env,
		Timeout:    request.Timeout,
		MediaType:  request.MediaType,
		Budget:     request.Budget,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", t.name, err)
	}
	return json.Marshal(BrowserHarnessToolResponse{
		Tool:           t.name,
		Evidence:       result.Evidence,
		ResultEvidence: result.Envelope.Evidence,
		Text:           result.Envelope.Text,
		Artifact:       result.Envelope.Tool.Artifact,
		Bytes:          result.Envelope.Tool.Bytes,
		ToolEvidence:   result.Envelope.Tool,
	})
}

func (t *BrowserHarnessTool) buildCommandRequest(args map[string]any) (BrowserHarnessCommandRequest, BrowserAction, error) {
	protocol := normalizeBrowserHarnessProtocol(t.cfg.Protocol, t.cfg.Command)
	if protocol == BrowserHarnessProtocolLegacy {
		code, action, err := t.buildCode(args)
		return BrowserHarnessCommandRequest{Code: code}, action, err
	}
	actionReq, action, err := t.buildActionRequest(args)
	if err != nil {
		return BrowserHarnessCommandRequest{}, action, err
	}
	raw, err := json.Marshal(actionReq)
	if err != nil {
		return BrowserHarnessCommandRequest{}, action, err
	}
	return BrowserHarnessCommandRequest{ActionJSON: raw}, action, nil
}

func (t *BrowserHarnessTool) buildActionRequest(args map[string]any) (BrowserHarnessActionRequest, BrowserAction, error) {
	taskID := stringArg(args, "task_id")
	req := BrowserHarnessActionRequest{SchemaVersion: browserHarnessActionSchemaVersion, TaskID: taskID}
	switch t.name {
	case BrowserToolNavigate:
		url := stringArg(args, "url")
		if strings.TrimSpace(url) == "" {
			return req, BrowserAction{}, errors.New("url is required")
		}
		action := BrowserAction{Kind: BrowserActionNavigate, TaskID: taskID, URL: url}
		if decision := ValidateBrowserAction(action); !decision.Allowed {
			return req, action, fmt.Errorf("browser action denied: %s", decision.Evidence)
		}
		req.Kind = BrowserActionNavigate
		req.URL = url
		req.NewTab = true
		return req, action, nil
	case BrowserToolSnapshot:
		req.Kind = BrowserActionSnapshot
		req.Full = boolArg(args, "full")
		return req, BrowserAction{Kind: BrowserActionSnapshot, TaskID: taskID}, nil
	case BrowserToolClick:
		ref := stringArg(args, "ref")
		if strings.TrimSpace(ref) == "" {
			return req, BrowserAction{}, errors.New("ref is required")
		}
		req.Kind = BrowserActionClick
		req.Ref = ref
		return req, BrowserAction{Kind: BrowserActionClick, TaskID: taskID, Selector: ref}, nil
	case BrowserToolType:
		ref := stringArg(args, "ref")
		text := stringArg(args, "text")
		if strings.TrimSpace(ref) == "" {
			return req, BrowserAction{}, errors.New("ref is required")
		}
		req.Kind = BrowserActionType
		req.Ref = ref
		req.Text = text
		return req, BrowserAction{Kind: BrowserActionType, TaskID: taskID, Selector: ref, Text: text}, nil
	case BrowserToolScroll:
		req.Kind = BrowserActionScroll
		req.Direction = strings.ToLower(firstNonEmpty(stringArg(args, "direction"), "down"))
		return req, BrowserAction{Kind: BrowserActionScroll, TaskID: taskID}, nil
	case BrowserToolBack:
		req.Kind = BrowserActionBack
		return req, BrowserAction{Kind: BrowserActionBack, TaskID: taskID}, nil
	case BrowserToolPress:
		key := stringArg(args, "key")
		if strings.TrimSpace(key) == "" {
			return req, BrowserAction{}, errors.New("key is required")
		}
		req.Kind = "press"
		req.Key = key
		return req, BrowserAction{Kind: BrowserActionWait, TaskID: taskID, Text: key}, nil
	case BrowserToolConsole:
		req.Kind = "console"
		req.Expression = stringArg(args, "expression")
		return req, BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
	case BrowserToolGetImages:
		req.Kind = "get_images"
		return req, BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
	case BrowserToolVision:
		req.Kind = "vision"
		req.Question = stringArg(args, "question")
		return req, BrowserAction{Kind: BrowserActionSnapshot, TaskID: taskID}, nil
	case BrowserToolCDP:
		method := stringArg(args, "method")
		if strings.TrimSpace(method) == "" {
			return req, BrowserAction{}, errors.New("method is required")
		}
		req.Kind = "cdp"
		req.Method = method
		req.Params = mapArg(args, "params")
		return req, BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
	case BrowserToolDialog:
		req.Kind = "dialog"
		req.DialogAction = strings.ToLower(firstNonEmpty(stringArg(args, "action"), "accept"))
		req.PromptText = stringArg(args, "prompt_text")
		return req, BrowserAction{Kind: BrowserActionWait, TaskID: taskID}, nil
	default:
		return req, BrowserAction{}, fmt.Errorf("unsupported browser harness tool %q", t.name)
	}
}

func (t *BrowserHarnessTool) buildCode(args map[string]any) (string, BrowserAction, error) {
	taskID := stringArg(args, "task_id")
	switch t.name {
	case BrowserToolNavigate:
		url := stringArg(args, "url")
		if strings.TrimSpace(url) == "" {
			return "", BrowserAction{}, errors.New("url is required")
		}
		action := BrowserAction{Kind: BrowserActionNavigate, TaskID: taskID, URL: url}
		if decision := ValidateBrowserAction(action); !decision.Allowed {
			return "", action, fmt.Errorf("browser action denied: %s", decision.Evidence)
		}
		return joinBrowserPython(
			"import json",
			"new_tab("+pyString(url)+")",
			"wait_for_load()",
			browserHarnessPrintSnapshot(false),
		), action, nil
	case BrowserToolSnapshot:
		full := boolArg(args, "full")
		return joinBrowserPython("import json", browserHarnessPrintSnapshot(full)), BrowserAction{Kind: BrowserActionSnapshot, TaskID: taskID}, nil
	case BrowserToolClick:
		ref := stringArg(args, "ref")
		if strings.TrimSpace(ref) == "" {
			return "", BrowserAction{}, errors.New("ref is required")
		}
		action := BrowserAction{Kind: BrowserActionClick, TaskID: taskID, Selector: ref}
		return joinBrowserPython(
			"import json",
			browserHarnessRefHelper(),
			"_gormes_target = _gormes_ref_center("+pyString(ref)+")",
			"if not _gormes_target: raise RuntimeError('browser ref not found: '+str("+pyString(ref)+"))",
			"click_at_xy(_gormes_target['x'], _gormes_target['y'])",
			"wait(0.2)",
			browserHarnessPrintSnapshot(false),
		), action, nil
	case BrowserToolType:
		ref := stringArg(args, "ref")
		text := stringArg(args, "text")
		if strings.TrimSpace(ref) == "" {
			return "", BrowserAction{}, errors.New("ref is required")
		}
		action := BrowserAction{Kind: BrowserActionType, TaskID: taskID, Selector: ref, Text: text}
		return joinBrowserPython(
			"import json",
			browserHarnessRefHelper(),
			"_gormes_target = _gormes_ref_center("+pyString(ref)+")",
			"if not _gormes_target: raise RuntimeError('browser ref not found: '+str("+pyString(ref)+"))",
			"click_at_xy(_gormes_target['x'], _gormes_target['y'])",
			"js(_gormes_clear_expression("+pyString(ref)+"))",
			"type_text("+pyString(text)+")",
			"wait(0.2)",
			browserHarnessPrintSnapshot(false),
		), action, nil
	case BrowserToolScroll:
		direction := strings.ToLower(firstNonEmpty(stringArg(args, "direction"), "down"))
		dy := 600
		if direction == "up" {
			dy = -600
		}
		return joinBrowserPython(
			"import json",
			"_gormes_info = page_info()",
			fmt.Sprintf("scroll((_gormes_info.get('w') or 1000)/2, (_gormes_info.get('h') or 800)/2, dy=%d)", dy),
			"wait(0.2)",
			browserHarnessPrintSnapshot(false),
		), BrowserAction{Kind: BrowserActionScroll, TaskID: taskID}, nil
	case BrowserToolBack:
		return joinBrowserPython(
			"import json",
			"js('history.back()')",
			"wait_for_load(5)",
			browserHarnessPrintSnapshot(false),
		), BrowserAction{Kind: BrowserActionBack, TaskID: taskID}, nil
	case BrowserToolPress:
		key := stringArg(args, "key")
		if strings.TrimSpace(key) == "" {
			return "", BrowserAction{}, errors.New("key is required")
		}
		return joinBrowserPython(
			"import json",
			"press_key("+pyString(key)+")",
			"wait(0.2)",
			browserHarnessPrintSnapshot(false),
		), BrowserAction{Kind: BrowserActionWait, TaskID: taskID, Text: key}, nil
	case BrowserToolConsole:
		expr := stringArg(args, "expression")
		if expr != "" {
			return joinBrowserPython(
				"import json",
				"_gormes_result = js("+pyString(expr)+")",
				"print(json.dumps({'expression': "+pyString(expr)+", 'result': _gormes_result}, ensure_ascii=False, default=str))",
			), BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
		}
		return joinBrowserPython(
			"import json",
			"print(json.dumps({'events': drain_events()}, ensure_ascii=False, default=str))",
		), BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
	case BrowserToolGetImages:
		return joinBrowserPython(
			"import json",
			"print(json.dumps(js("+pyString(browserHarnessImagesJS())+"), ensure_ascii=False, default=str))",
		), BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
	case BrowserToolVision:
		question := stringArg(args, "question")
		return joinBrowserPython(
			"import json",
			"_gormes_path = capture_screenshot(full=False, max_dim=1800)",
			"print(json.dumps({'screenshot_path': _gormes_path, 'question': "+pyString(question)+", 'analysis': 'browser_harness_screenshot_captured'}, ensure_ascii=False))",
		), BrowserAction{Kind: BrowserActionSnapshot, TaskID: taskID}, nil
	case BrowserToolCDP:
		method := stringArg(args, "method")
		if strings.TrimSpace(method) == "" {
			return "", BrowserAction{}, errors.New("method is required")
		}
		params := mapArg(args, "params")
		paramsRaw, _ := json.Marshal(params)
		return joinBrowserPython(
			"import json",
			"_gormes_params = json.loads("+pyString(string(paramsRaw))+")",
			"print(json.dumps(cdp("+pyString(method)+", **_gormes_params), ensure_ascii=False, default=str))",
		), BrowserAction{Kind: BrowserActionExtract, TaskID: taskID}, nil
	case BrowserToolDialog:
		action := strings.ToLower(firstNonEmpty(stringArg(args, "action"), "accept"))
		accept := action != "dismiss"
		payload := map[string]any{"accept": accept}
		if promptText := stringArg(args, "prompt_text"); promptText != "" {
			payload["promptText"] = promptText
		}
		raw, _ := json.Marshal(payload)
		return joinBrowserPython(
			"import json",
			"_gormes_params = json.loads("+pyString(string(raw))+")",
			"print(json.dumps(cdp('Page.handleJavaScriptDialog', **_gormes_params), ensure_ascii=False, default=str))",
		), BrowserAction{Kind: BrowserActionWait, TaskID: taskID}, nil
	default:
		return "", BrowserAction{}, fmt.Errorf("unsupported browser harness tool %q", t.name)
	}
}

func browserHarnessToolDescriptor(name string) (string, json.RawMessage) {
	switch name {
	case BrowserToolNavigate:
		return "Navigate to a URL in the browser. Opens a new tab and returns a compact snapshot with @e refs.", json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"task_id":{"type":"string"}},"required":["url"]}`)
	case BrowserToolSnapshot:
		return "Get a text snapshot of the current browser page with @e refs for follow-up click/type actions.", json.RawMessage(`{"type":"object","properties":{"full":{"type":"boolean"},"task_id":{"type":"string"}},"required":[]}`)
	case BrowserToolClick:
		return "Click an element by @e ref from browser_snapshot or browser_navigate.", json.RawMessage(`{"type":"object","properties":{"ref":{"type":"string"},"task_id":{"type":"string"}},"required":["ref"]}`)
	case BrowserToolType:
		return "Type text into an element by @e ref from browser_snapshot or browser_navigate.", json.RawMessage(`{"type":"object","properties":{"ref":{"type":"string"},"text":{"type":"string"},"task_id":{"type":"string"}},"required":["ref","text"]}`)
	case BrowserToolScroll:
		return "Scroll the current browser page up or down.", json.RawMessage(`{"type":"object","properties":{"direction":{"type":"string","enum":["up","down"]},"task_id":{"type":"string"}},"required":[]}`)
	case BrowserToolBack:
		return "Navigate back in browser history.", json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":[]}`)
	case BrowserToolPress:
		return "Press a keyboard key in the current browser page.", json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"},"task_id":{"type":"string"}},"required":["key"]}`)
	case BrowserToolConsole:
		return "Read browser events or evaluate a JavaScript expression in the current page.", json.RawMessage(`{"type":"object","properties":{"clear":{"type":"boolean"},"expression":{"type":"string"},"task_id":{"type":"string"}},"required":[]}`)
	case BrowserToolGetImages:
		return "List images on the current page with src, alt, and dimensions.", json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":[]}`)
	case BrowserToolVision:
		return "Capture a screenshot through the browser runtime and return the artifact path for visual inspection.", json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"},"annotate":{"type":"boolean"},"task_id":{"type":"string"}},"required":["question"]}`)
	case BrowserToolCDP:
		return "Send a raw Chrome DevTools Protocol command through the browser runtime.", json.RawMessage(`{"type":"object","properties":{"method":{"type":"string"},"params":{"type":"object"},"task_id":{"type":"string"}},"required":["method"]}`)
	case BrowserToolDialog:
		return "Accept or dismiss the current JavaScript dialog.", json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["accept","dismiss"]},"prompt_text":{"type":"string"},"task_id":{"type":"string"}},"required":["action"]}`)
	default:
		return "Unsupported browser harness tool.", json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
	}
}

func browserHarnessPrintSnapshot(full bool) string {
	limit := 4000
	if full {
		limit = 16000
	}
	return "print(json.dumps(js(" + pyString(browserHarnessSnapshotJS(limit)) + "), ensure_ascii=False, default=str))"
}

func browserHarnessRefHelper() string {
	return joinBrowserPython(
		"def _gormes_ref_center(ref):",
		"    return js("+pyString(browserHarnessRefCenterJS())+".replace('__GORMES_REF__', ref))",
		"def _gormes_clear_expression(ref):",
		"    return "+pyString(browserHarnessClearRefJS())+".replace('__GORMES_REF__', ref)",
	)
}

func browserHarnessSnapshotJS(limit int) string {
	return fmt.Sprintf(`(() => {
const selector = 'a,button,input,textarea,select,[role="button"],[role="link"],[onclick],[tabindex]:not([tabindex="-1"])';
const visible = (el) => {
  const r = el.getBoundingClientRect();
  const s = getComputedStyle(el);
  return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none';
};
const label = (el) => (el.innerText || el.getAttribute('aria-label') || el.getAttribute('title') || el.getAttribute('placeholder') || el.value || el.href || '').trim().replace(/\s+/g, ' ').slice(0, 180);
const nodes = Array.from(document.querySelectorAll(selector)).filter(visible).slice(0, 100);
const interactive = nodes.map((el, i) => {
  const r = el.getBoundingClientRect();
  return {ref: '@e' + (i + 1), tag: el.tagName.toLowerCase(), role: el.getAttribute('role') || '', text: label(el), x: Math.round(r.left + r.width / 2), y: Math.round(r.top + r.height / 2)};
});
const bodyText = ((document.body && document.body.innerText) || '').trim().replace(/\n{3,}/g, '\n\n').slice(0, %d);
return {url: location.href, title: document.title, text: bodyText, interactive};
})()`, limit)
}

func browserHarnessRefCenterJS() string {
	return `(() => {
const wanted = '__GORMES_REF__';
const n = Number(String(wanted).replace(/^@?e/, '')) - 1;
const selector = 'a,button,input,textarea,select,[role="button"],[role="link"],[onclick],[tabindex]:not([tabindex="-1"])';
const visible = (el) => {
  const r = el.getBoundingClientRect();
  const s = getComputedStyle(el);
  return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none';
};
const nodes = Array.from(document.querySelectorAll(selector)).filter(visible).slice(0, 100);
const el = nodes[n];
if (!el) return null;
el.scrollIntoView({block: 'center', inline: 'center'});
const r = el.getBoundingClientRect();
return {x: Math.round(r.left + r.width / 2), y: Math.round(r.top + r.height / 2)};
})()`
}

func browserHarnessClearRefJS() string {
	return `(() => {
const wanted = '__GORMES_REF__';
const n = Number(String(wanted).replace(/^@?e/, '')) - 1;
const selector = 'a,button,input,textarea,select,[role="button"],[role="link"],[onclick],[tabindex]:not([tabindex="-1"])';
const visible = (el) => {
  const r = el.getBoundingClientRect();
  const s = getComputedStyle(el);
  return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none';
};
const el = Array.from(document.querySelectorAll(selector)).filter(visible).slice(0, 100)[n];
if (!el) return false;
el.focus();
if ('value' in el) {
  el.value = '';
  el.dispatchEvent(new Event('input', {bubbles: true}));
  el.dispatchEvent(new Event('change', {bubbles: true}));
}
return true;
})()`
}

func browserHarnessImagesJS() string {
	return `Array.from(document.images).slice(0, 200).map((img, i) => ({index: i, src: img.currentSrc || img.src || '', alt: img.alt || '', width: img.naturalWidth || img.width || 0, height: img.naturalHeight || img.height || 0}))`
}

func joinBrowserPython(lines ...string) string {
	return strings.Join(lines, "\n")
}

func pyString(s string) string {
	return strconv.Quote(s)
}

func stringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func boolArg(args map[string]any, key string) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true") || typed == "1"
	default:
		return false
	}
}

func mapArg(args map[string]any, key string) map[string]any {
	value, ok := args[key]
	if !ok || value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	}
	return map[string]any{}
}

func cloneBrowserHarnessToolsConfig(cfg BrowserHarnessToolsConfig) BrowserHarnessToolsConfig {
	cfg.Env = cloneStringMap(cfg.Env)
	cfg.Budget.OutputDir = strings.TrimSpace(cfg.Budget.OutputDir)
	return cfg
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
