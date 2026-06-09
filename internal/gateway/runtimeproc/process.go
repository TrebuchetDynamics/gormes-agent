package runtimeproc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidationStatus classifies how much trust callers can place in persisted
// process identity evidence.
type ValidationStatus string

const (
	ValidationMissingState     ValidationStatus = "missing_state"
	ValidationMissingPIDFile   ValidationStatus = "missing_pid_file"
	ValidationStalePID         ValidationStatus = "stale_pid"
	ValidationPIDReused        ValidationStatus = "pid_reused"
	ValidationStopped          ValidationStatus = "stopped_process"
	ValidationPermissionDenied ValidationStatus = "permission_denied"
	ValidationLive             ValidationStatus = "live"
)

// Validation is read-only evidence produced when a runtime status snapshot is
// checked against process identity evidence.
type Validation struct {
	Status            ValidationStatus `json:"status,omitempty"`
	Live              bool             `json:"live"`
	Message           string           `json:"message,omitempty"`
	PID               int              `json:"pid,omitempty"`
	ExpectedStartTime int64            `json:"expected_start_time,omitempty"`
	ActualStartTime   int64            `json:"actual_start_time,omitempty"`
	Command           string           `json:"command,omitempty"`
	CheckedAt         string           `json:"checked_at,omitempty"`
}

var (
	ErrNotFound         = errors.New("runtime process not found")
	ErrPermissionDenied = errors.New("runtime process permission denied")
)

// ProcessTable looks up machine-local process identity evidence.
type ProcessTable interface {
	LookupRuntimeProcess(pid int) (ProcessInfo, error)
}

// ProcessInfo is the process identity evidence needed by runtime validation.
type ProcessInfo struct {
	PID       int
	StartTime int64
	Command   string
	Stopped   bool
}

// ProcTable reads Linux /proc process identity evidence.
type ProcTable struct{}

func (ProcTable) LookupRuntimeProcess(pid int) (ProcessInfo, error) {
	if pid <= 0 {
		return ProcessInfo{}, ErrNotFound
	}
	statPath := filepath.Join("/proc", fmt.Sprint(pid), "stat")
	raw, err := os.ReadFile(statPath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return ProcessInfo{}, ErrPermissionDenied
		}
		if errors.Is(err, os.ErrNotExist) {
			return ProcessInfo{}, ErrNotFound
		}
		return ProcessInfo{}, err
	}
	startTime, state, ok := parseProcStatIdentity(string(raw))
	if !ok {
		return ProcessInfo{}, ErrNotFound
	}
	info := ProcessInfo{
		PID:       pid,
		StartTime: startTime,
		Stopped:   stoppedProcState(state),
	}
	if cmdline, ok := readProcCmdline(pid); ok {
		info.Command = cmdline
	}
	return info, nil
}

func stoppedProcState(state string) bool {
	switch state {
	case "T", "t", "Z", "z", "X", "x":
		return true
	default:
		return false
	}
}

func readProcCmdline(pid int) (string, bool) {
	raw, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	parts := strings.Split(string(raw), "\x00")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, " "), true
}

// ProcessStartTime returns the kernel start-time tick for pid when available.
func ProcessStartTime(pid int) (int64, bool) {
	if pid <= 0 {
		return 0, false
	}
	raw, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(pid), "stat"))
	if err != nil {
		return 0, false
	}
	return parseProcStatStartTime(string(raw))
}

func parseProcStatStartTime(stat string) (int64, bool) {
	startTime, _, ok := parseProcStatIdentity(stat)
	return startTime, ok
}

func parseProcStatIdentity(stat string) (int64, string, bool) {
	commEnd := strings.LastIndex(stat, ")")
	if commEnd < 0 || commEnd+2 >= len(stat) {
		return 0, "", false
	}
	fields := strings.Fields(stat[commEnd+2:])
	if len(fields) <= 19 {
		return 0, "", false
	}
	var start int64
	if _, err := fmt.Sscan(fields[19], &start); err != nil || start <= 0 {
		return 0, "", false
	}
	return start, fields[0], true
}
