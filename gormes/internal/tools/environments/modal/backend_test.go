package modal

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestConfigCreateRequestNormalizesResourcesAndRestoreSnapshot(t *testing.T) {
	cfg := Config{
		TaskID:   "task-123",
		CPU:      2,
		MemoryMB: 6144,
		DiskMB:   20480,
	}

	normalized := cfg.Normalized()
	if normalized.AppName != "gormes-agent" {
		t.Fatalf("Normalized().AppName = %q, want %q", normalized.AppName, "gormes-agent")
	}
	if normalized.Image != "nikolaik/python-nodejs:python3.11-nodejs20" {
		t.Fatalf("Normalized().Image = %q, want default image", normalized.Image)
	}
	if normalized.CWD != "/root" {
		t.Fatalf("Normalized().CWD = %q, want %q", normalized.CWD, "/root")
	}

	req := normalized.CreateRequest("im-snapshot-123")
	if req.LogicalKey != "task-123" {
		t.Fatalf("CreateRequest().LogicalKey = %q, want %q", req.LogicalKey, "task-123")
	}
	if req.Image != "nikolaik/python-nodejs:python3.11-nodejs20" {
		t.Fatalf("CreateRequest().Image = %q, want default image", req.Image)
	}
	if req.RestoreSnapshot != "im-snapshot-123" {
		t.Fatalf("CreateRequest().RestoreSnapshot = %q, want %q", req.RestoreSnapshot, "im-snapshot-123")
	}
	if req.Timeout != time.Hour {
		t.Fatalf("CreateRequest().Timeout = %v, want %v", req.Timeout, time.Hour)
	}
	if req.CPU != 2 {
		t.Fatalf("CreateRequest().CPU = %d, want %d", req.CPU, 2)
	}
	if req.MemoryMB != 6144 {
		t.Fatalf("CreateRequest().MemoryMB = %d, want %d", req.MemoryMB, 6144)
	}
	if req.DiskMB != 20480 {
		t.Fatalf("CreateRequest().DiskMB = %d, want %d", req.DiskMB, 20480)
	}
	if !reflect.DeepEqual(req.Command, []string{"sleep", "infinity"}) {
		t.Fatalf("CreateRequest().Command = %#v, want sleep infinity", req.Command)
	}
}

func TestBackendExecuteRestoresSnapshotOnceAndWrapsShellCommand(t *testing.T) {
	sb := &fakeSandbox{
		id: "sb-1",
		results: []CommandResult{
			{Output: "hi\n", ExitCode: 0},
			{Output: "/root\n", ExitCode: 0},
		},
	}
	client := &fakeClient{sandbox: sb}
	store := &fakeSnapshotStore{
		snapshots: map[string]string{"task-123": "im-restore-123"},
	}

	backend := New(client, store, Config{
		TaskID:               "task-123",
		CWD:                  "~",
		Timeout:              90 * time.Second,
		PersistentFilesystem: true,
	})

	first, err := backend.Execute(context.Background(), ExecRequest{
		Command: "echo hi",
		Login:   true,
		Env:     map[string]string{"NAME": "gormes"},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute(first) error = %v", err)
	}
	if first.Output != "hi\n" || first.ExitCode != 0 {
		t.Fatalf("Execute(first) = %+v, want output %q exit %d", first, "hi\n", 0)
	}

	second, err := backend.Execute(context.Background(), ExecRequest{
		Command: "pwd",
	})
	if err != nil {
		t.Fatalf("Execute(second) error = %v", err)
	}
	if second.Output != "/root\n" || second.ExitCode != 0 {
		t.Fatalf("Execute(second) = %+v, want output %q exit %d", second, "/root\n", 0)
	}

	if got := backend.WorkingDir(); got != "/root" {
		t.Fatalf("WorkingDir() = %q, want %q", got, "/root")
	}

	if !reflect.DeepEqual(client.appCalls, []appCall{{Name: "gormes-agent", CreateIfMissing: true}}) {
		t.Fatalf("App() calls = %#v, want one default app lookup", client.appCalls)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("len(createRequests) = %d, want 1", len(client.createRequests))
	}
	if got := client.createRequests[0].RestoreSnapshot; got != "im-restore-123" {
		t.Fatalf("createRequests[0].RestoreSnapshot = %q, want %q", got, "im-restore-123")
	}

	wantExecs := []execCall{
		{
			Command: []string{"bash", "-l", "-c", "echo hi"},
			Options: ExecOptions{
				CWD:     "/root",
				Env:     map[string]string{"NAME": "gormes"},
				Timeout: 5 * time.Second,
			},
		},
		{
			Command: []string{"bash", "-c", "pwd"},
			Options: ExecOptions{
				CWD:     "/root",
				Timeout: 90 * time.Second,
			},
		},
	}
	if !reflect.DeepEqual(sb.execCalls, wantExecs) {
		t.Fatalf("execCalls = %#v, want %#v", sb.execCalls, wantExecs)
	}
}

func TestBackendSandboxFallsBackWhenStoredSnapshotIsGone(t *testing.T) {
	sb := &fakeSandbox{id: "sb-1"}
	client := &fakeClient{
		sandbox:    sb,
		createErrs: []error{ErrSnapshotNotFound, nil},
	}
	store := &fakeSnapshotStore{
		snapshots: map[string]string{"task-123": "im-stale"},
	}

	backend := New(client, store, Config{
		TaskID:               "task-123",
		PersistentFilesystem: true,
	})

	got, err := backend.Sandbox(context.Background())
	if err != nil {
		t.Fatalf("Sandbox() error = %v", err)
	}
	if got != sb {
		t.Fatalf("Sandbox() = %p, want %p", got, sb)
	}
	if !reflect.DeepEqual(store.deleted, []string{"task-123"}) {
		t.Fatalf("deleted snapshots = %#v, want stale task snapshot removed", store.deleted)
	}
	if len(client.createRequests) != 2 {
		t.Fatalf("len(createRequests) = %d, want 2", len(client.createRequests))
	}
	if got := client.createRequests[0].RestoreSnapshot; got != "im-stale" {
		t.Fatalf("first restore snapshot = %q, want %q", got, "im-stale")
	}
	if got := client.createRequests[1].RestoreSnapshot; got != "" {
		t.Fatalf("second restore snapshot = %q, want empty retry", got)
	}
}

func TestBackendCleanupSnapshotsPersistentFilesystem(t *testing.T) {
	t.Run("persistent sandboxes store a new snapshot before terminate", func(t *testing.T) {
		sb := &fakeSandbox{id: "sb-persist", snapshotID: "im-next"}
		client := &fakeClient{sandbox: sb}
		store := &fakeSnapshotStore{}
		backend := New(client, store, Config{
			TaskID:               "task-123",
			PersistentFilesystem: true,
		})
		if _, err := backend.Sandbox(context.Background()); err != nil {
			t.Fatalf("Sandbox() error = %v", err)
		}

		if err := backend.Cleanup(context.Background()); err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}
		if sb.snapshotCalls != 1 {
			t.Fatalf("snapshotCalls = %d, want %d", sb.snapshotCalls, 1)
		}
		if sb.terminateCalls != 1 {
			t.Fatalf("terminateCalls = %d, want %d", sb.terminateCalls, 1)
		}
		if got := store.saved["task-123"]; got != "im-next" {
			t.Fatalf("saved snapshot = %q, want %q", got, "im-next")
		}
	})

	t.Run("ephemeral sandboxes terminate without snapshotting", func(t *testing.T) {
		sb := &fakeSandbox{id: "sb-ephemeral"}
		client := &fakeClient{sandbox: sb}
		store := &fakeSnapshotStore{}
		backend := New(client, store, Config{
			TaskID:               "task-123",
			PersistentFilesystem: false,
		})
		if _, err := backend.Sandbox(context.Background()); err != nil {
			t.Fatalf("Sandbox() error = %v", err)
		}

		if err := backend.Cleanup(context.Background()); err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}
		if sb.snapshotCalls != 0 {
			t.Fatalf("snapshotCalls = %d, want %d", sb.snapshotCalls, 0)
		}
		if sb.terminateCalls != 1 {
			t.Fatalf("terminateCalls = %d, want %d", sb.terminateCalls, 1)
		}
		if len(store.saved) != 0 {
			t.Fatalf("saved snapshots = %#v, want none", store.saved)
		}
	})
}

type fakeClient struct {
	appCalls       []appCall
	createRequests []CreateRequest
	createErrs     []error
	sandbox        Sandbox
}

type appCall struct {
	Name            string
	CreateIfMissing bool
}

func (c *fakeClient) App(_ context.Context, name string, createIfMissing bool) (App, error) {
	c.appCalls = append(c.appCalls, appCall{
		Name:            name,
		CreateIfMissing: createIfMissing,
	})
	return App{Name: name}, nil
}

func (c *fakeClient) CreateSandbox(_ context.Context, _ App, req CreateRequest) (Sandbox, error) {
	c.createRequests = append(c.createRequests, req)
	if len(c.createErrs) > 0 {
		err := c.createErrs[0]
		c.createErrs = c.createErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	return c.sandbox, nil
}

type fakeSandbox struct {
	id             string
	results        []CommandResult
	execCalls      []execCall
	snapshotID     string
	snapshotErr    error
	snapshotCalls  int
	terminateCalls int
}

type execCall struct {
	Command []string
	Options ExecOptions
}

func (s *fakeSandbox) ID() string { return s.id }

func (s *fakeSandbox) Exec(_ context.Context, command []string, opts ExecOptions) (CommandResult, error) {
	s.execCalls = append(s.execCalls, execCall{
		Command: append([]string(nil), command...),
		Options: opts,
	})
	if len(s.results) == 0 {
		return CommandResult{}, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

func (s *fakeSandbox) SnapshotFilesystem(context.Context) (string, error) {
	s.snapshotCalls++
	if s.snapshotErr != nil {
		return "", s.snapshotErr
	}
	return s.snapshotID, nil
}

func (s *fakeSandbox) Terminate(context.Context) error {
	s.terminateCalls++
	return nil
}

type fakeSnapshotStore struct {
	snapshots map[string]string
	saved     map[string]string
	deleted   []string
	loadErr   error
	saveErr   error
	deleteErr error
}

func (s *fakeSnapshotStore) Lookup(taskID string) (string, bool, error) {
	if s.loadErr != nil {
		return "", false, s.loadErr
	}
	if s.snapshots == nil {
		return "", false, nil
	}
	snapshotID, ok := s.snapshots[taskID]
	return snapshotID, ok, nil
}

func (s *fakeSnapshotStore) Save(taskID, snapshotID string) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	if s.saved == nil {
		s.saved = map[string]string{}
	}
	s.saved[taskID] = snapshotID
	return nil
}

func (s *fakeSnapshotStore) Delete(taskID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, taskID)
	if s.snapshots != nil {
		delete(s.snapshots, taskID)
	}
	return nil
}

var _ Client = (*fakeClient)(nil)
var _ Sandbox = (*fakeSandbox)(nil)
var _ SnapshotStore = (*fakeSnapshotStore)(nil)
