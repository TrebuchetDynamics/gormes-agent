package progress

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/workitem"
)

// statusIcon maps derived status to the glyph shown in markdown tables.
func statusIcon(s Status) string {
	switch s {
	case StatusComplete:
		return "✅"
	case StatusInProgress:
		return "🔨"
	default:
		return "⏳"
	}
}

// RenderReadmeRollup returns the 6-row phase table inserted into the
// README's `## Architecture` section between the PROGRESS markers.
func RenderReadmeRollup(p *Progress) string {
	var b strings.Builder
	b.WriteString("| Phase | Status | Shipped |\n")
	b.WriteString("|-------|--------|---------|\n")
	for _, key := range sortedMapKeys(p.Phases) {
		ph := p.Phases[key]
		total := len(ph.Subphases)
		complete := 0
		for _, sp := range ph.Subphases {
			if sp.DerivedStatus() == StatusComplete {
				complete++
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %d/%d subphases |\n",
			ph.Name, statusIcon(ph.DerivedStatus()), complete, total)
	}
	return b.String()
}

// RenderDocsChecklist returns the full item-level checklist embedded
// in _index.md between the PROGRESS markers. Emits:
//   - an **Overall** stats line
//   - a phase-level table matching the README rollup
//   - a per-subphase section with - [x] / - [ ] checkboxes
func RenderDocsChecklist(p *Progress) string {
	s := p.Stats()
	var b strings.Builder

	fmt.Fprintf(&b, "**Overall:** %d/%d subphases shipped · %d in progress · %d planned\n\n",
		s.Subphases.Complete, s.Subphases.Total, s.Subphases.InProgress, s.Subphases.Planned)

	b.WriteString(RenderReadmeRollup(p))
	b.WriteString("\n---\n\n")

	for _, key := range sortedMapKeys(p.Phases) {
		ph := p.Phases[key]
		fmt.Fprintf(&b, "## %s %s\n\n", ph.Name, statusIcon(ph.DerivedStatus()))
		if ph.Deliverable != "" {
			fmt.Fprintf(&b, "*%s*\n\n", ph.Deliverable)
		}
		for _, spKey := range sortedMapKeys(ph.Subphases) {
			sp := ph.Subphases[spKey]
			fmt.Fprintf(&b, "### %s — %s %s\n\n", spKey, sp.Name, statusIcon(sp.DerivedStatus()))
			if len(sp.Items) == 0 {
				st := string(sp.Status)
				if st == "" {
					st = "unspecified"
				}
				fmt.Fprintf(&b, "*(no item breakdown — tracked at subphase level: %s)*\n\n", st)
				continue
			}
			for _, it := range sp.Items {
				box := "[ ]"
				if it.Status == StatusComplete {
					box = "[x]"
				}
				fmt.Fprintf(&b, "- %s `%s` %s\n", box, Module(it, key, spKey), it.Name)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// RenderContractReadiness returns a markdown table for progress rows that have
// contract metadata. The canonical progress JSON remains the source of truth.
func RenderContractReadiness(p *Progress) string {
	rows := contractRows(p)
	if len(rows) == 0 {
		return "_No progress rows currently carry contract metadata._\n"
	}

	var b strings.Builder
	b.WriteString("| Phase | Progress item | Contract status | Owner | Size | Trust class | Fixture | Degraded mode |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, row := range rows {
		it := row.Item
		fmt.Fprintf(&b, "| %s | %s | `%s` | `%s` | `%s` | %s | `%s` | %s |\n",
			mdCell(row.PhaseKey+" / "+row.SubphaseKey),
			mdCell(it.Name+" — "+it.Contract),
			mdCell(string(it.ContractStatus)),
			mdCell(string(it.ExecutionOwner)),
			mdCell(string(it.SliceSize)),
			mdCell(joinOrDash(it.TrustClass)),
			mdCell(it.Fixture),
			mdCell(it.DegradedMode),
		)
	}
	return b.String()
}

// RenderNextSlices returns the highest-leverage unblocked, non-umbrella
// contract-bearing rows.
func RenderNextSlices(p *Progress, limit int) string {
	if limit <= 0 {
		limit = 10
	}
	rows := nextSliceRows(allItemRows(p), limit)
	if len(rows) == 0 {
		return "_No contract-ready progress rows are available._\n"
	}

	var b strings.Builder
	b.WriteString("| Phase | Slice | Contract | Trust class | Fixture | Why now |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, row := range rows {
		it := row.Item
		fmt.Fprintf(&b, "| %s | %s | %s | %s | `%s` | %s |\n",
			mdCell(row.PhaseKey+" / "+row.SubphaseKey),
			mdCell(it.Name),
			mdCell(it.Contract),
			mdCell(joinOrDash(it.TrustClass)),
			mdCell(it.Fixture),
			mdCell(whyNow(it)),
		)
	}
	return b.String()
}

// RenderAgentQueue returns execution cards for unblocked, non-umbrella rows
// that a builder skill can turn into a focused implementation attempt.
func RenderAgentQueue(p *Progress, limit int) string {
	if limit <= 0 {
		limit = 10
	}
	rows := nextSliceRows(allItemRows(p), limit)
	if len(rows) == 0 {
		return "_No unblocked contract rows are ready for autonomous execution._\n"
	}

	var b strings.Builder
	for i, row := range rows {
		it := row.Item
		fmt.Fprintf(&b, "## %d. %s\n\n", i+1, it.Name)
		fmt.Fprintf(&b, "- Phase: %s / %s\n", row.PhaseKey, row.SubphaseKey)
		fmt.Fprintf(&b, "- Owner: `%s`\n", it.ExecutionOwner)
		fmt.Fprintf(&b, "- Size: `%s`\n", it.SliceSize)
		fmt.Fprintf(&b, "- Status: `%s`\n", it.Status)
		if it.Priority != "" {
			fmt.Fprintf(&b, "- Priority: `%s`\n", it.Priority)
		}
		fmt.Fprintf(&b, "- Contract: %s\n", mdCell(it.Contract))
		fmt.Fprintf(&b, "- Trust class: %s\n", mdCell(joinOrDash(it.TrustClass)))
		fmt.Fprintf(&b, "- Ready when: %s\n", mdCell(joinOrDash(it.ReadyWhen)))
		fmt.Fprintf(&b, "- Not ready when: %s\n", mdCell(joinOrDash(it.NotReadyWhen)))
		fmt.Fprintf(&b, "- Degraded mode: %s\n", mdCell(it.DegradedMode))
		fmt.Fprintf(&b, "- Fixture: `%s`\n", mdCell(it.Fixture))
		fmt.Fprintf(&b, "- Write scope: %s\n", mdCell(joinCodeOrDash(it.WriteScope)))
		fmt.Fprintf(&b, "- Test commands: %s\n", mdCell(joinCodeOrDash(it.TestCommands)))
		if strings.TrimSpace(it.NoTestRequiredReason) != "" {
			fmt.Fprintf(&b, "- No test required: %s\n", mdCell(it.NoTestRequiredReason))
		}
		fmt.Fprintf(&b, "- Done signal: %s\n", mdCell(joinOrDash(it.DoneSignal)))
		fmt.Fprintf(&b, "- Acceptance: %s\n", mdCell(joinOrDash(it.Acceptance)))
		fmt.Fprintf(&b, "- Source refs: %s\n", mdCell(joinOrDash(it.SourceRefs)))
		if len(it.Unblocks) > 0 {
			fmt.Fprintf(&b, "- Unblocks: %s\n", mdCell(joinOrDash(it.Unblocks)))
		}
		fmt.Fprintf(&b, "- Why now: %s\n\n", mdCell(whyNow(it)))
	}
	return b.String()
}

// RenderBlockedSlices returns rows that cannot start until another roadmap row
// is complete or another readiness condition becomes true.
func RenderBlockedSlices(p *Progress) string {
	rows := blockedRows(allItemRows(p))
	if len(rows) == 0 {
		return "_No contract-bearing rows are currently blocked._\n"
	}

	var b strings.Builder
	b.WriteString("| Phase | Slice | Blocked by | Ready when | Unblocks |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, row := range rows {
		it := row.Item
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			mdCell(row.PhaseKey+" / "+row.SubphaseKey),
			mdCell(it.Name),
			mdCell(joinOrDash(it.BlockedBy)),
			mdCell(joinOrDash(it.ReadyWhen)),
			mdCell(joinOrDash(it.Unblocks)),
		)
	}
	return b.String()
}

// RenderUmbrellaCleanup returns planned rows that are inventory buckets rather
// than executable implementation slices.
func RenderUmbrellaCleanup(p *Progress) string {
	rows := umbrellaRows(p)
	if len(rows) == 0 {
		return "_No umbrella rows are currently marked for cleanup._\n"
	}

	var b strings.Builder
	b.WriteString("| Phase | Umbrella row | Owner | Not ready when | Split into |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, row := range rows {
		it := row.Item
		fmt.Fprintf(&b, "| %s | %s | `%s` | %s | %s |\n",
			mdCell(row.PhaseKey+" / "+row.SubphaseKey),
			mdCell(it.Name),
			mdCell(string(it.ExecutionOwner)),
			mdCell(joinOrDash(it.NotReadyWhen)),
			mdCell(joinOrDash(it.Unblocks)),
		)
	}
	return b.String()
}

// RenderBuilderLoopHandoff returns the control-plane facts used by skill-driven
// planner and builder passes. The JSON field keeps the historical builder_loop
// name so older progress files still load.
func RenderBuilderLoopHandoff(p *Progress) string {
	m := p.Meta.BuilderLoop
	if !builderLoopMetaDeclared(m) {
		return "_No skill handoff metadata declared in canonical progress._\n"
	}

	var b strings.Builder
	b.WriteString("## Control Plane\n\n")
	fmt.Fprintf(&b, "- Entrypoint: `%s`\n", mdCell(m.Entrypoint))
	fmt.Fprintf(&b, "- Plan: `%s`\n", mdCell(m.Plan))
	fmt.Fprintf(&b, "- Candidate source: `%s`\n", mdCell(m.CandidateSource))
	fmt.Fprintf(&b, "- Agent queue: `%s`\n", mdCell(m.AgentQueue))
	fmt.Fprintf(&b, "- Progress schema: `%s`\n", mdCell(m.ProgressSchema))
	fmt.Fprintf(&b, "- Unit tests: `%s`\n", mdCell(m.UnitTest))

	b.WriteString("\n## Candidate Policy\n\n")
	if len(m.CandidatePolicy) == 0 {
		b.WriteString("- (not declared)\n")
	} else {
		for _, policy := range m.CandidatePolicy {
			fmt.Fprintf(&b, "- %s\n", mdCell(policy))
		}
	}
	return b.String()
}

// RenderModuleRoadmapIndex returns the generated index for module-scoped
// roadmap review pages. It is a view over the single logical backlog, not a
// second queue.
func RenderModuleRoadmapIndex(p *Progress) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(`title: "Module Roadmaps"` + "\n")
	b.WriteString("weight: 35\n")
	b.WriteString("---\n\n")
	b.WriteString("# Module Roadmaps\n\n")
	b.WriteString("Generated from the single logical backlog. These pages are scoped review views; `progress.json` remains canonical.\n\n")
	b.WriteString("| Module | Rows | Complete | In progress | Planned | Priorities |\n")
	b.WriteString("|---|---:|---:|---:|---:|---|\n")
	for _, module := range AllowedModules() {
		counts := countModuleRows(moduleRoadmapRows(p, module))
		fmt.Fprintf(&b, "| [%s](%s/) | %d | %d | %d | %d | %s |\n",
			moduleDisplayName(module),
			module,
			counts.Total,
			counts.Status[StatusComplete],
			counts.Status[StatusInProgress],
			counts.Status[StatusPlanned],
			formatPriorityCounts(counts.Priority),
		)
	}
	return b.String()
}

// RenderModuleRoadmapPage returns one generated module page grouped by the
// original phase/subphase coordinates so physical module review does not lose
// the unified roadmap context.
func RenderModuleRoadmapPage(p *Progress, module string) string {
	display := moduleDisplayName(module)
	rows := moduleRoadmapRows(p, module)
	counts := countModuleRows(rows)

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", display+" Module Roadmap")
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s Module Roadmap\n\n", display)
	b.WriteString("Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.\n\n")
	fmt.Fprintf(&b, "**Module:** `%s`\n", module)
	fmt.Fprintf(&b, "**Rows:** %d\n", counts.Total)
	fmt.Fprintf(&b, "**Status counts:** `complete`: %d · `in_progress`: %d · `planned`: %d\n",
		counts.Status[StatusComplete], counts.Status[StatusInProgress], counts.Status[StatusPlanned])
	fmt.Fprintf(&b, "**Priority counts:** %s\n\n", formatPriorityCounts(counts.Priority))

	if len(rows) == 0 {
		fmt.Fprintf(&b, "_No rows currently assigned to `%s`._\n", module)
		return b.String()
	}

	lastPhase := ""
	lastSubphase := ""
	for _, row := range rows {
		if row.PhaseKey != lastPhase {
			if lastPhase != "" {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "## %s\n\n", row.PhaseName)
			lastPhase = row.PhaseKey
			lastSubphase = ""
		}
		if row.SubphaseKey != lastSubphase {
			if lastSubphase != "" {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "### %s — %s\n\n", row.SubphaseKey, row.Subphase)
			b.WriteString("| Status | Priority | Module | Row |\n")
			b.WriteString("|---|---|---|---|\n")
			lastSubphase = row.SubphaseKey
		}
		priority := row.Item.Priority
		if priority == "" {
			priority = "unset"
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %s |\n",
			row.Item.Status, priority, Module(row.Item, row.PhaseKey, row.SubphaseKey), mdCell(row.Item.Name))
	}
	return b.String()
}

// RenderProgressSchema returns the operator-facing schema reference for
// contract-aware progress rows.
func RenderProgressSchema() string {
	return strings.TrimSpace(`
## Item Fields

| Field | Required when | Meaning |
|---|---|---|
| `+"`name`"+` | every item | Human-readable roadmap row name. |
| `+"`status`"+` | every item | `+"`planned`"+`, `+"`in_progress`"+`, or `+"`complete`"+`. |
| `+"`priority`"+` | optional | `+"`P0`"+` through `+"`P4`"+`. Item-level `+"`P0`"+` rows require contract metadata. |
| `+"`contract`"+` | active/P0 handoffs | The upstream behavior or Gormes-native behavior being preserved. |
| `+"`contract_status`"+` | contract rows | `+"`missing`"+`, `+"`draft`"+`, `+"`fixture_ready`"+`, or `+"`validated`"+`. |
| `+"`slice_size`"+` | contract rows and umbrella rows | `+"`small`"+`, `+"`medium`"+`, `+"`large`"+`, or `+"`umbrella`"+`. |
| `+"`execution_owner`"+` | contract rows and umbrella rows | `+"`docs`"+`, `+"`gateway`"+`, `+"`memory`"+`, `+"`provider`"+`, `+"`tools`"+`, `+"`skills`"+`, or `+"`orchestrator`"+`. |
| `+"`trust_class`"+` | active/P0 handoffs | Allowed caller classes: `+"`operator`"+`, `+"`gateway`"+`, `+"`child-agent`"+`, `+"`system`"+`. |
| `+"`degraded_mode`"+` | active/P0 handoffs | How partial capability is visible in doctor, status, audit, logs, or generated docs. |
| `+"`fixture`"+` | active/P0 handoffs | Local package/path/fixture set proving compatibility without live credentials. |
| `+"`source_refs`"+` | active/P0 handoffs | Docs or code references used to derive the contract. |
| `+"`blocked_by`"+` | optional | Roadmap rows or conditions blocking this slice. Requires `+"`ready_when`"+`. |
| `+"`unblocks`"+` | optional | Downstream rows enabled by this slice. |
| `+"`ready_when`"+` | contract rows and blocked rows | Concrete condition that makes the row assignable. |
| `+"`not_ready_when`"+` | umbrella rows, optional elsewhere | Conditions that make the row unsafe or too broad to assign. |
| `+"`acceptance`"+` | active/P0 handoffs | Testable done criteria. |
| `+"`write_scope`"+` | contract rows | Files, directories, or packages a builder skill may edit for this slice. |
| `+"`test_commands`"+` | contract rows | Commands that prove the slice without live provider or platform credentials. Required for skill-builder selection unless `+"`no_test_required`"+` is present. |
| `+"`no_test_required`"+` | rare testless contract rows | Explicit reason a row has no focused executable test command. Rows without `+"`test_commands`"+` or this field are not worker-ready. |
| `+"`done_signal`"+` | contract rows | Observable evidence that the row can move forward or close. |

## Meta Fields

| Field | Required when | Meaning |
|---|---|---|
| `+"`meta.builder_loop.entrypoint`"+` | skill handoff metadata is declared | Primary skill-routing entrypoint. Historical field name retained for schema compatibility. |
| `+"`meta.builder_loop.plan`"+` | skill handoff metadata is declared | Canonical completion plan for skill-driven work. |
| `+"`meta.builder_loop.agent_queue`"+` | skill handoff metadata is declared | Generated queue page for assignable rows. |
| `+"`meta.builder_loop.progress_schema`"+` | skill handoff metadata is declared | This schema reference. |
| `+"`meta.builder_loop.candidate_source`"+` | skill handoff metadata is declared | Canonical progress file consumed by skills. |
| `+"`meta.builder_loop.unit_test`"+` | skill handoff metadata is declared | Fast verification command for progress docs/schema behavior. |
| `+"`meta.builder_loop.candidate_policy`"+` | skill handoff metadata is declared | Shared selection rules used by builder skills. |

## Validation Rules

- `+"`docs/data/progress.json`"+` must not exist.
- if `+"`meta.builder_loop`"+` is declared, entrypoint, plan, candidate source, generated docs, unit test, and candidate policy must all be present.
- `+"`in_progress`"+` rows cannot use `+"`slice_size: umbrella`"+`.
- item-level `+"`P0`"+` and `+"`in_progress`"+` rows must include full contract metadata.
- contract rows must declare `+"`slice_size`"+`, `+"`execution_owner`"+`, `+"`ready_when`"+`, `+"`write_scope`"+`, `+"`test_commands`"+` (or explicit `+"`no_test_required`"+`), and `+"`done_signal`"+`.
- blocked rows must declare `+"`ready_when`"+`.
- `+"`fixture_ready`"+` rows must name a concrete fixture package or path.
- complete rows with contract metadata must use `+"`contract_status: validated`"+`.

## Planning Metrics

Progress is measured from derived status counts, not from free-form narrative.
`+"`Progress.Stats()`"+` walks phases, subphases, and items and tallies
`+"`complete`"+`, `+"`in_progress`"+`, and `+"`planned`"+`. A subphase is
`+"`complete`"+` only when every item is complete, `+"`in_progress`"+` when any
item has started, and `+"`planned`"+` when no item has started. README and the
architecture-plan index use those derived counts for shipped/subphase totals.

Future work is measured from contract-bearing rows. A row becomes assignable
when it is not `+"`complete`"+`, has no `+"`blocked_by`"+` dependency, is not
`+"`slice_size: umbrella`"+`, and declares the handoff fields builder skills need:
`+"`source_refs`"+`, `+"`write_scope`"+`, `+"`test_commands`"+` or `+"`no_test_required`"+`,
`+"`acceptance`"+`, `+"`ready_when`"+`, `+"`not_ready_when`"+`, and
`+"`done_signal`"+` whenever applicable. `+"`agent-queue.md`"+` is the
assignable-work view; `+"`blocked-slices.md`"+` is the deferred-work view; and
`+"`umbrella-cleanup.md`"+` is the work that must be split before assignment.

Planner quality is measured by reducing ambiguity for builder skills:
exact upstream refs, local file paths, fixture names, validation commands,
dependency edges, and degraded-mode behavior count as useful planning;
generic notes without bounded tests or write scope do not.

## Generated Agent Surfaces

- `+"`docs/content/building-gormes/builder-loop/builder-loop-handoff.md`"+` lists shared skill entrypoint, plan, candidate source, generated docs, test command, and candidate policy.
- `+"`docs/content/building-gormes/builder-loop/agent-queue.md`"+` lists only unblocked, non-umbrella contract rows with owner, size, readiness, degraded mode, fixture, write scope, test commands or a no-test-required reason, done signal, acceptance, and source references.
- `+"`docs/content/building-gormes/builder-loop/blocked-slices.md`"+` keeps blocked rows out of the execution queue while preserving their unblock condition.
- `+"`docs/content/building-gormes/builder-loop/umbrella-cleanup.md`"+` lists broad inventory rows that must be split before assignment.
- `+"`docs/content/building-gormes/modules/`"+` contains generated module-scoped roadmap review pages. These are views over the single logical backlog, not side queues.

## Good Row

`+"```json"+`
{
  "name": "Provider transcript harness",
  "status": "planned",
  "priority": "P1",
  "contract": "Provider-neutral request and stream event transcript harness",
  "contract_status": "fixture_ready",
  "slice_size": "medium",
  "execution_owner": "provider",
  "trust_class": ["system"],
  "degraded_mode": "Provider status reports missing fixture coverage before routing can select the adapter.",
  "fixture": "internal/hermes/testdata/provider_transcripts",
  "source_refs": ["docs/content/upstream-hermes/source-study.md"],
  "ready_when": ["Anthropic transcript fixtures replay without live credentials."],
  "write_scope": ["internal/hermes/"],
  "test_commands": ["go test ./internal/hermes -count=1"],
  "done_signal": ["Provider transcript replay passes from captured fixtures."],
  "acceptance": ["All provider transcript fixtures pass under go test ./internal/hermes."]
}
`+"```"+`

## Bad Row

`+"```json"+`
{
  "name": "Port CLI",
  "status": "in_progress",
  "slice_size": "umbrella"
}
`+"```"+`

This is invalid because an active execution row cannot be an umbrella, and it
does not explain the contract, fixture, caller trust class, degraded mode, or
acceptance criteria.
`) + "\n"
}

type contractRow struct {
	PhaseKey    string
	PhaseName   string
	SubphaseKey string
	Subphase    string
	Item        Item
}

type moduleRowCounts struct {
	Total    int
	Status   map[Status]int
	Priority map[string]int
}

func countModuleRows(rows []contractRow) moduleRowCounts {
	counts := moduleRowCounts{
		Status:   map[Status]int{StatusComplete: 0, StatusInProgress: 0, StatusPlanned: 0},
		Priority: map[string]int{},
	}
	for _, row := range rows {
		counts.Total++
		counts.Status[row.Item.Status]++
		priority := row.Item.Priority
		if priority == "" {
			priority = "unset"
		}
		counts.Priority[priority]++
	}
	return counts
}

func moduleRoadmapRows(p *Progress, module string) []contractRow {
	if p == nil {
		return nil
	}
	var rows []contractRow
	for _, phKey := range sortedMapKeys(p.Phases) {
		ph := p.Phases[phKey]
		for _, spKey := range sortedMapKeys(ph.Subphases) {
			sp := ph.Subphases[spKey]
			for _, it := range sp.Items {
				if Module(it, phKey, spKey) != module {
					continue
				}
				rows = append(rows, contractRow{
					PhaseKey:    phKey,
					PhaseName:   ph.Name,
					SubphaseKey: spKey,
					Subphase:    sp.Name,
					Item:        it,
				})
			}
		}
	}
	return rows
}

func contractRows(p *Progress) []contractRow {
	var rows []contractRow
	for _, row := range allItemRows(p) {
		if row.Item.Contract == "" {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func allItemRows(p *Progress) []contractRow {
	if p == nil {
		return nil
	}
	var rows []contractRow
	for _, phKey := range sortedMapKeys(p.Phases) {
		ph := p.Phases[phKey]
		for _, spKey := range sortedMapKeys(ph.Subphases) {
			sp := ph.Subphases[spKey]
			for _, it := range sp.Items {
				rows = append(rows, contractRow{
					PhaseKey:    phKey,
					PhaseName:   ph.Name,
					SubphaseKey: spKey,
					Subphase:    sp.Name,
					Item:        it,
				})
			}
		}
	}
	return rows
}

func nextSliceRows(rows []contractRow, limit int) []contractRow {
	byKey := map[string]contractRow{}
	for _, row := range rows {
		byKey[contractRowKey(row.PhaseKey, row.SubphaseKey, row.Item.Name)] = row
	}

	handoffs := projectActiveHandoffsFromRows(rows, limit)
	out := make([]contractRow, 0, len(handoffs))
	for _, handoff := range handoffs {
		original, ok := byKey[contractRowKey(handoff.Identity.PhaseID, handoff.Identity.SubphaseID, handoff.Identity.ItemName)]
		if !ok {
			continue
		}
		if len(original.Item.BlockedBy) > 0 {
			// In this generated assignable-work view, completed dependencies are
			// no longer blockers. Keep the canonical row untouched; only avoid
			// rendering misleading "Blocked by ..." why-now text.
			original.Item.BlockedBy = nil
		}
		out = append(out, original)
	}
	return out
}

func contractRowKey(phaseKey, subphaseKey, itemName string) string {
	return phaseKey + "\x00" + subphaseKey + "\x00" + itemName
}

func workitemInputFromContractRow(row contractRow) workitem.RowInput {
	it := row.Item
	input := workitem.RowInput{
		PhaseID:        row.PhaseKey,
		PhaseName:      row.PhaseName,
		SubphaseID:     row.SubphaseKey,
		SubphaseName:   row.Subphase,
		ItemName:       it.Name,
		Status:         string(it.Status),
		Priority:       it.Priority,
		Contract:       it.Contract,
		ContractStatus: string(it.ContractStatus),
		SliceSize:      string(it.SliceSize),
		ExecutionOwner: string(it.ExecutionOwner),
		TrustClass:     it.TrustClass,
		DegradedMode:   it.DegradedMode,
		Fixture:        it.Fixture,
		SourceRefs:     it.SourceRefs,
		BlockedBy:      it.BlockedBy,
		Unblocks:       it.Unblocks,
		ReadyWhen:      it.ReadyWhen,
		NotReadyWhen:   it.NotReadyWhen,
		Acceptance:     it.Acceptance,
		WriteScope:     it.WriteScope,
		TestCommands:   it.TestCommands,
		NoTestRequired: it.NoTestRequiredReason,
		DoneSignal:     it.DoneSignal,
		Note:           it.Note,
		HasBlocker:     it.Blocker != nil,
	}
	if it.PlannerVerdict != nil && it.PlannerVerdict.NeedsHuman {
		input.NeedsHuman = true
	}
	if it.Health != nil {
		input.PenaltyApplied = workitem.FailurePenalty(it.Health.ConsecutiveFailures) + 2*len(it.Health.BackendsTried)
		if it.Health.Quarantine != nil {
			input.Quarantined = true
			input.StaleQuarantine = it.Health.Quarantine.SpecHash != ItemSpecHash(&it)
		}
	}
	return input
}

func whyNow(it Item) string {
	switch {
	case it.Status == StatusInProgress:
		return "Already active; contract metadata keeps execution bounded."
	case it.Priority == "P0":
		return "P0 handoff; needs contract proof before closeout."
	case len(it.BlockedBy) > 0:
		return "Blocked by " + joinOrDash(it.BlockedBy) + "; keep dependencies visible."
	case len(it.Unblocks) > 0:
		return "Unblocks " + joinOrDash(it.Unblocks) + "."
	default:
		return "Contract metadata is present; ready for a focused spec or fixture slice."
	}
}

func blockedRows(rows []contractRow) []contractRow {
	byKey := map[string]contractRow{}
	inputs := make([]workitem.RowInput, 0, len(rows))
	for _, row := range rows {
		byKey[contractRowKey(row.PhaseKey, row.SubphaseKey, row.Item.Name)] = row
		inputs = append(inputs, workitemInputFromContractRow(row))
	}
	classified := workitem.Classify(inputs, workitem.Options{ActiveFirst: true})

	var out []contractRow
	for _, row := range classified {
		if row.Classification != workitem.ClassificationBlocked {
			continue
		}
		original, ok := byKey[contractRowKey(row.PhaseID, row.SubphaseID, row.ItemName)]
		if !ok || original.Item.Contract == "" {
			continue
		}
		out = append(out, original)
	}
	return out
}

func umbrellaRows(p *Progress) []contractRow {
	var rows []contractRow
	for _, phKey := range sortedMapKeys(p.Phases) {
		ph := p.Phases[phKey]
		for _, spKey := range sortedMapKeys(ph.Subphases) {
			sp := ph.Subphases[spKey]
			for _, it := range sp.Items {
				if it.SliceSize != SliceSizeUmbrella {
					continue
				}
				rows = append(rows, contractRow{
					PhaseKey:    phKey,
					PhaseName:   ph.Name,
					SubphaseKey: spKey,
					Subphase:    sp.Name,
					Item:        it,
				})
			}
		}
	}
	return rows
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func joinCodeOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

func formatPriorityCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "-"
	}
	var parts []string
	for _, priority := range []string{"P0", "P1", "P2", "P3", "P4", "unset"} {
		if n := counts[priority]; n > 0 {
			parts = append(parts, fmt.Sprintf("`%s`: %d", priority, n))
		}
	}
	for _, priority := range sortedMapKeys(counts) {
		switch priority {
		case "P0", "P1", "P2", "P3", "P4", "unset":
			continue
		}
		parts = append(parts, fmt.Sprintf("`%s`: %d", priority, counts[priority]))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " · ")
}

func moduleDisplayName(module string) string {
	switch module {
	case ModuleCLI, ModuleSTT, ModuleTTS, ModuleTUI:
		return strings.ToUpper(module)
	}
	parts := strings.Split(module, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	if s == "" {
		return "-"
	}
	return s
}
