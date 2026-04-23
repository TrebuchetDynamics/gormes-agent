package modal

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	defaultTaskID    = "default"
	defaultAppName   = "gormes-agent"
	defaultImage     = "nikolaik/python-nodejs:python3.11-nodejs20"
	defaultWorkdir   = "/root"
	defaultCPU       = 1
	defaultMemoryMB  = 5120
	defaultDiskMB    = 51200
	sandboxKeepalive = time.Hour
)

var (
	ErrSnapshotNotFound = errors.New("modal: snapshot not found")
	errNilClient        = errors.New("modal: nil client")
)

type App struct {
	Name string
}

type CommandResult struct {
	Output   string
	ExitCode int
}

type CreateRequest struct {
	LogicalKey      string
	Image           string
	RestoreSnapshot string
	Command         []string
	Timeout         time.Duration
	CPU             int
	MemoryMB        int
	DiskMB          int
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

type Sandbox interface {
	ID() string
	Exec(ctx context.Context, command []string, opts ExecOptions) (CommandResult, error)
	SnapshotFilesystem(ctx context.Context) (string, error)
	Terminate(ctx context.Context) error
}

type Client interface {
	App(ctx context.Context, name string, createIfMissing bool) (App, error)
	CreateSandbox(ctx context.Context, app App, req CreateRequest) (Sandbox, error)
}

type SnapshotStore interface {
	Lookup(taskID string) (snapshotID string, ok bool, err error)
	Save(taskID, snapshotID string) error
	Delete(taskID string) error
}

type Config struct {
	AppName              string
	Image                string
	TaskID               string
	CWD                  string
	Timeout              time.Duration
	CPU                  int
	MemoryMB             int
	DiskMB               int
	PersistentFilesystem bool
}

type Backend struct {
	client    Client
	snapshots SnapshotStore
	config    Config
	sandbox   Sandbox
	workdir   string
}

func (c Config) Normalized() Config {
	if strings.TrimSpace(c.AppName) == "" {
		c.AppName = defaultAppName
	}
	if strings.TrimSpace(c.Image) == "" {
		c.Image = defaultImage
	}
	if strings.TrimSpace(c.TaskID) == "" {
		c.TaskID = defaultTaskID
	}
	c.CWD = resolveWorkdir(c.CWD)
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

func (c Config) CreateRequest(snapshotID string) CreateRequest {
	n := c.Normalized()
	return CreateRequest{
		LogicalKey:      n.TaskID,
		Image:           n.Image,
		RestoreSnapshot: strings.TrimSpace(snapshotID),
		Command:         []string{"sleep", "infinity"},
		Timeout:         sandboxKeepalive,
		CPU:             n.CPU,
		MemoryMB:        n.MemoryMB,
		DiskMB:          n.DiskMB,
	}
}

func New(client Client, snapshots SnapshotStore, cfg Config) *Backend {
	n := cfg.Normalized()
	return &Backend{
		client:    client,
		snapshots: snapshots,
		config:    n,
		workdir:   n.CWD,
	}
}

func (b *Backend) WorkingDir() string {
	return b.workdir
}

func (b *Backend) Sandbox(ctx context.Context) (Sandbox, error) {
	if b.sandbox != nil {
		return b.sandbox, nil
	}
	if b.client == nil {
		return nil, errNilClient
	}

	app, err := b.client.App(ctx, b.config.AppName, true)
	if err != nil {
		return nil, err
	}

	sb, err := b.createSandbox(ctx, app)
	if err != nil {
		return nil, err
	}
	b.sandbox = sb
	return sb, nil
}

func (b *Backend) Execute(ctx context.Context, req ExecRequest) (CommandResult, error) {
	sb, err := b.Sandbox(ctx)
	if err != nil {
		return CommandResult{}, err
	}

	return sb.Exec(ctx, shellArgs(req.Command, req.Login), ExecOptions{
		CWD:     b.workdir,
		Env:     cloneEnv(req.Env),
		Timeout: b.effectiveTimeout(req.Timeout),
	})
}

func (b *Backend) Cleanup(ctx context.Context) error {
	if b.sandbox == nil {
		return nil
	}

	sb := b.sandbox
	b.sandbox = nil

	snapshotErr := b.storeSnapshot(ctx, sb)
	terminateErr := sb.Terminate(ctx)
	if snapshotErr != nil && terminateErr != nil {
		return errors.Join(snapshotErr, terminateErr)
	}
	if snapshotErr != nil {
		return snapshotErr
	}
	return terminateErr
}

func (b *Backend) createSandbox(ctx context.Context, app App) (Sandbox, error) {
	req := b.config.CreateRequest("")
	if !b.config.PersistentFilesystem || b.snapshots == nil {
		return b.client.CreateSandbox(ctx, app, req)
	}

	snapshotID, ok, err := b.snapshots.Lookup(b.config.TaskID)
	if err != nil {
		return nil, err
	}
	if !ok || strings.TrimSpace(snapshotID) == "" {
		return b.client.CreateSandbox(ctx, app, req)
	}

	sb, err := b.client.CreateSandbox(ctx, app, b.config.CreateRequest(snapshotID))
	if err == nil {
		return sb, nil
	}
	if !errors.Is(err, ErrSnapshotNotFound) {
		return nil, err
	}
	if err := b.snapshots.Delete(b.config.TaskID); err != nil {
		return nil, err
	}
	return b.client.CreateSandbox(ctx, app, req)
}

func resolveWorkdir(requested string) string {
	switch strings.TrimSpace(requested) {
	case "", "~", defaultWorkdir:
		return defaultWorkdir
	default:
		return requested
	}
}

func shellArgs(command string, login bool) []string {
	if login {
		return []string{"bash", "-l", "-c", command}
	}
	return []string{"bash", "-c", command}
}

func cloneEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		out[key] = value
	}
	return out
}

func (b *Backend) effectiveTimeout(requested time.Duration) time.Duration {
	if requested > 0 {
		return requested
	}
	return b.config.Timeout
}

func (b *Backend) storeSnapshot(ctx context.Context, sb Sandbox) error {
	if !b.config.PersistentFilesystem || b.snapshots == nil {
		return nil
	}
	snapshotID, err := sb.SnapshotFilesystem(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(snapshotID) == "" {
		return nil
	}
	return b.snapshots.Save(b.config.TaskID, snapshotID)
}
