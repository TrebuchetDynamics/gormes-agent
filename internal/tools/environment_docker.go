package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DockerConfig holds configuration for a Docker environment.
type DockerConfig struct {
	Image                    string
	CWD                      string
	Timeout                  int
	CPU                      float64
	MemoryMB                 int
	DiskMB                   int
	PersistentFilesystem     bool
	TaskID                   string
	Volumes                  []string
	ForwardEnv               []string
	Env                      map[string]string
	Network                  bool
	HostCWD                  string
	AutoMountCWD             bool
	RunAsHostUser            bool
	DockerExecutable         string
}

// DockerEnvironment implements the Environment interface using Docker containers.
type DockerEnvironment struct {
	config       DockerConfig
	containerID  string
	dockerExe   string
	workspaceDir string
	homeDir      string
}

// NewDockerEnvironment creates a new Docker environment and starts a container.
func NewDockerEnvironment(ctx context.Context, config DockerConfig) (*DockerEnvironment, error) {
	if config.CWD == "" {
		config.CWD = "/root"
	}
	if config.CWD == "~" {
		config.CWD = "/root"
	}
	if config.Timeout == 0 {
		config.Timeout = 60
	}
	if config.TaskID == "" {
		config.TaskID = "default"
	}

	env := &DockerEnvironment{
		config: config,
	}

	// Find docker executable
	env.dockerExe = findDockerExecutable(config.DockerExecutable)

	// Ensure docker is available
	if err := ensureDockerAvailable(env.dockerExe); err != nil {
		return nil, fmt.Errorf("docker environment: %w", err)
	}

	// Create the container
	if err := env.createContainer(ctx); err != nil {
		return nil, fmt.Errorf("docker environment: failed to create container: %w", err)
	}

	return env, nil
}

// MapPath maps a host path to the container path.
func (e *DockerEnvironment) MapPath(hostPath string) (string, error) {
	// For Docker, paths inside the container are absolute
	// The host path is mapped based on workspace/home directories
	absPath, err := filepath.Abs(hostPath)
	if err != nil {
		return "", err
	}
	
	// If the path is under the workspace or home mount, map it
	cleanPath := filepath.Clean(absPath)
	
	if e.workspaceDir != "" && strings.HasPrefix(cleanPath, e.workspaceDir) {
		rel, _ := filepath.Rel(e.workspaceDir, cleanPath)
		return filepath.Join("/workspace", rel), nil
	}
	
	if e.homeDir != "" && strings.HasPrefix(cleanPath, e.homeDir) {
		rel, _ := filepath.Rel(e.homeDir, cleanPath)
		return filepath.Join("/root", rel), nil
	}
	
	// Default: return the absolute path as-is
	return cleanPath, nil
}

// Upload copies a file from host to container.
func (e *DockerEnvironment) Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}

	intent.Direction = FileSyncUpload
	intent.EnvironmentPath = normalizeEnvironmentPath(intent.EnvironmentPath)

	// Use docker cp to copy file into container
	cmd := exec.CommandContext(ctx, e.dockerExe, "cp", intent.HostPath, e.containerID+":"+intent.EnvironmentPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return FileSyncResult{}, fmt.Errorf("docker cp failed: %s: %w", stderr.String(), err)
	}

	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileUploadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "docker",
			Operation: "upload",
			Resource:  intent.EnvironmentPath,
			Message:   "docker uploaded file",
		},
	}, nil
}

// Download copies a file from container to host.
func (e *DockerEnvironment) Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}

	intent.Direction = FileSyncDownload
	intent.EnvironmentPath = normalizeEnvironmentPath(intent.EnvironmentPath)

	// Create parent directory on host
	hostDir := filepath.Dir(intent.HostPath)
	if err := os.MkdirAll(hostDir, 0755); err != nil {
		return FileSyncResult{}, fmt.Errorf("mkdir for docker download: %w", err)
	}

	// Use docker cp to copy file from container
	cmd := exec.CommandContext(ctx, e.dockerExe, "cp", e.containerID+":"+intent.EnvironmentPath, intent.HostPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return FileSyncResult{}, fmt.Errorf("docker cp download failed: %s: %w", stderr.String(), err)
	}

	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileDownloadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "docker",
			Operation: "download",
			Resource:  intent.EnvironmentPath,
			Message:   "docker downloaded file",
		},
	}, nil
}

// Execute runs a command inside the Docker container.
func (e *DockerEnvironment) Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentResult{}, err
	}

	if command.Timeout == 0 {
		command.Timeout = time.Duration(e.config.Timeout) * time.Second
	}

	// Build docker exec command
	args := []string{"exec", "-i"}
	
	if command.WorkingDir != "" {
		args = append(args, "-w", command.WorkingDir)
	} else if e.config.CWD != "" {
		args = append(args, "-w", e.config.CWD)
	}

	// Add environment variables
	for key, val := range e.config.Env {
		args = append(args, "-e", key+"="+val)
	}

	args = append(args, e.containerID, "bash", "-c", command.Command)

	cmd := exec.CommandContext(ctx, e.dockerExe, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return EnvironmentResult{}, ctx.Err()
	case err := <-done:
		if err != nil {
			// Check if it's a timeout
			if ctx.Err() == context.DeadlineExceeded {
				return EnvironmentResult{
					Command:  command,
					Output:   stdout.String() + "\n[Command timed out after " + command.Timeout.String() + "]",
					ExitCode: 124,
					Evidence: []EnvironmentEvidence{{
						Code:      EnvironmentCommandRecorded,
						Status:    EnvironmentStatusRecorded,
						Backend:   "docker",
						Operation: "execute",
						Resource:  command.Command,
						Message:   "command timed out",
					}},
				}, nil
			}
		}

		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		return EnvironmentResult{
			Command:  command,
			Output:   stdout.String() + stderr.String(),
			ExitCode: exitCode,
			Evidence: []EnvironmentEvidence{{
				Code:      EnvironmentCommandRecorded,
				Status:    EnvironmentStatusRecorded,
				Backend:   "docker",
				Operation: "execute",
				Resource:  command.Command,
				Message:   "docker executed command",
			}},
		}, nil
	}
}

// Cleanup stops and removes the Docker container.
func (e *DockerEnvironment) Cleanup(ctx context.Context) (EnvironmentCleanupResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentCleanupResult{}, err
	}

	evidence := []EnvironmentEvidence{}

	// Stop and remove container
	stopCmd := exec.CommandContext(ctx, e.dockerExe, "stop", "-t", "10", e.containerID)
	stopCmd.Run() // Best effort

	rmCmd := exec.CommandContext(ctx, e.dockerExe, "rm", "-f", e.containerID)
	rmCmd.Run() // Best effort

	// Clean up temp directories if not persistent
	if !e.config.PersistentFilesystem {
		for _, d := range []string{e.workspaceDir, e.homeDir} {
			if d != "" {
				os.RemoveAll(d)
			}
		}
	}

	evidence = append(evidence, EnvironmentEvidence{
		Code:      EnvironmentCleanupRecorded,
		Status:    EnvironmentStatusRecorded,
		Backend:   "docker",
		Operation: "cleanup",
		Resource:  e.containerID,
		Message:   "docker container cleaned up",
	})

	return EnvironmentCleanupResult{Evidence: evidence}, nil
}

// createContainer creates and starts the Docker container.
func (e *DockerEnvironment) createContainer(ctx context.Context) error {
	args := []string{
		"run", "-d",
		"--init",
		"--name", e.containerID,
		"-w", e.config.CWD,
	}

	// Security: drop all capabilities, no privilege escalation
	args = append(args, "--cap-drop", "ALL")
	args = append(args, "--cap-add", "DAC_OVERRIDE")  // Root can write to bind-mounted dirs
	args = append(args, "--cap-add", "CHOWN")
	args = append(args, "--cap-add", "FOWNER")
	args = append(args, "--security-opt", "no-new-privileges")
	args = append(args, "--pids-limit", "256")

	// Tmpfs for scratch directories
	args = append(args, "--tmpfs", "/tmp:rw,nosuid,size=512m")
	args = append(args, "--tmpfs", "/var/tmp:rw,noexec,nosuid,size=256m")
	args = append(args, "--tmpfs", "/run:rw,noexec,nosuid,size=64m")

	// Add Gosu caps if not running as host user
	if !e.config.RunAsHostUser {
		args = append(args, "--cap-add", "SETUID")
		args = append(args, "--cap-add", "SETGID")
	}

	// Resource limits
	if e.config.CPU > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(e.config.CPU, 'f', 2, 64))
	}
	if e.config.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", e.config.MemoryMB))
	}

	// Network
	if !e.config.Network {
		args = append(args, "--network=none")
	}

	// User mapping
	if e.config.RunAsHostUser {
		if uid := os.Getuid(); uid >= 0 {
			args = append(args, "--user", fmt.Sprintf("%d:%d", uid, os.Getgid()))
		}
	}

	// Volume mounts
	if e.config.Volumes != nil {
		for _, vol := range e.config.Volumes {
			if strings.Contains(vol, ":") {
				args = append(args, "-v", vol)
			}
		}
	}

	// Workspace and home directories
	if e.config.PersistentFilesystem {
		sandboxBase := os.ExpandEnv("$HOME/.gormes/sandboxes/docker/" + e.config.TaskID)
		e.homeDir = filepath.Join(sandboxBase, "home")
		e.workspaceDir = filepath.Join(sandboxBase, "workspace")
		
		os.MkdirAll(e.homeDir, 0755)
		os.MkdirAll(e.workspaceDir, 0755)
		
		args = append(args, "-v", e.homeDir+":/root")
		if !e.config.AutoMountCWD {
			args = append(args, "-v", e.workspaceDir+":/workspace")
		}
	} else {
		args = append(args, "--tmpfs", "/workspace:rw,exec,size=10g")
		args = append(args, "--tmpfs", "/home:rw,exec,size=1g")
		args = append(args, "--tmpfs", "/root:rw,exec,size=1g")
	}

	// Environment variables
	for key, val := range e.config.Env {
		args = append(args, "-e", key+"="+val)
	}

	args = append(args, e.config.Image, "sleep", "infinity")

	// Generate container name
	e.containerID = "gormes-" + randomID(8)

	// Rebuild command with name
	cmdArgs := make([]string, len(args))
	copy(cmdArgs, args)
	for i, arg := range cmdArgs {
		if arg == "--name" && i+1 < len(cmdArgs) {
			cmdArgs[i+1] = e.containerID
			break
		}
	}

	cmd := exec.CommandContext(ctx, e.dockerExe, cmdArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %s: %w", stderr.String(), err)
	}

	// Store actual container ID returned by docker
	idCmd := exec.CommandContext(ctx, e.dockerExe, "inspect", "-f", "{{.Id}}", e.containerID)
	idOut, err := idCmd.Output()
	if err == nil {
		e.containerID = strings.TrimSpace(string(idOut))
	}

	return nil
}

// DockerProvider is a factory for creating Docker environments.
type DockerProvider struct{}

// NewDockerProvider creates a new DockerProvider.
func NewDockerProvider() *DockerProvider {
	return &DockerProvider{}
}

// Create creates a new Docker environment with the given configuration.
func (p *DockerProvider) Create(ctx context.Context, config DockerConfig) (Environment, error) {
	return NewDockerEnvironment(ctx, config)
}

// findDockerExecutable finds the docker executable path.
func findDockerExecutable(override string) string {
	if override != "" {
		if _, err := os.Stat(override); err == nil {
			return override
		}
	}

	// Check PATH
	paths := []string{
		"/usr/local/bin/docker",
		"/opt/homebrew/bin/docker",
		"/usr/bin/docker",
		"/usr/bin/podman",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to whatever is in PATH
	if p, err := exec.LookPath("docker"); err == nil {
		return p
	}
	if p, err := exec.LookPath("podman"); err == nil {
		return p
	}

	return "docker" // Fallback, will fail with helpful error
}

// ensureDockerAvailable checks if docker is available and running.
func ensureDockerAvailable(dockerExe string) error {
	cmd := exec.Command(dockerExe, "version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %s", stderr.String())
	}
	return nil
}

// randomID generates a random hex string of given length.
func randomID(n int) string {
	const letters = "0123456789abcdef"
	b := make([]byte, n)
	osrand := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[osrand%16]
		osrand = osrand/16 + int64(i)
	}
	return string(b)
}

// envVarNameRE validates environment variable names.
var envVarNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// normalizeEnvVars validates and normalizes environment variable names.
func normalizeEnvVars(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string)
	for key, val := range env {
		key = strings.TrimSpace(key)
		if !envVarNameRE.MatchString(key) {
			continue
		}
		result[key] = val
	}
	return result
}

var _ Environment = (*DockerEnvironment)(nil)

// DockerEnvironmentFromConfig creates a Docker environment from a JSON configuration.
func DockerEnvironmentFromConfig(ctx context.Context, configJSON json.RawMessage) (Environment, error) {
	var cfg DockerConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("docker config: %w", err)
	}
	cfg.Env = normalizeEnvVars(cfg.Env)
	return NewDockerEnvironment(ctx, cfg)
}
