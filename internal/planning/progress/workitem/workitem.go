// Package workitem classifies progress rows for handoff surfaces.
// It deliberately does not import the parent progress package, so progress
// renderers and command selectors can both cross this seam without an import
// cycle.
package workitem

import (
	"sort"
	"strconv"
	"strings"
)

// Classification is the Progress Control Plane handoff state for a row.
type Classification string

const (
	ClassificationAssignable   Classification = "assignable"
	ClassificationBlocked      Classification = "blocked"
	ClassificationUmbrella     Classification = "umbrella"
	ClassificationMissingProof Classification = "missing_proof"
	ClassificationComplete     Classification = "complete"
	ClassificationNeedsHuman   Classification = "needs_human"
	ClassificationQuarantined  Classification = "quarantined"
	ClassificationDeferred     Classification = "deferred"
)

// Options controls row classification and assignable-row ordering.
type Options struct {
	ActiveFirst        bool
	PriorityBoost      []string
	MaxPhase           int
	IncludeBlocked     bool
	IncludeUmbrella    bool
	IncludePaused      bool
	IncludeQuarantined bool
	IncludeNeedsHuman  bool
}

// RowInput is the small, package-independent row view needed to classify and
// order progress work. Adapters in the parent progress package convert richer
// row structs into this shape.
type RowInput struct {
	PhaseID        string
	PhaseName      string
	SubphaseID     string
	SubphaseName   string
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

	HasBlocker      bool
	NeedsHuman      bool
	Quarantined     bool
	StaleQuarantine bool
	PenaltyApplied  int
}

// Row is a classified RowInput plus derived handoff facts.
type Row struct {
	RowInput
	Classification    Classification
	BlockedByPending  []string
	NeedsHumanVisible bool
}

// Classify returns every input row with a single handoff classification.
func Classify(inputs []RowInput, opts Options) []Row {
	rows := make([]RowInput, 0, len(inputs))
	for _, input := range inputs {
		rows = append(rows, normalizeInput(input))
	}
	completed := completedItemSet(rows)

	out := make([]Row, 0, len(rows))
	for _, input := range rows {
		classification, pending := classify(input, completed, opts)
		out = append(out, Row{
			RowInput:          input,
			Classification:    classification,
			BlockedByPending:  pending,
			NeedsHumanVisible: input.NeedsHuman && opts.IncludeNeedsHuman,
		})
	}
	return out
}

// Assignable returns rows classified as assignable in deterministic handoff
// order. The first row is the row progress next-work should surface.
func Assignable(inputs []RowInput, opts Options) []Row {
	classified := Classify(inputs, opts)
	out := make([]Row, 0, len(classified))
	seen := map[string]struct{}{}
	for _, row := range classified {
		if row.Classification != ClassificationAssignable {
			continue
		}
		key := rowSortKey(row)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, row)
	}
	boosts := priorityBoostSet(opts.PriorityBoost)
	sort.Slice(out, func(i, j int) bool {
		left := rowRank(out[i], opts.ActiveFirst, boosts) + out[i].PenaltyApplied
		right := rowRank(out[j], opts.ActiveFirst, boosts) + out[j].PenaltyApplied
		if left != right {
			return left < right
		}
		return rowSortKey(out[i]) < rowSortKey(out[j])
	})
	return out
}

func classify(input RowInput, completed map[string]struct{}, opts Options) (Classification, []string) {
	if input.Status == "complete" {
		return ClassificationComplete, nil
	}
	if phaseAboveMax(input.PhaseID, opts.MaxPhase) || (phasePaused(input.PhaseID) && !opts.IncludePaused) {
		return ClassificationDeferred, nil
	}
	pending := pendingBlockers(input.BlockedBy, completed)
	if len(pending) > 0 && !opts.IncludeBlocked {
		return ClassificationBlocked, pending
	}
	if input.HasBlocker && !opts.IncludeBlocked {
		return ClassificationBlocked, nil
	}
	if input.SliceSize == "umbrella" && !opts.IncludeUmbrella {
		return ClassificationUmbrella, nil
	}
	if strings.TrimSpace(input.Contract) == "" {
		return ClassificationDeferred, nil
	}
	if !hasTestProof(input) {
		return ClassificationMissingProof, nil
	}
	if rowBucket(input) > rowBucketDraft {
		return ClassificationDeferred, nil
	}
	if input.NeedsHuman && !opts.IncludeNeedsHuman {
		return ClassificationNeedsHuman, nil
	}
	if input.Quarantined && !input.StaleQuarantine && !opts.IncludeQuarantined {
		return ClassificationQuarantined, nil
	}
	return ClassificationAssignable, nil
}

func normalizeInput(input RowInput) RowInput {
	input.PhaseID = strings.TrimSpace(input.PhaseID)
	input.PhaseName = strings.TrimSpace(input.PhaseName)
	input.SubphaseID = strings.TrimSpace(input.SubphaseID)
	input.SubphaseName = strings.TrimSpace(input.SubphaseName)
	input.ItemName = strings.TrimSpace(input.ItemName)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	if input.Status == "" {
		input.Status = "unknown"
	}
	input.Priority = strings.TrimSpace(input.Priority)
	input.Contract = strings.TrimSpace(input.Contract)
	input.ContractStatus = strings.ToLower(strings.TrimSpace(input.ContractStatus))
	input.SliceSize = strings.ToLower(strings.TrimSpace(input.SliceSize))
	input.ExecutionOwner = strings.ToLower(strings.TrimSpace(input.ExecutionOwner))
	input.TrustClass = trimStringSlice(input.TrustClass)
	input.DegradedMode = strings.TrimSpace(input.DegradedMode)
	input.Fixture = strings.TrimSpace(input.Fixture)
	input.SourceRefs = trimStringSlice(input.SourceRefs)
	input.BlockedBy = trimStringSlice(input.BlockedBy)
	input.Unblocks = trimStringSlice(input.Unblocks)
	input.ReadyWhen = trimStringSlice(input.ReadyWhen)
	input.NotReadyWhen = trimStringSlice(input.NotReadyWhen)
	input.Acceptance = trimStringSlice(input.Acceptance)
	input.WriteScope = trimStringSlice(input.WriteScope)
	input.TestCommands = trimStringSlice(input.TestCommands)
	input.NoTestRequired = strings.TrimSpace(input.NoTestRequired)
	input.DoneSignal = trimStringSlice(input.DoneSignal)
	input.Note = strings.TrimSpace(input.Note)
	return input
}

func trimStringSlice(values []string) []string {
	var trimmed []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	return trimmed
}

func completedItemSet(rows []RowInput) map[string]struct{} {
	completed := make(map[string]struct{})
	for _, row := range rows {
		if row.Status != "complete" {
			continue
		}
		for _, key := range blockerKeys(row.PhaseID, row.SubphaseID, row.ItemName) {
			completed[key] = struct{}{}
		}
	}
	return completed
}

func pendingBlockers(blockers []string, completed map[string]struct{}) []string {
	var pending []string
	for _, blocker := range blockers {
		key := strings.ToLower(strings.TrimSpace(blocker))
		if key == "" {
			continue
		}
		if _, ok := completed[key]; !ok {
			pending = append(pending, strings.TrimSpace(blocker))
		}
	}
	return pending
}

func blockerKeys(phaseID, subphaseID, itemName string) []string {
	phaseID = strings.TrimSpace(phaseID)
	subphaseID = strings.TrimSpace(subphaseID)
	itemName = strings.TrimSpace(itemName)
	if itemName == "" {
		return nil
	}
	keys := []string{strings.ToLower(itemName)}
	if subphaseID != "" {
		keys = append(keys, strings.ToLower(subphaseID+"/"+itemName))
	}
	if phaseID != "" && subphaseID != "" {
		keys = append(keys, strings.ToLower(phaseID+"/"+subphaseID+"/"+itemName))
	}
	return keys
}

func hasTestProof(input RowInput) bool {
	return len(input.TestCommands) > 0 || strings.TrimSpace(input.NoTestRequired) != ""
}

const (
	rowBucketP0 = iota
	rowBucketInProgress
	rowBucketFixtureReady
	rowBucketUnblocks
	rowBucketDraft
	rowBucketPlanned
	rowBucketOther
)

func rowBucket(input RowInput) int {
	switch {
	case strings.EqualFold(strings.TrimSpace(input.Priority), "P0"):
		return rowBucketP0
	case input.Status == "in_progress":
		return rowBucketInProgress
	case input.ContractStatus == "fixture_ready":
		return rowBucketFixtureReady
	case len(input.Unblocks) > 0:
		return rowBucketUnblocks
	case input.ContractStatus == "draft":
		return rowBucketDraft
	case input.Status == "planned":
		return rowBucketPlanned
	default:
		return rowBucketOther
	}
}

func rowRank(row Row, activeFirst bool, boosts map[string]struct{}) int {
	rank := 0
	if _, ok := boosts[strings.ToLower(strings.TrimSpace(row.SubphaseID))]; !ok {
		rank += 1000
	}
	if activeFirst {
		rank += rowBucket(row.RowInput) * 10
	}
	rank += priorityTie(row.Priority)
	return rank
}

func rowSortKey(row Row) string {
	return row.PhaseID + "/" + row.SubphaseID + "/" + row.ItemName
}

// FailurePenalty returns the current row-health ranking penalty used by
// assignable-row ordering.
func FailurePenalty(consecutiveFailures int) int {
	switch {
	case consecutiveFailures <= 0:
		return 0
	case consecutiveFailures == 1:
		return 5
	case consecutiveFailures == 2:
		return 20
	default:
		return 45
	}
}

func priorityTie(priority string) int {
	normalized := strings.ToUpper(strings.TrimSpace(priority))
	if len(normalized) < 2 || normalized[0] != 'P' {
		return 9
	}
	value, err := strconv.Atoi(normalized[1:])
	if err != nil || value < 0 || value > 9 {
		return 9
	}
	return value
}

func priorityBoostSet(boosts []string) map[string]struct{} {
	set := make(map[string]struct{}, len(boosts))
	for _, boost := range boosts {
		key := strings.ToLower(strings.TrimSpace(boost))
		if key != "" {
			set[key] = struct{}{}
		}
	}
	return set
}

func phaseAboveMax(phaseID string, maxPhase int) bool {
	if maxPhase < 1 {
		return false
	}
	phaseNum, ok := phaseNumber(phaseID)
	if !ok {
		return false
	}
	return phaseNum > maxPhase
}

func phaseNumber(phaseID string) (int, bool) {
	phaseNum, err := strconv.Atoi(strings.TrimSpace(phaseID))
	if err != nil {
		return 0, false
	}
	return phaseNum, true
}

func phasePaused(phaseID string) bool {
	return strings.TrimSpace(phaseID) == "7"
}
