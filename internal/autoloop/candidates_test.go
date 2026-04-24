package autoloop

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeCandidatesSkipsCompleteAndSortsActiveFirst(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"3": {
				"subphases": {
					"3.E.6": {
						"items": [
							{"item_name": "planned candidate", "status": "planned"},
							{"item_name": "complete candidate", "status": "complete"},
							{"item_name": "active candidate", "status": "IN_PROGRESS"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "3", SubphaseID: "3.E.6", ItemName: "active candidate", Status: "in_progress"},
		{PhaseID: "3", SubphaseID: "3.E.6", ItemName: "planned candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesPriorityBoostWins(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"3": {
				"subphases": {
					"3.E.7": {
						"items": [
							{"name": "boosted planned candidate", "status": "planned"}
						]
					},
					"3.E.6": {
						"items": [
							{"name": "normal active candidate", "status": "in_progress"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{
		ActiveFirst:   true,
		PriorityBoost: []string{" 3.e.7 "},
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "3", SubphaseID: "3.E.7", ItemName: "boosted planned candidate", Status: "planned"},
		{PhaseID: "3", SubphaseID: "3.E.6", ItemName: "normal active candidate", Status: "in_progress"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesHonorsMaxPhase(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"3": {
				"subphases": {
					"3.E": {
						"items": [
							{"name": "phase 3 candidate", "status": "planned"}
						]
					}
				}
			},
			"4": {
				"subphases": {
					"4.A": {
						"items": [
							{"name": "phase 4 candidate", "status": "in_progress"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true, MaxPhase: 3})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "3", SubphaseID: "3.E", ItemName: "phase 3 candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesUsesSubPhasesFallback(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"4": {
				"sub_phases": {
					"4.A.1": {
						"items": [
							{"item_name": "fallback candidate", "status": "planned"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "4", SubphaseID: "4.A.1", ItemName: "fallback candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesPrefersSubphasesWhenBothKeysExist(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"4": {
				"subphases": {
					"4.A.1": {
						"items": [
							{"item_name": "preferred candidate", "status": "planned"}
						]
					}
				},
				"sub_phases": {
					"4.A.2": {
						"items": [
							{"item_name": "fallback candidate", "status": "planned"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "4", SubphaseID: "4.A.1", ItemName: "preferred candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesItemNameFallbacksAndUnknownStatus(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"5": {
				"subphases": {
					"5.B.1": {
						"items": [
							{"item_name": "item-name candidate", "name": "ignored name", "status": "planned"},
							{"item_name": " ", "name": "name candidate", "title": "ignored title", "status": "planned"},
							{"name": " ", "title": "title candidate", "id": "ignored id", "status": "planned"},
							{"title": " ", "id": "id candidate"},
							{"item_name": " "}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "5", SubphaseID: "5.B.1", ItemName: "id candidate", Status: "unknown"},
		{PhaseID: "5", SubphaseID: "5.B.1", ItemName: "item-name candidate", Status: "planned"},
		{PhaseID: "5", SubphaseID: "5.B.1", ItemName: "name candidate", Status: "planned"},
		{PhaseID: "5", SubphaseID: "5.B.1", ItemName: "title candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesDeduplicatesByPhaseSubphaseAndItemName(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"6": {
				"subphases": {
					"6.C.1": {
						"items": [
							{"item_name": "duplicate candidate", "status": "planned"},
							{"item_name": "duplicate candidate", "status": "in_progress"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "6", SubphaseID: "6.C.1", ItemName: "duplicate candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesReturnsMalformedJSONError(t *testing.T) {
	path := writeProgressJSON(t, `{"phases":`)

	_, err := NormalizeCandidates(path, CandidateOptions{})
	if err == nil {
		t.Fatal("NormalizeCandidates() error = nil, want error")
	}
}

func TestNormalizeCandidatesReturnsMissingFileError(t *testing.T) {
	_, err := NormalizeCandidates(filepath.Join(t.TempDir(), "missing.json"), CandidateOptions{})
	if err == nil {
		t.Fatal("NormalizeCandidates() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "missing.json") {
		t.Fatalf("NormalizeCandidates() error = %q, want missing filename", err)
	}
}

func writeProgressJSON(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
