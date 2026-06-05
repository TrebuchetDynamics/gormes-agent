package gormescli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/remoteruntime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type RemoteTUIClient = remoteruntime.RemoteTUIClient

type TUIProgram interface {
	Run() (tea.Model, error)
	Quit()
}

type TUIProgramFactory func(tea.Model, ...tea.ProgramOption) TUIProgram

type RemoteTUIOptions struct {
	RemoteURL      string
	SidecarURL     string
	MouseTracking  bool
	ModelOptions   func(context.Context) tui.Options
	ProgramFactory TUIProgramFactory
	Dial           func(context.Context, string, string) (RemoteTUIClient, error)
	StartupTimeout time.Duration
	ShutdownBudget time.Duration
	Logger         *slog.Logger
	Exit           func(int)
}

func RunRemoteTUI(ctx context.Context, errOut io.Writer, opts RemoteTUIOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if opts.ProgramFactory == nil {
		opts.ProgramFactory = func(model tea.Model, options ...tea.ProgramOption) TUIProgram {
			return tea.NewProgram(model, options...)
		}
	}
	if opts.Dial == nil {
		opts.Dial = DialRemoteTUI
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = 5 * time.Second
	}
	if opts.ShutdownBudget <= 0 {
		opts.ShutdownBudget = kernel.ShutdownBudget
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Exit == nil {
		opts.Exit = os.Exit
	}

	rootCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	type dialResult struct {
		client RemoteTUIClient
		err    error
	}
	dialDone := make(chan dialResult, 1)
	go func() {
		client, err := opts.Dial(rootCtx, opts.RemoteURL, opts.SidecarURL)
		dialDone <- dialResult{client: client, err: err}
	}()

	var client RemoteTUIClient
	select {
	case result := <-dialDone:
		if result.err != nil {
			writeRemoteTUIUnavailable(errOut, opts.RemoteURL, result.err)
			return result.err
		}
		client = result.client
	case <-time.After(opts.StartupTimeout):
		err := fmt.Errorf("remote streaming startup timed out")
		writeRemoteTUIUnavailable(errOut, opts.RemoteURL, err)
		return err
	}
	defer client.Close()

	modelOpts := tui.Options{MouseTracking: opts.MouseTracking}
	if opts.ModelOptions != nil {
		modelOpts = opts.ModelOptions(rootCtx)
		modelOpts.MouseTracking = opts.MouseTracking
	}
	model := tui.NewModelWithOptions(client.Frames(), func(text string) {
		if err := client.Submit(rootCtx, text); err != nil {
			opts.Logger.Warn("remote submit failed", "err", err)
		}
	}, func() {
		if err := client.Cancel(rootCtx); err != nil {
			opts.Logger.Warn("remote cancel failed", "err", err)
		}
	}, modelOpts)

	programOptions := []tea.ProgramOption{tea.WithAltScreen()}
	if opts.MouseTracking {
		programOptions = append(programOptions, tea.WithMouseAllMotion())
	}
	prog := opts.ProgramFactory(model, programOptions...)

	programDone := make(chan struct{}, 1)
	go func() {
		<-rootCtx.Done()
		prog.Quit()
		select {
		case <-programDone:
		case <-time.After(opts.ShutdownBudget):
			opts.Logger.Error("shutdown budget exceeded; forcing exit")
			opts.Exit(3)
		}
	}()

	_, err := prog.Run()
	close(programDone)
	return err
}

func writeRemoteTUIUnavailable(out io.Writer, remoteURL string, err error) {
	fmt.Fprintf(out,
		"remote streaming unavailable at %s: %v\n\nLocal Bubble Tea mode still works: re-run gormes without --remote.\n",
		RedactRemoteTUIURL(remoteURL), err,
	)
}

func DialRemoteTUI(ctx context.Context, remoteURL, sidecarURL string) (RemoteTUIClient, error) {
	return remoteruntime.DialRemoteTUI(ctx, remoteURL, sidecarURL)
}

func RedactRemoteTUIURL(raw string) string {
	return remoteruntime.RedactRemoteTUIURL(raw)
}

func IsWebSocketRemoteURL(raw string) bool {
	return remoteruntime.IsWebSocketRemoteURL(raw)
}
