package progress

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/jsonfields"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/workitem"
)

// RowIdentity is the stable coordinate shared by progress projections. It
// identifies a row without exposing the whole progress.Item schema.
type RowIdentity struct {
	PhaseID      string
	PhaseName    string
	SubphaseID   string
	SubphaseName string
	ItemName     string
	Module       string
}

// RowReadinessProjection is the readiness-only part of an active handoff.
type RowReadinessProjection struct {
	ReadyWhen    []string
	NotReadyWhen []string
}

// ActiveHandoffProjection is a narrow builder handoff read model over the
// Logical Backlog. It intentionally omits shipped evidence, health, provenance,
// and broad row metadata so active-work callers do not learn the full Item
// schema.
type ActiveHandoffProjection struct {
	Identity             RowIdentity
	Order                int
	Classification       workitem.Classification
	Priority             string
	Status               string
	Contract             string
	ContractStatus       string
	SliceSize            string
	ExecutionOwner       string
	Fixture              string
	Readiness            RowReadinessProjection
	Unblocks             []string
	WriteScope           []string
	TestCommands         []string
	NoTestRequiredReason string
	Acceptance           []string
	DoneSignal           []string
}

// ShippedEvidenceProjection is a completed-row evidence read model over the
// Logical Backlog. It stays separate from active handoff fields.
type ShippedEvidenceProjection struct {
	Identity          RowIdentity
	Provenance        *Provenance
	Note              string
	EvidenceSummary   string
	SourceRefs        []string
	ValidationSignals []string
}

// BlockerStateProjection is the dependency/blocker view used by health-facing
// callers without requiring those callers to parse unrelated Item fields.
type BlockerStateProjection struct {
	BlockedBy []string
	Pending   []string
	Metadata  *BlockerMetadata
}

// RowHealthProjection is the execution-health read model over one progress row.
type RowHealthProjection struct {
	Identity          RowIdentity
	Classification    workitem.Classification
	Health            *RowHealth
	Quarantine        *Quarantine
	StaleQuarantine   bool
	PlannerVerdict    *PlannerVerdict
	NeedsHumanVisible bool
	Blockers          BlockerStateProjection
}

// ProjectActiveHandoffs returns assignable rows in the same deterministic order
// used by progress next-work and generated handoff views. limit <= 0 means no
// cap.
func ProjectActiveHandoffs(p *Progress, limit int) []ActiveHandoffProjection {
	return projectActiveHandoffsFromRows(allItemRows(p), limit)
}

func projectActiveHandoffsFromRows(rows []contractRow, limit int) []ActiveHandoffProjection {
	inputs := make([]workitem.RowInput, 0, len(rows))
	byKey := make(map[string]contractRow, len(rows))
	for _, row := range rows {
		inputs = append(inputs, workitemInputFromContractRow(row))
		byKey[contractRowKey(row.PhaseKey, row.SubphaseKey, row.Item.Name)] = row
	}

	assignable := workitem.Assignable(inputs, workitem.Options{ActiveFirst: true})
	if limit > 0 && len(assignable) > limit {
		assignable = assignable[:limit]
	}
	out := make([]ActiveHandoffProjection, 0, len(assignable))
	for i, row := range assignable {
		original, ok := byKey[contractRowKey(row.PhaseID, row.SubphaseID, row.ItemName)]
		if !ok {
			continue
		}
		it := original.Item
		out = append(out, ActiveHandoffProjection{
			Identity:       identityFromContractRow(original),
			Order:          i + 1,
			Classification: row.Classification,
			Priority:       it.Priority,
			Status:         string(it.Status),
			Contract:       it.Contract,
			ContractStatus: string(it.ContractStatus),
			SliceSize:      string(it.SliceSize),
			ExecutionOwner: string(it.ExecutionOwner),
			Fixture:        it.Fixture,
			Readiness: RowReadinessProjection{
				ReadyWhen:    cloneStringSlice(it.ReadyWhen),
				NotReadyWhen: cloneStringSlice(it.NotReadyWhen),
			},
			Unblocks:             cloneStringSlice(it.Unblocks),
			WriteScope:           cloneStringSlice(it.WriteScope),
			TestCommands:         cloneStringSlice(it.TestCommands),
			NoTestRequiredReason: strings.TrimSpace(it.NoTestRequiredReason),
			Acceptance:           cloneStringSlice(it.Acceptance),
			DoneSignal:           cloneStringSlice(it.DoneSignal),
		})
	}
	return out
}

// ProjectShippedEvidence returns completed-row evidence in canonical row order.
func ProjectShippedEvidence(p *Progress) []ShippedEvidenceProjection {
	rows := allItemRows(p)
	out := make([]ShippedEvidenceProjection, 0, len(rows))
	for _, row := range rows {
		it := row.Item
		if it.Status != StatusComplete {
			continue
		}
		out = append(out, ShippedEvidenceProjection{
			Identity:          identityFromContractRow(row),
			Provenance:        cloneProvenance(it.Provenance),
			Note:              strings.TrimSpace(it.Note),
			EvidenceSummary:   evidenceSummary(it),
			SourceRefs:        cloneStringSlice(it.SourceRefs),
			ValidationSignals: validationSignals(it),
		})
	}
	return out
}

// ProjectRowHealth returns health, planner-verdict, quarantine, and blocker
// state for every row in canonical order. It is a read model only; pointer
// fields are cloned so callers cannot mutate the Logical Backlog accidentally.
func ProjectRowHealth(p *Progress) []RowHealthProjection {
	rows := allItemRows(p)
	inputs := make([]workitem.RowInput, 0, len(rows))
	byKey := make(map[string]contractRow, len(rows))
	for _, row := range rows {
		inputs = append(inputs, workitemInputFromContractRow(row))
		byKey[contractRowKey(row.PhaseKey, row.SubphaseKey, row.Item.Name)] = row
	}

	classified := workitem.Classify(inputs, workitem.Options{ActiveFirst: true})
	out := make([]RowHealthProjection, 0, len(classified))
	for _, classifiedRow := range classified {
		original, ok := byKey[contractRowKey(classifiedRow.PhaseID, classifiedRow.SubphaseID, classifiedRow.ItemName)]
		if !ok {
			continue
		}
		it := original.Item
		health := cloneRowHealth(it.Health)
		var quarantine *Quarantine
		if health != nil {
			quarantine = health.Quarantine
		}
		staleQuarantine := false
		if it.Health != nil && it.Health.Quarantine != nil {
			staleQuarantine = it.Health.Quarantine.SpecHash != ItemSpecHash(&it)
		}
		out = append(out, RowHealthProjection{
			Identity:          identityFromContractRow(original),
			Classification:    classifiedRow.Classification,
			Health:            health,
			Quarantine:        quarantine,
			StaleQuarantine:   staleQuarantine,
			PlannerVerdict:    clonePlannerVerdict(it.PlannerVerdict),
			NeedsHumanVisible: classifiedRow.NeedsHumanVisible,
			Blockers: BlockerStateProjection{
				BlockedBy: cloneStringSlice(it.BlockedBy),
				Pending:   cloneStringSlice(classifiedRow.BlockedByPending),
				Metadata:  cloneBlockerMetadata(it.Blocker),
			},
		})
	}
	return out
}

func identityFromContractRow(row contractRow) RowIdentity {
	return RowIdentity{
		PhaseID:      row.PhaseKey,
		PhaseName:    row.PhaseName,
		SubphaseID:   row.SubphaseKey,
		SubphaseName: row.Subphase,
		ItemName:     row.Item.Name,
		Module:       Module(row.Item, row.PhaseKey, row.SubphaseKey),
	}
}

func evidenceSummary(it Item) string {
	if note := strings.TrimSpace(it.Note); note != "" {
		return note
	}
	if len(it.DoneSignal) > 0 {
		return strings.Join(cloneStringSlice(it.DoneSignal), "; ")
	}
	if len(it.Acceptance) > 0 {
		return strings.Join(cloneStringSlice(it.Acceptance), "; ")
	}
	if strings.TrimSpace(it.Contract) != "" {
		return strings.TrimSpace(it.Contract)
	}
	return "complete"
}

func validationSignals(it Item) []string {
	var out []string
	if it.ContractStatus != "" {
		out = append(out, "contract_status="+string(it.ContractStatus))
	}
	for _, command := range cloneStringSlice(it.TestCommands) {
		out = append(out, "test: "+command)
	}
	if reason := strings.TrimSpace(it.NoTestRequiredReason); reason != "" {
		out = append(out, "no_test_required: "+reason)
	}
	for _, signal := range cloneStringSlice(it.DoneSignal) {
		out = append(out, "done: "+signal)
	}
	return out
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneRowHealth(in *RowHealth) *RowHealth {
	if in == nil {
		return nil
	}
	out := *in
	out.BackendsTried = cloneStringSlice(in.BackendsTried)
	if in.LastFailure != nil {
		last := *in.LastFailure
		out.LastFailure = &last
	}
	out.Quarantine = cloneQuarantine(in.Quarantine)
	out.Extra = jsonfields.CloneRawMessageMap(in.Extra)
	return &out
}

func cloneQuarantine(in *Quarantine) *Quarantine {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func clonePlannerVerdict(in *PlannerVerdict) *PlannerVerdict {
	if in == nil {
		return nil
	}
	out := *in
	out.Extra = jsonfields.CloneRawMessageMap(in.Extra)
	return &out
}

func cloneProvenance(in *Provenance) *Provenance {
	if in == nil {
		return nil
	}
	out := *in
	out.UpstreamRefs = cloneStringSlice(in.UpstreamRefs)
	out.Extra = jsonfields.CloneRawMessageMap(in.Extra)
	return &out
}

func cloneBlockerMetadata(in *BlockerMetadata) *BlockerMetadata {
	if in == nil {
		return nil
	}
	out := *in
	out.MissingFields = cloneStringSlice(in.MissingFields)
	return &out
}
