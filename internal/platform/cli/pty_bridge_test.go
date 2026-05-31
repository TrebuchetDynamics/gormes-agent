package cli

import (
	"errors"
	"testing"
	"time"

	clipty "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/pty"
)

func TestPtyBridgeFacadeDelegatesToPtyPackage(t *testing.T) {
	if PtyAvailable() != clipty.PtyAvailable() {
		t.Fatalf("PtyAvailable facade = %v, want pty package result %v", PtyAvailable(), clipty.PtyAvailable())
	}

	session := &recordingPtySession{}
	bridge := NewPtyAdapterForSession(session)
	if bridge == nil {
		t.Fatal("NewPtyAdapterForSession facade returned nil")
	}
	if err := bridge.Write(nil); !errors.Is(err, ErrInvalidPtyMessage) {
		t.Fatalf("Write(nil) err = %v, want ErrInvalidPtyMessage", err)
	}
}

type recordingPtySession struct {
	writes  [][]byte
	resizes []PtySize
	closed  bool
}

func (s *recordingPtySession) Read(time.Duration, int) ([]byte, error) {
	return []byte{}, nil
}

func (s *recordingPtySession) Write(data []byte) error {
	s.writes = append(s.writes, append([]byte(nil), data...))
	return nil
}

func (s *recordingPtySession) Resize(cols, rows int) error {
	s.resizes = append(s.resizes, PtySize{Cols: cols, Rows: rows})
	return nil
}

func (s *recordingPtySession) Close() error {
	s.closed = true
	return nil
}

func (s *recordingPtySession) IsAlive() bool {
	return !s.closed
}

func (s *recordingPtySession) PID() int {
	return 123
}
