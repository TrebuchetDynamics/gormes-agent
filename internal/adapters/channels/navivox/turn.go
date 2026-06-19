package navivox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	navivoxMaxRequestIDLen = 128
	navivoxMaxSessionIDLen = 256
)

type turnRequest struct {
	RequestID string         `json:"request_id"`
	SessionID string         `json:"session_id,omitempty"`
	Text      string         `json:"text"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// turnInput is the transport-neutral Navivox turn seam. HTTP and WebSocket
// callers normalize into this shape before the channel records session/profile
// state or enqueues a gateway event. Voice capture, STT, TTS, and run-record
// evidence should land behind this seam instead of teaching transport handlers
// provider or audio-retention policy.
type turnInput struct {
	RequestID string
	SessionID string
	Text      string
	Metadata  map[string]any
}

func turnInputFromRequest(req turnRequest) turnInput {
	return turnInput{
		RequestID: req.RequestID,
		SessionID: req.SessionID,
		Text:      req.Text,
		Metadata:  req.Metadata,
	}
}

func turnInputFromClientMessage(msg ClientMessage) turnInput {
	return turnInput{
		RequestID: msg.RequestID,
		SessionID: msg.SessionID,
		Text:      msg.Text,
		Metadata:  msg.Metadata,
	}
}

func (c *Channel) handleTurn(inbox chan<- gateway.InboundEvent) func(http.ResponseWriter, *http.Request, string) {
	return func(w http.ResponseWriter, r *http.Request, identity string) {
		if r.Method != http.MethodPost {
			writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
			return
		}
		if !navivoxRequestHasJSONContentType(r) {
			writeNavivoxError(w, http.StatusUnsupportedMediaType, "", "unsupported_media_type", "Content-Type must be application/json")
			return
		}
		var req turnRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, navivoxMaxTurnRequestBytes))
		if err := decoder.Decode(&req); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				writeNavivoxError(w, http.StatusRequestEntityTooLarge, req.RequestID, "request_too_large", "Request is too large")
				return
			}
			writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Invalid JSON")
			return
		}
		if err := navivoxRejectTrailingJSONValue(decoder); err != nil {
			writeNavivoxError(w, http.StatusBadRequest, req.RequestID, "bad_request", "Invalid JSON")
			return
		}
		sessionID, contact, err := c.enqueueTurn(r.Context(), inbox, turnInputFromRequest(req), identity)
		if err != nil {
			writeNavivoxError(w, statusForNavivoxError(err), safeNavivoxRequestID(req.RequestID), codeForNavivoxError(err), safeNavivoxError(err))
			return
		}
		requestID, _ := normalizeNavivoxRequestID(req.RequestID)
		writeNavivoxJSON(w, http.StatusAccepted, map[string]any{
			"request_id": requestID,
			"session_id": sessionID,
			"status":     "queued",
		})
		if contact != nil {
			c.broadcastProfileContact(*contact)
		}
	}
}

func navivoxRejectTrailingJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return errors.New("unexpected trailing JSON value")
}

func navivoxRequestHasJSONContentType(r *http.Request) bool {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func (c *Channel) enqueueTurn(ctx context.Context, inbox chan<- gateway.InboundEvent, turn turnInput, identity string) (string, *ProfileContact, error) {
	requestID, ok := normalizeNavivoxRequestID(turn.RequestID)
	if requestID == "" {
		return "", nil, navivoxError{code: "bad_request", message: "request_id is required"}
	}
	if !ok {
		return "", nil, navivoxError{code: "bad_request", message: "request_id is too long"}
	}
	text := strings.TrimSpace(turn.Text)
	if text == "" {
		return "", nil, navivoxError{code: "bad_request", message: "text is required"}
	}
	sessionID, sessionIDOK := normalizeNavivoxSessionID(turn.SessionID)
	if !sessionIDOK {
		return "", nil, navivoxError{code: "bad_request", message: "session_id is too long"}
	}
	if sessionID == "" {
		sessionID = "navivox-" + c.newID()
	}
	serverID, profileID := profileScopeFromMetadata(turn.Metadata)
	metadata := c.voiceProfileMetadataForTurn(turn.Metadata, profileID)
	c.mu.Lock()
	if len(c.activeTurnBySession) > 0 {
		c.mu.Unlock()
		return "", nil, navivoxError{
			code:    "turn_in_progress",
			message: "A turn is already in progress.",
		}
	}
	if c.activeTurnBySession == nil {
		c.activeTurnBySession = make(map[string]string)
	}
	c.activeTurnBySession[sessionID] = identity
	session := c.ensureSessionLocked(sessionID, requestID)
	session.ProfileServer = serverID
	session.ProfileID = profileID
	c.mu.Unlock()
	ev := gateway.InboundEvent{
		Platform:  PlatformName,
		ChatID:    sessionID,
		ChatType:  "private",
		UserID:    "navivox",
		UserName:  identity,
		MsgID:     requestID,
		MessageID: requestID,
		Kind:      gateway.EventSubmit,
		Text:      text,
	}
	if err := enqueue(ctx, inbox, ev); err != nil {
		return "", nil, err
	}
	c.mu.Lock()
	c.recordRunStartLocked(sessionID, requestID, text, metadata)
	contact := c.profileContactRuntimeUpdateLocked(serverID, profileID, text, "user", ProfileContactTurnActive)
	c.mu.Unlock()
	c.log.Info("navivox turn queued", "client_identity", identity, "request_id", requestID, "session_id", sessionID, "action", "start_turn", "status", "queued")
	return sessionID, &contact, nil
}

func normalizeNavivoxRequestID(raw string) (string, bool) {
	requestID := strings.TrimSpace(raw)
	if len([]rune(requestID)) > navivoxMaxRequestIDLen {
		return "", false
	}
	return requestID, true
}

func safeNavivoxRequestID(raw string) string {
	requestID, ok := normalizeNavivoxRequestID(raw)
	if !ok {
		return ""
	}
	return requestID
}

func normalizeNavivoxSessionID(raw string) (string, bool) {
	sessionID := strings.TrimSpace(raw)
	if len([]rune(sessionID)) > navivoxMaxSessionIDLen {
		return "", false
	}
	return sessionID, true
}

func enqueue(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent) error {
	select {
	case inbox <- ev:
		return nil
	case <-ctx.Done():
		return navivoxError{code: "timeout", message: "request canceled"}
	default:
		return navivoxError{code: "runtime_error", message: "gateway inbox is full"}
	}
}
