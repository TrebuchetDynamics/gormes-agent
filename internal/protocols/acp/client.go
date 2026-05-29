package acp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

const (
	ClientEvidenceConnected = "acp_client_connected"
	ClientEvidenceRowBacked = "acp_client_row_backed"

	clientDefaultSessionKey = "acp:default"
)

var (
	ErrClientSessionNotFound     = errors.New("acp client: session key not found")
	ErrClientLabelNotFound       = errors.New("acp client: session label not found")
	ErrClientPermissionTimeout   = errors.New("acp client: permission prompt timeout")
	ErrClientUnsupportedProvider = errors.New("acp client: unsupported ACP provider")
	ErrClientMissingAuth         = errors.New("acp client: missing auth")
)

type ProvenanceMode string

const (
	ProvenanceOff         ProvenanceMode = "off"
	ProvenanceMeta        ProvenanceMode = "meta"
	ProvenanceMetaReceipt ProvenanceMode = "meta+receipt"
)

func ParseProvenanceMode(raw string) (ProvenanceMode, error) {
	mode := ProvenanceMode(strings.TrimSpace(raw))
	if mode == "" {
		mode = ProvenanceOff
	}
	if !validProvenanceMode(mode) {
		return "", fmt.Errorf("invalid --provenance value %q; use off, meta, or meta+receipt", raw)
	}
	return mode, nil
}

type ClientOptions struct {
	CWD             string
	ServerCommand   string
	ServerArgs      []string
	ServerVerbose   bool
	Verbose         bool
	SessionKey      string
	SessionLabel    string
	RequireExisting bool
	ResetSession    bool
	NoPrefixCWD     bool
	ProvenanceMode  ProvenanceMode
	IDGenerator     func() string
}

type ClientBridge struct {
	Resolver    SessionResolver
	Connector   ClientConnector
	IDGenerator func() string
}

type SessionResolveRequest struct {
	SessionKey      string
	SessionLabel    string
	RequireExisting bool
	NewSessionID    func() string
}

type SessionResolution struct {
	SessionKey   string `json:"session_key"`
	SessionID    string `json:"session_id"`
	SessionLabel string `json:"session_label,omitempty"`
	Existing     bool   `json:"existing"`
	Reset        bool   `json:"reset,omitempty"`
}

type SessionResolver interface {
	Resolve(ctx context.Context, req SessionResolveRequest) (SessionResolution, error)
}

type SessionResetter interface {
	Reset(ctx context.Context, sessionKey string, newSessionID func() string) (SessionResolution, error)
}

type ClientInputProvenance struct {
	Kind            string `json:"kind"`
	OriginSessionID string `json:"origin_session_id"`
	SourceChannel   string `json:"source_channel"`
	SourceTool      string `json:"source_tool"`
}

type ClientConnectRequest struct {
	SessionKey    string                 `json:"session_key"`
	SessionID     string                 `json:"session_id"`
	CWD           string                 `json:"cwd,omitempty"`
	PrefixCWD     bool                   `json:"prefix_cwd"`
	Provenance    *ClientInputProvenance `json:"provenance,omitempty"`
	Receipt       string                 `json:"receipt,omitempty"`
	ServerCommand string                 `json:"server_command,omitempty"`
	ServerArgs    []string               `json:"server_args,omitempty"`
	ServerVerbose bool                   `json:"server_verbose,omitempty"`
	Verbose       bool                   `json:"verbose,omitempty"`
}

type ClientConnectResult struct {
	Connected    bool         `json:"connected"`
	ServerStatus ServerStatus `json:"server_status"`
	Message      string       `json:"message,omitempty"`
}

type ClientConnector interface {
	Connect(ctx context.Context, req ClientConnectRequest) (ClientConnectResult, error)
}

type ClientEvidence struct {
	Code          string   `json:"code"`
	Reason        string   `json:"reason,omitempty"`
	FallbackModes []string `json:"fallback_modes,omitempty"`
}

type ClientResult struct {
	OK             bool           `json:"ok"`
	Code           string         `json:"code"`
	Message        string         `json:"message,omitempty"`
	SessionKey     string         `json:"session_key,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	SessionLabel   string         `json:"session_label,omitempty"`
	Reset          bool           `json:"reset,omitempty"`
	ProvenanceMode ProvenanceMode `json:"provenance_mode,omitempty"`
	Receipt        string         `json:"receipt,omitempty"`
	PrefixCWD      bool           `json:"prefix_cwd"`
	Evidence       ClientEvidence `json:"evidence"`
	ServerStatus   ServerStatus   `json:"server_status"`
}

func (c ClientBridge) Run(ctx context.Context, opts ClientOptions) (ClientResult, error) {
	mode := opts.ProvenanceMode
	if mode == "" {
		mode = ProvenanceOff
	}
	if !validProvenanceMode(mode) {
		return ClientResult{}, fmt.Errorf("invalid provenance mode %q", mode)
	}

	resolver := c.Resolver
	if resolver == nil {
		return degradedClientResult("session_resolver_unavailable", "ACP client session resolver is unavailable", SessionResolution{}, opts, mode), nil
	}
	connector := c.Connector
	if connector == nil {
		connector = LocalClientConnector{}
	}
	newSessionID := firstIDGenerator(opts.IDGenerator, c.IDGenerator, defaultClientSessionID)
	prefixCWD := !opts.NoPrefixCWD

	resolution, err := resolver.Resolve(ctx, SessionResolveRequest{
		SessionKey:      opts.SessionKey,
		SessionLabel:    opts.SessionLabel,
		RequireExisting: opts.RequireExisting,
		NewSessionID:    newSessionID,
	})
	if err != nil {
		return degradedFromError(err, resolution, opts, mode), nil
	}
	if opts.ResetSession {
		resetter, ok := resolver.(SessionResetter)
		if !ok {
			return degradedClientResult("reset_session_unavailable", "ACP client session reset is unavailable", resolution, opts, mode), nil
		}
		reset, err := resetter.Reset(ctx, resolution.SessionKey, newSessionID)
		if err != nil {
			return degradedFromError(err, resolution, opts, mode), nil
		}
		reset.SessionLabel = resolution.SessionLabel
		reset.Reset = true
		resolution = reset
	}

	provenance, receipt := buildClientProvenance(mode, resolution)
	connectResult, err := connector.Connect(ctx, ClientConnectRequest{
		SessionKey:    resolution.SessionKey,
		SessionID:     resolution.SessionID,
		CWD:           opts.CWD,
		PrefixCWD:     prefixCWD,
		Provenance:    provenance,
		Receipt:       receipt,
		ServerCommand: opts.ServerCommand,
		ServerArgs:    append([]string(nil), opts.ServerArgs...),
		ServerVerbose: opts.ServerVerbose,
		Verbose:       opts.Verbose,
	})
	if err != nil {
		return degradedFromError(err, resolution, opts, mode), nil
	}
	if !connectResult.Connected {
		return degradedClientResult("acp_server_unavailable", "ACP client could not connect to the Go-native ACP server", resolution, opts, mode), nil
	}

	return ClientResult{
		OK:             true,
		Code:           ClientEvidenceConnected,
		Message:        firstNonEmpty(connectResult.Message, "connected to Go-native ACP server"),
		SessionKey:     resolution.SessionKey,
		SessionID:      resolution.SessionID,
		SessionLabel:   resolution.SessionLabel,
		Reset:          resolution.Reset,
		ProvenanceMode: mode,
		Receipt:        receipt,
		PrefixCWD:      prefixCWD,
		Evidence:       ClientEvidence{Code: ClientEvidenceConnected},
		ServerStatus:   connectResult.ServerStatus,
	}, nil
}

type SessionMapResolver struct {
	Map session.Map
}

type sessionMetadataLister interface {
	ListAllMetadata(ctx context.Context) ([]session.Metadata, error)
}

func NewSessionMapResolver(smap session.Map) *SessionMapResolver {
	return &SessionMapResolver{Map: smap}
}

func (r *SessionMapResolver) Resolve(ctx context.Context, req SessionResolveRequest) (SessionResolution, error) {
	if r == nil || r.Map == nil {
		return SessionResolution{}, errors.New("acp client: nil session map")
	}
	key := strings.TrimSpace(req.SessionKey)
	label := strings.TrimSpace(req.SessionLabel)
	if label != "" {
		resolution, ok, err := r.resolveLabel(ctx, label)
		if err != nil {
			return SessionResolution{SessionLabel: label}, err
		}
		if !ok {
			return SessionResolution{SessionLabel: label}, fmt.Errorf("%w: %s", ErrClientLabelNotFound, label)
		}
		resolution.SessionLabel = label
		return resolution, nil
	}
	if key == "" {
		key = clientDefaultSessionKey
	}
	return r.resolveKey(ctx, key, req.RequireExisting, req.NewSessionID, "")
}

func (r *SessionMapResolver) Reset(ctx context.Context, sessionKey string, newSessionID func() string) (SessionResolution, error) {
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		key = clientDefaultSessionKey
	}
	id := callIDGenerator(newSessionID)
	if err := r.Map.Put(ctx, key, id); err != nil {
		return SessionResolution{SessionKey: key}, err
	}
	return SessionResolution{SessionKey: key, SessionID: id, Existing: true, Reset: true}, nil
}

func (r *SessionMapResolver) resolveLabel(ctx context.Context, label string) (SessionResolution, bool, error) {
	lister, ok := r.Map.(sessionMetadataLister)
	if !ok {
		return SessionResolution{}, false, nil
	}
	items, err := lister.ListAllMetadata(ctx)
	if err != nil {
		return SessionResolution{}, false, err
	}
	for _, meta := range items {
		if metadataMatchesLabel(meta, label) {
			key := sessionKeyFromMetadata(meta)
			if key == "" {
				key = meta.SessionID
			}
			resolution, err := r.resolveKey(ctx, key, true, nil, meta.SessionID)
			return resolution, err == nil, err
		}
	}
	return SessionResolution{}, false, nil
}

func (r *SessionMapResolver) resolveKey(ctx context.Context, key string, requireExisting bool, newSessionID func() string, fallbackSessionID string) (SessionResolution, error) {
	sessionID, err := r.Map.Get(ctx, key)
	if err != nil {
		return SessionResolution{SessionKey: key}, err
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = strings.TrimSpace(fallbackSessionID)
	}
	if strings.TrimSpace(sessionID) == "" {
		if requireExisting {
			return SessionResolution{SessionKey: key}, fmt.Errorf("%w: %s", ErrClientSessionNotFound, key)
		}
		sessionID = callIDGenerator(newSessionID)
		if err := r.Map.Put(ctx, key, sessionID); err != nil {
			return SessionResolution{SessionKey: key}, err
		}
	}
	return SessionResolution{
		SessionKey: key,
		SessionID:  sessionID,
		Existing:   sessionID != "",
	}, nil
}

type LocalClientConnector struct {
	Server *ACPServer
}

func (c LocalClientConnector) Connect(_ context.Context, _ ClientConnectRequest) (ClientConnectResult, error) {
	server := c.Server
	if server == nil {
		server = NewACPServer(ACPConfig{Enabled: true})
	}
	return ClientConnectResult{
		Connected:    true,
		ServerStatus: server.Status(),
		Message:      "connected to Go-native ACP server",
	}, nil
}

func validProvenanceMode(mode ProvenanceMode) bool {
	switch mode {
	case ProvenanceOff, ProvenanceMeta, ProvenanceMetaReceipt:
		return true
	default:
		return false
	}
}

func buildClientProvenance(mode ProvenanceMode, resolution SessionResolution) (*ClientInputProvenance, string) {
	if mode == ProvenanceOff {
		return nil, ""
	}
	provenance := &ClientInputProvenance{
		Kind:            "external_user",
		OriginSessionID: resolution.SessionID,
		SourceChannel:   "acp",
		SourceTool:      "gormes_acp",
	}
	if mode != ProvenanceMetaReceipt {
		return provenance, ""
	}
	return provenance, signedSourceReceipt(resolution)
}

func signedSourceReceipt(resolution SessionResolution) string {
	body := fmt.Sprintf("[Source Receipt]\nbridge=gormes-acp\noriginSessionId=%s\ntargetSession=%s\n", resolution.SessionID, resolution.SessionKey)
	sum := sha256.Sum256([]byte(body))
	return body + "signature=sha256:" + hex.EncodeToString(sum[:]) + "\n"
}

func degradedFromError(err error, resolution SessionResolution, opts ClientOptions, mode ProvenanceMode) ClientResult {
	reason := "acp_client_error"
	switch {
	case errors.Is(err, ErrClientSessionNotFound):
		reason = "session_key_not_found"
	case errors.Is(err, ErrClientLabelNotFound):
		reason = "session_label_not_found"
	case errors.Is(err, ErrClientPermissionTimeout):
		reason = "permission_prompt_timeout"
	case errors.Is(err, ErrClientUnsupportedProvider):
		reason = "unsupported_acp_provider"
	case errors.Is(err, ErrClientMissingAuth):
		reason = "missing_auth"
	}
	return degradedClientResult(reason, err.Error(), resolution, opts, mode)
}

func degradedClientResult(reason, message string, resolution SessionResolution, opts ClientOptions, mode ProvenanceMode) ClientResult {
	prefixCWD := !opts.NoPrefixCWD
	return ClientResult{
		OK:             false,
		Code:           ClientEvidenceRowBacked,
		Message:        message,
		SessionKey:     resolution.SessionKey,
		SessionID:      resolution.SessionID,
		SessionLabel:   firstNonEmpty(resolution.SessionLabel, opts.SessionLabel),
		ProvenanceMode: mode,
		PrefixCWD:      prefixCWD,
		Evidence: ClientEvidence{
			Code:   ClientEvidenceRowBacked,
			Reason: reason,
			FallbackModes: []string{
				"--session <key>",
				"--session-label <label>",
				"--reset-session",
				"--provenance off",
			},
		},
	}
}

func metadataMatchesLabel(meta session.Metadata, label string) bool {
	label = strings.TrimSpace(label)
	if label == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(meta.Title), label) || strings.EqualFold(strings.TrimSpace(meta.SessionID), label)
}

func sessionKeyFromMetadata(meta session.Metadata) string {
	source := strings.TrimSpace(meta.Source)
	chatID := strings.TrimSpace(meta.ChatID)
	if source == "" || chatID == "" {
		return ""
	}
	return source + ":" + chatID
}

func firstIDGenerator(generators ...func() string) func() string {
	for _, generator := range generators {
		if generator != nil {
			return generator
		}
	}
	return defaultClientSessionID
}

func callIDGenerator(generator func() string) string {
	if generator == nil {
		generator = defaultClientSessionID
	}
	id := strings.TrimSpace(generator())
	if id == "" {
		return defaultClientSessionID()
	}
	return id
}

func defaultClientSessionID() string {
	return "acp-client-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
