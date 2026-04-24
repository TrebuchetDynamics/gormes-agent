package progress

import (
	"errors"
	"fmt"
	"strings"
)

// Validate enforces schema invariants. All violations are reported,
// not just the first — authors should be able to see the full list
// and fix them in one pass. Phase and subphase keys are iterated in
// sorted order so output is deterministic.
func Validate(p *Progress) error {
	if p.Meta.Version != "2.0" {
		return fmt.Errorf("progress: meta.version = %q, want %q", p.Meta.Version, "2.0")
	}
	var errs []error
	for _, phKey := range sortedMapKeys(p.Phases) {
		ph := p.Phases[phKey]
		for _, spKey := range sortedMapKeys(ph.Subphases) {
			sp := ph.Subphases[spKey]
			hasItems := len(sp.Items) > 0
			hasStatus := sp.Status != ""
			if hasItems == hasStatus {
				errs = append(errs, fmt.Errorf("progress: phase %s subphase %s must have exactly one of items or status", phKey, spKey))
				continue // further item-level checks would just add noise
			}
			if hasStatus && !validStatus(sp.Status) {
				errs = append(errs, fmt.Errorf("progress: phase %s subphase %s: invalid status %q", phKey, spKey, sp.Status))
			}
			for i, it := range sp.Items {
				if !validStatus(it.Status) {
					errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): invalid status %q",
						phKey, spKey, i, it.Name, it.Status))
				}
				if it.Priority != "" && !validPriority(it.Priority) {
					errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): invalid priority %q",
						phKey, spKey, i, it.Name, it.Priority))
				}
				if it.ContractStatus != "" && !validContractStatus(it.ContractStatus) {
					errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): invalid contract_status %q",
						phKey, spKey, i, it.Name, it.ContractStatus))
				}
				if it.SliceSize != "" && !validSliceSize(it.SliceSize) {
					errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): invalid slice_size %q",
						phKey, spKey, i, it.Name, it.SliceSize))
				}
				if it.ExecutionOwner != "" && !validExecutionOwner(it.ExecutionOwner) {
					errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): invalid execution_owner %q",
						phKey, spKey, i, it.Name, it.ExecutionOwner))
				}
				for _, tc := range it.TrustClass {
					if !validTrustClass(tc) {
						errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): invalid trust_class %q",
							phKey, spKey, i, it.Name, tc))
					}
				}
				if requiresContractMetadata(it) {
					errs = append(errs, validateContractMetadata(phKey, spKey, i, it)...)
				}
				errs = append(errs, validateExecutionMetadata(phKey, spKey, i, it)...)
			}
		}
	}
	return errors.Join(errs...)
}

func validStatus(s Status) bool {
	return s == StatusComplete || s == StatusInProgress || s == StatusPlanned
}

func validPriority(s string) bool {
	switch s {
	case "P0", "P1", "P2", "P3", "P4":
		return true
	default:
		return false
	}
}

func validContractStatus(s ContractStatus) bool {
	switch s {
	case ContractStatusMissing, ContractStatusDraft, ContractStatusFixtureReady, ContractStatusValidated:
		return true
	default:
		return false
	}
}

func validSliceSize(s SliceSize) bool {
	switch s {
	case SliceSizeSmall, SliceSizeMedium, SliceSizeLarge, SliceSizeUmbrella:
		return true
	default:
		return false
	}
}

func validExecutionOwner(s ExecutionOwner) bool {
	switch s {
	case ExecutionOwnerDocs, ExecutionOwnerGateway, ExecutionOwnerMemory, ExecutionOwnerProvider, ExecutionOwnerTools, ExecutionOwnerSkills, ExecutionOwnerOrchestrator:
		return true
	default:
		return false
	}
}

func validTrustClass(s string) bool {
	switch s {
	case "operator", "gateway", "child-agent", "system":
		return true
	default:
		return false
	}
}

func requiresContractMetadata(it Item) bool {
	return it.Status == StatusInProgress || it.Priority == "P0"
}

func validateContractMetadata(phKey, spKey string, index int, it Item) []error {
	var errs []error
	add := func(field string) {
		errs = append(errs, fmt.Errorf("progress: phase %s subphase %s item[%d] (%q): active/P0 item missing %s",
			phKey, spKey, index, it.Name, field))
	}
	if it.Contract == "" {
		add("contract")
	}
	if it.ContractStatus == "" {
		add("contract_status")
	}
	if len(it.TrustClass) == 0 {
		add("trust_class")
	}
	if it.DegradedMode == "" {
		add("degraded_mode")
	}
	if it.Fixture == "" {
		add("fixture")
	}
	if len(it.SourceRefs) == 0 {
		add("source_refs")
	}
	if len(it.Acceptance) == 0 {
		add("acceptance")
	}
	return errs
}

func validateExecutionMetadata(phKey, spKey string, index int, it Item) []error {
	var errs []error
	add := func(message string, args ...any) {
		prefix := fmt.Sprintf("progress: phase %s subphase %s item[%d] (%q): ", phKey, spKey, index, it.Name)
		errs = append(errs, fmt.Errorf(prefix+message, args...))
	}

	hasContract := it.Contract != ""
	if hasContract {
		if it.SliceSize == "" {
			add("contract row missing slice_size")
		}
		if it.ExecutionOwner == "" {
			add("contract row missing execution_owner")
		}
		if len(it.ReadyWhen) == 0 {
			add("contract row missing ready_when")
		}
	}
	if it.Status == StatusInProgress && it.SliceSize == SliceSizeUmbrella {
		add("in_progress item cannot have slice_size %q", SliceSizeUmbrella)
	}
	if it.SliceSize == SliceSizeUmbrella {
		if it.ExecutionOwner == "" {
			add("umbrella item missing execution_owner")
		}
		if len(it.NotReadyWhen) == 0 {
			add("umbrella item missing not_ready_when")
		}
	}
	if len(it.BlockedBy) > 0 && len(it.ReadyWhen) == 0 {
		add("blocked item missing ready_when")
	}
	if it.ContractStatus == ContractStatusFixtureReady && !concreteFixtureRef(it.Fixture) {
		add("fixture_ready item needs concrete fixture path or package, got %q", it.Fixture)
	}
	if it.Status == StatusComplete && it.Contract != "" && it.ContractStatus != ContractStatusValidated {
		add("complete contract row must use contract_status %q", ContractStatusValidated)
	}
	return errs
}

func concreteFixtureRef(fixture string) bool {
	if fixture == "" {
		return false
	}
	return strings.Contains(fixture, "/") || strings.Contains(fixture, ".")
}
