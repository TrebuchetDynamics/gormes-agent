// Package tui — Consolidated widget facades.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/autotitle"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/completion"
	drafthistory "github.com/TrebuchetDynamics/gormes-agent/internal/tui/history"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/keyhelp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/todo"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/transientpage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/details"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessiontree"
)

// ─── Auto-title ─────────────────────────────────────────────────────────────

type AutoTitleInput = autotitle.Input
type AutoTitleRequest = autotitle.Request

func BuildAutoTitleRequest(in AutoTitleInput) (AutoTitleRequest, bool) {
	return autotitle.BuildRequest(in)
}

// ─── Completion request ─────────────────────────────────────────────────────

type TUICompletionMethod = completion.Method

const (
	TUICompletionPath  TUICompletionMethod = completion.Path
	TUICompletionSlash TUICompletionMethod = completion.Slash
)

type TUICompletionRequest = completion.Request

func CompletionRequestForInput(input string) (TUICompletionRequest, bool) {
	return completion.RequestForInput(input)
}

// ─── Details ────────────────────────────────────────────────────────────────

type DetailsMode = details.Mode

const (
	DetailsModeHidden    DetailsMode = details.ModeHidden
	DetailsModeCollapsed DetailsMode = details.ModeCollapsed
	DetailsModeExpanded  DetailsMode = details.ModeExpanded
)

type DetailsSection = details.Section

const (
	DetailsSectionThinking  DetailsSection = details.SectionThinking
	DetailsSectionTools     DetailsSection = details.SectionTools
	DetailsSectionSubagents DetailsSection = details.SectionSubagents
	DetailsSectionActivity  DetailsSection = details.SectionActivity
)

type DetailsState = details.State

func DefaultDetailsState() DetailsState              { return details.DefaultState() }
func NormalizeDetailsState(state DetailsState) DetailsState { return details.NormalizeState(state) }
func ParseDetailsMode(raw string) (DetailsMode, bool)       { return details.ParseMode(raw) }
func ParseDetailsSection(raw string) (DetailsSection, bool) { return details.ParseSection(raw) }
func NextDetailsMode(mode DetailsMode) DetailsMode           { return details.NextMode(mode) }

// ─── History ────────────────────────────────────────────────────────────────

type HermesHistory = drafthistory.HermesHistory

func NewHermesHistory() *HermesHistory {
	return drafthistory.NewHermesHistory()
}

// ─── Indicator ──────────────────────────────────────────────────────────────

type IndicatorStyle = indicator.Style

const (
	IndicatorStyleASCII   IndicatorStyle = indicator.StyleASCII
	IndicatorStyleEmoji   IndicatorStyle = indicator.StyleEmoji
	IndicatorStyleKaomoji IndicatorStyle = indicator.StyleKaomoji
	IndicatorStyleUnicode IndicatorStyle = indicator.StyleUnicode
)

func NormalizeIndicatorStyle(raw string) IndicatorStyle {
	return indicator.NormalizeStyle(raw)
}

func RenderIndicatorFrame(style IndicatorStyle, frameIndex int) string {
	return indicator.RenderFrame(style, frameIndex)
}

// ─── Key help ───────────────────────────────────────────────────────────────

type KeyHelp = keyhelp.Item

type KeyHelpProvider interface {
	KeyHelp() []KeyHelp
}

func RenderKeyHelpBar(width int, skin HermesSkin, items []KeyHelp) string {
	return keyhelp.RenderBar(width, keyHelpStyles(skin), items)
}

func RenderKeyBindingHelpBar(width int, skin HermesSkin, bindings []key.Binding) string {
	return keyhelp.RenderBindingBar(width, keyHelpStyles(skin), bindings)
}

func keyHelpStyles(skin HermesSkin) keyhelp.Styles {
	styles := SkinStylesFor(skin)
	return keyhelp.Styles{
		Separator: styles.Dim,
		Key:       styles.Label,
		Desc:      styles.Dim,
	}
}

// ─── Prompt templates ───────────────────────────────────────────────────────

type PromptTemplateCatalog = prompttemplates.Catalog

func PromptTemplateCatalogFromRoots(roots []string) PromptTemplateCatalog {
	catalog, _ := prompttemplates.Discover(prompttemplates.DiscoverOptions{
		Roots:         roots,
		ReservedNames: DefaultSlashCommandNames(),
	})
	return catalog
}

// ─── Queued messages ────────────────────────────────────────────────────────

const QueueWindowSize = queue.WindowSize

type QueuedMessages = queue.Messages
type QueueWindow = queue.Window

func ComputeQueueWindow(n int, editIdx *int) QueueWindow {
	return queue.ComputeWindow(n, editIdx)
}

func (m Model) renderQueuedMessageWidgets(width int) string {
	blocks := make([]string, 0, 2)
	if steering := RenderQueuedMessageWidget("steering", m.steeringMessages, width); steering != "" {
		blocks = append(blocks, steering)
	}
	if queued := RenderQueuedMessageWidget("queued", m.queuedMessages, width); queued != "" {
		blocks = append(blocks, queued)
	}
	return strings.Join(blocks, "\n")
}

func RenderQueuedMessageWidget(label string, q QueuedMessages, width int) string {
	return queue.RenderWidget(label, q, width, hermesStatusTrimToWidth)
}

// ─── Todo panel ─────────────────────────────────────────────────────────────

type TodoStatus = todo.Status

const (
	TodoStatusPending TodoStatus = todo.StatusPending
	TodoStatusDone    TodoStatus = todo.StatusDone
)

type TodoItem = todo.Item

func RenderTodoPanel(items []TodoItem, width int) string {
	return todo.Render(items, width)
}

func RenderTodoPanelWithSkin(items []TodoItem, width int, skin HermesSkin) string {
	styles := SkinStylesFor(skin)
	return todo.RenderWithStyles(items, width, todo.Styles{
		Accent: func(text string) string { return styles.Accent.Render(text) },
		Good:   func(text string) string { return styles.Good.Render(text) },
		Dim:    func(text string) string { return styles.Dim.Render(text) },
		Text:   func(text string) string { return styles.Text.Render(text) },
	})
}

// ─── Transient page ─────────────────────────────────────────────────────────

type TransientPageState = transientpage.State

func RenderTransientPage(page TransientPageState, width, height int) string {
	return renderTransientPage(page, width, height, SkinStyles{}, false)
}

func RenderTransientPageWithSkin(page TransientPageState, width, height int, skin HermesSkin) string {
	return renderTransientPage(page, width, height, SkinStylesFor(skin), true)
}

func renderTransientPage(page TransientPageState, width, height int, styles SkinStyles, styled bool) string {
	return transientpage.Render(page, width, height, transientPageStyles(styles), styled, RenderMarkdownSoftWrapTrim)
}

func transientPageStyles(styles SkinStyles) transientpage.Styles {
	return transientpage.Styles{
		Title:     styles.Title,
		Dim:       styles.Dim,
		Separator: styles.Separator,
		Text:      styles.Text,
	}
}

func transientPageLines(body string, width int) []string {
	return transientpage.Lines(body, width, RenderMarkdownSoftWrapTrim)
}

// ─── Tree selector / session tree ───────────────────────────────────────────

type SessionTreeRequest = sessiontree.SessionTreeRequest
type SessionTreeResult = sessiontree.SessionTreeResult
type SessionTreeEntry = sessiontree.SessionTreeEntry
type SessionTreeMessage = sessiontree.SessionTreeMessage

func BuildSessionTreePage(tree SessionTreeResult) (TransientPageState, bool) {
	page, ok := sessiontree.BuildPage(tree)
	if !ok {
		return TransientPageState{}, false
	}
	return TransientPageState{Title: page.Title, Body: page.Body}, true
}