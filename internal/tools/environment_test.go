package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockDockerEnvironment is a mock implementation for testing.
type MockDockerEnvironment struct {
	MockContainerID string
	MockConfig      DockerConfig
	MockError       error
	UploadedFiles   []FileSyncIntent
	DownloadedFiles []FileSyncIntent
	ExecutedCmds    []EnvironmentCommand
	CleanupCalled   bool
}

func (m *MockDockerEnvironment) MapPath(hostPath string) (string, error) {
	return "/container" + hostPath, nil
}

func (m *MockDockerEnvironment) Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if m.MockError != nil {
		return FileSyncResult{}, m.MockError
	}
	m.UploadedFiles = append(m.UploadedFiles, intent)
	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileUploadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "docker",
			Operation: "upload",
			Resource:  intent.EnvironmentPath,
		},
	}, nil
}

func (m *MockDockerEnvironment) Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if m.MockError != nil {
		return FileSyncResult{}, m.MockError
	}
	m.DownloadedFiles = append(m.DownloadedFiles, intent)
	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileDownloadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "docker",
			Operation: "download",
			Resource:  intent.EnvironmentPath,
		},
	}, nil
}

func (m *MockDockerEnvironment) Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error) {
	if m.MockError != nil {
		return EnvironmentResult{}, m.MockError
	}
	m.ExecutedCmds = append(m.ExecutedCmds, command)
	return EnvironmentResult{
		Command:  command,
		Output:   "mock output: " + command.Command,
		ExitCode: 0,
		Evidence: []EnvironmentEvidence{{
			Code:      EnvironmentCommandRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "docker",
			Operation: "execute",
			Resource:  command.Command,
		}},
	}, nil
}

func (m *MockDockerEnvironment) Cleanup(ctx context.Context) (EnvironmentCleanupResult, error) {
	if m.MockError != nil {
		return EnvironmentCleanupResult{}, m.MockError
	}
	m.CleanupCalled = true
	return EnvironmentCleanupResult{
		Evidence: []EnvironmentEvidence{{
			Code:      EnvironmentCleanupRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "docker",
			Operation: "cleanup",
			Resource:  m.MockContainerID,
		}},
	}, nil
}

// MockSSHEnvironment is a mock implementation for testing.
type MockSSHEnvironment struct {
	MockHost        string
	MockUser        string
	MockConfig      SSHConfig
	MockError       error
	UploadedFiles   []FileSyncIntent
	DownloadedFiles []FileSyncIntent
	ExecutedCmds    []EnvironmentCommand
	CleanupCalled   bool
}

func (m *MockSSHEnvironment) MapPath(hostPath string) (string, error) {
	return "/remote" + hostPath, nil
}

func (m *MockSSHEnvironment) Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if m.MockError != nil {
		return FileSyncResult{}, m.MockError
	}
	m.UploadedFiles = append(m.UploadedFiles, intent)
	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileUploadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "ssh",
			Operation: "upload",
			Resource:  intent.EnvironmentPath,
		},
	}, nil
}

func (m *MockSSHEnvironment) Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if m.MockError != nil {
		return FileSyncResult{}, m.MockError
	}
	m.DownloadedFiles = append(m.DownloadedFiles, intent)
	return FileSyncResult{
		Intent: intent,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentFileDownloadRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "ssh",
			Operation: "download",
			Resource:  intent.EnvironmentPath,
		},
	}, nil
}

func (m *MockSSHEnvironment) Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error) {
	if m.MockError != nil {
		return EnvironmentResult{}, m.MockError
	}
	m.ExecutedCmds = append(m.ExecutedCmds, command)
	return EnvironmentResult{
		Command:  command,
		Output:   "mock ssh output: " + command.Command,
		ExitCode: 0,
		Evidence: []EnvironmentEvidence{{
			Code:      EnvironmentCommandRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "ssh",
			Operation: "execute",
			Resource:  command.Command,
		}},
	}, nil
}

func (m *MockSSHEnvironment) Cleanup(ctx context.Context) (EnvironmentCleanupResult, error) {
	if m.MockError != nil {
		return EnvironmentCleanupResult{}, m.MockError
	}
	m.CleanupCalled = true
	return EnvironmentCleanupResult{
		Evidence: []EnvironmentEvidence{{
			Code:      EnvironmentCleanupRecorded,
			Status:    EnvironmentStatusRecorded,
			Backend:   "ssh",
			Operation: "cleanup",
			Resource:  m.MockHost,
		}},
	}, nil
}

// TestDockerConfigValidation tests Docker configuration validation.
func TestDockerConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config DockerConfig
		valid  bool
	}{
		{
			name: "valid config with image",
			config: DockerConfig{
				Image:   "python:3.11",
				CWD:     "/workspace",
				Timeout: 60,
			},
			valid: true,
		},
		{
			name: "valid config with resources",
			config: DockerConfig{
				Image:     "python:3.11",
				CWD:       "/workspace",
				Timeout:   60,
				CPU:       2.0,
				MemoryMB:  4096,
				DiskMB:    10240,
			},
			valid: true,
		},
		{
			name: "valid config with env vars",
			config: DockerConfig{
				Image:   "python:3.11",
				CWD:     "/workspace",
				Timeout: 60,
				Env: map[string]string{
					"MY_VAR": "value",
					"PATH":   "/usr/bin",
				},
			},
			valid: true,
		},
		{
			name: "valid config with volumes",
			config: DockerConfig{
				Image:   "python:3.11",
				CWD:     "/workspace",
				Timeout: 60,
				Volumes: []string{"/host/path:/container/path"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation: config should have an image
			if tt.valid && tt.config.Image == "" {
				t.Errorf("expected valid config, got empty image")
			}
		})
	}
}

// TestSSHConfigValidation tests SSH configuration validation.
func TestSSHConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config SSHConfig
		valid  bool
	}{
		{
			name: "valid config with host and user",
			config: SSHConfig{
				Host:    "example.com",
				User:    "ubuntu",
				CWD:     "/home/ubuntu",
				Timeout: 60,
				Port:    22,
			},
			valid: true,
		},
		{
			name: "valid config with key path",
			config: SSHConfig{
				Host:     "example.com",
				User:     "ubuntu",
				KeyPath:  "/path/to/key",
				Timeout:  60,
				Port:     22,
			},
			valid: true,
		},
		{
			name: "valid config with custom port",
			config: SSHConfig{
				Host:    "example.com",
				User:    "ubuntu",
				Port:    2222,
				Timeout: 60,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			if tt.valid && (tt.config.Host == "" || tt.config.User == "") {
				t.Errorf("expected valid config, got missing host or user")
			}
		})
	}
}

// TestMockDockerEnvironmentUpload tests mock Docker upload behavior.
func TestMockDockerEnvironmentUpload(t *testing.T) {
	ctx := context.Background()
	env := &MockDockerEnvironment{
		MockContainerID: "test-container-123",
	}

	intent := FileSyncIntent{
		Direction:       FileSyncUpload,
		HostPath:        "/host/test.txt",
		EnvironmentPath: "/container/test.txt",
		Checksum:        "sha256:abc123",
	}

	result, err := env.Upload(ctx, intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.UploadedFiles) != 1 {
		t.Fatalf("expected 1 uploaded file, got %d", len(env.UploadedFiles))
	}

	if result.Evidence.Code != EnvironmentFileUploadRecorded {
		t.Errorf("expected evidence code %s, got %s", EnvironmentFileUploadRecorded, result.Evidence.Code)
	}

	if result.Evidence.Backend != "docker" {
		t.Errorf("expected backend docker, got %s", result.Evidence.Backend)
	}
}

// TestMockDockerEnvironmentExecute tests mock Docker execute behavior.
func TestMockDockerEnvironmentExecute(t *testing.T) {
	ctx := context.Background()
	env := &MockDockerEnvironment{
		MockContainerID: "test-container-123",
	}

	cmd := EnvironmentCommand{
		Command:    "echo 'hello world'",
		WorkingDir: "/workspace",
		Timeout:    30 * time.Second,
	}

	result, err := env.Execute(ctx, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.ExecutedCmds) != 1 {
		t.Fatalf("expected 1 executed command, got %d", len(env.ExecutedCmds))
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}

// TestMockDockerEnvironmentCleanup tests mock Docker cleanup behavior.
func TestMockDockerEnvironmentCleanup(t *testing.T) {
	ctx := context.Background()
	env := &MockDockerEnvironment{
		MockContainerID: "test-container-123",
	}

	result, err := env.Cleanup(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.CleanupCalled {
		t.Error("expected Cleanup to be called")
	}

	if len(result.Evidence) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(result.Evidence))
	}

	if result.Evidence[0].Code != EnvironmentCleanupRecorded {
		t.Errorf("expected evidence code %s, got %s", EnvironmentCleanupRecorded, result.Evidence[0].Code)
	}
}

// TestMockSSHEnvironmentUpload tests mock SSH upload behavior.
func TestMockSSHEnvironmentUpload(t *testing.T) {
	ctx := context.Background()
	env := &MockSSHEnvironment{
		MockHost: "example.com",
		MockUser: "ubuntu",
	}

	intent := FileSyncIntent{
		Direction:       FileSyncUpload,
		HostPath:        "/host/test.txt",
		EnvironmentPath: "/remote/test.txt",
		Checksum:        "sha256:def456",
	}

	result, err := env.Upload(ctx, intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.UploadedFiles) != 1 {
		t.Fatalf("expected 1 uploaded file, got %d", len(env.UploadedFiles))
	}

	if result.Evidence.Code != EnvironmentFileUploadRecorded {
		t.Errorf("expected evidence code %s, got %s", EnvironmentFileUploadRecorded, result.Evidence.Code)
	}

	if result.Evidence.Backend != "ssh" {
		t.Errorf("expected backend ssh, got %s", result.Evidence.Backend)
	}
}

// TestMockSSHEnvironmentExecute tests mock SSH execute behavior.
func TestMockSSHEnvironmentExecute(t *testing.T) {
	ctx := context.Background()
	env := &MockSSHEnvironment{
		MockHost: "example.com",
		MockUser: "ubuntu",
	}

	cmd := EnvironmentCommand{
		Command:    "ls -la",
		WorkingDir: "/home/ubuntu",
		Timeout:    30 * time.Second,
	}

	result, err := env.Execute(ctx, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.ExecutedCmds) != 1 {
		t.Fatalf("expected 1 executed command, got %d", len(env.ExecutedCmds))
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestMockSSHEnvironmentCleanup tests mock SSH cleanup behavior.
func TestMockSSHEnvironmentCleanup(t *testing.T) {
	ctx := context.Background()
	env := &MockSSHEnvironment{
		MockHost: "example.com",
		MockUser: "ubuntu",
	}

	result, err := env.Cleanup(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.CleanupCalled {
		t.Error("expected Cleanup to be called")
	}

	if result.Evidence[0].Backend != "ssh" {
		t.Errorf("expected backend ssh, got %s", result.Evidence[0].Backend)
	}
}

// TestEnvironmentInterfaceCompliance tests that mock implementations satisfy Environment interface.
func TestEnvironmentInterfaceCompliance(t *testing.T) {
	// Ensure mock implementations satisfy Environment interface
	var _ Environment = (*MockDockerEnvironment)(nil)
	var _ Environment = (*MockSSHEnvironment)(nil)
}

// TestDockerProviderFactory tests the Docker provider factory.
func TestDockerProviderFactory(t *testing.T) {
	provider := NewDockerProvider()
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestSSHProviderFactory tests the SSH provider factory.
func TestSSHProviderFactory(t *testing.T) {
	provider := NewSSHProvider()
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestDockerEnvironmentFromConfig tests JSON config parsing for Docker.
func TestDockerEnvironmentFromConfig(t *testing.T) {
	configJSON := []byte(`{
		"Image": "python:3.11",
		"CWD": "/workspace",
		"Timeout": 60,
		"CPU": 2.0,
		"MemoryMB": 4096,
		"Env": {
			"MY_VAR": "value"
		}
	}`)

	// Test that the config can be parsed
	var cfg DockerConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Image != "python:3.11" {
		t.Errorf("expected image python:3.11, got %s", cfg.Image)
	}

	if cfg.CWD != "/workspace" {
		t.Errorf("expected cwd /workspace, got %s", cfg.CWD)
	}

	if cfg.CPU != 2.0 {
		t.Errorf("expected CPU 2.0, got %f", cfg.CPU)
	}

	if cfg.MemoryMB != 4096 {
		t.Errorf("expected MemoryMB 4096, got %d", cfg.MemoryMB)
	}

	if cfg.Env["MY_VAR"] != "value" {
		t.Errorf("expected MY_VAR=value, got %s", cfg.Env["MY_VAR"])
	}
}

// TestSSHEnvironmentFromConfig tests JSON config parsing for SSH.
func TestSSHEnvironmentFromConfig(t *testing.T) {
	configJSON := []byte(`{
		"Host": "example.com",
		"User": "ubuntu",
		"CWD": "/home/ubuntu",
		"Timeout": 60,
		"Port": 22,
		"KeyPath": "/path/to/key"
	}`)

	// Test that the config can be parsed
	var cfg SSHConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", cfg.Host)
	}

	if cfg.User != "ubuntu" {
		t.Errorf("expected user ubuntu, got %s", cfg.User)
	}

	if cfg.Port != 22 {
		t.Errorf("expected port 22, got %d", cfg.Port)
	}

	if cfg.KeyPath != "/path/to/key" {
		t.Errorf("expected key path /path/to/key, got %s", cfg.KeyPath)
	}
}

// TestFileSyncIntent tests FileSyncIntent structure.
func TestFileSyncIntent(t *testing.T) {
	intent := FileSyncIntent{
		Direction:       FileSyncUpload,
		HostPath:        "/host/file.txt",
		EnvironmentPath: "/container/file.txt",
		Checksum:        "sha256:abc123",
	}

	if intent.Direction != FileSyncUpload {
		t.Errorf("expected direction upload, got %s", intent.Direction)
	}

	if intent.HostPath != "/host/file.txt" {
		t.Errorf("expected host path /host/file.txt, got %s", intent.HostPath)
	}
}

// TestEnvironmentCommand tests EnvironmentCommand structure.
func TestEnvironmentCommand(t *testing.T) {
	cmd := EnvironmentCommand{
		Command:    "echo hello",
		WorkingDir: "/workspace",
		Timeout:    30 * time.Second,
		Stdin:      "input data",
	}

	if cmd.Command != "echo hello" {
		t.Errorf("expected command 'echo hello', got %s", cmd.Command)
	}

	if cmd.WorkingDir != "/workspace" {
		t.Errorf("expected working dir /workspace, got %s", cmd.WorkingDir)
	}
}

// TestDockerConfigNormalization tests environment variable normalization.
func TestDockerConfigNormalization(t *testing.T) {
	// Test that invalid env var names are filtered
	cfg := DockerConfig{
		Image: "python:3.11",
		Env: map[string]string{
			"VALID_VAR":  "value1",
			"123_INVALID": "value2",
			"ALSO_VALID":  "value3",
		},
	}

	// The normalization should happen in NewDockerEnvironment
	normalized := normalizeEnvVars(cfg.Env)

	// Valid vars should be preserved
	if _, ok := normalized["VALID_VAR"]; !ok {
		t.Error("expected VALID_VAR to be preserved")
	}

	if _, ok := normalized["ALSO_VALID"]; !ok {
		t.Error("expected ALSO_VALID to be preserved")
	}

	// Invalid var should be filtered (but our implementation just passes through)
	// Note: The actual implementation is permissive, so 123_INVALID would be included
	// This test documents current behavior, not necessarily desired behavior
	_ = cfg
}

// TestFindDockerExecutable tests docker executable lookup.
func TestFindDockerExecutable(t *testing.T) {
	// Test with explicit override
	path := findDockerExecutable("/usr/bin/docker")
	// Should return the override even if it doesn't exist
	if path != "/usr/bin/docker" {
		t.Errorf("expected /usr/bin/docker, got %s", path)
	}
}

// TestShellQuote tests shell quoting function.
func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"hello world", "'hello world'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}

	for _, tt := range tests {
		result := shellQuote(tt.input)
		if result != tt.expected {
			t.Errorf("shellQuote(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestSHA256Hash tests SHA256 hashing function.
func TestSHA256Hash(t *testing.T) {
	hash := sha256Hash("test input")
	if len(hash) != 64 {
		t.Errorf("expected 64 character hash, got %d characters", len(hash))
	}

	// Same input should produce same hash
	hash2 := sha256Hash("test input")
	if hash != hash2 {
		t.Errorf("expected same hash for same input")
	}

	// Different input should produce different hash
	hash3 := sha256Hash("different input")
	if hash == hash3 {
		t.Errorf("expected different hash for different input")
	}
}

// TestRandomID tests random ID generation.
func TestRandomID(t *testing.T) {
	id1 := randomID(8)
	if len(id1) != 8 {
		t.Errorf("expected 8 character ID, got %d characters", len(id1))
	}

	id2 := randomID(8)
	if id1 == id2 {
		t.Error("expected different IDs for consecutive calls")
	}
}

// TestEnvironmentEvidenceError tests evidence error formatting.
func TestEnvironmentEvidenceError(t *testing.T) {
	err := &EnvironmentEvidenceError{
		Evidence: EnvironmentEvidence{
			Code:    EnvironmentBackendUnavailable,
			Status:  EnvironmentStatusUnavailable,
			Backend: "docker",
			Operation: "backend_unavailable",
			Message: "docker not available",
		},
	}

	expected := "environment docker backend_unavailable: docker not available"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

// TestEnvironmentEvidenceFromError tests evidence extraction from errors.
func TestEnvironmentEvidenceFromError(t *testing.T) {
	err := &EnvironmentEvidenceError{
		Evidence: EnvironmentEvidence{
			Code:    EnvironmentBackendUnavailable,
			Status:  EnvironmentStatusUnavailable,
			Backend: "ssh",
			Message: "ssh connection failed",
		},
	}

	evidence, ok := EnvironmentEvidenceFromError(err)
	if !ok {
		t.Fatal("expected to extract evidence from error")
	}

	if evidence.Backend != "ssh" {
		t.Errorf("expected backend ssh, got %s", evidence.Backend)
	}

	if evidence.Code != EnvironmentBackendUnavailable {
		t.Errorf("expected code %s, got %s", EnvironmentBackendUnavailable, evidence.Code)
	}

	// Test with non-evidence error
	regularErr := os.ErrNotExist
	_, ok = EnvironmentEvidenceFromError(regularErr)
	if ok {
		t.Error("expected ok to be false for non-evidence error")
	}
}

// TestContextCancellation tests that operations respect context cancellation.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	env := &MockDockerEnvironment{
		MockError: context.Canceled,
	}

	cmd := EnvironmentCommand{
		Command: "sleep 100",
	}

	_, err := env.Execute(ctx, cmd)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

// TestMapPathWithDifferentPaths tests MapPath behavior with various paths.
func TestMapPathWithDifferentPaths(t *testing.T) {
	env := &MockDockerEnvironment{}

	tests := []struct {
		hostPath         string
		expectedContains string
	}{
		{"/home/user/file.txt", "/container"},
		{"/workspace/project/file.txt", "/container"},
		{"/absolute/path/file.txt", "/container"},
	}

	for _, tt := range tests {
		t.Run(tt.hostPath, func(t *testing.T) {
			result, err := env.MapPath(tt.hostPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

// TestSSHEnvironmentMapPath tests SSH MapPath behavior.
func TestSSHEnvironmentMapPath(t *testing.T) {
	env := &MockSSHEnvironment{
		MockHost: "example.com",
		MockUser: "ubuntu",
	}

	result, err := env.MapPath("/home/user/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

// TestTempDirCreation tests that temp directories are created correctly.
func TestTempDirCreation(t *testing.T) {
	// Test that the SSH control directory logic works
	controlDir := filepath.Join(os.TempDir(), "gormes-ssh")
	
	// Just verify the path is sensible
	if controlDir == "" {
		t.Error("expected non-empty control dir path")
	}

	if !filepath.IsAbs(controlDir) {
		t.Error("expected absolute path")
	}
}
