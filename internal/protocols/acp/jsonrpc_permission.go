package acp

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// permissionResponse is the client result for session/request_permission.
type permissionResponse struct {
	Outcome struct {
		Outcome  string `json:"outcome"`
		OptionID string `json:"optionId,omitempty"`
	} `json:"outcome"`
}

// RequestPermission sends a session/request_permission request to the client
// and waits for its answer. Timeouts, transport failures, and unknown options
// fail closed as deny.
func (s *JSONRPCServer) RequestPermission(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
	write := s.currentWriter()
	if write == nil {
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_requester_unavailable"}, nil
	}
	options := req.Options
	if len(options) == 0 {
		options = BuildPermissionOptions(true, true, false)
	}
	toolCall := req.ToolCall
	if toolCall == nil {
		built := NewPermissionToolCall(req.Command, req.Description)
		toolCall = &built
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(req.Key)
	}

	id := s.nextID.Add(1)
	key := strconv.FormatInt(id, 10)
	ch := make(chan jsonRPCClientResponse, 1)
	s.pendingMu.Lock()
	if s.pending == nil {
		s.pending = make(map[string]chan jsonRPCClientResponse)
	}
	s.pending[key] = ch
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, key)
		s.pendingMu.Unlock()
	}()
	closed := s.clientClosedChannel()

	params, err := json.Marshal(map[string]any{
		"sessionId": sessionID,
		"toolCall":  toolCall,
		"options":   options,
	})
	if err != nil {
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_request_invalid"}, nil
	}
	if err := write(jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: "session/request_permission", Params: params}); err != nil {
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_request_failed"}, nil
	}

	timeout := s.PermissionTimeout
	if timeout <= 0 {
		timeout = defaultPermissionTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_error"}, nil
		}
		var parsed permissionResponse
		if err := json.Unmarshal(resp.Result, &parsed); err != nil {
			return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_response_invalid"}, nil
		}
		switch strings.TrimSpace(parsed.Outcome.Outcome) {
		case "selected":
			return PermissionDecisionForOption(parsed.Outcome.OptionID, options), nil
		case "cancelled":
			return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_cancelled"}, nil
		default:
			return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_outcome_invalid"}, nil
		}
	case <-timer.C:
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_timeout"}, nil
	case <-closed:
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_client_closed"}, nil
	case <-ctx.Done():
		reason := "permission_cancelled"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			reason = "permission_timeout"
		}
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: reason}, nil
	}
}

func (s *JSONRPCServer) currentWriter() func(any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.write
}

func (s *JSONRPCServer) clientClosedChannel() <-chan struct{} {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.clientClosed
}
