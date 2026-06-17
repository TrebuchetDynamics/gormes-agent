//go:build !slim

package tts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/tts/commandtemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/tts/configvalue"
)

const (
	defaultTTSCommandTimeout       = 120 * time.Second
	defaultCommandTTSOutputFormat  = "mp3"
	defaultCommandTTSMaxTextLength = 5000
)

// TTSCommandProviderConfig is the Hermes-compatible user-declared
// tts.providers.<name> command provider block after normalization.
type TTSCommandProviderConfig struct {
	Command         string
	Timeout         time.Duration
	OutputFormat    string
	Voice           string
	Model           string
	Speed           string
	VoiceCompatible bool
	MaxTextLength   int
}

// TTSCommandExecution is the fakeable command-provider execution request.
type TTSCommandExecution struct {
	Command    string
	Timeout    time.Duration
	InputPath  string
	OutputPath string
}

// TTSCommandRunner executes a rendered command-provider shell command.
type TTSCommandRunner interface {
	RunTTSCommand(context.Context, TTSCommandExecution) error
}

// TTSCommandProvider runs a user-declared command provider from
// tts.providers.<name> without letting that provider shadow built-ins.
type TTSCommandProvider struct {
	name   string
	cfg    TTSCommandProviderConfig
	runner TTSCommandRunner
}

func NewTTSCommandProvider(name string, cfg TTSCommandProviderConfig, runner TTSCommandRunner) TTSProvider {
	if runner == nil {
		runner = shellTTSCommandRunner{}
	}
	cfg.Command = strings.TrimSpace(cfg.Command)
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTTSCommandTimeout
	}
	cfg.OutputFormat = normalizeCommandTTSOutputFormat(cfg.OutputFormat)
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = defaultCommandTTSOutputFormat
	}
	return &TTSCommandProvider{
		name:   normalizeTTSProviderName(name),
		cfg:    cfg,
		runner: runner,
	}
}

func (p *TTSCommandProvider) Available(context.Context) bool {
	return p != nil && strings.TrimSpace(p.cfg.Command) != ""
}

func (p *TTSCommandProvider) MaxTextLength() int {
	if p == nil || p.cfg.MaxTextLength <= 0 {
		return defaultCommandTTSMaxTextLength
	}
	return p.cfg.MaxTextLength
}

func (p *TTSCommandProvider) PreferredOutputFormat() string {
	if p == nil {
		return defaultCommandTTSOutputFormat
	}
	return firstNonEmptyTTS(p.cfg.OutputFormat, defaultCommandTTSOutputFormat)
}

func (p *TTSCommandProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	if p == nil || strings.TrimSpace(p.cfg.Command) == "" {
		return TTSProviderResult{}, errors.New("TTS command provider is not configured")
	}
	outputPath := filepath.Clean(req.OutputPath)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o700); err != nil {
		return TTSProviderResult{}, fmt.Errorf("TTS command provider mkdir: %w", err)
	}
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		return TTSProviderResult{}, fmt.Errorf("TTS command provider remove stale output: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "gormes-tts-command-*")
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("TTS command provider tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(inputPath, []byte(req.Text), 0o600); err != nil {
		return TTSProviderResult{}, fmt.Errorf("TTS command provider write input: %w", err)
	}

	outputFormat := commandTTSOutputFormat(p.cfg, outputPath)
	speed := strings.TrimSpace(p.cfg.Speed)
	if req.Speed > 0 {
		speed = fmt.Sprintf("%.2f", req.Speed)
	}
	if speed == "" {
		speed = "1.0"
	}
	command := commandtemplate.Render(p.cfg.Command, map[string]string{
		"input_path":  inputPath,
		"text_path":   inputPath,
		"output_path": outputPath,
		"format":      outputFormat,
		"voice":       firstNonEmptyTTS(req.Voice, p.cfg.Voice),
		"model":       p.cfg.Model,
		"speed":       speed,
		"language":    req.Language,
	})

	if err := p.runner.RunTTSCommand(ctx, TTSCommandExecution{
		Command:    command,
		Timeout:    p.cfg.Timeout,
		InputPath:  inputPath,
		OutputPath: outputPath,
	}); err != nil {
		return TTSProviderResult{}, formatTTSCommandError(err)
	}
	if validation := validateTTSOutputFile(p.name, outputPath); validation.Evidence != "" {
		return TTSProviderResult{}, errors.New(validation.Error)
	}
	return TTSProviderResult{
		FilePath:        outputPath,
		Provider:        firstNonEmptyTTS(p.name, req.Provider),
		VoiceCompatible: p.cfg.VoiceCompatible,
	}, nil
}

type shellTTSCommandRunner struct{}

func (shellTTSCommandRunner) RunTTSCommand(ctx context.Context, execReq TTSCommandExecution) error {
	timeout := execReq.Timeout
	if timeout <= 0 {
		timeout = defaultTTSCommandTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", execReq.Command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", execReq.Command)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
		if ctx.Err() != nil {
			return fmt.Errorf("TTS command timed out after %s: %w", timeout, ctx.Err())
		}
		return err
	}
	return nil
}

func ResolveTTSCommandProviderConfig(provider string, ttsConfig map[string]any) (TTSCommandProviderConfig, bool) {
	key := normalizeTTSProviderName(provider)
	if key == "" || isBuiltinTTSProviderName(key) {
		return TTSCommandProviderConfig{}, false
	}
	raw := namedTTSProviderConfig(ttsConfig, key)
	if !isTTSCommandProviderConfig(raw) {
		return TTSCommandProviderConfig{}, false
	}
	out := TTSCommandProviderConfig{
		Command:         configvalue.String(raw["command"]),
		Timeout:         commandTimeoutFromAny(configvalue.FirstPresent(raw, "timeout", "timeout_seconds")),
		OutputFormat:    commandTTSOutputFormatFromAny(configvalue.FirstPresent(raw, "format", "output_format"), ""),
		Voice:           configvalue.String(raw["voice"]),
		Model:           configvalue.String(raw["model"]),
		Speed:           configvalue.String(raw["speed"]),
		VoiceCompatible: configvalue.Bool(raw["voice_compatible"]),
		MaxTextLength:   configvalue.PositiveInt(raw["max_text_length"]),
	}
	if out.OutputFormat == "" {
		out.OutputFormat = defaultCommandTTSOutputFormat
	}
	return out, true
}

func RegisterTTSCommandProviders(into map[string]TTSProvider, ttsConfig map[string]any, runner TTSCommandRunner) {
	if into == nil || ttsConfig == nil {
		return
	}
	providers := configvalue.Map(ttsConfig["providers"])
	for name := range providers {
		key := normalizeTTSProviderName(name)
		if resolved, ok := ResolveTTSCommandProviderConfig(key, ttsConfig); ok {
			into[key] = NewTTSCommandProvider(key, resolved, runner)
		}
	}
	for name := range ttsConfig {
		key := normalizeTTSProviderName(name)
		if key == "" || key == "provider" || key == "providers" || isBuiltinTTSProviderName(key) {
			continue
		}
		if _, exists := into[key]; exists {
			continue
		}
		if resolved, ok := ResolveTTSCommandProviderConfig(key, ttsConfig); ok {
			into[key] = NewTTSCommandProvider(key, resolved, runner)
		}
	}
}

func namedTTSProviderConfig(ttsConfig map[string]any, name string) map[string]any {
	if ttsConfig == nil {
		return nil
	}
	providers := configvalue.Map(ttsConfig["providers"])
	if providers != nil {
		if section := configvalue.Map(configvalue.LookupCaseInsensitive(providers, name)); section != nil {
			return section
		}
	}
	if !isBuiltinTTSProviderName(name) {
		if legacy := configvalue.Map(configvalue.LookupCaseInsensitive(ttsConfig, name)); legacy != nil {
			return legacy
		}
	}
	return nil
}

func isTTSCommandProviderConfig(raw map[string]any) bool {
	if raw == nil {
		return false
	}
	providerType := strings.ToLower(strings.TrimSpace(configvalue.String(raw["type"])))
	if providerType != "" && providerType != "command" {
		return false
	}
	return strings.TrimSpace(configvalue.String(raw["command"])) != ""
}

func commandTimeoutFromAny(raw any) time.Duration {
	seconds := configvalue.Float(raw)
	if seconds <= 0 {
		return defaultTTSCommandTimeout
	}
	return time.Duration(seconds * float64(time.Second))
}

func commandTTSOutputFormat(cfg TTSCommandProviderConfig, outputPath string) string {
	return commandTTSOutputFormatFromAny(firstNonEmptyTTS(cfg.OutputFormat, defaultCommandTTSOutputFormat), outputPath)
}

func commandTTSOutputFormatFromAny(raw any, outputPath string) string {
	if outputPath != "" {
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(outputPath)), ".")
		if isSupportedCommandTTSOutputFormat(ext) {
			return ext
		}
	}
	return normalizeCommandTTSOutputFormat(configvalue.String(raw))
}

func normalizeCommandTTSOutputFormat(format string) string {
	format = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")
	if isSupportedCommandTTSOutputFormat(format) {
		return format
	}
	return ""
}

func isSupportedCommandTTSOutputFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3", "wav", "ogg", "flac":
		return true
	default:
		return false
	}
}

// EdgeTTSCommandProvider runs an edge-tts compatible CLI. It is native Gormes
// wiring: no Hermes runtime process is launched or imported.
type EdgeTTSCommandProvider struct {
	Command string
	Voice   string
	Timeout time.Duration
}

func NewEdgeTTSCommandProviderFromEnv() TTSProvider {
	cmd := strings.TrimSpace(os.Getenv("GORMES_TTS_COMMAND"))
	if cmd == "" {
		var err error
		cmd, err = exec.LookPath("edge-tts")
		if err != nil {
			return nil
		}
	}
	return EdgeTTSCommandProvider{
		Command: cmd,
		Voice:   strings.TrimSpace(os.Getenv("GORMES_TTS_VOICE")),
		Timeout: defaultTTSCommandTimeout,
	}
}

func (p EdgeTTSCommandProvider) Available(context.Context) bool {
	return strings.TrimSpace(p.Command) != ""
}

func (p EdgeTTSCommandProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	cmdPath := strings.TrimSpace(p.Command)
	if cmdPath == "" {
		return TTSProviderResult{}, errors.New("edge-tts command unavailable")
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultTTSCommandTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"--text", req.Text, "--write-media", req.OutputPath}
	voice := firstNonEmptyTTS(req.Voice, p.Voice)
	if voice == "" || strings.EqualFold(voice, "en-US-AriaNeural") {
		if languageVoice := edgeTTSVoiceForLanguage(req.Language, req.Text); languageVoice != "" {
			voice = languageVoice
		}
	}
	if voice != "" {
		args = append(args, "--voice", voice)
	}
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
		if ctx.Err() != nil {
			return TTSProviderResult{}, ctx.Err()
		}
		return TTSProviderResult{}, formatTTSCommandError(err)
	}
	return TTSProviderResult{
		FilePath: req.OutputPath,
		Provider: req.Provider,
	}, nil
}
