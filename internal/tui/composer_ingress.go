package tui

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer"
)

type ComposerDropOptions = composer.ComposerDropOptions
type ComposerDropResult = composer.ComposerDropResult
type ComposerPasteOptions = composer.ComposerPasteOptions
type ComposerPasteSnippet = composer.ComposerPasteSnippet
type ComposerPasteResult = composer.ComposerPasteResult
type ComposerCopyResult = composer.ComposerCopyResult

func DetectComposerDroppedFile(input string, opts ComposerDropOptions) ComposerDropResult {
	return composer.DetectComposerDroppedFile(input, opts)
}

func LooksLikeComposerDroppedPath(text string) bool {
	return composer.LooksLikeComposerDroppedPath(text)
}

func IsComposerImagePath(path string) bool {
	return composer.IsComposerImagePath(path)
}

func CollapseComposerPaste(text string, opts ComposerPasteOptions) ComposerPasteResult {
	return composer.CollapseComposerPaste(text, opts)
}

func ExpandComposerPasteSnippets(input string, snippets []ComposerPasteSnippet, readFile func(string) ([]byte, error)) (string, error) {
	return composer.ExpandComposerPasteSnippets(input, snippets, readFile)
}

func RecoverComposerBracketedPaste(input string) string {
	return composer.RecoverComposerBracketedPaste(input)
}

func SelectComposerCopyText(history []llm.Message, arg string) ComposerCopyResult {
	return composer.SelectComposerCopyText(history, arg)
}

func StripComposerReasoningBlocks(text string) string {
	return composer.StripComposerReasoningBlocks(text)
}
