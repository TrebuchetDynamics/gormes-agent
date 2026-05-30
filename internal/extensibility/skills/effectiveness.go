package skills

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/analytics"
)

type SkillOutcome = analytics.SkillOutcome

const (
	SkillOutcomePositive SkillOutcome = analytics.SkillOutcomePositive
	SkillOutcomeNeutral  SkillOutcome = analytics.SkillOutcomeNeutral
	SkillOutcomeNegative SkillOutcome = analytics.SkillOutcomeNegative
)

const (
	SkillEffectivenessReasonPositive         = analytics.SkillEffectivenessReasonPositive
	SkillEffectivenessReasonNeutral          = analytics.SkillEffectivenessReasonNeutral
	SkillEffectivenessReasonNegative         = analytics.SkillEffectivenessReasonNegative
	SkillEffectivenessReasonOperatorFeedback = analytics.SkillEffectivenessReasonOperatorFeedback
	SkillEffectivenessReasonStaleDecay       = analytics.SkillEffectivenessReasonStaleDecay
)

type SkillEffectivenessEvent = analytics.SkillEffectivenessEvent

type SkillEffectivenessRecord = analytics.SkillEffectivenessRecord

type SkillEffectivenessLedger = analytics.SkillEffectivenessLedger

type SkillEffectivenessLoad = analytics.SkillEffectivenessLoad

type SkillEffectivenessInvalidRecord = analytics.SkillEffectivenessInvalidRecord

type SkillEffectivenessScoreOptions = analytics.SkillEffectivenessScoreOptions

type SkillEffectivenessScore = analytics.SkillEffectivenessScore

func SkillEffectivenessLedgerPath(root string) string {
	return analytics.SkillEffectivenessLedgerPath(root)
}

func NewSkillEffectivenessLedger(path string, now func() time.Time) *SkillEffectivenessLedger {
	return analytics.NewSkillEffectivenessLedger(path, now)
}

func ScoreSkillEffectiveness(records []SkillEffectivenessRecord, opts SkillEffectivenessScoreOptions) []SkillEffectivenessScore {
	return analytics.ScoreSkillEffectiveness(records, opts)
}
