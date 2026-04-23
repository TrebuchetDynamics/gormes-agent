package daytona

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	defaultTaskID      = "default"
	defaultWorkdir     = "/home/daytona"
	defaultCPU         = 1
	defaultMemoryMB    = 1024
	defaultDiskMB      = 1024
	maxDiskGiB         = 10
	currentTaskIDLabel = "gormes_task_id"
	legacyTaskIDLabel  = "hermes_task_id"
)

var ErrNotFound = errors.New("daytona: sandbox not found")

type State string

const (
	StateRunning  State = "running"
	StateStopped  State = "stopped"
	StateArchived State = "archived"
)

type Resources struct {
	CPU       int
	MemoryGiB int
	DiskGiB   int
}

type CreateRequest struct {
	Image            string
	Name             string
	Labels           map[string]string
	AutoStopInterval int
	Resources        Resources
}

type ExecOptions struct {
	CWD     string
	Env     map[string]string
	Timeout time.Duration
}

type ExecRequest struct {
	Command string
	Login   bool
	Env     map[string]string
	Timeout time.Duration
}

type CommandResult struct {
	Output   string
	ExitCode int
}

type Sandbox interface {
	ID() string
	Name() string
	State() State
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	HomeDir(ctx context.Context) (string, error)
	ExecuteCommand(ctx context.Context, command string, opts ExecOptions) (CommandResult, error)
}

type Client interface {
	Get(ctx context.Context, name string) (Sandbox, error)
	Create(ctx context.Context, req CreateRequest) (Sandbox, error)
	Delete(ctx context.Context, sandbox Sandbox) error
}

type Config struct {
	Image                string
	CWD                  string
	Timeout              time.Duration
	CPU                  int
	MemoryMB             int
	DiskMB               int
	PersistentFilesystem bool
	TaskID               string
}

func (c Config) Normalized() Config {
	if strings.TrimSpace(c.TaskID) == "" {
		c.TaskID = defaultTaskID
	}
	if strings.TrimSpace(c.CWD) == "" {
		c.CWD = defaultWorkdir
	}
	if c.CPU <= 0 {
		c.CPU = defaultCPU
	}
	if c.MemoryMB <= 0 {
		c.MemoryMB = defaultMemoryMB
	}
	if c.DiskMB <= 0 {
		c.DiskMB = defaultDiskMB
	}
	return c
}

func (c Config) SandboxName() string {
	return "gormes-" + c.Normalized().TaskID
}

func (c Config) LegacySandboxName() string {
	return "hermes-" + c.Normalized().TaskID
}

func (c Config) Labels() map[string]string {
	taskID := c.Normalized().TaskID
	return map[string]string{
		currentTaskIDLabel: taskID,
		legacyTaskIDLabel:  taskID,
	}
}

func (c Config) CreateRequest() CreateRequest {
	n := c.Normalized()
	return CreateRequest{
		Image:            n.Image,
		Name:             n.SandboxName(),
		Labels:           n.Labels(),
		AutoStopInterval: 0,
		Resources: Resources{
			CPU:       n.CPU,
			MemoryGiB: ceilGiB(n.MemoryMB),
			DiskGiB:   min(ceilGiB(n.DiskMB), maxDiskGiB),
		},
	}
}

type Backend struct {
	client  Client
	config  Config
	sandbox Sandbox
	workdir string
}

func New(client Client, cfg Config) *Backend {
	n := cfg.Normalized()
	return &Backend{
		client:  client,
		config:  n,
		workdir: n.CWD,
	}
}

func (b *Backend) WorkingDir() string {
	return b.workdir
}

func (b *Backend) Sandbox(ctx context.Context) (Sandbox, error) {
	if b.sandbox != nil {
		return b.prepare(ctx, b.sandbox)
	}
	if b.client == nil {
		return nil, errors.New("daytona: nil client")
	}

	sb, err := b.lookupPersistentSandbox(ctx)
	if err != nil {
		return nil, err
	}

	if sb == nil {
		created, err := b.client.Create(ctx, b.config.CreateRequest())
		if err != nil {
			return nil, err
		}
		sb = created
	}

	b.sandbox = sb
	return b.prepare(ctx, sb)
}

func (b *Backend) Execute(ctx context.Context, req ExecRequest) (CommandResult, error) {
	sb, err := b.Sandbox(ctx)
	if err != nil {
		return CommandResult{}, err
	}
	opts := ExecOptions{
		CWD:     b.workdir,
		Env:     cloneEnv(req.Env),
		Timeout: req.Timeout,
	}
	if opts.Timeout <= 0 {
		opts.Timeout = b.config.Timeout
	}
	return sb.ExecuteCommand(ctx, buildShellCommand(req.Command, req.Login), opts)
}

func (b *Backend) Cleanup(ctx context.Context) error {
	if b.sandbox == nil {
		return nil
	}
	sb := b.sandbox
	b.sandbox = nil
	if b.config.PersistentFilesystem {
		return sb.Stop(ctx)
	}
	return b.client.Delete(ctx, sb)
}

func (b *Backend) prepare(ctx context.Context, sb Sandbox) (Sandbox, error) {
	if needsStart(sb.State()) {
		if err := sb.Start(ctx); err != nil {
			return nil, err
		}
	}
	if home, err := sb.HomeDir(ctx); err == nil {
		b.workdir = resolveWorkdir(b.config.CWD, home)
	}
	return sb, nil
}

func (b *Backend) lookupPersistentSandbox(ctx context.Context) (Sandbox, error) {
	if !b.config.PersistentFilesystem {
		return nil, nil
	}
	for _, name := range []string{b.config.SandboxName(), b.config.LegacySandboxName()} {
		sb, err := b.client.Get(ctx, name)
		if err == nil {
			return sb, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

func needsStart(state State) bool {
	return state == StateStopped || state == StateArchived
}

func resolveWorkdir(requested, home string) string {
	switch strings.TrimSpace(requested) {
	case "", "~", defaultWorkdir:
		if strings.TrimSpace(home) != "" {
			return home
		}
		return defaultWorkdir
	default:
		return requested
	}
}

func buildShellCommand(command string, login bool) string {
	shell := "bash -c "
	if login {
		shell = "bash -l -c "
	}
	return shell + shellQuote(command)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func cloneEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = v
	}
	return out
}

func ceilGiB(mebibytes int) int {
	if mebibytes <= 0 {
		return 1
	}
	return (mebibytes + 1023) / 1024
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
