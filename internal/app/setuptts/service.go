package setuptts

import (
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupchoice"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type Runtime struct {
	ConfigPath                  string
	LoadConfig                  func() (config.Config, error)
	WriteTOMLValue              func(string, string, string) error
	PromptChoice                func(title, linePrompt, defaultValue string, choices []setup.Choice) (string, error)
	PromptString                func(prompt, defaultValue string) (string, error)
	ShouldPrintStaticChoiceMenu bool
	NewExitCodeError            func(int, error) error
}

func Run(out, _ io.Writer, nonInteractive bool, runtime Runtime) error {
	runtime = runtimeDefaults(runtime)
	cfg, _ := runtime.LoadConfig()
	current := firstNonEmpty(cfg.Runtime.TTSProvider, "edge")
	options := TTSProviderOptionsWithCurrent(current)
	voice := VoiceModel(cfg.TTS, current)

	fmt.Fprintln(out, "Text-to-Speech Provider")
	fmt.Fprintf(out, "Default provider: %s\n", setup.TTSProviderLabel(current))
	fmt.Fprintf(out, "Default voice/model: %s\n", firstNonEmpty(voice, "provider default"))
	fmt.Fprintln(out, "Built-in/default TTS: Edge TTS")
	fmt.Fprintln(out, "Help: choose a provider with arrows or a number, choose/test a voice before saving, or keep the current default.")
	fmt.Fprintln(out)
	if runtime.ShouldPrintStaticChoiceMenu {
		PrintChoiceList(out, options, "keep")
	}
	if nonInteractive {
		fmt.Fprintln(out, "\nSkipped (keeping current)")
		return nil
	}

	choice, err := runtime.PromptChoice("Select TTS provider", "Select TTS provider [keep]: ", "keep", options)
	if err != nil {
		return err
	}
	choice = setupchoice.NormalizeValue(choice)
	if choice == "" || choice == "keep" {
		fmt.Fprintln(out, "Keeping current TTS provider.")
		return nil
	}
	if !IsProviderChoice(choice) {
		label := setup.TTSProviderLabel(choice)
		return exitCodeError(runtime, 2, fmt.Errorf("TTS provider %q is not available in this setup screen. Choose a listed provider, or configure a custom command provider under [tts.providers] and rerun setup.", label))
	}
	fmt.Fprintf(out, "Selected provider: %s\n", setup.TTSProviderLabel(choice))
	testChoice, err := runtime.PromptString("Test voice before saving? [Y/n]: ", "y")
	if err != nil {
		return err
	}
	if setupchoice.NormalizeValue(testChoice) != "n" && setupchoice.NormalizeValue(testChoice) != "no" {
		fmt.Fprintf(out, "Test voice: %s with %s\n", setup.TTSProviderLabel(choice), firstNonEmpty(VoiceModel(cfg.TTS, choice), "provider default voice"))
		fmt.Fprintln(out, "Test voice passed (provider availability will be checked again when audio is generated).")
	}
	if err := runtime.WriteTOMLValue(runtime.ConfigPath, "runtime.tts_provider", choice); err != nil {
		return err
	}
	fmt.Fprintf(out, "TTS provider set to: %s\n", setup.TTSProviderLabel(choice))
	return nil
}

func TTSProviderOptionsWithCurrent(current string) []setup.Choice {
	options := append([]setup.Choice(nil), setup.TTSProviderOptions()...)
	label := setup.TTSProviderLabel(firstNonEmpty(current, "edge"))
	for i := range options {
		if setupchoice.NormalizeValue(options[i].Value) == "keep" {
			options[i].Label = fmt.Sprintf("Keep current (%s)", label)
			break
		}
	}
	return options
}

func IsProviderChoice(value string) bool {
	value = setupchoice.NormalizeValue(value)
	for _, option := range setup.TTSProviderOptions() {
		if option.Value == value && value != "keep" {
			return true
		}
	}
	return false
}

func VoiceModel(ttsConfig map[string]any, provider string) string {
	provider = setupchoice.NormalizeValue(provider)
	for _, key := range []string{"voice", "voice_id", "model", "default_voice", "default_model"} {
		if value := stringFromAny(ttsConfig[key]); strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if raw, ok := ttsConfig[provider].(map[string]any); ok {
		for _, key := range []string{"voice", "voice_id", "model", "default_voice", "default_model"} {
			if value := stringFromAny(raw[key]); strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func PrintChoiceList(out io.Writer, options []setup.Choice, selectedValue string) {
	selectedValue = setupchoice.NormalizeValue(selectedValue)
	for _, option := range options {
		selected := "○"
		if setupchoice.NormalizeValue(option.Value) == selectedValue {
			selected = "●"
		}
		fmt.Fprintf(out, "  (%s) %s\n", selected, option.Label)
	}
}

func runtimeDefaults(runtime Runtime) Runtime {
	if runtime.ConfigPath == "" {
		runtime.ConfigPath = config.ConfigPath()
	}
	if runtime.LoadConfig == nil {
		runtime.LoadConfig = func() (config.Config, error) { return config.Load(nil) }
	}
	if runtime.WriteTOMLValue == nil {
		runtime.WriteTOMLValue = config.WriteTOMLValue
	}
	if runtime.NewExitCodeError == nil {
		runtime.NewExitCodeError = newExitCodeError
	}
	return runtime
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func exitCodeError(runtime Runtime, code int, err error) error {
	if runtime.NewExitCodeError != nil {
		return runtime.NewExitCodeError(code, err)
	}
	return newExitCodeError(code, err)
}

type exitCodeErrorValue struct {
	code int
	err  error
}

func newExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeErrorValue{code: code, err: err}
}

func (e exitCodeErrorValue) Error() string { return e.err.Error() }
func (e exitCodeErrorValue) Unwrap() error { return e.err }
func (e exitCodeErrorValue) ExitCode() int { return e.code }
