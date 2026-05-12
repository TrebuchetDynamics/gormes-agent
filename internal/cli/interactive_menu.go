// Package cli provides shared CLI utilities including interactive menus
// with arrow-key navigation for setup, onboarding, and model selection.
package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// MenuOption is a single selectable item in an InteractiveMenu.
type MenuOption struct {
	ID      string // stable identifier used as the action value
	Label   string // human-readable label shown in the menu
	Enabled bool   // when false, the option is shown but dimmed/unselectable
}

// InteractiveMenu renders a list of options with a highlighted selection
// and supports Up/Down arrow keys, Enter to confirm, and number-key access.
type InteractiveMenu struct {
	Options  []MenuOption
	Selected int // index of the currently highlighted option

	out    io.Writer
	in     *os.File
	old    *term.State // saved terminal state, restored on Close
	open   bool
	prompt string
	header string
}

// NewInteractiveMenu creates a menu that reads keystrokes from stdin.
// Caller must call Close() when done to restore terminal state.
func NewInteractiveMenu(out io.Writer, in *os.File, prompt string) *InteractiveMenu {
	return &InteractiveMenu{
		Selected: 0,
		out:      out,
		in:       in,
		prompt:   prompt,
	}
}

// WithHeader sets a header line printed above the menu.
func (m *InteractiveMenu) WithHeader(h string) *InteractiveMenu {
	m.header = h
	return m
}

// WithOptions sets the menu options and resets selection to the first
// enabled option.
func (m *InteractiveMenu) WithOptions(opts []MenuOption) *InteractiveMenu {
	m.Options = opts
	m.Selected = firstEnabled(opts)
	return m
}

// WithDefaultIndex sets the selected index (clamped to valid range).
func (m *InteractiveMenu) WithDefaultIndex(idx int) *InteractiveMenu {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.Options) {
		idx = len(m.Options) - 1
	}
	m.Selected = idx
	return m
}

// openTerminal switches stdin to raw mode so we can read individual
// keystrokes (arrow keys, Enter, etc.).
func (m *InteractiveMenu) openTerminal() error {
	if m.open {
		return nil
	}
	if !term.IsTerminal(int(m.in.Fd())) {
		return fmt.Errorf("not a terminal")
	}
	old, err := term.MakeRaw(int(m.in.Fd()))
	if err != nil {
		return fmt.Errorf("terminal raw mode: %w", err)
	}
	m.old = old
	m.open = true
	return nil
}

// Close restores the terminal to its original mode.
func (m *InteractiveMenu) Close() {
	if m.open && m.old != nil {
		_ = term.Restore(int(m.in.Fd()), m.old)
		m.open = false
	}
}

// Run displays the menu and returns the selected option's ID when the
// user presses Enter. Returns empty string if the user cancels (Ctrl+C,
// Escape, q).
func (m *InteractiveMenu) Run() (string, error) {
	if err := m.openTerminal(); err != nil {
		// Fallback: use line-input mode if terminal is not available.
		return m.runFallback()
	}
	defer m.Close()

	m.render()

	for {
		key, err := m.readKey()
		if err != nil {
			return "", err
		}

		switch key {
		case keyUp:
			m.moveUp()
			m.render()
		case keyDown:
			m.moveDown()
			m.render()
		case keyEnter:
			// Clear the menu lines before returning.
			m.clearMenu()
			sel := m.Options[m.Selected]
			if sel.Enabled {
				return sel.ID, nil
			}
			// If current selection is disabled, try the next enabled.
			m.moveDown()
			m.render()
		case keyEscape, keyCancel:
			m.clearMenu()
			return "", nil
		case keyDigit:
			// Number key: try direct selection.
			digit := _lastDigit
			if digit >= 1 && digit <= len(m.Options) {
				idx := digit - 1
				if m.Options[idx].Enabled {
					m.Selected = idx
					m.clearMenu()
					return m.Options[idx].ID, nil
				}
			}
		}
	}
}

// clearMenu moves the cursor up and clears each line of the rendered menu.
func (m *InteractiveMenu) clearMenu() {
	lineCount := len(m.Options) + 3 // options + header + prompt
	if m.header != "" {
		lineCount++ // + blank line after header
	}
	for i := 0; i < lineCount; i++ {
		fmt.Fprint(m.out, "\033[F\033[K")
	}
}

// render draws the menu to the output.
func (m *InteractiveMenu) render() {
	if m.header != "" {
		fmt.Fprintln(m.out, m.header)
		fmt.Fprintln(m.out)
	}
	for i, opt := range m.Options {
		marker := " "
		style := "%s"
		if i == m.Selected {
			marker = "❯"
			style = "\033[7m%s\033[0m" // reverse video
		}
		label := fmt.Sprintf("%d. %s", i+1, opt.Label)
		if !opt.Enabled {
			label = "  " + opt.Label + " (N/A)"
			if i == m.Selected {
				// Don't highlight disabled options.
				label = fmt.Sprintf("  \033[2m%s (N/A)\033[0m", opt.Label)
			}
		}
		line := fmt.Sprintf(" %s %s", marker, label)
		if i == m.Selected && opt.Enabled {
			line = fmt.Sprintf(style, line)
		}
		fmt.Fprintln(m.out, line)
	}
	fmt.Fprintf(m.out, " %s ", m.prompt)
}

// moveUp moves selection to the previous enabled option, wrapping.
func (m *InteractiveMenu) moveUp() {
	for i := 0; i < len(m.Options); i++ {
		m.Selected--
		if m.Selected < 0 {
			m.Selected = len(m.Options) - 1
		}
		if m.Options[m.Selected].Enabled {
			return
		}
	}
}

// moveDown moves selection to the next enabled option, wrapping.
func (m *InteractiveMenu) moveDown() {
	for i := 0; i < len(m.Options); i++ {
		m.Selected++
		if m.Selected >= len(m.Options) {
			m.Selected = 0
		}
		if m.Options[m.Selected].Enabled {
			return
		}
	}
}

type menuKey int

const (
	keyUnknown menuKey = iota
	keyUp
	keyDown
	keyEnter
	keyEscape
	keyCancel
	keyDigit
)

var _lastDigit int

// readKey reads a single keystroke from stdin.
// Arrow keys produce 3-byte sequences (ESC [ A/B/C/D).
// We read one byte at a time and compose multi-byte sequences.
func (m *InteractiveMenu) readKey() (menuKey, error) {
	buf := make([]byte, 8)
	n, err := m.in.Read(buf)
	if err != nil {
		return keyUnknown, err
	}
	if n == 0 {
		return keyUnknown, nil
	}

	b := buf[0]

	// Escape sequence: could be ESC, arrow key (ESC [ A), or other.
	if b == 0x1b {
		if n == 1 {
			// Just ESC — treat as cancel.
			return keyEscape, nil
		}
		if n >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return keyUp, nil
			case 'B':
				return keyDown, nil
			}
		}
		return keyUnknown, nil
	}

	// Ctrl+C
	if b == 0x03 {
		return keyCancel, nil
	}

	// Enter
	if b == 0x0d || b == 0x0a {
		return keyEnter, nil
	}

	// 'q' or 'Q'
	if b == 'q' || b == 'Q' {
		return keyCancel, nil
	}

	// Digit
	if b >= '0' && b <= '9' {
		_lastDigit = int(b - '0')
		return keyDigit, nil
	}

	return keyUnknown, nil
}

// runFallback uses line-buffered input when terminal raw mode is not
// available (piped stdin, non-TTY environment).
func (m *InteractiveMenu) runFallback() (string, error) {
	fmt.Fprintln(m.out)
	if m.header != "" {
		fmt.Fprintln(m.out, m.header)
		fmt.Fprintln(m.out)
	}
	for i, opt := range m.Options {
		marker := " "
		if i == m.Selected {
			marker = "❯"
		}
		fmt.Fprintf(m.out, " %s %d. %s\n", marker, i+1, opt.Label)
	}
	fmt.Fprintf(m.out, " %s [%d]: ", m.prompt, m.Selected+1)

	var input string
	_, err := fmt.Fscanln(m.in, &input)
	if err != nil {
		return m.Options[m.Selected].ID, nil
	}
	input = strings.TrimSpace(StripANSI(input))
	if input == "" {
		return m.Options[m.Selected].ID, nil
	}
	if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(m.Options) {
		return m.Options[n-1].ID, nil
	}
	// Try matching by ID or label prefix
	for _, opt := range m.Options {
		if strings.EqualFold(input, opt.ID) || strings.HasPrefix(strings.ToLower(opt.Label), strings.ToLower(input)) {
			return opt.ID, nil
		}
	}
	return m.Options[m.Selected].ID, nil
}

// firstEnabled returns the index of the first enabled option, or 0.
func firstEnabled(opts []MenuOption) int {
	for i, o := range opts {
		if o.Enabled {
			return i
		}
	}
	return 0
}

// StripANSI is exposed for setup.go compatibility.
func StripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8DecodeRune(s[i:])
		if r == '\x1b' {
			// Skip ANSI escape sequence
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) {
					c := s[i]
					i++
					if (c >= 0x40 && c <= 0x7e) || c == '\x1b' {
						break
					}
				}
			}
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

func utf8DecodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return -1, 0
	}
	if s[0] < 0x80 {
		return rune(s[0]), 1
	}
	r, size := rune(s[0]), 1
	for size < len(s) && s[size] >= 0x80 && s[size] < 0xc0 {
		r = (r << 6) | rune(s[size]&0x3f)
		size++
	}
	return r, size
}

// PromptYesNo is a simple yes/no confirmation using terminal raw mode.
func PromptYesNo(out io.Writer, in *os.File, prompt string) (bool, error) {
	if !term.IsTerminal(int(in.Fd())) {
		return false, fmt.Errorf("not a terminal")
	}
	old, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return false, err
	}
	defer term.Restore(int(in.Fd()), old)

	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	for {
		buf := make([]byte, 4)
		n, _ := in.Read(buf)
		if n == 0 {
			return false, nil
		}
		if buf[0] == 'y' || buf[0] == 'Y' {
			fmt.Fprintln(out, "y")
			return true, nil
		}
		if buf[0] == 0x0d || buf[0] == 0x0a || buf[0] == 'n' || buf[0] == 'N' {
			fmt.Fprintln(out, "n")
			return false, nil
		}
	}
}
