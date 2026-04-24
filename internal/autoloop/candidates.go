package autoloop

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

type CandidateOptions struct {
	ActiveFirst   bool
	PriorityBoost []string
}

type Candidate struct {
	PhaseID    string
	SubphaseID string
	ItemName   string
	Status     string
}

func NormalizeCandidates(path string, opts CandidateOptions) ([]Candidate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var progress progressJSON
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, err
	}

	var candidates []Candidate
	for _, phase := range progress.Phases {
		for _, subphase := range phase.allSubphases() {
			for _, item := range subphase.Items {
				name := firstNonEmpty(item.ItemName, item.Name, item.Title, item.ID)
				if name == "" {
					continue
				}

				status := strings.ToLower(strings.TrimSpace(item.Status))
				if status == "complete" {
					continue
				}

				candidates = append(candidates, Candidate{
					PhaseID:    strings.TrimSpace(phase.ID),
					SubphaseID: strings.TrimSpace(subphase.ID),
					ItemName:   name,
					Status:     status,
				})
			}
		}
	}

	boosts := priorityBoostSet(opts.PriorityBoost)
	sort.Slice(candidates, func(i, j int) bool {
		left := candidateRank(candidates[i], opts.ActiveFirst, boosts)
		right := candidateRank(candidates[j], opts.ActiveFirst, boosts)
		if left != right {
			return left < right
		}

		return candidateSortKey(candidates[i]) < candidateSortKey(candidates[j])
	})

	return candidates, nil
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
	Phases []progressPhase `json:"phases"`
}

type progressPhase struct {
	ID        string             `json:"id"`
	Subphases []progressSubphase `json:"subphases"`
	SubPhases []progressSubphase `json:"sub_phases"`
}

func (phase progressPhase) allSubphases() []progressSubphase {
	if len(phase.SubPhases) == 0 {
		return phase.Subphases
	}

	return append(append([]progressSubphase{}, phase.Subphases...), phase.SubPhases...)
}

type progressSubphase struct {
	ID    string         `json:"id"`
	Items []progressItem `json:"items"`
}

type progressItem struct {
	ItemName string `json:"item_name"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	ID       string `json:"id"`
	Status   string `json:"status"`
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

func candidateRank(candidate Candidate, activeFirst bool, boosts map[string]struct{}) int {
	rank := 0
	if _, ok := boosts[strings.ToLower(strings.TrimSpace(candidate.PhaseID)+"."+strings.TrimSpace(candidate.SubphaseID))]; !ok {
		rank += 10
	}

	if activeFirst {
		switch candidate.Status {
		case "in_progress":
		case "planned":
			rank += 1
		default:
			rank += 2
		}
	}

	return rank
}

func candidateSortKey(candidate Candidate) string {
	return candidate.PhaseID + "/" + candidate.SubphaseID + "/" + candidate.ItemName
}
