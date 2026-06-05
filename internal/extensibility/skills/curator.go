package skills

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/curation"
)

const (
	DefaultCuratorIntervalHours = curation.DefaultCuratorIntervalHours
	DefaultCuratorMinIdleHours  = curation.DefaultCuratorMinIdleHours
	DefaultCuratorStaleDays     = curation.DefaultCuratorStaleDays
	DefaultCuratorArchiveDays   = curation.DefaultCuratorArchiveDays

	CuratorEvidenceDisabled         = curation.CuratorEvidenceDisabled
	CuratorEvidencePaused           = curation.CuratorEvidencePaused
	CuratorEvidenceFirstRunDeferred = curation.CuratorEvidenceFirstRunDeferred
	CuratorEvidenceIntervalPending  = curation.CuratorEvidenceIntervalPending
	CuratorEvidenceReady            = curation.CuratorEvidenceReady
)

type CuratorConfig = curation.CuratorConfig
type CuratorReviewer = curation.CuratorReviewer
type Curator = curation.Curator
type CuratorState = curation.CuratorState
type CuratorDecision = curation.CuratorDecision
type CuratorTransitionCounts = curation.CuratorTransitionCounts
type CuratorRunOptions = curation.CuratorRunOptions
type CuratorReviewInput = curation.CuratorReviewInput
type CuratorReviewResult = curation.CuratorReviewResult
type CuratorToolCall = curation.CuratorToolCall
type CuratorRunReport = curation.CuratorRunReport
type CuratorBackup = curation.CuratorBackup
type CuratorRollback = curation.CuratorRollback
type CuratorClassification = curation.CuratorClassification
type CuratorConsolidation = curation.CuratorConsolidation

func NewCurator(cfg CuratorConfig) *Curator { return curation.NewCurator(cfg) }

func ClassifyRemovedSkills(removed, afterNames []string, calls []CuratorToolCall) CuratorClassification {
	return curation.ClassifyRemovedSkills(removed, afterNames, calls)
}

func CreateCuratorBackup(root string, now time.Time, reason string, cronSkillRefs map[string][]string) (CuratorBackup, error) {
	return curation.CreateCuratorBackup(root, now, reason, cronSkillRefs)
}

func RollbackCuratorBackup(root, id string, now time.Time) (CuratorRollback, error) {
	return curation.RollbackCuratorBackup(root, id, now)
}

func buildCuratorRenameSummary(classification CuratorClassification) string {
	return curation.BuildCuratorRenameSummary(classification)
}

var _ CuratorReviewer = func(context.Context, CuratorReviewInput) (CuratorReviewResult, error) {
	return CuratorReviewResult{}, nil
}
