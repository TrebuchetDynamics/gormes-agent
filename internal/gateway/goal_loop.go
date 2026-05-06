package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

const (
	GoalJudgeVerdictDone     = "done"
	GoalJudgeVerdictContinue = "continue"
	GoalJudgeVerdictSkipped  = "skipped"
)

type GoalJudge interface {
	JudgeGoal(context.Context, string, string) (string, error)
}

type GoalJudgeFunc func(context.Context, string, string) (string, error)

func (f GoalJudgeFunc) JudgeGoal(ctx context.Context, goal, lastResponse string) (string, error) {
	return f(ctx, goal, lastResponse)
}

type GoalJudgeVerdict struct {
	Verdict string
	Reason  string
}

func EvaluateGoalJudge(ctx context.Context, judge GoalJudge, goal, lastResponse string) GoalJudgeVerdict {
	if strings.TrimSpace(goal) == "" {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictSkipped, Reason: "empty goal"}
	}
	if strings.TrimSpace(lastResponse) == "" {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictContinue, Reason: "empty response (nothing to evaluate)"}
	}
	if judge == nil {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictContinue, Reason: "auxiliary client unavailable"}
	}
	raw, err := judge.JudgeGoal(ctx, goal, lastResponse)
	if err != nil {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictContinue, Reason: "judge error: " + err.Error()}
	}
	return ParseGoalJudgeResponse(raw)
}

func ParseGoalJudgeResponse(raw string) GoalJudgeVerdict {
	if strings.TrimSpace(raw) == "" {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictContinue, Reason: "judge returned empty response"}
	}
	text := strings.TrimSpace(raw)
	if strings.HasPrefix(text, "```") {
		text = strings.Trim(text, "`")
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			text = text[i+1:]
		}
		text = strings.TrimSpace(text)
	}
	if !strings.HasPrefix(text, "{") {
		if start := strings.IndexByte(text, '{'); start >= 0 {
			if end := strings.LastIndexByte(text, '}'); end >= start {
				text = text[start : end+1]
			}
		}
	}

	var data struct {
		Done   any    `json:"done"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictContinue, Reason: fmt.Sprintf("judge reply was not JSON: %q", truncateGoalReason(raw, 200))}
	}
	reason := strings.TrimSpace(data.Reason)
	if reason == "" {
		reason = "no reason provided"
	}
	if goalJudgeDone(data.Done) {
		return GoalJudgeVerdict{Verdict: GoalJudgeVerdictDone, Reason: reason}
	}
	return GoalJudgeVerdict{Verdict: GoalJudgeVerdictContinue, Reason: reason}
}

func goalJudgeDone(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "yes", "1", "done":
			return true
		default:
			return false
		}
	case float64:
		return v != 0
	default:
		return false
	}
}

func truncateGoalReason(s string, limit int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func (m *Manager) handleGoalCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	store, storeOK := m.goalStore()
	if !storeOK {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_metadata_unavailable: Goals unavailable on this session.")
		return
	}

	args := goalCommandArgs(ev.Text)
	subcmd := strings.ToLower(strings.TrimSpace(args))
	switch subcmd {
	case "", "status":
		m.sendGoalStatus(ctx, ch, ev)
		return
	case "pause":
		m.pauseGoal(ctx, ch, ev)
		return
	case "resume":
		m.resumeGoal(ctx, ch, ev)
		return
	case "clear", "stop":
		m.clearGoal(ctx, ch, ev)
		return
	case "done":
		m.doneGoal(ctx, ch, ev)
		return
	}

	if m.hasActiveTurn() {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Agent is running — use /goal status / pause / clear mid-run, or /stop before setting a new goal.")
		return
	}
	if m.kernel == nil && m.cfg.AgentRuntimeFactory == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_submit_unavailable: Goals need an agent runtime to start the first turn.")
		return
	}
	sessionID, ok := m.ensureGoalSessionForSet(ctx, ev)
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_session_unavailable: Goal session metadata was unavailable.")
		return
	}
	goal, err := session.SetGoal(ctx, store, sessionID, args, m.goalMaxTurns(), m.now())
	if err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_metadata_unavailable: "+err.Error())
		return
	}

	kickoff := ev
	kickoff.Kind = EventSubmit
	kickoff.Text = args
	kickoff.Attachments = nil
	m.pinTurn(ev.Platform, ev.ChatID, ev.MsgID)
	if !m.submitPinned(ctx, ch, kickoff) {
		_, _ = session.PauseGoal(ctx, store, sessionID, "goal first turn queue failed", m.now())
		return
	}
	state, ok := m.activeTurnSnapshot()
	if !ok || strings.TrimSpace(state.SessionID) == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_session_unavailable: Goal first turn started but session metadata was unavailable.")
		return
	}
	if state.SessionID != sessionID {
		_, _ = session.ClearGoal(ctx, store, sessionID, m.now())
		goal, err = session.SetGoal(ctx, store, state.SessionID, args, m.goalMaxTurns(), m.now())
		if err != nil {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_metadata_unavailable: "+err.Error())
			return
		}
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf(
		"⊙ Goal set (%d-turn budget): %s\nI'll keep working until the goal is done, you pause/clear it, or the budget is exhausted.\nControls: /goal status · /goal pause · /goal resume · /goal clear",
		goal.MaxTurns,
		goal.Goal,
	))
}

func goalCommandArgs(text string) string {
	body := strings.TrimSpace(text)
	if body == "" {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return ""
	}
	first := strings.TrimPrefix(strings.ToLower(fields[0]), "/")
	if first != "goal" {
		return body
	}
	if len(body) <= len(fields[0]) {
		return ""
	}
	return strings.TrimSpace(body[len(fields[0]):])
}

func (m *Manager) sendGoalStatus(ctx context.Context, ch Channel, ev InboundEvent) {
	state, ok := m.loadGoalForInbound(ctx, ev)
	if !ok || state == nil || state.Status == session.GoalStatusCleared || strings.TrimSpace(state.Goal) == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, formatGoalStatusLine(state))
}

func formatGoalStatusLine(state *session.GoalState) string {
	if state == nil {
		return "No goal set."
	}
	status := strings.ToLower(strings.TrimSpace(string(state.Status)))
	if status == "" {
		status = string(session.GoalStatusActive)
	}
	line := fmt.Sprintf("Goal %s (%d/%d): %s", status, state.TurnsUsed, state.MaxTurns, state.Goal)
	if reason := strings.TrimSpace(state.LastReason); reason != "" {
		line += "\nLast verdict: " + strings.TrimSpace(state.LastVerdict) + " — " + reason
	}
	if reason := strings.TrimSpace(state.PausedReason); reason != "" {
		line += "\nPaused reason: " + reason
	}
	return line
}

func (m *Manager) pauseGoal(ctx context.Context, ch Channel, ev InboundEvent) {
	sessionID, ok := m.goalSessionIDForInbound(ctx, ev)
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	store, ok := m.goalStore()
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	state, err := session.PauseGoal(ctx, store, sessionID, "user-paused", m.now())
	if err != nil || state == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "⏸ Goal paused: "+state.Goal)
}

func (m *Manager) resumeGoal(ctx context.Context, ch Channel, ev InboundEvent) {
	sessionID, ok := m.goalSessionIDForInbound(ctx, ev)
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal to resume.")
		return
	}
	store, ok := m.goalStore()
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal to resume.")
		return
	}
	state, err := session.ResumeGoal(ctx, store, sessionID, true, m.now())
	if err != nil || state == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal to resume.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "▶ Goal resumed: "+state.Goal+"\nSend any message to continue, or wait — I'll take the next step on the next turn.")
}

func (m *Manager) clearGoal(ctx context.Context, ch Channel, ev InboundEvent) {
	sessionID, ok := m.goalSessionIDForInbound(ctx, ev)
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No active goal.")
		return
	}
	store, ok := m.goalStore()
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No active goal.")
		return
	}
	state, err := session.ClearGoal(ctx, store, sessionID, m.now())
	if err != nil || state == nil || state.Status == session.GoalStatusCleared && strings.TrimSpace(state.Goal) == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No active goal.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "✓ Goal cleared.")
}

func (m *Manager) doneGoal(ctx context.Context, ch Channel, ev InboundEvent) {
	sessionID, ok := m.goalSessionIDForInbound(ctx, ev)
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	store, ok := m.goalStore()
	if !ok {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	state, err := session.DoneGoal(ctx, store, sessionID, "marked done by user", m.now())
	if err != nil || state == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "✓ Goal achieved: marked done by user")
}

func (m *Manager) loadGoalForInbound(ctx context.Context, ev InboundEvent) (*session.GoalState, bool) {
	sessionID, ok := m.goalSessionIDForInbound(ctx, ev)
	if !ok {
		return nil, false
	}
	store, ok := m.goalStore()
	if !ok {
		return nil, false
	}
	state, ok, err := session.LoadGoal(ctx, store, sessionID)
	if err != nil {
		m.log.Warn("load goal state", "session_id", sessionID, "err", err)
		return nil, false
	}
	return state, ok
}

func (m *Manager) goalSessionIDForInbound(ctx context.Context, ev InboundEvent) (string, bool) {
	if state, ok := m.activeTurnSnapshot(); ok &&
		state.Platform == ev.Platform &&
		state.ChatID == ev.ChatID &&
		strings.TrimSpace(state.SessionID) != "" {
		return state.SessionID, true
	}
	resolved, err := resolveSession(ctx, m.cfg.SessionMap, m.sessionKeyForInbound(ev))
	if err != nil {
		m.log.Warn("resolve goal session", "platform", ev.Platform, "chat_id", ev.ChatID, "err", err)
	}
	sessionID := strings.TrimSpace(resolved.SessionID)
	return sessionID, sessionID != ""
}

func (m *Manager) ensureGoalSessionForSet(ctx context.Context, ev InboundEvent) (string, bool) {
	route := m.agentRouteForInbound(ev)
	sessionKey := strings.TrimSpace(route.SessionKey)
	if sessionKey == "" {
		sessionKey = ev.ChatKey()
	}
	resolved, err := resolveSession(ctx, m.cfg.SessionMap, sessionKey)
	if err != nil {
		m.log.Warn("resolve goal session for set", "key", sessionKey, "err", err)
	}
	resolved = m.refreshConversationalSessionMetadata(ctx, ev, sessionKey, resolved, sessionSourceFromInbound(ev))
	sessionID := strings.TrimSpace(resolved.SessionID)
	return sessionID, sessionID != ""
}

func (m *Manager) goalMaxTurns() int {
	if m.cfg.GoalMaxTurns > 0 {
		return m.cfg.GoalMaxTurns
	}
	return session.DefaultGoalMaxTurns
}

func (m *Manager) goalStore() (session.GoalMetadataStore, bool) {
	if m.cfg.SessionMap == nil {
		return nil, false
	}
	store, ok := m.cfg.SessionMap.(session.GoalMetadataStore)
	return store, ok
}

func (m *Manager) handleGoalPostTurnContinuation(ctx context.Context, ch Channel, f kernel.RenderFrame) {
	state, ok := m.activeTurnSnapshot()
	store, storeOK := m.goalStore()
	if !ok || state.Cancelled || strings.TrimSpace(state.SessionID) == "" || !storeOK {
		return
	}
	if m.hasQueuedFollowUp() {
		return
	}
	goal, ok, err := session.LoadGoal(ctx, store, state.SessionID)
	if err != nil {
		m.log.Warn("load goal for post-turn continuation", "session_id", state.SessionID, "err", err)
		return
	}
	if !ok || !session.GoalIsActive(goal) {
		return
	}

	goal.TurnsUsed++
	goal.LastTurnAt = m.now().Unix()
	verdict := EvaluateGoalJudge(ctx, m.cfg.GoalJudge, goal.Goal, FinalAssistantText(f))
	goal.LastVerdict = verdict.Verdict
	goal.LastReason = verdict.Reason

	switch verdict.Verdict {
	case GoalJudgeVerdictDone:
		goal.Status = session.GoalStatusDone
		if err := session.SaveGoal(ctx, store, state.SessionID, goal, m.now()); err != nil {
			m.log.Warn("save done goal", "session_id", state.SessionID, "err", err)
			return
		}
		_, _ = m.sendWithHooks(ctx, ch, state.ChatID, "✓ Goal achieved: "+verdict.Reason)
		return
	}

	if goal.TurnsUsed >= goal.MaxTurns {
		goal.Status = session.GoalStatusPaused
		goal.PausedReason = fmt.Sprintf("turn budget exhausted (%d/%d)", goal.TurnsUsed, goal.MaxTurns)
		if err := session.SaveGoal(ctx, store, state.SessionID, goal, m.now()); err != nil {
			m.log.Warn("save paused goal", "session_id", state.SessionID, "err", err)
			return
		}
		_, _ = m.sendWithHooks(ctx, ch, state.ChatID, fmt.Sprintf("⏸ Goal paused — %d/%d turns used. Use /goal resume to keep going, or /goal clear to stop.", goal.TurnsUsed, goal.MaxTurns))
		return
	}

	if err := session.SaveGoal(ctx, store, state.SessionID, goal, m.now()); err != nil {
		m.log.Warn("save continuing goal", "session_id", state.SessionID, "err", err)
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, state.ChatID, fmt.Sprintf("↻ Continuing toward goal (%d/%d): %s", goal.TurnsUsed, goal.MaxTurns, verdict.Reason))
	m.queueGoalContinuation(ctx, ch, state, session.ContinuationPrompt(goal.Goal))
}

func (m *Manager) hasQueuedFollowUp() bool {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	return len(m.followUps) > 0
}

func (m *Manager) queueGoalContinuation(ctx context.Context, ch Channel, state activeTurnSnapshot, prompt string) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return
	}
	ev := InboundEvent{
		Platform: state.Platform,
		ChatID:   state.ChatID,
		UserID:   state.Source.UserID,
		MsgID:    "goal-continuation-" + m.now().UTC().Format("20060102150405.000000000"),
		Kind:     EventSubmit,
		Text:     prompt,
	}
	queued, full := m.queueFollowUpIfActive(ev)
	if full {
		_, _ = m.sendWithHooks(ctx, ch, state.ChatID, "goal_queue_full: continuation not queued because follow-up queue is full.")
	}
	if !queued && !full {
		_, _ = m.sendWithHooks(ctx, ch, state.ChatID, "goal_queue_unavailable: continuation not queued because no active turn was available.")
	}
}
