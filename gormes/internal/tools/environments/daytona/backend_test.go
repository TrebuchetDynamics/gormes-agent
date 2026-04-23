package daytona

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestConfigCreateRequestNormalizesResourcesAndLabels(t *testing.T) {
	cfg := Config{
		Image:    "ghcr.io/daytonaio/workspace:latest",
		TaskID:   "task-123",
		CPU:      2,
		MemoryMB: 5120,
		DiskMB:   11264,
	}

	normalized := cfg.Normalized()
	if normalized.CWD != "/home/daytona" {
		t.Fatalf("Normalized().CWD = %q, want %q", normalized.CWD, "/home/daytona")
	}

	req := normalized.CreateRequest()
	if req.Name != "gormes-task-123" {
		t.Fatalf("CreateRequest().Name = %q, want %q", req.Name, "gormes-task-123")
	}
	if req.Resources.CPU != 2 {
		t.Fatalf("CreateRequest().Resources.CPU = %d, want %d", req.Resources.CPU, 2)
	}
	if req.Resources.MemoryGiB != 5 {
		t.Fatalf("CreateRequest().Resources.MemoryGiB = %d, want %d", req.Resources.MemoryGiB, 5)
	}
	if req.Resources.DiskGiB != 10 {
		t.Fatalf("CreateRequest().Resources.DiskGiB = %d, want %d", req.Resources.DiskGiB, 10)
	}
	if req.AutoStopInterval != 0 {
		t.Fatalf("CreateRequest().AutoStopInterval = %d, want %d", req.AutoStopInterval, 0)
	}
	if got := req.Labels["gormes_task_id"]; got != "task-123" {
		t.Fatalf("gormes_task_id label = %q, want %q", got, "task-123")
	}
	if got := req.Labels["hermes_task_id"]; got != "task-123" {
		t.Fatalf("hermes_task_id label = %q, want %q", got, "task-123")
	}
}

func TestBackendSandboxReusesPersistentSandboxBeforeCreate(t *testing.T) {
	sb := &fakeSandbox{
		id:      "sb-1",
		name:    "gormes-task-123",
		state:   StateStopped,
		homeDir: "/workspaces/alice",
	}
	client := &fakeClient{
		byName: map[string]Sandbox{
			"gormes-task-123": sb,
		},
	}

	backend := New(client, Config{
		Image:                "ghcr.io/daytonaio/workspace:latest",
		TaskID:               "task-123",
		CWD:                  "/home/daytona",
		PersistentFilesystem: true,
	})

	got, err := backend.Sandbox(context.Background())
	if err != nil {
		t.Fatalf("Sandbox() error = %v", err)
	}
	if got != sb {
		t.Fatalf("Sandbox() returned %p, want %p", got, sb)
	}
	if sb.startCalls != 1 {
		t.Fatalf("startCalls = %d, want %d", sb.startCalls, 1)
	}
	if client.createCalls != 0 {
		t.Fatalf("createCalls = %d, want %d", client.createCalls, 0)
	}
	if got := backend.WorkingDir(); got != "/workspaces/alice" {
		t.Fatalf("WorkingDir() = %q, want %q", got, "/workspaces/alice")
	}
}

func TestBackendSandboxFallsBackToLegacyHermesName(t *testing.T) {
	sb := &fakeSandbox{
		id:      "sb-legacy",
		name:    "hermes-task-123",
		state:   StateRunning,
		homeDir: "/home/daytona",
	}
	client := &fakeClient{
		byName: map[string]Sandbox{
			"hermes-task-123": sb,
		},
		getErr: map[string]error{
			"gormes-task-123": ErrNotFound,
		},
	}

	backend := New(client, Config{
		Image:                "ghcr.io/daytonaio/workspace:latest",
		TaskID:               "task-123",
		PersistentFilesystem: true,
	})

	got, err := backend.Sandbox(context.Background())
	if err != nil {
		t.Fatalf("Sandbox() error = %v", err)
	}
	if got != sb {
		t.Fatalf("Sandbox() returned %p, want %p", got, sb)
	}
	if !reflect.DeepEqual(client.getOrder, []string{"gormes-task-123", "hermes-task-123"}) {
		t.Fatalf("getOrder = %v, want %v", client.getOrder, []string{"gormes-task-123", "hermes-task-123"})
	}
}

func TestBackendExecuteBuildsShellCommandAndExecOptions(t *testing.T) {
	sb := &fakeSandbox{
		id:      "sb-1",
		name:    "gormes-task-123",
		state:   StateRunning,
		homeDir: "/workspaces/alice",
		result: CommandResult{
			Output:   "ok",
			ExitCode: 0,
		},
	}
	client := &fakeClient{
		byName: map[string]Sandbox{
			"gormes-task-123": sb,
		},
	}

	backend := New(client, Config{
		Image:                "ghcr.io/daytonaio/workspace:latest",
		TaskID:               "task-123",
		CWD:                  "~",
		Timeout:              90 * time.Second,
		PersistentFilesystem: true,
	})

	result, err := backend.Execute(context.Background(), ExecRequest{
		Command: "echo hi",
		Login:   true,
		Env: map[string]string{
			"NAME": "gormes",
		},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != sb.result {
		t.Fatalf("Execute() result = %+v, want %+v", result, sb.result)
	}
	if sb.lastCommand != "bash -l -c 'echo hi'" {
		t.Fatalf("lastCommand = %q, want %q", sb.lastCommand, "bash -l -c 'echo hi'")
	}
	if sb.lastExec.CWD != "/workspaces/alice" {
		t.Fatalf("lastExec.CWD = %q, want %q", sb.lastExec.CWD, "/workspaces/alice")
	}
	if sb.lastExec.Timeout != 5*time.Second {
		t.Fatalf("lastExec.Timeout = %v, want %v", sb.lastExec.Timeout, 5*time.Second)
	}
	if !reflect.DeepEqual(sb.lastExec.Env, map[string]string{"NAME": "gormes"}) {
		t.Fatalf("lastExec.Env = %v, want %v", sb.lastExec.Env, map[string]string{"NAME": "gormes"})
	}
}

func TestBackendCleanupRespectsPersistence(t *testing.T) {
	t.Run("persistent sandboxes are stopped", func(t *testing.T) {
		sb := &fakeSandbox{id: "sb-persist", name: "gormes-task-123", state: StateRunning}
		client := &fakeClient{
			byName: map[string]Sandbox{"gormes-task-123": sb},
		}
		backend := New(client, Config{
			Image:                "ghcr.io/daytonaio/workspace:latest",
			TaskID:               "task-123",
			PersistentFilesystem: true,
		})
		if _, err := backend.Sandbox(context.Background()); err != nil {
			t.Fatalf("Sandbox() error = %v", err)
		}

		if err := backend.Cleanup(context.Background()); err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}
		if sb.stopCalls != 1 {
			t.Fatalf("stopCalls = %d, want %d", sb.stopCalls, 1)
		}
		if client.deleteCalls != 0 {
			t.Fatalf("deleteCalls = %d, want %d", client.deleteCalls, 0)
		}
	})

	t.Run("ephemeral sandboxes are deleted", func(t *testing.T) {
		sb := &fakeSandbox{id: "sb-ephemeral", name: "gormes-task-123", state: StateRunning}
		client := &fakeClient{
			byName: map[string]Sandbox{"gormes-task-123": sb},
		}
		backend := New(client, Config{
			Image:                "ghcr.io/daytonaio/workspace:latest",
			TaskID:               "task-123",
			PersistentFilesystem: false,
		})
		if _, err := backend.Sandbox(context.Background()); err != nil {
			t.Fatalf("Sandbox() error = %v", err)
		}

		if err := backend.Cleanup(context.Background()); err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}
		if client.deleteCalls != 1 {
			t.Fatalf("deleteCalls = %d, want %d", client.deleteCalls, 1)
		}
		if client.createCalls != 1 {
			t.Fatalf("createCalls = %d, want %d", client.createCalls, 1)
		}
		if got := client.deleted[0].Name(); got != "gormes-task-123" {
			t.Fatalf("deleted[0].Name() = %q, want %q", got, "gormes-task-123")
		}
	})
}

type fakeClient struct {
	byName      map[string]Sandbox
	getErr      map[string]error
	getOrder    []string
	createCalls int
	deleteCalls int
	deleted     []Sandbox
}

func (c *fakeClient) Get(_ context.Context, name string) (Sandbox, error) {
	c.getOrder = append(c.getOrder, name)
	if err := c.getErr[name]; err != nil {
		return nil, err
	}
	if sb, ok := c.byName[name]; ok {
		return sb, nil
	}
	return nil, ErrNotFound
}

func (c *fakeClient) Create(_ context.Context, req CreateRequest) (Sandbox, error) {
	c.createCalls++
	sb := &fakeSandbox{id: "created", name: req.Name, state: StateRunning, homeDir: "/home/daytona"}
	if c.byName == nil {
		c.byName = map[string]Sandbox{}
	}
	c.byName[req.Name] = sb
	return sb, nil
}

func (c *fakeClient) Delete(_ context.Context, sandbox Sandbox) error {
	c.deleteCalls++
	c.deleted = append(c.deleted, sandbox)
	return nil
}

type fakeSandbox struct {
	id          string
	name        string
	state       State
	homeDir     string
	result      CommandResult
	execErr     error
	startCalls  int
	stopCalls   int
	lastCommand string
	lastExec    ExecOptions
}

func (s *fakeSandbox) ID() string   { return s.id }
func (s *fakeSandbox) Name() string { return s.name }
func (s *fakeSandbox) State() State { return s.state }

func (s *fakeSandbox) Start(context.Context) error {
	s.startCalls++
	s.state = StateRunning
	return nil
}

func (s *fakeSandbox) Stop(context.Context) error {
	s.stopCalls++
	s.state = StateStopped
	return nil
}

func (s *fakeSandbox) HomeDir(context.Context) (string, error) {
	if s.homeDir == "" {
		return "", errors.New("missing home dir")
	}
	return s.homeDir, nil
}

func (s *fakeSandbox) ExecuteCommand(_ context.Context, command string, opts ExecOptions) (CommandResult, error) {
	s.lastCommand = command
	s.lastExec = opts
	if s.execErr != nil {
		return CommandResult{}, s.execErr
	}
	return s.result, nil
}
