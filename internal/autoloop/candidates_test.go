package autoloop

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeCandidatesSkipsCompleteAndSortsActiveFirst(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": [
			{
				"id": "3",
				"subphases": [
					{
						"id": "E",
						"items": [
							{"item_name": "planned candidate", "status": "planned"},
							{"item_name": "complete candidate", "status": "complete"},
							{"item_name": "active candidate", "status": "IN_PROGRESS"}
						]
					}
				]
			}
		]
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "3", SubphaseID: "E", ItemName: "active candidate", Status: "in_progress"},
		{PhaseID: "3", SubphaseID: "E", ItemName: "planned candidate", Status: "planned"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesPriorityBoostWins(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": [
			{
				"id": "3",
				"sub_phases": [
					{
						"id": "E.7",
						"items": [
							{"name": "boosted planned candidate", "status": "planned"}
						]
					},
					{
						"id": "E.6",
						"items": [
							{"name": "normal active candidate", "status": "in_progress"}
						]
					}
				]
			}
		]
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{
		ActiveFirst:   true,
		PriorityBoost: []string{" 3.e.7 "},
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{PhaseID: "3", SubphaseID: "E.7", ItemName: "boosted planned candidate", Status: "planned"},
		{PhaseID: "3", SubphaseID: "E.6", ItemName: "normal active candidate", Status: "in_progress"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
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
