package modelpicker

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/view"

// Styles contains the semantic Lip Gloss styles needed by the model picker renderer.
type Styles = view.Styles

// Render renders the model picker overlay as a string.
func Render(state State, styles Styles) string {
	return view.Render(state, styles)
}
