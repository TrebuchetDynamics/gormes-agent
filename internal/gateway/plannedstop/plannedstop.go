package plannedstop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/jsonfile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/markerfile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/runtimeproc"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const markerKind = "gormes-gateway-planned-stop"

// MarkerTTL mirrors Hermes' short-lived planned-stop marker window. Stale
// markers are removed and never mask later unexpected exits.
const MarkerTTL = time.Minute

// Marker is written before an operator-initiated gateway stop sends
// SIGTERM/SIGINT to the live gateway process.
type Marker struct {
	Kind            string `json:"kind"`
	TargetPID       int    `json:"target_pid"`
	TargetStartTime int64  `json:"target_start_time"`
	StopperPID      int    `json:"stopper_pid"`
	Generation      uint64 `json:"generation"`
	Reason          string `json:"reason,omitempty"`
	WrittenAt       string `json:"written_at"`
}

type ConsumeStatus string

const (
	ConsumeMissing    ConsumeStatus = "missing"
	ConsumeMatched    ConsumeStatus = "matched"
	ConsumeStale      ConsumeStatus = "stale"
	ConsumeMismatched ConsumeStatus = "mismatched"
	ConsumeInvalid    ConsumeStatus = "invalid"
)

type ConsumeResult struct {
	Status  ConsumeStatus
	Matched bool
	Reason  string
	Marker  Marker
}

// Store persists one planned-stop marker as atomic JSON.
type Store struct {
	path      string
	now       func() time.Time
	pid       func() int
	startTime func(int) (int64, bool)
	ttl       time.Duration
}

func NewStore(path string) *Store {
	return &Store{
		path:      path,
		now:       func() time.Time { return time.Now().UTC() },
		pid:       os.Getpid,
		startTime: runtimeproc.ProcessStartTime,
		ttl:       MarkerTTL,
	}
}

func DefaultMarkerPath(runtimeStatusPath string) string {
	if runtimeStatusPath == "" {
		return ".gateway-planned-stop.json"
	}
	return filepath.Join(filepath.Dir(runtimeStatusPath), ".gateway-planned-stop.json")
}

func (s *Store) Write(ctx context.Context, marker Marker) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if marker.Kind == "" {
		marker.Kind = markerKind
	}
	if marker.StopperPID == 0 {
		marker.StopperPID = s.currentPID()
	}
	if marker.WrittenAt == "" {
		marker.WrittenAt = s.currentTime().Format(time.RFC3339Nano)
	}
	marker.Reason = sanitizeMarkerReason(marker.Reason)
	return jsonfile.WriteAtomic(ctx, s.path, marker, "planned stop marker")
}

func sanitizeMarkerReason(reason string) string {
	reason = strings.Join(strings.Fields(reason), " ")
	reason = redaction.RedactSecrets(reason)
	fields := strings.Fields(reason)
	for i, field := range fields {
		lower := strings.ToLower(field)
		if strings.Contains(lower, "[redacted]") && (strings.Contains(lower, "api_key") || strings.Contains(lower, "api-key") || strings.Contains(lower, "apikey") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password")) {
			fields[i] = "[redacted]"
		}
	}
	return strings.Join(fields, " ")
}

func (s *Store) ConsumeForSelf(ctx context.Context) (ConsumeResult, error) {
	if s == nil || s.path == "" {
		return ConsumeResult{Status: ConsumeMissing}, nil
	}
	if err := ctx.Err(); err != nil {
		return ConsumeResult{}, err
	}
	var marker Marker
	exists, err := jsonfile.Read(ctx, s.path, &marker, "planned stop marker")
	if errors.Is(err, jsonfile.ErrEmpty) {
		_ = s.Clear(context.Background())
		return ConsumeResult{Status: ConsumeInvalid, Reason: "empty marker"}, nil
	}
	if err != nil {
		if jsonfile.IsReadError(err) || !exists {
			return ConsumeResult{}, err
		}
		_ = s.Clear(context.Background())
		return ConsumeResult{Status: ConsumeInvalid, Reason: "decode marker: " + err.Error()}, nil
	}
	if !exists {
		return ConsumeResult{Status: ConsumeMissing}, nil
	}
	marker.Reason = sanitizeMarkerReason(marker.Reason)
	if marker.Kind != "" && marker.Kind != markerKind {
		_ = s.Clear(context.Background())
		return ConsumeResult{Status: ConsumeInvalid, Reason: "marker kind mismatch", Marker: marker}, nil
	}
	if s.markerStale(marker) {
		_ = s.Clear(context.Background())
		return ConsumeResult{Status: ConsumeStale, Reason: "marker is stale", Marker: marker}, nil
	}

	result := ConsumeResult{Status: ConsumeMismatched, Reason: "target pid/start_time mismatch", Marker: marker}
	if s.markerMatchesSelf(marker) {
		result.Status = ConsumeMatched
		result.Matched = true
		result.Reason = ""
	}
	_ = s.Clear(context.Background())
	return result, nil
}

func (s *Store) Clear(ctx context.Context) error {
	if s == nil {
		return nil
	}
	return markerfile.Clear(ctx, s.path, "planned stop marker")
}

func (s *Store) currentTime() time.Time {
	if s == nil {
		return markerfile.CurrentTime(nil)
	}
	return markerfile.CurrentTime(s.now)
}

func (s *Store) currentPID() int {
	if s != nil && s.pid != nil {
		return s.pid()
	}
	return os.Getpid()
}

func (s *Store) markerTTL() time.Duration {
	if s == nil {
		return MarkerTTL
	}
	return markerfile.PositiveDuration(s.ttl, MarkerTTL)
}

func (s *Store) markerStale(marker Marker) bool {
	writtenAt, err := time.Parse(time.RFC3339Nano, marker.WrittenAt)
	if err != nil {
		return true
	}
	return markerOutsideTTLWindow(s.currentTime(), writtenAt, s.markerTTL())
}

func markerOutsideTTLWindow(now, writtenAt time.Time, ttl time.Duration) bool {
	if ttl <= 0 {
		ttl = MarkerTTL
	}
	age := now.Sub(writtenAt)
	return age > ttl || age < -ttl
}

func (s *Store) markerMatchesSelf(marker Marker) bool {
	if marker.TargetPID <= 0 || marker.TargetStartTime == 0 {
		return false
	}
	pid := s.currentPID()
	if marker.TargetPID != pid {
		return false
	}
	startTime := s.startTime
	if startTime == nil {
		startTime = runtimeproc.ProcessStartTime
	}
	actualStartTime, ok := startTime(pid)
	return ok && actualStartTime != 0 && actualStartTime == marker.TargetStartTime
}
