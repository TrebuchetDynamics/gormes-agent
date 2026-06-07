package composer

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer/ingress"
)

type ComposerDropOptions = ingress.ComposerDropOptions
type ComposerDropResult = ingress.ComposerDropResult
type ComposerPasteOptions = ingress.ComposerPasteOptions
type ComposerPasteSnippet = ingress.ComposerPasteSnippet
type ComposerPasteResult = ingress.ComposerPasteResult
type ComposerCopyResult = ingress.ComposerCopyResult

func DetectComposerDroppedFile(input string, opts ComposerDropOptions) ComposerDropResult {
	return ingress.DetectComposerDroppedFile(input, opts)
}

func LooksLikeComposerDroppedPath(text string) bool {
	return ingress.LooksLikeComposerDroppedPath(text)
}

func IsComposerImagePath(path string) bool {
	return ingress.IsComposerImagePath(path)
}

func CollapseComposerPaste(text string, opts ComposerPasteOptions) ComposerPasteResult {
	return ingress.CollapseComposerPaste(text, opts)
}

func ExpandComposerPasteSnippets(input string, snippets []ComposerPasteSnippet, readFile func(string) ([]byte, error)) (string, error) {
	return ingress.ExpandComposerPasteSnippets(input, snippets, readFile)
}

func RecoverComposerBracketedPaste(input string) string {
	return ingress.RecoverComposerBracketedPaste(input)
}

func SelectComposerCopyText(history []llm.Message, arg string) ComposerCopyResult {
	return ingress.SelectComposerCopyText(history, arg)
}

func StripComposerReasoningBlocks(text string) string {
	return ingress.StripComposerReasoningBlocks(text)
}
