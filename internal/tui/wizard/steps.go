package wizard

// Kind identifies the input component a wizard step uses.
type Kind string

const (
	KindText      Kind = "text"
	KindMultiLine Kind = "multiline"
	KindPassword  Kind = "password"
	KindPick      Kind = "pick"
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

	value    Answer
	hasValue bool
}

// Answer is the typed result for one step.
type Answer struct {
	Kind      Kind
	Text      string
	ChoiceID  string
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
