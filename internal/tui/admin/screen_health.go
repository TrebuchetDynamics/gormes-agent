package admin

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/ncruces/go-sqlite3/driver"
)

const (
	healthItemProvider = "provider"
	healthItemAuth     = "auth"
)

// HealthItem is one operator-facing row in the Setup health screen.
type HealthItem struct {
	ID      string
	Status  doctor.Status
	Title   string
	Detail  string
	Fixable bool
}

// HealthSource recomputes the Setup health rows from current runtime state.
type HealthSource interface {
	Check(context.Context) ([]HealthItem, error)
}

type healthSourceFunc func(context.Context) ([]HealthItem, error)

func (f healthSourceFunc) Check(ctx context.Context) ([]HealthItem, error) {
	return f(ctx)
}

// SetupHealthScreen is the default admin tab. It is intentionally read-mostly:
// checks derive from existing config/auth/Goncho state and fixes write through
// the same config surfaces used by setup/auth commands.
type SetupHealthScreen struct {
	source   HealthSource
	items    []HealthItem
	selected int
	message  string
	err      error
	fix      *providerFixState
}

// HealthOption configures a SetupHealthScreen.
type HealthOption func(*SetupHealthScreen)

// WithHealthSource replaces the default config-backed health source.
func WithHealthSource(source HealthSource) HealthOption {
	return func(s *SetupHealthScreen) {
		if source != nil {
			s.source = source
		}
	}
}

// NewSetupHealthScreen returns the Setup tab used by `gormes admin`.
func NewSetupHealthScreen(opts ...HealthOption) *SetupHealthScreen {
	s := &SetupHealthScreen{source: defaultHealthSource{}}
	for _, opt := range opts {
		opt(s)
	}
	s.refresh(context.Background())
	return s
}

type defaultScreensConfig struct {
	commandEntries []CommandEntry
}

// DefaultScreensOption configures the production admin screen registry.
type DefaultScreensOption func(*defaultScreensConfig)

// WithCommandEntries adds the root CLI command catalog to the Commands tab.
func WithCommandEntries(entries []CommandEntry) DefaultScreensOption {
	return func(cfg *defaultScreensConfig) {
		cfg.commandEntries = cloneCommandEntries(entries)
	}
}

// NewDefaultScreens returns the ordered admin screen registry. The Setup
// health screen is intentionally first so `gormes admin` opens on actionable
// configuration state.
func NewDefaultScreens(opts ...DefaultScreensOption) []Screen {
	var cfg defaultScreensConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return []Screen{
		NewSetupHealthScreen(),
		NewChatScreen(),
		NewAgentsScreen(),
		NewCommandsScreen(cfg.commandEntries),
	}
}

func (s *SetupHealthScreen) Title() string { return "Setup" }

func (s *SetupHealthScreen) Init() tea.Cmd {
	return s.refreshCmd()
}

func (s *SetupHealthScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if s.fix != nil {
		return s.updateFix(msg)
	}
	switch msg := msg.(type) {
	case healthCheckedMsg:
		s.applyHealth(msg.items, msg.err)
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			return s, s.refreshCmd()
		case "up", "k":
			if s.selected > 0 {
				s.selected--
			}
		case "down", "j":
			if s.selected < len(s.items)-1 {
				s.selected++
			}
		case "enter":
			if item, ok := s.selectedItem(); ok && item.Fixable {
				cfg, _ := config.Load(nil)
				s.fix = newProviderFixState(cfg)
				s.message = ""
			}
		}
	}
	return s, nil
}

func (s *SetupHealthScreen) View() string {
	if s.fix != nil {
		return s.fix.View()
	}
	var b strings.Builder
	b.WriteString("Setup health\n")
	if s.err != nil {
		fmt.Fprintf(&b, "health_error: %v\n", s.err)
	}
	if s.message != "" {
		fmt.Fprintf(&b, "%s\n", s.message)
	}
	b.WriteString("\n")
	if len(s.items) == 0 {
		b.WriteString("no health checks available\n")
		return b.String()
	}
	for i, item := range s.items {
		cursor := " "
		if i == s.selected {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s %s - %s", cursor, item.Status.Symbol(), item.Title, item.Detail)
		if item.Fixable {
			b.WriteString(" [Fix]")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (s *SetupHealthScreen) ShortHelp() []KeyHelp {
	return []KeyHelp{
		{Keys: []string{"r"}, Description: "refresh checks"},
		{Keys: []string{"up", "down"}, Description: "select check"},
		{Keys: []string{"enter"}, Description: "run fix action"},
	}
}

func (s *SetupHealthScreen) Items() []HealthItem {
	out := make([]HealthItem, len(s.items))
	copy(out, s.items)
	return out
}

func (s *SetupHealthScreen) selectedItem() (HealthItem, bool) {
	if s.selected < 0 || s.selected >= len(s.items) {
		return HealthItem{}, false
	}
	return s.items[s.selected], true
}

func (s *SetupHealthScreen) updateFix(msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if key.Type == tea.KeyEscape {
		s.fix = nil
		s.message = "fix cancelled"
		return s, nil
	}
	done, err := s.fix.Update(key)
	if err != nil {
		s.fix = nil
		s.message = "fix failed: " + err.Error()
		return s, nil
	}
	if !done {
		return s, nil
	}
	result, err := s.fix.Result()
	if err == nil {
		err = writeProviderSetup(result)
	}
	s.fix = nil
	if err != nil {
		s.message = "fix failed: " + err.Error()
		return s, nil
	}
	s.message = "provider configured"
	s.refresh(context.Background())
	return s, nil
}

func (s *SetupHealthScreen) refresh(ctx context.Context) {
	items, err := s.source.Check(ctx)
	s.applyHealth(items, err)
}

func (s *SetupHealthScreen) refreshCmd() tea.Cmd {
	source := s.source
	return func() tea.Msg {
		items, err := source.Check(context.Background())
		return healthCheckedMsg{items: items, err: err}
	}
}

func (s *SetupHealthScreen) applyHealth(items []HealthItem, err error) {
	s.items = append([]HealthItem(nil), items...)
	s.err = err
	if s.selected >= len(s.items) {
		s.selected = len(s.items) - 1
	}
	if s.selected < 0 {
		s.selected = 0
	}
}

type healthCheckedMsg struct {
	items []HealthItem
	err   error
}

type providerSetupResult struct {
	Provider string
	Endpoint string
	APIKey   string
	Model    string
}

type providerFixState struct {
	steps      []wizard.Step
	answers    map[string]wizard.Answer
	defaults   providerSetupResult
	index      int
	input      textinput.Model
	pickCursor int
}

var adminProviderEndpoints = map[string]string{
	"openai":       "https://api.openai.com/v1",
	"anthropic":    "https://api.anthropic.com/v1",
	"deepseek":     "https://api.deepseek.com/v1",
	"groq":         "https://api.groq.com/openai/v1",
	"ollama":       "http://localhost:11434/v1",
	"openai-codex": "https://chatgpt.com/backend-api/codex",
	"opencode":     "https://opencode.ai/zen/v1",
}

var adminProviderModels = map[string]string{
	"openai":       "gpt-4o",
	"anthropic":    "claude-sonnet-4-20250514",
	"deepseek":     "deepseek-chat",
	"groq":         "llama-3.3-70b-versatile",
	"ollama":       "llama3",
	"openai-codex": "gpt-5.2",
	"opencode":     "gpt-5.2",
}

func newProviderFixState(cfg config.Config) *providerFixState {
	defaults := providerSetupDefaults(cfg)
	f := &providerFixState{
		steps: []wizard.Step{
			wizard.Pick("provider", "Provider", []wizard.Choice{
				{ID: "openai", Label: "OpenAI"},
				{ID: "anthropic", Label: "Anthropic"},
				{ID: "deepseek", Label: "DeepSeek"},
				{ID: "groq", Label: "Groq"},
				{ID: "ollama", Label: "Ollama"},
				{ID: "openai-codex", Label: "OpenAI Codex"},
				{ID: "opencode", Label: "OpenCode"},
				{ID: "custom", Label: "Custom endpoint"},
			}),
			wizard.Text("endpoint", "Endpoint URL"),
			wizard.Password("api_key", "API key"),
			wizard.Text("model", "Model"),
		},
		answers:    map[string]wizard.Answer{},
		defaults:   defaults,
		pickCursor: providerChoiceIndex(defaults.Provider),
	}
	f.prepareInput()
	return f
}

func (f *providerFixState) Update(msg tea.KeyMsg) (bool, error) {
	step, ok := f.activeStep()
	if !ok {
		return true, nil
	}
	switch step.Kind {
	case wizard.KindPick:
		switch msg.String() {
		case "up", "k":
			if f.pickCursor > 0 {
				f.pickCursor--
			}
		case "down", "j":
			if f.pickCursor < len(step.Choices)-1 {
				f.pickCursor++
			}
		case "enter":
			if len(step.Choices) == 0 {
				return false, fmt.Errorf("provider choices missing")
			}
			return f.finishStep(wizard.Answer{Kind: step.Kind, ChoiceID: step.Choices[f.pickCursor].ID})
		}
	case wizard.KindText, wizard.KindPassword:
		if msg.Type == tea.KeyEnter {
			return f.finishStep(wizard.Answer{Kind: step.Kind, Text: strings.TrimSpace(f.input.Value())})
		}
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		if cmd != nil {
			_ = cmd()
		}
	default:
		return false, fmt.Errorf("unsupported provider fix step %q", step.Kind)
	}
	return false, nil
}

func (f *providerFixState) View() string {
	step, ok := f.activeStep()
	if !ok {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Provider setup %d/%d\n\n", f.index+1, len(f.steps))
	fmt.Fprintf(&b, "%s\n\n", step.Prompt)
	switch step.Kind {
	case wizard.KindPick:
		for i, choice := range step.Choices {
			cursor := " "
			if i == f.pickCursor {
				cursor = ">"
			}
			fmt.Fprintf(&b, "%s %s\n", cursor, choice.Label)
		}
	case wizard.KindText, wizard.KindPassword:
		b.WriteString(f.input.View())
		b.WriteByte('\n')
	}
	b.WriteString("\nEnter submit  Esc cancel")
	return b.String()
}

func (f *providerFixState) Result() (providerSetupResult, error) {
	provider := strings.TrimSpace(f.answers["provider"].ChoiceID)
	if provider == "" {
		provider = "openai"
	}
	endpoint := strings.TrimSpace(f.answers["endpoint"].Text)
	if endpoint == "" {
		endpoint = adminProviderEndpoints[provider]
	}
	apiKey := strings.TrimSpace(f.answers["api_key"].Text)
	model := strings.TrimSpace(f.answers["model"].Text)
	if model == "" {
		model = adminProviderModels[provider]
	}
	if endpoint == "" {
		return providerSetupResult{}, fmt.Errorf("endpoint URL is required")
	}
	if apiKey == "" {
		return providerSetupResult{}, fmt.Errorf("API key is required")
	}
	return providerSetupResult{
		Provider: provider,
		Endpoint: endpoint,
		APIKey:   apiKey,
		Model:    model,
	}, nil
}

func (f *providerFixState) activeStep() (wizard.Step, bool) {
	if f == nil || f.index < 0 || f.index >= len(f.steps) {
		return wizard.Step{}, false
	}
	return f.steps[f.index], true
}

func (f *providerFixState) finishStep(answer wizard.Answer) (bool, error) {
	step, ok := f.activeStep()
	if !ok {
		return true, nil
	}
	f.answers[step.ID] = answer
	f.index++
	if f.index >= len(f.steps) {
		return true, nil
	}
	f.prepareInput()
	return false, nil
}

func (f *providerFixState) prepareInput() {
	step, ok := f.activeStep()
	if !ok {
		return
	}
	input := textinput.New()
	input.Focus()
	input.Prompt = "> "
	switch step.Kind {
	case wizard.KindPassword:
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '*'
	case wizard.KindText:
		input.SetValue(f.defaultTextValue(step.ID))
		input.CursorEnd()
	}
	f.input = input
}

func (f *providerFixState) defaultTextValue(stepID string) string {
	provider := f.selectedProvider()
	switch stepID {
	case "endpoint":
		if strings.TrimSpace(f.defaults.Endpoint) != "" && provider == strings.TrimSpace(f.defaults.Provider) {
			return f.defaults.Endpoint
		}
		return adminProviderEndpoints[provider]
	case "model":
		if strings.TrimSpace(f.defaults.Model) != "" && provider == strings.TrimSpace(f.defaults.Provider) {
			return f.defaults.Model
		}
		return adminProviderModels[provider]
	default:
		return ""
	}
}

func (f *providerFixState) selectedProvider() string {
	provider := strings.TrimSpace(f.answers["provider"].ChoiceID)
	if provider != "" {
		return provider
	}
	if defaults := strings.TrimSpace(f.defaults.Provider); defaults != "" {
		return defaults
	}
	return "openai"
}

func providerSetupDefaults(cfg config.Config) providerSetupResult {
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		provider = "openai"
	}
	if _, ok := adminProviderEndpoints[provider]; !ok {
		provider = "custom"
	}
	return providerSetupResult{
		Provider: provider,
		Endpoint: strings.TrimSpace(cfg.Hermes.Endpoint),
		Model:    strings.TrimSpace(cfg.Hermes.Model),
	}
}

func providerChoiceIndex(provider string) int {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "openai"
	}
	choices := []string{"openai", "anthropic", "deepseek", "groq", "ollama", "openai-codex", "opencode", "custom"}
	for i, choice := range choices {
		if choice == provider {
			return i
		}
	}
	return len(choices) - 1
}

func writeProviderSetup(result providerSetupResult) error {
	provider := strings.TrimSpace(result.Provider)
	endpoint := strings.TrimSpace(result.Endpoint)
	apiKey := strings.TrimSpace(result.APIKey)
	model := strings.TrimSpace(result.Model)
	if provider == "" {
		provider = "custom"
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.provider", provider); err != nil {
		return fmt.Errorf("write provider: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.endpoint", endpoint); err != nil {
		return fmt.Errorf("write endpoint: %w", err)
	}
	if model != "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.model", model); err != nil {
			return fmt.Errorf("write model: %w", err)
		}
	}
	if err := config.WriteEnvValue(config.EnvPath(), "GORMES_API_KEY", apiKey); err != nil {
		return fmt.Errorf("write API key: %w", err)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return fmt.Errorf("load credential pool: %w", err)
	}
	entries := pool.Entries()
	entries = append(entries, config.PooledCredential{
		ID:               nextAdminCredentialID(provider, entries),
		Label:            "admin",
		AuthType:         config.CredentialAuthAPIKey,
		Source:           "manual",
		AccessToken:      apiKey,
		BaseURL:          endpoint,
		InferenceBaseURL: endpoint,
		LastStatus:       config.CredentialStatusOK,
	})
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		return fmt.Errorf("save credential pool: %w", err)
	}
	return nil
}

func nextAdminCredentialID(provider string, entries []config.PooledCredential) string {
	base := strings.ToLower(strings.TrimSpace(provider))
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	base = strings.Trim(base, "-_")
	if base == "" {
		base = "provider"
	}
	for i := 1; ; i++ {
		id := fmt.Sprintf("%s-admin-%d", base, i)
		if !slices.ContainsFunc(entries, func(entry config.PooledCredential) bool {
			return entry.ID == id
		}) {
			return id
		}
	}
}

type defaultHealthSource struct{}

func (defaultHealthSource) Check(context.Context) ([]HealthItem, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return nil, err
	}
	return buildHealthItems(cfg), nil
}

func buildHealthItems(cfg config.Config) []HealthItem {
	items := make([]HealthItem, 0, 9)
	items = append(items,
		profileHealthItem(),
		providerHealthItem(cfg),
		authHealthItem(cfg),
		defaultAgentHealthItem(cfg),
		agentTemplateHealthItem(cfg),
		memoryDBHealthItem(),
		telegramHealthItem(cfg),
		discordHealthItem(cfg),
		slackHealthItem(cfg),
	)
	return items
}

func profileHealthItem() HealthItem {
	path := config.ConfigPath()
	if _, err := os.Stat(path); err == nil {
		return HealthItem{ID: "profile", Status: doctor.StatusPass, Title: "profile initialized", Detail: path}
	}
	return HealthItem{ID: "profile", Status: doctor.StatusWarn, Title: "profile missing", Detail: "config.toml has not been written"}
}

func providerHealthItem(cfg config.Config) HealthItem {
	endpoint := strings.TrimSpace(cfg.Hermes.Endpoint)
	model := strings.TrimSpace(cfg.Hermes.Model)
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if endpoint != "" && model != "" {
		label := provider
		if label == "" {
			label = "custom"
		}
		return HealthItem{ID: healthItemProvider, Status: doctor.StatusPass, Title: "provider configured", Detail: fmt.Sprintf("%s %s model=%s", label, endpoint, model)}
	}
	return HealthItem{ID: healthItemProvider, Status: doctor.StatusFail, Title: "no provider", Detail: "hermes.endpoint is not configured", Fixable: true}
}

func authHealthItem(cfg config.Config) HealthItem {
	if providerAuthPresent(cfg) {
		return HealthItem{ID: healthItemAuth, Status: doctor.StatusPass, Title: "auth credentials present", Detail: "provider credential available"}
	}
	return HealthItem{ID: healthItemAuth, Status: doctor.StatusFail, Title: "no auth credentials", Detail: "no API key, secret ref, or credential pool entry", Fixable: true}
}

func providerAuthPresent(cfg config.Config) bool {
	if strings.TrimSpace(cfg.Hermes.APIKey) != "" || cfg.Hermes.APIKeyRef != nil {
		return true
	}
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		return false
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return false
	}
	for _, entry := range pool.Entries() {
		if strings.TrimSpace(entry.AccessToken) != "" || strings.TrimSpace(entry.RefreshToken) != "" || strings.TrimSpace(entry.AgentKey) != "" {
			return true
		}
	}
	return false
}

func defaultAgentHealthItem(cfg config.Config) HealthItem {
	id := cfg.Agents.DefaultAgentID()
	if agent, ok := cfg.Agents.AgentByID(id); ok {
		return HealthItem{ID: "default-agent", Status: doctor.StatusPass, Title: "default agent created", Detail: fmt.Sprintf("%s at %s", agent.ID, agent.AgentDir)}
	}
	return HealthItem{ID: "default-agent", Status: doctor.StatusFail, Title: "default agent missing", Detail: "agents.list has no default agent"}
}

func agentTemplateHealthItem(cfg config.Config) HealthItem {
	id := cfg.Agents.DefaultAgentID()
	agent, ok := cfg.Agents.AgentByID(id)
	if !ok {
		return HealthItem{ID: "agent-template", Status: doctor.StatusFail, Title: "agent template missing", Detail: "default agent is unavailable"}
	}
	for _, file := range agenttemplate.DefaultFiles() {
		if _, err := os.Stat(filepath.Join(agent.AgentDir, file.Path)); err != nil {
			return HealthItem{ID: "agent-template", Status: doctor.StatusWarn, Title: "agent template missing", Detail: filepath.ToSlash(file.Path)}
		}
	}
	return HealthItem{ID: "agent-template", Status: doctor.StatusPass, Title: "agent template seeded", Detail: agent.AgentDir}
}

func memoryDBHealthItem() HealthItem {
	path := config.MemoryDBPath()
	if _, err := os.Stat(path); err != nil {
		return HealthItem{ID: "memory-db", Status: doctor.StatusWarn, Title: "Goncho memory.db missing", Detail: path}
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return HealthItem{ID: "memory-db", Status: doctor.StatusFail, Title: "Goncho memory.db unreachable", Detail: err.Error()}
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return HealthItem{ID: "memory-db", Status: doctor.StatusFail, Title: "Goncho memory.db unreachable", Detail: err.Error()}
	}
	return HealthItem{ID: "memory-db", Status: doctor.StatusPass, Title: "Goncho memory.db reachable", Detail: path}
}

func telegramHealthItem(cfg config.Config) HealthItem {
	if strings.TrimSpace(cfg.Telegram.BotToken) != "" || cfg.Telegram.BotTokenRef != nil {
		return HealthItem{ID: "telegram-token", Status: doctor.StatusPass, Title: "Telegram token present", Detail: "bot token configured"}
	}
	return HealthItem{ID: "telegram-token", Status: doctor.StatusWarn, Title: "Telegram token missing", Detail: "gateway Telegram bot token not configured"}
}

func discordHealthItem(cfg config.Config) HealthItem {
	if strings.TrimSpace(cfg.Discord.Token) != "" || cfg.Discord.TokenRef != nil {
		return HealthItem{ID: "discord-token", Status: doctor.StatusPass, Title: "Discord token present", Detail: "bot token configured"}
	}
	return HealthItem{ID: "discord-token", Status: doctor.StatusWarn, Title: "Discord token missing", Detail: "gateway Discord token not configured"}
}

func slackHealthItem(cfg config.Config) HealthItem {
	if (strings.TrimSpace(cfg.Slack.BotToken) != "" || cfg.Slack.BotTokenRef != nil) &&
		(strings.TrimSpace(cfg.Slack.AppToken) != "" || cfg.Slack.AppTokenRef != nil) {
		return HealthItem{ID: "slack-tokens", Status: doctor.StatusPass, Title: "Slack tokens present", Detail: "bot and app tokens configured"}
	}
	return HealthItem{ID: "slack-tokens", Status: doctor.StatusWarn, Title: "Slack tokens missing", Detail: "gateway Slack bot/app tokens not configured"}
}
