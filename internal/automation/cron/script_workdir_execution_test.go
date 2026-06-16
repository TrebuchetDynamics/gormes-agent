package cron

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	hermesclient "github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"go.etcd.io/bbolt"
)

func TestCronScriptExecutionBuildsPromptSections(t *testing.T) {
	for _, tc := range []struct {
		name       string
		result     CronScriptResult
		wantHeader string
		wantText   string
	}{
		{
			name:       "success",
			result:     CronScriptResult{Success: true, Output: "new PR: #123 fix typo"},
			wantHeader: "## Script Output",
			wantText:   "new PR: #123 fix typo",
		},
		{
			name:       "failure",
			result:     CronScriptResult{Success: false, Output: "Script not found: monitor.py"},
			wantHeader: "## Script Error",
			wantText:   "Script not found: monitor.py",
		},
		{
			name:       "empty output",
			result:     CronScriptResult{Success: true},
			wantHeader: "## Script Output",
			wantText:   "script produced no output",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fk := newFakeKernel("reported", 0)
			e, _, cleanup := newTestExecutorEnv(t, fk)
			defer cleanup()
			var calls int
			e.cfg.ScriptRunner = CronScriptRunnerFunc(func(_ context.Context, req CronScriptRequest) CronScriptResult {
				calls++
				if req.Path != "monitor.py" {
					t.Fatalf("script path = %q, want monitor.py", req.Path)
				}
				return tc.result
			})
			job := NewJob("scripted", "@daily", "Report any notable changes.")
			job.Script = "monitor.py"
			if err := e.cfg.JobStore.Create(job); err != nil {
				t.Fatalf("Create job: %v", err)
			}

			e.Run(context.Background(), job)

			if calls != 1 {
				t.Fatalf("script runner calls = %d, want 1", calls)
			}
			prompt := submittedCronPrompt(t, fk)
			for _, want := range []string{CronHeartbeatPrefix, tc.wantHeader, tc.wantText, "Report any notable changes."} {
				if !strings.Contains(prompt, want) {
					t.Fatalf("submitted prompt missing %q:\n%s", want, prompt)
				}
			}
		})
	}
}

func TestCronWorkdirExecutionBinding(t *testing.T) {
	workdir := t.TempDir()
	t.Setenv("TERMINAL_CWD", "/before")
	k := &envCapturingKernel{
		render: make(chan kernel.RenderFrame, 2),
	}
	e, _, cleanup := newTestExecutorEnv(t, k)
	defer cleanup()
	job := NewJob("workdir", "@daily", "Run from project.")
	job.Workdir = workdir
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	e.Run(context.Background(), job)

	if k.terminalCWD != workdir {
		t.Fatalf("TERMINAL_CWD during submit = %q, want %q", k.terminalCWD, workdir)
	}
	if got := os.Getenv("TERMINAL_CWD"); got != "/before" {
		t.Fatalf("TERMINAL_CWD after run = %q, want restored /before", got)
	}

	var orderMu sync.Mutex
	var order []string
	runner := RunnerFunc(func(_ context.Context, job Job) {
		orderMu.Lock()
		defer orderMu.Unlock()
		order = append(order, job.ID)
	})
	s := NewScheduler(SchedulerConfig{Executor: runner, MCPOrphanCleanup: func() {}}, nil)
	parallel := NewJob("parallel", "@daily", "parallel")
	workdirJob := NewJob("workdir", "@daily", "workdir")
	workdirJob.Workdir = workdir

	s.runTick(context.Background(), []Job{parallel, workdirJob})

	if len(order) != 2 {
		t.Fatalf("run order = %#v, want two jobs", order)
	}
	if order[0] != workdirJob.ID {
		t.Fatalf("run order = %#v, want workdir job first and sequential", order)
	}
}

func TestCronStateFilePermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cron")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	dbPath := filepath.Join(root, "jobs.db")
	db, err := bbolt.Open(dbPath, 0o666, nil)
	if err != nil {
		t.Fatalf("Open bolt: %v", err)
	}
	defer db.Close()
	if _, err := NewStore(db); err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if got := modeOf(t, root); got != 0o700 {
		t.Fatalf("cron dir mode = %#o, want 0700", got)
	}
	if got := modeOf(t, dbPath); got != 0o600 {
		t.Fatalf("cron state file mode = %#o, want 0600", got)
	}
}

type envCapturingKernel struct {
	render      chan kernel.RenderFrame
	terminalCWD string
}

func (k *envCapturingKernel) Submit(e kernel.PlatformEvent) error {
	k.terminalCWD = os.Getenv("TERMINAL_CWD")
	k.render <- kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: e.SessionID,
		History: []hermesclient.Message{
			{Role: "assistant", Content: "done"},
		},
	}
	return nil
}

func (k *envCapturingKernel) Subscribe() (<-chan kernel.RenderFrame, func()) {
	return k.render, func() {}
}

type RunnerFunc func(context.Context, Job)

func (f RunnerFunc) Run(ctx context.Context, job Job) { f(ctx, job) }

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

var errCronTest401 = errors.New("Error code: 401 - unauthorized")
