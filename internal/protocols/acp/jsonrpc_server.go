package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const ACPProtocolVersion = 1

// defaultPermissionTimeout mirrors the Hermes approval default. Editors can
// still answer sooner; a timeout fails closed as deny.
const defaultPermissionTimeout = 300 * time.Second

type JSONRPCServer struct {
	Runtime           *SessionRuntime
	Version           string
	Diagnostics       *StdioDiagnostics
	PermissionTimeout time.Duration

	writeMu      sync.Mutex
	write        func(any) error
	clientClosed chan struct{}

	pendingMu sync.Mutex
	pending   map[string]chan jsonRPCClientResponse
	nextID    atomic.Int64
}

func NewJSONRPCServer(runtime *SessionRuntime) *JSONRPCServer {
	if runtime == nil {
		runtime = NewSessionRuntime(SessionRuntimeConfig{})
	}
	return &JSONRPCServer{
		Runtime: runtime,
		Version: "0.0.0",
		pending: make(map[string]chan jsonRPCClientResponse),
	}
}

// Handle serves newline-delimited JSON-RPC 2.0 until the input closes. Requests
// dispatch concurrently so the read loop can route client responses for
// server-initiated requests such as session/request_permission.
func (s *JSONRPCServer) Handle(ctx context.Context, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(out)
	write := func(value any) error {
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		return enc.Encode(value)
	}

	closed := make(chan struct{}, 1)
	s.writeMu.Lock()
	s.write = write
	s.clientClosed = closed
	s.writeMu.Unlock()
	defer func() {
		s.writeMu.Lock()
		s.write = nil
		s.clientClosed = nil
		s.writeMu.Unlock()
	}()

	var wg sync.WaitGroup
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var envelope jsonRPCEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			s.reportStdioError("parse_error", "")
			_ = write(jsonRPCResponse{JSONRPC: "2.0", Result: nil, Error: &jsonRPCError{Code: -32700, Message: "parse error"}})
			continue
		}
		if envelope.Method == "" {
			s.routeClientResponse(envelope)
			continue
		}
		req := jsonRPCRequest{
			JSONRPC: envelope.JSONRPC,
			ID:      envelope.ID,
			Method:  envelope.Method,
			Params:  envelope.Params,
		}
		if req.Method == "session/prompt" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, rpcErr := s.dispatch(ctx, req, write)
				if req.ID == nil {
					return
				}
				_ = write(jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr})
			}()
			continue
		}
		// Lifecycle and control methods stay ordered; prompts run concurrently
		// so cancel and permission responses keep flowing.
		result, rpcErr := s.dispatch(ctx, req, write)
		if req.ID != nil {
			_ = write(jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr})
		}
	}
	err := scanner.Err()
	// Stop waiting for client responses, but let in-flight requests (such as
	// session/new) finish against the caller's context.
	s.writeMu.Lock()
	if s.clientClosed == closed {
		close(closed)
		s.clientClosed = nil
	}
	s.writeMu.Unlock()
	wg.Wait()
	cancel()
	return err
}

type jsonRPCEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCClientResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

func (s *JSONRPCServer) routeClientResponse(envelope jsonRPCEnvelope) {
	key := jsonRPCIDKey(envelope.ID)
	if key == "" {
		return
	}
	s.pendingMu.Lock()
	ch := s.pending[key]
	s.pendingMu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- jsonRPCClientResponse{JSONRPC: envelope.JSONRPC, ID: envelope.ID, Result: envelope.Result, Error: envelope.Error}:
	default:
	}
}

func jsonRPCIDKey(id any) string {
	switch value := id.(type) {
	case nil:
		return ""
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return fmt.Sprint(value)
	}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (s *JSONRPCServer) dispatch(ctx context.Context, req jsonRPCRequest, write func(any) error) (any, *jsonRPCError) {
	switch req.Method {
	case "initialize":
		return s.initializeResult(), nil
	case "authenticate":
		var params authenticateParams
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		provider := strings.ToLower(strings.TrimSpace(s.Runtime.Provider()))
		method := strings.ToLower(strings.TrimSpace(params.MethodID))
		if method == TerminalSetupAuthMethodID {
			return nil, nil
		}
		if provider == "" || method == "" || provider != method {
			return nil, nil
		}
		return map[string]any{"authenticated": true}, nil
	case "session/new":
		var params sessionNewParams
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		sess, err := s.Runtime.NewSession(ctx, params.CWD)
		if err != nil {
			return nil, internalError(err)
		}
		s.bindPermissionRequester(sess.ID)
		return sessionResult(sess), nil
	case "session/load":
		var params sessionIDParams
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		sess, err := s.Runtime.LoadSession(ctx, params.SessionID, params.CWD)
		if err != nil {
			return nil, internalError(err)
		}
		if sess == nil {
			return nil, nil
		}
		s.bindPermissionRequester(sess.ID)
		return sessionResult(*sess), nil
	case "session/resume":
		var params sessionIDParams
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		sess, err := s.Runtime.ResumeSession(ctx, params.SessionID, params.CWD)
		if err != nil {
			return nil, internalError(err)
		}
		if sess == nil {
			return nil, nil
		}
		s.bindPermissionRequester(sess.ID)
		return sessionResult(*sess), nil
	case "session/cancel":
		var params sessionIDParams
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		return map[string]any{"cancelled": s.Runtime.CancelSession(ctx, params.SessionID)}, nil
	case "session/prompt":
		var params promptParams
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		result, err := s.Runtime.Prompt(ctx, RuntimePromptRequest{
			SessionID: params.SessionID,
			Blocks:    params.Prompt,
		}, func(event PromptEvent) {
			_ = write(jsonRPCNotification{
				JSONRPC: "2.0",
				Method:  "session/update",
				Params: map[string]any{
					"sessionId": params.SessionID,
					"update":    promptEventUpdate(event),
				},
			})
		})
		if err != nil && !strings.Contains(err.Error(), ErrJSONRPCSessionNotFound.Error()) {
			return nil, internalError(err)
		}
		return promptResult(result), nil
	default:
		if IsBenignProbeMethod(req.Method) {
			s.reportBenignProbe(req.Method)
		} else {
			s.reportStdioError("method_not_found", req.Method)
		}
		return nil, &jsonRPCError{Code: -32601, Message: "method not found", Data: map[string]any{"method": req.Method}}
	}
}

func (s *JSONRPCServer) bindPermissionRequester(sessionID string) {
	if s == nil || s.Runtime == nil {
		return
	}
	broker := s.Runtime.PermissionBroker()
	if broker == nil {
		return
	}
	broker.SetRequester(sessionID, s)
}

func (s *JSONRPCServer) reportBenignProbe(method string) {
	if s != nil && s.Diagnostics != nil {
		s.Diagnostics.BenignProbeSuppressed(method)
	}
}

func (s *JSONRPCServer) reportStdioError(reason, method string) {
	if s != nil && s.Diagnostics != nil {
		s.Diagnostics.Error(reason, method)
	}
}

type authenticateParams struct {
	MethodID string `json:"methodId"`
}

type sessionNewParams struct {
	CWD string `json:"cwd"`
}

type sessionIDParams struct {
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
}

type promptParams struct {
	SessionID string            `json:"sessionId"`
	Prompt    []ACPContentBlock `json:"prompt"`
}

func (s *JSONRPCServer) initializeResult() map[string]any {
	return map[string]any{
		"protocolVersion": ACPProtocolVersion,
		"agentInfo": map[string]any{
			"name":    "gormes-agent",
			"version": s.Version,
		},
		"agentCapabilities": map[string]any{
			"loadSession": true,
			"promptCapabilities": map[string]any{
				"image": true,
			},
			"sessionCapabilities": map[string]any{
				"fork":   map[string]any{},
				"list":   map[string]any{},
				"resume": map[string]any{},
			},
		},
		"authMethods": s.Runtime.AuthMethods(),
	}
}

func decodeParams(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, dst)
}

func invalidParams(err error) *jsonRPCError {
	return &jsonRPCError{Code: -32602, Message: "invalid params", Data: err.Error()}
}

func internalError(err error) *jsonRPCError {
	return &jsonRPCError{Code: -32000, Message: "acp_jsonrpc_error", Data: err.Error()}
}

func sessionResult(sess RuntimeSession) map[string]any {
	modelID := sess.Model
	if sess.Provider != "" && sess.Model != "" {
		modelID = sess.Provider + ":" + sess.Model
	}
	return map[string]any{
		"sessionId": sess.ID,
		"cwd":       sess.CWD,
		"models": map[string]any{
			"currentModelId": modelID,
			"availableModels": []map[string]any{
				{
					"modelId":     modelID,
					"name":        sess.Model,
					"description": fmt.Sprintf("Provider: %s", sess.Provider),
				},
			},
		},
	}
}

func promptResult(result PromptResult) map[string]any {
	out := map[string]any{
		"stopReason": firstNonEmpty(result.StopReason, "end_turn"),
	}
	if result.Usage != nil {
		out["usage"] = result.Usage
	}
	return out
}

// promptEventUpdate builds ACP session/update payloads. Agent and thought
// chunks carry a content block, matching the wire shape real editors read.
func promptEventUpdate(event PromptEvent) map[string]any {
	switch event.Kind {
	case PromptEventUserMessageChunk:
		return map[string]any{
			"sessionUpdate": string(PromptEventUserMessageChunk),
			"content":       map[string]any{"type": "text", "text": event.Text},
		}
	case PromptEventAgentThoughtChunk:
		return map[string]any{
			"sessionUpdate": string(PromptEventAgentThoughtChunk),
			"content":       map[string]any{"type": "text", "text": event.Text},
		}
	case PromptEventSessionTitle:
		return map[string]any{"sessionUpdate": string(PromptEventSessionTitle), "title": event.Title}
	case PromptEventUsage:
		return map[string]any{"sessionUpdate": string(PromptEventUsage), "usage": event.Usage}
	case PromptEventToolCall:
		out := map[string]any{"sessionUpdate": string(PromptEventToolCall)}
		mergeToolCallEvent(out, event.ToolCall)
		return out
	case PromptEventToolCallUpdate:
		out := map[string]any{"sessionUpdate": string(PromptEventToolCallUpdate)}
		mergeToolCallEvent(out, event.ToolCall)
		return out
	case PromptEventAgentMessageChunk:
		fallthrough
	default:
		return map[string]any{
			"sessionUpdate": string(PromptEventAgentMessageChunk),
			"content":       map[string]any{"type": "text", "text": event.Text},
		}
	}
}

func mergeToolCallEvent(out map[string]any, toolCall *ToolCallEvent) {
	if toolCall == nil {
		return
	}
	if toolCall.ToolCallID != "" {
		out["toolCallId"] = toolCall.ToolCallID
	}
	if toolCall.Title != "" {
		out["title"] = toolCall.Title
	}
	if toolCall.Kind != "" {
		out["kind"] = toolCall.Kind
	}
	if toolCall.Status != "" {
		out["status"] = toolCall.Status
	}
	if len(toolCall.Content) > 0 {
		out["content"] = toolCall.Content
	}
	if len(toolCall.RawInput) > 0 {
		out["rawInput"] = toolCall.RawInput
	}
}
