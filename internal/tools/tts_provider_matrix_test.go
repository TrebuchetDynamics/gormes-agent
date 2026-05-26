//go:build !slim

package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTTSCommandProviderResolution(t *testing.T) {
	cfg := map[string]any{
		"providers": map[string]any{
			"openai":    map[string]any{"type": "command", "command": "echo shadow"},
			"piper":     map[string]any{"type": "command", "command": "echo shadow"},
			"piper-cli": map[string]any{"command": "piper --in {input_path} --out {output_path}"},
			"broken":    map[string]any{"type": "command", "command": "   "},
			"native":    map[string]any{"type": "python", "command": "native"},
		},
		"legacy-cli": map[string]any{"type": "command", "command": "legacy --out {output_path}"},
	}

	for _, name := range BuiltinTTSProviderNames() {
		if resolved, ok := ResolveTTSCommandProviderConfig(name, cfg); ok {
			t.Fatalf("builtin provider %q resolved to command provider: %+v", name, resolved)
		}
	}

	resolved, ok := ResolveTTSCommandProviderConfig("PIPER-CLI", cfg)
	if !ok || resolved.Command != "piper --in {input_path} --out {output_path}" {
		t.Fatalf("PIPER-CLI resolved = %+v ok=%t, want implied command provider", resolved, ok)
	}
	if _, ok := ResolveTTSCommandProviderConfig("broken", cfg); ok {
		t.Fatal("blank command provider resolved, want rejection")
	}
	if _, ok := ResolveTTSCommandProviderConfig("native", cfg); ok {
		t.Fatal("unsupported command provider type resolved, want rejection")
	}
	legacy, ok := ResolveTTSCommandProviderConfig("legacy-cli", cfg)
	if !ok || legacy.Command != "legacy --out {output_path}" {
		t.Fatalf("legacy command provider resolved = %+v ok=%t", legacy, ok)
	}

	providersWins := map[string]any{
		"providers": map[string]any{"dupe": map[string]any{"command": "new"}},
		"dupe":      map[string]any{"command": "old"},
	}
	dupe, ok := ResolveTTSCommandProviderConfig("dupe", providersWins)
	if !ok || dupe.Command != "new" {
		t.Fatalf("dupe resolved = %+v ok=%t, want providers block to win", dupe, ok)
	}

	registered := map[string]TTSProvider{}
	RegisterTTSCommandProviders(registered, cfg, nil)
	if _, ok := registered["piper-cli"]; !ok {
		t.Fatalf("registered command providers = %#v, want piper-cli", registered)
	}
	if _, ok := registered["openai"]; ok {
		t.Fatalf("registered command providers = %#v, built-in openai must not be shadowed", registered)
	}
}

func TestTTSCommandProviderExecution(t *testing.T) {
	ctx := context.Background()
	output := filepath.Join(t.TempDir(), "voice clip.ogg")
	fake := &fakeTTSCommandRunner{writeOutput: true}
	provider := NewTTSCommandProvider("py-copy", TTSCommandProviderConfig{
		Command:         `copy --in {input_path} --out "{output_path}" --voice "{voice}" --format {format} --speed {speed}`,
		Timeout:         2 * time.Second,
		OutputFormat:    "ogg",
		Voice:           "bob's $voice",
		Speed:           "1.25",
		VoiceCompatible: true,
		MaxTextLength:   5,
	}, fake)

	result := NewTTSRunner(TTSConfig{Provider: "py-copy"}, map[string]TTSProvider{
		"py-copy": provider,
	}).Synthesize(ctx, TTSRequest{
		Text:       "hello world",
		OutputPath: output,
		Platform:   "telegram",
	})

	if !result.Success || result.Provider != "py-copy" || !result.VoiceCompatible {
		t.Fatalf("result = %+v, want successful voice-compatible command provider", result)
	}
	if fake.calls != 1 {
		t.Fatalf("fake command calls = %d, want 1", fake.calls)
	}
	if fake.last.Timeout != 2*time.Second {
		t.Fatalf("timeout = %s, want 2s", fake.last.Timeout)
	}
	if fake.inputText != "hello" {
		t.Fatalf("input text = %q, want provider max_text_length truncation", fake.inputText)
	}
	if !strings.Contains(fake.last.Command, `--out "`+output+`"`) {
		t.Fatalf("rendered command missing double-quoted output path:\n%s", fake.last.Command)
	}
	if !strings.Contains(fake.last.Command, `--voice "bob's \$voice"`) {
		t.Fatalf("rendered command did not escape double-quote context:\n%s", fake.last.Command)
	}
	if !strings.Contains(fake.last.Command, "--format ogg") || !strings.Contains(fake.last.Command, "--speed 1.25") {
		t.Fatalf("rendered command missing format/speed placeholders:\n%s", fake.last.Command)
	}

	timeoutFake := &fakeTTSCommandRunner{err: context.DeadlineExceeded}
	timeoutProvider := NewTTSCommandProvider("slow", TTSCommandProviderConfig{
		Command: "slow --in {input_path} --out {output_path}",
		Timeout: time.Nanosecond,
	}, timeoutFake)
	timeoutResult := NewTTSRunner(TTSConfig{Provider: "slow"}, map[string]TTSProvider{
		"slow": timeoutProvider,
	}).Synthesize(ctx, TTSRequest{Text: "hello", OutputPath: filepath.Join(t.TempDir(), "slow.mp3")})
	if timeoutResult.Success || timeoutResult.Evidence != TTSEvidenceAPIError {
		t.Fatalf("timeout result = %+v, want redacted command error", timeoutResult)
	}
}

func TestTTSDotenvFallbackPerProvider(t *testing.T) {
	home := t.TempDir()
	restoreEnv(t, "ELEVENLABS_API_KEY", "XAI_API_KEY", "MINIMAX_API_KEY", "MISTRAL_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY", "GORMES_TTS_OPENAI_KEY", "OPENAI_API_KEY")
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte(strings.Join([]string{
		"ELEVENLABS_API_KEY=eleven-dotenv",
		"XAI_API_KEY=xai-dotenv",
		"MINIMAX_API_KEY=minimax-dotenv",
		"MISTRAL_API_KEY=mistral-dotenv",
		"GEMINI_API_KEY=gemini-dotenv",
		"GORMES_TTS_OPENAI_KEY=openai-dotenv",
	}, "\n")), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	tests := []struct {
		provider string
		key      string
		source   string
	}{
		{"elevenlabs", "eleven-dotenv", "ELEVENLABS_API_KEY"},
		{"xai", "xai-dotenv", "XAI_API_KEY"},
		{"minimax", "minimax-dotenv", "MINIMAX_API_KEY"},
		{"mistral", "mistral-dotenv", "MISTRAL_API_KEY"},
		{"gemini", "gemini-dotenv", "GEMINI_API_KEY"},
		{"openai", "openai-dotenv", "GORMES_TTS_OPENAI_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			credential := ResolveTTSProviderCredential(tt.provider, TTSProviderConfig{})
			if credential.APIKey != tt.key || credential.SourceName != tt.source {
				t.Fatalf("credential = %+v, want key from %s", credential, tt.source)
			}
		})
	}
}

func TestTTSSpeedAndCapMatrix(t *testing.T) {
	speedCfg := map[string]any{
		"speed":  1.5,
		"edge":   map[string]any{"speed": 2.0},
		"openai": map[string]any{"speed": 0.1},
	}
	if got := EdgeTTSRateString(TTSSpeedForProvider("edge", speedCfg)); got != "+100%" {
		t.Fatalf("edge rate = %q, want +100%%", got)
	}
	if got := EdgeTTSRateString(TTSSpeedForProvider("edge", map[string]any{"speed": 1.0})); got != "" {
		t.Fatalf("edge exact 1.0 rate = %q, want empty rate", got)
	}
	if got := OpenAITTSSpeed(TTSSpeedForProvider("openai", speedCfg)); got != 0.25 {
		t.Fatalf("openai clamped speed = %v, want 0.25", got)
	}
	if got := OpenAITTSSpeed(TTSSpeedForProvider("openai", map[string]any{"speed": 10.0})); got != 4.0 {
		t.Fatalf("openai high clamped speed = %v, want 4.0", got)
	}

	capCfg := map[string]any{
		"providers": map[string]any{
			"piper-cli": map[string]any{"command": "x", "max_text_length": 2500},
		},
		"elevenlabs": map[string]any{"model_id": "eleven_flash_v2_5"},
	}
	cases := map[string]int{
		"edge":       5000,
		"openai":     4096,
		"xai":        15000,
		"minimax":    10000,
		"mistral":    4000,
		"gemini":     5000,
		"piper":      5000,
		"kittentts":  2000,
		"piper-cli":  2500,
		"elevenlabs": 40000,
	}
	for provider, want := range cases {
		if got := TTSProviderMaxTextLengthForConfig(provider, capCfg); got != want {
			t.Fatalf("%s max length = %d, want %d", provider, got, want)
		}
	}

	output := filepath.Join(t.TempDir(), "openai.mp3")
	fake := &fakeTTSProvider{available: true}
	result := NewTTSRunner(TTSConfig{Provider: "openai"}, map[string]TTSProvider{
		"openai": fake,
	}).Synthesize(context.Background(), TTSRequest{Text: strings.Repeat("A", 5000), OutputPath: output})
	if !result.Success {
		t.Fatalf("synthesize result = %+v, want success", result)
	}
	if got := len([]rune(fake.last.Text)); got != MaxTextLengthOpenAI {
		t.Fatalf("runner sent %d chars to openai, want %d", got, MaxTextLengthOpenAI)
	}
}

func TestTTSLocalProviderAliasSelectsAvailableLocalBackend(t *testing.T) {
	local := &fakeTTSProvider{available: true}
	edge := &fakeTTSProvider{available: true}
	output := filepath.Join(t.TempDir(), "local.mp3")

	result := NewTTSRunner(TTSConfig{Provider: "local"}, map[string]TTSProvider{
		"edge":      edge,
		"kittentts": local,
	}).Synthesize(context.Background(), TTSRequest{Text: "hello", OutputPath: output})

	if !result.Success || result.Provider != "kittentts" {
		t.Fatalf("local alias result = %+v, want kittentts success", result)
	}
	if edge.calls != 0 || local.calls != 1 {
		t.Fatalf("provider calls edge/local = %d/%d, want 0/1", edge.calls, local.calls)
	}
	if local.last.Provider != "kittentts" {
		t.Fatalf("local provider request = %+v, want concrete provider name", local.last)
	}
}

func TestTTSLocalProviderLazyDependencyEvidence(t *testing.T) {
	ctx := context.Background()
	probes := 0
	kittentts := NewLazyLocalTTSProvider("kittentts", func(context.Context) error {
		probes++
		return errors.New("missing kittentts module")
	})
	piper := NewLazyLocalTTSProvider("piper", func(context.Context) error {
		return errors.New("missing piper module")
	})

	edge := &fakeTTSProvider{available: true}
	result := NewTTSRunner(TTSConfig{Provider: "auto"}, map[string]TTSProvider{
		"edge":      edge,
		"kittentts": kittentts,
	}).Synthesize(ctx, TTSRequest{Text: "hello", OutputPath: filepath.Join(t.TempDir(), "edge.mp3")})
	if !result.Success || result.Provider != "edge" {
		t.Fatalf("auto result = %+v, want edge success while local provider is absent", result)
	}
	if probes != 0 {
		t.Fatalf("local provider was probed during unrelated edge dispatch")
	}

	status := kittentts.DependencyStatus(ctx)
	if status.Available || status.Provider != "kittentts" || status.Evidence != TTSEvidenceProviderUnavailable {
		t.Fatalf("kittentts status = %+v, want provider-specific unavailable evidence", status)
	}
	if probes != 1 {
		t.Fatalf("dependency probes = %d, want one lazy probe", probes)
	}
	if piper.Available(ctx) {
		t.Fatal("piper should be unavailable with missing dependency")
	}
}

type fakeTTSCommandRunner struct {
	calls       int
	last        TTSCommandExecution
	inputText   string
	writeOutput bool
	err         error
}

func (f *fakeTTSCommandRunner) RunTTSCommand(_ context.Context, exec TTSCommandExecution) error {
	f.calls++
	f.last = exec
	data, err := os.ReadFile(exec.InputPath)
	if err != nil {
		return err
	}
	f.inputText = string(data)
	if f.err != nil {
		return f.err
	}
	if f.writeOutput {
		if err := os.WriteFile(exec.OutputPath, []byte("audio"), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func restoreEnv(t *testing.T, keys ...string) {
	t.Helper()
	originals := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		originals[key] = value
		present[key] = ok
		os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				os.Setenv(key, originals[key])
			} else {
				os.Unsetenv(key)
			}
		}
	})
}
