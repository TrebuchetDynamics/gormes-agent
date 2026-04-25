package progress

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// RowHealth is execution-history metadata about one progress.json item.
// Owned by autoloop. The planner READS it to prioritize repairs and MUST
// preserve any unknown fields verbatim across regenerations.
type RowHealth struct {
	AttemptCount        int             `json:"attempt_count,omitempty"`
	ConsecutiveFailures int             `json:"consecutive_failures,omitempty"`
	LastAttempt         string          `json:"last_attempt,omitempty"` // RFC3339
	LastSuccess         string          `json:"last_success,omitempty"` // RFC3339
	LastFailure         *FailureSummary `json:"last_failure,omitempty"`
	BackendsTried       []string        `json:"backends_tried,omitempty"`
	Quarantine          *Quarantine     `json:"quarantine,omitempty"`
}

// FailureSummary is autoloop's classification of a worker outcome.
type FailureSummary struct {
	RunID      string          `json:"run_id"`
	Category   FailureCategory `json:"category"`
	Backend    string          `json:"backend,omitempty"`
	StderrTail string          `json:"stderr_tail,omitempty"` // capped at 2 KiB by writer
}

// FailureCategory is the closed set of failure classifications autoloop emits.
type FailureCategory string

const (
	FailureWorkerError      FailureCategory = "worker_error"
	FailureReportValidation FailureCategory = "report_validation_failed"
	FailureProgressSummary  FailureCategory = "progress_summary_failed"
	FailureTimeout          FailureCategory = "timeout"
	FailureBackendDegraded  FailureCategory = "backend_degraded"
)

// Quarantine is set when ConsecutiveFailures crosses QUARANTINE_THRESHOLD.
// Cleared when (a) a future run succeeds on the row, (b) the row's spec hash
// changes (planner reshape detected), or (c) a human deletes the block.
type Quarantine struct {
	Reason       string          `json:"reason"`
	Since        string          `json:"since"` // RFC3339
	AfterRunID   string          `json:"after_run_id"`
	Threshold    int             `json:"threshold"`
	SpecHash     string          `json:"spec_hash"`
	LastCategory FailureCategory `json:"last_category"`
}

// ItemSpecHash returns a stable SHA-256 hex digest of the row's spec fields
// used for quarantine auto-clear detection. Excludes Name, Status, Health,
// and other run-state metadata so a quarantine survives cosmetic edits but
// clears when the planner materially reshapes the contract.
//
// BlockedBy and WriteScope are sorted before hashing so reorderings don't
// invalidate quarantine. The view is JSON-encoded with omitempty so absent
// optional fields contribute nothing to the digest.
func ItemSpecHash(item *Item) string {
	type specView struct {
		Contract       string         `json:"contract,omitempty"`
		ContractStatus ContractStatus `json:"contract_status,omitempty"`
		BlockedBy      []string       `json:"blocked_by,omitempty"`
		WriteScope     []string       `json:"write_scope,omitempty"`
		Fixture        string         `json:"fixture,omitempty"`
	}
	view := specView{
		Contract:       item.Contract,
		ContractStatus: item.ContractStatus,
		BlockedBy:      append([]string(nil), item.BlockedBy...),
		WriteScope:     append([]string(nil), item.WriteScope...),
		Fixture:        item.Fixture,
	}
	sort.Strings(view.BlockedBy)
	sort.Strings(view.WriteScope)

	body, _ := json.Marshal(view)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// HealthUpdate is one mutation in a batched run-end write. Mutate is invoked
// with a pointer to the row's Health block; if the row had no prior block,
// a fresh zero-value RowHealth is allocated and attached before the callback
// runs, so Mutate never receives nil.
type HealthUpdate struct {
	PhaseID    string
	SubphaseID string
	ItemName   string
	// Mutate receives the current Health pointer (never nil — a fresh
	// zero-value RowHealth is allocated if the row has no health block yet).
	Mutate func(current *RowHealth)
}

// ApplyHealthUpdates loads progress.json, applies a batch of mutations in
// memory, and writes the file back atomically (temp + rename). Returns an
// error if any update targets a row that does not exist; the file is left
// untouched on error because the rename only happens after every mutation
// has resolved its target. Caller is responsible for serializing concurrent
// writers — last writer wins.
func ApplyHealthUpdates(path string, updates []HealthUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	prog, err := Load(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	for _, upd := range updates {
		item, err := findItem(prog, upd.PhaseID, upd.SubphaseID, upd.ItemName)
		if err != nil {
			return fmt.Errorf("apply update %s/%s/%s: %w", upd.PhaseID, upd.SubphaseID, upd.ItemName, err)
		}
		if item.Health == nil {
			item.Health = &RowHealth{}
		}
		upd.Mutate(item.Health)
	}

	return SaveProgress(path, prog)
}

// SaveProgress writes the Progress document atomically: marshal to a temp
// file in the target directory, then rename. The temp file is created in
// the same directory as the target so rename(2) is an atomic same-filesystem
// op on POSIX. Stable key ordering is provided by the typed structs.
//
// HTML escaping is disabled so user-authored content with `<`, `>`, or `&`
// (common in notes that quote command syntax or markdown) round-trips
// verbatim instead of being mangled into `<` / `>` / `&`.
func SaveProgress(path string, prog *Progress) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(prog); err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}
	body := buf.Bytes() // Encoder.Encode already appends a trailing newline.

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".progress-*.json")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp into place: %w", err)
	}
	return nil
}

// findItem returns a pointer to the Item inside prog identified by the IDs.
// Returns an error if any segment is missing. The returned pointer is into
// the slice backing the Subphase's Items, so mutations through it propagate
// even though Phases/Subphases are value-typed maps (map values themselves
// are copied on lookup, but the slice header inside the copy still references
// the same underlying array).
func findItem(prog *Progress, phaseID, subphaseID, itemName string) (*Item, error) {
	if prog == nil || prog.Phases == nil {
		return nil, fmt.Errorf("progress has no phases")
	}
	phase, ok := prog.Phases[phaseID]
	if !ok {
		return nil, fmt.Errorf("phase %q not found", phaseID)
	}
	sub, ok := phase.Subphases[subphaseID]
	if !ok {
		return nil, fmt.Errorf("subphase %q not found in phase %q", subphaseID, phaseID)
	}
	for i := range sub.Items {
		if sub.Items[i].Name == itemName {
			return &sub.Items[i], nil
		}
	}
	return nil, fmt.Errorf("item %q not found in subphase %q", itemName, subphaseID)
}
