package cli

import (
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display"
)

func FormatInfo(text string) string { return display.FormatInfo(text) }

func FormatSuccess(text string) string { return display.FormatSuccess(text) }

func FormatWarning(text string) string { return display.FormatWarning(text) }

func FormatError(text string) string { return display.FormatError(text) }

func FormatHeader(text string) string { return display.FormatHeader(text) }

func WriteInfo(w io.Writer, text string) error { return display.WriteInfo(w, text) }

func WriteSuccess(w io.Writer, text string) error { return display.WriteSuccess(w, text) }

func WriteWarning(w io.Writer, text string) error { return display.WriteWarning(w, text) }

func WriteError(w io.Writer, text string) error { return display.WriteError(w, text) }

func WriteHeader(w io.Writer, text string) error { return display.WriteHeader(w, text) }

func FormatPrompt(question string, defaultValue string) string {
	return display.FormatPrompt(question, defaultValue)
}

func ResolvePromptInput(input string, defaultValue string) string {
	return display.ResolvePromptInput(input, defaultValue)
}

func FormatYesNoPrompt(question string, defaultAnswer bool) string {
	return display.FormatYesNoPrompt(question, defaultAnswer)
}

func ResolveYesNoAnswer(answer string, defaultAnswer bool) bool {
	return display.ResolveYesNoAnswer(answer, defaultAnswer)
}
