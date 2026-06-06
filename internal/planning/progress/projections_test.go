package progress

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/workitem"
)

func TestProjectActiveHandoffsExposesNarrowHandoffRows(t *testing.T) {
	p := projectionFixture()

	got := ProjectActiveHandoffs(p, 10)
	if names := activeProjectionNames(got); !reflect.DeepEqual(names, []string{"Active row", "Ready row", "No-test row"}) {
		t.Fatalf("active handoff names = %#v", names)
	}

	row := got[1]
	if row.Order != 2 {
		t.Fatalf("Order = %d, want 2", row.Order)
	}
	if row.Classification != workitem.ClassificationAssignable {
		t.Fatalf("Classification = %q, want assignable", row.Classification)
	}
	if row.Identity.PhaseID != "1" || row.Identity.SubphaseID != "1.A" || row.Identity.ItemName != "Ready row" || row.Identity.Module != ModuleProgress {
		t.Fatalf("Identity = %#v", row.Identity)
	}
	if row.Contract != "ready contract" || row.Fixture != "internal/progress/projections_test.go" {
		t.Fatalf("contract/fixture not projected: %#v", row)
	}
	if !reflect.DeepEqual(row.Readiness.ReadyWhen, []string{"fixture exists"}) || !reflect.DeepEqual(row.Readiness.NotReadyWhen, []string{"scope drifts"}) {
		t.Fatalf("Readiness = %#v", row.Readiness)
	}
	if !reflect.DeepEqual(row.WriteScope, []string{"internal/progress/"}) {
		t.Fatalf("WriteScope = %#v", row.WriteScope)
	}
	if !reflect.DeepEqual(row.TestCommands, []string{"go test ./internal/progress -run TestProjectActiveHandoffs"}) {
		t.Fatalf("TestCommands = %#v", row.TestCommands)
	}
	if row.NoTestRequiredReason != "" {
		t.Fatalf("NoTestRequiredReason = %q, want empty", row.NoTestRequiredReason)
	}
	if !reflect.DeepEqual(row.Acceptance, []string{"active projection is narrow"}) {
		t.Fatalf("Acceptance = %#v", row.Acceptance)
	}
	if !reflect.DeepEqual(row.DoneSignal, []string{"projection fixture passes"}) {
		t.Fatalf("DoneSignal = %#v", row.DoneSignal)
	}
	assertNoProjectionFields(t, ActiveHandoffProjection{}, "SourceRefs", "TrustClass", "DegradedMode", "Health", "PlannerVerdict", "Provenance", "Blocker", "Note")
}

func TestProjectShippedEvidenceExposesCompletedRowEvidence(t *testing.T) {
	p := projectionFixture()

	got := ProjectShippedEvidence(p)
	if names := shippedProjectionNames(got); !reflect.DeepEqual(names, []string{"Foundation", "Shipped row"}) {
		t.Fatalf("shipped evidence names = %#v", names)
	}
	row := got[1]
	if row.Identity.ItemName != "Shipped row" {
		t.Fatalf("Identity = %#v", row.Identity)
	}
	if row.Provenance == nil || row.Provenance.OriginType != "gormes" || row.Provenance.OwnedSince != "2026-05-25" {
		t.Fatalf("Provenance = %#v", row.Provenance)
	}
	if row.Note != "SHIPPED 2026-05-25 see git log — projection evidence" {
		t.Fatalf("Note = %q", row.Note)
	}
	if row.EvidenceSummary != row.Note {
		t.Fatalf("EvidenceSummary = %q, want note", row.EvidenceSummary)
	}
	if !reflect.DeepEqual(row.SourceRefs, []string{"CONTEXT.md:Progress Projection"}) {
		t.Fatalf("SourceRefs = %#v", row.SourceRefs)
	}
	wantSignals := []string{
		"contract_status=validated",
		"test: go test ./internal/progress -run TestProjectShippedEvidence",
		"done: shipped evidence projection passes",
	}
	if !reflect.DeepEqual(row.ValidationSignals, wantSignals) {
		t.Fatalf("ValidationSignals = %#v, want %#v", row.ValidationSignals, wantSignals)
	}
	row.Provenance.OriginType = "mutated"
	if p.Phases["1"].Subphases["1.A"].Items[5].Provenance.OriginType != "gormes" {
		t.Fatal("projection provenance mutation changed the logical backlog")
	}
	assertNoProjectionFields(t, ShippedEvidenceProjection{}, "ReadyWhen", "NotReadyWhen", "WriteScope", "Acceptance", "DoneSignal", "Health", "PlannerVerdict", "Blocker")
}

func TestProjectRowHealthExposesHealthVerdictsAndBlockerState(t *testing.T) {
	p := projectionFixture()

	got := ProjectRowHealth(p)
	byName := map[string]RowHealthProjection{}
	for _, row := range got {
		byName[row.Identity.ItemName] = row
	}
	sick := byName["Sick row"]
	if sick.Health == nil || sick.Health.AttemptCount != 4 || sick.Health.ConsecutiveFailures != 3 {
		t.Fatalf("Health = %#v", sick.Health)
	}
	if sick.Quarantine == nil || sick.Quarantine.SpecHash != "stale-spec" {
		t.Fatalf("Quarantine = %#v", sick.Quarantine)
	}
	if !sick.StaleQuarantine {
		t.Fatalf("StaleQuarantine = false, want true")
	}
	if sick.PlannerVerdict == nil || !sick.PlannerVerdict.NeedsHuman || sick.PlannerVerdict.Reason != "needs owner decision" {
		t.Fatalf("PlannerVerdict = %#v", sick.PlannerVerdict)
	}
	sick.Health.AttemptCount = 99
	if p.Phases["1"].Subphases["1.A"].Items[4].Health.AttemptCount != 4 {
		t.Fatal("projection health mutation changed the logical backlog")
	}

	blocked := byName["Blocked row"]
	if blocked.Classification != workitem.ClassificationBlocked {
		t.Fatalf("blocked Classification = %q, want blocked", blocked.Classification)
	}
	if !reflect.DeepEqual(blocked.Blockers.BlockedBy, []string{"Missing dependency"}) || !reflect.DeepEqual(blocked.Blockers.Pending, []string{"Missing dependency"}) {
		t.Fatalf("Blockers = %#v", blocked.Blockers)
	}
	if blocked.Blockers.Metadata == nil || blocked.Blockers.Metadata.Type != "dependency" || blocked.Blockers.Metadata.Status != "blocked" {
		t.Fatalf("Blocker metadata = %#v", blocked.Blockers.Metadata)
	}
	assertNoProjectionFields(t, RowHealthProjection{}, "Contract", "Fixture", "WriteScope", "TestCommands", "Acceptance", "DoneSignal", "SourceRefs", "Provenance")
}

func projectionFixture() *Progress {
	return &Progress{Phases: map[string]Phase{
		"1": {Name: "Phase 1", Subphases: map[string]Subphase{
			"1.A": {Name: "Alpha", Items: []Item{
				{Name: "Foundation", Status: StatusComplete, Contract: "foundation", ContractStatus: ContractStatusValidated},
				{
					Name:           "Ready row",
					Status:         StatusPlanned,
					Module:         ModuleProgress,
					Contract:       "ready contract",
					ContractStatus: ContractStatusFixtureReady,
					SliceSize:      SliceSizeSmall,
					TrustClass:     []string{"operator", "system"},
					DegradedMode:   "wide fields stay out of active projection",
					Fixture:        "internal/progress/projections_test.go",
					SourceRefs:     []string{"CONTEXT.md:Progress Projection"},
					ReadyWhen:      []string{"fixture exists"},
					NotReadyWhen:   []string{"scope drifts"},
					BlockedBy:      []string{"Foundation"},
					Acceptance:     []string{"active projection is narrow"},
					WriteScope:     []string{"internal/progress/"},
					TestCommands:   []string{"go test ./internal/progress -run TestProjectActiveHandoffs"},
					DoneSignal:     []string{"projection fixture passes"},
					Note:           "internal note is not active handoff data",
				},
				{
					Name:                 "No-test row",
					Status:               StatusPlanned,
					Contract:             "docs contract",
					ContractStatus:       ContractStatusDraft,
					SliceSize:            SliceSizeSmall,
					ReadyWhen:            []string{"docs source exists"},
					WriteScope:           []string{"webpages/docs/content/building-gormes/"},
					NoTestRequiredReason: "docs-only projection proof",
					Acceptance:           []string{"docs copy checked"},
					DoneSignal:           []string{"docs projection listed"},
				},
				{
					Name:                 "Active row",
					Status:               StatusInProgress,
					Priority:             "P1",
					Contract:             "active contract",
					ContractStatus:       ContractStatusDraft,
					SliceSize:            SliceSizeSmall,
					ReadyWhen:            []string{"already active"},
					WriteScope:           []string{"internal/progress/"},
					NoTestRequiredReason: "active projection ranking proof",
					Acceptance:           []string{"active row appears before ready rows"},
					DoneSignal:           []string{"active row projected"},
				},
				{
					Name:           "Sick row",
					Status:         StatusPlanned,
					Contract:       "health contract",
					ContractStatus: ContractStatusFixtureReady,
					SliceSize:      SliceSizeSmall,
					ReadyWhen:      []string{"health repair identified"},
					WriteScope:     []string{"internal/progress/"},
					TestCommands:   []string{"go test ./internal/progress -run TestProjectRowHealth"},
					DoneSignal:     []string{"health projection passes"},
					Health: &RowHealth{
						AttemptCount:        4,
						ConsecutiveFailures: 3,
						Quarantine:          &Quarantine{Reason: "threshold", Since: "2026-05-25T00:00:00Z", AfterRunID: "run-1", Threshold: 3, SpecHash: "stale-spec", LastCategory: FailureNoProgress},
					},
					PlannerVerdict: &PlannerVerdict{NeedsHuman: true, Reason: "needs owner decision", Since: "2026-05-25T00:00:00Z"},
				},
				{
					Name:           "Shipped row",
					Status:         StatusComplete,
					Contract:       "shipped contract",
					ContractStatus: ContractStatusValidated,
					SourceRefs:     []string{"CONTEXT.md:Progress Projection"},
					TestCommands:   []string{"go test ./internal/progress -run TestProjectShippedEvidence"},
					DoneSignal:     []string{"shipped evidence projection passes"},
					Note:           "SHIPPED 2026-05-25 see git log — projection evidence",
					Provenance:     &Provenance{OriginType: "gormes", OwnedSince: "2026-05-25", Note: "projection-only row"},
				},
				{
					Name:           "Blocked row",
					Status:         StatusPlanned,
					Contract:       "blocked contract",
					ContractStatus: ContractStatusFixtureReady,
					SliceSize:      SliceSizeSmall,
					BlockedBy:      []string{"Missing dependency"},
					ReadyWhen:      []string{"dependency ships"},
					WriteScope:     []string{"internal/progress/"},
					TestCommands:   []string{"go test ./internal/progress -run TestProjectRowHealth"},
					DoneSignal:     []string{"blocked projection passes"},
					Blocker: &BlockerMetadata{
						Type:         "dependency",
						Status:       "blocked",
						Blocker:      "Missing dependency",
						Evidence:     "fixture dependency absent",
						UnblocksWhen: "dependency ships",
						Owner:        "tools",
						Pivot:        "project other rows",
						NextCheck:    "2026-05-26",
					},
				},
			}},
		}},
	}}
}

func activeProjectionNames(rows []ActiveHandoffProjection) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Identity.ItemName)
	}
	return out
}

func shippedProjectionNames(rows []ShippedEvidenceProjection) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Identity.ItemName)
	}
	return out
}

func assertNoProjectionFields(t *testing.T, value any, forbidden ...string) {
	t.Helper()
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	for _, name := range forbidden {
		if _, ok := typ.FieldByName(name); ok {
			t.Fatalf("%s unexpectedly exposes field %s", typ.Name(), name)
		}
	}
}
