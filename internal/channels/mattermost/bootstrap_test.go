package mattermost

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestMattermostBootstrapConfigAndAuth(t *testing.T) {
	env := map[string]string{
		"MATTERMOST_URL":                    " https://mm.example.com/ ",
		"MATTERMOST_TOKEN":                  " mm-token-secret ",
		"MATTERMOST_HOME_CHANNEL":           " town-square ",
		"MATTERMOST_HOME_CHANNEL_NAME":      " Town Square ",
		"MATTERMOST_REPLY_MODE":             " thread ",
		"MATTERMOST_ALLOWED_CHANNELS":       "town-square, ops",
		"MATTERMOST_FREE_RESPONSE_CHANNELS": "lounge",
		"MATTERMOST_REQUIRE_MENTION":        "false",
	}

	cfg := ResolveBootstrapConfig(nil, func(k string) string { return env[k] })
	if cfg.BaseURL != "https://mm.example.com" || cfg.Token != "mm-token-secret" {
		t.Fatalf("ResolveBootstrapConfig auth = %+v", cfg)
	}
	if cfg.HomeChannel != "town-square" || cfg.HomeChannelName != "Town Square" || cfg.ReplyMode != "thread" {
		t.Fatalf("ResolveBootstrapConfig home/reply = %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.AllowedChannels, []string{"town-square", "ops"}) {
		t.Fatalf("AllowedChannels = %#v", cfg.AllowedChannels)
	}
	if !reflect.DeepEqual(cfg.FreeResponseChannels, []string{"lounge"}) {
		t.Fatalf("FreeResponseChannels = %#v", cfg.FreeResponseChannels)
	}
	if cfg.RequireMention {
		t.Fatalf("RequireMention = true, want false from MATTERMOST_REQUIRE_MENTION=false")
	}

	factoryCalled := false
	b := NewBootstrap(Config{}, func(Config) (Transport, error) {
		factoryCalled = true
		return newFakeMattermostTransport(), nil
	})
	result := b.Start(context.Background())
	if result.Ready || result.Evidence != MattermostEvidenceConfigMissing {
		t.Fatalf("Start missing config = %+v", result)
	}
	if factoryCalled {
		t.Fatal("Start constructed a transport despite missing config")
	}

	transport := newFakeMattermostTransport()
	transport.getResponses["users/me"] = mattermostRESTResponse{
		Status: 200,
		JSON:   map[string]any{"id": "bot-user", "username": "gormes-bot"},
	}
	result = NewBootstrap(cfg, func(Config) (Transport, error) {
		return transport, nil
	}).Start(context.Background())
	if !result.Ready || result.Evidence != "" {
		t.Fatalf("Start auth = %+v", result)
	}
	if result.BotUserID != "bot-user" || result.BotUsername != "gormes-bot" {
		t.Fatalf("auth identity = %+v", result)
	}
	if got := transport.getPaths; !reflect.DeepEqual(got, []string{"users/me"}) {
		t.Fatalf("GET paths = %#v, want users/me probe", got)
	}

	transport = newFakeMattermostTransport()
	transport.getResponses["users/me"] = mattermostRESTResponse{Status: 401, Body: "bad token mm-token-secret"}
	result = NewBootstrap(cfg, func(Config) (Transport, error) {
		return transport, nil
	}).Start(context.Background())
	if result.Ready || result.Evidence != MattermostEvidenceAuthFailed {
		t.Fatalf("Start auth failure = %+v", result)
	}
	if result.Error == "" || strings.Contains(result.Error, "mm-token-secret") {
		t.Fatalf("auth failure error should be sanitized, got %q", result.Error)
	}
}

func TestMattermostRESTHelpersSanitizeErrors(t *testing.T) {
	ctx := context.Background()
	cfg := Config{BaseURL: "https://mm.example.com", Token: "mm-token-secret"}
	transport := newFakeMattermostTransport()
	transport.getResponses["users/me"] = mattermostRESTResponse{
		Status: 200,
		JSON:   map[string]any{"id": "bot-user", "username": "gormes-bot"},
	}
	b := NewBootstrap(cfg, func(Config) (Transport, error) { return transport, nil })
	if result := b.Start(ctx); !result.Ready {
		t.Fatalf("Start = %+v", result)
	}

	transport.postResponses["posts"] = mattermostRESTResponse{
		Status: 500,
		Body:   "token mm-token-secret " + strings.Repeat("<html>oversized internal body</html>", 20),
	}
	_, evidence := b.APIPost(ctx, "posts", map[string]any{"message": "hello"})
	if evidence.Evidence != MattermostEvidenceTransportUnavailable || evidence.Status != 500 {
		t.Fatalf("APIPost evidence = %+v", evidence)
	}
	if strings.Contains(evidence.Error, "mm-token-secret") {
		t.Fatalf("APIPost leaked bearer token in %q", evidence.Error)
	}
	if len(evidence.Error) > 260 || !strings.Contains(evidence.Error, "...") {
		t.Fatalf("APIPost should truncate oversized body; got len=%d %q", len(evidence.Error), evidence.Error)
	}
}

func TestMattermostWebsocketFeedsSeam(t *testing.T) {
	ctx := context.Background()
	transport := newFakeMattermostTransport()
	transport.getResponses["users/me"] = mattermostRESTResponse{
		Status: 200,
		JSON:   map[string]any{"id": "bot-123", "username": "gormes-bot"},
	}
	b := NewBootstrap(Config{BaseURL: "https://mm.example.com", Token: "mm-token-secret", RequireMention: true}, func(Config) (Transport, error) {
		return transport, nil
	})
	if result := b.Start(ctx); !result.Ready {
		t.Fatalf("Start = %+v", result)
	}

	raw := postedEventPayload("post-1", "town-square", "O", "user-1", "@bot-123 hello from Mattermost", "thread-1")
	ev, ok := b.HandleWebsocketEvent(raw)
	if !ok {
		t.Fatal("HandleWebsocketEvent ok=false, want posted event to enter seam")
	}
	if ev.Platform != "mattermost" || ev.Kind != gateway.EventSubmit || ev.Text != "hello from Mattermost" {
		t.Fatalf("websocket event = %+v", ev)
	}
	if ev.ThreadID != "thread-1" {
		t.Fatalf("ThreadID = %q, want thread-1", ev.ThreadID)
	}

	if _, ok := b.HandleWebsocketEvent(`{"event":"typing","data":{}}`); ok {
		t.Fatal("typing websocket event entered seam, want ignored")
	}
	if _, ok := b.HandleWebsocketEvent(`{"event":"posted","data":{"post":"not-json"}}`); ok {
		t.Fatal("malformed posted websocket event entered seam, want ignored")
	}
}

func TestMattermostReconnectPolicyUsesFakeClock(t *testing.T) {
	b := NewBootstrap(Config{BaseURL: "https://mm.example.com", Token: "mm-token-secret"}, nil)
	outcome := b.StepReconnect(errors.New("temporary websocket drain mm-token-secret"))
	if !outcome.Retry || outcome.Attempt != 1 || outcome.Evidence != MattermostEvidenceWSUnavailable {
		t.Fatalf("StepReconnect = %+v", outcome)
	}
	if outcome.Delay != 2*time.Second {
		t.Fatalf("Delay = %v, want 2s first retry evidence without sleeping", outcome.Delay)
	}
	if strings.Contains(outcome.Error, "mm-token-secret") {
		t.Fatalf("StepReconnect leaked token in %q", outcome.Error)
	}
}

func TestMattermostUploadEditSendShapesRequests(t *testing.T) {
	ctx := context.Background()
	cfg := Config{BaseURL: "https://mm.example.com", Token: "mm-token-secret", ReplyMode: "thread"}
	transport := newFakeMattermostTransport()
	transport.getResponses["users/me"] = mattermostRESTResponse{
		Status: 200,
		JSON:   map[string]any{"id": "bot-user", "username": "gormes-bot"},
	}
	transport.postResponses["posts"] = mattermostRESTResponse{Status: 200, JSON: map[string]any{"id": "post-1"}}
	transport.putResponses["posts/post-1/patch"] = mattermostRESTResponse{Status: 200, JSON: map[string]any{"id": "post-1"}}
	transport.uploadResponses = append(transport.uploadResponses, mattermostRESTResponse{
		Status: 200,
		JSON: map[string]any{
			"file_infos": []any{map[string]any{"id": "file-1"}},
		},
	})

	b := NewBootstrap(cfg, func(Config) (Transport, error) { return transport, nil })
	if result := b.Start(ctx); !result.Ready {
		t.Fatalf("Start = %+v", result)
	}

	send := b.Send(ctx, "chan-1", "**hello**", "root-1")
	if !send.Success || send.MessageID != "post-1" {
		t.Fatalf("Send = %+v", send)
	}
	if got := transport.postCalls[0].payload; got["channel_id"] != "chan-1" || got["message"] != "**hello**" || got["root_id"] != "root-1" {
		t.Fatalf("send payload = %#v", got)
	}

	edit := b.Edit(ctx, "post-1", "edited")
	if !edit.Success || edit.MessageID != "post-1" {
		t.Fatalf("Edit = %+v", edit)
	}
	if got := transport.putCalls[0].payload; got["message"] != "edited" {
		t.Fatalf("edit payload = %#v", got)
	}

	uploaded := b.UploadAndPost(ctx, MattermostUpload{
		ChannelID:   "chan-1",
		Data:        []byte("file body"),
		Filename:    "report.txt",
		ContentType: "text/plain",
	}, "caption", "root-2")
	if !uploaded.Success {
		t.Fatalf("UploadAndPost = %+v", uploaded)
	}
	if got := transport.uploadCalls[0].upload; got.ChannelID != "chan-1" || got.Filename != "report.txt" || got.ContentType != "text/plain" {
		t.Fatalf("upload request = %+v", got)
	}
	if auth := transport.uploadCalls[0].headers["Authorization"]; auth != "Bearer mm-token-secret" {
		t.Fatalf("upload auth header = %q", auth)
	}
	filePost := transport.postCalls[len(transport.postCalls)-1].payload
	if filePost["channel_id"] != "chan-1" || filePost["message"] != "caption" || filePost["root_id"] != "root-2" {
		t.Fatalf("file post payload = %#v", filePost)
	}
	if !reflect.DeepEqual(filePost["file_ids"], []string{"file-1"}) {
		t.Fatalf("file_ids = %#v, want file-1", filePost["file_ids"])
	}
}

type fakeMattermostTransport struct {
	getResponses    map[string]mattermostRESTResponse
	postResponses   map[string]mattermostRESTResponse
	putResponses    map[string]mattermostRESTResponse
	uploadResponses []mattermostRESTResponse

	getPaths    []string
	postCalls   []mattermostPostCall
	putCalls    []mattermostPostCall
	uploadCalls []mattermostUploadCall
}

type mattermostPostCall struct {
	path    string
	payload map[string]any
	headers map[string]string
}

type mattermostUploadCall struct {
	upload  MattermostUpload
	headers map[string]string
}

func newFakeMattermostTransport() *fakeMattermostTransport {
	return &fakeMattermostTransport{
		getResponses:  map[string]mattermostRESTResponse{},
		postResponses: map[string]mattermostRESTResponse{},
		putResponses:  map[string]mattermostRESTResponse{},
	}
}

func (f *fakeMattermostTransport) Get(_ context.Context, path string, _ map[string]string) (mattermostRESTResponse, error) {
	f.getPaths = append(f.getPaths, path)
	if resp, ok := f.getResponses[path]; ok {
		return resp, nil
	}
	return mattermostRESTResponse{Status: 404, Body: "not found"}, nil
}

func (f *fakeMattermostTransport) Post(_ context.Context, path string, payload map[string]any, headers map[string]string) (mattermostRESTResponse, error) {
	f.postCalls = append(f.postCalls, mattermostPostCall{path: path, payload: payload, headers: headers})
	if resp, ok := f.postResponses[path]; ok {
		return resp, nil
	}
	return mattermostRESTResponse{Status: 404, Body: "not found"}, nil
}

func (f *fakeMattermostTransport) Put(_ context.Context, path string, payload map[string]any, headers map[string]string) (mattermostRESTResponse, error) {
	f.putCalls = append(f.putCalls, mattermostPostCall{path: path, payload: payload, headers: headers})
	if resp, ok := f.putResponses[path]; ok {
		return resp, nil
	}
	return mattermostRESTResponse{Status: 404, Body: "not found"}, nil
}

func (f *fakeMattermostTransport) UploadFile(_ context.Context, upload MattermostUpload, headers map[string]string) (mattermostRESTResponse, error) {
	f.uploadCalls = append(f.uploadCalls, mattermostUploadCall{upload: upload, headers: headers})
	if len(f.uploadResponses) == 0 {
		return mattermostRESTResponse{Status: 404, Body: "not found"}, nil
	}
	resp := f.uploadResponses[0]
	f.uploadResponses = f.uploadResponses[1:]
	return resp, nil
}
