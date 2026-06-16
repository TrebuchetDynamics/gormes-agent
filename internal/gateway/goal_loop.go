package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/goalloop"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

const (
	GoalJudgeVerdictDone     = goalloop.JudgeVerdictDone
	GoalJudgeVerdictContinue = goalloop.JudgeVerdictContinue
	GoalJudgeVerdictSkipped  = goalloop.JudgeVerdictSkipped
)

type GoalJudge = goalloop.Judge

type GoalJudgeFunc = goalloop.JudgeFunc

type GoalJudgeVerdict = goalloop.JudgeVerdict

func EvaluateGoalJudge(ctx context.Context, judge GoalJudge, goal, lastResponse string) GoalJudgeVerdict {
	return goalloop.EvaluateJudge(ctx, judge, goal, lastResponse)
}

func ParseGoalJudgeResponse(raw string) GoalJudgeVerdict {
	return goalloop.ParseJudgeResponse(raw)
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
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_metadata_unavailable: "+goalCommandErrorText(err))
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
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "goal_metadata_unavailable: "+goalCommandErrorText(err))
			return
		}
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf(
		"⊙ Goal set (%d-turn budget): %s\nI'll keep working until the goal is done, you pause/clear it, or the budget is exhausted.\nControls: /goal status · /goal pause · /goal resume · /goal clear",
		goal.MaxTurns,
		goalDisplayText(goal.Goal),
	))
}

func goalCommandArgs(text string) string { return goalloop.CommandArgs(text) }

func (m *Manager) sendGoalStatus(ctx context.Context, ch Channel, ev InboundEvent) {
	state, ok := m.loadGoalForInbound(ctx, ev)
	if !ok || state == nil || state.Status == session.GoalStatusCleared || strings.TrimSpace(state.Goal) == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No goal set.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, formatGoalStatusLine(state))
}

func formatGoalStatusLine(state *session.GoalState) string { return goalloop.FormatStatusLine(state) }

func goalCommandErrorText(err error) string {
	if err == nil {
		return ""
	}
	return goalDisplayText(err.Error())
}

func goalDisplayText(value string) string {
	msg := strings.TrimSpace(value)
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactGoalSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func compactGoalSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
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
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "⏸ Goal paused: "+goalDisplayText(state.Goal))
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
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "▶ Goal resumed: "+goalDisplayText(state.Goal)+"\nSend any message to continue, or wait — I'll take the next step on the next turn.")
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
	if !ok || strings.TrimSpace(state.SessionID) == "" || !storeOK {
		return
	}
	if state.Cancelled {
		m.pauseInterruptedGoal(ctx, ch, state)
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
		_, _ = m.sendWithHooks(ctx, ch, state.ChatID, "✓ Goal achieved: "+goalDisplayText(verdict.Reason))
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
	_, _ = m.sendWithHooks(ctx, ch, state.ChatID, fmt.Sprintf("↻ Continuing toward goal (%d/%d): %s", goal.TurnsUsed, goal.MaxTurns, goalDisplayText(verdict.Reason)))
	m.queueGoalContinuation(ctx, ch, state, session.ContinuationPrompt(goal.Goal, goal.Subgoals))
}

func (m *Manager) pauseInterruptedGoal(ctx context.Context, ch Channel, state activeTurnSnapshot) {
	if !state.Cancelled || strings.TrimSpace(state.SessionID) == "" {
		return
	}
	store, ok := m.goalStore()
	if !ok {
		return
	}
	goal, ok, err := session.LoadGoal(ctx, store, state.SessionID)
	if err != nil {
		m.log.Warn("load goal for interrupted turn", "session_id", state.SessionID, "err", err)
		return
	}
	if !ok || !session.GoalIsActive(goal) {
		return
	}
	paused, err := session.PauseGoal(ctx, store, state.SessionID, "interrupted by operator", m.now())
	if err != nil {
		m.log.Warn("pause interrupted goal", "session_id", state.SessionID, "err", err)
		return
	}
	if paused == nil {
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, state.ChatID, "⏸ Goal paused — interrupted. Use /goal resume to keep going, or /goal clear to stop.")
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
