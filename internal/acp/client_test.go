package acp

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestClientConnectsWithSessionKeyAndProvenanceReceipt(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "agent:main:main", "sess-main"); err != nil {
		t.Fatalf("Put session: %v", err)
	}

	connector := &recordingClientConnector{}
	client := ClientBridge{
		Resolver:  NewSessionMapResolver(smap),
		Connector: connector,
	}
	result, err := client.Run(ctx, ClientOptions{
		SessionKey:      "agent:main:main",
		RequireExisting: true,
		ProvenanceMode:  ProvenanceMetaReceipt,
		CWD:             "/repo",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.OK {
		t.Fatalf("OK = false, result=%+v", result)
	}
	if result.Code != ClientEvidenceConnected {
		t.Fatalf("Code = %q, want %q", result.Code, ClientEvidenceConnected)
	}
	if result.SessionKey != "agent:main:main" || result.SessionID != "sess-main" {
		t.Fatalf("session resolution = key %q id %q, want agent:main:main/sess-main", result.SessionKey, result.SessionID)
	}
	if connector.calls != 1 {
		t.Fatalf("connector calls = %d, want 1", connector.calls)
	}
	req := connector.last
	if req.SessionKey != "agent:main:main" || req.SessionID != "sess-main" {
		t.Fatalf("connector session = key %q id %q", req.SessionKey, req.SessionID)
	}
	if req.Provenance == nil {
		t.Fatalf("Provenance = nil, want metadata")
	}
	if req.Provenance.Kind != "external_user" || req.Provenance.SourceChannel != "acp" || req.Provenance.SourceTool != "gormes_acp" {
		t.Fatalf("Provenance = %+v, want Gormes ACP external user metadata", req.Provenance)
	}
	for _, want := range []string{
		"[Source Receipt]",
		"bridge=gormes-acp",
		"originSessionId=sess-main",
		"targetSession=agent:main:main",
		"signature=sha256:",
	} {
		if !strings.Contains(req.Receipt, want) {
			t.Fatalf("receipt missing %q:\n%s", want, req.Receipt)
		}
	}
	if !req.PrefixCWD || req.CWD != "/repo" {
		t.Fatalf("cwd prefix = %v cwd=%q, want enabled /repo", req.PrefixCWD, req.CWD)
	}
}

func TestClientRequireExistingFailsMissingSession(t *testing.T) {
	ctx := context.Background()
	client := ClientBridge{
		Resolver:  NewSessionMapResolver(session.NewMemMap()),
		Connector: &recordingClientConnector{},
	}

	result, err := client.Run(ctx, ClientOptions{
		SessionKey:      "agent:missing:main",
		RequireExisting: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.OK {
		t.Fatalf("OK = true, want degraded result")
	}
	if result.Code != ClientEvidenceRowBacked || result.Evidence.Code != ClientEvidenceRowBacked {
		t.Fatalf("evidence = result code %q evidence %+v, want row-backed", result.Code, result.Evidence)
	}
	if result.Evidence.Reason != "session_key_not_found" {
		t.Fatalf("reason = %q, want session_key_not_found", result.Evidence.Reason)
	}
	if !strings.Contains(result.Message, "agent:missing:main") {
		t.Fatalf("message missing session key: %q", result.Message)
	}
}

func TestClientResetSessionReinitializesSessionKey(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "agent:main:main", "sess-old"); err != nil {
		t.Fatalf("Put session: %v", err)
	}

	client := ClientBridge{
		Resolver:  NewSessionMapResolver(smap),
		Connector: &recordingClientConnector{},
		IDGenerator: func() string {
			return "sess-new"
		},
	}
	result, err := client.Run(ctx, ClientOptions{
		SessionKey:      "agent:main:main",
		RequireExisting: true,
		ResetSession:    true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.OK || !result.Reset {
		t.Fatalf("result = %+v, want ok reset", result)
	}
	if result.SessionID != "sess-new" {
		t.Fatalf("SessionID = %q, want sess-new", result.SessionID)
	}
	stored, err := smap.Get(ctx, "agent:main:main")
	if err != nil {
		t.Fatalf("Get session: %v", err)
	}
	if stored != "sess-new" {
		t.Fatalf("stored session = %q, want sess-new", stored)
	}
}

func TestClientLabelResolutionAndNoPrefixCWD(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "agent:main:main", "sess-main"); err != nil {
		t.Fatalf("Put session: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID: "sess-main",
		Source:    "agent",
		ChatID:    "main:main",
		Title:     "Support Inbox",
		UpdatedAt: 10,
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	connector := &recordingClientConnector{}
	client := ClientBridge{
		Resolver:  NewSessionMapResolver(smap),
		Connector: connector,
	}
	result, err := client.Run(ctx, ClientOptions{
		SessionLabel: "support inbox",
		NoPrefixCWD:  true,
		CWD:          "/repo",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.OK {
		t.Fatalf("OK = false, result=%+v", result)
	}
	if result.SessionKey != "agent:main:main" || result.SessionID != "sess-main" || result.SessionLabel != "support inbox" {
		t.Fatalf("label resolution = %+v, want agent:main:main/sess-main with label", result)
	}
	if connector.last.PrefixCWD {
		t.Fatalf("PrefixCWD = true, want false for --no-prefix-cwd")
	}
}

func TestClientPermissionTimeoutReturnsRowBackedEvidence(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "agent:main:main", "sess-main"); err != nil {
		t.Fatalf("Put session: %v", err)
	}

	result, err := (ClientBridge{
		Resolver: NewSessionMapResolver(smap),
		Connector: &recordingClientConnector{
			err: ErrClientPermissionTimeout,
		},
	}).Run(ctx, ClientOptions{
		SessionKey:      "agent:main:main",
		RequireExisting: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.OK {
		t.Fatalf("OK = true, want degraded result")
	}
	if result.Evidence.Code != ClientEvidenceRowBacked || result.Evidence.Reason != "permission_prompt_timeout" {
		t.Fatalf("evidence = %+v, want permission timeout row-backed", result.Evidence)
	}
	if len(result.Evidence.FallbackModes) == 0 {
		t.Fatalf("FallbackModes empty, want available fallback modes")
	}
}

type recordingClientConnector struct {
	calls int
	last  ClientConnectRequest
	err   error
}

func (c *recordingClientConnector) Connect(_ context.Context, req ClientConnectRequest) (ClientConnectResult, error) {
	c.calls++
	c.last = req
	if c.err != nil {
		return ClientConnectResult{}, c.err
	}
	return ClientConnectResult{Connected: true}, nil
}
