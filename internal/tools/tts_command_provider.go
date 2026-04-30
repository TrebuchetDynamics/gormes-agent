package tools

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultTTSCommandTimeout = 90 * time.Second

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
	if voice := strings.TrimSpace(p.Voice); voice != "" {
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
