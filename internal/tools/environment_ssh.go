package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SSHConfig holds configuration for an SSH environment.
type SSHConfig struct {
	Host    string
	User    string
	CWD     string
	Timeout int
	Port    int
	KeyPath string
}

// SSHEnvironment implements the Environment interface using SSH remote execution.
type SSHEnvironment struct {
	config        SSHConfig
	controlSocket string
	controlDir    string
	remoteHome    string
}

// NewSSHEnvironment creates a new SSH environment and establishes connection.
func NewSSHEnvironment(ctx context.Context, config SSHConfig) (*SSHEnvironment, error) {
	if config.CWD == "" {
		config.CWD = "~"
	}
	if config.Timeout == 0 {
		config.Timeout = 60
	}
	if config.Port == 0 {
		config.Port = 22
	}

	env := &SSHEnvironment{
		config: config,
	}

	// Set up control directory and socket
	env.controlDir = filepath.Join(os.TempDir(), "gormes-ssh")
	if err := os.MkdirAll(env.controlDir, 0755); err != nil {
		return nil, fmt.Errorf("ssh control dir: %w", err)
	}

	// Generate unique socket name based on connection params
	socketID := sha256Hash(fmt.Sprintf("%s@%s:%d", config.User, config.Host, config.Port))
	env.controlSocket = filepath.Join(env.controlDir, socketID[:16]+".sock")

	// Test connection
	if err := env.testConnection(ctx); err != nil {
		return nil, fmt.Errorf("ssh connection test failed: %w", err)
	}

	// Detect remote home directory
	env.remoteHome = env.detectRemoteHome(ctx)
	if env.remoteHome == "" {
		if config.User == "root" {
			env.remoteHome = "/root"
		} else {
			env.remoteHome = "/home/" + config.User
		}
	}

	// Ensure remote directories exist
	if err := env.ensureRemoteDirs(ctx); err != nil {
		return nil, fmt.Errorf("ssh setup remote dirs: %w", err)
	}

	return env, nil
}

// MapPath maps a host path to the remote path.
func (e *SSHEnvironment) MapPath(hostPath string) (string, error) {
	absPath, err := filepath.Abs(hostPath)
	if err != nil {
		return "", err
	}

	// For SSH, we map paths relative to the remote home directory
	cleanPath := filepath.Clean(absPath)
	if strings.HasPrefix(cleanPath, e.remoteHome) {
		rel, _ := filepath.Rel(e.remoteHome, cleanPath)
		return filepath.Join(e.remoteHome, rel), nil
	}

	return cleanPath, nil
}

// Upload copies a file from host to remote.
func (e *SSHEnvironment) Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}

	intent.Direction = FileSyncUpload
	intent.EnvironmentPath = normalizeEnvironmentPath(intent.EnvironmentPath)

	// Ensure parent directory exists
	mkdirCmd := e.buildSSHCommand()
	mkdirCmd = append(mkdirCmd, "mkdir", "-p", filepath.Dir(intent.EnvironmentPath))
	if err := runSSHCommand(ctx, mkdirCmd); err != nil {
		return FileSyncResult{}, fmt.Errorf("ssh mkdir: %w", err)
	}

	// Use scp to copy file
	scpCmd := e.buildSCPCommand(false)
	scpCmd = append(scpCmd, intent.HostPath, e.config.User+"@"+e.config.Host+":"+intent.EnvironmentPath)

	if err := runSCPCommand(ctx, scpCmd); err != nil {
		return FileSyncResult{}, fmt.Errorf("scp upload failed: %w", err)
	}

	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileUploadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "ssh",
			Operation: "upload",
			Resource:  intent.EnvironmentPath,
			Message:   "ssh uploaded file",
		},
	}, nil
}

// Download copies a file from remote to host.
func (e *SSHEnvironment) Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}

	intent.Direction = FileSyncDownload
	intent.EnvironmentPath = normalizeEnvironmentPath(intent.EnvironmentPath)

	// Ensure host directory exists
	hostDir := filepath.Dir(intent.HostPath)
	if err := os.MkdirAll(hostDir, 0755); err != nil {
		return FileSyncResult{}, fmt.Errorf("mkdir for download: %w", err)
	}

	// Use scp to copy file
	scpCmd := e.buildSCPCommand(false)
	scpCmd = append(scpCmd, e.config.User+"@"+e.config.Host+":"+intent.EnvironmentPath, intent.HostPath)

	if err := runSCPCommand(ctx, scpCmd); err != nil {
		return FileSyncResult{}, fmt.Errorf("scp download failed: %w", err)
	}

	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileDownloadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "ssh",
			Operation: "download",
			Resource:  intent.EnvironmentPath,
			Message:   "ssh downloaded file",
		},
	}, nil
}

// Execute runs a command on the remote host via SSH.
func (e *SSHEnvironment) Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentResult{}, err
	}

	if command.Timeout == 0 {
		command.Timeout = time.Duration(e.config.Timeout) * time.Second
	}

	// Build the full bash command with working directory
	fullCmd := command.Command
	if command.WorkingDir != "" {
		fullCmd = fmt.Sprintf("cd %s && %s", shellQuote(command.WorkingDir), command.Command)
	} else if e.config.CWD != "" && e.config.CWD != "~" {
		fullCmd = fmt.Sprintf("cd %s && %s", shellQuote(e.config.CWD), command.Command)
	}

	// Build SSH command
	sshCmd := e.buildSSHCommand()
	sshCmd = append(sshCmd, "bash", "-c", shellQuote(fullCmd))

	// Execute with timeout
	cmd := exec.CommandContext(ctx, sshCmd[0], sshCmd[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return EnvironmentResult{}, ctx.Err()
	case err := <-done:
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		output := stdout.String()
		if stderr.String() != "" {
			output += "\n" + stderr.String()
		}

		return EnvironmentResult{
			Command:  command,
			Output:   output,
			ExitCode: exitCode,
			Evidence: []EnvironmentEvidence{{
				Code:      EnvironmentCommandRecorded,
				Status:    EnvironmentStatusRecorded,
				Backend:   "ssh",
				Operation: "execute",
				Resource:  command.Command,
				Message:   "ssh executed command",
			}},
		}, nil
	}
}

// Cleanup closes the SSH connection and cleans up.
func (e *SSHEnvironment) Cleanup(ctx context.Context) (EnvironmentCleanupResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentCleanupResult{}, err
	}

	evidence := []EnvironmentEvidence{}

	// Close control connection
	exitCmd := exec.Command("ssh",
		"-o", "ControlPath="+e.controlSocket,
		"-O", "exit",
		e.config.User+"@"+e.config.Host,
	)
	exitCmd.Run() // Best effort

	// Remove control socket
	os.Remove(e.controlSocket)

	evidence = append(evidence, EnvironmentEvidence{
		Code:      EnvironmentCleanupRecorded,
		Status:    EnvironmentStatusRecorded,
		Backend:   "ssh",
		Operation: "cleanup",
		Resource:  e.controlSocket,
		Message:   "ssh connection cleaned up",
	})

	return EnvironmentCleanupResult{Evidence: evidence}, nil
}

// testConnection tests the SSH connection.
func (e *SSHEnvironment) testConnection(ctx context.Context) error {
	cmd := e.buildSSHCommand()
	cmd = append(cmd, "echo", "SSH connection established")

	if err := runSSHCommand(ctx, cmd); err != nil {
		return fmt.Errorf("connection test: %w", err)
	}
	return nil
}

// detectRemoteHome detects the remote home directory.
func (e *SSHEnvironment) detectRemoteHome(ctx context.Context) string {
	cmd := e.buildSSHCommand()
	cmd = append(cmd, "echo", "$HOME")

	out, err := runSSHCommandOutput(ctx, cmd)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ensureRemoteDirs ensures the remote .gormes directory tree exists.
func (e *SSHEnvironment) ensureRemoteDirs(ctx context.Context) error {
	base := filepath.Join(e.remoteHome, ".gormes")
	dirs := []string{
		base,
		filepath.Join(base, "skills"),
		filepath.Join(base, "credentials"),
		filepath.Join(base, "cache"),
	}

	mkdirCmd := e.buildSSHCommand()
	mkdirCmd = append(mkdirCmd, "mkdir", "-p")
	mkdirCmd = append(mkdirCmd, dirs...)

	return runSSHCommand(ctx, mkdirCmd)
}

// buildSSHCommand builds the base SSH command with control socket.
func (e *SSHEnvironment) buildSSHCommand() []string {
	cmd := []string{"ssh",
		"-o", "ControlPath=" + e.controlSocket,
		"-o", "ControlMaster=auto",
		"-o", "ControlPersist=300",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
	}

	if e.config.Port != 22 {
		cmd = append(cmd, "-p", strconv.Itoa(e.config.Port))
	}

	if e.config.KeyPath != "" {
		cmd = append(cmd, "-i", e.config.KeyPath)
	}

	cmd = append(cmd, e.config.User+"@"+e.config.Host)
	return cmd
}

// buildSCPCommand builds the base SCP command with control socket.
func (e *SSHEnvironment) buildSCPCommand(recursive bool) []string {
	cmd := []string{"scp",
		"-o", "ControlPath=" + e.controlSocket,
		"-o", "BatchMode=yes",
	}

	if recursive {
		cmd = append(cmd, "-r")
	}

	if e.config.Port != 22 {
		cmd = append(cmd, "-P", strconv.Itoa(e.config.Port))
	}

	if e.config.KeyPath != "" {
		cmd = append(cmd, "-i", e.config.KeyPath)
	}

	return cmd
}

// runSSHCommand runs an SSH command and returns an error if it fails.
func runSSHCommand(ctx context.Context, cmd []string) error {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stderr bytes.Buffer
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %s: %w", cmd[0], stderr.String(), err)
	}
	return nil
}

// runSSHCommandOutput runs an SSH command and returns the stdout output.
func runSSHCommandOutput(ctx context.Context, cmd []string) (string, error) {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%s: %s: %w", cmd[0], stderr.String(), err)
	}
	return stdout.String(), nil
}

// runSCPCommand runs an SCP command.
func runSCPCommand(ctx context.Context, cmd []string) error {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stderr bytes.Buffer
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %s: %w", cmd[0], stderr.String(), err)
	}
	return nil
}

// sha256Hash returns a hex-encoded SHA-256 hash of the input string.
func sha256Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// shellQuote quotes a string for shell safety.
func shellQuote(s string) string {
	// Simple quoting - use single quotes and escape any single quotes
	s = strings.ReplaceAll(s, "'", "'\\''")
	return "'" + s + "'"
}

// SSHProvider is a factory for creating SSH environments.
type SSHProvider struct{}

// NewSSHProvider creates a new SSHProvider.
func NewSSHProvider() *SSHProvider {
	return &SSHProvider{}
}

// Create creates a new SSH environment with the given configuration.
func (p *SSHProvider) Create(ctx context.Context, config SSHConfig) (Environment, error) {
	return NewSSHEnvironment(ctx, config)
}

var _ Environment = (*SSHEnvironment)(nil)

// SSHEnvironmentFromConfig creates an SSH environment from a JSON configuration.
func SSHEnvironmentFromConfig(ctx context.Context, configJSON json.RawMessage) (Environment, error) {
	var cfg SSHConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("ssh config: %w", err)
	}
	return NewSSHEnvironment(ctx, cfg)
}

// BulkUpload uploads multiple files using tar over SSH.
func (e *SSHEnvironment) BulkUpload(ctx context.Context, files []FileSyncIntent) error {
	if len(files) == 0 {
		return nil
	}

	// Create tar archive in memory, stream over SSH
	pr, pw := io.Pipe()

	// Start tar process
	tarCmd := exec.CommandContext(ctx, "tar", "-cf", "-")
	tarCmd.Args = append(tarCmd.Args, "-C", filepath.Dir(files[0].HostPath))
	for _, f := range files {
		tarCmd.Args = append(tarCmd.Args, filepath.Base(f.HostPath))
	}
	tarCmd.Stdout = pw

	// Start SSH process
	sshCmd := e.buildSSHCommand()
	sshCmd = append(sshCmd, "tar", "xf", "-", "-C", "/")

	sshProcess := exec.CommandContext(ctx, sshCmd[0], sshCmd[1:]...)
	sshProcess.Stdin = pr

	var tarErr, sshErr bytes.Buffer
	tarCmd.Stderr = &tarErr
	sshProcess.Stderr = &sshErr

	if err := tarCmd.Start(); err != nil {
		return fmt.Errorf("tar start: %w", err)
	}
	if err := sshProcess.Start(); err != nil {
		return fmt.Errorf("ssh start: %w", err)
	}

	tarErrCh := make(chan error, 1)
	sshErrCh := make(chan error, 1)

	go func() {
		tarErrCh <- tarCmd.Wait()
	}()
	go func() {
		sshErrCh <- sshProcess.Wait()
	}()

	select {
	case <-ctx.Done():
		if tarCmd.Process != nil {
			tarCmd.Process.Kill()
		}
		if sshProcess.Process != nil {
			sshProcess.Process.Kill()
		}
		return ctx.Err()
	case err := <-tarErrCh:
		if err != nil {
			return fmt.Errorf("tar failed: %s: %w", tarErr.String(), err)
		}
	case err := <-sshErrCh:
		if err != nil {
			return fmt.Errorf("ssh failed: %s: %w", sshErr.String(), err)
		}
	}

	return nil
}
