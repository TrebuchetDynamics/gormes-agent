package admin

import (
	"context"
	"fmt"
	"strings"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/navigation"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/wizardflow"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AgentsRegistry interface {
	List(context.Context) ([]goncho.AgentRecord, error)
	Create(context.Context, goncho.CreateAgentOptions) (goncho.AgentRecord, error)
	Bind(context.Context, string, goncho.BindingMatch) error
	Unbind(context.Context, goncho.BindingMatch) error
	Resolve(context.Context, goncho.BindingMatch) (string, bool, error)
}

type AgentsScreen struct {
	registry AgentsRegistry

	records      []goncho.AgentRecord
	selected     int
	message      string
	err          error
	wizard       *agentsWizardState
	lastBinding  *goncho.BindingMatch
	lastAgentID  string
	inspectOpen  bool
	inspectMatch *goncho.BindingMatch
}

type AgentsOption func(*AgentsScreen)

func WithAgentsRegistry(registry AgentsRegistry) AgentsOption {
	return func(s *AgentsScreen) {
		if registry != nil {
			s.registry = registry
		}
	}
}

func NewAgentsScreen(opts ...AgentsOption) *AgentsScreen {
	s := &AgentsScreen{registry: memoryAgentsRegistry{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *AgentsScreen) Title() string { return "Agents" }

func (s *AgentsScreen) Init() tea.Cmd { return s.refreshCmd() }

func (s *AgentsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if s.wizard != nil {
		return s.updateWizard(msg)
	}
	switch msg := msg.(type) {
	case agentsLoadedMsg:
		s.records = append([]goncho.AgentRecord(nil), msg.records...)
		s.err = msg.err
		s.selected = navigation.ClampIndex(s.selected, len(s.records))
	case agentCreatedMsg:
		if msg.err != nil {
			s.message = "spawn failed: " + msg.err.Error()
			return s, nil
		}
		s.message = "spawned " + msg.record.ID
		return s, s.refreshCmd()
	case agentBoundMsg:
		if msg.err != nil {
			s.message = "bind failed: " + msg.err.Error()
			return s, nil
		}
		s.lastAgentID = msg.agentID
		s.lastBinding = &msg.match
		s.message = "bound " + msg.agentID
	case agentUnboundMsg:
		if msg.err != nil {
			s.message = "unbind failed: " + msg.err.Error()
			return s, nil
		}
		s.message = "binding removed"
		s.lastBinding = nil
		s.lastAgentID = ""
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			return s, s.refreshCmd()
		case "up", "k":
			if s.selected > 0 {
				s.selected--
			}
		case "down", "j":
			if s.selected < len(s.records)-1 {
				s.selected++
			}
		case "n":
			s.wizard = newSpawnAgentWizard()
		case "b":
			if len(s.records) == 0 {
				s.message = "spawn an agent before binding"
				return s, nil
			}
			s.wizard = newBindAgentWizard(s.records)
		case "u":
			if s.lastBinding == nil {
				s.message = "no binding selected"
				return s, nil
			}
			match := *s.lastBinding
			return s, s.unbindCmd(match)
		case "i":
			if s.lastBinding == nil {
				s.message = "no binding selected"
				return s, nil
			}
			match := *s.lastBinding
			s.inspectMatch = &match
			s.inspectOpen = true
		case "esc":
			s.inspectOpen = false
		}
	}
	return s, nil
}

func (s *AgentsScreen) View() string {
	if s.wizard != nil {
		return s.wizard.View()
	}
	var b strings.Builder
	b.WriteString("Agents\n")
	if s.message != "" {
		fmt.Fprintf(&b, "Status: %s\n", s.message)
	}
	if s.err != nil {
		fmt.Fprintf(&b, "agents_error: %v\n", s.err)
	}
	if s.inspectOpen && s.inspectMatch != nil {
		fmt.Fprintf(&b, "Inspect binding\nagent: %s\nmatch: %s/%s/%s thread=%s\n\n",
			s.lastAgentID, s.inspectMatch.Channel, s.inspectMatch.PeerKind, s.inspectMatch.PeerID, s.inspectMatch.ThreadID)
	}
	if len(s.records) == 0 {
		b.WriteString("\npress n to spawn your first agent\n")
		return b.String()
	}
	b.WriteByte('\n')
	for i, record := range s.records {
		cursor := " "
		if i == s.selected {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s  %s\n", cursor, record.ID, record.Name)
	}
	if s.lastBinding != nil {
		fmt.Fprintf(&b, "\nlast binding: %s -> %s/%s/%s thread=%s\n",
			s.lastAgentID, s.lastBinding.Channel, s.lastBinding.PeerKind, s.lastBinding.PeerID, s.lastBinding.ThreadID)
	}
	return b.String()
}

func (s *AgentsScreen) ShortHelp() []KeyHelp {
	return []KeyHelp{
		{Keys: []string{"n"}, Description: "spawn agent"},
		{Keys: []string{"b"}, Description: "bind agent"},
		{Keys: []string{"u"}, Description: "unbind last binding"},
		{Keys: []string{"i"}, Description: "inspect binding"},
		{Keys: []string{"r"}, Description: "refresh agents"},
	}
}

func (s *AgentsScreen) refreshCmd() tea.Cmd {
	registry := s.registry
	return func() tea.Msg {
		records, err := registry.List(context.Background())
		return agentsLoadedMsg{records: records, err: err}
	}
}

func (s *AgentsScreen) updateWizard(msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if key.Type == tea.KeyEscape {
		s.wizard = nil
		s.message = "wizard cancelled"
		return s, nil
	}
	done, err := s.wizard.Update(key)
	if err != nil {
		s.message = err.Error()
		s.wizard = nil
		return s, nil
	}
	if !done {
		return s, nil
	}
	w := s.wizard
	s.wizard = nil
	switch w.kind {
	case agentsWizardSpawn:
		opts, err := w.spawnOptions()
		if err != nil {
			s.message = err.Error()
			return s, nil
		}
		return s, s.createCmd(opts)
	case agentsWizardBind:
		agentID, match, err := w.bindOptions()
		if err != nil {
			s.message = err.Error()
			return s, nil
		}
		return s, s.bindCmd(agentID, match)
	default:
		return s, nil
	}
}

func (s *AgentsScreen) createCmd(opts goncho.CreateAgentOptions) tea.Cmd {
	registry := s.registry
	return func() tea.Msg {
		record, err := registry.Create(context.Background(), opts)
		return agentCreatedMsg{record: record, err: err}
	}
}

func (s *AgentsScreen) bindCmd(agentID string, match goncho.BindingMatch) tea.Cmd {
	registry := s.registry
	return func() tea.Msg {
		err := registry.Bind(context.Background(), agentID, match)
		return agentBoundMsg{agentID: agentID, match: match, err: err}
	}
}

func (s *AgentsScreen) unbindCmd(match goncho.BindingMatch) tea.Cmd {
	registry := s.registry
	return func() tea.Msg {
		return agentUnboundMsg{err: registry.Unbind(context.Background(), match)}
	}
}

type agentsLoadedMsg struct {
	records []goncho.AgentRecord
	err     error
}

type agentCreatedMsg struct {
	record goncho.AgentRecord
	err    error
}

type agentBoundMsg struct {
	agentID string
	match   goncho.BindingMatch
	err     error
}

type agentUnboundMsg struct {
	err error
}

type agentsWizardKind string

const (
	agentsWizardSpawn agentsWizardKind = "spawn"
	agentsWizardBind  agentsWizardKind = "bind"
)

type agentsWizardState struct {
	kind       agentsWizardKind
	title      string
	flow       *wizardflow.Flow
	input      textinput.Model
	pickCursor int
	confirmYes bool
}

func newSpawnAgentWizard() *agentsWizardState {
	w := &agentsWizardState{
		kind:  agentsWizardSpawn,
		title: "Spawn agent",
		flow: wizardflow.New([]wizard.Step{
			wizard.Text("name", "Agent name"),
			wizard.MultiLine("persona", "Persona"),
			wizard.Confirm("confirm", "Create agent?"),
		}),
		confirmYes: true,
	}
	w.prepareInput()
	return w
}

func newBindAgentWizard(records []goncho.AgentRecord) *agentsWizardState {
	choices := make([]wizard.Choice, 0, len(records))
	for _, record := range records {
		choices = append(choices, wizard.Choice{ID: record.ID, Label: record.Name + " (" + record.ID + ")"})
	}
	w := &agentsWizardState{
		kind:  agentsWizardBind,
		title: "Bind agent",
		flow: wizardflow.New([]wizard.Step{
			wizard.Pick("agent", "Agent", choices),
			wizard.Pick("channel", "Channel", []wizard.Choice{{ID: "telegram", Label: "Telegram"}, {ID: "discord", Label: "Discord"}, {ID: "slack", Label: "Slack"}}),
			wizard.Pick("peer_kind", "Peer kind", []wizard.Choice{{ID: "user", Label: "User"}, {ID: "group", Label: "Group"}, {ID: "channel", Label: "Channel"}, {ID: "thread", Label: "Thread"}}),
			wizard.Text("peer_id", "Peer ID"),
			wizard.Text("thread_id", "Thread ID (optional)"),
			wizard.Confirm("confirm", "Bind agent?"),
		}),
		confirmYes: true,
	}
	w.prepareInput()
	return w
}

func (w *agentsWizardState) Update(msg tea.KeyMsg) (bool, error) {
	step, ok := w.flow.ActiveStep()
	if !ok {
		return true, nil
	}
	switch step.Kind {
	case wizard.KindPick:
		switch msg.String() {
		case "up", "k":
			w.pickCursor = navigation.MoveIndex(w.pickCursor, len(step.Choices), -1)
		case "down", "j":
			w.pickCursor = navigation.MoveIndex(w.pickCursor, len(step.Choices), 1)
		case "enter":
			if len(step.Choices) == 0 {
				return false, fmt.Errorf("wizard step %q has no choices", step.ID)
			}
			return w.finishStep(wizard.Answer{Kind: step.Kind, ChoiceID: step.Choices[w.pickCursor].ID})
		}
	case wizard.KindText, wizard.KindMultiLine:
		if msg.Type == tea.KeyEnter {
			return w.finishStep(wizard.Answer{Kind: step.Kind, Text: strings.TrimSpace(w.input.Value())})
		}
		var cmd tea.Cmd
		w.input, cmd = w.input.Update(msg)
		if cmd != nil {
			_ = cmd()
		}
	case wizard.KindConfirm:
		switch msg.String() {
		case "y", "Y":
			w.confirmYes = true
			return w.finishStep(wizard.Answer{Kind: step.Kind, Confirmed: true})
		case "n", "N":
			w.confirmYes = false
			return w.finishStep(wizard.Answer{Kind: step.Kind, Confirmed: false})
		case "left", "h":
			w.confirmYes = false
		case "right", "l", " ":
			w.confirmYes = !w.confirmYes
		case "enter":
			return w.finishStep(wizard.Answer{Kind: step.Kind, Confirmed: w.confirmYes})
		}
	}
	return false, nil
}

func (w *agentsWizardState) View() string {
	step, ok := w.flow.ActiveStep()
	if !ok {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %d/%d\n\n%s\n\n", w.title, w.flow.Index()+1, w.flow.Len(), step.Prompt)
	switch step.Kind {
	case wizard.KindPick:
		for i, choice := range step.Choices {
			cursor := " "
			if i == w.pickCursor {
				cursor = ">"
			}
			fmt.Fprintf(&b, "%s %s\n", cursor, choice.Label)
		}
	case wizard.KindText, wizard.KindMultiLine:
		b.WriteString(w.input.View())
		b.WriteByte('\n')
	case wizard.KindConfirm:
		if w.confirmYes {
			b.WriteString("> yes    no\n")
		} else {
			b.WriteString("  yes  > no\n")
		}
	}
	b.WriteString("\nEnter submit  Esc cancel")
	return b.String()
}

func (w *agentsWizardState) finishStep(answer wizard.Answer) (bool, error) {
	done := w.flow.Finish(answer)
	w.pickCursor = 0
	if done {
		return true, nil
	}
	w.prepareInput()
	return false, nil
}

func (w *agentsWizardState) prepareInput() {
	_, ok := w.flow.ActiveStep()
	if !ok {
		return
	}
	input := textinput.New()
	input.Prompt = "> "
	input.Focus()
	w.input = input
}

func (w *agentsWizardState) spawnOptions() (goncho.CreateAgentOptions, error) {
	if !w.flow.Bool("confirm") {
		return goncho.CreateAgentOptions{}, fmt.Errorf("spawn cancelled")
	}
	name := strings.TrimSpace(w.flow.Text("name"))
	if name == "" {
		return goncho.CreateAgentOptions{}, fmt.Errorf("agent name is required")
	}
	return goncho.CreateAgentOptions{Name: name, Persona: strings.TrimSpace(w.flow.Text("persona"))}, nil
}

func (w *agentsWizardState) bindOptions() (string, goncho.BindingMatch, error) {
	if !w.flow.Bool("confirm") {
		return "", goncho.BindingMatch{}, fmt.Errorf("bind cancelled")
	}
	agentID := strings.TrimSpace(w.flow.Choice("agent"))
	match := goncho.BindingMatch{
		Channel:  strings.TrimSpace(w.flow.Choice("channel")),
		PeerKind: strings.TrimSpace(w.flow.Choice("peer_kind")),
		PeerID:   strings.TrimSpace(w.flow.Text("peer_id")),
		ThreadID: strings.TrimSpace(w.flow.Text("thread_id")),
	}
	if agentID == "" {
		return "", goncho.BindingMatch{}, fmt.Errorf("agent is required")
	}
	if match.Channel == "" || match.PeerKind == "" || match.PeerID == "" {
		return "", goncho.BindingMatch{}, fmt.Errorf("binding channel, peer kind, and peer ID are required")
	}
	return agentID, match, nil
}

type memoryAgentsRegistry struct{}

func (memoryAgentsRegistry) List(ctx context.Context) ([]goncho.AgentRecord, error) {
	return withDynamicRegistry(ctx, func(registry *goncho.DynamicAgentRegistry) ([]goncho.AgentRecord, error) {
		return registry.List(ctx)
	})
}

func (memoryAgentsRegistry) Create(ctx context.Context, opts goncho.CreateAgentOptions) (goncho.AgentRecord, error) {
	return withDynamicRegistry(ctx, func(registry *goncho.DynamicAgentRegistry) (goncho.AgentRecord, error) {
		return registry.Create(ctx, opts)
	})
}

func (memoryAgentsRegistry) Bind(ctx context.Context, agentID string, match goncho.BindingMatch) error {
	_, err := withDynamicRegistry(ctx, func(registry *goncho.DynamicAgentRegistry) (struct{}, error) {
		return struct{}{}, registry.Bind(ctx, agentID, match)
	})
	return err
}

func (memoryAgentsRegistry) Unbind(ctx context.Context, match goncho.BindingMatch) error {
	_, err := withDynamicRegistry(ctx, func(registry *goncho.DynamicAgentRegistry) (struct{}, error) {
		return struct{}{}, registry.Unbind(ctx, match)
	})
	return err
}

func (memoryAgentsRegistry) Resolve(ctx context.Context, match goncho.BindingMatch) (string, bool, error) {
	result, err := withDynamicRegistry(ctx, func(registry *goncho.DynamicAgentRegistry) (struct {
		id    string
		found bool
	}, error) {
		id, found, err := registry.Resolve(ctx, match)
		return struct {
			id    string
			found bool
		}{id: id, found: found}, err
	})
	return result.id, result.found, err
}

func withDynamicRegistry[T any](ctx context.Context, fn func(*goncho.DynamicAgentRegistry) (T, error)) (T, error) {
	var zero T
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		return zero, err
	}
	defer store.Close(ctx)
	registry, err := goncho.NewDynamicAgentRegistry(store.DB())
	if err != nil {
		return zero, err
	}
	return fn(registry)
}
