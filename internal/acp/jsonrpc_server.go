package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const ACPProtocolVersion = 1

type JSONRPCServer struct {
	Runtime *SessionRuntime
	Version string
}

func NewJSONRPCServer(runtime *SessionRuntime) *JSONRPCServer {
	if runtime == nil {
		runtime = NewSessionRuntime(SessionRuntimeConfig{})
	}
	return &JSONRPCServer{
		Runtime: runtime,
		Version: "0.0.0",
	}
}

func (s *JSONRPCServer) Handle(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if err := enc.Encode(jsonRPCResponse{JSONRPC: "2.0", Result: nil, Error: &jsonRPCError{Code: -32700, Message: "parse error"}}); err != nil {
				return err
			}
			continue
		}
		result, rpcErr := s.dispatch(ctx, req, enc)
		if req.ID == nil {
			continue
		}
		if err := enc.Encode(jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr}); err != nil {
			return err
		}
	}
	return scanner.Err()
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

func (s *JSONRPCServer) dispatch(ctx context.Context, req jsonRPCRequest, enc *json.Encoder) (any, *jsonRPCError) {
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
			_ = enc.Encode(jsonRPCNotification{
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
		return nil, &jsonRPCError{Code: -32601, Message: "method not found", Data: map[string]any{"method": req.Method}}
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
	provider := strings.TrimSpace(s.Runtime.Provider())
	var authMethods []map[string]any
	if provider != "" {
		authMethods = append(authMethods, map[string]any{
			"id":          provider,
			"name":        provider + " runtime credentials",
			"description": "Authenticate Gormes using the currently configured " + provider + " runtime credentials.",
		})
	}
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
		"authMethods": authMethods,
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

func promptEventUpdate(event PromptEvent) map[string]any {
	switch event.Kind {
	case PromptEventUserMessageChunk:
		return map[string]any{"sessionUpdate": string(event.Kind), "text": event.Text}
	case PromptEventSessionTitle:
		return map[string]any{"sessionUpdate": string(event.Kind), "title": event.Title}
	case PromptEventUsage:
		return map[string]any{"sessionUpdate": string(event.Kind), "usage": event.Usage}
	case PromptEventAgentMessageChunk:
		fallthrough
	default:
		return map[string]any{"sessionUpdate": string(PromptEventAgentMessageChunk), "text": event.Text}
	}
}
