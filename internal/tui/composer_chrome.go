package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/chrome"

// TextInputChrome is the small reusable presentation wrapper around TUI text
// inputs. It deliberately owns presentation only; Bubble Tea textinput/textarea
// widgets still own editing behaviour.
type TextInputChrome struct {
	Width int
	Label string
	Hint  string
	Value string
	Skin  HermesSkin
}

func RenderTextInputChrome(in TextInputChrome) string {
	styles := SkinStylesFor(in.Skin)
	return chrome.RenderTextInput(chrome.TextInput{
		Width: in.Width,
		Label: in.Label,
		Hint:  in.Hint,
		Value: in.Value,
		Styles: chrome.TextInputStyles{
			Label: styles.Label,
			Dim:   styles.Dim,
		},
	})
}

// ComposerInputChrome wraps the Bubble Tea chat textarea with a contextual
// input affordance. The textarea still owns editing behaviour.
type ComposerInputChrome struct {
	Width     int
	Prompt    string
	Draft     string
	Skin      HermesSkin
	Focused   bool
	Multiline bool
}

func RenderComposerInputChrome(in ComposerInputChrome) string {
	styles := SkinStylesFor(in.Skin)
	return chrome.RenderComposerInput(chrome.ComposerInput{
		Width:     in.Width,
		Prompt:    in.Prompt,
		Draft:     in.Draft,
		Focused:   in.Focused,
		Multiline: in.Multiline,
		Styles: chrome.TextInputStyles{
			Label: styles.Label,
			Dim:   styles.Dim,
		},
		KeyHelp: keyHelpStyles(in.Skin),
	})
}

func (in ComposerInputChrome) KeyHelp() []KeyHelp {
	return chrome.ComposerKeyHelp(in.Draft, in.Multiline)
}

func composerInputChromeExtraRows(width int) int {
	return chrome.ComposerInputExtraRows(width)
}

func showComposerInputChrome(width int, height int) bool {
	return chrome.ShowComposerInput(width, height)
}
