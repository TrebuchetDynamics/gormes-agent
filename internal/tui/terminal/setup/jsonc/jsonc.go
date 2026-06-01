package jsonc

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/jsonc/parser"

// StripJSONComments removes JSONC line/block comments and trailing commas while
// preserving comment-like content inside JSON strings.
func StripJSONComments(input string) string {
	return parser.StripJSONComments(input)
}

// ParseKeybindings decodes a VS Code keybindings JSONC document.
func ParseKeybindings(body []byte) ([]map[string]any, error) {
	return parser.ParseKeybindings(body)
}
