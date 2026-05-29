package skills

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type SkillOutcome string

const (
	SkillOutcomePositive SkillOutcome = "positive"
	SkillOutcomeNeutral  SkillOutcome = "neutral"
	SkillOutcomeNegative SkillOutcome = "negative"
)

const (
	SkillEffectivenessReasonPositive         = "positive_outcome"
	SkillEffectivenessReasonNeutral          = "neutral_outcome"
	SkillEffectivenessReasonNegative         = "negative_outcome"
	SkillEffectivenessReasonOperatorFeedback = "operator_feedback"
	SkillEffectivenessReasonStaleDecay       = "stale_decay"
)

// SkillEffectivenessEvent is the caller-facing evidence for one selected skill
// outcome. Raw prompt and feedback text are accepted only long enough to derive
// redacted counts and hashes; they are never stored in the JSONL ledger.
type SkillEffectivenessEvent struct {
	SkillName string
	SessionID string
	TurnID    string

	Prompt string

	SelectionSource string
	LexicalScore    int
	SemanticScore   float64
	TotalScore      float64

	Outcome SkillOutcome

	OperatorFeedback string
	FeedbackReason   string

	RecordedAt time.Time
}

// SkillEffectivenessRecord is the stable append-only JSONL schema for skill
// outcome evidence. It intentionally excludes raw prompts, tool output, and raw
// operator feedback so reports can be logged and replayed safely.
type SkillEffectivenessRecord struct {
	SkillName  string    `json:"skill_name"`
	SessionID  string    `json:"session_id,omitempty"`
	TurnID     string    `json:"turn_id,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`

	PromptSHA256       string `json:"prompt_sha256,omitempty"`
	PromptBytes        int    `json:"prompt_bytes,omitempty"`
	RedactedInputCount int    `json:"redacted_input_count,omitempty"`

	SelectionSource string  `json:"selection_source,omitempty"`
	LexicalScore    int     `json:"lexical_score,omitempty"`
	SemanticScore   float64 `json:"semantic_score,omitempty"`
	TotalScore      float64 `json:"total_score,omitempty"`

	Outcome SkillOutcome `json:"outcome"`

	OperatorFeedbackCount int    `json:"operator_feedback_count,omitempty"`
	FeedbackReason        string `json:"feedback_reason,omitempty"`
}

type SkillEffectivenessLedger struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type SkillEffectivenessLoad struct {
	Records []SkillEffectivenessRecord
	Invalid []SkillEffectivenessInvalidRecord
}

type SkillEffectivenessInvalidRecord struct {
	Line  int    `json:"line"`
	Error string `json:"error"`
}

type SkillEffectivenessScoreOptions struct {
	Now        time.Time
	StaleAfter time.Duration
}

type SkillEffectivenessScore struct {
	SkillName string `json:"skill_name"`

	Score float64 `json:"score"`

	PositiveOutcomes int `json:"positive_outcomes"`
	NeutralOutcomes  int `json:"neutral_outcomes"`
	NegativeOutcomes int `json:"negative_outcomes"`

	OperatorFeedbackCount int       `json:"operator_feedback_count"`
	LastOutcomeAt         time.Time `json:"last_outcome_at,omitempty"`
	ReasonCodes           []string  `json:"reason_codes,omitempty"`
}

func SkillEffectivenessLedgerPath(root string) string {
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".skill_effectiveness.jsonl")
}

func NewSkillEffectivenessLedger(path string, now func() time.Time) *SkillEffectivenessLedger {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &SkillEffectivenessLedger{path: path, now: now}
}

func (l *SkillEffectivenessLedger) Record(ctx context.Context, event SkillEffectivenessEvent) error {
	if l == nil || l.path == "" {
		return nil
	}
	if strings.TrimSpace(event.SkillName) == "" {
		return errors.New("skills: effectiveness event missing skill name")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	record := event.toRecord(l.now)
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(line)
	return err
}

func (l *SkillEffectivenessLedger) Load(ctx context.Context) (SkillEffectivenessLoad, error) {
	var out SkillEffectivenessLoad
	if l == nil || l.path == "" {
		return out, nil
	}
	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record SkillEffectivenessRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			out.Invalid = append(out.Invalid, SkillEffectivenessInvalidRecord{Line: lineNo, Error: err.Error()})
			continue
		}
		if strings.TrimSpace(record.SkillName) == "" {
			out.Invalid = append(out.Invalid, SkillEffectivenessInvalidRecord{Line: lineNo, Error: "missing skill_name"})
			continue
		}
		out.Records = append(out.Records, record)
	}
	if err := scanner.Err(); err != nil {
		return out, err
	}
	return out, nil
}

func ScoreSkillEffectiveness(records []SkillEffectivenessRecord, opts SkillEffectivenessScoreOptions) []SkillEffectivenessScore {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bySkill := map[string]*SkillEffectivenessScore{}
	for _, record := range records {
		name := strings.TrimSpace(record.SkillName)
		if name == "" {
			continue
		}
		score := bySkill[name]
		if score == nil {
			score = &SkillEffectivenessScore{SkillName: name}
			bySkill[name] = score
		}
		weight := effectivenessWeight(record.RecordedAt, now, opts.StaleAfter)
		if weight < 1 {
			appendEffectivenessReason(score, SkillEffectivenessReasonStaleDecay)
		}
		switch record.Outcome {
		case SkillOutcomePositive:
			score.PositiveOutcomes++
			score.Score += 100 * weight
			appendEffectivenessReason(score, SkillEffectivenessReasonPositive)
		case SkillOutcomeNegative:
			score.NegativeOutcomes++
			score.Score -= 100 * weight
			appendEffectivenessReason(score, SkillEffectivenessReasonNegative)
		default:
			score.NeutralOutcomes++
			appendEffectivenessReason(score, SkillEffectivenessReasonNeutral)
		}
		feedbackCount := record.OperatorFeedbackCount
		if feedbackCount > 0 {
			score.OperatorFeedbackCount += feedbackCount
			feedbackWeight := 25 * float64(feedbackCount) * weight
			switch record.Outcome {
			case SkillOutcomePositive:
				score.Score += feedbackWeight
			case SkillOutcomeNegative:
				score.Score -= feedbackWeight
			}
			appendEffectivenessReason(score, SkillEffectivenessReasonOperatorFeedback)
		}
		if record.RecordedAt.After(score.LastOutcomeAt) {
			score.LastOutcomeAt = record.RecordedAt
		}
	}

	out := make([]SkillEffectivenessScore, 0, len(bySkill))
	for _, score := range bySkill {
		out = append(out, *score)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].SkillName < out[j].SkillName
	})
	return out
}

func (e SkillEffectivenessEvent) toRecord(now func() time.Time) SkillEffectivenessRecord {
	recordedAt := e.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = now()
	}
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}
	out := SkillEffectivenessRecord{
		SkillName:       strings.TrimSpace(e.SkillName),
		SessionID:       strings.TrimSpace(e.SessionID),
		TurnID:          strings.TrimSpace(e.TurnID),
		RecordedAt:      recordedAt.UTC(),
		SelectionSource: strings.TrimSpace(e.SelectionSource),
		LexicalScore:    e.LexicalScore,
		SemanticScore:   e.SemanticScore,
		TotalScore:      e.TotalScore,
		Outcome:         normalizeSkillOutcome(e.Outcome),
		FeedbackReason:  strings.TrimSpace(e.FeedbackReason),
	}
	if e.Prompt != "" {
		sum := sha256.Sum256([]byte(e.Prompt))
		out.PromptSHA256 = hex.EncodeToString(sum[:])
		out.PromptBytes = len([]byte(e.Prompt))
		if effectivenessLooksSensitive(e.Prompt) {
			out.RedactedInputCount++
		}
	}
	if strings.TrimSpace(e.OperatorFeedback) != "" {
		out.OperatorFeedbackCount = 1
		if effectivenessLooksSensitive(e.OperatorFeedback) {
			out.RedactedInputCount++
		}
	}
	return out
}

func normalizeSkillOutcome(outcome SkillOutcome) SkillOutcome {
	switch outcome {
	case SkillOutcomePositive, SkillOutcomeNegative, SkillOutcomeNeutral:
		return outcome
	default:
		return SkillOutcomeNeutral
	}
}

func effectivenessWeight(recordedAt, now time.Time, staleAfter time.Duration) float64 {
	if recordedAt.IsZero() || staleAfter <= 0 || now.IsZero() {
		return 1
	}
	age := now.Sub(recordedAt)
	if age <= staleAfter {
		return 1
	}
	periods := int(age / staleAfter)
	if periods < 1 {
		periods = 1
	}
	weight := 1.0
	for i := 0; i < periods; i++ {
		weight *= 0.5
	}
	return weight
}

func appendEffectivenessReason(score *SkillEffectivenessScore, reason string) {
	for _, existing := range score.ReasonCodes {
		if existing == reason {
			return
		}
	}
	score.ReasonCodes = append(score.ReasonCodes, reason)
}

func effectivenessLooksSensitive(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"api key", "token", "secret", "sk-", "ghp_", "password"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func (r SkillEffectivenessInvalidRecord) String() string {
	return fmt.Sprintf("line %d: %s", r.Line, r.Error)
}
