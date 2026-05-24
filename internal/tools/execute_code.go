package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/compact"
)

const (
	defaultExecuteCodeTimeout     = 30 * time.Second
	defaultExecuteCodeStdoutLimit = 50 * 1024
	defaultExecuteCodeStderrLimit = 10 * 1024
	pythonRuntimeDisabledMessage  = "Python runtime execution is disabled in Gormes"

	ExecuteCodeModeEvidenceInvalid = "execute_code_mode_invalid"
)

type ExecuteCodeMode string

const (
	ExecuteCodeModeProject ExecuteCodeMode = "project"
	ExecuteCodeModeStrict  ExecuteCodeMode = "strict"
)

const DefaultExecuteCodeMode = ExecuteCodeModeStrict

type ExecuteCodeToolConfig struct {
	ConfigSet        bool
	ConfigValue      any
	DefaultMode      ExecuteCodeMode
	StrictSandbox    CodeSandbox
	ProjectSandbox   CodeSandbox
	SubprocessHome   SubprocessHomeResolver
	WorkspaceScope   *ProfileWorkspaceScope
	OutputCompaction compact.Config
}

type ExecuteCodeModeResolverInput struct {
	ConfigSet   bool
	ConfigValue any
	Default     ExecuteCodeMode
}

type ExecuteCodeModeResolution struct {
	Mode     ExecuteCodeMode
	Evidence []ExecuteCodeModeEvidence
}

type ExecuteCodeModeEvidence struct {
	Code    string
	Source  string
	Message string
}

func ResolveExecuteCodeMode(input ExecuteCodeModeResolverInput) ExecuteCodeModeResolution {
	var evidence []ExecuteCodeModeEvidence
	if input.ConfigSet {
		if mode, ok := normalizeExecuteCodeMode(input.ConfigValue); ok {
			return ExecuteCodeModeResolution{Mode: mode}
		}
		evidence = append(evidence, invalidExecuteCodeModeEvidence("config"))
	}

	defaultMode, ok := normalizeDefaultExecuteCodeMode(input.Default)
	if !ok {
		evidence = append(evidence, invalidExecuteCodeModeEvidence("default"))
		defaultMode = ExecuteCodeModeProject
	}
	return ExecuteCodeModeResolution{Mode: defaultMode, Evidence: evidence}
}

func ValidExecuteCodeModes() []ExecuteCodeMode {
	return []ExecuteCodeMode{ExecuteCodeModeProject, ExecuteCodeModeStrict}
}

func IsValidExecuteCodeMode(mode ExecuteCodeMode) bool {
	_, ok := normalizeExecuteCodeMode(mode)
	return ok
}

func normalizeDefaultExecuteCodeMode(mode ExecuteCodeMode) (ExecuteCodeMode, bool) {
	if strings.TrimSpace(string(mode)) == "" {
		return ExecuteCodeModeProject, true
	}
	return normalizeExecuteCodeMode(mode)
}

func normalizeExecuteCodeMode(value any) (ExecuteCodeMode, bool) {
	if value == nil {
		return "", false
	}
	var raw string
	switch v := value.(type) {
	case ExecuteCodeMode:
		raw = string(v)
	case string:
		raw = v
	default:
		raw = fmt.Sprint(v)
	}
	mode := ExecuteCodeMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case ExecuteCodeModeProject, ExecuteCodeModeStrict:
		return mode, true
	default:
		return "", false
	}
}

func invalidExecuteCodeModeEvidence(source string) ExecuteCodeModeEvidence {
	return ExecuteCodeModeEvidence{
		Code:    ExecuteCodeModeEvidenceInvalid,
		Source:  source,
		Message: "execute_code mode must be one of: project, strict",
	}
}

// CodeExecutionRequest is the sandbox contract consumed by execute_code.
type CodeExecutionRequest struct {
	Language         string
	Code             string
	Timeout          time.Duration
	StdoutLimitBytes int
	StderrLimitBytes int
}

// CodeExecutionResult is the structured response returned by execute_code.
type CodeExecutionResult struct {
	Status           string                `json:"status"`
	Language         string                `json:"language,omitempty"`
	ExitCode         int                   `json:"exit_code"`
	Stdout           string                `json:"stdout,omitempty"`
	Stderr           string                `json:"stderr,omitempty"`
	StdoutTruncated  bool                  `json:"stdout_truncated,omitempty"`
	StderrTruncated  bool                  `json:"stderr_truncated,omitempty"`
	DurationMs       int64                 `json:"duration_ms"`
	Error            string                `json:"error,omitempty"`
	Evidence         string                `json:"evidence,omitempty"`
	FilesystemAccess bool                  `json:"filesystem_access"`
	NetworkAccess    bool                  `json:"network_access"`
	Compaction       *CodeOutputCompaction `json:"compaction,omitempty"`
}

type CodeOutputCompaction struct {
	Stdout *CodeStreamCompaction `json:"stdout,omitempty"`
	Stderr *CodeStreamCompaction `json:"stderr,omitempty"`
}

type CodeStreamCompaction struct {
	Applied        bool     `json:"applied"`
	Reducer        string   `json:"reducer,omitempty"`
	OriginalBytes  int      `json:"original_bytes"`
	CompactedBytes int      `json:"compacted_bytes"`
	Evidence       []string `json:"evidence,omitempty"`
}

// CodeSandbox executes a code snippet under Gormes's guardrails.
type CodeSandbox interface {
	Execute(ctx context.Context, req CodeExecutionRequest) (CodeExecutionResult, error)
}

// ExecuteCodeTool ports the upstream execute_code surface to Go.
type ExecuteCodeTool struct {
	Sandbox          CodeSandbox
	Mode             ExecuteCodeMode
	ModeEvidence     []ExecuteCodeModeEvidence
	DefaultTimeout   time.Duration
	DefaultStdoutCap int
	DefaultStderrCap int
	WorkspaceScope   *ProfileWorkspaceScope
	OutputCompaction compact.Config
}

func NewExecuteCodeTool(configs ...ExecuteCodeToolConfig) *ExecuteCodeTool {
	cfg := ExecuteCodeToolConfig{}
	if len(configs) > 0 {
		cfg = configs[0]
	}
	defaultMode := cfg.DefaultMode
	if strings.TrimSpace(string(defaultMode)) == "" {
		defaultMode = DefaultExecuteCodeMode
	}
	resolution := ResolveExecuteCodeMode(ExecuteCodeModeResolverInput{
		ConfigSet:   cfg.ConfigSet,
		ConfigValue: cfg.ConfigValue,
		Default:     defaultMode,
	})
	return &ExecuteCodeTool{
		Sandbox:          sandboxForExecuteCodeMode(resolution.Mode, cfg),
		Mode:             resolution.Mode,
		ModeEvidence:     append([]ExecuteCodeModeEvidence(nil), resolution.Evidence...),
		DefaultTimeout:   defaultExecuteCodeTimeout,
		DefaultStdoutCap: defaultExecuteCodeStdoutLimit,
		DefaultStderrCap: defaultExecuteCodeStderrLimit,
		WorkspaceScope:   cfg.WorkspaceScope,
		OutputCompaction: cfg.OutputCompaction,
	}
}

func (*ExecuteCodeTool) Name() string { return "execute_code" }

func (t *ExecuteCodeTool) Description() string {
	switch t.Mode {
	case ExecuteCodeModeProject:
		return "Run a short POSIX shell snippet from the session working directory with output caps, timeout handling, and filesystem/network guards."
	default:
		return "Run a short POSIX shell snippet in a guarded temp directory with output caps, timeout handling, and filesystem/network guards."
	}
}

func (*ExecuteCodeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"language":{"type":"string","enum":["sh","shell"],"description":"optional runtime to use (defaults to sh)"},"code":{"type":"string","description":"POSIX shell snippet to execute"},"timeout_ms":{"type":"integer","description":"optional per-run timeout in milliseconds"},"stdout_limit_bytes":{"type":"integer","description":"optional stdout capture cap in bytes"},"stderr_limit_bytes":{"type":"integer","description":"optional stderr capture cap in bytes"},"full_output":{"type":"boolean","description":"return exact stdout/stderr without model-visible compaction"}},"required":["code"]}`)
}

func sandboxForExecuteCodeMode(mode ExecuteCodeMode, cfg ExecuteCodeToolConfig) CodeSandbox {
	switch mode {
	case ExecuteCodeModeProject:
		if cfg.ProjectSandbox != nil {
			return cfg.ProjectSandbox
		}
		return newProjectModeSandboxWithSubprocessHome(cfg.SubprocessHome)
	default:
		if cfg.StrictSandbox != nil {
			return cfg.StrictSandbox
		}
		return newStrictModeSandboxWithSubprocessHome(cfg.SubprocessHome)
	}
}

func (t *ExecuteCodeTool) Timeout() time.Duration {
	if t.DefaultTimeout > 0 {
		return t.DefaultTimeout + 5*time.Second
	}
	return defaultExecuteCodeTimeout + 5*time.Second
}

func (t *ExecuteCodeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Language         string `json:"language"`
		Code             string `json:"code"`
		TimeoutMS        int    `json:"timeout_ms"`
		StdoutLimitBytes int    `json:"stdout_limit_bytes"`
		StderrLimitBytes int    `json:"stderr_limit_bytes"`
		FullOutput       bool   `json:"full_output"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("execute_code: invalid args: %w", err)
	}
	if strings.TrimSpace(in.Code) == "" {
		return nil, fmt.Errorf("execute_code: code is required")
	}
	language := strings.ToLower(strings.TrimSpace(in.Language))
	if language == "" {
		language = "sh"
	}
	if !executeCodeShellLanguage(language) {
		return json.Marshal(CodeExecutionResult{
			Status:   "blocked",
			Language: language,
			ExitCode: -1,
			Error:    fmt.Sprintf("%s; execute_code only supports sh snippets", pythonRuntimeDisabledMessage),
		})
	}

	req := CodeExecutionRequest{
		Language:         language,
		Code:             in.Code,
		Timeout:          durationOrDefault(in.TimeoutMS, t.DefaultTimeout, defaultExecuteCodeTimeout),
		StdoutLimitBytes: intOrDefault(in.StdoutLimitBytes, t.DefaultStdoutCap, defaultExecuteCodeStdoutLimit),
		StderrLimitBytes: intOrDefault(in.StderrLimitBytes, t.DefaultStderrCap, defaultExecuteCodeStderrLimit),
	}

	if t.Mode == ExecuteCodeModeProject && t.WorkspaceScope != nil && t.WorkspaceScope.Configured() {
		return json.Marshal(CodeExecutionResult{
			Status:   "blocked",
			Language: language,
			ExitCode: -1,
			Error:    ProfileWorkspaceScopeViolation + ": project-mode execute_code cannot prove confinement for a non-empty profile workspace allow-list; fail closed before spawning",
			Evidence: ProfileWorkspaceScopeViolation,
		})
	}

	sandbox := t.Sandbox
	if sandbox == nil {
		sandbox = sandboxForExecuteCodeMode(t.Mode, ExecuteCodeToolConfig{})
	}
	result, err := sandbox.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	t.applyOutputCompaction(&result, in.Code, in.FullOutput)
	return json.Marshal(result)
}

func (t *ExecuteCodeTool) applyOutputCompaction(result *CodeExecutionResult, code string, fullOutput bool) {
	if result == nil || fullOutput {
		return
	}
	stdout := compact.Compact(compact.Request{
		ToolName: "execute_code",
		Command:  code,
		Stream:   "stdout",
		Text:     result.Stdout,
		ExitCode: result.ExitCode,
	}, t.OutputCompaction)
	stderr := compact.Compact(compact.Request{
		ToolName: "execute_code",
		Command:  code,
		Stream:   "stderr",
		Text:     result.Stderr,
		ExitCode: result.ExitCode,
	}, t.OutputCompaction)

	var evidence CodeOutputCompaction
	if stdout.Applied {
		result.Stdout = stdout.Text
		evidence.Stdout = codeStreamCompaction(stdout)
	}
	if stderr.Applied {
		result.Stderr = stderr.Text
		evidence.Stderr = codeStreamCompaction(stderr)
	}
	if evidence.Stdout != nil || evidence.Stderr != nil {
		result.Compaction = &evidence
	}
}

func codeStreamCompaction(result compact.Result) *CodeStreamCompaction {
	return &CodeStreamCompaction{
		Applied:        result.Applied,
		Reducer:        result.Reducer,
		OriginalBytes:  result.OriginalBytes,
		CompactedBytes: result.CompactedBytes,
		Evidence:       append([]string(nil), result.Evidence...),
	}
}

func executeCodeShellLanguage(language string) bool {
	switch language {
	case "sh", "shell":
		return true
	default:
		return false
	}
}

func durationOrDefault(ms int, preferred, fallback time.Duration) time.Duration {
	if ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	if preferred > 0 {
		return preferred
	}
	return fallback
}

func intOrDefault(v, preferred, fallback int) int {
	if v > 0 {
		return v
	}
	if preferred > 0 {
		return preferred
	}
	return fallback
}

type LocalCodeSandbox struct {
	lookPath       func(string) (string, error)
	languages      map[string]runtimeSpec
	workdir        func(stagingDir string) string
	subprocessHome SubprocessHomeResolver
}

type runtimeSpec struct {
	Binaries  []string
	Args      []string
	Extension string
}

func NewLocalCodeSandbox() *LocalCodeSandbox {
	return &LocalCodeSandbox{
		lookPath: exec.LookPath,
		languages: map[string]runtimeSpec{
			"sh":    {Binaries: []string{"sh"}, Extension: ".sh"},
			"shell": {Binaries: []string{"sh"}, Extension: ".sh"},
		},
	}
}

func (s *LocalCodeSandbox) Execute(ctx context.Context, req CodeExecutionRequest) (CodeExecutionResult, error) {
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	req.Code = strings.TrimSpace(req.Code)
	req.Timeout = durationOrDefault(0, req.Timeout, defaultExecuteCodeTimeout)
	req.StdoutLimitBytes = intOrDefault(req.StdoutLimitBytes, 0, defaultExecuteCodeStdoutLimit)
	req.StderrLimitBytes = intOrDefault(req.StderrLimitBytes, 0, defaultExecuteCodeStderrLimit)

	result := CodeExecutionResult{
		Language:         req.Language,
		FilesystemAccess: false,
		NetworkAccess:    false,
	}

	if blockedReason := sandboxGuardReason(req.Language, req.Code); blockedReason != "" {
		result.Status = "blocked"
		result.Error = blockedReason
		return result, nil
	}

	spec, err := s.resolveRuntime(req.Language)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, nil
	}

	tempDir, err := os.MkdirTemp("", "gormes-execute-code-*")
	if err != nil {
		return CodeExecutionResult{}, fmt.Errorf("execute_code: create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	scriptPath := filepath.Join(tempDir, "snippet"+spec.Extension)
	if err := os.WriteFile(scriptPath, []byte(req.Code), 0o600); err != nil {
		return CodeExecutionResult{}, fmt.Errorf("execute_code: write script: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	args := append(append([]string(nil), spec.Args...), scriptPath)
	cmd := exec.CommandContext(runCtx, spec.Binaries[0], args...)
	cmd.Dir = tempDir
	if s.workdir != nil {
		if workdir := strings.TrimSpace(s.workdir(tempDir)); workdir != "" {
			cmd.Dir = workdir
		}
	}
	cmd.Env = safeSandboxEnv(s.subprocessHome)

	stdout := newLimitedBuffer(req.StdoutLimitBytes)
	stderr := newLimitedBuffer(req.StderrLimitBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	runErr := cmd.Run()
	result.DurationMs = time.Since(start).Milliseconds()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.StdoutTruncated = stdout.Truncated()
	result.StderrTruncated = stderr.Truncated()

	switch {
	case runCtx.Err() == context.DeadlineExceeded:
		result.Status = "timeout"
		result.Error = fmt.Sprintf("execution timed out after %s", req.Timeout)
	case runCtx.Err() == context.Canceled:
		result.Status = "interrupted"
		result.ExitCode = 130
		result.Error = "execution interrupted"
	case runErr != nil:
		result.Status = "error"
		result.ExitCode = exitCode(runErr)
		result.Error = runErr.Error()
	default:
		result.Status = "success"
	}

	return result, nil
}

func (s *LocalCodeSandbox) resolveRuntime(language string) (runtimeSpec, error) {
	spec, ok := s.languages[language]
	if !ok {
		return runtimeSpec{}, fmt.Errorf("execute_code: unsupported language %q", language)
	}
	for _, candidate := range spec.Binaries {
		if resolved, err := s.lookPath(candidate); err == nil {
			spec.Binaries = []string{resolved}
			return spec, nil
		}
	}
	return runtimeSpec{}, fmt.Errorf("execute_code: runtime for %q is unavailable", language)
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return -1
	}
	return exitErr.ExitCode()
}

func safeSandboxEnv(resolve SubprocessHomeResolver) []string {
	keys := []string{"PATH", "HOME", "LANG", "LC_ALL", "TMPDIR", "TMP", "TEMP"}
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			env = append(env, key+"="+value)
		}
	}
	return envWithSubprocessHome(env, resolve)
}

var (
	shellFilesystemPattern = regexp.MustCompile(`\b(cat|touch|ls|find|mkdir|rm|cp|mv)\b`)
	shellNetworkPattern    = regexp.MustCompile(`\b(curl|wget|ping|nc|ssh|scp|ftp|dig|host)\b`)
)

func sandboxGuardReason(language, code string) string {
	switch language {
	case "sh", "shell":
		switch {
		case shellFilesystemPattern.MatchString(code):
			return "filesystem access is disabled in sandboxed exec"
		case shellNetworkPattern.MatchString(code):
			return "network access is disabled in sandboxed exec"
		}
	}
	return ""
}

type limitedBuffer struct {
	limit     int
	builder   strings.Builder
	truncated bool
}

func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.truncated = len(p) > 0
		return len(p), nil
	}
	remaining := b.limit - b.builder.Len()
	switch {
	case remaining <= 0:
		b.truncated = true
	case len(p) <= remaining:
		_, _ = b.builder.Write(p)
	default:
		_, _ = b.builder.Write(p[:remaining])
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	if !b.truncated {
		return b.builder.String()
	}
	return fmt.Sprintf("%s\n[truncated at %d bytes]", b.builder.String(), b.limit)
}

func (b *limitedBuffer) Truncated() bool { return b.truncated }
