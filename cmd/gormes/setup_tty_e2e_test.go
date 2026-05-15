//go:build linux

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

const setupProviderTTYE2EHelperEnv = "GORMES_SETUP_PROVIDER_TTY_E2E_HELPER"
const setupToolsTTYE2EHelperEnv = "GORMES_SETUP_TOOLS_TTY_E2E_HELPER"

func TestSetupProviderTTYE2EConsumesArrowKeys(t *testing.T) {
	if os.Getenv(setupProviderTTYE2EHelperEnv) == "1" {
		runSetupProviderTTYE2EHelper()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSetupProviderTTYE2EConsumesArrowKeys", "--")
	cmd.Env = append(os.Environ(),
		setupProviderTTYE2EHelperEnv+"=1",
		"GORMES_HOME="+t.TempDir(),
		"TERM=xterm-256color",
	)

	tty, err := startLinuxPTY(cmd, 40, 120)
	if err != nil {
		t.Fatalf("start setup provider tty helper: %v", err)
	}
	t.Cleanup(func() {
		_ = tty.Close()
		if cmd.Process != nil && cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
	})

	events := readPTY(tty)
	var transcript bytes.Buffer
	waitForSetupProviderTTYOutput(t, tty, events, &transcript, "Up/Down or j/k navigate")
	if strings.Contains(transcript.String(), "Choice [1-40]") {
		t.Fatalf("setup provider prompt fell back to line input instead of the TTY picker:\n%s", transcript.String())
	}

	if _, err := tty.Write([]byte("\x1b[B")); err != nil {
		t.Fatalf("write arrow key to setup provider picker: %v", err)
	}
	if _, err := tty.Write([]byte("q")); err != nil {
		t.Fatalf("abort setup provider picker: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("setup provider tty helper exited with %v\ntranscript:\n%s", err, transcript.String())
		}
	case <-ctx.Done():
		t.Fatalf("setup provider tty helper timed out\ntranscript:\n%s", transcript.String())
	}
}

func TestSetupToolsTTYE2EConsumesChecklistKeys(t *testing.T) {
	if os.Getenv(setupToolsTTYE2EHelperEnv) == "1" {
		runSetupToolsTTYE2EHelper()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	home := t.TempDir()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSetupToolsTTYE2EConsumesChecklistKeys", "--")
	cmd.Env = append(os.Environ(),
		setupToolsTTYE2EHelperEnv+"=1",
		"GORMES_HOME="+home,
		"HASS_TOKEN=",
		"TERM=xterm-256color",
	)

	tty, err := startLinuxPTY(cmd, 40, 120)
	if err != nil {
		t.Fatalf("start setup tools tty helper: %v", err)
	}
	t.Cleanup(func() {
		_ = tty.Close()
		if cmd.Process != nil && cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
	})

	events := readPTY(tty)
	var transcript bytes.Buffer
	waitForSetupTTYOutput(t, tty, events, &transcript, "SPACE toggle", "Toolsets (comma-separated")

	if _, err := tty.Write([]byte(" \r")); err != nil {
		t.Fatalf("toggle and confirm setup tools checklist: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("setup tools tty helper exited with %v\ntranscript:\n%s", err, transcript.String())
		}
	case <-ctx.Done():
		t.Fatalf("setup tools tty helper timed out\ntranscript:\n%s", transcript.String())
	}

	configBody, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read setup tools config: %v\ntranscript:\n%s", err, transcript.String())
	}
	bodyText := string(configBody)
	if strings.Contains(bodyText, `"web"`) || strings.Contains(bodyText, `'web'`) {
		t.Fatalf("space toggle did not remove the first toolset from persisted config:\n%s\ntranscript:\n%s", string(configBody), transcript.String())
	}
	if !strings.Contains(bodyText, `"browser"`) && !strings.Contains(bodyText, `'browser'`) {
		t.Fatalf("setup tools checklist did not persist the confirmed selection:\n%s\ntranscript:\n%s", string(configBody), transcript.String())
	}
}

func startLinuxPTY(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("open pty master: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = master.Close()
		}
	}()

	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		return nil, fmt.Errorf("unlock pty: %w", err)
	}
	ptyNumber, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		return nil, fmt.Errorf("resolve pty slave: %w", err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", ptyNumber), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("open pty slave: %w", err)
	}
	defer slave.Close()

	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: rows, Col: cols}); err != nil {
		return nil, fmt.Errorf("set pty size: %w", err)
	}
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command with pty: %w", err)
	}

	success = true
	return master, nil
}

func runSetupProviderTTYE2EHelper() {
	seams := (&setupCommandFakeSeams{
		isTTY: true,
		current: cli.ProviderModel{
			Provider: "openai-codex",
			Model:    "gpt-5.5",
		},
	}).seams()
	seams.ChooseSetupProvider = promptSetupProviderChoice

	cmd := &cobra.Command{Use: "setup-provider-tty-e2e-helper"}
	cmd.SetIn(os.Stdin)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	if _, err := runSetupInferenceProviderSection(cmd, seams); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runSetupToolsTTYE2EHelper() {
	cmd := &cobra.Command{Use: "setup-tools-tty-e2e-helper"}
	cmd.SetIn(os.Stdin)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	if err := runSetupToolsSection(cmd, false); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

type ptyReadEvent struct {
	data []byte
	err  error
}

func readPTY(tty *os.File) <-chan ptyReadEvent {
	events := make(chan ptyReadEvent, 16)
	go func() {
		defer close(events)
		buf := make([]byte, 4096)
		for {
			n, err := tty.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				events <- ptyReadEvent{data: chunk}
			}
			if err != nil {
				events <- ptyReadEvent{err: err}
				return
			}
		}
	}()
	return events
}

func waitForSetupProviderTTYOutput(t *testing.T, tty *os.File, events <-chan ptyReadEvent, transcript *bytes.Buffer, want string) {
	waitForSetupTTYOutput(t, tty, events, transcript, want, "Choice [1-40]")
}

func waitForSetupTTYOutput(t *testing.T, tty *os.File, events <-chan ptyReadEvent, transcript *bytes.Buffer, want string, forbidden ...string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	var sentBackground, sentCursor bool
	for {
		output := transcript.String()
		if !sentBackground && (strings.Contains(output, "\x1b]11;?\x07") || strings.Contains(output, "\x1b]11;?\x1b\\")) {
			if _, err := tty.Write([]byte("\x1b]11;rgb:0000/0000/0000\x1b\\")); err != nil {
				t.Fatalf("write background color response to setup provider TTY: %v", err)
			}
			sentBackground = true
		}
		if !sentCursor && strings.Contains(output, "\x1b[6n") {
			if _, err := tty.Write([]byte("\x1b[1;1R")); err != nil {
				t.Fatalf("write cursor position response to setup provider TTY: %v", err)
			}
			sentCursor = true
		}
		if strings.Contains(output, want) {
			return
		}
		for _, needle := range forbidden {
			if strings.Contains(output, needle) {
				t.Fatalf("setup TTY e2e reached fallback prompt %q before %q:\n%s", needle, want, output)
			}
		}

		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("setup provider TTY e2e ended before %q:\n%s", want, transcript.String())
			}
			if len(event.data) > 0 {
				transcript.Write(event.data)
			}
			if event.err != nil && !errors.Is(event.err, io.EOF) && !strings.Contains(event.err.Error(), "input/output error") {
				t.Fatalf("read setup provider TTY output: %v\ntranscript:\n%s", event.err, transcript.String())
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for setup provider TTY output %q:\n%s", want, transcript.String())
		}
	}
}
