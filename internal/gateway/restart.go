package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	restartTakeoverMarkerKind = "gormes-gateway-restart-takeover"

	// GatewayServiceRestartExitCode is EX_TEMPFAIL from sysexits.h. Service
	// managers can use it as the intentional restart handoff signal.
	GatewayServiceRestartExitCode = 75

	// RestartTakeoverMarkerTTL bounds how long a restart marker may suppress
	// redelivered platform updates.
	RestartTakeoverMarkerTTL = 5 * time.Minute
)

// RestartConfig wires restart behavior without letting unit tests invoke a
// real service manager.
type RestartConfig struct {
	MarkerStore             *RestartTakeoverStore
	ServiceManagerAvailable func() bool
	DrainTimeout            time.Duration
}

// RestartRequestedError carries the process exit code expected by the service
// manager restart path.
type RestartRequestedError struct {
	Code    int
	Message string
}

func (e RestartRequestedError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return "gateway restart requested"
}

func (e RestartRequestedError) ExitCode() int {
	if e.Code != 0 {
		return e.Code
	}
	return GatewayServiceRestartExitCode
}

// RestartTakeoverMarker is the short-lived cross-process marker written before
// returning the service-manager restart exit code.
type RestartTakeoverMarker struct {
	Kind               string `json:"kind"`
	SourcePlatform     string `json:"source_platform"`
	ChatID             string `json:"chat_id"`
	ThreadID           string `json:"thread_id,omitempty"`
	UpdateID           string `json:"update_id,omitempty"`
	MessageID          string `json:"message_id,omitempty"`
	Generation         uint64 `json:"generation"`
	RequestedAt        string `json:"requested_at"`
	NotificationSentAt string `json:"notification_sent_at,omitempty"`
}

// RestartTakeoverStore persists one restart takeover marker as atomic JSON.
type RestartTakeoverStore struct {
	path string
	now  func() time.Time
	ttl  time.Duration
}

func NewRestartTakeoverStore(path string) *RestartTakeoverStore {
	return &RestartTakeoverStore{
		path: path,
		now:  func() time.Time { return time.Now().UTC() },
		ttl:  RestartTakeoverMarkerTTL,
	}
}

func DefaultRestartTakeoverMarkerPath(runtimeStatusPath string) string {
	if runtimeStatusPath == "" {
		return ".restart_takeover.json"
	}
	return filepath.Join(filepath.Dir(runtimeStatusPath), ".restart_takeover.json")
}

func EnvironmentServiceManagerAvailable() bool {
	return strings.TrimSpace(os.Getenv("INVOCATION_ID")) != "" ||
		strings.TrimSpace(os.Getenv("LAUNCHD_JOB")) != "" ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("GORMES_GATEWAY_SERVICE_MANAGER")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("GORMES_GATEWAY_SERVICE_MANAGER")), "true")
}

func (s *RestartTakeoverStore) Write(ctx context.Context, marker RestartTakeoverMarker) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if marker.Kind == "" {
		marker.Kind = restartTakeoverMarkerKind
	}
	if marker.RequestedAt == "" {
		marker.RequestedAt = s.currentTime().Format(time.RFC3339Nano)
	}
	return writeRestartJSONAtomic(ctx, s.path, marker)
}

func (s *RestartTakeoverStore) Read(ctx context.Context) (RestartTakeoverMarker, bool, bool, error) {
	if s == nil || s.path == "" {
		return RestartTakeoverMarker{}, false, false, nil
	}
	if err := ctx.Err(); err != nil {
		return RestartTakeoverMarker{}, false, false, err
	}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return RestartTakeoverMarker{}, false, false, nil
	}
	if err != nil {
		return RestartTakeoverMarker{}, false, false, fmt.Errorf("read restart takeover marker: %w", err)
	}
	if len(raw) == 0 {
		_ = s.Clear(context.Background())
		return RestartTakeoverMarker{}, false, true, nil
	}
	var marker RestartTakeoverMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		_ = s.Clear(context.Background())
		return RestartTakeoverMarker{}, false, true, nil
	}
	if marker.Kind != "" && marker.Kind != restartTakeoverMarkerKind {
		_ = s.Clear(context.Background())
		return RestartTakeoverMarker{}, false, true, nil
	}
	if s.expired(marker) {
		_ = s.Clear(context.Background())
		return marker, false, true, nil
	}
	return marker, true, false, nil
}

func (s *RestartTakeoverStore) SuppressDuplicate(ctx context.Context, ev InboundEvent) (RestartTakeoverMarker, bool, error) {
	marker, ok, _, err := s.Read(ctx)
	if err != nil || !ok {
		return marker, false, err
	}
	if !restartMarkerMatchesEvent(marker, ev) {
		return marker, false, nil
	}
	if err := s.Clear(ctx); err != nil {
		return marker, false, err
	}
	return marker, true, nil
}

func (s *RestartTakeoverStore) MarkNotificationSent(ctx context.Context, marker RestartTakeoverMarker, at time.Time) error {
	marker.NotificationSentAt = at.UTC().Format(time.RFC3339Nano)
	return s.Write(ctx, marker)
}

func (s *RestartTakeoverStore) Clear(ctx context.Context) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove restart takeover marker: %w", err)
	}
	return nil
}

func (s *RestartTakeoverStore) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *RestartTakeoverStore) markerTTL() time.Duration {
	if s != nil && s.ttl > 0 {
		return s.ttl
	}
	return RestartTakeoverMarkerTTL
}

func (s *RestartTakeoverStore) expired(marker RestartTakeoverMarker) bool {
	requestedAt, err := time.Parse(time.RFC3339Nano, marker.RequestedAt)
	if err != nil {
		return true
	}
	return s.currentTime().Sub(requestedAt) > s.markerTTL()
}

func restartMarkerMatchesEvent(marker RestartTakeoverMarker, ev InboundEvent) bool {
	if !strings.EqualFold(strings.TrimSpace(marker.SourcePlatform), strings.TrimSpace(ev.Platform)) {
		return false
	}
	if strings.TrimSpace(marker.ChatID) != strings.TrimSpace(ev.ChatID) {
		return false
	}
	if strings.TrimSpace(marker.ThreadID) != strings.TrimSpace(ev.ThreadID) {
		return false
	}
	updateID := restartUpdateID(ev)
	messageID := strings.TrimSpace(ev.MsgID)
	if marker.UpdateID != "" && marker.UpdateID == updateID {
		return true
	}
	return marker.MessageID != "" && marker.MessageID == messageID
}

func restartUpdateID(ev InboundEvent) string {
	if id := strings.TrimSpace(ev.MessageID); id != "" {
		return id
	}
	return strings.TrimSpace(ev.MsgID)
}

func writeRestartJSONAtomic(ctx context.Context, path string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create restart marker dir: %w", err)
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode restart marker: %w", err)
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".restart-takeover-*.tmp")
	if err != nil {
		return fmt.Errorf("create restart marker temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write restart marker temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close restart marker temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace restart marker: %w", err)
	}
	return nil
}
