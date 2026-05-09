package matrix

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMatrixBootstrapConfigRequiresAuthAndHomeserver(t *testing.T) {
	env := map[string]string{
		"MATRIX_HOMESERVER":          " https://matrix.example.org/ ",
		"MATRIX_ACCESS_TOKEN":        " syt_token ",
		"MATRIX_USER_ID":             " @bot:example.org ",
		"MATRIX_DEVICE_ID":           " DEV1 ",
		"MATRIX_ENCRYPTION":          "true",
		"MATRIX_FREE_RESPONSE_ROOMS": "!free:example.org, !ops:example.org ",
		"MATRIX_ALLOWED_ROOMS":       "!ops:example.org",
	}
	cfg := ResolveBootstrapConfig(nil, func(k string) string { return env[k] })
	if cfg.Homeserver != "https://matrix.example.org" || cfg.AccessToken != "syt_token" {
		t.Fatalf("ResolveBootstrapConfig auth = %+v", cfg)
	}
	if cfg.UserID != "@bot:example.org" || cfg.DeviceID != "DEV1" || !cfg.Encryption {
		t.Fatalf("ResolveBootstrapConfig identity = %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.FreeResponseRooms, []string{"!free:example.org", "!ops:example.org"}) {
		t.Fatalf("FreeResponseRooms = %#v", cfg.FreeResponseRooms)
	}
	if !reflect.DeepEqual(cfg.AllowedRooms, []string{"!ops:example.org"}) {
		t.Fatalf("AllowedRooms = %#v", cfg.AllowedRooms)
	}

	factoryCalled := false
	b := NewBootstrap(Config{}, func(Config) (MatrixClient, error) {
		factoryCalled = true
		return newFakeMatrixClient(), nil
	})
	result := b.Start(context.Background())
	if result.Ready || result.Evidence != MatrixEvidenceConfigMissing {
		t.Fatalf("Start missing config = %+v", result)
	}
	if factoryCalled {
		t.Fatal("Start constructed a Matrix client despite missing config")
	}

	missingPasswordUser := Config{Homeserver: "https://matrix.example.org", Password: "pw"}
	result = NewBootstrap(missingPasswordUser, nil).Start(context.Background())
	if result.Ready || result.Evidence != MatrixEvidenceConfigMissing {
		t.Fatalf("Start missing user for password auth = %+v", result)
	}
}

func TestMatrixBootstrapWhoamiAndPasswordLogin(t *testing.T) {
	ctx := context.Background()

	tokenClient := newFakeMatrixClient()
	tokenClient.whoamiResp = MatrixIdentity{UserID: "@bot:example.org", DeviceID: "WHOAMI_DEV"}
	tokenClient.initialSync = MatrixSyncData{NextBatch: "s0"}
	tokenBootstrap := NewBootstrap(Config{Homeserver: "https://matrix.example.org", AccessToken: "syt"}, func(Config) (MatrixClient, error) {
		return tokenClient, nil
	})
	result := tokenBootstrap.Start(ctx)
	if !result.Ready || result.Evidence != "" {
		t.Fatalf("token Start = %+v", result)
	}
	if tokenClient.whoamiCalls != 1 || tokenClient.loginCalls != 0 {
		t.Fatalf("token auth calls whoami=%d login=%d", tokenClient.whoamiCalls, tokenClient.loginCalls)
	}
	if result.UserID != "@bot:example.org" || result.DeviceID != "WHOAMI_DEV" {
		t.Fatalf("token identity = %+v", result)
	}

	passwordClient := newFakeMatrixClient()
	passwordClient.loginResp = MatrixIdentity{UserID: "@bot:example.org", DeviceID: "PW_DEV"}
	passwordClient.initialSync = MatrixSyncData{NextBatch: "s1"}
	passwordBootstrap := NewBootstrap(Config{
		Homeserver: "https://matrix.example.org",
		UserID:     "@bot:example.org",
		Password:   "pw",
		DeviceID:   "CONFIG_DEV",
	}, func(Config) (MatrixClient, error) {
		return passwordClient, nil
	})
	result = passwordBootstrap.Start(ctx)
	if !result.Ready || result.Evidence != "" {
		t.Fatalf("password Start = %+v", result)
	}
	if passwordClient.loginCalls != 1 || passwordClient.whoamiCalls != 0 {
		t.Fatalf("password auth calls whoami=%d login=%d", passwordClient.whoamiCalls, passwordClient.loginCalls)
	}
	if passwordClient.loginRequest.DeviceID != "CONFIG_DEV" {
		t.Fatalf("login DeviceID = %q, want CONFIG_DEV", passwordClient.loginRequest.DeviceID)
	}
	if result.UserID != "@bot:example.org" || result.DeviceID != "CONFIG_DEV" {
		t.Fatalf("password identity = %+v", result)
	}
}

func TestMatrixBootstrapInitialSyncRegistersHandlers(t *testing.T) {
	client := newFakeMatrixClient()
	client.whoamiResp = MatrixIdentity{UserID: "@bot:example.org", DeviceID: "DEV1"}
	client.initialSync = MatrixSyncData{
		JoinedRooms: []string{"!ops:example.org", "!dm:example.org"},
		NextBatch:   "s1234",
	}

	result := NewBootstrap(Config{Homeserver: "https://matrix.example.org", AccessToken: "syt"}, func(Config) (MatrixClient, error) {
		return client, nil
	}).Start(context.Background())
	if !result.Ready {
		t.Fatalf("Start = %+v", result)
	}

	wantHandlers := []string{MatrixEventRoomMessage, MatrixEventReaction, MatrixEventInvite}
	if !reflect.DeepEqual(client.handlers, wantHandlers) {
		t.Fatalf("handlers = %#v, want %#v", client.handlers, wantHandlers)
	}
	if !reflect.DeepEqual(result.JoinedRooms, []string{"!ops:example.org", "!dm:example.org"}) {
		t.Fatalf("JoinedRooms = %#v", result.JoinedRooms)
	}
	if result.NextBatch != "s1234" || !reflect.DeepEqual(client.nextBatches, []string{"s1234"}) {
		t.Fatalf("next batch result=%q stored=%#v", result.NextBatch, client.nextBatches)
	}

	order := strings.Join(client.order, ",")
	for _, event := range wantHandlers {
		if strings.Index(order, "register:"+event) > strings.Index(order, "sync:initial") {
			t.Fatalf("handler %s registered after initial sync: %s", event, order)
		}
	}
	if strings.Index(order, "handle_sync") < strings.Index(order, "sync:initial") {
		t.Fatalf("initial sync dispatched before sync call: %s", order)
	}
}

func TestMatrixBootstrapSyncLoopStopsOnPermanentAuthError(t *testing.T) {
	t.Run("sync error object stops", func(t *testing.T) {
		client := newStartedFakeMatrixClient(t)
		client.syncResponses = []MatrixSyncData{{ErrorMessage: "M_UNKNOWN_TOKEN expired"}}

		outcome := client.bootstrap.StepSync(context.Background())
		if !outcome.Stopped || outcome.Retry || outcome.Evidence != MatrixEvidenceAuthFailed {
			t.Fatalf("StepSync permanent auth = %+v", outcome)
		}
	})

	t.Run("transport auth error stops", func(t *testing.T) {
		client := newStartedFakeMatrixClient(t)
		client.syncErr = errors.New("403 forbidden")

		outcome := client.bootstrap.StepSync(context.Background())
		if !outcome.Stopped || outcome.Retry || outcome.Evidence != MatrixEvidenceAuthFailed {
			t.Fatalf("StepSync forbidden = %+v", outcome)
		}
	})

	t.Run("transient sync error retries without sleep", func(t *testing.T) {
		client := newStartedFakeMatrixClient(t)
		client.syncErr = errors.New("temporary network drain")

		outcome := client.bootstrap.StepSync(context.Background())
		if outcome.Stopped || !outcome.Retry || outcome.Evidence != MatrixEvidenceSyncUnavailable {
			t.Fatalf("StepSync transient = %+v", outcome)
		}
	})
}

func TestMatrixBootstrapMediaAndE2EEHooksAreInjectable(t *testing.T) {
	b := NewBootstrap(Config{Homeserver: "https://matrix.example.org", AccessToken: "syt"}, nil)
	status := b.HookStatus()
	if status.MediaEvidence != MatrixEvidenceTransportUnavailable || status.E2EEEvidence != MatrixEvidenceE2EEUnavailable {
		t.Fatalf("default HookStatus = %+v", status)
	}

	mediaCalled := false
	e2eeCalled := false
	client := newFakeMatrixClient()
	client.whoamiResp = MatrixIdentity{UserID: "@bot:example.org", DeviceID: "DEV1"}
	client.initialSync = MatrixSyncData{NextBatch: "s0"}
	b = NewBootstrap(Config{Homeserver: "https://matrix.example.org", AccessToken: "syt", Encryption: true}, func(Config) (MatrixClient, error) {
		return client, nil
	}, WithBootstrapHooks(BootstrapHooks{
		Media: MatrixMediaHookFunc(func(context.Context, MatrixMediaUpload) (MatrixMediaResult, error) {
			mediaCalled = true
			return MatrixMediaResult{URI: "mxc://example.org/file"}, nil
		}),
		E2EE: MatrixE2EEHookFunc(func(context.Context, MatrixClient, Config) MatrixHookEvidence {
			e2eeCalled = true
			return MatrixHookEvidence{}
		}),
	}))

	if result := b.Start(context.Background()); !result.Ready {
		t.Fatalf("Start with hooks = %+v", result)
	}
	upload, evidence := b.UploadMedia(context.Background(), MatrixMediaUpload{Name: "a.txt", ContentType: "text/plain"})
	if evidence.Evidence != "" || upload.URI != "mxc://example.org/file" || !mediaCalled || !e2eeCalled {
		t.Fatalf("hook results upload=%+v evidence=%+v mediaCalled=%v e2eeCalled=%v", upload, evidence, mediaCalled, e2eeCalled)
	}
}

type fakeMatrixClient struct {
	whoamiResp    MatrixIdentity
	loginResp     MatrixIdentity
	initialSync   MatrixSyncData
	syncResponses []MatrixSyncData
	syncErr       error

	bootstrap    *Bootstrap
	whoamiCalls  int
	loginCalls   int
	loginRequest MatrixLoginRequest
	handlers     []string
	nextBatches  []string
	order        []string
}

func newFakeMatrixClient() *fakeMatrixClient {
	return &fakeMatrixClient{}
}

func newStartedFakeMatrixClient(t *testing.T) *fakeMatrixClient {
	t.Helper()
	client := newFakeMatrixClient()
	client.whoamiResp = MatrixIdentity{UserID: "@bot:example.org", DeviceID: "DEV1"}
	client.initialSync = MatrixSyncData{NextBatch: "s0"}
	b := NewBootstrap(Config{Homeserver: "https://matrix.example.org", AccessToken: "syt"}, func(Config) (MatrixClient, error) {
		return client, nil
	})
	result := b.Start(context.Background())
	if !result.Ready {
		t.Fatalf("Start = %+v", result)
	}
	client.bootstrap = b
	return client
}

func (f *fakeMatrixClient) Whoami(context.Context) (MatrixIdentity, error) {
	f.whoamiCalls++
	f.order = append(f.order, "whoami")
	return f.whoamiResp, nil
}

func (f *fakeMatrixClient) Login(_ context.Context, req MatrixLoginRequest) (MatrixIdentity, error) {
	f.loginCalls++
	f.loginRequest = req
	f.order = append(f.order, "login")
	return f.loginResp, nil
}

func (f *fakeMatrixClient) RegisterHandler(eventType string, _ MatrixEventHandler) {
	f.handlers = append(f.handlers, eventType)
	f.order = append(f.order, "register:"+eventType)
}

func (f *fakeMatrixClient) Sync(_ context.Context, req MatrixSyncRequest) (MatrixSyncData, error) {
	if req.Initial {
		f.order = append(f.order, "sync:initial")
		return f.initialSync, nil
	}
	f.order = append(f.order, "sync:loop")
	if f.syncErr != nil {
		return MatrixSyncData{}, f.syncErr
	}
	if len(f.syncResponses) == 0 {
		return MatrixSyncData{}, nil
	}
	next := f.syncResponses[0]
	f.syncResponses = f.syncResponses[1:]
	return next, nil
}

func (f *fakeMatrixClient) PutNextBatch(_ context.Context, token string) error {
	f.nextBatches = append(f.nextBatches, token)
	f.order = append(f.order, "put_next_batch:"+token)
	return nil
}

func (f *fakeMatrixClient) HandleSync(context.Context, MatrixSyncData) error {
	f.order = append(f.order, "handle_sync")
	return nil
}
