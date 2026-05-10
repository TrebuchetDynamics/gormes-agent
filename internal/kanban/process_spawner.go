package kanban

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultWorkerLogRotateBytes int64 = 1 << 20

type ProcessStartRequest struct {
	Binary     string
	Args       []string
	Env        map[string]string
	Dir        string
	StdoutPath string
	StderrPath string
}

type ProcessStartResult struct {
	PID       int
	StartedAt time.Time
}

type ProcessStarter interface {
	StartKanbanProcess(context.Context, ProcessStartRequest) (ProcessStartResult, error)
}

type ProcessSpawner struct {
	Binary         string
	LogRoot        string
	MaxLogBytes    int64
	Starter        ProcessStarter
	Now            func() time.Time
	BinaryLookPath func(string) (string, error)
	ExecutablePath func() (string, error)
}

func (s ProcessSpawner) SpawnKanbanWorker(ctx context.Context, req SpawnRequest) (SpawnResult, error) {
	profile := strings.TrimSpace(req.Task.Assignee)
	if profile == "" {
		return SpawnResult{}, fmt.Errorf("worker_spawn_failed: task %s has no assignee", req.Task.ID)
	}
	binary, err := s.resolveBinary()
	if err != nil {
		return SpawnResult{}, fmt.Errorf("worker_spawn_failed: %w", err)
	}
	logRoot := strings.TrimSpace(s.LogRoot)
	if logRoot == "" {
		logRoot = kanbanWorkerLogRootForDBPath(req.Env["GORMES_KANBAN_DB"])
	}
	logPath := filepath.Join(logRoot, req.Task.ID+".log")
	if err := rotateWorkerLog(logPath, s.maxLogBytes()); err != nil {
		return SpawnResult{}, fmt.Errorf("worker_spawn_failed: prepare kanban worker log: %w", err)
	}

	starter := s.Starter
	if starter == nil {
		starter = OSProcessStarter{}
	}
	started, err := starter.StartKanbanProcess(ctx, ProcessStartRequest{
		Binary: binary,
		Args: []string{
			"-p", profile,
			"--skills", "kanban-worker",
			"chat", "-q", "work kanban task " + req.Task.ID,
		},
		Env:        kanbanWorkerEnv(req, profile),
		Dir:        req.WorkspacePath,
		StdoutPath: logPath,
		StderrPath: logPath,
	})
	if err != nil {
		return SpawnResult{}, fmt.Errorf("worker_spawn_failed: %w", err)
	}
	if started.StartedAt.IsZero() {
		started.StartedAt = s.now()
	}
	return SpawnResult{PID: started.PID, StartedAt: started.StartedAt}, nil
}

func (s ProcessSpawner) resolveBinary() (string, error) {
	explicit := strings.TrimSpace(s.Binary)
	if explicit != "" {
		return explicit, nil
	}
	lookPath := s.BinaryLookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if path, err := lookPath("gormes"); err == nil && strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path), nil
	}
	executablePath := s.ExecutablePath
	if executablePath == nil {
		executablePath = os.Executable
	}
	path, err := executablePath()
	if err != nil {
		return "", fmt.Errorf("resolve default gormes worker executable: %w", err)
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("resolve default gormes worker executable: current executable path is empty")
	}
	return path, nil
}

func (s ProcessSpawner) maxLogBytes() int64 {
	if s.MaxLogBytes > 0 {
		return s.MaxLogBytes
	}
	return defaultWorkerLogRotateBytes
}

func (s ProcessSpawner) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func kanbanWorkerEnv(req SpawnRequest, profile string) map[string]string {
	env := make(map[string]string, len(req.Env)+4)
	for key, value := range req.Env {
		if strings.HasPrefix(key, "HERMES") {
			continue
		}
		env[key] = value
	}
	env["GORMES_KANBAN_TASK"] = req.Task.ID
	env["GORMES_KANBAN_WORKSPACE"] = req.WorkspacePath
	env["GORMES_PROFILE"] = profile
	return env
}

func rotateWorkerLog(path string, maxBytes int64) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("log path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil && info.Size() >= maxBytes {
		rotated := path + ".1"
		if err := os.Remove(rotated); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Rename(path, rotated); err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

type OSProcessStarter struct{}

func (OSProcessStarter) StartKanbanProcess(ctx context.Context, req ProcessStartRequest) (ProcessStartResult, error) {
	if strings.TrimSpace(req.Binary) == "" {
		return ProcessStartResult{}, errors.New("gormes binary is required")
	}
	log, err := os.OpenFile(req.StdoutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return ProcessStartResult{}, err
	}
	defer log.Close()

	cmd := exec.CommandContext(ctx, req.Binary, req.Args...)
	cmd.Dir = req.Dir
	cmd.Env = envMapToList(req.Env)
	cmd.Stdin = nil
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		return ProcessStartResult{}, err
	}
	return ProcessStartResult{PID: cmd.Process.Pid, StartedAt: time.Now().UTC()}, nil
}

func envMapToList(env map[string]string) []string {
	merged := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok && !strings.HasPrefix(key, "HERMES") {
			merged[key] = value
		}
	}
	for key, value := range env {
		if !strings.HasPrefix(key, "HERMES") {
			merged[key] = value
		}
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+merged[key])
	}
	return out
}
