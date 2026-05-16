package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type recordingSetSessionModel struct {
	calls    int
	provider string
	model    string
	err      error
}

func (r *recordingSetSessionModel) call(provider, model string) error {
	r.calls++
	r.provider = provider
	r.model = model
	return r.err
}

func TestModelSlashOpensPickerAndDoesNotSubmit(t *testing.T) {
	for _, input := range []string{"/model", "/m"} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			setter := &recordingSetSessionModel{}
			m := newModelSlashTestModel(sub, setter, fakeModelCatalog, nil)

			m = enterModelSlash(t, m, input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if setter.calls != 0 {
				t.Fatalf("%s called SetSessionModel %d time(s), want 0 before confirmation", input, setter.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			if m.modelPicker == nil {
				t.Fatalf("%s did not open model picker", input)
			}
			if got := m.modelPicker.Providers[m.modelPicker.SelectedProviderIndex].ID; got != "anthropic" {
				t.Fatalf("selected provider = %q, want current provider anthropic", got)
			}
			view := m.View()
			if !strings.Contains(view, "Select Model") || !strings.Contains(view, "Claude Opus Test") {
				t.Fatalf("View() missing reused model picker chrome/model:\n%s", view)
			}
		})
	}
}

func TestModelSlashConfirmAppliesSelectionThroughSessionModelSeam(t *testing.T) {
	sub := &nopSubmitter{}
	setter := &recordingSetSessionModel{}
	m := enterModelSlash(t, newModelSlashTestModel(sub, setter, fakeModelCatalog, nil), "/model")

	m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if sub.calls != 0 {
		t.Fatalf("/model confirmation reached Submitter %d time(s), want 0", sub.calls)
	}
	if setter.calls != 1 {
		t.Fatalf("SetSessionModel calls = %d, want 1", setter.calls)
	}
	if setter.provider != "anthropic" || setter.model != "claude-opus-test" {
		t.Fatalf("SetSessionModel(%q, %q), want anthropic/claude-opus-test", setter.provider, setter.model)
	}
	if m.modelPicker != nil {
		t.Fatal("model picker still open after confirmation")
	}
	if !strings.Contains(m.statusMessage, "model -> claude-opus-test") {
		t.Fatalf("status after confirmation = %q, want switched model evidence", m.statusMessage)
	}
	if m.frame.Model != "claude-opus-test" {
		t.Fatalf("frame.Model = %q, want immediate local model status update", m.frame.Model)
	}
}

func TestModelSlashCancelLeavesSessionModelUnchanged(t *testing.T) {
	setter := &recordingSetSessionModel{}
	m := enterModelSlash(t, newModelSlashTestModel(&nopSubmitter{}, setter, fakeModelCatalog, nil), "/model")

	m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyEscape})

	if setter.calls != 0 {
		t.Fatalf("SetSessionModel calls after cancel = %d, want 0", setter.calls)
	}
	if m.modelPicker != nil {
		t.Fatal("model picker still open after cancel")
	}
	if !strings.Contains(m.statusMessage, "model: unchanged") {
		t.Fatalf("status after cancel = %q, want unchanged evidence", m.statusMessage)
	}
}

func TestModelSlashDirectArgumentUsesCurrentProvider(t *testing.T) {
	sub := &nopSubmitter{}
	setter := &recordingSetSessionModel{}
	m := enterModelSlash(t, newModelSlashTestModel(sub, setter, fakeModelCatalog, nil), "/model claude-haiku-test")

	if sub.calls != 0 {
		t.Fatalf("/model arg reached Submitter %d time(s), want 0", sub.calls)
	}
	if setter.calls != 1 {
		t.Fatalf("SetSessionModel calls = %d, want 1", setter.calls)
	}
	if setter.provider != "anthropic" || setter.model != "claude-haiku-test" {
		t.Fatalf("SetSessionModel(%q, %q), want anthropic/claude-haiku-test", setter.provider, setter.model)
	}
	if m.modelPicker != nil {
		t.Fatal("direct /model argument opened picker; want direct switch")
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/model arg fell through to recognized-unavailable fallback: %q", m.statusMessage)
	}
}

func TestModelSlashDirectArgumentRejectedWhileTurnIsRunning(t *testing.T) {
	sub := &nopSubmitter{}
	setter := &recordingSetSessionModel{}
	m := newModelSlashTestModel(sub, setter, fakeModelCatalog, nil)
	m.inFlight = true

	m = enterModelSlash(t, m, "/model claude-haiku-test")

	if sub.calls != 0 {
		t.Fatalf("running /model arg reached Submitter %d time(s), want 0", sub.calls)
	}
	if setter.calls != 0 {
		t.Fatalf("running /model arg called SetSessionModel %d time(s), want 0", setter.calls)
	}
	if !strings.Contains(m.statusMessage, "cannot switch models while a turn is running") {
		t.Fatalf("status after running /model arg = %q, want in-flight rejection evidence", m.statusMessage)
	}
}

func TestModelSlashCatalogFailureConsumesWithoutSlashLeak(t *testing.T) {
	sub := &nopSubmitter{}
	setter := &recordingSetSessionModel{}
	catalogErr := errors.New("catalog boom")
	m := enterModelSlash(t, newModelSlashTestModel(sub, setter, func() ([]ModelPickerCatalogProvider, error) {
		return nil, catalogErr
	}, nil), "/model")

	if sub.calls != 0 {
		t.Fatalf("catalog failure reached Submitter %d time(s), want 0", sub.calls)
	}
	if setter.calls != 0 {
		t.Fatalf("catalog failure called SetSessionModel %d time(s), want 0", setter.calls)
	}
	if m.modelPicker != nil {
		t.Fatal("catalog failure opened picker; want degraded status only")
	}
	if !strings.Contains(m.statusMessage, "model: catalog unavailable") || !strings.Contains(m.statusMessage, catalogErr.Error()) {
		t.Fatalf("status after catalog failure = %q, want model catalog evidence", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/model catalog failure fell through to fallback: %q", m.statusMessage)
	}
}

func TestModelSlashWithoutCatalogConsumesWithoutDefaultLeak(t *testing.T) {
	sub := &nopSubmitter{}
	setter := &recordingSetSessionModel{}
	m := enterModelSlash(t, newModelSlashTestModel(sub, setter, nil, nil), "/model")

	if sub.calls != 0 {
		t.Fatalf("missing catalog reached Submitter %d time(s), want 0", sub.calls)
	}
	if setter.calls != 0 {
		t.Fatalf("missing catalog called SetSessionModel %d time(s), want 0", setter.calls)
	}
	if m.modelPicker != nil {
		t.Fatal("missing catalog opened picker; remote/degraded TUI must not leak the default local catalog")
	}
	if !strings.Contains(m.statusMessage, "model: catalog unavailable") {
		t.Fatalf("status after missing catalog = %q, want catalog unavailable evidence", m.statusMessage)
	}
}

func newModelSlashTestModel(sub *nopSubmitter, setter *recordingSetSessionModel, catalog ModelPickerCatalogFunc, opts *Options) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	options := Options{
		MouseTracking:       true,
		ModelProvider:       "anthropic",
		ModelName:           "claude-sonnet-test",
		ModelPickerCatalog:  catalog,
		SetSessionModelFunc: setter.call,
	}
	if opts != nil {
		options = *opts
	}
	m := NewModelWithOptions(frames, sub.submit, func() {}, options)
	m.width = 90
	m.height = 28
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, Model: "claude-sonnet-test"}
	return m
}

func fakeModelCatalog() ([]ModelPickerCatalogProvider, error) {
	return []ModelPickerCatalogProvider{
		{
			Provider: ProviderEntry{ID: "anthropic", Label: "Anthropic"},
			Models: []ModelEntry{
				{ID: "claude-sonnet-test", Label: "Claude Sonnet Test"},
				{ID: "claude-opus-test", Label: "Claude Opus Test"},
				{ID: "claude-haiku-test", Label: "Claude Haiku Test"},
			},
		},
		{
			Provider: ProviderEntry{ID: "openai-codex", Label: "OpenAI Codex"},
			Models: []ModelEntry{
				{ID: "gpt-5.5", Label: "GPT-5.5"},
			},
		},
	}, nil
}

func enterModelSlash(t *testing.T, m Model, input string) Model {
	t.Helper()
	m.editor.SetValue(input)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	return applyModelSlashCmd(t, updated, cmd)
}

func updateModelSlashKey(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, cmd := m.Update(msg)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	return applyModelSlashCmd(t, updated, cmd)
}

func applyModelSlashCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	switch msg := msg.(type) {
	case nil:
		return m
	case tea.BatchMsg:
		for _, nested := range msg {
			m = applyModelSlashCmd(t, m, nested)
		}
		return m
	default:
		next, nextCmd := m.Update(msg)
		updated, ok := next.(Model)
		if !ok {
			t.Fatalf("Update returned %T, want tui.Model", next)
		}
		return applyModelSlashCmd(t, updated, nextCmd)
	}
}
