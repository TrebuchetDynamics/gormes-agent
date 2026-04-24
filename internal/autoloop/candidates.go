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
	seen := make(map[string]struct{})
	for _, phase := range progress.Phases {
		for _, subphase := range phase.allSubphases() {
			for _, item := range subphase.Items {
				name := firstNonEmpty(item.ItemName, item.Name, item.Title, item.ID)
				if name == "" {
					continue
				}

				status := strings.ToLower(strings.TrimSpace(item.Status))
				if status == "" {
					status = "unknown"
				}
				if status == "complete" {
					continue
				}

				candidate := Candidate{
					PhaseID:    strings.TrimSpace(phase.ID),
					SubphaseID: strings.TrimSpace(subphase.ID),
					ItemName:   name,
					Status:     status,
				}
				seenKey := candidateSortKey(candidate)
				if _, ok := seen[seenKey]; ok {
					continue
				}
				seen[seenKey] = struct{}{}

				candidates = append(candidates, candidate)
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
	if _, ok := boosts[strings.ToLower(strings.TrimSpace(candidate.SubphaseID))]; !ok {
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
