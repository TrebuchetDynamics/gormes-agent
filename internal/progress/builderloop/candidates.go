package builderloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/workitem"
)

type CandidateOptions struct {
	ActiveFirst     bool
	PriorityBoost   []string
	MaxPhase        int
	IncludeBlocked  bool
	IncludeUmbrella bool
	IncludePaused   bool
	// IncludeQuarantined causes NormalizeCandidates to surface rows whose
	// Health.Quarantine block is current (spec hash matches). Default false:
	// quarantined rows are filtered out so the run loop avoids known-bad
	// targets. Stale quarantines (spec hash mismatch) are always surfaced and
	// flagged with Candidate.StaleQuarantine regardless of this setting.
	IncludeQuarantined bool
	// IncludeNeedsHuman causes NormalizeCandidates to surface rows whose
	// PlannerVerdict.NeedsHuman is true. Default false: such rows are
	// filtered out so the autoloop honors the planner's escalation. Mirrors
	// IncludeQuarantined exactly. Surfaced rows are flagged with
	// Candidate.NeedsHumanFlag for downstream visibility.
	IncludeNeedsHuman bool
}

type Candidate struct {
	PhaseID        string
	SubphaseID     string
	ItemName       string
	Status         string
	Priority       string
	Contract       string
	ContractStatus string
	SliceSize      string
	ExecutionOwner string
	TrustClass     []string
	DegradedMode   string
	Fixture        string
	SourceRefs     []string
	BlockedBy      []string
	Unblocks       []string
	ReadyWhen      []string
	NotReadyWhen   []string
	Acceptance     []string
	WriteScope     []string
	TestCommands   []string
	NoTestRequired string
	DoneSignal     []string
	Note           string
	// Blocker is the fleet-standard blocker metadata attached to this row.
	// Default selection skips rows with this metadata so the builder pivots
	// instead of repeatedly claiming known-blocked work.
	Blocker *progress.BlockerMetadata
	// Health is the row's autoloop execution-history block, if any. Surfaced
	// here so the run loop and reporting can consult quarantine / failure
	// counts without re-loading progress.json.
	Health *progress.RowHealth
	// StaleQuarantine is set by Task 5's selection logic when the row's
	// existing Quarantine.SpecHash no longer matches the current ItemSpecHash
	// (planner reshape detected). The run loop forwards this to the health
	// accumulator so Flush clears the stale block atomically with run health.
	StaleQuarantine bool
	// PenaltyApplied is the ranking penalty derived from Health
	// (ConsecutiveFailures + 2*len(BackendsTried)). Recorded so the reason
	// string and downstream tooling can surface why a row sank in priority.
	PenaltyApplied int
	// NeedsHumanFlag is set when the row's PlannerVerdict.NeedsHuman is true
	// AND the candidate was surfaced anyway via IncludeNeedsHuman. Allows
	// reporting / status tooling to highlight the override without re-loading
	// the verdict block. Always false in the default skip-NeedsHuman path.
	NeedsHumanFlag bool
	// Speculative indicates this candidate has blocked_by dependencies that
	// are not yet complete, but ready_when is satisfied. The builder-loop
	// will start work on it speculatively, verifying before promotion that
	// (1) the spec hasn't changed, and (2) all blocked_by completed successfully.
	Speculative bool
	// SpecHashAtClaim is the ItemSpecHash snapshot at the time of claim.
	// Used to detect spec changes during speculative execution.
	SpecHashAtClaim string
	// BlockedByPending lists the blocked_by dependencies not yet complete.
	// Only populated when Speculative is true.
	BlockedByPending []string
}

// failurePenalty returns the ranking penalty for n consecutive failures.
// 0 -> 0, 1 -> 5, 2 -> 20, 3+ -> 45 (capped). Rows past the quarantine
// threshold should already be filtered by NormalizeCandidates, but the cap
// covers manual-override scenarios where IncludeQuarantined is set.
func failurePenalty(n int) int {
	switch {
	case n <= 0:
		return 0
	case n == 1:
		return 5
	case n == 2:
		return 20
	default:
		return 45
	}
}

func (candidate Candidate) SelectionReason() string {
	var base string
	switch candidateBucket(candidate) {
	case candidateBucketP0:
		base = "P0 handoff"
	case candidateBucketInProgress:
		base = "already active"
	case candidateBucketFixtureReady:
		base = "fixture ready"
	case candidateBucketUnblocks:
		base = "unblocks downstream work"
	case candidateBucketDraft:
		base = "draft contract"
	default:
		base = "planned row"
	}
	if candidate.PenaltyApplied > 0 {
		base += fmt.Sprintf(" penalty=%d", candidate.PenaltyApplied)
	}
	if candidate.StaleQuarantine {
		base += " quarantine_stale_cleared"
	}
	if candidate.NeedsHumanFlag {
		base += " needs_human_visible"
	}
	if candidate.Speculative {
		base += " speculative"
	}
	return base
}

func NormalizeCandidates(path string, opts CandidateOptions) ([]Candidate, error) {
	data, err := readProgressCandidateBytes(path)
	if err != nil {
		return nil, err
	}

	var progressDoc progressJSON
	if err := json.Unmarshal(data, &progressDoc); err != nil {
		return nil, err
	}

	sources := candidateSources(progressDoc)
	inputs := make([]workitem.RowInput, 0, len(sources))
	byKey := make(map[string]candidateSource, len(sources))
	for _, source := range sources {
		inputs = append(inputs, source.Input)
		byKey[workitemKey(source.Input.PhaseID, source.Input.SubphaseID, source.Input.ItemName)] = source
	}

	assignable := workitem.Assignable(inputs, workitem.Options{
		ActiveFirst:        opts.ActiveFirst,
		PriorityBoost:      opts.PriorityBoost,
		MaxPhase:           opts.MaxPhase,
		IncludeBlocked:     opts.IncludeBlocked,
		IncludeUmbrella:    opts.IncludeUmbrella,
		IncludePaused:      opts.IncludePaused,
		IncludeQuarantined: opts.IncludeQuarantined,
		IncludeNeedsHuman:  opts.IncludeNeedsHuman,
	})

	candidates := make([]Candidate, 0, len(assignable))
	for _, row := range assignable {
		source, ok := byKey[workitemKey(row.PhaseID, row.SubphaseID, row.ItemName)]
		if !ok {
			continue
		}
		candidates = append(candidates, candidateFromWorkitem(row, source.Item))
	}
	return candidates, nil
}

type candidateSource struct {
	Input workitem.RowInput
	Item  progressItem
}

func candidateSources(progressDoc progressJSON) []candidateSource {
	var sources []candidateSource
	for _, phase := range progressDoc.Phases {
		for _, subphase := range phase.allSubphases() {
			for _, item := range subphase.Items {
				name := firstNonEmpty(item.ItemName, item.Name, item.Title, item.ID)
				if name == "" {
					continue
				}
				input := workitem.RowInput{
					PhaseID:        phase.ID,
					SubphaseID:     subphase.ID,
					ItemName:       name,
					Status:         item.Status,
					Priority:       firstNonEmpty(item.Priority, subphase.Priority),
					Contract:       item.Contract,
					ContractStatus: item.ContractStatus,
					SliceSize:      item.SliceSize,
					ExecutionOwner: item.ExecutionOwner,
					TrustClass:     item.TrustClass,
					DegradedMode:   item.DegradedMode,
					Fixture:        item.Fixture,
					SourceRefs:     item.SourceRefs,
					BlockedBy:      item.BlockedBy,
					Unblocks:       item.Unblocks,
					ReadyWhen:      item.ReadyWhen,
					NotReadyWhen:   item.NotReadyWhen,
					Acceptance:     item.Acceptance,
					WriteScope:     item.WriteScope,
					TestCommands:   item.TestCommands,
					NoTestRequired: item.NoTestRequired,
					DoneSignal:     item.DoneSignal,
					Note:           item.Note,
					HasBlocker:     item.Blocker != nil,
				}
				if item.PlannerVerdict != nil && item.PlannerVerdict.NeedsHuman {
					input.NeedsHuman = true
				}
				if item.Health != nil {
					input.PenaltyApplied = failurePenalty(item.Health.ConsecutiveFailures) + 2*len(item.Health.BackendsTried)
					if item.Health.Quarantine != nil {
						input.Quarantined = true
						currentHash := progress.ItemSpecHash(itemPtr(item))
						input.StaleQuarantine = currentHash != item.Health.Quarantine.SpecHash
					}
				}
				sources = append(sources, candidateSource{Input: input, Item: item})
			}
		}
	}
	return sources
}

func candidateFromWorkitem(row workitem.Row, item progressItem) Candidate {
	return Candidate{
		PhaseID:         row.PhaseID,
		SubphaseID:      row.SubphaseID,
		ItemName:        row.ItemName,
		Status:          row.Status,
		Priority:        row.Priority,
		Contract:        row.Contract,
		ContractStatus:  row.ContractStatus,
		SliceSize:       row.SliceSize,
		ExecutionOwner:  row.ExecutionOwner,
		TrustClass:      append([]string(nil), row.TrustClass...),
		DegradedMode:    row.DegradedMode,
		Fixture:         row.Fixture,
		SourceRefs:      append([]string(nil), row.SourceRefs...),
		BlockedBy:       append([]string(nil), row.BlockedBy...),
		Unblocks:        append([]string(nil), row.Unblocks...),
		ReadyWhen:       append([]string(nil), row.ReadyWhen...),
		NotReadyWhen:    append([]string(nil), row.NotReadyWhen...),
		Acceptance:      append([]string(nil), row.Acceptance...),
		WriteScope:      append([]string(nil), row.WriteScope...),
		TestCommands:    append([]string(nil), row.TestCommands...),
		NoTestRequired:  row.NoTestRequired,
		DoneSignal:      append([]string(nil), row.DoneSignal...),
		Note:            row.Note,
		Blocker:         cloneBlockerMetadata(item.Blocker),
		Health:          item.Health,
		StaleQuarantine: row.StaleQuarantine,
		PenaltyApplied:  row.PenaltyApplied,
		NeedsHumanFlag:  row.NeedsHumanVisible,
	}
}

func workitemKey(phaseID, subphaseID, itemName string) string {
	return strings.TrimSpace(phaseID) + "\x00" + strings.TrimSpace(subphaseID) + "\x00" + strings.TrimSpace(itemName)
}

func readProgressCandidateBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if fi, statErr := os.Stat(path); statErr != nil || !fi.IsDir() {
		return nil, err
	}
	p, loadErr := progress.Load(path)
	if loadErr != nil {
		return nil, loadErr
	}
	tmpDir, mkErr := os.MkdirTemp("", "builderloop-progress-")
	if mkErr != nil {
		return nil, mkErr
	}
	defer os.RemoveAll(tmpDir)
	tmp := filepath.Join(tmpDir, "progress.json")
	if saveErr := progress.SaveProgress(tmp, p); saveErr != nil {
		return nil, saveErr
	}
	return os.ReadFile(tmp)
}

// itemPtr returns a pointer to a progress.Item view of the given progressItem
// suitable for passing to progress.ItemSpecHash. Lifted to a helper so the
// conversion happens in one place.
func itemPtr(item progressItem) *progress.Item {
	view := item.toProgressItem()
	return &view
}

func phaseNumber(phaseID string) (int, bool) {
	phaseNum, err := strconv.Atoi(strings.TrimSpace(phaseID))
	if err != nil {
		return 0, false
	}

	return phaseNum, true
}

func firstNonEmpty(vals ...string) string {
	for _, val := range vals {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

type progressJSON struct {
	Phases progressPhases `json:"phases"`
}

type progressPhases []progressPhase

func (phases *progressPhases) UnmarshalJSON(data []byte) error {
	var keyed map[string]progressPhase
	if err := json.Unmarshal(data, &keyed); err == nil {
		*phases = make([]progressPhase, 0, len(keyed))
		for id, phase := range keyed {
			phase.ID = firstNonEmpty(id, phase.ID)
			*phases = append(*phases, phase)
		}

		return nil
	}

	var listed []progressPhase
	if err := json.Unmarshal(data, &listed); err != nil {
		return err
	}
	*phases = listed

	return nil
}

type progressPhase struct {
	ID        string            `json:"id"`
	Subphases progressSubphases `json:"subphases"`
	SubPhases progressSubphases `json:"sub_phases"`
}

func (phase progressPhase) allSubphases() []progressSubphase {
	if len(phase.Subphases) > 0 {
		return phase.Subphases
	}

	return phase.SubPhases
}

type progressSubphases []progressSubphase

func (subphases *progressSubphases) UnmarshalJSON(data []byte) error {
	var keyed map[string]progressSubphase
	if err := json.Unmarshal(data, &keyed); err == nil {
		*subphases = make([]progressSubphase, 0, len(keyed))
		for id, subphase := range keyed {
			subphase.ID = firstNonEmpty(id, subphase.ID)
			*subphases = append(*subphases, subphase)
		}

		return nil
	}

	var listed []progressSubphase
	if err := json.Unmarshal(data, &listed); err != nil {
		return err
	}
	*subphases = listed

	return nil
}

type progressSubphase struct {
	ID       string         `json:"id"`
	Priority string         `json:"priority"`
	Items    []progressItem `json:"items"`
}

type progressItem struct {
	ItemName       string                    `json:"item_name"`
	Name           string                    `json:"name"`
	Title          string                    `json:"title"`
	ID             string                    `json:"id"`
	Status         string                    `json:"status"`
	Priority       string                    `json:"priority"`
	Contract       string                    `json:"contract"`
	ContractStatus string                    `json:"contract_status"`
	SliceSize      string                    `json:"slice_size"`
	ExecutionOwner string                    `json:"execution_owner"`
	TrustClass     []string                  `json:"trust_class"`
	DegradedMode   string                    `json:"degraded_mode"`
	Fixture        string                    `json:"fixture"`
	SourceRefs     []string                  `json:"source_refs"`
	BlockedBy      []string                  `json:"blocked_by"`
	Unblocks       []string                  `json:"unblocks"`
	ReadyWhen      []string                  `json:"ready_when"`
	NotReadyWhen   []string                  `json:"not_ready_when"`
	Acceptance     []string                  `json:"acceptance"`
	WriteScope     []string                  `json:"write_scope"`
	TestCommands   []string                  `json:"test_commands"`
	NoTestRequired string                    `json:"no_test_required"`
	DoneSignal     []string                  `json:"done_signal"`
	Note           string                    `json:"note"`
	Blocker        *progress.BlockerMetadata `json:"blocker,omitempty"`
	// Health mirrors progress.Item.Health so candidate selection can honor
	// quarantine and ranking penalties without re-loading the file through
	// the canonical progress.Load path.
	Health *progress.RowHealth `json:"health,omitempty"`
	// PlannerVerdict mirrors progress.Item.PlannerVerdict so candidate
	// selection can honor planner-set NeedsHuman escalations without
	// re-loading the file through progress.Load. Mirrors Health exactly.
	PlannerVerdict *progress.PlannerVerdict `json:"planner_verdict,omitempty"`
}

// toProgressItem builds a progress.Item view containing only the fields used
// by progress.ItemSpecHash. Values are passed through verbatim so the digest
// matches the one progress.Load + progress.ItemSpecHash would produce against
// the same file.
func (item progressItem) toProgressItem() progress.Item {
	return progress.Item{
		Contract:             item.Contract,
		ContractStatus:       progress.ContractStatus(item.ContractStatus),
		BlockedBy:            append([]string(nil), item.BlockedBy...),
		WriteScope:           append([]string(nil), item.WriteScope...),
		TestCommands:         append([]string(nil), item.TestCommands...),
		NoTestRequiredReason: item.NoTestRequired,
		Fixture:              item.Fixture,
		Blocker:              cloneBlockerMetadata(item.Blocker),
	}
}

func cloneBlockerMetadata(blocker *progress.BlockerMetadata) *progress.BlockerMetadata {
	if blocker == nil {
		return nil
	}
	clone := *blocker
	clone.MissingFields = append([]string(nil), blocker.MissingFields...)
	return &clone
}

const (
	candidateBucketP0 = iota
	candidateBucketInProgress
	candidateBucketFixtureReady
	candidateBucketUnblocks
	candidateBucketDraft
	candidateBucketPlanned
	candidateBucketOther
)

func candidateBucket(candidate Candidate) int {
	switch {
	case strings.EqualFold(strings.TrimSpace(candidate.Priority), "P0"):
		return candidateBucketP0
	case candidate.Status == "in_progress":
		return candidateBucketInProgress
	case candidate.ContractStatus == "fixture_ready":
		return candidateBucketFixtureReady
	case len(candidate.Unblocks) > 0:
		return candidateBucketUnblocks
	case candidate.ContractStatus == "draft":
		return candidateBucketDraft
	case candidate.Status == "planned":
		return candidateBucketPlanned
	default:
		return candidateBucketOther
	}
}

func blockerKeys(phaseID, subphaseID, itemName string) []string {
	phaseID = strings.TrimSpace(phaseID)
	subphaseID = strings.TrimSpace(subphaseID)
	itemName = strings.TrimSpace(itemName)
	if itemName == "" {
		return nil
	}

	var keys []string
	for _, key := range []string{itemName} {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized != "" {
			keys = append(keys, normalized)
		}
	}
	if subphaseID != "" {
		keys = append(keys, strings.ToLower(subphaseID+"/"+itemName))
	}
	if phaseID != "" && subphaseID != "" {
		keys = append(keys, strings.ToLower(phaseID+"/"+subphaseID+"/"+itemName))
	}
	return keys
}

// selectSpeculativeCandidates identifies candidates that are blocked by
// incomplete dependencies but have their ready_when satisfied. These can
// be started speculatively for parallel execution, with verification before
// promotion ensuring the spec hasn't changed and blockers completed.
//
// readyWhenSatisfied is a predicate function that checks if a candidate's
// ready_when conditions are met (passed as a function to avoid circular
// dependencies with the progress package).
func selectSpeculativeCandidates(
	candidates []Candidate,
	completed map[string]struct{},
	readyWhenSatisfied func(Candidate) bool,
	maxSpeculative int,
) []Candidate {
	if maxSpeculative <= 0 {
		return nil
	}

	var speculative []Candidate
	for _, c := range candidates {
		// Skip if already selected or no blocked_by
		if len(c.BlockedBy) == 0 {
			continue
		}

		// Check which blockers are pending
		var pending []string
		for _, blocker := range c.BlockedBy {
			key := strings.ToLower(strings.TrimSpace(blocker))
			if key == "" {
				continue
			}
			if _, ok := completed[key]; !ok {
				pending = append(pending, key)
			}
		}

		// If all blockers complete, not speculative
		if len(pending) == 0 {
			continue
		}

		// Check if ready_when is satisfied
		if !readyWhenSatisfied(c) {
			continue
		}

		// This is a speculative candidate
		c.Speculative = true
		c.BlockedByPending = pending
		speculative = append(speculative, c)

		if len(speculative) >= maxSpeculative {
			break
		}
	}

	return speculative
}

// enrichCandidatesWithSpecHash populates the SpecHashAtClaim field for all
// candidates using the provided hash function. This snapshot is used to
// detect spec changes during speculative execution.
func enrichCandidatesWithSpecHash(candidates []Candidate, hashOf func(Candidate) string) []Candidate {
	for i := range candidates {
		candidates[i].SpecHashAtClaim = hashOf(candidates[i])
	}
	return candidates
}
