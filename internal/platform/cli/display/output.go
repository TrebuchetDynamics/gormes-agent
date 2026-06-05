package display

import (
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display/formatting"
)

func FormatInfo(text string) string { return formatting.FormatInfo(text) }

func FormatSuccess(text string) string { return formatting.FormatSuccess(text) }

func FormatWarning(text string) string { return formatting.FormatWarning(text) }

func FormatError(text string) string { return formatting.FormatError(text) }

func FormatHeader(text string) string { return formatting.FormatHeader(text) }

func WriteInfo(w io.Writer, text string) error { return formatting.WriteInfo(w, text) }

func WriteSuccess(w io.Writer, text string) error { return formatting.WriteSuccess(w, text) }

func WriteWarning(w io.Writer, text string) error { return formatting.WriteWarning(w, text) }

func WriteError(w io.Writer, text string) error { return formatting.WriteError(w, text) }

func WriteHeader(w io.Writer, text string) error { return formatting.WriteHeader(w, text) }

func FormatPrompt(question string, defaultValue string) string {
	return formatting.FormatPrompt(question, defaultValue)
}

func ResolvePromptInput(input string, defaultValue string) string {
	return formatting.ResolvePromptInput(input, defaultValue)
}

func FormatYesNoPrompt(question string, defaultAnswer bool) string {
	return formatting.FormatYesNoPrompt(question, defaultAnswer)
}

func ResolveYesNoAnswer(answer string, defaultAnswer bool) bool {
	return formatting.ResolveYesNoAnswer(answer, defaultAnswer)
}
