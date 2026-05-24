package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

// runRemoteTUIWithRuntime is the --remote <url> startup path. It does not
// instantiate a local kernel, provider client, or session DB; instead it
// dials the remote gateway's SSE event stream and forwards Bubble Tea
// submit/cancel callbacks to the gateway over plain HTTP.
//
// This keeps the local Bubble Tea path entirely intact when --remote is
// not set: the function is only reachable via runResolvedTUIWithRuntime
// when invocation.RemoteURL is non-empty, so existing fixtures continue
// through the kernel-backed branch.
func runRemoteTUIWithRuntime(cmd *cobra.Command, invocation tuiInvocation, runtime rootRuntime) error {
	if runtime.tuiProgramFactory == nil {
		runtime.tuiProgramFactory = defaultTUIProgramFactory
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	type dialResult struct {
		client gormescli.RemoteTUIClient
		err    error
	}
	dialDone := make(chan dialResult, 1)
	go func() {
		client, err := dialRemoteTUI(rootCtx, invocation.RemoteURL)
		dialDone <- dialResult{client: client, err: err}
	}()
	var client gormescli.RemoteTUIClient
	select {
	case result := <-dialDone:
		if result.err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"remote streaming unavailable at %s: %v\n\nLocal Bubble Tea mode still works: re-run gormes without --remote.\n",
				gormescli.RedactRemoteTUIURL(invocation.RemoteURL), result.err,
			)
			return result.err
		}
		client = result.client
	case <-time.After(5 * time.Second):
		err := fmt.Errorf("remote streaming startup timed out")
		fmt.Fprintf(cmd.ErrOrStderr(),
			"remote streaming unavailable at %s: %v\n\nLocal Bubble Tea mode still works: re-run gormes without --remote.\n",
			gormescli.RedactRemoteTUIURL(invocation.RemoteURL), err,
		)
		return err
	}
	defer client.Close()

	frames := client.Frames()
	submit := func(text string) {
		if err := client.Submit(rootCtx, text); err != nil {
			slog.Warn("remote submit failed", "err", err)
		}
	}
	cancelTurn := func() {
		if err := client.Cancel(rootCtx); err != nil {
			slog.Warn("remote cancel failed", "err", err)
		}
	}

	model := tui.NewModelWithOptions(frames, submit, cancelTurn, tui.Options{
		MouseTracking:  invocation.Config.TUI.MouseTracking,
		VoiceRecordKey: invocation.Config.Voice.RecordKey,
	})
	programOptions := []tea.ProgramOption{tea.WithAltScreen()}
	if invocation.Config.TUI.MouseTracking {
		programOptions = append(programOptions, tea.WithMouseAllMotion())
	}
	prog := runtime.tuiProgramFactory(model, programOptions...)

	programDone := make(chan struct{})
	go func() {
		<-rootCtx.Done()
		prog.Quit()
		select {
		case <-programDone:
		case <-time.After(kernel.ShutdownBudget):
			slog.Error("shutdown budget exceeded; forcing exit")
			os.Exit(3)
		}
	}()

	_, err := prog.Run()
	close(programDone)
	return err
}

func dialRemoteTUI(ctx context.Context, remoteURL string) (gormescli.RemoteTUIClient, error) {
	return gormescli.DialRemoteTUI(ctx, remoteURL, resolveRemoteTUISidecarURL())
}

func resolveRemoteTUIURL(flagValue string) string {
	if raw := strings.TrimSpace(flagValue); raw != "" {
		return raw
	}
	for _, key := range []string{"GORMES_TUI_GATEWAY_URL", "HERMES_TUI_GATEWAY_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return raw
		}
	}
	return ""
}

func resolveRemoteTUISidecarURL() string {
	for _, key := range []string{"GORMES_TUI_SIDECAR_URL", "HERMES_TUI_SIDECAR_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return raw
		}
	}
	return ""
}

func isWebSocketRemoteURL(raw string) bool {
	return gormescli.IsWebSocketRemoteURL(raw)
}
