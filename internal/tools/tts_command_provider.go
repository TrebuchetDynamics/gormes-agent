//go:build !slim

package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
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
	command := renderTTSCommandTemplate(p.cfg.Command, map[string]string{
		"input_path":  inputPath,
		"text_path":   inputPath,
		"output_path": outputPath,
		"format":      outputFormat,
		"voice":       firstNonEmptyTTS(req.Voice, p.cfg.Voice),
		"model":       p.cfg.Model,
		"speed":       speed,
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
		Command:         stringFromAny(raw["command"]),
		Timeout:         commandTimeoutFromAny(firstPresentAny(raw, "timeout", "timeout_seconds")),
		OutputFormat:    commandTTSOutputFormatFromAny(firstPresentAny(raw, "format", "output_format"), ""),
		Voice:           stringFromAny(raw["voice"]),
		Model:           stringFromAny(raw["model"]),
		Speed:           stringFromAny(raw["speed"]),
		VoiceCompatible: boolFromAny(raw["voice_compatible"]),
		MaxTextLength:   positiveIntFromAny(raw["max_text_length"]),
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
	providers := mapFromAny(ttsConfig["providers"])
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
	providers := mapFromAny(ttsConfig["providers"])
	if providers != nil {
		if section := mapFromAny(lookupCaseInsensitiveAny(providers, name)); section != nil {
			return section
		}
	}
	if !isBuiltinTTSProviderName(name) {
		if legacy := mapFromAny(lookupCaseInsensitiveAny(ttsConfig, name)); legacy != nil {
			return legacy
		}
	}
	return nil
}

func isTTSCommandProviderConfig(raw map[string]any) bool {
	if raw == nil {
		return false
	}
	providerType := strings.ToLower(strings.TrimSpace(stringFromAny(raw["type"])))
	if providerType != "" && providerType != "command" {
		return false
	}
	return strings.TrimSpace(stringFromAny(raw["command"])) != ""
}

func commandTimeoutFromAny(raw any) time.Duration {
	seconds := floatFromAny(raw)
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
	return normalizeCommandTTSOutputFormat(stringFromAny(raw))
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

func renderTTSCommandTemplate(template string, placeholders map[string]string) string {
	if template == "" || len(placeholders) == 0 {
		return template
	}
	markerOpen := "\x00GORMES_TTS_OPEN\x00"
	markerClose := "\x00GORMES_TTS_CLOSE\x00"
	protected := strings.ReplaceAll(template, "{{", markerOpen)
	protected = strings.ReplaceAll(protected, "}}", markerClose)
	pattern := regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	matches := pattern.FindAllStringSubmatchIndex(protected, -1)
	var rendered strings.Builder
	rendered.Grow(len(protected))
	last := 0
	for _, match := range matches {
		rendered.WriteString(protected[last:match[0]])
		token := protected[match[0]:match[1]]
		name := protected[match[2]:match[3]]
		value, ok := placeholders[name]
		if !ok {
			rendered.WriteString(token)
		} else {
			rendered.WriteString(quoteTTSCommandPlaceholder(value, shellQuoteContext(protected, match[0])))
		}
		last = match[1]
	}
	rendered.WriteString(protected[last:])
	out := rendered.String()
	out = strings.ReplaceAll(out, markerOpen, "{")
	out = strings.ReplaceAll(out, markerClose, "}")
	return out
}

func shellQuoteContext(template string, position int) string {
	quote := byte(0)
	escaped := false
	for i := 0; i < position && i < len(template); i++ {
		ch := template[i]
		switch quote {
		case '\'':
			if ch == '\'' {
				quote = 0
			}
		case '"':
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				quote = 0
			}
		default:
			if ch == '\'' || ch == '"' {
				quote = ch
			} else if ch == '\\' {
				i++
			}
		}
	}
	if quote == 0 {
		return ""
	}
	return string(quote)
}

func quoteTTSCommandPlaceholder(value, quoteContext string) string {
	switch quoteContext {
	case "'":
		return strings.ReplaceAll(value, "'", `'\''`)
	case `"`:
		replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
		return replacer.Replace(value)
	default:
		return shellQuoteTTSPlaceholder(value)
	}
}

var shellSafeTTSPlaceholder = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=,-]+$`)

func shellQuoteTTSPlaceholder(value string) string {
	if value == "" {
		return "''"
	}
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(value, `"`, `\"`)
	}
	if shellSafeTTSPlaceholder.MatchString(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func mapFromAny(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return nil
	}
}

func lookupCaseInsensitiveAny(values map[string]any, key string) any {
	if values == nil {
		return nil
	}
	if value, ok := values[key]; ok {
		return value
	}
	for candidate, value := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), key) {
			return value
		}
	}
	return nil
}

func firstPresentAny(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if values == nil {
			return nil
		}
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func positiveIntFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case float64:
		return typed
	case float32:
		return float64(typed)
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%f", &parsed); err == nil {
			return parsed
		}
	}
	return 0
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
	if voice := firstNonEmptyTTS(req.Voice, p.Voice); voice != "" {
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
