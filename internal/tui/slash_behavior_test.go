package tui

import (
	"context"
	"errors"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestProfileSlashShowsAndPersistsNextActiveProfile(t *testing.T) {
	baseHome := t.TempDir()
	m := NewModelWithOptions(nil, func(string) {}, func() {}, Options{ProfileName: "main", ProfileNames: []string{"main", "mineru"}, ProfileBaseHome: baseHome})

	show := profileSlashHandler("/profile", &m)
	if !show.Handled || !strings.Contains(show.StatusMessage, "profile: main") {
		t.Fatalf("/profile result = %+v, want current profile", show)
	}

	switched := profileSlashHandler("/profile mineru", &m)
	if !switched.Handled || !strings.Contains(switched.StatusMessage, "profile: mineru") {
		t.Fatalf("/profile mineru result = %+v", switched)
	}
	if m.profileName != "mineru" || !strings.Contains(m.editor.View(), "mineru ❯") {
		t.Fatalf("/profile mineru did not update visible TUI label: profile=%q editor=%q", m.profileName, m.editor.View())
	}
	data, err := os.ReadFile(filepath.Join(baseHome, "active_profile"))
	if err != nil {
		t.Fatalf("read active profile: %v", err)
	}
	if strings.TrimSpace(string(data)) != "mineru" {
		t.Fatalf("active_profile = %q, want mineru", data)
	}

	unknown := profileSlashHandler("/profile miner", &m)
	if !unknown.Handled || !strings.Contains(unknown.StatusMessage, "profile_unknown: miner") || !strings.Contains(unknown.StatusMessage, "mineru") {
		t.Fatalf("/profile miner result = %+v, want unknown with available profiles", unknown)
	}
}

func TestProfileSlashCompletesKnownProfileNames(t *testing.T) {
	got := ProfileNameCompletions("/profile miner", []string{"main", "mineru", "minero"})
	if names := completionNames(got); !reflect.DeepEqual(names, []string{"minero", "mineru"}) {
		t.Fatalf("ProfileNameCompletions = %v, want minero/mineru", names)
	}
	if got := ProfileNameCompletions("/profile miner typo", []string{"mineru"}); len(got) != 0 {
		t.Fatalf("ProfileNameCompletions with extra args = %+v, want none", got)
	}
}

func TestProfileSlashCompletionAcceptanceInsertsArgument(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		OfflineSmoke: true,
		ProfileName:  "mineru",
		ProfileNames: []string{"gormed", "main", "minero", "mineru", "rijuriju"},
	})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/profile")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if got := m.editor.Value(); got != "/profile rijuriju" {
		t.Fatalf("selected /profile completion editor = %q, want /profile rijuriju", got)
	}
	if strings.Contains(m.statusMessage, "next launch") {
		t.Fatalf("accepting completion dispatched /profile instead of completing editor: %q", m.statusMessage)
	}
}

func TestProfileSlashCurrentProfileIsNoop(t *testing.T) {
	baseHome := t.TempDir()
	m := NewModelWithOptions(nil, func(string) {}, func() {}, Options{ProfileName: "mineru", ProfileNames: []string{"main", "mineru"}, ProfileBaseHome: baseHome})

	got := profileSlashHandler("/profile mineru", &m)
	if !got.Handled || !strings.Contains(got.StatusMessage, "already using mineru") {
		t.Fatalf("/profile mineru while active = %+v, want already-using status", got)
	}
	if _, err := os.Stat(filepath.Join(baseHome, "active_profile")); !os.IsNotExist(err) {
		t.Fatalf("current profile selection should not rewrite active_profile; stat err=%v", err)
	}
}

// ---- slash_branch_test.go ----

// recordingBranchFunc captures the BranchRequest it receives and returns
// the configured BranchResult or error. Used by the TUI /branch tests to
// prove the slash handler builds the right request and applies the result
// to the model without going through kernel.Submit.
type recordingBranchFunc struct {
	calls  int
	gotReq BranchRequest
	gotCtx context.Context
	result BranchResult
	err    error
}

func (r *recordingBranchFunc) call(ctx context.Context, req BranchRequest) (BranchResult, error) {
	r.calls++
	r.gotReq = req
	r.gotCtx = ctx
	if r.err != nil {
		return BranchResult{}, r.err
	}
	return r.result, nil
}

// nopSubmitter records whether kernel.Submit was reached. The /branch
// handler MUST never let the slash text fall through to the kernel.
type nopSubmitter struct {
	calls int
}

func (s *nopSubmitter) submit(string) { s.calls++ }

func newBranchTestModel(t *testing.T, history []llm.Message, frameSessionID string, fn SessionBranchFunc, sub Submitter) Model {
	t.Helper()
	frames := make(chan kernel.RenderFrame, 1)
	if sub == nil {
		sub = func(string) {}
	}
	m := NewModelWithOptions(frames, sub, func() {}, Options{
		MouseTracking: true,
		SessionBranch: fn,
	})
	m.frame.History = history
	m.frame.SessionID = frameSessionID
	return m
}

func TestSlashBranch_EmptyHistoryReturnsStatus(t *testing.T) {
	rec := &recordingBranchFunc{result: BranchResult{SessionID: "must-not-be-used"}}
	sub := &nopSubmitter{}
	m := newBranchTestModel(t, nil, "sess-parent", rec.call, sub.submit)

	res := branchSlashHandler("/branch", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true (slash MUST be consumed even when history is empty)")
	}
	if res.StatusMessage != "branch: no conversation" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "branch: no conversation")
	}
	if rec.calls != 0 {
		t.Fatalf("SessionBranchFunc called %d times, want 0 (must short-circuit before calling fork)", rec.calls)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
	if m.SessionID() != "sess-parent" {
		t.Fatalf("SessionID = %q, want sess-parent (no fork happened)", m.SessionID())
	}
}

func TestSlashBranch_HappyPathSwitchesSessionIDAndDoesNotSubmit(t *testing.T) {
	history := []llm.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "ack"},
	}
	rec := &recordingBranchFunc{result: BranchResult{
		SessionID:        "sess-child",
		ParentSessionID:  "sess-parent",
		Title:            "",
		TranscriptCopied: 2,
	}}
	sub := &nopSubmitter{}
	m := newBranchTestModel(t, history, "sess-parent", rec.call, sub.submit)

	res := branchSlashHandler("/branch", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if rec.calls != 1 {
		t.Fatalf("SessionBranchFunc called %d times, want 1", rec.calls)
	}
	if rec.gotReq.ParentSessionID != "sess-parent" {
		t.Fatalf("BranchRequest.ParentSessionID = %q, want sess-parent (current frame SessionID)", rec.gotReq.ParentSessionID)
	}
	if rec.gotReq.HistoryCount != 2 {
		t.Fatalf("BranchRequest.HistoryCount = %d, want 2", rec.gotReq.HistoryCount)
	}
	if len(rec.gotReq.History) != 2 || rec.gotReq.History[0].Content != "first" || rec.gotReq.History[1].Content != "ack" {
		t.Fatalf("BranchRequest.History = %+v, want cloned visible history", rec.gotReq.History)
	}
	history[0].Content = "mutated after handler"
	if rec.gotReq.History[0].Content != "first" {
		t.Fatalf("BranchRequest.History was not cloned: %+v", rec.gotReq.History)
	}
	if rec.gotReq.Title != "" {
		t.Fatalf("BranchRequest.Title = %q, want empty for /branch with no name", rec.gotReq.Title)
	}
	if got := m.SessionID(); got != "sess-child" {
		t.Fatalf("model SessionID = %q, want sess-child after fork", got)
	}
	if got := m.frame.SessionID; got != "sess-child" {
		t.Fatalf("frame.SessionID = %q, want sess-child after fork", got)
	}
	if sub.calls != 0 {
		t.Fatalf("kernel.Submit called %d times, want 0 (slash must never reach the kernel)", sub.calls)
	}
}

func TestSlashBranch_CustomNamePreserved(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}
	rec := &recordingBranchFunc{result: BranchResult{SessionID: "sess-child", ParentSessionID: "sess-parent"}}
	m := newBranchTestModel(t, history, "sess-parent", rec.call, nil)

	res := branchSlashHandler("/branch refactor path", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if rec.gotReq.Title != "refactor path" {
		t.Fatalf("BranchRequest.Title = %q, want %q", rec.gotReq.Title, "refactor path")
	}
}

func TestSlashBranch_NoActiveSessionReturnsStatus(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}
	rec := &recordingBranchFunc{}
	m := newBranchTestModel(t, history, "", rec.call, nil)

	res := branchSlashHandler("/branch", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "branch: no active session" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "branch: no active session")
	}
	if rec.calls != 0 {
		t.Fatalf("SessionBranchFunc called %d times, want 0", rec.calls)
	}
}

func TestSlashBranch_StoreUnavailableWhenFuncMissing(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}
	m := newBranchTestModel(t, history, "sess-parent", nil, nil)

	res := branchSlashHandler("/branch", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "branch: store unavailable" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "branch: store unavailable")
	}
	if m.SessionID() != "sess-parent" {
		t.Fatalf("SessionID = %q, want sess-parent (fork must not switch when store missing)", m.SessionID())
	}
}

func TestSlashBranch_ForkErrorLeavesParentActive(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}
	rec := &recordingBranchFunc{err: errors.New("disk full")}
	m := newBranchTestModel(t, history, "sess-parent", rec.call, nil)

	res := branchSlashHandler("/branch", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "branch: fork failed: disk full" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "branch: fork failed: disk full")
	}
	if m.SessionID() != "sess-parent" {
		t.Fatalf("SessionID = %q, want sess-parent (fork failure must leave parent active)", m.SessionID())
	}
}

func TestSlashBranch_RegisteredOnDefaultRegistry(t *testing.T) {
	rec := &recordingBranchFunc{result: BranchResult{SessionID: "sess-child", ParentSessionID: "sess-parent"}}
	sub := &nopSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SessionBranch: rec.call,
	})
	m.frame.History = []llm.Message{{Role: "user", Content: "hi"}}
	m.frame.SessionID = "sess-parent"

	res := m.slashRegistry.Dispatch("/branch", &m)
	if !res.Handled {
		t.Fatal("Default registry did not route /branch — slash must be registered out of the box")
	}
	if rec.calls != 1 {
		t.Fatalf("SessionBranchFunc called %d times via registry, want 1", rec.calls)
	}
}

// ---- slash_browser_test.go ----

func TestBrowserSlashStatusReportsMissingConnection(t *testing.T) {
	t.Setenv("BROWSER_CDP_URL", "")
	t.Setenv("CHROME_REMOTE_DEBUGGING_URL", "")

	res := browserSlashHandler("/browser status", nil)
	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if !strings.Contains(res.StatusMessage, "browser: not connected") || !strings.Contains(res.StatusMessage, "/browser connect http://127.0.0.1:9222") {
		t.Fatalf("StatusMessage = %q, want setup guidance", res.StatusMessage)
	}
}

func TestBrowserSlashConnectSetsBothCDPAliases(t *testing.T) {
	t.Setenv("BROWSER_CDP_URL", "")
	t.Setenv("CHROME_REMOTE_DEBUGGING_URL", "")

	res := browserSlashHandler("/browser connect http://127.0.0.1:9333", nil)
	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "browser: connected http://127.0.0.1:9333" {
		t.Fatalf("StatusMessage = %q, want connected evidence", res.StatusMessage)
	}
	if got := os.Getenv("BROWSER_CDP_URL"); got != "http://127.0.0.1:9333" {
		t.Fatalf("BROWSER_CDP_URL = %q, want configured endpoint", got)
	}
	if got := os.Getenv("CHROME_REMOTE_DEBUGGING_URL"); got != "http://127.0.0.1:9333" {
		t.Fatalf("CHROME_REMOTE_DEBUGGING_URL = %q, want configured endpoint", got)
	}
}

func TestBrowserSlashRejectsInvalidConnectURL(t *testing.T) {
	res := browserSlashHandler("/browser connect file:///tmp/browser", nil)
	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if !strings.Contains(res.StatusMessage, "browser: invalid CDP URL") {
		t.Fatalf("StatusMessage = %q, want invalid URL evidence", res.StatusMessage)
	}
}

func TestDefaultSlashRegistryRoutesBrowser(t *testing.T) {
	t.Setenv("BROWSER_CDP_URL", "http://127.0.0.1:9222")
	res := NewDefaultSlashRegistry().Dispatch("/browser status", nil)
	if !res.Handled {
		t.Fatal("Default registry did not route /browser")
	}
	if res.StatusMessage != "browser: connected http://127.0.0.1:9222" {
		t.Fatalf("StatusMessage = %q, want connected status", res.StatusMessage)
	}
}

// ---- slash_commands_test.go ----

func TestSlashCommandAliasAndPrefixDispatch(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus string
	}{
		{name: "catalog alias", input: "/provider openrouter", wantStatus: "/provider -> /model"},
		{name: "unique prefix", input: "/kanb --json", wantStatus: "/kanb -> /kanban"},
		{name: "exact wins over prefix", input: "/status now", wantStatus: "no active session"},
		{name: "ambiguous prefix", input: "/stat now", wantStatus: "ambiguous command: /status, /statusbar"},
		{name: "platform prefix ambiguity", input: "/platf --json", wantStatus: "ambiguous command: /platform, /platforms"},
		{name: "unknown command", input: "/no-such-command-xyzzy", wantStatus: "unknown command /no-such-command-xyzzy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			m.editor.SetValue(tt.input)
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runTestCmd(t, cmd)
			updated := next.(Model)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if got := updated.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if !strings.Contains(updated.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want it to contain %q", tt.input, updated.statusMessage, tt.wantStatus)
			}
		})
	}
}

// ---- slash_compact_test.go ----

func TestCompactSlashTogglesTranscriptRenderingWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newCompactSlashModel(sub, false)
	m.width = 96
	m.height = 28
	m.frame.History = compactSlashHistory()

	full := m.View()
	if !strings.Contains(full, "───") {
		t.Fatalf("full transcript view missing turn separator before /compact on:\n%s", full)
	}

	m = enterSlashDispatchBehavior(t, m, "/compact on")

	if sub.calls != 0 {
		t.Fatalf("/compact reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /compact = %q, want cleared", got)
	}
	if !m.compactTranscript {
		t.Fatal("compactTranscript = false after /compact on, want true")
	}
	if !strings.Contains(m.statusMessage, "compact on") {
		t.Fatalf("status after /compact on = %q, want compact on", m.statusMessage)
	}
	compact := m.View()
	if strings.Contains(compact, "───") {
		t.Fatalf("compact transcript still rendered turn separator:\n%s", compact)
	}
	if strings.Contains(m.statusMessage, "recognized but unavailable") {
		t.Fatalf("/compact fell through to unavailable fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/compact off")
	if m.compactTranscript {
		t.Fatal("compactTranscript = true after /compact off, want false")
	}
	if !strings.Contains(m.statusMessage, "compact off") {
		t.Fatalf("status after /compact off = %q, want compact off", m.statusMessage)
	}
	if restored := m.View(); !strings.Contains(restored, "───") {
		t.Fatalf("full transcript view missing turn separator after /compact off:\n%s", restored)
	}
}

func TestCompactSlashToggleAndUsage(t *testing.T) {
	tests := []struct {
		name        string
		initial     bool
		input       string
		wantCompact bool
		wantStatus  string
	}{
		{name: "bare toggles on", input: "/compact", wantCompact: true, wantStatus: "compact on"},
		{name: "toggle flips off", initial: true, input: "/compact toggle", wantCompact: false, wantStatus: "compact off"},
		{name: "invalid usage", input: "/compact maybe", wantCompact: false, wantStatus: "usage: /compact [on|off|toggle]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newCompactSlashModel(sub, tt.initial), tt.input)
			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if m.compactTranscript != tt.wantCompact {
				t.Fatalf("compactTranscript after %s = %v, want %v", tt.input, m.compactTranscript, tt.wantCompact)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.wantStatus)
			}
		})
	}
}

func TestCompactSlashCompletionsAndBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/com")
	for _, completion := range completions {
		if completion.Name == "compact" {
			if !completion.Available {
				t.Fatalf("completion %+v marked unavailable, want available", completion)
			}
			goto foundCompletion
		}
	}
	t.Fatalf("HermesSlashCommandCompletions(/com) = %+v, want compact", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "compact" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want compact", busy)
}

func TestCompactSlashKeepsTinyTerminalAutoCompact(t *testing.T) {
	m := newCompactSlashModel(&nopSubmitter{}, false)
	m.width = 5
	m.height = 12
	m.frame.History = compactSlashHistory()

	got := m.View()
	if strings.Contains(got, "───") {
		t.Fatalf("tiny terminal should remain auto-compact even when /compact is off:\n%s", got)
	}
}

func newCompactSlashModel(sub *nopSubmitter, compact bool) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, CompactTranscript: compact})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 96
	m.height = 28
	return m
}

func compactSlashHistory() []llm.Message {
	return []llm.Message{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "second question with enough words to make compact mode visibly collapse the transcript row into one line"},
		{Role: "assistant", Content: "second answer"},
	}
}

// ---- slash_completion_interaction_test.go ----

func TestSlashCompletionInteraction_UpDownOwnMenuBeforeHistory(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{OfflineSmoke: true})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20

	m.editor.SetValue("older prompt")
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.editor.SetValue("/r")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if got := m.editor.Value(); got != "/r" {
		t.Fatalf("Up with slash completions active changed editor to %q, want draft preserved and history not recalled", got)
	}

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.editor.Value(); got == "/r" || got == "older prompt" || !strings.HasPrefix(got, "/") {
		t.Fatalf("Enter after completion navigation editor = %q, want accepted slash completion, not history", got)
	}
}

func TestSlashCompletionInteraction_EnterAcceptsNonExactBeforeDispatch(t *testing.T) {
	sub := &nopSubmitter{}
	m := NewModelWithOptions(make(chan kernel.RenderFrame), sub.submit, func() {}, Options{OfflineSmoke: true})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/he")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.editor.Value(); got != "/help" {
		t.Fatalf("Enter on non-exact slash completion editor = %q, want /help", got)
	}
	if sub.calls != 0 {
		t.Fatalf("Enter on non-exact completion reached submitter %d times, want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("Enter on non-exact completion opened page %+v, want accept-only", *m.transientPage)
	}

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.editor.Value(); got != "" {
		t.Fatalf("second Enter after exact /help editor = %q, want dispatched and cleared", got)
	}
	if !strings.Contains(m.statusMessage, "Native TUI commands") {
		t.Fatalf("second Enter after exact /help status = %q, want dispatched slash help", m.statusMessage)
	}
}

func TestSlashCompletionInteraction_TabAddsArgumentSpaceAndNoPlaceholder(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		PromptTemplates: prompttemplates.Catalog{Templates: []prompttemplates.Template{{
			Name:         "zz-review",
			Description:  "review a scope",
			ArgumentHint: "<scope>",
			Body:         "review ${@:1}",
		}}},
	})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/zz")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if got := m.editor.Value(); got != "/zz-review " {
		t.Fatalf("Tab accepted template completion = %q, want command token plus trailing space only", got)
	}
	if strings.Contains(m.editor.Value(), "<scope>") {
		t.Fatalf("Tab inserted placeholder text into editor: %q", m.editor.Value())
	}
}

func TestSlashCompletionInteraction_EscapeDismissesMenuKeepsDraft(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/he")

	if view := m.View(); !strings.Contains(view, "Search /he") {
		t.Fatalf("precondition: slash completion menu not visible:\n%s", view)
	}

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEscape})
	if got := m.editor.Value(); got != "/he" {
		t.Fatalf("Escape changed draft to %q, want /he", got)
	}
	if view := m.View(); strings.Contains(view, "Search /he") {
		t.Fatalf("Escape did not dismiss slash completion menu:\n%s", view)
	}
}

func updateModelCompletionKey(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, cmd := m.Update(msg)
	runTestCmd(t, cmd)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	return updated
}

// ---- slash_completion_test.go ----

// TestHermesSlashCompletion_CommandPrefix proves typed slash prefixes resolve
// to matching canonical commands and aliases from internal/cli.CommandRegistry,
// preserving Hermes prompt_toolkit completion semantics: the leading "/" is
// stripped before prefix matching, completions are returned in stable
// alphabetical order, and an exact name match still surfaces (so the dropdown
// can stay open like Hermes does).
func TestHermesSlashCompletion_CommandPrefix(t *testing.T) {
	cases := []struct {
		input string
		want  []string // names without leading "/"
	}{
		{input: "/he", want: []string{"help"}},
		{input: "/r", want: collectRegistryNamesWithPrefix("r")},
		{input: "/", want: collectAllRegistryNames()},
		{input: "/RES", want: []string{"reset", "restart", "resume"}},
		{input: "/help", want: []string{"help"}},
		{input: "/no-such-command-xyzzy", want: nil},
	}
	for _, tc := range cases {
		got := HermesSlashCommandCompletions(tc.input)
		gotNames := completionNames(got)
		if !reflect.DeepEqual(gotNames, tc.want) {
			t.Errorf("HermesSlashCommandCompletions(%q) = %v, want %v", tc.input, gotNames, tc.want)
		}
	}
}

// TestHermesSlashCompletion_SubcommandPrefix proves reasoning subcommands
// surface in the Hermes-canonical order ("none", "minimal", "low", "medium",
// "high", "xhigh", "show", "hide", "on", "off") for `/reasoning ` (trailing
// space, no prefix) and that `/reasoning sh` filters down to the prefix-match
// subset preserving order.
func TestHermesSlashCompletion_SubcommandPrefix(t *testing.T) {
	wantAll := []string{"none", "minimal", "low", "medium", "high", "xhigh", "show", "hide", "on", "off"}
	got := HermesSlashSubcommandCompletions("/reasoning ")
	if !reflect.DeepEqual(completionNames(got), wantAll) {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning \") = %v, want %v", completionNames(got), wantAll)
	}

	wantSh := []string{"show"}
	gotSh := HermesSlashSubcommandCompletions("/reasoning sh")
	if !reflect.DeepEqual(completionNames(gotSh), wantSh) {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning sh\") = %v, want %v", completionNames(gotSh), wantSh)
	}

	// Exact subcommand match still surfaces so the user can keep editing the
	// completion menu (mirrors prompt_toolkit's _completion_text trailing-space
	// behavior at the helper level: the entry is kept; the UI layer is free to
	// append a trailing space).
	gotExact := HermesSlashSubcommandCompletions("/reasoning none")
	if !reflect.DeepEqual(completionNames(gotExact), []string{"none"}) {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning none\") = %v, want [none]", completionNames(gotExact))
	}

	// Subcommands only resolve while editing the first sub-token. After a
	// space inside the args (e.g. `/reasoning show extra`) no further static
	// subcommand completions are surfaced.
	gotPast := HermesSlashSubcommandCompletions("/reasoning show extra")
	if len(gotPast) != 0 {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning show extra\") = %v, want empty (past first sub-token)", completionNames(gotPast))
	}

	// Commands without a registered subcommand inventory return no static
	// subcommand completions; the dynamic /model, /skin, /personality menus
	// are intentionally not part of this slice.
	gotUnknownSub := HermesSlashSubcommandCompletions("/help ")
	if len(gotUnknownSub) != 0 {
		t.Errorf("HermesSlashSubcommandCompletions(\"/help \") = %v, want empty (no subcommand inventory)", completionNames(gotUnknownSub))
	}
}

func TestHermesSlashCompletion_LocalNativeHandlersAreAvailable(t *testing.T) {
	cases := []struct {
		input string
		name  string
	}{
		{input: "/bra", name: "branch"},
		{input: "/fo", name: "fork"},
		{input: "/sav", name: "save"},
		{input: "/cop", name: "copy"},
		{input: "/mou", name: "mouse"},
		{input: "/scr", name: "scroll"},
		{input: "/qui", name: "quit"},
		{input: "/exi", name: "exit"},
		{input: "/red", name: "redraw"},
		{input: "/det", name: "details"},
		{input: "/ind", name: "indicator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			completions := HermesSlashCommandCompletions(tc.input)
			for _, completion := range completions {
				if completion.Name != tc.name {
					continue
				}
				if !completion.Available {
					t.Fatalf("completion for %s = unavailable; want available for shipped native handler (all=%+v)", tc.name, completions)
				}
				return
			}
			t.Fatalf("HermesSlashCommandCompletions(%q) = %+v, want %s", tc.input, completions, tc.name)
		})
	}
}

// TestHermesSlashCompletion_UnavailableCommandsStillComplete proves
// recognized-but-unported commands appear in completion (so users can discover
// them) while EvaluateActiveTurnVerdict still classifies their dispatch as
// ActiveTurnPolicyUnavailable with explicit evidence — never letting the slash
// text leak to the kernel.
func TestHermesSlashCompletion_UnavailableCommandsStillComplete(t *testing.T) {
	completions := HermesSlashCommandCompletions("/too")
	names := completionNames(completions)
	if !containsString(names, "tools") {
		t.Fatalf("HermesSlashCommandCompletions(\"/too\") = %v, want to include unavailable command \"tools\"", names)
	}

	verdict := cli.EvaluateActiveTurnVerdict("/tools", false)
	if !verdict.Known {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Known = false, want true (registry recognizes /tools)")
	}
	if verdict.Allowed {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Allowed = true, want false for unavailable command")
	}
	if verdict.Policy != cli.ActiveTurnPolicyUnavailable {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Policy = %q, want %q", verdict.Policy, cli.ActiveTurnPolicyUnavailable)
	}
	if !strings.Contains(strings.ToLower(verdict.Evidence), "unavailable") {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Evidence = %q, want to mention unavailable", verdict.Evidence)
	}
}

func TestHermesSlashCompletion_BrowserSubcommands(t *testing.T) {
	got := HermesSlashSubcommandCompletions("/browser ")
	if !reflect.DeepEqual(completionNames(got), []string{"status", "connect"}) {
		t.Fatalf("HermesSlashSubcommandCompletions(/browser) = %v, want status/connect", completionNames(got))
	}
	verdict := cli.EvaluateActiveTurnVerdict("/browser status", false)
	if !verdict.Known || !verdict.Allowed || verdict.Policy != cli.ActiveTurnPolicyBypass {
		t.Fatalf("EvaluateActiveTurnVerdict(/browser status) = %+v, want local ported bypass command", verdict)
	}
}

func TestRenderSlashCompletionMenu_CommandPrefix(t *testing.T) {
	got := renderSlashCompletionMenu("/n", 72)
	if !strings.Contains(got, "/new") {
		t.Fatalf("renderSlashCompletionMenu(/n) missing /new:\n%s", got)
	}
	if !strings.Contains(got, "Start a fresh session") {
		t.Fatalf("renderSlashCompletionMenu(/n) missing description:\n%s", got)
	}
	if strings.Contains(got, "mouse: disabled") {
		t.Fatalf("completion menu leaked unrelated status text:\n%s", got)
	}
}

func TestRenderSlashCompletionMenu_Subcommands(t *testing.T) {
	got := renderSlashCompletionMenu("/browser ", 72)
	if !strings.Contains(got, "status") || !strings.Contains(got, "connect") {
		t.Fatalf("renderSlashCompletionMenu(/browser ) missing static subcommands:\n%s", got)
	}
}

func TestRenderSlashCompletionMenu_SearchListChrome(t *testing.T) {
	got := renderSlashCompletionMenuWithSkin("/sta", 52, BuiltinSkins()["poseidon"])
	for _, want := range []string{"Search /sta", "❯", "/status", "↑/↓ select", "Enter complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("search completion menu missing %q:\n%s", want, got)
		}
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 52 {
			t.Fatalf("search completion line width %d exceeds 52:\n%q\n\n%s", w, line, got)
		}
	}
}

func TestRenderSlashCompletionMenu_NarrowWidthKeepsChromeCompact(t *testing.T) {
	got := renderSlashCompletionMenu("/", 36)
	lines := strings.Split(got, "\n")
	if len(lines) > 6 {
		t.Fatalf("narrow slash completion menu rendered %d lines, want at most 6 to avoid swallowing the chat UI:\n%s", len(lines), got)
	}
	for _, line := range lines {
		if w := lipgloss.Width(line); w > 36 {
			t.Fatalf("narrow completion line width %d exceeds 36:\n%q\n\n%s", w, line, got)
		}
	}
}

// TestHermesSlashCompletion_AutoSuggest proves the inline ghost suffix matches
// what Hermes' SlashCommandAutoSuggest returns: the unique remaining tail of a
// command or subcommand name, or empty when ambiguous, unknown, or already
// complete. Mirrors hermes_cli/commands.py:SlashCommandAutoSuggest.
func TestHermesSlashCompletion_AutoSuggest(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		// Unique command prefix → suffix to ghost.
		{input: "/he", want: "lp"},
		{input: "/restar", want: "t"},
		// Ambiguous prefix → no suggestion (multiple commands match).
		{input: "/re", want: ""},
		// Already a complete command → no suggestion.
		{input: "/help", want: ""},
		// Unknown prefix → no suggestion.
		{input: "/zzz", want: ""},
		// Empty / non-slash input → no suggestion.
		{input: "", want: ""},
		{input: "hello", want: ""},
		// Subcommand suggestion: unique prefix after command + space.
		{input: "/reasoning mi", want: "nimal"},
		{input: "/reasoning xh", want: "igh"},
		// Ambiguous subcommand → no suggestion (`/reasoning ` matches all 10).
		{input: "/reasoning ", want: ""},
		// Subcommand exact match → no suggestion.
		{input: "/reasoning none", want: ""},
		// Subcommand suggestion only resolves on the first sub-token.
		{input: "/reasoning show ex", want: ""},
		// Commands without a subcommand inventory return no subcommand
		// suggestion even after the space.
		{input: "/help arg", want: ""},
	}
	for _, tc := range cases {
		got := HermesSlashAutoSuggest(tc.input)
		if got != tc.want {
			t.Errorf("HermesSlashAutoSuggest(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestHermesSlashCompletion_DeterministicAndPure proves the helpers run with
// no provider, config, plugin, or filesystem dependency: repeated calls return
// the exact same slice of completions, and the slice is independent across
// calls (mutating one does not poison cached state).
func TestHermesSlashCompletion_DeterministicAndPure(t *testing.T) {
	first := HermesSlashCommandCompletions("/r")
	second := HermesSlashCommandCompletions("/r")
	if !reflect.DeepEqual(completionNames(first), completionNames(second)) {
		t.Errorf("HermesSlashCommandCompletions is non-deterministic: %v vs %v", completionNames(first), completionNames(second))
	}
	// Mutate the returned slice; a subsequent call must still return the
	// canonical list (proves we did not hand out a shared backing array).
	if len(first) > 0 {
		first[0] = SlashCompletion{Name: "POISON"}
	}
	third := HermesSlashCommandCompletions("/r")
	if !reflect.DeepEqual(completionNames(second), completionNames(third)) {
		t.Errorf("HermesSlashCommandCompletions leaks shared state: %v vs %v", completionNames(second), completionNames(third))
	}

	subFirst := HermesSlashSubcommandCompletions("/reasoning ")
	subSecond := HermesSlashSubcommandCompletions("/reasoning ")
	if !reflect.DeepEqual(completionNames(subFirst), completionNames(subSecond)) {
		t.Errorf("HermesSlashSubcommandCompletions is non-deterministic")
	}
}

// completionNames extracts the Name field from a slice of SlashCompletion so
// table tests can compare on a single dimension while still proving the
// returned struct carries a name.
func completionNames(c []SlashCompletion) []string {
	if len(c) == 0 {
		return nil
	}
	out := make([]string, 0, len(c))
	for _, x := range c {
		out = append(out, x.Name)
	}
	return out
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// collectRegistryNamesWithPrefix returns all canonical command names plus
// aliases that match the prefix, in stable alphabetical order. Used by the
// command-prefix table test to avoid hard-coding a list that drifts as the
// registry grows.
func collectRegistryNamesWithPrefix(prefix string) []string {
	prefix = strings.ToLower(prefix)
	seen := map[string]struct{}{}
	for _, cmd := range cli.CommandRegistry {
		if strings.HasPrefix(cmd.Name, prefix) {
			seen[cmd.Name] = struct{}{}
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				seen[alias] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func collectAllRegistryNames() []string {
	seen := map[string]struct{}{}
	for _, cmd := range cli.CommandRegistry {
		seen[cmd.Name] = struct{}{}
		for _, alias := range cmd.Aliases {
			seen[alias] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---- slash_details_test.go ----

func TestDetailsSlashControlsThinkingAndToolVisibilityWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newDetailsSlashModel(sub)

	initial := m.View()
	if !strings.Contains(initial, "terminal: ls -la") {
		t.Fatalf("initial view missing tool progress before /details hidden:\n%s", initial)
	}

	m = enterSlashDispatchBehavior(t, m, "/details hidden")
	if sub.calls != 0 {
		t.Fatalf("/details reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /details = %q, want cleared", got)
	}
	if m.detailsState.Global != DetailsModeHidden || !m.detailsState.CommandOverride {
		t.Fatalf("details state after /details hidden = %+v, want hidden command override", m.detailsState)
	}
	if !strings.Contains(m.statusMessage, "details: hidden") {
		t.Fatalf("status after /details hidden = %q, want hidden evidence", m.statusMessage)
	}
	if got := m.View(); strings.Contains(got, "terminal: ls -la") {
		t.Fatalf("/details hidden still rendered tool progress:\n%s", got)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/details fell through to fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/details tools expanded")
	if got := m.detailsState.SectionMode(DetailsSectionTools); got != DetailsModeExpanded {
		t.Fatalf("tools section after override = %q, want expanded", got)
	}
	if got := m.View(); !strings.Contains(got, "terminal: ls -la") {
		t.Fatalf("/details tools expanded did not restore tool progress:\n%s", got)
	}

	m = enterSlashDispatchBehavior(t, m, "/details tools reset")
	if _, ok := m.detailsState.Sections[DetailsSectionTools]; ok {
		t.Fatalf("tools override still present after reset: %+v", m.detailsState.Sections)
	}
	if got := m.View(); strings.Contains(got, "terminal: ls -la") {
		t.Fatalf("/details tools reset should fall back to global hidden:\n%s", got)
	}
}

func TestDetailsSlashKeepsQuietActiveTranscriptQuiet(t *testing.T) {
	m := newDetailsSlashModel(&nopSubmitter{})
	m.frame.SoulEvents = nil
	if got := m.View(); strings.Contains(got, "Reasoning") {
		t.Fatalf("active view rendered noisy fallback thinking before /details hidden:\n%s", got)
	}

	m = enterSlashDispatchBehavior(t, m, "/details hidden")
	if got := m.View(); strings.Contains(got, "Reasoning") {
		t.Fatalf("/details hidden rendered fallback thinking:\n%s", got)
	}
}

func TestDetailsSlashToggleSectionUsageAndCompletions(t *testing.T) {
	m := newDetailsSlashModel(&nopSubmitter{})
	m = enterSlashDispatchBehavior(t, m, "/details toggle")
	if m.detailsState.Global != DetailsModeExpanded || !m.detailsState.CommandOverride {
		t.Fatalf("details after toggle from default = %+v, want expanded command override", m.detailsState)
	}
	m = enterSlashDispatchBehavior(t, m, "/details activity collapsed")
	if got := m.detailsState.SectionMode(DetailsSectionActivity); got != DetailsModeCollapsed {
		t.Fatalf("activity override = %q, want collapsed", got)
	}
	m = enterSlashDispatchBehavior(t, m, "/details tools blink")
	if !strings.Contains(m.statusMessage, "usage: /details <section> [hidden|collapsed|expanded|reset]") {
		t.Fatalf("status after invalid section mode = %q, want section usage", m.statusMessage)
	}
	m = enterSlashDispatchBehavior(t, m, "/details nope")
	if !strings.Contains(m.statusMessage, "usage: /details [hidden|collapsed|expanded|cycle]") {
		t.Fatalf("status after invalid global mode = %q, want global usage", m.statusMessage)
	}

	var detailsCompletion *SlashCompletion
	for _, completion := range HermesSlashCommandCompletions("/det") {
		if completion.Name == "details" {
			c := completion
			detailsCompletion = &c
			break
		}
	}
	if detailsCompletion == nil || !detailsCompletion.Available {
		t.Fatalf("/det completion = %+v, want available details", detailsCompletion)
	}
	wantSubs := []string{"hidden", "collapsed", "expanded", "cycle", "toggle", "thinking", "tools", "subagents", "activity"}
	if got := completionNames(HermesSlashSubcommandCompletions("/details ")); !reflect.DeepEqual(got, wantSubs) {
		t.Fatalf("/details subcommands = %v, want %v", got, wantSubs)
	}
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "details" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want details", busy)
}

func newDetailsSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "openai/gpt-4.1",
		SessionID: "sess-details",
		History:   []llm.Message{{Role: "user", Content: "inspect details"}},
		SoulEvents: []kernel.SoulEntry{
			{Text: "tool: terminal: ls -la"},
		},
	}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_dispatch_behavior_test.go ----

func TestHermesSlashDispatchBehavior_LocalHandlersStillRun(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		withHistory bool
		wantStatus  string
	}{
		{name: "save", input: "/save", wantStatus: "save: no conversation"},
		{name: "branch", input: "/branch branch-name", wantStatus: "branch: no conversation"},
		{name: "browser", input: "/browser status", wantStatus: "browser:"},
		{name: "mouse", input: "/mouse on", wantStatus: "mouse tracking on"},
		{name: "scroll", input: "/scroll off", wantStatus: "mouse tracking off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			if tt.withHistory {
				m.frame.History = []llm.Message{{Role: "user", Content: "hello"}}
				m.frame.SessionID = "sess-parent"
			}

			m = enterSlashDispatchBehavior(t, m, tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want it to contain %q", tt.input, m.statusMessage, tt.wantStatus)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_RedrawClearsVisibleFrameLocally(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSlashDispatchBehaviorModel(sub)
	m.frame.History = []llm.Message{{Role: "user", Content: "keep in kernel, clear from view"}}
	m.frame.DraftText = "streaming draft"
	m.frame.LastError = "stale terminal error"
	m.frame.SessionID = "sess-redraw"

	m = enterSlashDispatchBehavior(t, m, "/redraw")

	if sub.calls != 0 {
		t.Fatalf("/redraw reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /redraw = %q, want cleared", got)
	}
	if len(m.frame.History) != 0 || m.frame.DraftText != "" || m.frame.LastError != "" {
		t.Fatalf("/redraw did not clear visible frame: history=%d draft=%q err=%q", len(m.frame.History), m.frame.DraftText, m.frame.LastError)
	}
	if m.frame.SessionID != "sess-redraw" {
		t.Fatalf("/redraw SessionID = %q, want preserved", m.frame.SessionID)
	}
	if !strings.Contains(strings.ToLower(m.statusMessage), "ui redrawn") {
		t.Fatalf("status after /redraw = %q, want ui redrawn", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/redraw fell through to unavailable fallback: %q", m.statusMessage)
	}
}

func TestHermesSlashDispatchBehavior_QuitExitsLocally(t *testing.T) {
	for _, input := range []string{"/quit", "/exit"} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			m.editor.SetValue(input)

			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			updated, ok := next.(Model)
			if !ok {
				t.Fatalf("Update returned %T, want tui.Model", next)
			}
			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if got := updated.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			if !cmdEmitsQuit(cmd) {
				t.Fatalf("%s did not emit tea.Quit", input)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_SkillsSlashRunsLocallyWhileBusy(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSlashDispatchBehaviorModel(sub)
	m.inFlight = true
	m.frame.Phase = kernel.PhaseStreaming

	m = enterSlashDispatchBehavior(t, m, "/skills search planner")

	if sub.calls != 0 {
		t.Fatalf("/skills search reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /skills search = %q, want cleared", got)
	}
	status := strings.ToLower(m.statusMessage)
	if !strings.Contains(status, "skill hub search") || strings.Contains(status, "recognized but unavailable") {
		t.Fatalf("status after /skills search = %q, want local skills command output", m.statusMessage)
	}
}

func TestHermesSlashDispatchBehavior_SkillsInstallRunsLocally(t *testing.T) {
	sub := &nopSubmitter{}
	calls := 0
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillsCommand: func(input string) string {
			calls++
			if input != "/skills install https://example.com/SKILL.md --name tui-skill" {
				t.Fatalf("SkillsCommand input = %q", input)
			}
			return "url_skill_installed: installed tui-skill"
		},
	})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, "/skills install https://example.com/SKILL.md --name tui-skill")

	if calls != 1 {
		t.Fatalf("SkillsCommand calls = %d, want 1", calls)
	}
	if sub.calls != 0 {
		t.Fatalf("/skills install reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /skills install = %q, want cleared", got)
	}
	status := strings.ToLower(m.statusMessage)
	if !strings.Contains(status, "url_skill_installed") || !strings.Contains(status, "tui-skill") {
		t.Fatalf("status after /skills install = %q, want install evidence", m.statusMessage)
	}
}

func TestHermesSlashDispatchBehavior_KnownUnhandledCommandsNeverSubmit(t *testing.T) {
	for _, input := range []string{
		"/provider openrouter",
		"/image ./diagram.png",
		"/tools list",
		"/rollback",
	} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newSlashDispatchBehaviorModel(sub), input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			status := strings.ToLower(m.statusMessage)
			if !strings.Contains(status, "recognized") {
				t.Fatalf("status after %s = %q, want recognized-command evidence", input, m.statusMessage)
			}
			if !strings.Contains(status, "native tui") && !strings.Contains(status, "gateway") {
				t.Fatalf("status after %s = %q, want native TUI or gateway degraded-mode evidence", input, m.statusMessage)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_UnknownAndAmbiguousSlashGuidance(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		wantStatus        string
		dismissCompletion bool
	}{
		{name: "unknown", input: "/no-such-command-xyzzy", wantStatus: "unknown command"},
		{name: "ambiguous", input: "/s", wantStatus: "ambiguous command", dismissCompletion: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			if tt.dismissCompletion {
				m.editor.SetValue(tt.input)
				m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyEscape})
				m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
			} else {
				m = enterSlashDispatchBehavior(t, m, tt.input)
			}

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if !strings.Contains(strings.ToLower(m.statusMessage), tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q guidance", tt.input, m.statusMessage, tt.wantStatus)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_MutatingCommandsDoNotFallback(t *testing.T) {
	mutating := []string{
		"/background run later",
		"/branch branch-name",
		"/browser status",
		"/busy queue",
		"/fast",
		"/model gpt-5.2",
		"/new",
		"/queue later",
		"/reasoning high",
		"/rollback",
		"/stop",
		"/title new title",
		"/tools disable terminal",
		"/undo",
		"/verbose",
		"/voice",
		"/yolo",
	}
	for _, input := range mutating {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newSlashDispatchBehaviorModel(sub), input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			if strings.TrimSpace(m.statusMessage) == "" {
				t.Fatalf("status after %s is empty, want visible routing/degraded evidence", input)
			}
		})
	}
}

func newSlashDispatchBehaviorModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

func enterSlashDispatchBehavior(t *testing.T, m Model, input string) Model {
	t.Helper()
	m.editor.SetValue(input)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	runTestCmd(t, cmd)
	return updated
}

func cmdEmitsQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, nested := range msg {
			if cmdEmitsQuit(nested) {
				return true
			}
		}
	}
	return false
}

// ---- slash_dispatch_test.go ----

func TestSlashRegistry_DispatchRoutesToRegisteredHandler(t *testing.T) {
	registry := NewSlashRegistry()

	var got string
	registry.Register("foo", func(input string, model *Model) SlashResult {
		got = input
		return SlashResult{Handled: true, StatusMessage: "foo ok"}
	})

	res := registry.Dispatch("/foo bar", nil)
	if !res.Handled {
		t.Fatalf("Handled = false, want true")
	}
	if got != "/foo bar" {
		t.Fatalf("handler received input %q, want %q (full input including slash)", got, "/foo bar")
	}
	if res.StatusMessage != "foo ok" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "foo ok")
	}
}

func TestSlashRegistry_UnknownSlashIsNotHandled(t *testing.T) {
	registry := NewSlashRegistry()

	res := registry.Dispatch("/unknown some args", nil)
	if res.Handled {
		t.Fatalf("Handled = true for /unknown, want false (must fall through to submit)")
	}
}

func TestSlashRegistry_NonSlashInputIsNotHandled(t *testing.T) {
	registry := NewDefaultSlashRegistry()

	for _, input := range []string{"", "   ", "hello world", "save"} {
		t.Run(input, func(t *testing.T) {
			res := registry.Dispatch(input, nil)
			if res.Handled {
				t.Fatalf("Dispatch(%q) Handled = true, want false (no leading slash)", input)
			}
		})
	}
}

func TestSlashRegistry_HelpHandlerShowsNativeInventory(t *testing.T) {
	res := NewDefaultSlashRegistry().Dispatch("/help", nil)
	if !res.Handled {
		t.Fatalf("Handled = false for /help, want true")
	}
	if res.StatusMessage == "" {
		t.Fatal("StatusMessage is empty, want native TUI help inventory")
	}
	for _, want := range []string{
		"Native TUI commands",
		"/model",
		"/save",
		"/branch",
		"/mouse",
		"recognized but unavailable",
	} {
		if !strings.Contains(res.StatusMessage, want) {
			t.Fatalf("StatusMessage = %q, want it to contain %q", res.StatusMessage, want)
		}
	}
	if strings.Contains(res.StatusMessage, "is recognized but unavailable in the native TUI") {
		t.Fatalf("/help fell through to unavailable fallback: %q", res.StatusMessage)
	}
}

func TestSlashRegistry_HelpAdvertisedWhileBusy(t *testing.T) {
	names := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range names {
		if name == "help" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want help", names)
}

func TestSlashRegistry_MouseHandlerMigrationParity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current bool
		want    string
	}{
		{name: "on", input: "/mouse on", current: false, want: "mouse tracking on"},
		{name: "off", input: "/mouse off", current: true, want: "mouse tracking off"},
		{name: "invalid", input: "/mouse foo", current: true, want: "usage: /mouse [on|off|toggle]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &mouseModeRecorder{}
			m := NewModelWithOptions(
				make(chan kernel.RenderFrame),
				func(string) {},
				func() {},
				Options{MouseTracking: tt.current, MouseModeCmd: rec.cmd},
			)

			registry := NewDefaultSlashRegistry()
			res := registry.Dispatch(tt.input, &m)

			if !res.Handled {
				t.Fatalf("Handled = false for %q, want true (parity with parseMouseTrackingSlash)", tt.input)
			}
			if res.StatusMessage != tt.want {
				t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, tt.want)
			}
		})
	}
}

// ---- slash_history_test.go ----

func TestHistorySlashRendersCurrentTranscriptPageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newHistorySlashModel(sub)
	longAssistant := "assistant " + strings.Repeat("x", 96)
	m.frame.History = []llm.Message{
		{Role: "system", Content: "hidden system row"},
		{Role: "user", Content: "hello from Juan"},
		{Role: "assistant", Content: longAssistant},
		{Role: "tool", Name: "read_file", Content: "tool output stays out of /history"},
		{Role: "assistant", ToolCalls: []llm.ToolCall{{Name: "read_file"}}},
	}

	m = enterSlashDispatchBehavior(t, m, "/history 12")

	if sub.calls != 0 {
		t.Fatalf("/history reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /history = %q, want cleared", got)
	}
	if m.transientPage == nil {
		t.Fatal("/history did not open a transient page")
	}
	if m.transientPage.Title != "History" {
		t.Fatalf("page title = %q, want History", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range []string{"[You #1]", "hello from Juan", "[Gormes #2]", "[Gormes #3]", "(1 tool calls)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("history page body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "hidden system row") || strings.Contains(body, "tool output stays out") {
		t.Fatalf("history page body included non-user/assistant rows:\n%s", body)
	}
	if strings.Contains(body, strings.Repeat("x", 96)) || !strings.Contains(body, "…") {
		t.Fatalf("history page body did not clip long assistant text:\n%s", body)
	}
	view := m.View()
	if !strings.Contains(view, "History") || !strings.Contains(view, "hello from Juan") {
		t.Fatalf("View() did not render transient history page:\n%s", view)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/history fell through to fallback: %q", m.statusMessage)
	}
}

func TestHistorySlashEmptyConversationAndDismiss(t *testing.T) {
	sub := &nopSubmitter{}
	m := newHistorySlashModel(sub)

	m = enterSlashDispatchBehavior(t, m, "/history")
	if sub.calls != 0 {
		t.Fatalf("empty /history reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("empty /history page = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "no conversation yet") {
		t.Fatalf("empty /history status = %q, want no conversation evidence", m.statusMessage)
	}

	m.frame.History = []llm.Message{{Role: "user", Content: "keep visible until dismissed"}}
	m = enterSlashDispatchBehavior(t, m, "/history")
	if m.transientPage == nil {
		t.Fatal("/history with conversation did not open a transient page")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	if updated.transientPage != nil {
		t.Fatalf("Escape left transient page open: %+v", *updated.transientPage)
	}
}

func TestHistorySlashCompletionsAndBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/his")
	for _, completion := range completions {
		if completion.Name != "history" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/his) = %+v, want history", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "history" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want history", busy)
}

func newHistorySlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-history"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_indicator_test.go ----

func TestIndicatorSlashControlsBusyHintWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newIndicatorSlashModel(sub)

	m = enterSlashDispatchBehavior(t, m, "/indicator unicode")
	if sub.calls != 0 {
		t.Fatalf("/indicator reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /indicator = %q, want cleared", got)
	}
	if m.indicatorStyle != IndicatorStyleUnicode {
		t.Fatalf("indicatorStyle after /indicator unicode = %q, want %q", m.indicatorStyle, IndicatorStyleUnicode)
	}
	if !strings.Contains(m.statusMessage, "indicator → unicode") {
		t.Fatalf("status after /indicator unicode = %q, want indicator evidence", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/indicator fell through to fallback: %q", m.statusMessage)
	}

	hint := m.renderHermesHint()
	if !strings.Contains(hint, "⠋") {
		t.Fatalf("unicode indicator hint = %q, want braille frame", hint)
	}
	if strings.Contains(hint, "◕") || strings.Contains(hint, "⚕") {
		t.Fatalf("unicode indicator hint leaked another style: %q", hint)
	}

	m = enterSlashDispatchBehavior(t, m, "/indicator")
	if !strings.Contains(m.statusMessage, "indicator: unicode") {
		t.Fatalf("bare /indicator status = %q, want current style", m.statusMessage)
	}
}

func TestIndicatorSlashUsageAndCompletions(t *testing.T) {
	m := newIndicatorSlashModel(&nopSubmitter{})
	m.indicatorStyle = IndicatorStyleEmoji

	m = enterSlashDispatchBehavior(t, m, "/indicator sparkle")
	if m.indicatorStyle != IndicatorStyleEmoji {
		t.Fatalf("invalid /indicator changed style to %q, want %q", m.indicatorStyle, IndicatorStyleEmoji)
	}
	if !strings.Contains(m.statusMessage, "usage: /indicator [ascii|emoji|kaomoji|unicode]") {
		t.Fatalf("invalid /indicator status = %q, want usage", m.statusMessage)
	}

	completions := HermesSlashCommandCompletions("/ind")
	for _, completion := range completions {
		if completion.Name != "indicator" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/ind) = %+v, want indicator", completions)

foundCompletion:
	got := completionNames(HermesSlashSubcommandCompletions("/indicator "))
	want := []string{"ascii", "emoji", "kaomoji", "unicode"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("HermesSlashSubcommandCompletions(/indicator ) = %v, want %v", got, want)
	}

	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "indicator" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want indicator", busy)
}

func newIndicatorSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseStreaming, SessionID: "sess-indicator"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_kanban_test.go ----

func TestKanbanSlashDispatchUsesInjectedRunner(t *testing.T) {
	var gotInput string
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		KanbanSlash: func(input string) (string, error) {
			gotInput = input
			return "No Kanban tasks.", nil
		},
	})

	res := NewDefaultSlashRegistry().Dispatch("/kanban list", &m)
	if !res.Handled {
		t.Fatal("Handled = false for /kanban, want native TUI handler")
	}
	if gotInput != "/kanban list" {
		t.Fatalf("runner input = %q, want full slash input", gotInput)
	}
	if res.StatusMessage != "No Kanban tasks." {
		t.Fatalf("StatusMessage = %q, want command output", res.StatusMessage)
	}
}

func TestKanbanSlashDispatchConsumesEditorWithoutSubmit(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSlashDispatchBehaviorModel(sub)
	m.kanbanSlash = func(string) (string, error) {
		return "kanban initialized at /tmp/gormes/kanban.db", nil
	}

	m = enterSlashDispatchBehavior(t, m, "/kanban init")
	if sub.calls != 0 {
		t.Fatalf("/kanban reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /kanban = %q, want cleared", got)
	}
	if !strings.Contains(m.statusMessage, "kanban initialized") {
		t.Fatalf("status after /kanban = %q, want command output", m.statusMessage)
	}
}

func TestKanbanSlashDispatchFailureIsEvidence(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		KanbanSlash: func(string) (string, error) {
			return "", errors.New("kanban db unavailable")
		},
	})

	res := NewDefaultSlashRegistry().Dispatch("/kanban list", &m)
	if !res.Handled {
		t.Fatal("Handled = false for /kanban error, want consumed")
	}
	if !strings.Contains(res.StatusMessage, "kanban: kanban db unavailable") {
		t.Fatalf("StatusMessage = %q, want kanban error evidence", res.StatusMessage)
	}
}

func TestKanbanSlashIsBusyAvailable(t *testing.T) {
	names := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range names {
		if name == "kanban" {
			return
		}
	}
	t.Fatalf("busy-available slashes = %v, want kanban", names)
}

func TestKanbanSlashStatusIsBounded(t *testing.T) {
	long := strings.Repeat("task\n", 240)
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		KanbanSlash: func(string) (string, error) {
			return long, nil
		},
	})

	res := NewDefaultSlashRegistry().Dispatch("/kanban list", &m)
	if len(res.StatusMessage) > maxKanbanSlashStatusRunes+len("...") {
		t.Fatalf("status length = %d, want bounded to %d", len(res.StatusMessage), maxKanbanSlashStatusRunes)
	}
}

// ---- slash_logs_test.go ----

func TestLogsSlashRendersGatewayTailPageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var gotLimit int
	m := newLogsSlashModel(sub, func(limit int) (string, error) {
		gotLimit = limit
		return "gateway line one\ngateway line two", nil
	})

	m = enterSlashDispatchBehavior(t, m, "/logs 500")

	if sub.calls != 0 {
		t.Fatalf("/logs reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /logs = %q, want cleared", got)
	}
	if gotLimit != 80 {
		t.Fatalf("/logs 500 requested limit %d, want Hermes clamp to 80", gotLimit)
	}
	if m.transientPage == nil {
		t.Fatal("/logs did not open a transient page")
	}
	if m.transientPage.Title != "Logs" {
		t.Fatalf("page title = %q, want Logs", m.transientPage.Title)
	}
	for _, want := range []string{"gateway line one", "gateway line two"} {
		if !strings.Contains(m.transientPage.Body, want) {
			t.Fatalf("logs page body missing %q:\n%s", want, m.transientPage.Body)
		}
	}
	view := m.View()
	if !strings.Contains(view, "Logs") || !strings.Contains(view, "gateway line one") {
		t.Fatalf("View() did not render transient logs page:\n%s", view)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/logs fell through to fallback: %q", m.statusMessage)
	}
}

func TestLogsSlashNoLogsAndBusyAvailability(t *testing.T) {
	sub := &nopSubmitter{}
	m := newLogsSlashModel(sub, func(limit int) (string, error) {
		if limit != 20 {
			t.Fatalf("/logs default limit = %d, want 20", limit)
		}
		return "", nil
	})

	m = enterSlashDispatchBehavior(t, m, "/logs 0")
	if sub.calls != 0 {
		t.Fatalf("/logs with empty tail reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("/logs with empty tail page = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "no gateway logs") {
		t.Fatalf("/logs empty status = %q, want no gateway logs", m.statusMessage)
	}

	completions := HermesSlashCommandCompletions("/lo")
	for _, completion := range completions {
		if completion.Name != "logs" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/lo) = %+v, want logs", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "logs" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want logs", busy)
}

func newLogsSlashModel(sub *nopSubmitter, tail GatewayLogTailFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-logs"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, GatewayLogTail: tail})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_model_test.go ----

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
	for _, tc := range []struct {
		input             string
		acceptsCompletion bool
	}{
		{input: "/model"},
		{input: "/m", acceptsCompletion: true},
	} {
		t.Run(tc.input, func(t *testing.T) {
			sub := &nopSubmitter{}
			setter := &recordingSetSessionModel{}
			m := newModelSlashTestModel(sub, setter, fakeModelCatalog, nil)

			if tc.acceptsCompletion {
				m.editor.SetValue(tc.input)
				m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
				if got := m.editor.Value(); got != "/model" {
					t.Fatalf("first Enter after %s editor = %q, want accepted /model completion", tc.input, got)
				}
				m = updateModelSlashKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
			} else {
				m = enterModelSlash(t, m, tc.input)
			}

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tc.input, sub.calls)
			}
			if setter.calls != 0 {
				t.Fatalf("%s called SetSessionModel %d time(s), want 0 before confirmation", tc.input, setter.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tc.input, got)
			}
			if m.modelPicker == nil {
				t.Fatalf("%s did not open model picker", tc.input)
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

// ---- slash_reset_test.go ----

type recordingSessionReset struct {
	calls int
	err   error
}

func (r *recordingSessionReset) call() error {
	r.calls++
	return r.err
}

func TestSessionSlashClearAndNewResetWithoutSubmitting(t *testing.T) {
	for _, tt := range []struct {
		input      string
		wantStatus string
	}{
		{input: "/clear", wantStatus: "session cleared"},
		{input: "/new", wantStatus: "new session started"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			reset := &recordingSessionReset{}
			sub := &nopSubmitter{}
			m := newSessionResetModel(sub, reset.call)
			m.frame.History = []llm.Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}}
			m.frame.DraftText = "draft"
			m.frame.LastError = "boom"
			m.frame.SessionID = "sess-frame"
			m.sessionID = "sess-local"

			m = enterSlashDispatchBehavior(t, m, tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if reset.calls != 1 {
				t.Fatalf("SessionResetFunc calls = %d, want 1", reset.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if len(m.frame.History) != 0 || m.frame.DraftText != "" || m.frame.LastError != "" || m.frame.SessionID != "" || m.sessionID != "" {
				t.Fatalf("visible session state not cleared after %s: frame=%+v sessionID=%q", tt.input, m.frame, m.sessionID)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.wantStatus)
			}
			if strings.Contains(m.statusMessage, "recognized but unavailable") {
				t.Fatalf("%s fell through to unavailable fallback: %q", tt.input, m.statusMessage)
			}
		})
	}
}

func TestSessionSlashResetUnavailableAndErrorsDoNotLeak(t *testing.T) {
	for _, tt := range []struct {
		name       string
		input      string
		reset      *recordingSessionReset
		wantStatus string
		wantCalls  int
	}{
		{name: "missing reset seam", input: "/clear", wantStatus: "clear: reset unavailable", wantCalls: 0},
		{name: "reset error", input: "/new", reset: &recordingSessionReset{err: errors.New("db locked")}, wantStatus: "new: reset failed: db locked", wantCalls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			var resetFn SessionResetFunc
			if tt.reset != nil {
				resetFn = tt.reset.call
			}
			m := newSessionResetModel(sub, resetFn)
			m.frame.History = []llm.Message{{Role: "user", Content: "keep me"}}
			m.frame.SessionID = "sess-frame"

			m = enterSlashDispatchBehavior(t, m, tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if tt.reset != nil && tt.reset.calls != tt.wantCalls {
				t.Fatalf("SessionResetFunc calls = %d, want %d", tt.reset.calls, tt.wantCalls)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.wantStatus)
			}
			if len(m.frame.History) != 1 || m.frame.SessionID != "sess-frame" {
				t.Fatalf("failed reset should preserve visible session, got frame=%+v", m.frame)
			}
		})
	}
}

func TestSessionSlashResetRejectedWhileTurnRunning(t *testing.T) {
	reset := &recordingSessionReset{}
	sub := &nopSubmitter{}
	m := newSessionResetModel(sub, reset.call)
	m.inFlight = true
	m.frame.Phase = kernel.PhaseStreaming
	m.frame.History = []llm.Message{{Role: "user", Content: "keep me"}}

	m = enterSlashDispatchBehavior(t, m, "/new")

	if sub.calls != 0 {
		t.Fatalf("/new reached Submitter %d time(s), want 0", sub.calls)
	}
	if reset.calls != 0 {
		t.Fatalf("SessionResetFunc calls = %d, want 0 while turn is running", reset.calls)
	}
	if !strings.Contains(m.statusMessage, "interrupt the current turn before trying to switch sessions") {
		t.Fatalf("status = %q, want busy session-switch guidance", m.statusMessage)
	}
	if len(m.frame.History) != 1 {
		t.Fatalf("running reset should preserve history, got %+v", m.frame.History)
	}
}

func TestSessionSlashCompletionsMarkClearAndNewAvailable(t *testing.T) {
	for _, input := range []string{"/cle", "/ne"} {
		t.Run(input, func(t *testing.T) {
			completions := HermesSlashCommandCompletions(input)
			if len(completions) == 0 {
				t.Fatalf("HermesSlashCommandCompletions(%q) returned no completions", input)
			}
			for _, completion := range completions {
				if completion.Name == "clear" || completion.Name == "new" {
					if !completion.Available {
						t.Fatalf("completion %+v marked unavailable, want available", completion)
					}
					return
				}
			}
			t.Fatalf("HermesSlashCommandCompletions(%q) = %+v, want clear/new", input, completions)
		})
	}
}

func newSessionResetModel(sub *nopSubmitter, reset SessionResetFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionReset: reset})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

func TestSessionSlashRegistryConsumesClearAndNew(t *testing.T) {
	for _, input := range []string{"/clear", "/new"} {
		t.Run(input, func(t *testing.T) {
			res := NewDefaultSlashRegistry().Dispatch(input, &Model{sessionReset: func() error { return nil }})
			if !res.Handled {
				t.Fatalf("Dispatch(%q) Handled = false, want true", input)
			}
			if strings.Contains(res.StatusMessage, "recognized but unavailable") {
				t.Fatalf("Dispatch(%q) fell through to fallback: %q", input, res.StatusMessage)
			}
		})
	}
}

var _ tea.Model = Model{}

// ---- slash_save_test.go ----

// recordingExportFunc captures invocations of SessionExportFunc and returns
// the configured path/error or invokes a side-effect (used by the partial
// file test to seed a real file that the handler must os.Remove).
type recordingExportFunc struct {
	calls        int
	gotCtx       context.Context
	gotID        string
	beforeReturn func()
	path         string
	err          error
}

func (r *recordingExportFunc) call(ctx context.Context, sessionID string) (string, error) {
	r.calls++
	r.gotCtx = ctx
	r.gotID = sessionID
	if r.beforeReturn != nil {
		r.beforeReturn()
	}
	return r.path, r.err
}

func newSaveTestModel(t *testing.T, history []llm.Message, frameSessionID string, fn SessionExportFunc, sub Submitter) Model {
	t.Helper()
	frames := make(chan kernel.RenderFrame, 1)
	if sub == nil {
		sub = func(string) {}
	}
	m := NewModelWithOptions(frames, sub, func() {}, Options{
		MouseTracking: true,
		SessionExport: fn,
	})
	m.frame.History = history
	m.frame.SessionID = frameSessionID
	return m
}

func TestSlashSave_NoConversationReturnsStatus(t *testing.T) {
	rec := &recordingExportFunc{path: "/should/not/be/used.md"}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, nil, "sess-1", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true (slash MUST be consumed even with empty history)")
	}
	if res.StatusMessage != "save: no conversation" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "save: no conversation")
	}
	if rec.calls != 0 {
		t.Fatalf("SessionExportFunc called %d times, want 0 (must short-circuit before export)", rec.calls)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0 (slash must never reach kernel)", sub.calls)
	}
}

func TestSlashSave_NoActiveSessionReturnsStatus(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}
	rec := &recordingExportFunc{path: "/should/not/be/used.md"}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "save: no active session" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "save: no active session")
	}
	if rec.calls != 0 {
		t.Fatalf("SessionExportFunc called %d times, want 0", rec.calls)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
}

func TestSlashSave_HappyPathReturnsWrittenPath(t *testing.T) {
	const wantPath = "/tmp/sess-export-fixture.md"
	history := []llm.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "ack"},
	}
	rec := &recordingExportFunc{path: wantPath}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "sess-parent", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if rec.calls != 1 {
		t.Fatalf("SessionExportFunc called %d times, want exactly 1", rec.calls)
	}
	if rec.gotID != "sess-parent" {
		t.Fatalf("SessionExportFunc got sessionID = %q, want sess-parent (m.frame.SessionID)", rec.gotID)
	}
	if !strings.Contains(res.StatusMessage, wantPath) {
		t.Fatalf("StatusMessage = %q, want it to contain %q", res.StatusMessage, wantPath)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0 (export must not go through kernel.Submit)", sub.calls)
	}
}

func TestSlashSave_ExportFailureRemovesPartialFile(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}

	tmp := t.TempDir()
	partialPath := filepath.Join(tmp, "partial.md")
	exportErr := errors.New("disk full")

	rec := &recordingExportFunc{
		path: partialPath,
		err:  exportErr,
		beforeReturn: func() {
			if err := os.WriteFile(partialPath, []byte("partial"), 0o644); err != nil {
				t.Fatalf("seed partial file: %v", err)
			}
		},
	}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "sess-1", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if rec.calls != 1 {
		t.Fatalf("SessionExportFunc called %d times, want 1", rec.calls)
	}
	if _, err := os.Stat(partialPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("partial file still present at %s (stat err=%v); handler must os.Remove on export failure", partialPath, err)
	}
	if !strings.HasPrefix(res.StatusMessage, "save: write failed:") {
		t.Fatalf("StatusMessage = %q, want prefix %q", res.StatusMessage, "save: write failed:")
	}
	if !strings.Contains(res.StatusMessage, exportErr.Error()) {
		t.Fatalf("StatusMessage = %q, want it to surface underlying error %q", res.StatusMessage, exportErr.Error())
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
}

func TestSlashSave_ErrSessionNotFoundReturnsStoreUnavailable(t *testing.T) {
	history := []llm.Message{{Role: "user", Content: "hi"}}
	rec := &recordingExportFunc{err: transcript.ErrSessionNotFound}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "sess-missing", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "save: store unavailable" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "save: store unavailable")
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
}

// ---- slash_sessions_test.go ----

func TestSessionsSlashOpensPickerPageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var limits []int
	m := newSessionsSlashModel(sub, func(limit int) ([]SessionDirectoryEntry, error) {
		limits = append(limits, limit)
		return []SessionDirectoryEntry{
			{ID: "sess-new", Title: "New Work", Preview: "ask gormes to continue", Source: "cli", LastActiveAt: 200, MessageCount: 3},
			{ID: "sess-old", Title: "", Preview: "older preview", Source: "telegram", LastActiveAt: 100, MessageCount: 1},
		}, nil
	})

	m = enterSlashDispatchBehavior(t, m, "/resume")
	if sub.calls != 0 {
		t.Fatalf("/resume reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /resume = %q, want cleared", got)
	}
	if len(limits) != 1 || limits[0] != 20 {
		t.Fatalf("/resume limits = %v, want default 20", limits)
	}
	if !strings.Contains(m.statusMessage, "sessions opened") {
		t.Fatalf("/resume status = %q, want sessions opened", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/resume fell through to fallback: %q", m.statusMessage)
	}
	assertSessionsPageContains(t, m, "New Work", "ask gormes to continue", "sess-new", "3 messages", "older preview", "telegram")

	m = enterSlashDispatchBehavior(t, m, "/sessions 1")
	if len(limits) != 2 || limits[1] != 1 {
		t.Fatalf("/sessions 1 limits = %v, want second call limit 1", limits)
	}
	assertSessionsPageContains(t, m, "New Work", "sess-new")
}

func TestResumeSlashWithSessionIDSwitchesVisibleSessionAndHistoryWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var requested string
	m := newSessionsSlashModelWithOptions(sub, nil, func(ctx context.Context, query string) (SessionResumeResult, error) {
		if ctx == nil {
			t.Fatal("resume context is nil")
		}
		requested = query
		return SessionResumeResult{
			SessionID: "sess-target",
			History: []llm.Message{
				{Role: "user", Content: "previous question"},
				{Role: "assistant", Content: "previous answer"},
			},
		}, nil
	})
	m.transientPage = &TransientPageState{Title: "Sessions", Body: "old picker"}

	m = enterSlashDispatchBehavior(t, m, "/resume sess-tar")
	if sub.calls != 0 {
		t.Fatalf("/resume <id> reached Submitter %d time(s), want 0", sub.calls)
	}
	if requested != "sess-tar" {
		t.Fatalf("resume query = %q, want sess-tar", requested)
	}
	if got := m.SessionID(); got != "sess-target" {
		t.Fatalf("SessionID() = %q, want sess-target", got)
	}
	if got := m.frame.SessionID; got != "sess-target" {
		t.Fatalf("frame.SessionID = %q, want sess-target", got)
	}
	if len(m.frame.History) != 2 || m.frame.History[0].Content != "previous question" || m.frame.History[1].Content != "previous answer" {
		t.Fatalf("frame.History = %+v, want replayed transcript", m.frame.History)
	}
	if m.transientPage != nil {
		t.Fatalf("transientPage after resume = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "resumed sess-target (2 messages)") {
		t.Fatalf("resume status = %q, want resumed sess-target", m.statusMessage)
	}
}

func TestResumeSlashUnavailableAndBusyStates(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSessionsSlashModelWithOptions(sub, nil, nil)
	m = enterSlashDispatchBehavior(t, m, "/resume sess-missing")
	if sub.calls != 0 {
		t.Fatalf("/resume without adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "resume: session switch unavailable") {
		t.Fatalf("/resume without adapter status = %q, want unavailable", m.statusMessage)
	}

	m = newSessionsSlashModelWithOptions(sub, nil, func(context.Context, string) (SessionResumeResult, error) {
		t.Fatal("resume adapter should not run while a turn is active")
		return SessionResumeResult{}, nil
	})
	m.frame.Phase = kernel.PhaseStreaming
	m = enterSlashDispatchBehavior(t, m, "/resume sess-busy")
	if !strings.Contains(m.statusMessage, "interrupt the current turn before trying to switch sessions") {
		t.Fatalf("/resume busy status = %q, want interrupt guidance", m.statusMessage)
	}
}

func TestSessionsSlashUnavailableAndEmptyStates(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSessionsSlashModel(sub, nil)
	m = enterSlashDispatchBehavior(t, m, "/sessions")
	if sub.calls != 0 {
		t.Fatalf("/sessions without adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "sessions: directory unavailable") {
		t.Fatalf("/sessions without adapter status = %q, want directory unavailable", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("/sessions without adapter opened page %+v, want nil", *m.transientPage)
	}

	m = newSessionsSlashModel(sub, func(limit int) ([]SessionDirectoryEntry, error) { return nil, nil })
	m = enterSlashDispatchBehavior(t, m, "/sessions")
	if !strings.Contains(m.statusMessage, "no sessions found") {
		t.Fatalf("/sessions empty status = %q, want no sessions found", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("/sessions empty opened page %+v, want nil", *m.transientPage)
	}
}

func TestSessionsSlashCompletionsMarkResumeAvailable(t *testing.T) {
	completions := HermesSlashCommandCompletions("/res")
	for _, completion := range completions {
		if completion.Name != "resume" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available after native picker port", completion)
		}
		return
	}
	t.Fatalf("HermesSlashCommandCompletions(/res) = %+v, want resume", completions)
}

func assertSessionsPageContains(t *testing.T, m Model, want ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("sessions slash did not open transient page")
	}
	if m.transientPage.Title != "Sessions" {
		t.Fatalf("page title = %q, want Sessions", m.transientPage.Title)
	}
	for _, item := range want {
		if !strings.Contains(m.transientPage.Body, item) {
			t.Fatalf("page body missing %q:\n%s", item, m.transientPage.Body)
		}
	}
}

func newSessionsSlashModel(sub *nopSubmitter, directory SessionDirectoryFunc) Model {
	return newSessionsSlashModelWithOptions(sub, directory, nil)
}

func newSessionsSlashModelWithOptions(sub *nopSubmitter, directory SessionDirectoryFunc, resume SessionResumeFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-current"}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionDirectory: directory, SessionResume: resume})
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-current"}
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_skin_test.go ----

type recordingSkinConfig struct {
	calls  int
	gotReq SkinConfigRequest
	result SkinConfigResult
	err    error
}

func (r *recordingSkinConfig) call(req SkinConfigRequest) (SkinConfigResult, error) {
	r.calls++
	r.gotReq = req
	if r.err != nil {
		return SkinConfigResult{}, r.err
	}
	return r.result, nil
}

func TestSkinSlashGetSetAdapter(t *testing.T) {
	rec := &recordingSkinConfig{result: SkinConfigResult{Name: "default"}}
	sub := &nopSubmitter{}
	m := newSkinSlashModel(sub, rec.call, "default")
	m.frame.SessionID = "sess-skin"

	m = enterSlashDispatchBehavior(t, m, "/skin")

	if sub.calls != 0 {
		t.Fatalf("/skin reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.calls != 1 {
		t.Fatalf("SkinConfig calls = %d, want 1", rec.calls)
	}
	wantReq := SkinConfigRequest{SessionID: "sess-skin"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("SkinConfig request = %#v, want %#v", rec.gotReq, wantReq)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /skin = %q, want cleared", got)
	}
	assertSkinPageContains(t, m, "skin: default")

	rec.result = SkinConfigResult{Name: "ares"}
	m = enterSlashDispatchBehavior(t, m, "/skin ares")

	wantReq = SkinConfigRequest{Name: "ares", SessionID: "sess-skin"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("SkinConfig request after set = %#v, want %#v", rec.gotReq, wantReq)
	}
	assertSkinPageContains(t, m, "skin → ares")
	if m.activeSkinName != "ares" {
		t.Fatalf("activeSkinName = %q, want ares", m.activeSkinName)
	}
	if !strings.Contains(m.editor.Prompt, "⚔") {
		t.Fatalf("editor prompt after accepted ares skin = %q, want ares prompt glyph", m.editor.Prompt)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/skin fell through to fallback: %q", m.statusMessage)
	}
}

func TestSkinSlashRejectedAndUnavailableDoNotMutate(t *testing.T) {
	rec := &recordingSkinConfig{err: errors.New("invalid skin zeus")}
	sub := &nopSubmitter{}
	m := newSkinSlashModel(sub, rec.call, "ares")
	beforePrompt := m.editor.Prompt

	m = enterSlashDispatchBehavior(t, m, "/skin zeus")

	if sub.calls != 0 {
		t.Fatalf("/skin zeus reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.gotReq.Name != "zeus" {
		t.Fatalf("SkinConfig name = %q, want zeus", rec.gotReq.Name)
	}
	if m.activeSkinName != "ares" {
		t.Fatalf("rejected skin mutated activeSkinName = %q, want ares", m.activeSkinName)
	}
	if m.editor.Prompt != beforePrompt {
		t.Fatalf("rejected skin mutated prompt = %q, want %q", m.editor.Prompt, beforePrompt)
	}
	if !strings.Contains(m.statusMessage, "skin: invalid skin zeus") {
		t.Fatalf("status after rejected skin = %q, want invalid evidence", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, newSkinSlashModel(sub, nil, "ares"), "/skin mono")
	if m.activeSkinName != "ares" {
		t.Fatalf("nil adapter mutated activeSkinName = %q, want ares", m.activeSkinName)
	}
	if !strings.Contains(m.statusMessage, "skin: configuration unavailable") {
		t.Fatalf("status after nil skin adapter = %q, want unavailable evidence", m.statusMessage)
	}
}

func assertSkinPageContains(t *testing.T, m Model, wants ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("skin page = nil, want rendered skin evidence")
	}
	if m.transientPage.Title != "Skin" {
		t.Fatalf("skin page title = %q, want Skin", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("skin page body missing %q:\n%s", want, body)
		}
	}
}

func newSkinSlashModel(sub *nopSubmitter, fn SkinConfigFunc, skinName string) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SkinName: skinName, SkinConfig: fn})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

// ---- slash_statusbar_test.go ----

func TestStatusBarSlashTogglesChromePlacementWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newStatusBarSlashModel(sub)

	initial := m.View()
	assertStatusBarBeforePrompt(t, initial)

	m = enterSlashDispatchBehavior(t, m, "/statusbar off")
	if sub.calls != 0 {
		t.Fatalf("/statusbar reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /statusbar = %q, want cleared", got)
	}
	if m.statusBarMode != StatusBarModeOff {
		t.Fatalf("statusBarMode after /statusbar off = %q, want %q", m.statusBarMode, StatusBarModeOff)
	}
	if !strings.Contains(m.statusMessage, "status bar off") {
		t.Fatalf("status after /statusbar off = %q, want off evidence", m.statusMessage)
	}
	if got := m.View(); strings.Contains(got, "─ ready │") {
		t.Fatalf("/statusbar off still rendered the status rule:\n%s", got)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/statusbar fell through to fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/statusbar on")
	if m.statusBarMode != StatusBarModeTop {
		t.Fatalf("statusBarMode after /statusbar on = %q, want %q", m.statusBarMode, StatusBarModeTop)
	}
	assertStatusBarBeforePrompt(t, m.View())

	m = enterSlashDispatchBehavior(t, m, "/statusbar bottom")
	if m.statusBarMode != StatusBarModeBottom {
		t.Fatalf("statusBarMode after /statusbar bottom = %q, want %q", m.statusBarMode, StatusBarModeBottom)
	}
	assertStatusBarAfterPrompt(t, m.View())

	m = enterSlashDispatchBehavior(t, m, "/sb top")
	if m.statusBarMode != StatusBarModeTop {
		t.Fatalf("statusBarMode after /sb top = %q, want %q", m.statusBarMode, StatusBarModeTop)
	}
	assertStatusBarBeforePrompt(t, m.View())
}

func TestStatusBarSlashToggleAndUsage(t *testing.T) {
	tests := []struct {
		name     string
		initial  StatusBarMode
		input    string
		wantMode StatusBarMode
		want     string
	}{
		{name: "bare toggles off from top", initial: StatusBarModeTop, input: "/statusbar", wantMode: StatusBarModeOff, want: "status bar off"},
		{name: "toggle restores top from off", initial: StatusBarModeOff, input: "/statusbar toggle", wantMode: StatusBarModeTop, want: "status bar top"},
		{name: "bottom accepted", initial: StatusBarModeTop, input: "/statusbar bottom", wantMode: StatusBarModeBottom, want: "status bar bottom"},
		{name: "invalid usage", initial: StatusBarModeBottom, input: "/statusbar sideways", wantMode: StatusBarModeBottom, want: "usage: /statusbar [on|off|top|bottom|toggle]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newStatusBarSlashModel(&nopSubmitter{})
			m.statusBarMode = tt.initial
			m = enterSlashDispatchBehavior(t, m, tt.input)
			if m.statusBarMode != tt.wantMode {
				t.Fatalf("statusBarMode after %s = %q, want %q", tt.input, m.statusBarMode, tt.wantMode)
			}
			if !strings.Contains(m.statusMessage, tt.want) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.want)
			}
		})
	}
}

func TestStatusBarSlashCompletionsAndBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/statusb")
	for _, completion := range completions {
		if completion.Name != "statusbar" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/statusb) = %+v, want statusbar", completions)

foundCompletion:
	got := completionNames(HermesSlashSubcommandCompletions("/statusbar "))
	want := []string{"on", "off", "top", "bottom", "toggle"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("HermesSlashSubcommandCompletions(/statusbar ) = %v, want %v", got, want)
	}

	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "statusbar" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want statusbar", busy)
}

func newStatusBarSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-statusbar",
		History: []llm.Message{
			{Role: "user", Content: "show chrome"},
			{Role: "assistant", Content: "chrome visible"},
		},
	}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

func assertStatusBarBeforePrompt(t *testing.T, view string) {
	t.Helper()
	statusIdx := strings.Index(view, "─ ready │")
	promptIdx := strings.LastIndex(view, "❯")
	if statusIdx < 0 || promptIdx < 0 || statusIdx >= promptIdx {
		t.Fatalf("want status bar before prompt, statusIdx=%d promptIdx=%d:\n%s", statusIdx, promptIdx, view)
	}
}

func assertStatusBarAfterPrompt(t *testing.T, view string) {
	t.Helper()
	statusIdx := strings.LastIndex(view, "─ ready │")
	promptIdx := strings.LastIndex(view, "❯")
	if statusIdx < 0 || promptIdx < 0 || promptIdx >= statusIdx {
		t.Fatalf("want status bar after prompt, statusIdx=%d promptIdx=%d:\n%s", statusIdx, promptIdx, view)
	}
}

// ---- slash_status_test.go ----

func TestStatusSlashRendersCurrentFramePageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newStatusSlashModel(sub)
	m.frame = kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		SessionID: "sess-status-123456",
		Model:     "openai/gpt-5.3-codex",
		ProviderStatus: llm.ProviderStatus{
			Provider: "openai-codex",
			Runtime:  "responses",
		},
		ReasoningEffort: llm.ReasoningEffortEvidence{
			Requested: "high",
			Forwarded: true,
		},
		Telemetry: telemetry.Snapshot{
			TokensInTotal:  1200,
			TokensOutTotal: 34,
			LatencyMsLast:  987,
			TokensPerSec:   12.5,
		},
		ContextStatus: &llm.ContextStatus{
			Engine:          "native-context",
			ContextLength:   200000,
			LastTotalTokens: 1234,
			UsagePercent:    0.617,
			Budget: llm.ContextBudgetStatus{
				State:           "ok",
				RemainingTokens: 198766,
			},
		},
		LastError: "recoverable provider warning",
		History: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	m = enterSlashDispatchBehavior(t, m, "/status")

	if sub.calls != 0 {
		t.Fatalf("/status reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /status = %q, want cleared", got)
	}
	if m.transientPage == nil {
		t.Fatal("/status did not open a transient page")
	}
	if m.transientPage.Title != "Status" {
		t.Fatalf("page title = %q, want Status", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range []string{
		"Gormes TUI Status",
		"Session: sess-status-123456",
		"Phase: Streaming",
		"Model: openai/gpt-5.3-codex",
		"Provider: openai-codex",
		"Runtime: responses",
		"Reasoning effort: high (forwarded)",
		"Context: 1234 / 200000 tokens",
		"Budget: ok, 198766 tokens remaining",
		"Telemetry: 1200 in / 34 out / 987 ms / 12.5 tok/s",
		"History messages: 2",
		"Last error: recoverable provider warning",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("status page body missing %q:\n%s", want, body)
		}
	}
	view := m.View()
	if !strings.Contains(view, "Status") || !strings.Contains(view, "Gormes TUI Status") {
		t.Fatalf("View() did not render transient status page:\n%s", view)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/status fell through to fallback: %q", m.statusMessage)
	}
}

func TestStatusSlashNoActiveSessionAndBusyAvailability(t *testing.T) {
	sub := &nopSubmitter{}
	m := newStatusSlashModel(sub)
	m.frame.SessionID = ""

	m = enterSlashDispatchBehavior(t, m, "/status")
	if sub.calls != 0 {
		t.Fatalf("/status without session reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("/status without session page = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "no active session") {
		t.Fatalf("/status without session status = %q, want no active session", m.statusMessage)
	}

	completions := HermesSlashCommandCompletions("/stat")
	for _, completion := range completions {
		if completion.Name != "status" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/stat) = %+v, want status", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "status" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want status", busy)
}

func newStatusSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-status"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_title_test.go ----

func TestTitleSlashGetsAndSetsSessionTitleWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var calls []titleSlashCall
	m := newTitleSlashModel(sub, func(sessionID, title string) (SessionTitleResult, error) {
		calls = append(calls, titleSlashCall{sessionID: sessionID, title: title})
		if title == "" {
			return SessionTitleResult{Title: "demo title"}, nil
		}
		return SessionTitleResult{Title: title}, nil
	})

	m = enterSlashDispatchBehavior(t, m, "/title")
	if sub.calls != 0 {
		t.Fatalf("/title reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /title = %q, want cleared", got)
	}
	if len(calls) != 1 || calls[0] != (titleSlashCall{sessionID: "sess-title"}) {
		t.Fatalf("/title calls = %+v, want get current title for sess-title", calls)
	}
	if !strings.Contains(m.statusMessage, "title: demo title") {
		t.Fatalf("/title status = %q, want current title", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/title fell through to fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/title my   title")
	if sub.calls != 0 {
		t.Fatalf("/title set reached Submitter %d time(s), want 0", sub.calls)
	}
	if len(calls) != 2 || calls[1] != (titleSlashCall{sessionID: "sess-title", title: "my title"}) {
		t.Fatalf("/title set calls = %+v, want collapsed multi-word title", calls)
	}
	if !strings.Contains(m.statusMessage, "session title set: my title") {
		t.Fatalf("/title set status = %q, want set confirmation", m.statusMessage)
	}
}

func TestTitleSlashRequiresActiveSessionAndAdapter(t *testing.T) {
	sub := &nopSubmitter{}
	called := false
	m := newTitleSlashModel(sub, func(sessionID, title string) (SessionTitleResult, error) {
		called = true
		return SessionTitleResult{Title: title}, nil
	})
	m.frame.SessionID = ""

	m = enterSlashDispatchBehavior(t, m, "/title new name")
	if sub.calls != 0 {
		t.Fatalf("/title without session reached Submitter %d time(s), want 0", sub.calls)
	}
	if called {
		t.Fatal("/title without an active session called SessionTitle adapter")
	}
	if !strings.Contains(m.statusMessage, "no active session") {
		t.Fatalf("/title without session status = %q, want no active session", m.statusMessage)
	}

	m = newTitleSlashModel(sub, nil)
	m = enterSlashDispatchBehavior(t, m, "/title new name")
	if !strings.Contains(m.statusMessage, "title: session title unavailable") {
		t.Fatalf("/title without adapter status = %q, want unavailable evidence", m.statusMessage)
	}
}

func TestTitleSlashBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/tit")
	for _, completion := range completions {
		if completion.Name != "title" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/tit) = %+v, want title", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "title" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want title", busy)
}

type titleSlashCall struct {
	sessionID string
	title     string
}

func newTitleSlashModel(sub *nopSubmitter, title SessionTitleFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-title"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionTitle: title})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

// ---- slash_tools_test.go ----

type recordingToolsConfigure struct {
	calls  int
	gotReq ToolsConfigureRequest
	result ToolsConfigureResult
	err    error
}

func (r *recordingToolsConfigure) call(req ToolsConfigureRequest) (ToolsConfigureResult, error) {
	r.calls++
	r.gotReq = req
	if r.err != nil {
		return ToolsConfigureResult{}, r.err
	}
	return r.result, nil
}

func TestToolsSlashEnableDisableAdapter(t *testing.T) {
	rec := &recordingToolsConfigure{result: ToolsConfigureResult{
		Changed:        []string{"web", "terminal"},
		Unknown:        []string{"unknown_toolset"},
		MissingServers: []string{"github"},
		Reset:          true,
	}}
	sub := &nopSubmitter{}
	m := newToolsSlashModel(sub, rec.call)
	m.frame.SessionID = "sess-tools"

	m = enterSlashDispatchBehavior(t, m, "/tools enable web terminal")

	if sub.calls != 0 {
		t.Fatalf("/tools enable reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.calls != 1 {
		t.Fatalf("ToolsConfigure calls = %d, want 1", rec.calls)
	}
	wantReq := ToolsConfigureRequest{Action: "enable", Names: []string{"web", "terminal"}, SessionID: "sess-tools"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("ToolsConfigure request = %#v, want %#v", rec.gotReq, wantReq)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /tools enable = %q, want cleared", got)
	}
	assertToolsPageContains(t, m, []string{
		"enabled: web, terminal",
		"unknown toolsets: unknown_toolset",
		"missing MCP servers: github",
		"session reset. new tool configuration is active.",
	})
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/tools enable fell through to fallback: %q", m.statusMessage)
	}
}

func TestToolsSlashDisableRendersDisabledEvidence(t *testing.T) {
	rec := &recordingToolsConfigure{result: ToolsConfigureResult{Changed: []string{"web"}, Reset: true}}
	m := newToolsSlashModel(&nopSubmitter{}, rec.call)

	m = enterSlashDispatchBehavior(t, m, "/tools disable web")

	if rec.gotReq.Action != "disable" {
		t.Fatalf("ToolsConfigure action = %q, want disable", rec.gotReq.Action)
	}
	assertToolsPageContains(t, m, []string{"disabled: web", "session reset. new tool configuration is active."})
}

func TestToolsSlashUsageAndUnavailableEvidence(t *testing.T) {
	for _, input := range []string{"/tools enable", "/tools disable"} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newToolsSlashModel(sub, nil), input)
			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			for _, want := range []string{
				"usage: " + input + " <name> [name ...]",
				"built-in toolset: " + input + " web",
				"MCP tool: " + input + " github:create_issue",
			} {
				if !strings.Contains(m.statusMessage, want) {
					t.Fatalf("status after %s missing %q:\n%s", input, want, m.statusMessage)
				}
			}
		})
	}

	sub := &nopSubmitter{}
	m := enterSlashDispatchBehavior(t, newToolsSlashModel(sub, nil), "/tools enable web")
	if sub.calls != 0 {
		t.Fatalf("/tools enable with nil adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "tools: configuration unavailable") {
		t.Fatalf("status after nil adapter = %q, want unavailable evidence", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("nil adapter tools page = %+v, want none", *m.transientPage)
	}
}

func assertToolsPageContains(t *testing.T, m Model, wants []string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("tools page = nil, want rendered tools evidence")
	}
	if m.transientPage.Title != "Tools" {
		t.Fatalf("tools page title = %q, want Tools", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("tools page body missing %q:\n%s", want, body)
		}
	}
}

func newToolsSlashModel(sub *nopSubmitter, fn ToolsConfigureFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, ToolsConfigure: fn})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

// ---- slash_usage_test.go ----

func TestUsageSlashOpensFrameUsagePageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newUsageSlashModel(sub, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: "sess-usage",
		Model:     "gpt-usage",
		Telemetry: telemetry.Snapshot{
			TokensInTotal:  1234,
			TokensOutTotal: 567,
			LatencyMsLast:  890,
			TokensPerSec:   12.34,
		},
		ContextStatus: &llm.ContextStatus{
			ContextLength:    200000,
			LastTotalTokens:  18000,
			UsagePercent:     9.0,
			CompressionCount: 2,
		},
	})

	m = enterSlashDispatchBehavior(t, m, "/usage")
	if sub.calls != 0 {
		t.Fatalf("/usage reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /usage = %q, want cleared", got)
	}
	if !strings.Contains(m.statusMessage, "usage opened") {
		t.Fatalf("/usage status = %q, want usage opened", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/usage fell through to fallback: %q", m.statusMessage)
	}
	assertUsagePageContains(t, m,
		"Usage source: local TUI frame",
		"Model: gpt-usage",
		"Session: sess-usage",
		"Input tokens: 1234",
		"Output tokens: 567",
		"Total tokens: 1801",
		"Last latency: 890 ms",
		"Speed: 12.34 tokens/sec",
		"Context: 18000 / 200000 tokens (9.0%)",
		"Compressions: 2",
	)
}

func TestUsageSlashFetchesAccountUsageAsynchronously(t *testing.T) {
	sub := &nopSubmitter{}
	used := 40.0
	var requested bool
	m := newUsageSlashModelWithOptions(sub, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: "sess-usage",
		Model:     "gpt-usage",
		Telemetry: telemetry.Snapshot{TokensInTotal: 10, TokensOutTotal: 5},
	}, Options{
		MouseTracking: true,
		AccountUsage: func(ctx context.Context) (llm.AccountUsageSnapshot, error) {
			if ctx == nil {
				t.Fatal("AccountUsage context is nil")
			}
			requested = true
			return llm.AccountUsageSnapshot{
				Provider:  "openrouter",
				Plan:      "Team",
				FetchedAt: time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC),
				Windows: []llm.AccountUsageWindow{{
					Label:       "Session",
					UsedPercent: &used,
				}},
				Details: []string{"Credits balance: $12.50"},
			}, nil
		},
	})

	m.editor.SetValue("/usage")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = requireUsageSlashModel(t, next)
	if sub.calls != 0 {
		t.Fatalf("/usage reached Submitter %d time(s), want 0", sub.calls)
	}
	if requested {
		t.Fatal("AccountUsage was called before the returned tea.Cmd ran")
	}
	assertUsagePageContains(t, m, "Provider account usage: loading...")

	msg := firstNonNilCmdMsg(t, cmd)
	if msg == nil {
		t.Fatal("/usage account command returned nil message, want usage account update")
	}
	next, _ = m.Update(msg)
	m = requireUsageSlashModel(t, next)
	if !requested {
		t.Fatal("AccountUsage was not called by returned tea.Cmd")
	}
	if !strings.Contains(m.statusMessage, "usage account updated") {
		t.Fatalf("status after account update = %q, want usage account updated", m.statusMessage)
	}
	assertUsagePageContains(t, m,
		"Provider: openrouter (Team)",
		"Session: 60% remaining (40% used)",
		"Credits balance: $12.50",
	)
}

func TestUsageSlashNoCallsConsumesWithoutPage(t *testing.T) {
	sub := &nopSubmitter{}
	m := newUsageSlashModel(sub, kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-empty", Model: "gpt-empty"})

	m = enterSlashDispatchBehavior(t, m, "/usage")
	if sub.calls != 0 {
		t.Fatalf("/usage with no counters reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "no API calls yet") {
		t.Fatalf("/usage no counters status = %q, want no API calls yet", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("/usage no counters opened page %+v, want nil", *m.transientPage)
	}
}

func TestUsageSlashCompletionsAndBusyAvailability(t *testing.T) {
	for _, completion := range HermesSlashCommandCompletions("/usa") {
		if completion.Name != "usage" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/usa) missing usage")

foundCompletion:
	for _, name := range NewDefaultSlashRegistry().BusyAvailableSlashes() {
		if name == "usage" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() missing usage")
}

func assertUsagePageContains(t *testing.T, m Model, want ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("/usage did not open a transient page")
	}
	if m.transientPage.Title != "Usage" {
		t.Fatalf("page title = %q, want Usage", m.transientPage.Title)
	}
	for _, item := range want {
		if !strings.Contains(m.transientPage.Body, item) {
			t.Fatalf("usage page missing %q:\n%s", item, m.transientPage.Body)
		}
	}
}

func newUsageSlashModel(sub *nopSubmitter, frame kernel.RenderFrame) Model {
	return newUsageSlashModelWithOptions(sub, frame, Options{MouseTracking: true})
}

func newUsageSlashModelWithOptions(sub *nopSubmitter, frame kernel.RenderFrame, opts Options) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- frame
	if !opts.MouseTracking {
		opts.MouseTracking = true
	}
	m := NewModelWithOptions(frames, sub.submit, func() {}, opts)
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

func requireUsageSlashModel(t *testing.T, model tea.Model) Model {
	t.Helper()
	m, ok := model.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", model)
	}
	return m
}

func firstNonNilCmdMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, nested := range batch {
			if got := firstNonNilCmdMsg(t, nested); got != nil {
				return got
			}
		}
		return nil
	}
	return msg
}

// ---- slash_voice_test.go ----

type recordingVoiceToggle struct {
	calls  int
	gotReq VoiceToggleRequest
	result VoiceToggleResult
	err    error
}

func (r *recordingVoiceToggle) call(req VoiceToggleRequest) (VoiceToggleResult, error) {
	r.calls++
	r.gotReq = req
	if r.err != nil {
		return VoiceToggleResult{}, r.err
	}
	return r.result, nil
}

func TestVoiceSlashStatusUpdatesRecordKey(t *testing.T) {
	rec := &recordingVoiceToggle{result: VoiceToggleResult{
		Enabled:   true,
		TTS:       false,
		RecordKey: "ctrl+space",
		Details:   "Audio: OK\nSTT: not configured\nTTS: not configured",
	}}
	sub := &nopSubmitter{}
	m := newVoiceSlashModel(sub, rec.call, "ctrl+b")
	m.frame.SessionID = "sess-voice"

	m = enterSlashDispatchBehavior(t, m, "/voice status")

	if sub.calls != 0 {
		t.Fatalf("/voice status reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.calls != 1 {
		t.Fatalf("VoiceToggle calls = %d, want 1", rec.calls)
	}
	wantReq := VoiceToggleRequest{Action: "status", SessionID: "sess-voice"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("VoiceToggle request = %#v, want %#v", rec.gotReq, wantReq)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /voice status = %q, want cleared", got)
	}
	assertVoicePageContains(t, m,
		"Voice Mode Status",
		"Mode:       ON",
		"TTS:        OFF",
		"Record key: Ctrl+Space",
		"Requirements:",
		"Audio: OK",
		"STT: not configured",
		"TTS: not configured",
	)
	if m.voiceRecordKey != "ctrl+space" {
		t.Fatalf("model voiceRecordKey = %q, want ctrl+space", m.voiceRecordKey)
	}
	if decision := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeySpace, Ctrl: true}, HermesInputState{VoiceRecordKey: m.voiceRecordKey}); decision.Action != HermesActionToggleVoiceRecording {
		t.Fatalf("Ctrl+Space after /voice status = %v, want voice toggle", decision.Action)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/voice status fell through to fallback: %q", m.statusMessage)
	}
}

func TestVoiceSlashToggleAndTTSDoNotClobberMissingRecordKey(t *testing.T) {
	rec := &recordingVoiceToggle{result: VoiceToggleResult{Enabled: true, TTS: false, RecordKey: "alt+r"}}
	m := newVoiceSlashModel(&nopSubmitter{}, rec.call, "ctrl+b")

	m = enterSlashDispatchBehavior(t, m, "/voice on")

	if rec.gotReq.Action != "on" {
		t.Fatalf("VoiceToggle action = %q, want on", rec.gotReq.Action)
	}
	if m.voiceRecordKey != "alt+r" {
		t.Fatalf("model voiceRecordKey after /voice on = %q, want alt+r", m.voiceRecordKey)
	}
	assertVoicePageContains(t, m,
		"Voice mode enabled",
		"Alt+R to start/stop recording",
		"/voice tts  to toggle speech output",
		"/voice off  to disable voice mode",
	)

	rec.result = VoiceToggleResult{Enabled: true, TTS: true}
	m = enterSlashDispatchBehavior(t, m, "/voice tts")
	if rec.gotReq.Action != "tts" {
		t.Fatalf("VoiceToggle action = %q, want tts", rec.gotReq.Action)
	}
	if m.voiceRecordKey != "alt+r" {
		t.Fatalf("model voiceRecordKey after missing record_key = %q, want cached alt+r", m.voiceRecordKey)
	}
	assertVoicePageContains(t, m, "Voice TTS enabled.")

	rec.result = VoiceToggleResult{Enabled: false, TTS: false}
	m = enterSlashDispatchBehavior(t, m, "/voice off")
	assertVoicePageContains(t, m, "Voice mode disabled.")
}

func TestVoiceSlashUnavailableAdapterConsumesWithRequirements(t *testing.T) {
	sub := &nopSubmitter{}
	m := newVoiceSlashModel(sub, nil, "ctrl+o")

	m = enterSlashDispatchBehavior(t, m, "/voice status")

	if sub.calls != 0 {
		t.Fatalf("/voice status with nil adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.voiceRecordKey != "ctrl+o" {
		t.Fatalf("nil adapter clobbered voiceRecordKey = %q, want ctrl+o", m.voiceRecordKey)
	}
	assertVoicePageContains(t, m,
		"Voice Mode Status",
		"Mode:       OFF",
		"TTS:        OFF",
		"Record key: Ctrl+B",
		"Requirements:",
		"voice adapter unavailable",
	)
}

func assertVoicePageContains(t *testing.T, m Model, wants ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("voice page = nil, want rendered voice evidence")
	}
	if m.transientPage.Title != "Voice" {
		t.Fatalf("voice page title = %q, want Voice", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("voice page body missing %q:\n%s", want, body)
		}
	}
}

func newVoiceSlashModel(sub *nopSubmitter, fn VoiceToggleFunc, recordKey string) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, VoiceRecordKey: recordKey, VoiceToggle: fn})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

// ---- queue_slash_test.go ----

func TestQueueSlashAppendsVisibleQueueWithoutModelLeak(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSlashDispatchBehaviorModel((*nopSubmitter)(nil))
	m.submit = sub.submit
	m.inFlight = true
	m.frame.Phase = kernel.PhaseStreaming

	m = enterSlashDispatchBehavior(t, m, "/queue follow up after tools")
	if sub.calls != 0 {
		t.Fatalf("/queue reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.queuedMessages.Len(); got != 1 {
		t.Fatalf("queued messages = %d, want 1", got)
	}
	if got := m.queuedMessages.Items()[0]; got != "follow up after tools" {
		t.Fatalf("queued text = %q, want stripped prompt", got)
	}
	if strings.Contains(strings.TrimSpace(m.statusMessage), "recognized") {
		t.Fatalf("/queue fell through to unavailable evidence: %q", m.statusMessage)
	}
	if !strings.Contains(m.statusMessage, "queued") {
		t.Fatalf("status after /queue = %q, want queued evidence", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/queue")
	if !strings.Contains(m.statusMessage, "1 queued message(s)") {
		t.Fatalf("status after /queue status = %q, want queue depth", m.statusMessage)
	}
}

func TestQueueSlashDrainsAfterTurnSettles(t *testing.T) {
	sub := &recordingSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame.Phase = kernel.PhaseStreaming
	m.inFlight = true

	m = enterSlashDispatchBehavior(t, m, "/queue next turn")
	if sub.calls != 0 {
		t.Fatalf("/queue submit calls before drain = %d, want 0", sub.calls)
	}

	next, cmd := m.Update(frameMsg(kernel.RenderFrame{Phase: kernel.PhaseIdle, DraftText: "done"}))
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", next)
	}
	runTestCmd(t, cmd)
	if sub.calls != 1 || sub.texts[0] != "next turn" {
		t.Fatalf("drained submits = %d %#v, want next turn", sub.calls, sub.texts)
	}
	if got := updated.queuedMessages.Len(); got != 0 {
		t.Fatalf("queued messages after drain = %d, want 0", got)
	}
	if !strings.Contains(updated.statusMessage, "queued follow-up sent") {
		t.Fatalf("status after drain = %q, want sent evidence", updated.statusMessage)
	}
}

func TestQueueSlashEmptyReportsDepthEvenIdle(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSlashDispatchBehaviorModel((*nopSubmitter)(nil))
	m.submit = sub.submit
	m.queuedMessages.Enqueue("already queued")

	m = enterSlashDispatchBehavior(t, m, "/queue   ")
	if sub.calls != 0 {
		t.Fatalf("/queue status reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "1 queued message(s)") {
		t.Fatalf("status after /queue = %q, want queue depth", m.statusMessage)
	}
}

// ---- reload_skills_slash_test.go ----

func TestReloadSkillsSlashRefreshesSkillSlashRegistry(t *testing.T) {
	sub := &recordingSubmitter{}
	calls := 0
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillSlashReload: func(context.Context) (SkillSlashReloadResult, error) {
			calls++
			return SkillSlashReloadResult{Commands: []skills.SkillSlashCommand{{
				Command:     "/fresh-skill",
				Name:        "fresh-skill",
				Description: "Fresh skill",
				Skill:       skills.Skill{Name: "fresh-skill", Body: "Fresh skill body."},
			}}, Output: "Skills Reloaded\n1 skill(s) available"}, nil
		},
	})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, "/reload-skills")
	if calls != 1 {
		t.Fatalf("reload calls = %d, want 1", calls)
	}
	if sub.calls != 0 {
		t.Fatalf("/reload-skills reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "1 skill(s) available") {
		t.Fatalf("status after reload = %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/fresh-skill now")
	if sub.calls != 1 {
		t.Fatalf("/fresh-skill submit calls = %d, want 1", sub.calls)
	}
	if !strings.Contains(sub.texts[0], "Fresh skill body.") || strings.Contains(sub.texts[0], "/fresh-skill now") {
		t.Fatalf("fresh skill submit did not expand correctly:\n%s", sub.texts[0])
	}
}

func TestReloadSkillsSlashConsumesUnavailableAndFailure(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSkillSlashModel(sub, nil)

	m = enterSlashDispatchBehavior(t, m, "/reload_skills")
	if sub.calls != 0 {
		t.Fatalf("unwired reload reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "reload-skills") || !strings.Contains(m.statusMessage, "unavailable") {
		t.Fatalf("unwired reload status = %q, want unavailable evidence", m.statusMessage)
	}

	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m = NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillSlashReload: func(context.Context) (SkillSlashReloadResult, error) {
			return SkillSlashReloadResult{}, errors.New("scan failed")
		},
	})
	m.frame.Phase = kernel.PhaseIdle
	m = enterSlashDispatchBehavior(t, m, "/reload-skills")
	if !strings.Contains(m.statusMessage, "scan failed") {
		t.Fatalf("failed reload status = %q, want error evidence", m.statusMessage)
	}
}

// ---- skill_slash_test.go ----

type recordingSubmitter struct {
	calls int
	texts []string
}

func (r *recordingSubmitter) submit(text string) {
	r.calls++
	r.texts = append(r.texts, text)
}

func TestSkillSlashDispatch_SubmitsExpandedSkillMessage(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSkillSlashModel(sub, []skills.SkillSlashCommand{{
		Command:     "/review-skill",
		Name:        "review-skill",
		Description: "Review code",
		SkillDir:    "/tmp/review-skill",
		Skill:       skills.Skill{Name: "review-skill", Body: "Review the requested code carefully."},
	}})

	m = enterSlashDispatchBehavior(t, m, "/review-skill inspect src")

	if sub.calls != 1 {
		t.Fatalf("Submitter calls = %d, want 1", sub.calls)
	}
	got := sub.texts[0]
	for _, want := range []string{
		`[IMPORTANT: The user has invoked the "review-skill" skill`,
		"Review the requested code carefully.",
		"[Skill directory: /tmp/review-skill]",
		"The user has provided the following instruction alongside the skill invocation: inspect src",
		"[Runtime note: native-tui]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("submitted skill message missing %q:\n%s", want, got)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(got), "/review-skill") {
		t.Fatalf("raw slash leaked into submit: %q", got)
	}
	if m.editor.Value() != "" {
		t.Fatalf("editor after skill slash = %q, want cleared", m.editor.Value())
	}
	if !strings.Contains(m.statusMessage, "skill_invoked: review-skill") {
		t.Fatalf("status = %q, want skill invocation evidence", m.statusMessage)
	}
}

func TestSkillSlashDispatch_BuiltinsAndSkillsPrecedePromptTemplates(t *testing.T) {
	sub := &recordingSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillSlashCommands: []skills.SkillSlashCommand{
			{Command: "/help", Name: "help", Skill: skills.Skill{Name: "help", Body: "must not run"}},
			{Command: "/review-skill", Name: "review-skill", Skill: skills.Skill{Name: "review-skill", Body: "Skill body wins."}},
		},
		PromptTemplates: PromptTemplateCatalog{Templates: []prompttemplates.Template{{Name: "review-skill", Body: "Template body must not win."}}},
	})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, "/help")
	if sub.calls != 0 {
		t.Fatalf("/help reached Submitter %d time(s), want builtin precedence", sub.calls)
	}

	m = enterSlashDispatchBehavior(t, m, "/review-skill now")
	if sub.calls != 1 {
		t.Fatalf("/review-skill Submitter calls = %d, want skill precedence over prompt template", sub.calls)
	}
	if !strings.Contains(sub.texts[0], "Skill body wins.") || strings.Contains(sub.texts[0], "Template body must not win") {
		t.Fatalf("skill/template precedence wrong:\n%s", sub.texts[0])
	}
}

func newSkillSlashModel(sub *recordingSubmitter, commands []skills.SkillSlashCommand) Model {
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking:      true,
		SkillSlashCommands: commands,
	})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

// ---- prompt_template_slash_test.go ----

func TestPromptTemplateSlashExpansionSeedsEditorWithoutSubmit(t *testing.T) {
	sub := &recordingPromptTemplateSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: "review", Description: "Review staged changes", ArgumentHint: "<scope>", Body: "Review $1 with args: $ARGUMENTS"},
	}}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{PromptTemplates: catalog})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, `/review staged "bug fix"`)

	if sub.calls != 0 {
		t.Fatalf("/review reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "Review staged with args: staged bug fix" {
		t.Fatalf("editor after template expansion = %q", got)
	}
	if !strings.Contains(m.statusMessage, "prompt_template_expanded") || !strings.Contains(m.statusMessage, "review") {
		t.Fatalf("status after template expansion = %q", m.statusMessage)
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		_ = cmd()
	}
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	if updated.editor.Value() != "" {
		t.Fatalf("editor after submitting expanded template = %q, want cleared", updated.editor.Value())
	}
	if sub.calls != 1 || sub.last != "Review staged with args: staged bug fix" {
		t.Fatalf("submitter calls=%d last=%q, want one expanded prompt", sub.calls, sub.last)
	}
}

type recordingPromptTemplateSubmitter struct {
	calls int
	last  string
}

func (s *recordingPromptTemplateSubmitter) submit(text string) {
	s.calls++
	s.last = text
}

func TestPromptTemplateSlashCompletions(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: "review", Description: "Review staged changes", ArgumentHint: "<scope>"},
	}}
	completions := SlashCompletionsWithPromptTemplates("/rev", catalog)
	if len(completions) != 1 {
		t.Fatalf("SlashCompletionsWithPromptTemplates = %+v, want one template", completions)
	}
	if got := completions[0]; got.Name != "review" || got.ArgumentHint != "<scope>" || got.Description != "Review staged changes" {
		t.Fatalf("template completion = %+v", got)
	}
	menu := renderSlashCompletionMenuWithTemplates("/rev", 80, DefaultHermesSkin(), catalog)
	if !strings.Contains(menu, "/review <scope>") || !strings.Contains(menu, "Review staged changes") {
		t.Fatalf("completion menu missing prompt template hint/description:\n%s", menu)
	}

	// Built-in slash commands keep precedence over prompt-template names.
	colliding := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: "skills", Body: "shadow"}}}
	sub := &nopSubmitter{}
	calls := 0
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		PromptTemplates: colliding,
		SkillsCommand: func(input string) string {
			calls++
			return "skills local command"
		},
	})
	m = enterSlashDispatchBehavior(t, m, "/skills list")
	if calls != 1 {
		t.Fatalf("colliding /skills template shadowed built-in skills command; calls=%d", calls)
	}
	if sub.calls != 0 {
		t.Fatalf("/skills list reached Submitter %d time(s), want 0", sub.calls)
	}
}
