package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	// ErrRequiresTTY is returned when an interactive wizard is requested
	// without a terminal-backed stdin.
	ErrRequiresTTY = errors.New("wizard_requires_tty")
	// ErrAbort is returned when the operator cancels with Ctrl-C or Escape.
	ErrAbort = errors.New("wizard_aborted")
)

// Wizard runs a sequence of step prompts and returns their typed answers.
type Wizard interface {
	Run(ctx context.Context, steps ...Step) (Result, error)
}

// Runner is the production Bubble Tea wizard runner.
type Runner struct {
	in  *os.File
	out io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// WithInput sets the input stream used for terminal interaction and TTY
// detection. Nil falls back to os.Stdin.
func WithInput(in *os.File) Option {
	return func(r *Runner) {
		r.in = in
	}
}

// WithOutput sets the output stream used by the Bubble Tea program.
func WithOutput(out io.Writer) Option {
	return func(r *Runner) {
		r.out = out
	}
}

// New returns a Bubble Tea wizard runner.
func New(opts ...Option) *Runner {
	r := &Runner{
		in:  os.Stdin,
		out: os.Stdout,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run starts the wizard unless every step already has a caller-supplied
// value. In the prefilled case no TTY is required and no Bubble Tea program
// is launched.
func (r *Runner) Run(ctx context.Context, steps ...Step) (Result, error) {
	if result, ok := prefilledResult(steps); ok {
		return result, nil
	}
	if r == nil {
		r = New()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	in := r.in
	if in == nil {
		in = os.Stdin
	}
	if !term.IsTerminal(int(in.Fd())) {
		return Result{}, ErrRequiresTTY
	}
	out := r.out
	if out == nil {
		out = io.Discard
	}
	program := tea.NewProgram(
		newModel(steps),
		tea.WithContext(ctx),
		tea.WithInput(in),
		tea.WithOutput(out),
	)
	final, err := program.Run()
	if err != nil {
		return Result{}, err
	}
	m, ok := final.(model)
	if !ok {
		return Result{}, fmt.Errorf("wizard returned unexpected model %T", final)
	}
	if m.err != nil {
		return Result{}, m.err
	}
	return m.result, nil
}

func prefilledResult(steps []Step) (Result, bool) {
	result := newResult()
	for _, step := range steps {
		if !step.hasValue {
			return Result{}, false
		}
		answer := step.value
		if answer.Kind == "" {
			answer.Kind = step.Kind
		}
		result = result.put(step.ID, answer)
	}
	return result, true
}

type model struct {
	steps []Step
	index int

	result Result
	err    error
	done   bool

	width  int
	height int

	text              textinput.Model
	area              textarea.Model
	pickCursor        int
	checklistSelected map[string]struct{}
	confirmYes        bool
}

func newModel(steps []Step) model {
	m := model{
		steps:  append([]Step(nil), steps...),
		result: newResult(),
		width:  80,
		height: 24,
	}
	m.prepareActiveStep()
	return m
}

func (m model) Init() tea.Cmd {
	switch m.activeKind() {
	case KindText, KindPassword:
		return textinput.Blink
	case KindMultiLine:
		return textarea.Blink
	default:
		return nil
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeInputs()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEscape:
			m.err = ErrAbort
			m.done = true
			return m, tea.Quit
		}
		return m.updateKey(msg)
	}
	return m.updateComponent(msg)
}

func (m model) View() string {
	if m.done {
		return ""
	}
	step, ok := m.activeStep()
	if !ok {
		return ""
	}

	var b strings.Builder
	if len(m.steps) > 1 {
		progress := fmt.Sprintf("Gormes Setup — step %d of %d", m.index+1, len(m.steps))
		b.WriteString(setupTrimToWidth(progress, m.viewWidth()))
		b.WriteString("\n\n")
	}
	if step.Prompt != "" {
		b.WriteString(wrapSetupText(step.Prompt, m.viewWidth()))
		b.WriteString("\n\n")
	}

	switch step.Kind {
	case KindText, KindPassword:
		b.WriteString(m.text.View())
	case KindMultiLine:
		b.WriteString(m.area.View())
	case KindPick:
		b.WriteString(m.renderPick(step))
	case KindChecklist:
		b.WriteString(m.renderChecklist(step))
	case KindConfirm:
		b.WriteString(m.renderConfirm())
	default:
		b.WriteString("unsupported step")
	}

	b.WriteString("\n\n")
	help := "Enter submit  Esc abort"
	switch step.Kind {
	case KindPick:
		if step.pickDisplay == pickDisplayRadio {
			help = "↑↓ navigate  ENTER/SPACE select  ESC cancel"
		} else {
			help = "Up/Down or j/k navigate  1-9 select  Enter submit  Esc/q abort"
		}
	case KindChecklist:
		help = "Up/Down or j/k navigate  SPACE toggle  ENTER confirm  ESC cancel"
	case KindMultiLine:
		help = "Enter submit  Ctrl+J newline  Esc abort"
	}
	b.WriteString(wrapSetupText(help, m.viewWidth()))
	return clampSetupView(trimSetupTrailingWhitespace(b.String()), m.viewWidth(), m.viewHeight())
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	step, ok := m.activeStep()
	if !ok {
		m.done = true
		return m, tea.Quit
	}

	switch step.Kind {
	case KindText, KindPassword:
		if msg.Type == tea.KeyEnter {
			return m.finishStep(Answer{Kind: step.Kind, Text: m.text.Value()})
		}
	case KindMultiLine:
		if msg.Type == tea.KeyCtrlJ || (msg.Type == tea.KeyEnter && msg.Alt) {
			m.area.InsertString("\n")
			return m, nil
		}
		if msg.Type == tea.KeyEnter {
			return m.finishStep(Answer{Kind: step.Kind, Text: m.area.Value()})
		}
	case KindPick:
		return m.updatePick(msg, step)
	case KindChecklist:
		return m.updateChecklist(msg, step)
	case KindConfirm:
		return m.updateConfirm(msg, step)
	default:
		m.err = fmt.Errorf("unsupported wizard step kind %q", step.Kind)
		m.done = true
		return m, tea.Quit
	}
	return m.updateComponent(msg)
}

func (m model) updateComponent(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeKind() {
	case KindText, KindPassword:
		m.text, cmd = m.text.Update(msg)
	case KindMultiLine:
		m.area, cmd = m.area.Update(msg)
	}
	return m, cmd
}

func (m model) updatePick(msg tea.KeyMsg, step Step) (tea.Model, tea.Cmd) {
	if len(step.Choices) == 0 {
		m.err = fmt.Errorf("wizard pick step %q has no choices", step.ID)
		m.done = true
		return m, tea.Quit
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch r := msg.Runes[0]; {
		case r == 'q' || r == 'Q':
			m.err = ErrAbort
			m.done = true
			return m, tea.Quit
		case r >= '1' && r <= '9':
			idx := int(r - '1')
			if idx < len(step.Choices) {
				return m.finishStep(Answer{Kind: step.Kind, ChoiceID: step.Choices[idx].ID})
			}
		}
	}
	switch msg.String() {
	case "up", "k":
		if m.pickCursor > 0 {
			m.pickCursor--
		}
	case "down", "j":
		if m.pickCursor < len(step.Choices)-1 {
			m.pickCursor++
		}
	case "enter", " ":
		return m.finishStep(Answer{Kind: step.Kind, ChoiceID: step.Choices[m.pickCursor].ID})
	}
	return m, nil
}

func (m model) updateChecklist(msg tea.KeyMsg, step Step) (tea.Model, tea.Cmd) {
	if len(step.Choices) == 0 {
		m.err = fmt.Errorf("wizard checklist step %q has no choices", step.ID)
		m.done = true
		return m, tea.Quit
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch r := msg.Runes[0]; {
		case r == 'q' || r == 'Q':
			m.err = ErrAbort
			m.done = true
			return m, tea.Quit
		case r >= '1' && r <= '9':
			idx := int(r - '1')
			if idx < len(step.Choices) {
				m.toggleChecklistChoice(step.Choices[idx].ID)
			}
			return m, nil
		}
	}
	switch msg.String() {
	case "up", "k":
		if m.pickCursor > 0 {
			m.pickCursor--
		}
	case "down", "j":
		if m.pickCursor < len(step.Choices)-1 {
			m.pickCursor++
		}
	case " ":
		m.toggleChecklistChoice(step.Choices[m.pickCursor].ID)
	case "enter":
		return m.finishStep(Answer{Kind: step.Kind, ChoiceIDs: m.orderedChecklistSelection(step)})
	}
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg, step Step) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "n", "N":
		m.confirmYes = false
	case "right", "l", "y", "Y":
		m.confirmYes = true
	case " ":
		m.confirmYes = !m.confirmYes
	case "enter":
		return m.finishStep(Answer{Kind: step.Kind, Confirmed: m.confirmYes})
	default:
		return m, nil
	}
	if msg.String() == "y" || msg.String() == "Y" || msg.String() == "n" || msg.String() == "N" {
		return m.finishStep(Answer{Kind: step.Kind, Confirmed: m.confirmYes})
	}
	return m, nil
}

func (m model) finishStep(answer Answer) (tea.Model, tea.Cmd) {
	step, ok := m.activeStep()
	if !ok {
		m.done = true
		return m, tea.Quit
	}
	if answer.Kind == "" {
		answer.Kind = step.Kind
	}
	m.result = m.result.put(step.ID, answer)
	m.index++
	if m.index >= len(m.steps) {
		m.done = true
		return m, tea.Quit
	}
	cmd := m.prepareActiveStep()
	return m, cmd
}

func (m *model) prepareActiveStep() tea.Cmd {
	step, ok := m.activeStep()
	if !ok {
		return nil
	}
	m.pickCursor = 0
	m.checklistSelected = nil
	m.confirmYes = false

	switch step.Kind {
	case KindText, KindPassword:
		ti := textinput.New()
		ti.Prompt = "> "
		ti.Placeholder = step.Placeholder
		ti.SetValue(step.value.Text)
		if step.Kind == KindPassword {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '*'
		}
		ti.Width = setupInputWidth(m.width)
		m.text = ti
		return m.text.Focus()
	case KindMultiLine:
		ta := textarea.New()
		ta.Prompt = "> "
		ta.ShowLineNumbers = false
		ta.Placeholder = step.Placeholder
		ta.SetWidth(setupInputWidth(m.width))
		ta.SetHeight(4)
		ta.SetValue(step.value.Text)
		m.area = ta
		return m.area.Focus()
	case KindPick:
		if step.hasValue {
			for i, choice := range step.Choices {
				if choice.ID == step.value.ChoiceID {
					m.pickCursor = i
					break
				}
			}
		} else if step.defaultChoiceID != "" {
			for i, choice := range step.Choices {
				if choice.ID == step.defaultChoiceID {
					m.pickCursor = i
					break
				}
			}
		}
	case KindChecklist:
		m.checklistSelected = make(map[string]struct{})
		seed := step.defaultChoiceIDs
		if step.hasValue {
			seed = step.value.ChoiceIDs
		}
		for _, id := range seed {
			m.checklistSelected[id] = struct{}{}
		}
	case KindConfirm:
		if step.hasValue {
			m.confirmYes = step.value.Confirmed
		}
	}
	return nil
}

func (m *model) resizeInputs() {
	width := setupInputWidth(m.width)
	switch m.activeKind() {
	case KindText, KindPassword:
		m.text.Width = width
	case KindMultiLine:
		m.area.SetWidth(width)
	}
}

func (m model) activeStep() (Step, bool) {
	if m.index < 0 || m.index >= len(m.steps) {
		return Step{}, false
	}
	return m.steps[m.index], true
}

func (m model) activeKind() Kind {
	step, ok := m.activeStep()
	if !ok {
		return ""
	}
	return step.Kind
}

func (m model) renderPick(step Step) string {
	if len(step.Choices) == 0 {
		return "(no choices)"
	}
	if step.pickDisplay == pickDisplayRadio {
		return m.renderRadioPick(step)
	}
	var b strings.Builder
	for i, choice := range step.Choices {
		prefix := "  "
		if i == m.pickCursor {
			prefix = "→ "
		}
		label := choice.Label
		if label == "" {
			label = choice.ID
		}
		b.WriteString(wrapSetupIndented(fmt.Sprintf("%s%d. ", prefix, i+1), label, m.viewWidth()))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderRadioPick(step Step) string {
	var b strings.Builder
	for i, choice := range step.Choices {
		prefix := "   "
		marker := "(○)"
		if i == m.pickCursor {
			prefix = " → "
			marker = "(●)"
		}
		label := choice.Label
		if label == "" {
			label = choice.ID
		}
		b.WriteString(wrapSetupIndented(prefix+marker+" ", label, m.viewWidth()))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderChecklist(step Step) string {
	if len(step.Choices) == 0 {
		return "(no choices)"
	}
	var b strings.Builder
	for i, choice := range step.Choices {
		prefix := "  "
		if i == m.pickCursor {
			prefix = "→ "
		}
		marker := "[ ]"
		if _, ok := m.checklistSelected[choice.ID]; ok {
			marker = "[✓]"
		}
		label := choice.Label
		if label == "" {
			label = choice.ID
		}
		b.WriteString(wrapSetupIndented(prefix+marker+" ", label, m.viewWidth()))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) toggleChecklistChoice(id string) {
	if m.checklistSelected == nil {
		m.checklistSelected = make(map[string]struct{})
	}
	if _, ok := m.checklistSelected[id]; ok {
		delete(m.checklistSelected, id)
		return
	}
	m.checklistSelected[id] = struct{}{}
}

func (m model) orderedChecklistSelection(step Step) []string {
	out := make([]string, 0, len(m.checklistSelected))
	for _, choice := range step.Choices {
		if _, ok := m.checklistSelected[choice.ID]; ok {
			out = append(out, choice.ID)
		}
	}
	return out
}

func (m model) renderConfirm() string {
	noPrefix := "  "
	yesPrefix := "  "
	if m.confirmYes {
		yesPrefix = "→ "
	} else {
		noPrefix = "→ "
	}
	return noPrefix + "No\n" + yesPrefix + "Yes"
}

func (m model) viewWidth() int {
	if m.width <= 0 {
		return 80
	}
	return max(1, m.width)
}

func (m model) viewHeight() int {
	if m.height <= 0 {
		return 24
	}
	return max(1, m.height)
}

func setupInputWidth(terminalWidth int) int {
	if terminalWidth <= 0 {
		return 76
	}
	// Bubbles textinput/textarea add prompt/cursor cells around this content
	// width. Keep the component inside the actual terminal instead of forcing a
	// 20-column minimum that overflows cramped setup terminals.
	return max(1, terminalWidth-4)
}

func wrapSetupIndented(prefix, text string, width int) string {
	if width <= 0 {
		width = 80
	}
	if prefix == "" {
		return wrapSetupText(text, width)
	}
	indentWidth := lipgloss.Width(prefix)
	bodyWidth := max(1, width-indentWidth)
	wrapped := wrapSetupText(text, bodyWidth)
	lines := strings.Split(wrapped, "\n")
	indent := strings.Repeat(" ", indentWidth)
	for i, line := range lines {
		if i == 0 {
			lines[i] = prefix + line
			continue
		}
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func wrapSetupText(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		out = append(out, wrapSetupLine(line, width)...)
	}
	return strings.Join(out, "\n")
}

func wrapSetupLine(line string, width int) []string {
	line = strings.TrimRight(line, " \t")
	if line == "" {
		return []string{""}
	}
	var lines []string
	for lipgloss.Width(line) > width {
		cut := setupWrapCut(line, width)
		lines = append(lines, strings.TrimRight(line[:cut], " \t"))
		line = strings.TrimLeft(line[cut:], " \t")
		if line == "" {
			break
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func setupWrapCut(line string, width int) int {
	if width <= 0 {
		_, size := utf8.DecodeRuneInString(line)
		return max(1, size)
	}
	lastSpace := -1
	used := 0
	for i, r := range line {
		if r == ' ' || r == '\t' {
			lastSpace = i
		}
		rw := lipgloss.Width(string(r))
		if used+rw > width {
			if lastSpace > 0 {
				return lastSpace
			}
			if i > 0 {
				return i
			}
			return i + len(string(r))
		}
		used += rw
	}
	return len(line)
}

func trimSetupTrailingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func clampSetupView(s string, width, height int) string {
	if height <= 0 || height > 10 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= height {
		return s
	}
	if height <= 2 {
		return setupTrimToWidth("terminal too small; resize", width)
	}
	marker := setupTrimToWidth("… omitted; resize", width)
	out := make([]string, 0, height)
	out = append(out, lines[0])
	if height > 5 {
		if prompt := setupPromptLine(lines); prompt != "" {
			out = append(out, setupTrimToWidth(prompt, width))
		} else if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
			out = append(out, lines[1])
		}
	}
	out = append(out, marker)
	if focal := setupFocalLine(lines); focal != "" && len(out) < height-2 {
		out = append(out, setupTrimToWidth(focal, width))
	}
	for _, line := range setupHelpTail(lines, height-len(out)) {
		out = append(out, line)
	}
	for len(out) > height {
		out = append(out[:len(out)-2], out[len(out)-1])
	}
	return strings.Join(out, "\n")
}

func setupPromptLine(lines []string) string {
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "→") || strings.Contains(trimmed, "submit") || strings.Contains(trimmed, "abort") {
			continue
		}
		return line
	}
	return ""
}

func setupFocalLine(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "→") {
			return line
		}
	}
	return ""
}

func setupHelpTail(lines []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	start := len(lines) - limit
	if start < 0 {
		start = 0
	}
	return append([]string(nil), lines[start:]...)
}

func setupTrimToWidth(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	ellipsis := "…"
	limit := width - lipgloss.Width(ellipsis)
	used := 0
	for i, r := range text {
		rw := lipgloss.Width(string(r))
		if used+rw > limit {
			return strings.TrimRight(text[:i], " \t") + ellipsis
		}
		used += rw
	}
	return text
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
