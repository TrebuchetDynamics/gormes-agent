package wizard

// Kind identifies the input component a wizard step uses.
type Kind string

const (
	KindText      Kind = "text"
	KindMultiLine Kind = "multiline"
	KindPassword  Kind = "password"
	KindPick      Kind = "pick"
	KindChecklist Kind = "checklist"
	KindConfirm   Kind = "confirm"
)

// Choice is a stable single-select picker option.
type Choice struct {
	ID    string
	Label string
}

// Step is one prompt in a wizard sequence. Callers should prefer the
// constructor functions so future fields can stay internal to this package.
type Step struct {
	ID          string
	Prompt      string
	Placeholder string
	Kind        Kind
	Choices     []Choice

	value            Answer
	hasValue         bool
	defaultChoiceID  string
	defaultChoiceIDs []string
	pickDisplay      pickDisplay
}

type pickDisplay string

const (
	pickDisplayNumbered pickDisplay = ""
	pickDisplayRadio    pickDisplay = "radio"
	pickDisplaySearch   pickDisplay = "search"
)

// Answer is the typed result for one step.
type Answer struct {
	Kind      Kind
	Text      string
	ChoiceID  string
	ChoiceIDs []string
	Confirmed bool
}

// Result contains the final answers keyed by Step.ID.
type Result struct {
	answers map[string]Answer
}

// Answer returns the raw typed answer for id.
func (r Result) Answer(id string) (Answer, bool) {
	if r.answers == nil {
		return Answer{}, false
	}
	answer, ok := r.answers[id]
	return answer, ok
}

// String returns the captured text-like value for id.
func (r Result) String(id string) string {
	answer, _ := r.Answer(id)
	return answer.Text
}

// Choice returns the selected Choice.ID for id.
func (r Result) Choice(id string) string {
	answer, _ := r.Answer(id)
	return answer.ChoiceID
}

// Choices returns the selected Choice.ID values for a checklist step.
func (r Result) Choices(id string) []string {
	answer, _ := r.Answer(id)
	return append([]string(nil), answer.ChoiceIDs...)
}

// Bool returns the confirmation value for id.
func (r Result) Bool(id string) bool {
	answer, _ := r.Answer(id)
	return answer.Confirmed
}

func newResult() Result {
	return Result{answers: map[string]Answer{}}
}

func (r Result) put(id string, answer Answer) Result {
	if r.answers == nil {
		r.answers = map[string]Answer{}
	}
	r.answers[id] = answer
	return r
}

// StepOption configures a step.
type StepOption func(*Step)

// WithPlaceholder sets the prompt placeholder for text-like steps.
func WithPlaceholder(placeholder string) StepOption {
	return func(step *Step) {
		step.Placeholder = placeholder
	}
}

// WithStringValue marks a text-like step as already supplied by the caller.
// Runner.Run uses this to bypass the TUI when every step is fully specified.
func WithStringValue(value string) StepOption {
	return func(step *Step) {
		step.value = Answer{Kind: step.Kind, Text: value}
		step.hasValue = true
	}
}

// WithChoiceValue marks a picker step as already supplied by the caller.
func WithChoiceValue(choiceID string) StepOption {
	return func(step *Step) {
		step.value = Answer{Kind: step.Kind, ChoiceID: choiceID}
		step.hasValue = true
	}
}

// WithChoiceValues marks a checklist step as already supplied by the caller.
func WithChoiceValues(choiceIDs ...string) StepOption {
	return func(step *Step) {
		step.value = Answer{Kind: step.Kind, ChoiceIDs: append([]string(nil), choiceIDs...)}
		step.hasValue = true
	}
}

// WithDefaultChoice sets the initial cursor for a picker without pre-filling
// the step. Unlike WithChoiceValue, it still launches the Bubble Tea UI.
func WithDefaultChoice(choiceID string) StepOption {
	return func(step *Step) {
		step.defaultChoiceID = choiceID
	}
}

// WithRadioChoices renders a single-select picker as radio rows. This matches
// the Hermes setup first-run mode prompt while preserving numbered pickers for
// long provider/model lists.
func WithRadioChoices() StepOption {
	return func(step *Step) {
		step.pickDisplay = pickDisplayRadio
	}
}

// WithSearchChoices renders a single-select picker with a typeahead filter
// input above a scrollable list. Typing narrows the visible choices by
// substring match. This is the right display mode for long lists (models,
// providers with many options) where scrolling through 40+ entries is poor
// UX.
func WithSearchChoices() StepOption {
	return func(step *Step) {
		step.pickDisplay = pickDisplaySearch
	}
}

// WithDefaultChoices sets the initially checked values for a checklist without
// pre-filling the step. Unlike WithChoiceValues, it still launches the UI.
func WithDefaultChoices(choiceIDs ...string) StepOption {
	return func(step *Step) {
		step.defaultChoiceIDs = append([]string(nil), choiceIDs...)
	}
}

// WithBoolValue marks a confirm step as already supplied by the caller.
func WithBoolValue(value bool) StepOption {
	return func(step *Step) {
		step.value = Answer{Kind: step.Kind, Confirmed: value}
		step.hasValue = true
	}
}

// Text creates a single-line text input step.
func Text(id, prompt string, opts ...StepOption) Step {
	return newStep(KindText, id, prompt, nil, opts...)
}

// MultiLine creates a multi-line text input step.
func MultiLine(id, prompt string, opts ...StepOption) Step {
	return newStep(KindMultiLine, id, prompt, nil, opts...)
}

// Password creates a masked single-line text input step.
func Password(id, prompt string, opts ...StepOption) Step {
	return newStep(KindPassword, id, prompt, nil, opts...)
}

// Pick creates a single-select choice step.
func Pick(id, prompt string, choices []Choice, opts ...StepOption) Step {
	return newStep(KindPick, id, prompt, choices, opts...)
}

// Checklist creates a multi-select choice step.
func Checklist(id, prompt string, choices []Choice, opts ...StepOption) Step {
	return newStep(KindChecklist, id, prompt, choices, opts...)
}

// Confirm creates a yes/no confirmation step.
func Confirm(id, prompt string, opts ...StepOption) Step {
	return newStep(KindConfirm, id, prompt, nil, opts...)
}

func newStep(kind Kind, id, prompt string, choices []Choice, opts ...StepOption) Step {
	step := Step{
		ID:      id,
		Prompt:  prompt,
		Kind:    kind,
		Choices: append([]Choice(nil), choices...),
	}
	for _, opt := range opts {
		opt(&step)
	}
	if step.value.Kind == "" {
		step.value.Kind = kind
	}
	return step
}
