package slashcompletion

import "sort"

func sortedCompletionKeys[T any](m map[string]T) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func flattenCompletionGroups(groups [][]Completion) []Completion {
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	if count == 0 {
		return nil
	}
	merged := make([]Completion, 0, count)
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return merged
}

type slashCompletionCandidate struct {
	Name         string
	Display      string
	Description  string
	ArgumentHint string
}

type candidateAdmission struct {
	Identity     completionIdentity
	Accepted     bool
	Empty        bool
	Duplicate    bool
	PrefixMissed bool
}

type candidateEvidence struct {
	EmptyDropped  int
	PrefixMissed  int
	DuplicateKeys []string
}

func (e *candidateEvidence) recordRejectedAdmission(admission candidateAdmission) bool {
	if admission.Empty {
		e.EmptyDropped++
		return true
	}
	if admission.Duplicate {
		e.DuplicateKeys = append(e.DuplicateKeys, admission.Identity.Key)
		return true
	}
	if !admission.Accepted {
		if admission.PrefixMissed {
			e.PrefixMissed++
		}
		return true
	}
	return false
}

type slashCompletionCandidatePlan struct {
	candidateEvidence
	Completions []Completion
}

func (p slashCompletionCandidatePlan) empty() bool {
	return len(p.Completions) == 0
}

func planSlashCompletionCandidates(prefix completionPrefix, candidates []slashCompletionCandidate) slashCompletionCandidatePlan {
	if len(candidates) == 0 {
		return slashCompletionCandidatePlan{}
	}
	seen := make(map[string]struct{}, len(candidates))
	plan := slashCompletionCandidatePlan{Completions: make([]Completion, 0, len(candidates))}
	for _, candidate := range candidates {
		admission := admitCompletionCandidate(candidate.Name, prefix, seen)
		if plan.recordRejectedAdmission(admission) {
			continue
		}
		plan.Completions = append(plan.Completions, Completion{
			Name:         candidate.Name,
			Display:      candidate.Display,
			Description:  candidate.Description,
			ArgumentHint: candidate.ArgumentHint,
			Available:    true,
		})
	}
	return finishSlashCompletionCandidatePlan(plan)
}

func admitCompletionCandidate(rawName string, prefix completionPrefix, seen map[string]struct{}) candidateAdmission {
	identity := newCompletionIdentity(rawName)
	if !identity.Valid() {
		return candidateAdmission{Identity: identity, Empty: true}
	}
	if !prefix.Matches(identity.Name) {
		return candidateAdmission{Identity: identity, PrefixMissed: true}
	}
	if _, ok := seen[identity.Key]; ok {
		return candidateAdmission{Identity: identity, Duplicate: true}
	}
	seen[identity.Key] = struct{}{}
	return candidateAdmission{Identity: identity, Accepted: true}
}

func finishSlashCompletionCandidatePlan(plan slashCompletionCandidatePlan) slashCompletionCandidatePlan {
	if len(plan.Completions) == 0 {
		plan.Completions = nil
		return plan
	}
	sort.Slice(plan.Completions, func(i, j int) bool {
		return completionKey(plan.Completions[i].Name) < completionKey(plan.Completions[j].Name)
	})
	return plan
}

type uniqueCompletionPlan struct {
	Completions   []Completion
	EmptyDropped  int
	DuplicateKeys []string
}

func (p uniqueCompletionPlan) empty() bool {
	return len(p.Completions) == 0
}

func planUniqueCompletions(candidates []Completion) uniqueCompletionPlan {
	if len(candidates) == 0 {
		return uniqueCompletionPlan{}
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]Completion, 0, len(candidates))
	plan := uniqueCompletionPlan{}
	for _, c := range candidates {
		identity := newCompletionIdentity(c.Name)
		if !identity.Valid() {
			plan.EmptyDropped++
			continue
		}
		if _, ok := seen[identity.Key]; ok {
			plan.DuplicateKeys = append(plan.DuplicateKeys, identity.Key)
			continue
		}
		seen[identity.Key] = struct{}{}
		out = append(out, c)
	}
	plan.Completions = out
	return finishUniqueCompletionPlan(plan)
}

func finishUniqueCompletionPlan(plan uniqueCompletionPlan) uniqueCompletionPlan {
	if len(plan.Completions) == 0 {
		plan.Completions = nil
		return plan
	}
	sort.Slice(plan.Completions, func(i, j int) bool {
		return completionKey(plan.Completions[i].Name) < completionKey(plan.Completions[j].Name)
	})
	return plan
}

func uniqueSortedCompletions(candidates []Completion) []Completion {
	plan := planUniqueCompletions(candidates)
	if plan.empty() {
		return nil
	}
	return plan.Completions
}
