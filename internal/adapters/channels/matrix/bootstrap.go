package matrix

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
)

const (
	MatrixEventRoomMessage = "m.room.message"
	MatrixEventReaction    = "m.reaction"
	MatrixEventInvite      = "internal.invite"
)

// MatrixClient is the fakeable subset of Hermes' mautrix client bootstrap used
// by the Go boundary. Live SDK binding remains a later row.
type MatrixClient interface {
	Whoami(context.Context) (MatrixIdentity, error)
	Login(context.Context, MatrixLoginRequest) (MatrixIdentity, error)
	RegisterHandler(eventType string, handler MatrixEventHandler)
	Sync(context.Context, MatrixSyncRequest) (MatrixSyncData, error)
	PutNextBatch(context.Context, string) error
	HandleSync(context.Context, MatrixSyncData) error
}

type MatrixEventHandler func(context.Context, MatrixEvent) error

type MatrixEvent struct{}

type MatrixIdentity struct {
	UserID   string
	DeviceID string
}

type MatrixLoginRequest struct {
	UserID   string
	Password string
	DeviceID string
}

type MatrixSyncRequest struct {
	Initial   bool
	Since     string
	TimeoutMS int
	FullState bool
}

type MatrixSyncData struct {
	JoinedRooms  []string
	NextBatch    string
	ErrorMessage string
}

type MatrixClientFactory func(Config) (MatrixClient, error)

type BootstrapOption func(*Bootstrap)

type Bootstrap struct {
	cfg     Config
	factory MatrixClientFactory
	hooks   BootstrapHooks

	client      MatrixClient
	userID      string
	deviceID    string
	joinedRooms []string
	nextBatch   string
}

type BootstrapResult struct {
	Ready       bool
	Evidence    Evidence
	Error       string
	UserID      string
	DeviceID    string
	JoinedRooms []string
	NextBatch   string
}

type MatrixSyncOutcome struct {
	Stopped     bool
	Retry       bool
	Evidence    Evidence
	Error       string
	JoinedRooms []string
	NextBatch   string
}

type BootstrapHooks struct {
	Media MatrixMediaHook
	E2EE  MatrixE2EEHook
}

type MatrixHookStatus struct {
	MediaEvidence Evidence
	E2EEEvidence  Evidence
}

type MatrixHookEvidence struct {
	Evidence Evidence
	Error    string
}

type MatrixE2EEBootstrap struct {
	Client   MatrixClient
	Config   Config
	UserID   string
	DeviceID string
}

type MatrixMediaUpload struct {
	Name        string
	ContentType string
}

type MatrixMediaResult struct {
	URI string
}

type MatrixMediaHook interface {
	UploadMatrixMedia(context.Context, MatrixMediaUpload) (MatrixMediaResult, error)
}

type MatrixMediaHookFunc func(context.Context, MatrixMediaUpload) (MatrixMediaResult, error)

func (f MatrixMediaHookFunc) UploadMatrixMedia(ctx context.Context, upload MatrixMediaUpload) (MatrixMediaResult, error) {
	return f(ctx, upload)
}

type MatrixE2EEHook interface {
	BootstrapMatrixE2EE(context.Context, MatrixE2EEBootstrap) MatrixHookEvidence
}

type MatrixE2EEHookFunc func(context.Context, MatrixE2EEBootstrap) MatrixHookEvidence

func (f MatrixE2EEHookFunc) BootstrapMatrixE2EE(ctx context.Context, input MatrixE2EEBootstrap) MatrixHookEvidence {
	return f(ctx, input)
}

type MatrixCryptoStore interface {
	PutDeviceID(context.Context, string) error
}

func BindMatrixCryptoStoreDeviceID(ctx context.Context, store MatrixCryptoStore, deviceID string) MatrixHookEvidence {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return MatrixHookEvidence{
			Evidence: MatrixEvidenceE2EEUnavailable,
			Error:    "Matrix E2EE requires a resolved device_id before encrypted sync can start",
		}
	}
	if store == nil {
		return MatrixHookEvidence{
			Evidence: MatrixEvidenceE2EEUnavailable,
			Error:    "Matrix E2EE crypto store is not configured",
		}
	}
	if err := store.PutDeviceID(ctx, deviceID); err != nil {
		return MatrixHookEvidence{
			Evidence: MatrixEvidenceE2EEUnavailable,
			Error:    "Matrix E2EE crypto store device_id binding failed: " + sanitizeMatrixError(err),
		}
	}
	return MatrixHookEvidence{}
}

func WithBootstrapHooks(hooks BootstrapHooks) BootstrapOption {
	return func(b *Bootstrap) {
		b.hooks = hooks
	}
}

func NewBootstrap(cfg Config, factory MatrixClientFactory, opts ...BootstrapOption) *Bootstrap {
	b := &Bootstrap{
		cfg:     normalizeMatrixConfig(cfg),
		factory: factory,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func ResolveBootstrapConfig(extra map[string]string, getenv func(string) string) Config {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	get := func(key, envKey string) string {
		if extra != nil {
			if v := strings.TrimSpace(extra[key]); v != "" {
				return v
			}
		}
		return strings.TrimSpace(getenv(envKey))
	}
	return normalizeMatrixConfig(Config{
		Homeserver:        get("homeserver", "MATRIX_HOMESERVER"),
		AccessToken:       get("access_token", "MATRIX_ACCESS_TOKEN"),
		UserID:            get("user_id", "MATRIX_USER_ID"),
		Password:          get("password", "MATRIX_PASSWORD"),
		DeviceID:          get("device_id", "MATRIX_DEVICE_ID"),
		Encryption:        parseMatrixBool(get("encryption", "MATRIX_ENCRYPTION"), false),
		AutoThread:        parseMatrixBool(get("auto_thread", "MATRIX_AUTO_THREAD"), true),
		RequireMention:    parseMatrixBool(get("require_mention", "MATRIX_REQUIRE_MENTION"), true),
		FreeResponseRooms: splitMatrixList(get("free_response_rooms", "MATRIX_FREE_RESPONSE_ROOMS")),
		AllowedRooms:      splitMatrixList(get("allowed_rooms", "MATRIX_ALLOWED_ROOMS")),
	})
}

func (b *Bootstrap) Start(ctx context.Context) BootstrapResult {
	if !b.cfg.IsAvailable() {
		return BootstrapResult{Evidence: MatrixEvidenceConfigMissing, Error: "Matrix homeserver and access token or user/password are required"}
	}
	if b.factory == nil {
		return BootstrapResult{Evidence: MatrixEvidenceTransportUnavailable, Error: "Matrix client factory is not configured"}
	}
	client, err := b.factory(b.cfg)
	if err != nil {
		return BootstrapResult{Evidence: MatrixEvidenceTransportUnavailable, Error: sanitizeMatrixError(err)}
	}
	b.client = client

	identity, err := b.authenticate(ctx, client)
	if err != nil {
		return BootstrapResult{Evidence: MatrixEvidenceAuthFailed, Error: sanitizeMatrixError(err)}
	}
	b.userID = firstMatrixNonEmpty(identity.UserID, b.cfg.UserID)
	b.deviceID = firstMatrixNonEmpty(b.cfg.DeviceID, identity.DeviceID)

	if b.cfg.Encryption {
		if b.deviceID == "" {
			return BootstrapResult{Evidence: MatrixEvidenceE2EEUnavailable, Error: "Matrix E2EE requires a resolved device_id before encrypted sync can start", UserID: b.userID}
		}
		if b.hooks.E2EE == nil {
			return BootstrapResult{Evidence: MatrixEvidenceE2EEUnavailable, Error: "Matrix E2EE hook is not configured"}
		}
		if evidence := b.hooks.E2EE.BootstrapMatrixE2EE(ctx, MatrixE2EEBootstrap{
			Client:   client,
			Config:   b.cfg,
			UserID:   b.userID,
			DeviceID: b.deviceID,
		}); evidence.Evidence != "" {
			return BootstrapResult{Evidence: evidence.Evidence, Error: evidence.Error}
		}
	}

	b.registerHandlers(client)
	syncData, err := client.Sync(ctx, MatrixSyncRequest{Initial: true, TimeoutMS: 10000, FullState: true})
	if err != nil {
		return BootstrapResult{Evidence: MatrixEvidenceSyncUnavailable, Error: sanitizeMatrixError(err), UserID: b.userID, DeviceID: b.deviceID}
	}
	if outcome := b.applySyncData(ctx, syncData); outcome.Evidence != "" && outcome.Stopped {
		return BootstrapResult{Evidence: outcome.Evidence, Error: outcome.Error, UserID: b.userID, DeviceID: b.deviceID}
	}

	return BootstrapResult{
		Ready:       true,
		UserID:      b.userID,
		DeviceID:    b.deviceID,
		JoinedRooms: append([]string(nil), b.joinedRooms...),
		NextBatch:   b.nextBatch,
	}
}

func (b *Bootstrap) StepSync(ctx context.Context) MatrixSyncOutcome {
	if b.client == nil {
		return MatrixSyncOutcome{Stopped: true, Evidence: MatrixEvidenceTransportUnavailable, Error: "Matrix bootstrap has not started"}
	}
	syncData, err := b.client.Sync(ctx, MatrixSyncRequest{Since: b.nextBatch, TimeoutMS: 30000})
	if err != nil {
		if matrixPermanentAuthError(err.Error()) {
			return MatrixSyncOutcome{Stopped: true, Evidence: MatrixEvidenceAuthFailed, Error: sanitizeMatrixError(err)}
		}
		return MatrixSyncOutcome{Retry: true, Evidence: MatrixEvidenceSyncUnavailable, Error: sanitizeMatrixError(err)}
	}
	if matrixPermanentAuthError(syncData.ErrorMessage) {
		return MatrixSyncOutcome{Stopped: true, Evidence: MatrixEvidenceAuthFailed, Error: sanitizeMatrixText(syncData.ErrorMessage)}
	}
	return b.applySyncData(ctx, syncData)
}

func (b *Bootstrap) HookStatus() MatrixHookStatus {
	status := MatrixHookStatus{}
	if b.hooks.Media == nil {
		status.MediaEvidence = MatrixEvidenceTransportUnavailable
	}
	if b.hooks.E2EE == nil {
		status.E2EEEvidence = MatrixEvidenceE2EEUnavailable
	}
	return status
}

func (b *Bootstrap) UploadMedia(ctx context.Context, upload MatrixMediaUpload) (MatrixMediaResult, MatrixHookEvidence) {
	if b.hooks.Media == nil {
		return MatrixMediaResult{}, MatrixHookEvidence{Evidence: MatrixEvidenceTransportUnavailable, Error: "Matrix media hook is not configured"}
	}
	result, err := b.hooks.Media.UploadMatrixMedia(ctx, upload)
	if err != nil {
		return MatrixMediaResult{}, MatrixHookEvidence{Evidence: MatrixEvidenceTransportUnavailable, Error: sanitizeMatrixError(err)}
	}
	return result, MatrixHookEvidence{}
}

func (b *Bootstrap) authenticate(ctx context.Context, client MatrixClient) (MatrixIdentity, error) {
	if b.cfg.AccessToken != "" {
		return client.Whoami(ctx)
	}
	return client.Login(ctx, MatrixLoginRequest{
		UserID:   b.cfg.UserID,
		Password: b.cfg.Password,
		DeviceID: b.cfg.DeviceID,
	})
}

func (b *Bootstrap) registerHandlers(client MatrixClient) {
	client.RegisterHandler(MatrixEventRoomMessage, nil)
	client.RegisterHandler(MatrixEventReaction, nil)
	client.RegisterHandler(MatrixEventInvite, nil)
}

func (b *Bootstrap) applySyncData(ctx context.Context, data MatrixSyncData) MatrixSyncOutcome {
	if matrixPermanentAuthError(data.ErrorMessage) {
		return MatrixSyncOutcome{Stopped: true, Evidence: MatrixEvidenceAuthFailed, Error: sanitizeMatrixText(data.ErrorMessage)}
	}
	if len(data.JoinedRooms) > 0 {
		b.joinedRooms = append([]string(nil), data.JoinedRooms...)
	}
	if data.NextBatch != "" {
		b.nextBatch = data.NextBatch
		_ = b.client.PutNextBatch(ctx, data.NextBatch)
	}
	if err := b.client.HandleSync(ctx, data); err != nil {
		return MatrixSyncOutcome{Retry: true, Evidence: MatrixEvidenceSyncUnavailable, Error: sanitizeMatrixError(err)}
	}
	return MatrixSyncOutcome{
		JoinedRooms: append([]string(nil), b.joinedRooms...),
		NextBatch:   b.nextBatch,
	}
}

func normalizeMatrixConfig(cfg Config) Config {
	cfg.Homeserver = trimMatrixHomeserver(cfg.Homeserver)
	cfg.AccessToken = strings.TrimSpace(cfg.AccessToken)
	cfg.UserID = strings.TrimSpace(cfg.UserID)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.DeviceID = strings.TrimSpace(cfg.DeviceID)
	if cfg.FreeResponseRooms != nil {
		cfg.FreeResponseRooms = compactMatrixStrings(cfg.FreeResponseRooms)
	}
	if cfg.AllowedRooms != nil {
		cfg.AllowedRooms = compactMatrixStrings(cfg.AllowedRooms)
	}
	return cfg
}

func trimMatrixHomeserver(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func parseMatrixBool(raw string, def bool) bool { return channelutil.ParseBoolDefault(raw, def) }

func splitMatrixList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return compactMatrixStrings(strings.Split(raw, ","))
}

func compactMatrixStrings(values []string) []string { return channelutil.CompactStrings(values) }

func firstMatrixNonEmpty(values ...string) string { return channelutil.FirstNonEmpty(values...) }

func matrixPermanentAuthError(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "m_unknown_token") ||
		strings.Contains(lower, "unknown_token") ||
		strings.Contains(lower, "401") ||
		strings.Contains(lower, "403") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden")
}

func sanitizeMatrixError(err error) string {
	if err == nil {
		return ""
	}
	return sanitizeMatrixText(err.Error())
}

func sanitizeMatrixText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	for _, secret := range []string{"MATRIX_ACCESS_TOKEN", "MATRIX_PASSWORD"} {
		text = strings.ReplaceAll(text, secret, "[redacted]")
	}
	if len(text) > 240 {
		return fmt.Sprintf("%s...", text[:240])
	}
	return text
}
