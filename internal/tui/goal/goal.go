// Package goal renders and edits persistent session goals for the native TUI.
package goal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"
)

// Line is the compact goal bar shown above the live-work dock.
type Line struct {
	Glyph  string
	Label  string
	Detail string
	Title  string
}

// BuildLine returns the Hermes-compatible goal line for an active or paused
// goal. Done and cleared goals return false so the bar disappears.
func BuildLine(state *session.GoalState) (Line, bool) {
	if state == nil {
		return Line{}, false
	}
	title := displayText(state.Goal)
	switch session.GoalStatus(strings.ToLower(strings.TrimSpace(string(state.Status)))) {
	case session.GoalStatusActive:
		return Line{
			Glyph:  "⊙",
			Label:  "goal",
			Detail: fmt.Sprintf("%d/%d turns", state.TurnsUsed, state.MaxTurns),
			Title:  title,
		}, true
	case session.GoalStatusPaused:
		detail := fmt.Sprintf("%d/%d turns", state.TurnsUsed, state.MaxTurns)
		if reason := displayText(state.PausedReason); reason != "" {
			detail = reason + " · " + detail
		}
		return Line{Glyph: "⏸", Label: "goal paused", Detail: detail, Title: title}, true
	default:
		return Line{}, false
	}
}

// RenderBar renders one goal line. Text is truncated to width when width is
// positive.
func RenderBar(line Line, width int) string {
	if line.Glyph == "" && line.Label == "" {
		return ""
	}
	head := strings.TrimSpace(line.Glyph + " " + line.Label)
	if line.Detail != "" {
		head += " · " + line.Detail
	}
	text := head
	if line.Title != "" {
		text += " · " + line.Title
	}
	if width > 0 && lipgloss.Width(text) > width {
		text = ansitext.TrimToWidth(text, width)
	}
	return text
}

// RenderStatus renders the full /goal status block.
func RenderStatus(state *session.GoalState) string {
	if state == nil || !goalVisible(state) {
		return "No goal set."
	}
	status := strings.ToLower(strings.TrimSpace(string(state.Status)))
	line := fmt.Sprintf("Goal %s (%d/%d): %s", status, state.TurnsUsed, state.MaxTurns, displayText(state.Goal))
	if reason := displayText(state.PausedReason); reason != "" {
		line += "\nPaused reason: " + reason
	}
	if len(state.Subgoals) > 0 {
		line += "\nSubgoals:"
		for i, subgoal := range state.Subgoals {
			line += fmt.Sprintf("\n  %d. %s", i+1, displayText(subgoal))
		}
	}
	return line
}

// Controller owns goal reads and writes for TUI slash commands.
type Controller struct {
	Store    session.GoalMetadataStore
	Now      func() time.Time
	MaxTurns int
}

// Status returns the compact bar for the session.
func (c *Controller) Status(sessionID string) (Line, bool) {
	state, ok, err := c.load(sessionID)
	if err != nil || !ok {
		return Line{}, false
	}
	return BuildLine(state)
}

// Handle executes one /goal invocation and returns the operator-facing reply.
func (c *Controller) Handle(sessionID, input string) (string, error) {
	if c == nil || c.Store == nil {
		return "goal: goal store unavailable", nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "goal: session unavailable", nil
	}
	args := commandArgs(input)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	ctx := context.Background()
	now := c.now()

	switch sub {
	case "", "status":
		state, ok, err := c.load(sessionID)
		if err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		if !ok {
			return "No goal set.", nil
		}
		return RenderStatus(state), nil
	case "pause":
		_, ok, err := c.load(sessionID)
		if err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		if !ok {
			return "No goal set.", nil
		}
		reason := strings.TrimSpace(strings.Join(args[1:], " "))
		if _, err := session.PauseGoal(ctx, c.Store, sessionID, reason, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return "Goal paused.", nil
	case "resume":
		_, ok, err := c.load(sessionID)
		if err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		if !ok {
			return "No goal set.", nil
		}
		if _, err := session.ResumeGoal(ctx, c.Store, sessionID, false, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return "Goal resumed.", nil
	case "clear", "stop":
		if _, ok, err := c.load(sessionID); err != nil {
			return "goal: " + displayText(err.Error()), nil
		} else if !ok {
			return "No goal set.", nil
		}
		if _, err := session.ClearGoal(ctx, c.Store, sessionID, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return "Goal cleared.", nil
	case "done":
		_, ok, err := c.load(sessionID)
		if err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		if !ok {
			return "No goal set.", nil
		}
		reason := strings.TrimSpace(strings.Join(args[1:], " "))
		if _, err := session.DoneGoal(ctx, c.Store, sessionID, reason, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return "Goal marked done.", nil
	case "subgoal", "subgoals":
		return c.handleSubgoal(sessionID, args[1:], now)
	}

	text := strings.TrimSpace(strings.TrimPrefix(input, "/goal"))
	text = strings.TrimSpace(strings.TrimPrefix(input, "/goal"))
	if strings.HasPrefix(strings.ToLower(text), "set ") {
		text = strings.TrimSpace(text[4:])
	}
	if text == "" {
		return "Usage: /goal [status|pause|resume|clear|done|subgoal add|remove|list] | /goal <objective>", nil
	}
	state, err := session.SetGoal(ctx, c.Store, sessionID, text, c.MaxTurns, now)
	if err != nil {
		return "goal: " + displayText(err.Error()), nil
	}
	return fmt.Sprintf("Goal set (%d/%d): %s", state.TurnsUsed, state.MaxTurns, displayText(state.Goal)), nil
}

func (c *Controller) handleSubgoal(sessionID string, args []string, now time.Time) (string, error) {
	ctx := context.Background()
	state, ok, err := c.load(sessionID)
	if err != nil {
		return "goal: " + displayText(err.Error()), nil
	}
	if !ok {
		return "No goal set.", nil
	}
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
	}
	switch action {
	case "list", "":
		if len(state.Subgoals) == 0 {
			return "No subgoals.", nil
		}
		lines := make([]string, 0, len(state.Subgoals)+1)
		lines = append(lines, "Subgoals:")
		for i, subgoal := range state.Subgoals {
			lines = append(lines, fmt.Sprintf("  %d. %s", i+1, displayText(subgoal)))
		}
		return strings.Join(lines, "\n"), nil
	case "add":
		text := strings.TrimSpace(strings.Join(args[1:], " "))
		if text == "" {
			return "Usage: /goal subgoal add <criterion>", nil
		}
		state.Subgoals = append(state.Subgoals, text)
		if err := session.SaveGoal(ctx, c.Store, sessionID, state, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return fmt.Sprintf("Added subgoal %d.", len(state.Subgoals)), nil
	case "remove", "rm", "delete":
		if len(args) < 2 {
			return "Usage: /goal subgoal remove <number>", nil
		}
		index, err := strconv.Atoi(strings.TrimSpace(args[1]))
		if err != nil || index < 1 || index > len(state.Subgoals) {
			return fmt.Sprintf("Subgoal number must be between 1 and %d.", len(state.Subgoals)), nil
		}
		state.Subgoals = append(state.Subgoals[:index-1], state.Subgoals[index:]...)
		if err := session.SaveGoal(ctx, c.Store, sessionID, state, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return fmt.Sprintf("Removed subgoal %d.", index), nil
	case "clear":
		state.Subgoals = nil
		if err := session.SaveGoal(ctx, c.Store, sessionID, state, now); err != nil {
			return "goal: " + displayText(err.Error()), nil
		}
		return "Cleared subgoals.", nil
	default:
		return "Usage: /goal subgoal [list|add <text>|remove <number>|clear]", nil
	}
}

func (c *Controller) load(sessionID string) (*session.GoalState, bool, error) {
	if c == nil || c.Store == nil {
		return nil, false, nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false, nil
	}
	return session.LoadGoal(context.Background(), c.Store, sessionID)
}

func (c *Controller) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func goalVisible(state *session.GoalState) bool {
	if state == nil || strings.TrimSpace(state.Goal) == "" {
		return false
	}
	switch session.GoalStatus(strings.ToLower(strings.TrimSpace(string(state.Status)))) {
	case session.GoalStatusDone, session.GoalStatusCleared:
		return false
	default:
		return true
	}
}

func commandArgs(input string) []string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return nil
	}
	return fields[1:]
}

func displayText(value string) string {
	msg := redaction.RedactSecrets(strings.TrimSpace(value))
	msg = strings.NewReplacer("`", "'", "*", "'", "#", "＃").Replace(msg)
	return strings.Join(strings.Fields(msg), " ")
}
