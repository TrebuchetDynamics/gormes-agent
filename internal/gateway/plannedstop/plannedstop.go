package plannedstop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/jsonfile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/runtimeproc"
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
	return jsonfile.WriteAtomic(ctx, s.path, marker, "planned stop marker")
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
	if !exists {
		return ConsumeResult{Status: ConsumeMissing}, nil
	}
	if errors.Is(err, jsonfile.ErrEmpty) {
		_ = s.Clear(context.Background())
		return ConsumeResult{Status: ConsumeInvalid, Reason: "empty marker"}, nil
	}
	if err != nil {
		if jsonfile.IsReadError(err) {
			return ConsumeResult{}, err
		}
		_ = s.Clear(context.Background())
		return ConsumeResult{Status: ConsumeInvalid, Reason: "decode marker: " + err.Error()}, nil
	}
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
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove planned stop marker: %w", err)
	}
	return nil
}

func (s *Store) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) currentPID() int {
	if s != nil && s.pid != nil {
		return s.pid()
	}
	return os.Getpid()
}

func (s *Store) markerTTL() time.Duration {
	if s != nil && s.ttl > 0 {
		return s.ttl
	}
	return MarkerTTL
}

func (s *Store) markerStale(marker Marker) bool {
	writtenAt, err := time.Parse(time.RFC3339Nano, marker.WrittenAt)
	if err != nil {
		return true
	}
	return s.currentTime().Sub(writtenAt) > s.markerTTL()
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
