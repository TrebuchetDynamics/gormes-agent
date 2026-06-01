package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	admincontracts "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/contracts"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/navigation"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ChatRequest = admincontracts.ChatRequest
type ChatResponder = admincontracts.ChatResponder
type AgentLister = admincontracts.AgentLister

type chatLine struct {
	role string
	text string
}

// ChatScreen is a slim in-admin chat surface. It deliberately does not embed
// the dedicated Gormes TUI; it only gives operators a quick provider smoke
// turn and active-agent switch inside `gormes admin`.
type Screen struct {
	responder ChatResponder
	agents    AgentLister

	activeAgent string
	input       textinput.Model
	lines       []chatLine
	status      string
	sending     bool

	pickerOpen   bool
	pickerCursor int
	pickerAgents []goncho.AgentRecord
}

// ChatOption configures a ChatScreen.
type Option func(*Screen)

// WithChatResponder replaces the default provider-backed responder.
func WithChatResponder(responder ChatResponder) Option {
	return func(s *Screen) {
		if responder != nil {
			s.responder = responder
		}
	}
}

// WithAgentLister replaces the default Goncho dynamic-agent lister.
func WithAgentLister(lister AgentLister) Option {
	return func(s *Screen) {
		if lister != nil {
			s.agents = lister
		}
	}
}

// NewChatScreen returns the admin Chat tab.
func NewScreen(opts ...Option) *Screen {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "Message"
	input.Focus()
	s := &Screen{
		responder:   providerChatResponder{},
		agents:      defaultAgentLister{},
		activeAgent: defaultChatAgentID(),
		input:       input,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Screen) Title() string { return "Chat" }

func (s *Screen) Init() tea.Cmd { return textinput.Blink }

func (s *Screen) Update(msg tea.Msg) (admincontracts.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case chatResponseMsg:
		s.sending = false
		if msg.err != nil {
			s.status = "send failed: " + msg.err.Error()
			return s, nil
		}
		s.lines = append(s.lines, chatLine{role: "assistant", text: msg.text})
		s.status = "ready"
	case chatAgentListMsg:
		if msg.err != nil {
			s.status = "agent picker failed: " + msg.err.Error()
			return s, nil
		}
		s.pickerAgents = append([]goncho.AgentRecord(nil), msg.agents...)
		s.pickerCursor = 0
		s.pickerOpen = true
	case tea.KeyMsg:
		if s.pickerOpen {
			return s.updateAgentPicker(msg)
		}
		switch msg.String() {
		case "enter":
			return s.submit()
		case "a":
			if strings.TrimSpace(s.input.Value()) == "" {
				return s, s.loadAgentsCmd()
			}
		}
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return s, cmd
	default:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *Screen) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Chat scroll\nAgent: %s\n", s.activeAgent)
	if s.status != "" {
		fmt.Fprintf(&b, "Status: %s\n", s.status)
	}
	b.WriteByte('\n')
	if s.pickerOpen {
		b.WriteString(s.renderAgentPicker())
		b.WriteByte('\n')
		return b.String()
	}
	if len(s.lines) == 0 {
		b.WriteString("No messages yet.\n")
	} else {
		for _, line := range s.lines {
			fmt.Fprintf(&b, "%s: %s\n", line.role, line.text)
		}
	}
	if s.sending {
		b.WriteString("assistant: ...\n")
	}
	b.WriteByte('\n')
	b.WriteString(s.input.View())
	b.WriteByte('\n')
	return b.String()
}

func (s *Screen) ShortHelp() []admincontracts.KeyHelp {
	return []admincontracts.KeyHelp{
		{Keys: []string{"enter"}, Description: "send message"},
		{Keys: []string{"a"}, Description: "choose agent"},
		{Keys: []string{"esc"}, Description: "return to previous screen"},
	}
}

func (s *Screen) submit() (admincontracts.Screen, tea.Cmd) {
	prompt := strings.TrimSpace(s.input.Value())
	if prompt == "" || s.sending {
		return s, nil
	}
	s.input.SetValue("")
	s.lines = append(s.lines, chatLine{role: "you", text: prompt})
	s.sending = true
	s.status = "sending"
	messages := make([]llm.Message, 0, len(s.lines))
	for _, line := range s.lines {
		role := line.role
		if role == "you" {
			role = "user"
		}
		messages = append(messages, llm.Message{Role: role, Content: line.text})
	}
	req := ChatRequest{
		AgentID:  s.activeAgent,
		Prompt:   prompt,
		Messages: messages,
	}
	responder := s.responder
	return s, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		text, err := responder.Respond(ctx, req)
		return chatResponseMsg{text: text, err: err}
	}
}

func (s *Screen) loadAgentsCmd() tea.Cmd {
	lister := s.agents
	return func() tea.Msg {
		agents, err := lister.ListAgents(context.Background())
		return chatAgentListMsg{agents: agents, err: err}
	}
}

func (s *Screen) updateAgentPicker(msg tea.KeyMsg) (admincontracts.Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.pickerOpen = false
	case "up", "k":
		s.pickerCursor = navigation.MoveIndex(s.pickerCursor, len(s.pickerAgents), -1)
	case "down", "j":
		s.pickerCursor = navigation.MoveIndex(s.pickerCursor, len(s.pickerAgents), 1)
	case "enter":
		if len(s.pickerAgents) > 0 {
			s.activeAgent = s.pickerAgents[s.pickerCursor].ID
			s.status = "agent selected: " + s.activeAgent
		}
		s.pickerOpen = false
	}
	return s, nil
}

func (s *Screen) renderAgentPicker() string {
	var b strings.Builder
	b.WriteString("Agent picker\n")
	if len(s.pickerAgents) == 0 {
		b.WriteString("No dynamic agents found.\n")
		return b.String()
	}
	for i, agent := range s.pickerAgents {
		cursor := " "
		if i == s.pickerCursor {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s (%s)\n", cursor, agent.Name, agent.ID)
	}
	return b.String()
}

type chatResponseMsg struct {
	text string
	err  error
}

type chatAgentListMsg struct {
	agents []goncho.AgentRecord
	err    error
}

type providerChatResponder struct{}

func (providerChatResponder) Respond(ctx context.Context, req ChatRequest) (string, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return "", err
	}
	provider, endpoint, apiKey := resolveChatProviderCredentials(cfg)
	if endpoint == "" {
		return "", errors.New("provider endpoint is not configured")
	}
	if apiKey == "" {
		return "", errors.New("provider auth is not configured")
	}
	model := strings.TrimSpace(cfg.Hermes.Model)
	if model == "" {
		model = "hermes-agent"
	}
	client := llm.NewHTTPClientWithProvider(endpoint, apiKey, provider)
	stream, err := client.OpenStream(ctx, llm.ChatRequest{
		Model:    model,
		Messages: req.Messages,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var b strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		switch ev.Kind {
		case llm.EventToken:
			b.WriteString(ev.Token)
		case llm.EventDone:
			return strings.TrimSpace(b.String()), nil
		}
	}
	if strings.TrimSpace(b.String()) == "" {
		return "", errors.New("provider returned no assistant content")
	}
	return strings.TrimSpace(b.String()), nil
}

func resolveChatProviderCredentials(cfg config.Config) (provider, endpoint, apiKey string) {
	provider = strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		provider = "custom"
	}
	endpoint = strings.TrimSpace(cfg.Hermes.Endpoint)
	apiKey = strings.TrimSpace(cfg.Hermes.APIKey)
	if apiKey != "" {
		return provider, endpoint, apiKey
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return provider, endpoint, apiKey
	}
	credential, _ := pool.Select()
	if credential == nil {
		return provider, endpoint, apiKey
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(firstNonEmpty(credential.InferenceBaseURL, credential.BaseURL))
	}
	apiKey = strings.TrimSpace(credential.AccessToken)
	return provider, endpoint, apiKey
}

type defaultAgentLister struct{}

func (defaultAgentLister) ListAgents(ctx context.Context) ([]goncho.AgentRecord, error) {
	path := config.MemoryDBPath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	registry, err := goncho.NewDynamicAgentRegistry(db)
	if err != nil {
		return nil, err
	}
	return registry.List(ctx)
}

func defaultChatAgentID() string {
	cfg, err := config.Load(nil)
	if err != nil {
		return config.DefaultAgentID
	}
	return cfg.Agents.DefaultAgentID()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
