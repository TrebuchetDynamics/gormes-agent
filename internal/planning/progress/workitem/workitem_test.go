package workitem

import (
	"reflect"
	"testing"
)

func TestCompletedBlockerMakesDependentRowAssignable(t *testing.T) {
	rows := []RowInput{
		{PhaseID: "1", SubphaseID: "1.A", ItemName: "Foundation", Status: "complete"},
		{
			PhaseID:        "1",
			SubphaseID:     "1.A",
			ItemName:       "Dependent",
			Status:         "planned",
			Contract:       "dependent contract",
			ContractStatus: "fixture_ready",
			SliceSize:      "small",
			BlockedBy:      []string{"Foundation"},
			TestCommands:   []string{"go test ./dependent"},
		},
	}

	classified := Classify(rows, Options{ActiveFirst: true})
	byName := rowsByName(classified)
	if got := byName["Dependent"].Classification; got != ClassificationAssignable {
		t.Fatalf("Dependent classification = %q, want %q", got, ClassificationAssignable)
	}
	if pending := byName["Dependent"].BlockedByPending; len(pending) != 0 {
		t.Fatalf("Dependent pending blockers = %#v, want none", pending)
	}

	assignable := Assignable(rows, Options{ActiveFirst: true})
	if got, want := rowNames(assignable), []string{"Dependent"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Assignable names = %#v, want %#v", got, want)
	}
}

func TestClassifyDeferredRows(t *testing.T) {
	rows := []RowInput{
		{ItemName: "complete", Status: "complete"},
		{ItemName: "blocked", Status: "planned", Contract: "c", ContractStatus: "fixture_ready", SliceSize: "small", BlockedBy: []string{"missing"}, TestCommands: []string{"go test ./x"}},
		{ItemName: "umbrella", Status: "planned", Contract: "c", ContractStatus: "draft", SliceSize: "umbrella", TestCommands: []string{"go test ./x"}},
		{ItemName: "missing proof", Status: "planned", Contract: "c", ContractStatus: "draft", SliceSize: "small"},
		{ItemName: "needs human", Status: "planned", Contract: "c", ContractStatus: "draft", SliceSize: "small", NoTestRequired: "fixture", NeedsHuman: true},
		{ItemName: "quarantined", Status: "planned", Contract: "c", ContractStatus: "draft", SliceSize: "small", NoTestRequired: "fixture", Quarantined: true},
	}

	byName := rowsByName(Classify(rows, Options{ActiveFirst: true}))
	want := map[string]Classification{
		"complete":      ClassificationComplete,
		"blocked":       ClassificationBlocked,
		"umbrella":      ClassificationUmbrella,
		"missing proof": ClassificationMissingProof,
		"needs human":   ClassificationNeedsHuman,
		"quarantined":   ClassificationQuarantined,
	}
	for name, wantClass := range want {
		if got := byName[name].Classification; got != wantClass {
			t.Fatalf("%s classification = %q, want %q", name, got, wantClass)
		}
	}
	if got := rowNames(Assignable(rows, Options{ActiveFirst: true})); len(got) != 0 {
		t.Fatalf("Assignable deferred rows = %#v, want none", got)
	}
}

func TestAssignableOrderPreservesCurrentRankingPolicy(t *testing.T) {
	rows := []RowInput{
		{ItemName: "draft", Status: "planned", Contract: "c", ContractStatus: "draft", SliceSize: "small", NoTestRequired: "fixture"},
		{ItemName: "unblocks", Status: "planned", Contract: "c", SliceSize: "small", Unblocks: []string{"next"}, NoTestRequired: "fixture"},
		{ItemName: "fixture", Status: "planned", Contract: "c", ContractStatus: "fixture_ready", SliceSize: "small", NoTestRequired: "fixture"},
		{ItemName: "active", Status: "in_progress", Contract: "c", ContractStatus: "draft", SliceSize: "small", NoTestRequired: "fixture"},
		{ItemName: "p0", Status: "planned", Priority: "P0", Contract: "c", ContractStatus: "draft", SliceSize: "small", NoTestRequired: "fixture"},
	}

	got := rowNames(Assignable(rows, Options{ActiveFirst: true}))
	want := []string{"p0", "active", "fixture", "unblocks", "draft"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Assignable order = %#v, want %#v", got, want)
	}
}

func rowsByName(rows []Row) map[string]Row {
	out := map[string]Row{}
	for _, row := range rows {
		out[row.ItemName] = row
	}
	return out
}

func rowNames(rows []Row) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ItemName)
	}
	return out
}
