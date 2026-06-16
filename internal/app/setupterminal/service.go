package setupterminal

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
	ShouldPrintStaticChoiceMenu bool
	NewExitCodeError            func(int, error) error
}

func Run(out, errOut io.Writer, nonInteractive bool, runtime Runtime) error {
	runtime = runtimeDefaults(runtime)
	cfg, _ := runtime.LoadConfig()
	current := firstNonEmpty(cfg.Runtime.TerminalBackend, "local")
	options := setup.TerminalBackendOptions()

	fmt.Fprintln(out, "Terminal Backend")
	fmt.Fprintf(out, "Current: %s\n", setup.TerminalBackendLabel(current))
	fmt.Fprintln(out)
	if runtime.ShouldPrintStaticChoiceMenu {
		PrintChoiceList(out, options, "keep")
	}
	if nonInteractive {
		fmt.Fprintf(out, "\nKeeping current backend: %s\n", current)
		return nil
	}

	choice, err := runtime.PromptChoice("Select terminal backend", "Select terminal backend [keep]: ", "keep", options)
	if err != nil {
		return err
	}
	choice = setupchoice.NormalizeValue(choice)
	if choice == "" || choice == "keep" {
		fmt.Fprintf(out, "Keeping current backend: %s\n", current)
		return nil
	}
	switch choice {
	case "local":
		if err := runtime.WriteTOMLValue(runtime.ConfigPath, "runtime.terminal_backend", choice); err != nil {
			return err
		}
		fmt.Fprintln(out, "Terminal backend set to: local")
		return nil
	default:
		fmt.Fprintf(errOut, "setup_terminal_backend_row_backed: backend=%s\n", choice)
		return exitCodeError(runtime, 2, fmt.Errorf("setup_terminal_backend_row_backed: %s", choice))
	}
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
